package lists

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/auth"
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.MediaItemId == "" || req.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	var statusId string
	err := h.db.QueryRow(
		`INSERT INTO user_media_status (user_id, media_item_id, status, rating, notes)
 		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id`,
		user.UserID, req.MediaItemId, req.Status, req.Rating, nullString(req.Notes),
	).Scan(&statusId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			http.Error(w, "status already exists", http.StatusConflict)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": statusId})
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)

	items := []UserMediaEntry{}

	queryString := `SELECT ums.id, ums.status, ums.rating, ums.notes, ums.updated_at,
       mi.id, mi.type, mi.title, mi.cover_image_url
		FROM user_media_status ums
		JOIN media_items mi ON mi.id = ums.media_item_id
		WHERE ums.user_id = $1`

	rows, err := h.db.Query(queryString, user.UserID)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item UserMediaEntry
		err := rows.Scan (
			&item.ID, &item.Status, &item.Rating, &item.Notes, &item.UpdatedAt,
			&item.MediaItemID, &item.MediaType, &item.MediaTitle, &item.CoverImageURL,
		)

		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	user := auth.GetUser(r)
	entryID := chi.URLParam(r, "id")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var mediaStatusId string
	err := h.db.QueryRow(
		`UPDATE user_media_status 
		SET status = $1, rating = $2, notes = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
		RETURNING id`,
		req.Status, req.Rating, nullString(req.Notes), entryID, user.UserID,
	).Scan(&mediaStatusId)

	if err == sql.ErrNoRows {
    http.Error(w, "not found", http.StatusNotFound)
    return
	}

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": mediaStatusId})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	entryID := chi.URLParam(r, "id")

	result, err := h.db.Exec(
		`DELETE FROM user_media_status WHERE id = $1 AND user_id = $2`,
		entryID, user.UserID,
	)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
    	return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "not found", http.StatusNotFound)
    	return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	user := auth.GetUser(r)
	entryID := chi.URLParam(r, "id")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.EpisodesWatched < 0 {
		http.Error(w, "episodes watched must be a positive number", http.StatusBadRequest)
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
		user.UserID, entryID, nullString(req.SeasonID), req.EpisodesWatched,
	).Scan(&progressID, &episodesWatched)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id": progressID,
		"episodes_watched": episodesWatched,
	})
}

// nullString returns nil for empty strings so Postgres stores NULL rather than "".
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}