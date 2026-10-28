package auth

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

type updateAccountRequest struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	CurrentPassword string `json:"current_password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

/*
Function:	checkCurrentPassword
Purpose:	confirm the signed in user knows their password before an account change, so an
unlocked device or a stolen access token can't take the account over. Writes the error
response and returns false when it doesn't match
*/
func (h *Handler) checkCurrentPassword(w http.ResponseWriter, userID, password string) bool {
	if !allowPasswordAttempt("account:" + userID) {
		w.Header().Set("Retry-After", "60")
		utils.Error(w, http.StatusTooManyRequests, "too many attempts, try again in a minute")
		return false
	}
	var hash string
	err := h.db.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		utils.InternalError(w, err)
		return false
	}
	if !CheckPasswordOrDummy(password, hash) {
		utils.Error(w, http.StatusForbidden, "current password is incorrect")
		return false
	}
	accountLimiter.Reset("account:" + userID)
	return true
}

/*
Function:	UpdateAccount
Purpose:	change the signed in user's username and/or email, after checking their password
Params:
  - w: http response writer to respond to the front end
  - r: JSON with username and/or email, and current_password
*/
func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	var req updateAccountRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	req.Email = NormalizeEmail(req.Email)
	if req.Username == "" && req.Email == "" {
		utils.Error(w, http.StatusBadRequest, "username or email is required")
		return
	}
	if req.Username != "" {
		if err := ValidateUsername(req.Username); err != nil {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.Email != "" {
		if err := ValidateEmail(req.Email); err != nil {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if !h.checkCurrentPassword(w, user.UserID, req.CurrentPassword) {
		return
	}

	var username, email string
	err := h.db.QueryRow(
		`UPDATE users SET
		username = COALESCE(NULLIF($1, ''), username),
		email = COALESCE(NULLIF($2, ''), email),
		updated_at = NOW()
		WHERE id = $3
		RETURNING username, email`,
		req.Username, req.Email, user.UserID,
	).Scan(&username, &email)
	if utils.IsUniqueViolation(err) {
		utils.Error(w, http.StatusConflict, "username or email already taken")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, map[string]string{"id": user.UserID, "username": username, "email": email, "role": user.Role})
}

/*
Function:	ChangePassword
Purpose:	change the signed in user's password and sign out every other session
Params:
  - w: http response writer to respond to the front end
  - r: JSON with current_password and new_password
*/
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	var req changePasswordRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if err := ValidatePassword(req.NewPassword); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.NewPassword == req.CurrentPassword {
		utils.Error(w, http.StatusBadRequest, "the new password must be different")
		return
	}
	if !h.checkCurrentPassword(w, user.UserID, req.CurrentPassword) {
		return
	}

	hash, err := HashPassword(req.NewPassword)
	if utils.InternalError(w, err) {
		return
	}
	if _, err := h.db.Exec(`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, hash, user.UserID); utils.InternalError(w, err) {
		return
	}

	// whoever else might know the old password is signed out; this browser stays signed in
	if utils.InternalError(w, RevokeSessions(h.db, user.UserID, currentSessionHash(r))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	LogoutAll
Purpose:	end every session of the signed in user, this one included
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	if utils.InternalError(w, RevokeSessions(h.db, GetUser(r).UserID, "")) {
		return
	}
	clearRefreshCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// the hash of the session cookie on this request, "" when there isn't one
func currentSessionHash(r *http.Request) string {
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		return hashToken(c.Value)
	}
	return ""
}
