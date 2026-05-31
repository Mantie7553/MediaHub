package sync

import (
	"database/sql"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

type Handler struct {
	db     *sql.DB
	sonarr *arr.ArrClient
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:     db,
		sonarr: arr.NewArrClient("SONARR_URL", "SONARR_API_KEY"),
	}
}

func (h *Handler) SyncSonar(w http.ResponseWriter, r *http.Request) {
	series, err := h.sonarr.GetAllSeries()
	if utils.InternalError(w, err) {
		return
	}

	for _, s := range series {
		var mediaItemID string
		err := h.db.QueryRow(
			`SELECT id FROM media_items WHERE title = $1 AND type = 'anime'`,
			s.Title,
		).Scan(&mediaItemID)

		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			logger.Error("failed to find media item for %s : %s", s.Title, err.Error())
		}

		_, err = h.db.Exec(
			`INSERT INTO sonarr_items (media_item_id, sonarr_series_id) 
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO UPDATE SET
			sonarr_series_id = EXCLUDED.sonarr_series_id,
			last_synced_at = NOW()`,
			mediaItemID, s.ID,
		)
		if err != nil {
			logger.Error("failed to update or insert sonarr_items for %s : %s", s.Title, err.Error())
			continue
		}

		episodes, err := h.sonarr.GetEpisodes(s.ID)
		if err != nil {
			logger.Error("failed to get episodes fro %s : %s", s.Title, err.Error())
			continue
		}

		for _, ep := range episodes {
			if !ep.HasFile {
				continue
			}

			filePath, err := h.sonarr.GetEpisodeFilePath(ep.EpisodeFileID)
			if err != nil {
				logger.Error("failed to get file path for episode %d : %s", ep.ID, err.Error())
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
				logger.Error("failed to update or insert episode %d : %s", ep.ID, err.Error())
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}
