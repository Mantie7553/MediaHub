package downloader

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/mangafiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

// the language chapters are downloaded in
const mangaLanguage = "en"

// pause between chapters, MangaDex@Home allows about 40 chapter lookups a minute
var chapterDelay = 1500 * time.Millisecond

// waits before retrying a failed page download
var imageRetryDelays = []time.Duration{2 * time.Second, 5 * time.Second}

/*
pageSource: where a chapter's page images come from, the MangaDex client or a fake in tests
*/
type pageSource interface {
	PageURLs(chapterID string) ([]string, error)
	Image(imageURL string) ([]byte, error)
}

/*
Function:	RunMangaDex
Purpose:	Download every chapter MangaDex hosts for a manga that the library doesn't have
yet, as CBZ files in the manga's folder, record them in manga_chapters and report progress
on the job. Running it again only fetches what is missing, so it also picks up new chapters

Params:
  - db: a pointer to our database
  - jobID: the download job to update
  - mediaItemID: the manga, its external_id is the MangaDex id
  - dest: the Manga folder
*/
func RunMangaDex(db *sql.DB, jobID string, mediaItemID string, dest string) {
	logger.Info("MangaDex running job %s", jobID)
	if _, err := db.Exec(`UPDATE download_jobs SET status = 'downloading', started_at = NOW() WHERE id = $1`, jobID); err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	var title string
	var mangadexID sql.NullString
	err := db.QueryRow(`SELECT title, external_id FROM media_items WHERE id = $1`, mediaItemID).Scan(&title, &mangadexID)
	if err != nil {
		markFailed(db, jobID, err.Error())
		return
	}
	if !mangadexID.Valid || mangadexID.String == "" {
		markFailed(db, jobID, "this manga has no MangaDex id to download from")
		return
	}

	client := mangadex.NewMangaDexClient("MANGADEX_URL")
	chapters, err := client.Chapters(mangadexID.String, mangaLanguage)
	if err != nil {
		markFailed(db, jobID, "could not list chapters: "+err.Error())
		return
	}
	if len(chapters) == 0 {
		markFailed(db, jobID, "MangaDex hosts no English chapters of this manga (licensed series often only link to the publisher)")
		return
	}

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

	added, failed, skipped := 0, 0, 0
	for i, ch := range chapters {
		progress := func() {
			db.Exec(`UPDATE download_jobs SET progress_pct = $1 WHERE id = $2`, (i+1)*100/len(chapters), jobID)
		}
		// already in the library, whatever its file is called (uploads, imports)
		if have[ch.NumberValue()] {
			skipped++
			progress()
			continue
		}

		path := mangafiles.ExistingChapter(dir, ch.Number)
		pages := 0
		var chErr error
		if path != "" {
			pages = mangafiles.PageCount(path)
		} else {
			path = filepath.Join(dir, mangafiles.FileName(ch.Number, ch.Title))
			pages, chErr = downloadChapter(client, ch, path)
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
			mediaItemID, ch.NumberValue(), ch.Title, path, pages,
		); dbErr != nil {
			failed++
			logger.Error("job %s: saving chapter %s failed: %s", jobID, ch.Number, dbErr.Error())
		} else {
			added++
		}
		progress()
	}

	if failed > 0 && added == 0 {
		markFailed(db, jobID, fmt.Sprintf("all %d missing chapters failed to download, see the server log", failed))
		return
	}

	// what happened, shown on the Downloads page
	note := fmt.Sprintf("%d new chapters", added)
	if skipped > 0 {
		note += fmt.Sprintf(", %d already in the library", skipped)
	}
	if failed > 0 {
		note += fmt.Sprintf(", %d failed (check again to retry them)", failed)
	}
	db.Exec(`UPDATE download_jobs
		SET status = 'complete', progress_pct = 100, completed_at = NOW(), error_message = $1
		WHERE id = $2`, note, jobID)
	logger.Info("MangaDex completed job %s: %s", jobID, note)
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
func downloadChapter(src pageSource, ch mangadex.Chapter, path string) (int, error) {
	urls, err := src.PageURLs(ch.ID)
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
		// zero padded so the pages sort in order
		w, err := zw.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("%03d%s", i+1, pageExt(u)), Method: zip.Store})
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

// pageExt is the image's extension from its URL, .jpg when it has none the reader knows
func pageExt(url string) string {
	ext := strings.ToLower(filepath.Ext(url))
	if mangafiles.ImageType(ext) != "" {
		return ext
	}
	return ".jpg"
}
