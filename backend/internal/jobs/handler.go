package jobs

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/downloader"
	"github.com/Mantie7553/MediaHub/backend/internal/utils"
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

	queryString := `SELECT j.id, j.request_id, j.status, j.handler,
    j.progress_pct, j.destination_path, j.source_url,
    j.error_message, j.started_at, j.completed_at, j.created_at,
    mi.title
    FROM download_jobs j
    JOIN media_items mi ON mi.id = j.media_item_id`

	rows, err := h.db.Query(queryString)

	if utils.InternalError(w, err) {
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
			&item.CreatedAt, &item.MediaTitle,
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

	if utils.InternalError(w, err) {
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
Function: Create
Purpose: Create a new job
*/
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req JobRequest
	var itemID string
	var mediaType string
	var externalID *string
	var jobID string

	// decode the request body into the JobRequest struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.db.QueryRow(
		`SELECT media_item_id FROM download_requests WHERE id = $1`,
		req.RequestID,
	).Scan(&itemID)

	if utils.InternalError(w, err) {
		return
	}

	err = h.db.QueryRow(
		`SELECT type, external_id FROM media_items
		WHERE id = $1`,
		itemID,
	).Scan(&mediaType, &externalID)

	if utils.InternalError(w, err) {
		return
	}

	jobID, err = downloader.Dispatch(h.db, req.RequestID, itemID, req.SourceURL, mediaType, externalID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JSON(w, map[string]string{"id": jobID})
}
