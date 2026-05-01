package media

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

type uploadRequest struct {
	Type           string `json:"type"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	CoverImageURL  string `json:"cover_image_url"`
	ReleaseDate    string `json:"release_date"`
	ExternalID     string `json:"external_id"`
	ExternalSource string `json:"external_source"`

	// anime
	Studio string   `json:"studio"`
	Status string   `json:"status"`
	Genres []string `json:"genres"`

	// movie
	RuntimeMins int    `json:"runtime_mins"`
	Director    string `json:"director"`

	// music_track
	Artist      string `json:"artist"`
	TrackNumber int    `json:"track_number"`
	DurationSec int    `json:"duration_secs"`
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Type == "" || req.Title == "" {
		http.Error(w, "type and title are required", http.StatusBadRequest)
		return
	}

	if req.Type == "music_track" && req.Artist == "" {
		http.Error(w, "artist is required for music_track", http.StatusBadRequest)
		return
	}

	var releaseDate *time.Time
	if req.ReleaseDate != "" {
		t, err := time.Parse("2006-01-02", req.ReleaseDate)
		if err != nil {
			http.Error(w, "invalid release_date format, use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		releaseDate = &t
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
		nullString(req.Description),
		nullString(req.CoverImageURL),
		releaseDate,
		nullString(req.ExternalID),
		nullString(req.ExternalSource),
	).Scan(&mediaID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			http.Error(w, "media item already exists", http.StatusConflict)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	switch req.Type {
	case "anime":
		_, err = tx.Exec(
			`INSERT INTO anime_metadata (media_item_id, studio, status, genres)
			 VALUES ($1, $2, $3, $4)`,
			mediaID,
			nullString(req.Studio),
			nullString(req.Status),
			pq.Array(req.Genres),
		)
	case "movie":
		_, err = tx.Exec(
			`INSERT INTO movie_metadata (media_item_id, runtime_mins, director, genres)
			 VALUES ($1, $2, $3, $4)`,
			mediaID,
			nullInt(req.RuntimeMins),
			nullString(req.Director),
			pq.Array(req.Genres),
		)
	case "music_track":
		_, err = tx.Exec(
			`INSERT INTO music_metadata (media_item_id, artist, track_number, duration_secs, genres)
			 VALUES ($1, $2, $3, $4, $5)`,
			mediaID,
			req.Artist,
			nullInt(req.TrackNumber),
			nullInt(req.DurationSec),
			pq.Array(req.Genres),
		)
	default:
		http.Error(w, "invalid media type, must be one of: anime, movie, music_track", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": mediaID})
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

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item MediaItem
		err := rows.Scan (
			&item.ID, &item.Type, &item.Title, &item.Description,
			&item.CoverImageURL, &item.ReleaseDate, &item.ExternalID,
			&item.ExternalSource, &item.CreatedAt,
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

func (h *Handler) GetSpecific(w http.ResponseWriter, r *http.Request) {
	mediaId := chi.URLParam(r, "id")
	item := MediaItem{}

	queryString := `SELECT id, type, title, description, 
		cover_image_url, release_date, external_id, 
		external_source, created_at 
		FROM media_items WHERE id = $1`

	err := h.db.QueryRow(queryString, mediaId).Scan (
		&item.ID, &item.Type, &item.Title, &item.Description,
		&item.CoverImageURL, &item.ReleaseDate, &item.ExternalID,
		&item.ExternalSource, &item.CreatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		metadata = meta
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MediaItemDetail{MediaItem: item, Metadata: metadata})

}

// nullString returns nil for empty strings so Postgres stores NULL rather than "".
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nullInt returns nil for zero values so Postgres stores NULL rather than 0.
func nullInt(i int) interface{} {
	if i == 0 {
		return nil
	}
	return i
}
