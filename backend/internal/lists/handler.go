package lists

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	var req rateRequest
	user := auth.GetUser(r)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.MediaItemId == "" || req.Status == "" {
		utils.Error(w, http.StatusBadRequest, "status is required")
		return
	}

	var statusId string
	err := h.db.QueryRow(
		`INSERT INTO user_media_status (user_id, media_item_id, status, rating, notes)
 		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id`,
		user.UserID, req.MediaItemId, req.Status, req.Rating, utils.NullString(req.Notes),
	).Scan(&statusId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			utils.Error(w, http.StatusConflict, "status already exists")
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	utils.JSON(w, map[string]string{"id": statusId}, http.StatusCreated)
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)

	items := []UserMediaEntry{}

	queryString := `SELECT ums.id, ums.status, ums.rating, ums.notes, ums.updated_at,
	mi.id, mi.type, mi.title, mi.cover_image_url, mi.release_date,
    mm.artist,
	uap.episodes_watched, uap.season_id,
	ans.season_number, ans.episode_count
	FROM user_media_status ums
	JOIN media_items mi ON mi.id = ums.media_item_id
	LEFT JOIN music_metadata mm ON mm.media_item_id = mi.id
	LEFT JOIN user_anime_progress uap ON uap.media_item_id = mi.id AND uap.user_id = ums.user_id
	LEFT JOIN anime_seasons ans ON ans.id = uap.season_id
	WHERE ums.user_id = $1`

	rows, err := h.db.Query(queryString, user.UserID)

	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item UserMediaEntry
		err := rows.Scan(
			&item.ID, &item.Status, &item.Rating, &item.Notes, &item.UpdatedAt,
			&item.MediaItemID, &item.MediaType, &item.MediaTitle, &item.CoverImageURL,
			&item.ReleaseDate, &item.Artist, &item.EpisodesWatched, &item.SeasonID,
			&item.SeasonNumber, &item.TotalEpisodes,
		)

		if utils.InternalError(w, err) {
			return
		}
		items = append(items, item)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, items)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	user := auth.GetUser(r)
	entryID := chi.URLParam(r, "id")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var mediaStatusId string
	err := h.db.QueryRow(
		`UPDATE user_media_status 
		SET status = $1, rating = $2, notes = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
		RETURNING id`,
		req.Status, req.Rating, utils.NullString(req.Notes), entryID, user.UserID,
	).Scan(&mediaStatusId)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, map[string]string{"id": mediaStatusId})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	entryID := chi.URLParam(r, "id")

	result, err := h.db.Exec(
		`DELETE FROM user_media_status WHERE id = $1 AND user_id = $2`,
		entryID, user.UserID,
	)

	if utils.InternalError(w, err) {
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	user := auth.GetUser(r)
	entryID := chi.URLParam(r, "id")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.EpisodesWatched < 0 {
		utils.Error(w, http.StatusBadRequest, "episodes watched must be a positive number")
		return
	}

	var progressID string
	var episodesWatched int
	err := h.db.QueryRow(
		`INSERT INTO user_anime_progress (user_id, media_item_id, season_id, episodes_watched, last_watched_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, media_item_id, season_id) 
		DO UPDATE SET episodes_watched = $4, last_watched_at = NOW()
		RETURNING id, episodes_watched`,
		user.UserID, entryID, utils.NullString(req.SeasonID), req.EpisodesWatched,
	).Scan(&progressID, &episodesWatched)

	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, map[string]any{
		"id":               progressID,
		"episodes_watched": episodesWatched,
	})
}
