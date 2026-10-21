package deletion

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/covers"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

/*
Permanent deletion of library content (admin only). The order makes a failed deletion safe to
try again: nothing is touched while a download runs, Sonarr/Radarr go first (a failure there
stops everything), then the files, and the database rows only once the files are gone, since
files left behind would be imported again by the next library sync
*/

type seriesDeleter interface {
	DeleteSeries(seriesID int) error
}

type movieDeleter interface {
	DeleteMovie(movieID int) error
}

type Handler struct {
	db       *sql.DB
	sonarr   seriesDeleter
	episodes episodeRemover
	radarr   movieDeleter
}

func NewHandler(db *sql.DB) *Handler {
	sonarr := arr.NewArrClient("SONARR_URL", "SONARR_API_KEY")
	return &Handler{
		db:       db,
		sonarr:   sonarr,
		episodes: sonarr,
		radarr:   arr.NewArrClient("RADARR_URL", "RADARR_API_KEY"),
	}
}

// what one deletion removes
type target struct {
	ids        []string // media items
	albumID    string   // set when deleting a whole album
	title      string
	kind       string   // libraryFolder type
	transcodes []string // temp folder prefixes
}

// an error with the HTTP status and the message the admin sees
type deleteError struct {
	status  int
	message string
}

func (e *deleteError) Error() string { return e.message }

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// every table that records a title's files, so the queries below don't depend on the type
const filesUnion = `
	SELECT media_item_id, file_path FROM episodes
	UNION ALL SELECT media_item_id, file_path FROM movie_metadata
	UNION ALL SELECT media_item_id, file_path FROM manga_chapters
	UNION ALL SELECT media_item_id, file_path FROM light_novel_volumes
	UNION ALL SELECT media_item_id, file_path FROM music_metadata`

/*
Function:	DeleteMedia
Purpose:	permanently delete an anime, movie, manga, light novel or music track: its Sonarr /
            Radarr entry, files and database rows (lists and progress of every user included)
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !uuidRe.MatchString(id) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	var mediaType, title string
	err := h.db.QueryRow(`SELECT type, title FROM media_items WHERE id = $1`, id).Scan(&mediaType, &title)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	t := target{ids: []string{id}, title: title, kind: mediaType}
	switch mediaType {
	case "anime":
		episodeIDs, err := h.strings(`SELECT id FROM episodes WHERE media_item_id = $1`, id)
		if utils.InternalError(w, err) {
			return
		}
		for _, e := range episodeIDs {
			t.transcodes = append(t.transcodes, "episode-"+e)
		}
	case "movie":
		t.transcodes = []string{"movie-" + id}
	case "music_album":
		// albums are deleted through DeleteAlbum, a stray item of this type is a single file
		t.kind = "music_track"
	}

	if h.respondError(w, h.checkNoDownload(t)) || h.respondError(w, h.removeFromArr(id, mediaType)) {
		return
	}
	h.remove(w, r, t)
}

/*
Function:	DeleteAlbum
Purpose:	permanently delete an album: every track's file and rows, then the album itself
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	if !uuidRe.MatchString(albumID) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	var title string
	err := h.db.QueryRow(`SELECT title FROM albums WHERE id = $1`, albumID).Scan(&title)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	trackIDs, err := h.strings(`SELECT media_item_id FROM music_metadata WHERE album_id = $1`, albumID)
	if utils.InternalError(w, err) {
		return
	}

	t := target{ids: trackIDs, albumID: albumID, title: title, kind: "music_album"}
	if h.respondError(w, h.checkNoDownload(t)) {
		return
	}
	h.remove(w, r, t)
}

/*
Function:	remove
Purpose:	delete a target's files, then its database rows in one transaction
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
  - t: what to delete
*/
func (h *Handler) remove(w http.ResponseWriter, r *http.Request, t target) {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	if mediaRoot == "" {
		utils.Error(w, http.StatusInternalServerError, "MEDIA_ROOT is not set, so files can't be deleted safely")
		return
	}

	paths, err := h.strings(`SELECT file_path FROM (`+filesUnion+`) f
		WHERE media_item_id = ANY($1::uuid[]) AND file_path IS NOT NULL AND file_path <> ''`, pq.Array(t.ids))
	if utils.InternalError(w, err) {
		return
	}

	folderName, depth := libraryFolder(t.kind)
	failed := deleteFiles(mediaRoot, filepath.Join(mediaRoot, folderName), paths, depth, func(folder string) (bool, error) {
		return h.otherFilesIn(folder, t.ids)
	})
	if len(failed) > 0 {
		logger.Error("deleting %q: could not remove %s", t.title, strings.Join(failed, ", "))
		utils.Error(w, http.StatusInternalServerError,
			fmt.Sprintf("couldn't delete %d file(s), nothing else was removed: %s", len(failed), strings.Join(failed, ", ")))
		return
	}
	removeTranscodes(mediaRoot, t.transcodes)
	for _, id := range t.ids {
		if err := covers.Remove(id); err != nil {
			logger.Warn("removing the uploaded cover of %s: %v", id, err)
		}
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	// download_jobs has no ON DELETE rule, its rows would block deleting the item. Everything
	// else (metadata, episodes, chapters, lists, progress) cascades from media_items
	if _, err := tx.Exec(`DELETE FROM download_jobs WHERE media_item_id = ANY($1::uuid[])`, pq.Array(t.ids)); utils.InternalError(w, err) {
		return
	}
	// requests keep their history, but deleting the item clears their link to it and a request
	// must name an item, an album or a title (download_requests CHECK). Copy the title over first
	if _, err := tx.Exec(
		`UPDATE download_requests r SET title_override = COALESCE(r.title_override, m.title, $2)
		FROM media_items m WHERE r.media_item_id = m.id AND m.id = ANY($1::uuid[])`,
		pq.Array(t.ids), t.title,
	); utils.InternalError(w, err) {
		return
	}
	if t.albumID != "" {
		if _, err := tx.Exec(
			`UPDATE download_requests SET title_override = COALESCE(title_override, $2) WHERE album_id = $1`,
			t.albumID, t.title,
		); utils.InternalError(w, err) {
			return
		}
	}
	if _, err := tx.Exec(`DELETE FROM media_items WHERE id = ANY($1::uuid[])`, pq.Array(t.ids)); utils.InternalError(w, err) {
		return
	}
	if t.albumID != "" {
		if _, err := tx.Exec(`DELETE FROM albums WHERE id = $1`, t.albumID); utils.InternalError(w, err) {
			return
		}
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}

	logger.Info("user %s deleted %q (%s, %d file(s))", auth.GetUser(r).UserID, t.title, t.kind, len(paths))
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	checkNoDownload
Purpose:	refuse while a download for the target is queued or running, its tool would keep
            writing into the folders being removed
Params:
  - t: what to delete
*/
func (h *Handler) checkNoDownload(t target) error {
	var active bool
	err := h.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM download_jobs
		WHERE media_item_id = ANY($1::uuid[]) AND status IN ('queued', 'downloading'))`,
		pq.Array(t.ids),
	).Scan(&active)
	if err != nil {
		return err
	}
	if active {
		return &deleteError{http.StatusConflict, "Still downloading, try again when it finishes"}
	}
	return nil
}

/*
Function:	removeFromArr
Purpose:	delete the Sonarr series or Radarr movie (and its files) behind an item, so it is not
            downloaded again. Skipped when another item uses the same series or movie
Params:
  - id: the media item
  - mediaType: its type, only anime and movies are managed by Sonarr / Radarr
*/
func (h *Handler) removeFromArr(id, mediaType string) error {
	var table, column, app string
	switch mediaType {
	case "anime":
		table, column, app = "sonarr_items", "sonarr_series_id", "Sonarr"
	case "movie":
		table, column, app = "radarr_items", "radarr_movie_id", "Radarr"
	default:
		return nil
	}

	var arrID int
	var shared bool
	err := h.db.QueryRow(
		`SELECT a.`+column+`, EXISTS (SELECT 1 FROM `+table+` b WHERE b.`+column+` = a.`+column+` AND b.media_item_id <> a.media_item_id)
		FROM `+table+` a WHERE a.media_item_id = $1`,
		id,
	).Scan(&arrID, &shared)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if shared {
		logger.Warn("%s id %d is shared with another item, only deleting this item's files", app, arrID)
		return nil
	}

	if mediaType == "anime" {
		err = h.sonarr.DeleteSeries(arrID)
	} else {
		err = h.radarr.DeleteMovie(arrID)
	}
	if err != nil {
		logger.Error("%s delete of %d failed: %v", app, arrID, err)
		return &deleteError{http.StatusBadGateway, fmt.Sprintf("%s couldn't delete it, nothing was removed: %v", app, err)}
	}
	return nil
}

/*
Function:	otherFilesIn
Purpose:	whether any item other than the ones being deleted has files inside a folder
Params:
  - folder: the folder to check
  - ids: the items being deleted
*/
func (h *Handler) otherFilesIn(folder string, ids []string) (bool, error) {
	var found bool
	err := h.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM (`+filesUnion+`) f
		WHERE NOT (media_item_id = ANY($2::uuid[])) AND file_path LIKE $1 ESCAPE '\')`,
		utils.LikePrefix(folder+string(filepath.Separator)), pq.Array(ids),
	).Scan(&found)
	return found, err
}

/*
Function:	respondError
Purpose:	send err to the client: a deleteError with its status and message, anything else as a
            500. Returns false when there is no error
Params:
  - w: http response writer to respond to the front end
  - err: the error, or nil
*/
func (h *Handler) respondError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var de *deleteError
	if errors.As(err, &de) {
		utils.Error(w, de.status, de.message)
		return true
	}
	return utils.InternalError(w, err)
}

// strings runs a query that returns one text column
func (h *Handler) strings(query string, args ...any) ([]string, error) {
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// likePrefix escapes LIKE wildcards in a path and matches everything under it
func likePrefix(path string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(path) + "%"
}
