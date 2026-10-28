package deletion

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

/*
Removing one episode, chapter or volume from a title, in the same order as whole titles: not
while a download runs, Sonarr first, then the file, then the row (everyone's progress for it
goes with it). Tracks are titles of their own and go through DeleteMedia
*/

// what Sonarr needs to forget one episode
type episodeRemover interface {
	UnmonitorEpisode(episodeID int) error
	EpisodeFileID(episodeID int) (int, error)
	DeleteEpisodeFile(episodeFileID int) error
}

// a piece of a title to remove
type piece struct {
	table      string // episodes, manga_chapters or light_novel_volumes
	label      string // for messages, "episode"
	kind       string // libraryFolder type of its title
	transcodes []string
}

var (
	episodePiece = piece{table: "episodes", label: "episode", kind: "anime"}
	chapterPiece = piece{table: "manga_chapters", label: "chapter", kind: "manga"}
	volumePiece  = piece{table: "light_novel_volumes", label: "volume", kind: "light_novel"}
)

/*
Function:	DeleteEpisode
Purpose:	Admin: remove one episode and its file. Sonarr stops monitoring it first, so it
isn't downloaded again, and deletes the file itself when it has it
*/
func (h *Handler) DeleteEpisode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p := episodePiece
	p.transcodes = []string{"episode-" + id}
	h.removePiece(w, r, id, p, func(sonarrID sql.NullInt64) error {
		if !sonarrID.Valid || sonarrID.Int64 == 0 {
			return nil
		}
		return h.removeFromSonarr(int(sonarrID.Int64))
	})
}

// DeleteChapter: Admin: remove one manga chapter and its file
func (h *Handler) DeleteChapter(w http.ResponseWriter, r *http.Request) {
	h.removePiece(w, r, chi.URLParam(r, "id"), chapterPiece, nil)
}

// DeleteVolume: Admin: remove one light novel volume and its file
func (h *Handler) DeleteVolume(w http.ResponseWriter, r *http.Request) {
	h.removePiece(w, r, chi.URLParam(r, "id"), volumePiece, nil)
}

/*
Function:	removePiece
Purpose:	delete one piece's file and row
Params:
  - id: the piece's id
  - p: what kind of piece it is
  - beforeFiles: run before the file is touched, with the piece's sonarr_id for episodes;
    an error stops everything. May be nil
*/
func (h *Handler) removePiece(w http.ResponseWriter, r *http.Request, id string, p piece, beforeFiles func(sql.NullInt64) error) {
	if !utils.IsUUID(id) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	mediaRoot := os.Getenv("MEDIA_ROOT")
	if mediaRoot == "" {
		utils.Error(w, http.StatusInternalServerError, "MEDIA_ROOT is not set, so files can't be deleted safely")
		return
	}

	var mediaItemID string
	var filePath sql.NullString
	var sonarrID sql.NullInt64
	query := `SELECT media_item_id, file_path, NULL::int FROM ` + p.table + ` WHERE id = $1`
	if p.table == "episodes" {
		query = `SELECT media_item_id, file_path, sonarr_id FROM episodes WHERE id = $1`
	}
	err := h.db.QueryRow(query, id).Scan(&mediaItemID, &filePath, &sonarrID)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, p.label+" not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	if h.respondError(w, h.checkNoDownload(target{ids: []string{mediaItemID}})) {
		return
	}
	if beforeFiles != nil && h.respondError(w, beforeFiles(sonarrID)) {
		return
	}

	if path := filePath.String; filePath.Valid && path != "" {
		if !utils.IsWithin(mediaRoot, path) {
			utils.Error(w, http.StatusInternalServerError, fmt.Sprintf("the file is outside the media folder, nothing was removed: %s", path))
			return
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			logger.Error("deleting %s: %v", path, err)
			utils.Error(w, http.StatusInternalServerError, fmt.Sprintf("couldn't delete the file, nothing else was removed: %v", err))
			return
		}
		cache.Disk().Forget(path)
		folder, _ := libraryFolder(p.kind)
		pruneEmptyDirs(filepath.Join(mediaRoot, folder), filepath.Dir(path))
	}
	removeTranscodes(mediaRoot, p.transcodes)

	if _, err := h.db.Exec(`DELETE FROM `+p.table+` WHERE id = $1`, id); utils.InternalError(w, err) {
		return
	}
	logger.Info("user %s deleted a %s of %s (%s)", auth.GetUser(r).UserID, p.label, mediaItemID, filepath.Base(filePath.String))
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	removeFromSonarr
Purpose:	stop Sonarr monitoring an episode, then have it delete the episode's file. A
failure stops the deletion with 502
*/
func (h *Handler) removeFromSonarr(episodeID int) error {
	if h.episodes == nil {
		return nil
	}
	fail := func(err error) error {
		logger.Error("Sonarr removal of episode %d failed: %v", episodeID, err)
		return &deleteError{http.StatusBadGateway, fmt.Sprintf("Sonarr couldn't remove it, nothing was deleted: %v", err)}
	}
	if err := h.episodes.UnmonitorEpisode(episodeID); err != nil {
		return fail(err)
	}
	fileID, err := h.episodes.EpisodeFileID(episodeID)
	if err != nil {
		return fail(err)
	}
	if fileID == 0 {
		return nil
	}
	if err := h.episodes.DeleteEpisodeFile(fileID); err != nil {
		return fail(err)
	}
	return nil
}
