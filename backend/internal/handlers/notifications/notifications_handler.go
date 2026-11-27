package notifications

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/notify"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// the most one list returns, far more than anyone reads through
const listLimit = 200

// how long a read notification is kept
const keepReadDays = 90

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	List
Purpose:	the user's notifications, unread first, newest first in each. ?unread=true leaves out
the read ones
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	items := []notificationResponse{}

	query := `SELECT id, kind, title, body, link, created_at, read_at
		FROM notifications WHERE user_id = $1`
	if r.URL.Query().Get("unread") == "true" {
		query += ` AND read_at IS NULL`
	}
	query += ` ORDER BY (read_at IS NULL) DESC, created_at DESC LIMIT $2`

	rows, err := h.db.Query(query, user.UserID, listLimit)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var n notificationResponse
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.Link, &n.CreatedAt, &n.ReadAt); utils.InternalError(w, err) {
			return
		}
		items = append(items, n)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}
	utils.JSON(w, items)
}

/*
Function:	UnreadCount
Purpose:	how many of the user's notifications are unread, for the bell
*/
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	var count int
	err := h.db.QueryRow(
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, user.UserID,
	).Scan(&count)
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, map[string]int{"count": count})
}

/*
Function:	MarkRead
Purpose:	mark one of the user's notifications read, or unread again
*/
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req markReadRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	var readAt *time.Time
	err := h.db.QueryRow(
		`UPDATE notifications
		SET read_at = CASE WHEN $3 THEN COALESCE(read_at, NOW()) ELSE NULL END
		WHERE id = $1 AND user_id = $2
		RETURNING read_at`,
		id, user.UserID, req.Read,
	).Scan(&readAt)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	// a newer unread one about the same thing is already there
	if utils.IsUniqueViolation(err) {
		utils.Error(w, http.StatusConflict, "a newer notification about this is already unread")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, map[string]any{"id": id, "read_at": readAt})
}

/*
Function:	MarkAllRead
Purpose:	mark every unread notification of the user's read
*/
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	res, err := h.db.Exec(
		`UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, user.UserID,
	)
	if utils.InternalError(w, err) {
		return
	}
	n, _ := res.RowsAffected()
	utils.JSON(w, map[string]int64{"updated": n})
}

/*
Function:	StartPruning
Purpose:	delete read notifications older than keepReadDays, now and once a day
*/
func (h *Handler) StartPruning() {
	go func() {
		for {
			if n, err := notify.PruneRead(h.db, keepReadDays); err != nil {
				logger.Warn("pruning read notifications: %s", err.Error())
			} else if n > 0 {
				logger.Info("pruned %d read notifications", n)
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}
