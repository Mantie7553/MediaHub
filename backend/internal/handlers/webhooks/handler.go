package webhooks

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	SonarrWebhook
Purpose:	Handle incoming webhook events from Sonarr
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) SonarrWebhook(w http.ResponseWriter, r *http.Request) {
	var payload sonarrPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// ignore test events and anything that isn't a download
	if payload.EventType != "Download" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// find the media_item_id that matches this sonarr series
	var mediaItemID string
	err := h.db.QueryRow(
		`SELECT media_item_id FROM sonarr_items WHERE sonarr_series_id = $1`,
		payload.Series.ID,
	).Scan(&mediaItemID)

	if err == sql.ErrNoRows {
		// sonarr series not linked to a media item, ignore
		w.WriteHeader(http.StatusOK)
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	// sync each episode in the payload
	sonarrClient := arr.NewArrClient("SONARR_URL", "SONARR_API_KEY")
	for _, ep := range payload.Episodes {
		if ep.EpisodeFileID == 0 {
			continue
		}

		filePath, err := sonarrClient.GetEpisodeFilePath(ep.EpisodeFileID)
		if err != nil {
			logger.Error("failed to get file path for episode file %d: %s", ep.EpisodeFileID, err.Error())
			continue
		}

		_, err = h.db.Exec(
			`INSERT INTO episodes (media_item_id, season_number, episode_number, title, file_path, sonarr_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (media_item_id, season_number, episode_number) DO UPDATE SET
			file_path = EXCLUDED.file_path,
			title = EXCLUDED.title,
			updated_at = NOW()`,
			mediaItemID, ep.SeasonNumber, ep.EpisodeNumber, ep.Title, filePath, ep.ID,
		)
		if err != nil {
			logger.Error("failed to upsert episode %d: %s", ep.ID, err.Error())
		}
	}

	w.WriteHeader(http.StatusOK)
}
