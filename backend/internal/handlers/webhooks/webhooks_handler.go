package webhooks

import (
	"database/sql"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/lib/pq"
)

// Syncer syncs one series or movie the way the full library sync does, creating its title when
// MediaHub doesn't have it yet
type Syncer interface {
	SyncSeriesByID(seriesID int) error
	SyncMovieByID(movieID int) error
}

type Handler struct {
	db     *sql.DB
	syncer Syncer
	// runs a sync; in the background, so Sonarr / Radarr aren't kept waiting on AniList
	run func(func())
}

func NewHandler(db *sql.DB, syncer Syncer) *Handler {
	return &Handler{db: db, syncer: syncer, run: func(f func()) { go f() }}
}

// what a webhook event asks MediaHub to do
type action int

const (
	ignore     action = iota
	syncItem          // sync the series or movie from Sonarr / Radarr
	clearFiles        // its files (some episodes, or the movie's) were deleted
	forgetItem        // the series or movie was removed from Sonarr / Radarr
)

/*
Function:	actionFor
Purpose:	what to do for a Sonarr or Radarr event. A file deleted to make way for an upgrade is
ignored, the Download event that follows brings the new one
Params:
  - eventType: the payload's eventType
  - deleteReason: the payload's deleteReason, for file deletes
*/
func actionFor(eventType, deleteReason string) action {
	switch eventType {
	case "Download", "Rename":
		return syncItem
	case "EpisodeFileDelete", "MovieFileDelete":
		if deleteReason == "upgrade" {
			return ignore
		}
		return clearFiles
	case "SeriesDelete", "MovieDelete":
		return forgetItem
	}
	return ignore
}

/*
Function:	SonarrWebhook
Purpose:	Handle incoming webhook events from Sonarr. Titles and users' progress are never deleted:
episodes whose file is gone stay, with no file
Params:
  - w: http response writer
  - r: the request from Sonarr
*/
func (h *Handler) SonarrWebhook(w http.ResponseWriter, r *http.Request) {
	var payload sonarrPayload
	if !utils.DecodeJSON(w, r, &payload) {
		return
	}
	seriesID := payload.Series.ID

	switch actionFor(payload.EventType, payload.DeleteReason) {
	case syncItem:
		h.run(func() {
			if err := h.syncer.SyncSeriesByID(seriesID); err != nil {
				logger.Error("webhook sync of Sonarr series %d failed: %s", seriesID, err.Error())
			}
		})
	case clearFiles:
		ids := make([]int64, 0, len(payload.Episodes))
		for _, ep := range payload.Episodes {
			ids = append(ids, int64(ep.ID))
		}
		_, err := h.db.Exec(
			`UPDATE episodes SET file_path = NULL, updated_at = NOW()
			WHERE sonarr_id = ANY($1) AND media_item_id IN
				(SELECT media_item_id FROM sonarr_items WHERE sonarr_series_id = $2)`,
			pq.Array(ids), seriesID,
		)
		if utils.InternalError(w, err) {
			return
		}
	case forgetItem:
		// the link goes too, so requesting the title again adds it to Sonarr again
		_, err := h.db.Exec(
			`WITH gone AS (DELETE FROM sonarr_items WHERE sonarr_series_id = $1 RETURNING media_item_id)
			UPDATE episodes SET file_path = NULL, updated_at = NOW()
			WHERE sonarr_id IS NOT NULL AND media_item_id IN (SELECT media_item_id FROM gone)`,
			seriesID,
		)
		if utils.InternalError(w, err) {
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

/*
Function:	RadarrWebhook
Purpose:	Handle incoming webhook events from Radarr. Titles and users' progress are never deleted:
a movie whose file is gone stays, with no file
Params:
  - w: http response writer
  - r: the request from Radarr
*/
func (h *Handler) RadarrWebhook(w http.ResponseWriter, r *http.Request) {
	var payload radarrPayload
	if !utils.DecodeJSON(w, r, &payload) {
		return
	}
	movieID := payload.Movie.ID

	var query string
	switch actionFor(payload.EventType, payload.DeleteReason) {
	case syncItem:
		h.run(func() {
			if err := h.syncer.SyncMovieByID(movieID); err != nil {
				logger.Error("webhook sync of Radarr movie %d failed: %s", movieID, err.Error())
			}
		})
	case clearFiles:
		query = `UPDATE movie_metadata SET file_path = NULL WHERE media_item_id IN
			(SELECT media_item_id FROM radarr_items WHERE radarr_movie_id = $1)`
	case forgetItem:
		query = `WITH gone AS (DELETE FROM radarr_items WHERE radarr_movie_id = $1 RETURNING media_item_id)
			UPDATE movie_metadata SET file_path = NULL WHERE media_item_id IN (SELECT media_item_id FROM gone)`
	}
	if query != "" {
		if _, err := h.db.Exec(query, movieID); utils.InternalError(w, err) {
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
