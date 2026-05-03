package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/utils"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		utils.Error(w, http.StatusBadRequest, "username, email and password are required")
		return
	}

	hash, err := HashPassword(req.Password)
	if utils.InternalError(w, err) {
		return
	}

	var userID string
	err = h.db.QueryRow(
		`INSERT INTO users (username, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		req.Username, req.Email, hash,
	).Scan(&userID)
	if err != nil {
		utils.Error(w, http.StatusConflict, "username or email already taken")
		return
	}

	token, err := GenerateToken(userID, "user")
	if utils.InternalError(w, err) {
		return
	}

	refreshToken, err := GenerateRefreshToken(h.db, userID)
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, authResponse{Token: token, RefreshToken: refreshToken}, http.StatusCreated)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var userID, passwordHash, role string
	err := h.db.QueryRow(
		`SELECT id, password_hash, role FROM users WHERE email = $1`,
		req.Email,
	).Scan(&userID, &passwordHash, &role)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if !CheckPassword(req.Password, passwordHash) {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := GenerateToken(userID, role)
	if utils.InternalError(w, err) {
		return
	}

	refreshToken, err := GenerateRefreshToken(h.db, userID)
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, authResponse{Token: token, RefreshToken: refreshToken})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokenHash := hashToken(req.RefreshToken)

	var userID, role string
	var expiresAt time.Time
	err := h.db.QueryRow(
		`SELECT u.id, u.role, rt.expires_at
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = $1`,
		tokenHash,
	).Scan(&userID, &role, &expiresAt)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	if time.Now().After(expiresAt) {
		utils.Error(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	token, err := GenerateToken(userID, role)
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, authResponse{Token: token, RefreshToken: req.RefreshToken})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err := h.db.Exec(
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		hashToken(req.RefreshToken),
	)
	if utils.InternalError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)

	var username, email, role string
	err := h.db.QueryRow(
		`SELECT username, email, role FROM users WHERE id = $1`,
		user.UserID,
	).Scan(&username, &email, &role)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}

	utils.JSON(w, map[string]string{
		"id":       user.UserID,
		"username": username,
		"email":    email,
		"role":     role,
	})
}
