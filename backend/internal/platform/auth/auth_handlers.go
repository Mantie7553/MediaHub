package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

/*
db: A connection to the database
*/
type Handler struct {
	db *sql.DB
}

/*
Function:	NewHandler
Purpose:	Create a new handler for auth endpoints
Params:
  - db: a connection to the database
*/
func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Username: the new user name
Email: an email
Password: raw password
*/
type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

/*
Email: hopefully an existing email
Password: raw password
*/
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

/*
Token: JWT token used for authorization
RefreshToken: Token used to refresh a JWT when it times out
*/
type authResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

/*
Function:	Register
Purpose:	Handle new user registration
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Decode the incoming request, check that the structure is correct
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// check that all fields have been provided if not return bad request
	if req.Username == "" || req.Email == "" || req.Password == "" {
		utils.Error(w, http.StatusBadRequest, "username, email and password are required")
		return
	}

	if err := ValidatePassword(req.Password); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// hash the password
	hash, err := HashPassword(req.Password)
	if utils.InternalError(w, err) {
		return
	}

	// add the new user to the database, get back the id
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

	// set the JWT
	token, err := GenerateToken(userID, "user")
	if utils.InternalError(w, err) {
		return
	}

	// set the refresh token
	refreshToken, err := GenerateRefreshToken(h.db, userID)
	if utils.InternalError(w, err) {
		return
	}

	// return the tokens
	utils.JSON(w, authResponse{Token: token, RefreshToken: refreshToken}, http.StatusCreated)
}

/*
Function:	Login
Purpose:	Handle user login
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// decode the incoming request, check that the structure is correct
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// look for the user in the database
	var userID, passwordHash, role string
	err := h.db.QueryRow(
		`SELECT id, password_hash, role FROM users WHERE email = $1`,
		req.Email,
	).Scan(&userID, &passwordHash, &role)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// check that the password actually matches
	if !CheckPassword(req.Password, passwordHash) {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// set the JWT token
	token, err := GenerateToken(userID, role)
	if utils.InternalError(w, err) {
		return
	}

	// set the refresh token
	refreshToken, err := GenerateRefreshToken(h.db, userID)
	if utils.InternalError(w, err) {
		return
	}

	// return the tokens
	utils.JSON(w, authResponse{Token: token, RefreshToken: refreshToken})
}

/*
Function:	Refresh
Purpose:	Handle refreshing the JWT
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	// decode the incoming request, check that the structure is correct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// hash the refresh token
	tokenHash := hashToken(req.RefreshToken)

	// find the refresh token from the database
	var userID, role string
	var expiresAt time.Time
	err := h.db.QueryRow(
		`SELECT u.id, u.role, rt.expires_at
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = $1`,
		tokenHash,
	).Scan(&userID, &role, &expiresAt)

	// check that something was actually returned
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	// check if the refresh token has expired
	if time.Now().After(expiresAt) {
		utils.Error(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	if _, err := h.db.Exec(`DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash); err != nil {
		utils.InternalError(w, err)
		return
	}
	newRefresh, err := GenerateRefreshToken(h.db, userID)
	if utils.InternalError(w, err) {
		return
	}

	// if the token is still valid generate a new JWT
	token, err := GenerateToken(userID, role)
	if utils.InternalError(w, err) {
		return
	}

	// return the tokens
	utils.JSON(w, authResponse{Token: token, RefreshToken: newRefresh})
}

/*
Function:	Logout
Purpose:	Handle logging out a user (removing tokens)
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	// decode the incoming request, check that the structure is correct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// delete the refresh token
	_, err := h.db.Exec(
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		hashToken(req.RefreshToken),
	)
	if utils.InternalError(w, err) {
		return
	}

	// return nothing
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	Me
Purpose:	Get the current users information from the database
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	// get the user information from the request
	user := GetUser(r)

	// find the rest of the users information from the database
	var username, email, role string
	err := h.db.QueryRow(
		`SELECT username, email, role FROM users WHERE id = $1`,
		user.UserID,
	).Scan(&username, &email, &role)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}

	// return the user information
	utils.JSON(w, map[string]string{
		"id":       user.UserID,
		"username": username,
		"email":    email,
		"role":     role,
	})
}
