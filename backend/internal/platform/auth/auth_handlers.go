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
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code"`
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
Token: JWT access token used for authorization. The refresh token goes in an HttpOnly cookie
and never appears in a response body
*/
type authResponse struct {
	Token string `json:"token"`
}

/*
Function:	startSession
Purpose:	create a refresh token for a user, set it as the session cookie and answer with an
access token
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
  - userID: the user signing in
  - role: their role, for the access token
  - status: the HTTP status to answer with
*/
func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, userID, role string, status int) {
	token, err := GenerateToken(userID, role)
	if utils.InternalError(w, err) {
		return
	}
	refreshToken, err := GenerateRefreshToken(h.db, userID)
	if utils.InternalError(w, err) {
		return
	}
	setRefreshCookie(w, r, refreshToken)
	utils.JSON(w, authResponse{Token: token}, status)
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
	req.Email = NormalizeEmail(req.Email)

	// check that all fields have been provided if not return bad request
	if req.Username == "" || req.Email == "" || req.Password == "" || req.InviteCode == "" {
		utils.Error(w, http.StatusBadRequest, "username, email, password and invite code are required")
		return
	}

	for _, err := range []error{ValidateUsername(req.Username), ValidateEmail(req.Email), ValidatePassword(req.Password)} {
		if err != nil {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// check the invite before the (slow) password hash so bad codes are cheap to reject
	codeHash := HashInviteCode(req.InviteCode)
	var inviteValid bool
	err := h.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM invite_codes WHERE code_hash = $1 AND used_at IS NULL AND expires_at > NOW())`,
		codeHash,
	).Scan(&inviteValid)
	if utils.InternalError(w, err) {
		return
	}
	if !inviteValid {
		utils.Error(w, http.StatusForbidden, "invalid or expired invite code")
		return
	}

	// hash the password
	hash, err := HashPassword(req.Password)
	if utils.InternalError(w, err) {
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	// claim the invite, the row lock means only one sign up can use it.
	// if creating the user fails the rollback releases the code again
	var inviteID string
	err = tx.QueryRow(
		`UPDATE invite_codes SET used_at = NOW()
		 WHERE code_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		 RETURNING id`,
		codeHash,
	).Scan(&inviteID)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusForbidden, "invalid or expired invite code")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	// add the new user to the database, get back the id
	var userID string
	err = tx.QueryRow(
		`INSERT INTO users (username, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		req.Username, req.Email, hash,
	).Scan(&userID)
	if err != nil {
		utils.Error(w, http.StatusConflict, "username or email already taken")
		return
	}

	if _, err := tx.Exec(`UPDATE invite_codes SET used_by = $1 WHERE id = $2`, userID, inviteID); utils.InternalError(w, err) {
		return
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}

	h.startSession(w, r, userID, "user", http.StatusCreated)
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
	email := NormalizeEmail(req.Email)

	// guessing one account's password from many addresses is limited too
	if !allowPasswordAttempt("login:" + email) {
		w.Header().Set("Retry-After", "60")
		utils.Error(w, http.StatusTooManyRequests, "too many attempts for this account, try again in a minute")
		return
	}

	// look for the user in the database
	var userID, passwordHash, role string
	err := h.db.QueryRow(
		`SELECT id, password_hash, role FROM users WHERE LOWER(email) = $1`,
		email,
	).Scan(&userID, &passwordHash, &role)
	if err != nil && err != sql.ErrNoRows {
		utils.InternalError(w, err)
		return
	}

	// an unknown email takes as long as a wrong password and gets the same answer
	if !CheckPasswordOrDummy(req.Password, passwordHash) {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	accountLimiter.Reset("login:" + email)

	// expired sessions are tidied up here rather than by a job of their own
	h.db.Exec(`DELETE FROM refresh_tokens WHERE expires_at < NOW()`)

	h.startSession(w, r, userID, role, http.StatusOK)
}

/*
Function:	Refresh
Purpose:	Give the browser a new access token for its session cookie, and extend the session
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := refreshTokenFrom(r)
	if refreshToken == "" {
		utils.Error(w, http.StatusUnauthorized, "not signed in")
		return
	}
	tokenHash := hashToken(refreshToken)

	// the session is extended rather than replaced: replacing it on every refresh signs out
	// tabs that refresh at the same moment (several tabs restored together)
	var userID, role string
	err := h.db.QueryRow(
		`UPDATE refresh_tokens rt SET expires_at = $2
		FROM users u
		WHERE u.id = rt.user_id AND rt.token_hash = $1 AND rt.expires_at > NOW()
		RETURNING u.id, u.role`,
		tokenHash, time.Now().Add(RefreshTokenTTL),
	).Scan(&userID, &role)
	if err == sql.ErrNoRows {
		clearRefreshCookie(w, r)
		utils.Error(w, http.StatusUnauthorized, "session expired, sign in again")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	token, err := GenerateToken(userID, role)
	if utils.InternalError(w, err) {
		return
	}

	// a browser that sent its token in the body moves it into the cookie now
	setRefreshCookie(w, r, refreshToken)
	utils.JSON(w, authResponse{Token: token})
}

/*
Function:	Logout
Purpose:	End this browser's session
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if refreshToken := refreshTokenFrom(r); refreshToken != "" {
		_, err := h.db.Exec(`DELETE FROM refresh_tokens WHERE token_hash = $1`, hashToken(refreshToken))
		if utils.InternalError(w, err) {
			return
		}
	}
	clearRefreshCookie(w, r)
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
