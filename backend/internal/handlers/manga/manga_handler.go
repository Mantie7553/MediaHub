package manga

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/downloader"
	"github.com/Mantie7553/MediaHub/backend/internal/mangafiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

// largest chapter file accepted. A chapter is usually 5-40 MB; this leaves room for a long
// colour one without letting a mistaken upload fill the disk
const maxChapterBytes = 500 << 20

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	UploadChapter
Purpose:	Admin: add one chapter file (CBZ or ZIP of page images) to a manga. The chapter
number comes from the "number" field, or is read from the file name. The file is checked
for page images, then saved beside the manga's other chapters as "[0098] Chapter 98.cbz"

Params:
  - w: http response writer to respond to the front end
  - r: multipart form with "file", and optional "number" and "title"
*/
func (h *Handler) UploadChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var mangaTitle, mediaType string
	err := h.db.QueryRow(`SELECT title, type FROM media_items WHERE id = $1`, id).Scan(&mangaTitle, &mediaType)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "manga not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	if mediaType != "manga" {
		utils.Error(w, http.StatusBadRequest, "chapters can only be added to a manga")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxChapterBytes+(1<<20))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		utils.Error(w, http.StatusRequestEntityTooLarge, "the file is too large or the upload was cut off")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "no file in the upload")
		return
	}
	defer file.Close()
	if !mangafiles.IsArchive(header.Filename) {
		utils.Error(w, http.StatusBadRequest, "chapters must be .cbz or .zip files")
		return
	}

	chapter, ok := chapterFromForm(r.FormValue("number"), header.Filename, mangaTitle)
	if !ok {
		utils.Error(w, http.StatusBadRequest, "couldn't tell the chapter number from the file name, enter it")
		return
	}
	if t := strings.TrimSpace(r.FormValue("title")); t != "" {
		chapter.Title = t
	}

	have, err := downloader.ChapterNumbers(h.db, id)
	if utils.InternalError(w, err) {
		return
	}
	if have[chapter.Value] {
		utils.Error(w, http.StatusConflict, fmt.Sprintf("chapter %s is already in the library", chapter.Number))
		return
	}

	dir, err := downloader.MangaFolder(h.db, id, filepath.Join(os.Getenv("MEDIA_ROOT"), "Manga"), mangaTitle)
	if utils.InternalError(w, err) {
		return
	}
	path := filepath.Join(dir, mangafiles.FileName(chapter.Number, chapter.Title))

	pages, status, err := saveChapter(file, path)
	if err != nil {
		if status == http.StatusInternalServerError {
			logger.Error("saving uploaded chapter %s of %s: %s", chapter.Number, id, err.Error())
			utils.Error(w, status, "internal server error")
			return
		}
		utils.Error(w, status, err.Error())
		return
	}

	var chapterID string
	err = h.db.QueryRow(
		`INSERT INTO manga_chapters (media_item_id, chapter_number, title, file_path, page_count, created_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, NOW())
		RETURNING id`,
		id, chapter.Value, chapter.Title, path, pages,
	).Scan(&chapterID)
	if err != nil {
		os.Remove(path)
		utils.InternalError(w, err)
		return
	}

	logger.Info("added chapter %s of %s (%d pages) from an upload", chapter.Number, mangaTitle, pages)
	utils.JSON(w, map[string]any{
		"id":             chapterID,
		"chapter_number": chapter.Value,
		"title":          chapter.Title,
		"page_count":     pages,
	}, http.StatusCreated)
}

/*
Function:	chapterFromForm
Purpose:	The chapter number typed in the form when there is one, otherwise the one in the
file name
*/
func chapterFromForm(number string, fileName string, series string) (mangafiles.Chapter, bool) {
	number = strings.TrimSpace(number)
	if number == "" {
		return mangafiles.ParseChapter(fileName, series)
	}
	v, err := strconv.ParseFloat(number, 64)
	if err != nil || v < 0 {
		return mangafiles.Chapter{}, false
	}
	// written the way ParseChapter writes numbers, "098" -> "98"
	return mangafiles.ParseChapter("Chapter "+number, "")
}

/*
Function:	saveChapter
Purpose:	Copy an uploaded archive to path, checking it opens and has page images. It is
written to a temporary file first, so a bad upload never takes the chapter's place
Returns:	the page count, and on failure an HTTP status and an error to show the user
*/
func saveChapter(src io.Reader, path string) (int, int, error) {
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return 0, http.StatusInternalServerError, err
	}
	if _, err := io.Copy(f, src); err != nil {
		f.Close()
		os.Remove(tmp)
		return 0, http.StatusInternalServerError, err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return 0, http.StatusInternalServerError, err
	}

	zr, err := zip.OpenReader(tmp)
	if err != nil {
		os.Remove(tmp)
		return 0, http.StatusBadRequest, fmt.Errorf("the file isn't a readable CBZ/ZIP archive")
	}
	pages := len(mangafiles.Pages(&zr.Reader))
	zr.Close()
	if pages == 0 {
		os.Remove(tmp)
		return 0, http.StatusBadRequest, fmt.Errorf("the archive has no page images in it")
	}

	// a file left on disk without a library entry is replaced
	os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return 0, http.StatusInternalServerError, err
	}
	return pages, 0, nil
}

/*
Function:	CheckForChapters
Purpose:	Admin: start a download job that fetches the chapters MangaDex has for a manga
that the library doesn't, for picking up new releases or filling gaps. Progress shows on
the Downloads page like any other job

Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) CheckForChapters(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := auth.GetUser(r)

	var mediaType string
	var externalID, externalSource sql.NullString
	err := h.db.QueryRow(
		`SELECT type, external_id, external_source FROM media_items WHERE id = $1`, id,
	).Scan(&mediaType, &externalID, &externalSource)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "manga not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	if mediaType != "manga" {
		utils.Error(w, http.StatusBadRequest, "only manga can be checked for new chapters")
		return
	}
	if externalSource.String != "mangadex" || externalID.String == "" {
		utils.Error(w, http.StatusBadRequest, "this manga isn't linked to MangaDex")
		return
	}

	var running bool
	err = h.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM download_jobs WHERE media_item_id = $1 AND status IN ('queued', 'downloading'))`, id,
	).Scan(&running)
	if utils.InternalError(w, err) {
		return
	}
	if running {
		utils.Error(w, http.StatusConflict, "this manga is already downloading")
		return
	}

	// a job belongs to a request: reuse the manga's latest one, or record one for this check
	var requestID string
	err = h.db.QueryRow(
		`SELECT id FROM download_requests WHERE media_item_id = $1 ORDER BY requested_at DESC LIMIT 1`, id,
	).Scan(&requestID)
	if err == sql.ErrNoRows {
		err = h.db.QueryRow(
			`INSERT INTO download_requests (requested_by, media_item_id, source_url, status, auto_approved, resolved_at)
			VALUES ($1, $2, '', 'approved', TRUE, NOW())
			RETURNING id`,
			user.UserID, id,
		).Scan(&requestID)
	}
	if utils.InternalError(w, err) {
		return
	}

	ext := externalID.String
	jobID, err := downloader.Dispatch(h.db, requestID, id, "", "manga", &ext)
	if err != nil {
		logger.Error("checking %s for new chapters: %s", id, err.Error())
		utils.Error(w, http.StatusInternalServerError, "could not start the check")
		return
	}
	utils.JSON(w, map[string]string{"id": jobID}, http.StatusCreated)
}
