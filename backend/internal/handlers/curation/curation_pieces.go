package curation

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/mangafiles"
	"github.com/Mantie7553/MediaHub/backend/internal/novelfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

/*
Function:	EditEpisode
Purpose:	Admin: rename an episode. The name is kept through Sonarr syncs; clearing it goes
back to Sonarr's title on the next sync
Params:
  - w: http response writer
  - r: JSON { title }
*/
func (h *Handler) EditEpisode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req editPieceRequest
	if !decodePiece(w, r, id, &req) {
		return
	}
	if req.ChapterNumber.Set || req.VolumeNumber.Set || !req.Title.Set {
		utils.Error(w, http.StatusBadRequest, "only an episode's title can be changed")
		return
	}
	title, ok := cleanText(req.Title.Value, maxTitleRunes)
	if !ok {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("the title can be at most %d characters", maxTitleRunes))
		return
	}

	res, err := h.db.Exec(
		`UPDATE episodes SET title = $1, title_locked = $2, updated_at = NOW() WHERE id = $3`,
		title, title != nil, id,
	)
	if utils.InternalError(w, err) {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		utils.Error(w, http.StatusNotFound, "episode not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	EditChapter
Purpose:	Admin: rename or renumber a chapter. A renumbered chapter's file is renamed to
match ("[0099] Chapter 99.cbz"), so its old number is free for a real chapter 98 and the
library sync reads the same number from it
Params:
  - w: http response writer
  - r: JSON { chapter_number, title }
*/
func (h *Handler) EditChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req editPieceRequest
	if !decodePiece(w, r, id, &req) {
		return
	}
	if req.VolumeNumber.Set {
		utils.Error(w, http.StatusBadRequest, "chapters have a chapter_number")
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	var mangaID string
	var number float64
	var title, filePath sql.NullString
	err = tx.QueryRow(
		`SELECT media_item_id, chapter_number, title, file_path FROM manga_chapters WHERE id = $1 FOR UPDATE`, id,
	).Scan(&mangaID, &number, &title, &filePath)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "chapter not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	newTitle := nullable(title)
	if req.Title.Set {
		var ok bool
		if newTitle, ok = cleanText(req.Title.Value, maxTitleRunes); !ok {
			utils.Error(w, http.StatusBadRequest, fmt.Sprintf("the title can be at most %d characters", maxTitleRunes))
			return
		}
	}

	newNumber := number
	renamed := ""
	if req.ChapterNumber.Set {
		chapter, ok := parseChapterNumber(req.ChapterNumber.Value)
		if !ok {
			utils.Error(w, http.StatusBadRequest, "the chapter number must be a number like 98 or 52.5")
			return
		}
		newNumber = chapter.Value
		if newNumber != number {
			if status, msg := h.renumberAllowed(tx, mangaID,
				`SELECT EXISTS (SELECT 1 FROM manga_chapters WHERE media_item_id = $1 AND chapter_number = $2)`, newNumber,
				fmt.Sprintf("chapter %s is already in the library", chapter.Number)); status != 0 {
				utils.Error(w, status, msg)
				return
			}
			if filePath.Valid && filePath.String != "" {
				renamed = filepath.Join(filepath.Dir(filePath.String), mangafiles.FileName(chapter.Number, deref(newTitle)))
			}
		}
	}

	if renamed != "" && !h.renameFile(w, filePath.String, renamed) {
		return
	}
	path := filePath.String
	if renamed != "" {
		path = renamed
	}

	_, err = tx.Exec(
		`UPDATE manga_chapters SET chapter_number = $1, title = $2, file_path = NULLIF($3, '') WHERE id = $4`,
		newNumber, newTitle, path, id,
	)
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		h.undoRename(filePath.String, renamed)
		utils.InternalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	EditVolume
Purpose:	Admin: rename or renumber a light novel volume. A renumbered volume's file is
renamed to match ("X Volume 4.epub") when its name has the number in it
Params:
  - w: http response writer
  - r: JSON { volume_number, title }
*/
func (h *Handler) EditVolume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req editPieceRequest
	if !decodePiece(w, r, id, &req) {
		return
	}
	if req.ChapterNumber.Set {
		utils.Error(w, http.StatusBadRequest, "volumes have a volume_number")
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	var novelID string
	var number int
	var title, filePath sql.NullString
	err = tx.QueryRow(
		`SELECT media_item_id, volume_number, title, file_path FROM light_novel_volumes WHERE id = $1 FOR UPDATE`, id,
	).Scan(&novelID, &number, &title, &filePath)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "volume not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	newTitle := nullable(title)
	if req.Title.Set {
		var ok bool
		if newTitle, ok = cleanText(req.Title.Value, maxTitleRunes); !ok {
			utils.Error(w, http.StatusBadRequest, fmt.Sprintf("the title can be at most %d characters", maxTitleRunes))
			return
		}
	}

	newNumber := number
	renamed := ""
	if req.VolumeNumber.Set {
		if req.VolumeNumber.Value == nil || *req.VolumeNumber.Value < 0 || *req.VolumeNumber.Value > maxCount {
			utils.Error(w, http.StatusBadRequest, "the volume number must be a whole number")
			return
		}
		newNumber = *req.VolumeNumber.Value
		if newNumber != number {
			if status, msg := h.renumberAllowed(tx, novelID,
				`SELECT EXISTS (SELECT 1 FROM light_novel_volumes WHERE media_item_id = $1 AND volume_number = $2)`, newNumber,
				fmt.Sprintf("volume %d is already in the library", newNumber)); status != 0 {
				utils.Error(w, status, msg)
				return
			}
			if filePath.Valid && filePath.String != "" {
				name := novelfiles.Renumber(filepath.Base(filePath.String), newNumber)
				if name != filepath.Base(filePath.String) {
					renamed = filepath.Join(filepath.Dir(filePath.String), name)
				}
			}
		}
	}

	if renamed != "" && !h.renameFile(w, filePath.String, renamed) {
		return
	}
	path := filePath.String
	if renamed != "" {
		path = renamed
	}

	_, err = tx.Exec(
		`UPDATE light_novel_volumes SET volume_number = $1, title = $2, file_path = NULLIF($3, '') WHERE id = $4`,
		newNumber, newTitle, path, id,
	)
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		h.undoRename(filePath.String, renamed)
		utils.InternalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	renumberAllowed
Purpose:	whether a chapter or volume can take a new number: not while a download is writing
into the title's folder, and not one another chapter or volume already has
Returns:	0, or the HTTP status and message to refuse with
*/
func (h *Handler) renumberAllowed(tx *sql.Tx, mediaItemID, takenQuery string, number any, takenMessage string) (int, string) {
	var downloading, taken bool
	err := tx.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM download_jobs WHERE media_item_id = $1 AND status IN ('queued', 'downloading'))`,
		mediaItemID,
	).Scan(&downloading)
	if err == nil {
		err = tx.QueryRow(takenQuery, mediaItemID, number).Scan(&taken)
	}
	switch {
	case err != nil:
		logger.Error("checking a renumber of %s: %v", mediaItemID, err)
		return http.StatusInternalServerError, "internal server error"
	case downloading:
		return http.StatusConflict, "still downloading, try again when it finishes"
	case taken:
		return http.StatusConflict, takenMessage
	}
	return 0, ""
}

/*
Function:	renameFile
Purpose:	rename a chapter or volume file inside the library, never over another file.
Responds with the problem and returns false when it can't
*/
func (h *Handler) renameFile(w http.ResponseWriter, from, to string) bool {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	if mediaRoot == "" || !utils.IsWithin(mediaRoot, from) || !utils.IsWithin(mediaRoot, to) {
		utils.Error(w, http.StatusConflict, "the file is outside the media folder, so it can't be renamed")
		return false
	}
	if _, err := os.Stat(from); os.IsNotExist(err) {
		utils.Error(w, http.StatusConflict, "the file is missing from the media folder: "+filepath.Base(from))
		return false
	}
	if _, err := os.Stat(to); err == nil {
		utils.Error(w, http.StatusConflict, "there's already a file named "+filepath.Base(to)+" in the folder")
		return false
	}
	if err := os.Rename(from, to); err != nil {
		logger.Error("renaming %s to %s: %v", from, to, err)
		utils.Error(w, http.StatusInternalServerError, "couldn't rename the file")
		return false
	}
	return true
}

// undoRename puts a renamed file back after the database refused the change
func (h *Handler) undoRename(original, renamed string) {
	if renamed == "" {
		return
	}
	if err := os.Rename(renamed, original); err != nil {
		logger.Error("putting %s back as %s: %v", renamed, original, err)
	}
}

/*
Function:	parseChapterNumber
Purpose:	a chapter number typed by an admin, written the way chapter files write it
("098" -> "98")
*/
func parseChapterNumber(s *string) (mangafiles.Chapter, bool) {
	if s == nil {
		return mangafiles.Chapter{}, false
	}
	n := strings.TrimSpace(*s)
	if v, err := strconv.ParseFloat(n, 64); err != nil || v < 0 || v > maxCount {
		return mangafiles.Chapter{}, false
	}
	return mangafiles.ParseChapter("Chapter "+n, "")
}

// decodePiece reads an edit of one piece, answering 404 for an id that isn't a UUID
func decodePiece(w http.ResponseWriter, r *http.Request, id string, req *editPieceRequest) bool {
	if !utils.IsUUID(id) {
		utils.Error(w, http.StatusNotFound, "not found")
		return false
	}
	return utils.DecodeJSON(w, r, req)
}

func nullable(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
