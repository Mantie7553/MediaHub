package users

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	GetAll
Purpose:	Get all users in the database
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		`SELECT id, username, email, role, download_permission, created_at
		FROM users
		ORDER BY created_at ASC`,
	)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	users := []userResponse{}
	for rows.Next() {
		var u userResponse
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.DownloadPermission, &u.CreatedAt); err != nil {
			utils.InternalError(w, err)
			return
		}
		users = append(users, u)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, users)
}

/*
Function:	Create
Purpose:	Create a new user as an admin
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || req.Role == "" || req.DownloadPermission == "" {
		utils.Error(w, http.StatusBadRequest, "username, email, password, role, and download_permission are required")
		return
	}

	if err := auth.ValidatePassword(req.Password); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if utils.InternalError(w, err) {
		return
	}

	var u userResponse
	err = h.db.QueryRow(
		`INSERT INTO users (username, email, password_hash, role, download_permission)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, email, role, download_permission, created_at`,
		req.Username, req.Email, string(hash), req.Role, req.DownloadPermission,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.DownloadPermission, &u.CreatedAt)
	if err != nil {
		utils.Error(w, http.StatusConflict, "username or email already taken")
		return
	}

	utils.JSON(w, u, http.StatusCreated)
}

/*
Function:	Update
Purpose:	Update a user's role, download permission, and/or password
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role == "" && req.DownloadPermission == "" && req.Password == "" {
		utils.Error(w, http.StatusBadRequest, "at least one of role, download_permission, or password is required")
		return
	}

	if req.Password != "" {
		if err := auth.ValidatePassword(req.Password); err != nil {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		hash, err := auth.HashPassword(req.Password)
		if utils.InternalError(w, err) {
			return
		}
		_, err = h.db.Exec(
			`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
			string(hash), targetID,
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	if req.Role != "" || req.DownloadPermission != "" {
		_, err := h.db.Exec(
			`UPDATE users SET
			role = CASE WHEN $1 != '' THEN $1::user_role ELSE role END,
			download_permission = CASE WHEN $2 != '' THEN $2::download_permission ELSE download_permission END,
			updated_at = NOW()
			WHERE id = $3`,
			req.Role, req.DownloadPermission, targetID,
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	var u userResponse
	err := h.db.QueryRow(
		`SELECT id, username, email, role, download_permission, created_at FROM users WHERE id = $1`,
		targetID,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.DownloadPermission, &u.CreatedAt)
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, u)
}

/*
Function:	Delete
Purpose:	Delete a user, cannot delete your own account
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	admin := auth.GetUser(r)

	if targetID == admin.UserID {
		utils.Error(w, http.StatusForbidden, "cannot delete your own account")
		return
	}

	result, err := h.db.Exec(`DELETE FROM users WHERE id = $1`, targetID)
	if utils.InternalError(w, err) {
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
