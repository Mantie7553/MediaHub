package lists

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	Add
Purpose:	Add a new entry to the user_media_status table
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	var req rateRequest
	user := auth.GetUser(r)

	// decode the incoming request, check that the structure is correct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// confirm the media item id and status have been provided
	if (req.MediaItemId == "" && req.AlbumId == nil) || req.Status == "" {
		utils.Error(w, http.StatusBadRequest, "status is required")
		return
	}

	// add the new entry in the database
	var statusId string
	err := h.db.QueryRow(
		`INSERT INTO user_media_status (user_id, media_item_id, album_id, status, rating)
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id`,
		user.UserID, utils.NullString(req.MediaItemId), req.AlbumId, req.Status, req.Rating,
	).Scan(&statusId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			utils.Error(w, http.StatusConflict, "status already exists")
			return
		}
		utils.InternalError(w, err)
		return
	}

	// return the id of the new entry
	utils.JSON(w, map[string]string{"id": statusId}, http.StatusCreated)
}

/*
Function:	GetAll
Purpose:	Get all of the media currently in the database
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)

	items := []UserMediaEntry{}

	queryString := `
	SELECT ums.id, ums.status, ums.rating, ums.updated_at, ums.album_id,
		mi.id, mi.type, mi.title, mi.cover_image_url, mi.release_date, mi.external_id,
		mm.artist,
		anime_progress.active_season, anime_progress.season_watched, anime_progress.season_total
	FROM user_media_status ums
	JOIN media_items mi ON mi.id = ums.media_item_id
	LEFT JOIN music_metadata mm ON mm.media_item_id = mi.id
	LEFT JOIN LATERAL (
		SELECT 
			e.season_number AS active_season,
			COUNT(*) FILTER (WHERE uap.watched = true) AS season_watched,
			COUNT(*) AS season_total
		FROM episodes e
		LEFT JOIN user_anime_progress uap ON uap.episode_id = e.id AND uap.user_id = ums.user_id
		WHERE e.media_item_id = mi.id
		AND e.season_number = COALESCE(
			(SELECT e2.season_number FROM episodes e2
			JOIN user_anime_progress uap2 ON uap2.episode_id = e2.id
			WHERE e2.media_item_id = mi.id AND uap2.user_id = ums.user_id AND uap2.watched = true
			ORDER BY uap2.watched_at DESC LIMIT 1),
			1
		)
		GROUP BY e.season_number
	) anime_progress ON true
	WHERE ums.user_id = $1 AND ums.media_item_id IS NOT NULL

	UNION ALL

	SELECT ums.id, ums.status, ums.rating, ums.updated_at, ums.album_id,
		NULL, 'music_album', a.title, a.cover_image_url, NULL, NULL,
		a.artist,
		NULL, NULL, NULL
	FROM user_media_status ums
	JOIN albums a ON a.id = ums.album_id
	WHERE ums.user_id = $1 AND ums.album_id IS NOT NULL`

	rows, err := h.db.Query(queryString, user.UserID)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item UserMediaEntry
		err := rows.Scan(
			&item.ID, &item.Status, &item.Rating, &item.UpdatedAt, &item.AlbumID,
			&item.MediaItemID, &item.MediaType, &item.MediaTitle, &item.CoverImageURL,
			&item.ReleaseDate, &item.ExternalID, &item.Artist,
			&item.ActiveSeason, &item.SeasonWatched, &item.SeasonTotal,
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

/*
Function:	Update
Purpose:	Handle updating the status for a given media item for a specific user
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	// get the user information from the request
	user := auth.GetUser(r)
	// get the entries id from the URL parameters
	entryID := chi.URLParam(r, "id")
	// decode the incoming request, check that the structure is correct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// do the update
	var mediaStatusId string
	err := h.db.QueryRow(
		`UPDATE user_media_status 
		SET status = $1, rating = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING id`,
		req.Status, req.Rating, entryID, user.UserID,
	).Scan(&mediaStatusId)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	// return the id for the updated status
	utils.JSON(w, map[string]string{"id": mediaStatusId})
}

/*
Function:	Delete
Purpose:	Remove a users status for a media item
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	// get the user information from the request
	user := auth.GetUser(r)
	// get the entries is from the URL parameters
	entryID := chi.URLParam(r, "id")

	// do the delete
	result, err := h.db.Exec(
		`DELETE FROM user_media_status WHERE id = $1 AND user_id = $2`,
		entryID, user.UserID,
	)

	if utils.InternalError(w, err) {
		return
	}

	// check that it was actually deleted
	rows, _ := result.RowsAffected()
	if rows == 0 {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	// return no content
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	UpdateProgress
Purpose:	Update a users progress for a specific piece of media
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	// get the user info from the request
	user := auth.GetUser(r)
	// get the entries id from the URL parameters
	episodeID := chi.URLParam(r, "id")

	// decode the incoming request, check that the structure is correct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// do the update
	var progressID string
	err := h.db.QueryRow(
		`INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, episode_id) 
		DO UPDATE SET watched = $3, watched_at = NOW()
		RETURNING id`,
		user.UserID, episodeID, req.Watched,
	).Scan(&progressID)

	if utils.InternalError(w, err) {
		return
	}

	// return the id
	utils.JSON(w, map[string]any{
		"id": progressID,
	})
}

func (h *Handler) MarkSeasonWatched(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	seasonNumber := chi.URLParam(r, "seasonNumber")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err := h.db.Exec(
		`INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at) 
		SELECT $1, e.id, $2, NOW() 
		FROM episodes e 
		WHERE e.media_item_id = $3 AND e.season_number = $4 
		ON CONFLICT (user_id, episode_id) 
		DO UPDATE SET watched = $2, watched_at = NOW()`,
		user.UserID, req.Watched, id, seasonNumber,
	)

	if utils.InternalError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkShowWatched(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err := h.db.Exec(
		`INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at) 
		SELECT $1, e.id, $2, NOW() 
		FROM episodes e 
		WHERE e.media_item_id = $3
		ON CONFLICT (user_id, episode_id) 
		DO UPDATE SET watched = $2, watched_at = NOW()`,
		user.UserID, req.Watched, id,
	)

	if utils.InternalError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
