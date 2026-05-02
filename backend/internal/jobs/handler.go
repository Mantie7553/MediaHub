package jobs

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
		err := rows.Scan (
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
		err := rows.Scan (
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
	user := auth.GetUser(r)

	// decode the request body into the JobRequest struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	itemId, err := h.db.QueryRow(
		`SELECT media_item_id FROM download_requests WHERE id = $1`,
		req.RequestID
	)

	row, err := h.db.QueryRow(
		`SELECT type, external_id FROM media_items
		WHERE id = $1`,
		itemId,
	)

	var dest string
	var handler string
	switch row.Type {
	case "anime" : 
		dest = "/Media/TV Shows/"
		handler = "sonarr"
	case "movie" : 
		dest = "/Media/Movies/"
		handler = "radarr"
	case "music_track" : 
		dest = "/Media/Music/"
		handler = "ytdlp"
	case default : 
		dest = "Media/Downloads/"
		handler = "ytdlp"
	}

}