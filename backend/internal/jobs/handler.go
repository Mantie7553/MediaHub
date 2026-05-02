package jobs

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/downloader"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function: GetMine
Purpose: Get a list of all jobs
*/
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	items := []JobResponse{}

	queryString := `SELECT id, request_id, status, handler,
    progress_pct, destination_path, source_url,
    error_message, started_at, completed_at, created_at
    FROM download_jobs`

	rows, err := h.db.Query(queryString)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// go through database rows and turn them into usable items
	for rows.Next() {
		var item JobResponse
		err := rows.Scan(
			&item.ID, &item.RequestID, &item.Status, &item.Handler,
			&item.ProgressPct, &item.DestinationPath, &item.SourceURL,
			&item.ErrorMessage, &item.StartedAt, &item.CompletedAt,
			&item.CreatedAt,
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

/*
Function: GetMine
Purpose: Get a list of jobs for a specific user
*/
func (h *Handler) GetMine(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	items := []JobResponse{}

	queryString := `SELECT j.id, j.request_id, j.status, j.handler,
    j.progress_pct, j.destination_path, j.source_url,
    j.error_message, j.started_at, j.completed_at, j.created_at
    FROM download_jobs j
    JOIN download_requests r ON j.request_id = r.id
	WHERE r.requested_by = $1`

	rows, err := h.db.Query(queryString, user.UserID)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// go through database rows and turn them into usable items
	for rows.Next() {
		var item JobResponse
		err := rows.Scan(
			&item.ID, &item.RequestID, &item.Status, &item.Handler,
			&item.ProgressPct, &item.DestinationPath, &item.SourceURL,
			&item.ErrorMessage, &item.StartedAt, &item.CompletedAt,
			&item.CreatedAt,
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

/*
Function: Create
Purpose: Create a new job
*/
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req JobRequest
	var itemID string
	var mediaType string
	var externalID *string
	var jobID string
	var dest string
	var handler string

	// decode the request body into the JobRequest struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.db.QueryRow(
		`SELECT media_item_id FROM download_requests WHERE id = $1`,
		req.RequestID,
	).Scan(&itemID)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = h.db.QueryRow(
		`SELECT type, external_id FROM media_items
		WHERE id = $1`,
		itemID,
	).Scan(&mediaType, &externalID)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	switch mediaType {
	case "anime":
		dest = "/Media/TV Shows/"
		handler = "sonarr"
	case "movie":
		dest = "/Media/Movies/"
		handler = "radarr"
	case "music_track":
		dest = "/Media/Music/"
		handler = "ytdlp"
	default:
		dest = "/Media/Downloads/"
		handler = "ytdlp"
	}

	err = h.db.QueryRow(
		`INSERT INTO download_jobs (request_id, media_item_id, source_url, destination_path, handler)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		req.RequestID, itemID, req.SourceURL, dest, handler,
	).Scan(&jobID)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if externalID == nil {
		http.Error(w, "media item has no external_id", http.StatusBadRequest)
		return
	}

	extIDInt, err := strconv.Atoi(*externalID)
	if err != nil {
		http.Error(w, "invalid external_id", http.StatusBadRequest)
		return
	}

	switch handler {
	case "sonarr":
		sClient := arr.NewArrClient("SONARR_URL", "SONARR_API_KEY")
		seriesID, err := sClient.AddSeries(extIDInt, 1, dest)

		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		h.db.Exec(
			`INSERT INTO sonarr_items (media_item_id, sonarr_series_id) 
			VALUES ($1, $2)`,
			itemID, seriesID,
		)

	case "radarr":
		rClient := arr.NewArrClient("RADARR_URL", "RADARR_API_KEY")
		movieID, err := rClient.AddMovie(extIDInt, 1, dest)

		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		h.db.Exec(
			`INSERT INTO radarr_items (media_item_id, radarr_movie_id) 
			VALUES ($1, $2)`,
			itemID, movieID,
		)

	default:
		go downloader.Run(h.db, jobID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": jobID})

}
