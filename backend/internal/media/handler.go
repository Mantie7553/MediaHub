package media

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

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

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type == "" || req.Title == "" {
		utils.Error(w, http.StatusBadRequest, "type and title are required")
		return
	}

	if req.Type == "music_track" && req.Artist == "" {
		utils.Error(w, http.StatusBadRequest, "artist is required for music_track")
		return
	}

	var releaseDate *time.Time
	if req.ReleaseDate != "" {
		t, err := time.Parse("2006-01-02", req.ReleaseDate)
		if err != nil {
			utils.Error(w, http.StatusBadRequest, "invalid release_date format, use YYYY-MM-DD")
			return
		}
		releaseDate = &t
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	var mediaID string
	err = tx.QueryRow(
		`INSERT INTO media_items (type, title, description, cover_image_url, release_date, external_id, external_source)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		req.Type,
		req.Title,
		utils.NullString(req.Description),
		utils.NullString(req.CoverImageURL),
		releaseDate,
		utils.NullString(req.ExternalID),
		utils.NullString(req.ExternalSource),
	).Scan(&mediaID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			utils.Error(w, http.StatusConflict, "media item already exists")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	switch req.Type {
	case "anime":
		_, err = tx.Exec(
			`INSERT INTO anime_metadata (media_item_id, studio, status, genres)
			 VALUES ($1, $2, $3, $4)`,
			mediaID,
			utils.NullString(req.Studio),
			utils.NullString(req.Status),
			pq.Array(req.Genres),
		)
	case "movie":
		_, err = tx.Exec(
			`INSERT INTO movie_metadata (media_item_id, runtime_mins, director, genres)
			 VALUES ($1, $2, $3, $4)`,
			mediaID,
			utils.NullInt(req.RuntimeMins),
			utils.NullString(req.Director),
			pq.Array(req.Genres),
		)
	case "music_track":
		_, err = tx.Exec(
			`INSERT INTO music_metadata (media_item_id, artist, track_number, duration_secs, genres)
			 VALUES ($1, $2, $3, $4, $5)`,
			mediaID,
			req.Artist,
			utils.NullInt(req.TrackNumber),
			utils.NullInt(req.DurationSec),
			pq.Array(req.Genres),
		)
	default:
		utils.Error(w, http.StatusBadRequest, "invalid media type, must be one of: anime, movie, music_track")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	if utils.InternalError(w, tx.Commit()) {
		return
	}

	utils.JSON(w, map[string]string{"id": mediaID}, http.StatusCreated)
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	mediaType := r.URL.Query().Get("type")

	items := []MediaItem{}

	var queryString string = `SELECT id, type, title, description, 
		cover_image_url, release_date, external_id, 
		external_source, created_at 
		FROM media_items`

	if mediaType != "" {
		queryString += ` WHERE type = $1`
	}

	var rows *sql.Rows
	var err error

	if mediaType != "" {
		rows, err = h.db.Query(queryString, mediaType)
	} else {
		rows, err = h.db.Query(queryString)
	}

	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item MediaItem
		err := rows.Scan(
			&item.ID, &item.Type, &item.Title, &item.Description,
			&item.CoverImageURL, &item.ReleaseDate, &item.ExternalID,
			&item.ExternalSource, &item.CreatedAt,
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

func (h *Handler) GetSpecific(w http.ResponseWriter, r *http.Request) {
	mediaId := chi.URLParam(r, "id")
	item := MediaItem{}

	queryString := `SELECT id, type, title, description, 
		cover_image_url, release_date, external_id, 
		external_source, created_at 
		FROM media_items WHERE id = $1`

	err := h.db.QueryRow(queryString, mediaId).Scan(
		&item.ID, &item.Type, &item.Title, &item.Description,
		&item.CoverImageURL, &item.ReleaseDate, &item.ExternalID,
		&item.ExternalSource, &item.CreatedAt,
	)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	var metadata any

	switch item.Type {
	case "anime":
		var meta AnimeMetadata
		err := h.db.QueryRow(
			`SELECT studio, status, genres FROM anime_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.Studio, &meta.Status, pq.Array(&meta.Genres))
		if err != nil && err != sql.ErrNoRows {
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		metadata = meta
	case "movie":
		var meta MovieMetadata
		err := h.db.QueryRow(
			`SELECT runtime_mins, director, genres FROM movie_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.RuntimeMins, &meta.Director, pq.Array(&meta.Genres))
		if err != nil && err != sql.ErrNoRows {
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		metadata = meta

	case "music_track":
		var meta MusicMetadata
		err := h.db.QueryRow(
			`SELECT artist, track_number, duration_secs, genres FROM music_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.Artist, &meta.TrackNumber, &meta.DurationSecs, pq.Array(&meta.Genres))
		if err != nil && err != sql.ErrNoRows {
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		metadata = meta
	}

	utils.JSON(w, MediaItemDetail{MediaItem: item, Metadata: metadata})
}
