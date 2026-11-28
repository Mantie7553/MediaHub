package webhooks

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/notify"
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
	// tells users new content arrived, after a Download synced it
	arrived func(table, column string, arrID, added int)
}

func NewHandler(db *sql.DB, syncer Syncer) *Handler {
	h := &Handler{db: db, syncer: syncer, run: func(f func()) { go f() }}
	h.arrived = h.notifyArrived
	return h
}

/*
Function:	newContent
Purpose:	whether an event brings content users haven't had: a Download that isn't an upgrade
*/
func newContent(eventType string, isUpgrade bool) bool {
	return eventType == "Download" && !isUpgrade
}

// what a webhook event asks MediaHub to do
type action int

const (
	ignore     action = iota
	syncItem          // sync the series or movie from Sonarr / Radarr
	clearFiles        // its files (some episodes, or the movie's) were deleted
	forgetItem        // the series or movie was removed from Sonarr / Radarr
	notifyGrab        // something was grabbed to download: the admins are told
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
	case "Grab":
		return notifyGrab
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
	case notifyGrab:
		title, body := payload.Series.Title, grabSummary(payload.Episodes)
		h.run(func() { h.notifyGrabbed("sonarr_items", "sonarr_series_id", seriesID, "anime", title, body) })
	case syncItem:
		h.run(func() {
			if err := h.syncer.SyncSeriesByID(seriesID); err != nil {
				logger.Error("webhook sync of Sonarr series %d failed: %s", seriesID, err.Error())
				return
			}
			if newContent(payload.EventType, payload.IsUpgrade) && h.arrived != nil {
				h.arrived("sonarr_items", "sonarr_series_id", seriesID, max(len(payload.Episodes), 1))
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
	case notifyGrab:
		title := payload.Movie.Title
		h.run(func() {
			h.notifyGrabbed("radarr_items", "radarr_movie_id", movieID, "movie", title, "Grabbed by Radarr")
		})
	case syncItem:
		h.run(func() {
			if err := h.syncer.SyncMovieByID(movieID); err != nil {
				logger.Error("webhook sync of Radarr movie %d failed: %s", movieID, err.Error())
				return
			}
			if newContent(payload.EventType, payload.IsUpgrade) && h.arrived != nil {
				h.arrived("radarr_items", "radarr_movie_id", movieID, 1)
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

/*
Function:	notifyGrabbed
Purpose:	tell the admins Sonarr or Radarr grabbed something. A title MediaHub has opens its
page and shares the one notification its download already has
Params:
  - table, column: where MediaHub links its titles to the app's ids, never user input
  - arrID: the series or movie id in Sonarr / Radarr
  - mediaType: anime or movie
  - title: the name the app gave, for one MediaHub doesn't have yet
  - body: what was grabbed
*/
func (h *Handler) notifyGrabbed(table, column string, arrID int, mediaType, title, body string) {
	var mediaItemID, known sql.NullString
	err := h.db.QueryRow(
		`SELECT mi.id, mi.title FROM `+table+` x JOIN media_items mi ON mi.id = x.media_item_id
		WHERE x.`+column+` = $1 LIMIT 1`,
		arrID,
	).Scan(&mediaItemID, &known)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Warn("looking up %s %d to notify about its grab: %s", column, arrID, err.Error())
	}
	if known.Valid {
		title = known.String
	}
	if title == "" {
		title = "Something"
	}
	key := mediaType + ":" + strconv.Itoa(arrID)
	notify.Grabbed(h.db, mediaItemID.String, mediaType, title, "grab:"+key, body)
}

/*
Function:	notifyArrived
Purpose:	tell users a Sonarr series or Radarr movie got new files
Params:
  - table, column: where MediaHub links its titles to the app's ids, never user input
  - arrID: the series or movie id in Sonarr / Radarr
  - added: how many episodes arrived
*/
func (h *Handler) notifyArrived(table, column string, arrID, added int) {
	rows, err := h.db.Query(`SELECT DISTINCT media_item_id::text FROM `+table+` WHERE `+column+` = $1`, arrID)
	if err != nil {
		logger.Warn("looking up %s %d to notify about its download: %s", column, arrID, err.Error())
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		notify.TitleArrived(h.db, id, added)
	}
}

/*
Function:	grabSummary
Purpose:	what a Sonarr grab is of: "S01E05 grabbed by Sonarr", or how many episodes
*/
func grabSummary(episodes []sonarrEpisode) string {
	switch len(episodes) {
	case 0:
		return "Grabbed by Sonarr"
	case 1:
		return fmt.Sprintf("S%02dE%02d grabbed by Sonarr", episodes[0].SeasonNumber, episodes[0].EpisodeNumber)
	}
	return fmt.Sprintf("%d episodes grabbed by Sonarr", len(episodes))
}
