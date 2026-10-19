package downloader

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
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
Purpose:	Download every chapter MangaDex hosts for a manga as CBZ files in
dest/<title>/, record them in manga_chapters and report progress on the job. Chapters
already on disk are kept, so running it again only fetches what is missing

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

	dir := filepath.Join(dest, sanitizeFileName(title))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		markFailed(db, jobID, err.Error())
		return
	}

	failed := 0
	for i, ch := range chapters {
		path := existingChapter(dir, ch)
		pages := 0
		var chErr error
		if path != "" {
			pages = cbzPageCount(path)
		} else {
			path = filepath.Join(dir, chapterFileName(ch))
			pages, chErr = downloadChapter(client, ch, path)
			if i < len(chapters)-1 {
				time.Sleep(chapterDelay)
			}
		}

		if chErr != nil {
			failed++
			logger.Error("job %s: chapter %s failed: %s", jobID, ch.Number, chErr.Error())
		} else if _, dbErr := db.Exec(
			`INSERT INTO manga_chapters (media_item_id, chapter_number, file_path, page_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id, chapter_number) DO NOTHING`,
			mediaItemID, ch.NumberValue(), path, pages,
		); dbErr != nil {
			logger.Error("job %s: saving chapter %s failed: %s", jobID, ch.Number, dbErr.Error())
		}

		db.Exec(`UPDATE download_jobs SET progress_pct = $1 WHERE id = $2`, (i+1)*100/len(chapters), jobID)
	}

	if failed == len(chapters) {
		markFailed(db, jobID, "every chapter failed to download, see the server log")
		return
	}

	// a few failed chapters still leave a usable manga, the note says to run it again
	var note sql.NullString
	if failed > 0 {
		note = sql.NullString{String: fmt.Sprintf("%d of %d chapters failed, request it again to retry them", failed, len(chapters)), Valid: true}
	}
	db.Exec(`UPDATE download_jobs
		SET status = 'complete', progress_pct = 100, completed_at = NOW(), error_message = $1
		WHERE id = $2`, note, jobID)
	logger.Info("MangaDex completed job %s: %d chapters, %d failed", jobID, len(chapters), failed)
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
		// zero padded so the reader's name sort is page order
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
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return ext
	}
	return ".jpg"
}

/*
Function:	chapterFileName
Purpose:	The CBZ name for a chapter: "[0012] Chapter 12 - Title.cbz". The number in
brackets is what the library sync reads back, and the same pattern mangal used, so
chapters it downloaded are recognised and not fetched again
*/
func chapterFileName(ch mangadex.Chapter) string {
	name := "Oneshot"
	if ch.Number != "" {
		name = "Chapter " + ch.Number
	}
	if ch.Title != "" {
		name += " - " + ch.Title
	}
	return sanitizeFileName(fmt.Sprintf("[%s] %s", paddedNumber(ch), name)) + ".cbz"
}

// paddedNumber writes the chapter number with 4 whole digits: 12 -> 0012, 52.5 -> 0052.5
func paddedNumber(ch mangadex.Chapter) string {
	whole, frac, _ := strings.Cut(ch.Number, ".")
	n, err := strconv.Atoi(whole)
	if err != nil {
		n = 0
	}
	if frac != "" {
		return fmt.Sprintf("%04d.%s", n, frac)
	}
	return fmt.Sprintf("%04d", n)
}

/*
Function:	existingChapter
Purpose:	The path of a CBZ already in dir for this chapter number, whatever its title
part says, or "" when there is none
*/
func existingChapter(dir string, ch mangadex.Chapter) string {
	prefix := "[" + paddedNumber(ch) + "]"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasPrefix(name, prefix) && strings.EqualFold(filepath.Ext(name), ".cbz") {
			return filepath.Join(dir, name)
		}
	}
	return ""
}

// cbzPageCount counts the images in a CBZ, 0 when it can't be opened
func cbzPageCount(path string) int {
	r, err := zip.OpenReader(path)
	if err != nil {
		return 0
	}
	defer r.Close()
	count := 0
	for _, f := range r.File {
		switch strings.ToLower(filepath.Ext(f.Name)) {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
			count++
		}
	}
	return count
}

// sanitizeFileName drops characters Windows and SMB shares refuse in file names
func sanitizeFileName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimRight(strings.TrimSpace(name), ".")
	if name == "" {
		return "Untitled"
	}
	return name
}
