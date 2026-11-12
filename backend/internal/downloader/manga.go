package downloader

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/suwayomi"
	"github.com/Mantie7553/MediaHub/backend/internal/mangafiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

// the language chapters are downloaded in
const mangaLanguage = "en"

// pause between chapters, MangaDex@Home allows about 40 chapter lookups a minute; it also
// keeps Suwayomi from hammering a source site
var chapterDelay = 1500 * time.Millisecond

// waits before retrying a failed page download
var imageRetryDelays = []time.Duration{2 * time.Second, 5 * time.Second}

/*
pageSource: where a chapter's page images come from, the MangaDex or Suwayomi client, or a
fake in tests
*/
type pageSource interface {
	PageURLs(chapterID string) ([]string, error)
	Image(imageURL string) ([]byte, error)
}

/*
plannedChapter: one chapter a manga's sources have, and where to get it from
  - Number: as written in file names ("12", "52.5"), empty for a oneshot
  - FromSuwayomi: true when MangaDex doesn't host it
*/
type plannedChapter struct {
	ID           string
	Number       string
	Value        float64
	Title        string
	FromSuwayomi bool
}

/*
Function:	mergeChapters
Purpose:	Every chapter number the sources have, in reading order. MangaDex's copy is used
when it hosts one, Suwayomi only fills the numbers MangaDex doesn't have
*/
func mergeChapters(md []mangadex.Chapter, sw []suwayomi.Chapter) []plannedChapter {
	planned := make([]plannedChapter, 0, len(md)+len(sw))
	onMangaDex := map[float64]bool{}
	for _, ch := range md {
		onMangaDex[ch.NumberValue()] = true
		planned = append(planned, plannedChapter{ID: ch.ID, Number: ch.Number, Value: ch.NumberValue(), Title: ch.Title})
	}
	for _, ch := range sw {
		if onMangaDex[ch.Number] {
			continue
		}
		planned = append(planned, plannedChapter{
			ID:           strconv.Itoa(ch.ID),
			Number:       strconv.FormatFloat(ch.Number, 'f', -1, 64),
			Value:        ch.Number,
			Title:        ch.Title,
			FromSuwayomi: true,
		})
	}
	sort.SliceStable(planned, func(i, j int) bool { return planned[i].Value < planned[j].Value })
	return planned
}

/*
Function:	knownLatest
Purpose:	The newest chapter MediaHub knows the series has reached, from anywhere but Suwayomi:
MangaDex's chapters, the series' chapter total, and chapters already in the library
*/
func knownLatest(db *sql.DB, mediaItemID string, chapters []plannedChapter, have map[float64]bool) float64 {
	latest := -1.0
	for _, ch := range chapters {
		if !ch.FromSuwayomi && ch.Value > latest {
			latest = ch.Value
		}
	}
	for n := range have {
		if n > latest {
			latest = n
		}
	}
	var total sql.NullInt64
	db.QueryRow(`SELECT total_chapters FROM manga_metadata WHERE media_item_id = $1`, mediaItemID).Scan(&total)
	if total.Valid && float64(total.Int64) > latest {
		latest = float64(total.Int64)
	}
	return latest
}

/*
Function:	staleLinkNote
Purpose:	A warning for the Downloads page when the linked Suwayomi title is behind what the
series is known to have reached. Sources often list a title several times and stop updating
all but one, so a link can go stale without anything failing. "" when it isn't behind
Params:
  - source: the linked source's name
  - swLatest: the linked title's newest chapter, -1 when it has none
  - known: the newest chapter known from elsewhere, -1 when nothing is known
*/
func staleLinkNote(source string, swLatest float64, known float64) string {
	if swLatest < 0 {
		return fmt.Sprintf(", the linked %s title has no chapters, re-link it from the manga's menu", source)
	}
	if known > swLatest {
		return fmt.Sprintf(", the linked %s title stops at ch. %s but the series has reached ch. %s, re-link it from the manga's menu to get newer chapters",
			source, strconv.FormatFloat(swLatest, 'f', -1, 64), strconv.FormatFloat(known, 'f', -1, 64))
	}
	return ""
}

/*
suwayomiLink: the Suwayomi source and title an admin linked a manga to
*/
type suwayomiLink struct {
	SourceName string
	MangaID    int
}

/*
Function:	SuwayomiLink
Purpose:	The Suwayomi title a manga is linked to, nil when it has none
*/
func SuwayomiLink(db *sql.DB, mediaItemID string) (*suwayomiLink, error) {
	var link suwayomiLink
	err := db.QueryRow(
		`SELECT source_name, suwayomi_manga_id FROM suwayomi_items WHERE media_item_id = $1`, mediaItemID,
	).Scan(&link.SourceName, &link.MangaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

/*
chapterPlan: what a manga's sources have, from listChapters
  - mdErr, swErr: a source that failed while the other answered
*/
type chapterPlan struct {
	title    string
	chapters []plannedChapter
	link     *suwayomiLink
	// the newest chapter the linked Suwayomi title has, -1 when it has none
	swLatest float64
	mangadex *mangadex.MangaDexClient
	suwayomi *suwayomi.Client
	mdErr    error
	swErr    error
}

/*
Function:	listChapters
Purpose:	Every chapter a manga's sources have: MangaDex, and its linked Suwayomi source for the
numbers MangaDex doesn't host. One source being down doesn't stop the other. The error is
written for the Downloads page
*/
func listChapters(db *sql.DB, mediaItemID string) (*chapterPlan, error) {
	plan := &chapterPlan{
		mangadex: mangadex.NewMangaDexClient("MANGADEX_URL"),
		suwayomi: suwayomi.NewClient(),
	}
	var mangadexID sql.NullString
	err := db.QueryRow(`SELECT title, external_id FROM media_items WHERE id = $1`, mediaItemID).Scan(&plan.title, &mangadexID)
	if err != nil {
		return nil, err
	}
	if plan.link, err = SuwayomiLink(db, mediaItemID); err != nil {
		return nil, err
	}
	hasMangaDex := mangadexID.Valid && mangadexID.String != ""
	if !hasMangaDex && plan.link == nil {
		return nil, errors.New("this manga isn't linked to MangaDex or a Suwayomi source")
	}

	var mdChapters []mangadex.Chapter
	if hasMangaDex {
		if mdChapters, plan.mdErr = plan.mangadex.Chapters(mangadexID.String, mangaLanguage); plan.mdErr != nil {
			logger.Error("%s: MangaDex chapters: %s", plan.title, plan.mdErr.Error())
		}
	}
	var swChapters []suwayomi.Chapter
	plan.swLatest = -1
	if plan.link != nil {
		if swChapters, plan.swErr = plan.suwayomi.Chapters(plan.link.MangaID); plan.swErr != nil {
			logger.Error("%s: %s chapters: %s", plan.title, plan.link.SourceName, plan.swErr.Error())
		}
	}
	if (plan.mdErr != nil || !hasMangaDex) && (plan.swErr != nil || plan.link == nil) {
		reason := plan.mdErr
		if reason == nil {
			reason = plan.swErr
		}
		return nil, fmt.Errorf("could not list chapters: %w", reason)
	}

	if len(swChapters) > 0 {
		plan.swLatest = swChapters[len(swChapters)-1].Number
	}
	plan.chapters = mergeChapters(mdChapters, swChapters)
	if len(plan.chapters) == 0 {
		if plan.link != nil {
			return nil, fmt.Errorf("neither MangaDex nor %s has chapters of this manga", plan.link.SourceName)
		}
		return nil, errors.New("MangaDex hosts no English chapters of this manga (licensed series often only link to the publisher). Link a Suwayomi source from the manga's menu to get them elsewhere")
	}
	return plan, nil
}

/*
Function:	RunMangaDex
Purpose:	Download every chapter of a manga that the library doesn't have yet, as CBZ files in
the manga's folder, record them in manga_chapters and report progress on the job. Chapters
come from MangaDex, and from the manga's linked Suwayomi source for the numbers MangaDex doesn't
host (licensed series). Running it again only fetches what is missing, so it also picks up new
chapters

Params:
  - db: a pointer to our database
  - jobID: the download job to update
  - mediaItemID: the manga, its external_id is the MangaDex id
  - dest: the Manga folder
*/
func RunMangaDex(db *sql.DB, jobID string, mediaItemID string, dest string) {
	logger.Info("Manga download running job %s", jobID)
	if _, err := db.Exec(`UPDATE download_jobs SET status = 'downloading', started_at = NOW() WHERE id = $1`, jobID); err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	plan, err := listChapters(db, mediaItemID)
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}
	title, chapters, link := plan.title, plan.chapters, plan.link
	mdClient, swClient, mdErr, swErr := plan.mangadex, plan.suwayomi, plan.mdErr, plan.swErr

	have, err := ChapterNumbers(db, mediaItemID)
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}
	dir, err := MangaFolder(db, mediaItemID, dest, title)
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	added, fromSuwayomi, failed, skipped := 0, 0, 0, 0
	for i, ch := range chapters {
		progress := func() {
			db.Exec(`UPDATE download_jobs SET progress_pct = $1 WHERE id = $2`, (i+1)*100/len(chapters), jobID)
		}
		// already in the library, whatever its file is called (uploads, imports)
		if have[ch.Value] {
			skipped++
			progress()
			continue
		}

		var src pageSource = mdClient
		if ch.FromSuwayomi {
			src = swClient
		}
		path := mangafiles.ExistingChapter(dir, ch.Number)
		pages := 0
		var chErr error
		if path != "" {
			pages = mangafiles.PageCount(path)
		} else {
			path = filepath.Join(dir, mangafiles.FileName(ch.Number, ch.Title))
			pages, chErr = downloadChapter(src, ch.ID, path)
			if i < len(chapters)-1 {
				time.Sleep(chapterDelay)
			}
		}

		if chErr != nil {
			failed++
			logger.Error("job %s: chapter %s failed: %s", jobID, ch.Number, chErr.Error())
		} else if _, dbErr := db.Exec(
			`INSERT INTO manga_chapters (media_item_id, chapter_number, title, file_path, page_count, created_at)
			VALUES ($1, $2, NULLIF($3, ''), $4, $5, NOW())
			ON CONFLICT (media_item_id, chapter_number) DO NOTHING`,
			mediaItemID, ch.Value, ch.Title, path, pages,
		); dbErr != nil {
			failed++
			logger.Error("job %s: saving chapter %s failed: %s", jobID, ch.Number, dbErr.Error())
		} else {
			added++
			if ch.FromSuwayomi {
				fromSuwayomi++
			}
		}
		progress()
	}

	if added > 0 {
		tracking.PartsAdded(db, mediaItemID)
	}

	if failed > 0 && added == 0 {
		markFailed(db, jobID, fmt.Sprintf("all %d missing chapters failed to download, see the server log", failed))
		return
	}

	// what happened, shown on the Downloads page
	note := fmt.Sprintf("%d new chapters", added)
	if fromSuwayomi > 0 {
		note += fmt.Sprintf(" (%d from %s)", fromSuwayomi, link.SourceName)
	}
	if skipped > 0 {
		note += fmt.Sprintf(", %d already in the library", skipped)
	}
	if failed > 0 {
		note += fmt.Sprintf(", %d failed (check again to retry them)", failed)
	}
	if mdErr != nil {
		note += ", MangaDex couldn't be reached"
	}
	if swErr != nil {
		note += fmt.Sprintf(", %s couldn't be reached", link.SourceName)
	} else if link != nil {
		note += staleLinkNote(link.SourceName, plan.swLatest, knownLatest(db, mediaItemID, chapters, have))
	}
	db.Exec(`UPDATE download_jobs
		SET status = 'complete', progress_pct = 100, completed_at = NOW(), error_message = $1
		WHERE id = $2`, note, jobID)
	logger.Info("Manga download completed job %s: %s", jobID, note)
}

/*
Function:	ChapterNumbers
Purpose:	The chapter numbers a manga already has in the library
*/
func ChapterNumbers(db *sql.DB, mediaItemID string) (map[float64]bool, error) {
	rows, err := db.Query(
		`SELECT chapter_number FROM manga_chapters WHERE media_item_id = $1 AND chapter_number IS NOT NULL`,
		mediaItemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	have := map[float64]bool{}
	for rows.Next() {
		var n float64
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		have[n] = true
	}
	return have, rows.Err()
}

/*
Function:	MangaFolder
Purpose:	The folder a manga's chapters go in: wherever its existing chapters are, so new
ones land beside them, otherwise <mangaRoot>/<title>. The folder is created if needed
*/
func MangaFolder(db *sql.DB, mediaItemID string, mangaRoot string, title string) (string, error) {
	var existing sql.NullString
	db.QueryRow(
		`SELECT file_path FROM manga_chapters
		WHERE media_item_id = $1 AND file_path IS NOT NULL
		ORDER BY chapter_number DESC LIMIT 1`,
		mediaItemID,
	).Scan(&existing)

	dir := filepath.Join(mangaRoot, mangafiles.SanitizeFileName(title))
	if existing.Valid && existing.String != "" {
		dir = filepath.Dir(existing.String)
	}
	return dir, os.MkdirAll(dir, 0o755)
}

/*
Function:	downloadChapter
Purpose:	Fetch a chapter's pages and write them to a CBZ at path. It is written to a
temporary file first, so an interrupted download never leaves a broken chapter behind
Returns:	the number of pages written
*/
func downloadChapter(src pageSource, chapterID string, path string) (int, error) {
	urls, err := src.PageURLs(chapterID)
	if err != nil {
		return 0, err
	}
	if len(urls) == 0 {
		return 0, fmt.Errorf("chapter has no pages")
	}

	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	// clears the temporary file on any failure below
	ok := false
	defer func() {
		if !ok {
			f.Close()
			os.Remove(tmp)
		}
	}()

	zw := zip.NewWriter(f)
	for i, u := range urls {
		img, err := fetchWithRetry(src, u)
		if err != nil {
			return 0, fmt.Errorf("page %d: %w", i+1, err)
		}
		// named from the bytes, not the URL: sources aren't trusted to send images, and
		// some (Suwayomi) serve pages from URLs with no extension
		ext := pageExt(img)
		if ext == "" {
			return 0, fmt.Errorf("page %d is not an image", i+1)
		}
		// zero padded so the pages sort in order
		w, err := zw.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("%03d%s", i+1, ext), Method: zip.Store})
		if err != nil {
			return 0, err
		}
		if _, err := w.Write(img); err != nil {
			return 0, err
		}
	}
	if err := zw.Close(); err != nil {
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return 0, err
	}
	ok = true
	return len(urls), nil
}

// fetchWithRetry downloads one image, retrying when a volunteer image server hiccups
func fetchWithRetry(src pageSource, url string) ([]byte, error) {
	img, err := src.Image(url)
	for _, wait := range imageRetryDelays {
		if err == nil {
			return img, nil
		}
		time.Sleep(wait)
		img, err = src.Image(url)
	}
	return img, err
}

// pageExt is the extension for a page image, judged from its bytes. "" when it isn't an
// image format the reader serves
func pageExt(img []byte) string {
	switch http.DetectContentType(img) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	}
	// Go's sniffer doesn't know AVIF: an ISO-BMFF "ftyp" box with an avif brand
	if len(img) >= 12 && bytes.Equal(img[4:8], []byte("ftyp")) &&
		(bytes.Equal(img[8:12], []byte("avif")) || bytes.Equal(img[8:12], []byte("avis"))) {
		return ".avif"
	}
	return ""
}
