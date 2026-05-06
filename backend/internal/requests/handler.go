package requests

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/downloader"
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

/*
Function:	Add
Purpose:	add an entry to the database for a Download Request
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	var downloadPermission string
	var err error
	// get the user info from the request
	user := auth.GetUser(r)

	// decode the incoming request, check that the structure is correct
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// check that at least one of the item id or title override were provided
	if req.MediaItemId == "" && req.TitleOverride == "" {
		utils.Error(w, http.StatusBadRequest, "media_item_id or title_override is required")
		return
	}

	// check if the user has permissions to download
	err = h.db.QueryRow(
		`SELECT download_permission FROM users WHERE id = $1`,
		user.UserID,
	).Scan(&downloadPermission)
	if utils.InternalError(w, err) {
		return
	}

	// set the staus appropriately
	status := "pending"
	autoApproved := false
	if downloadPermission == "auto_approved" {
		status = "approved"
		autoApproved = true
	}

	// add the new request
	var requestId string
	err = h.db.QueryRow(
		`INSERT INTO download_requests 
		(requested_by, media_item_id, album_id, title_override, 
		source_url, status, auto_approved, admin_notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		user.UserID, utils.NullString(req.MediaItemId), nil,
		utils.NullString(req.TitleOverride), req.SourceUrl,
		status, autoApproved, nil,
	).Scan(&requestId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			utils.Error(w, http.StatusConflict, "download request already made")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if autoApproved {
		var mediaType string
		var externalID *string
		err = h.db.QueryRow(
			`SELECT type, external_id FROM media_items WHERE id = $1`,
			req.MediaItemId,
		).Scan(&mediaType, &externalID)
		if err == nil {
			downloader.Dispatch(h.db, requestId, req.MediaItemId, req.SourceUrl, mediaType, externalID)
		}
	}

	// return the id of the new request
	utils.JSON(w, map[string]string{"id": requestId}, http.StatusCreated)
}

/*
Function:	GetAll
Purpose:	get all download requests from the database for a specific user
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	// get user info from the request
	user := auth.GetUser(r)

	items := []downloadRequestResponse{}

	queryString := `SELECT dr.id, dr.status, dr.auto_approved, dr.requested_at,
	dr.title_override, mi.title
	FROM download_requests dr
	LEFT JOIN media_items mi ON mi.id = dr.media_item_id
	WHERE dr.requested_by = $1`

	// run the query
	rows, err := h.db.Query(queryString, user.UserID)

	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	// map rows to useable struct
	for rows.Next() {
		var item downloadRequestResponse
		err := rows.Scan(
			&item.ID, &item.Status, &item.AutoApproved, &item.RequestedAt,
			&item.TitleOverride, &item.MediaTitle,
		)

		if utils.InternalError(w, err) {
			return
		}
		items = append(items, item)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	// return the requests
	utils.JSON(w, items)
}

/*
Function:	GetAllAdmin
Purpose:	get all download requests in the database
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetAllAdmin(w http.ResponseWriter, r *http.Request) {
	items := []downloadRequestResponse{}

	queryString := `SELECT dr.id, dr.status, dr.auto_approved, dr.requested_at,
	dr.title_override, mi.title, u.username
	FROM download_requests dr
	LEFT JOIN media_items mi ON mi.id = dr.media_item_id
	JOIN users u ON u.id = dr.requested_by`

	// run the query
	rows, err := h.db.Query(queryString)

	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	// map the rows to useable structs
	for rows.Next() {
		var item downloadRequestResponse
		err := rows.Scan(
			&item.ID, &item.Status, &item.AutoApproved, &item.RequestedAt,
			&item.TitleOverride, &item.MediaTitle, &item.RequestedBy,
		)

		if utils.InternalError(w, err) {
			return
		}
		items = append(items, item)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	// return the requests
	utils.JSON(w, items)
}

/*
Function:	Approve
Purpose:	Approve a download request as an admin
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	// get the request id from the URL parameters
	requestId := chi.URLParam(r, "id")

	// update the download request with status "approved" and the time it was updated
	var requestThingId string
	var mediaItemId sql.NullString
	var sourceURL sql.NullString
	err := h.db.QueryRow(
		`UPDATE download_requests
		SET status = 'approved', resolved_at = NOW()
		WHERE id = $1
		RETURNING id, media_item_id, source_url`,
		requestId,
	).Scan(&requestThingId, &mediaItemId, &sourceURL)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	// get the external id for the media item
	var mediaType string
	var externalID *string
	err = h.db.QueryRow(
		`SELECT type, external_id FROM media_items
		WHERE id = $1`,
		mediaItemId.String,
	).Scan(&mediaType, &externalID)

	if utils.InternalError(w, err) {
		return
	}

	// start the download
	_, err = downloader.Dispatch(h.db, requestThingId, mediaItemId.String, sourceURL.String, mediaType, externalID)

	if utils.InternalError(w, err) {
		return
	}

	// return the request id from the database
	utils.JSON(w, map[string]string{"id": requestThingId})
}

/*
Function:	Reject
Purpose:	Reject a download request as an admin
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	var req rejectRequest
	// get the request ID from the url parameters
	requestId := chi.URLParam(r, "id")

	// decode the incoming request, check that the structure is correct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// update the download request with status "rejected" and the time it was updated
	var requestThingId string
	err := h.db.QueryRow(
		`UPDATE download_requests
		SET status = 'rejected', resolved_at = NOW(), admin_notes = $2
		WHERE id = $1
		RETURNING id`,
		requestId, utils.NullString(req.AdminNotes),
	).Scan(&requestThingId)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	// return the  request id from the database
	utils.JSON(w, map[string]string{"id": requestThingId})
}
