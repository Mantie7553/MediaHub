package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const (
	// the refresh token's cookie. Scripts can't read it (HttpOnly), other sites can't make the
	// browser send it (SameSite=Strict), and it is only sent to the session endpoints (Path)
	refreshCookieName = "mediahub_session"
	refreshCookiePath = "/api/auth"
	// how long a session lasts without being used; every refresh starts it again
	RefreshTokenTTL = 30 * 24 * time.Hour
)

/*
Function:	setRefreshCookie
Purpose:	hand the browser its refresh token as an HttpOnly cookie. Secure when the request
reached Caddy over HTTPS (Cloudflare, or a TLS setup), so it still works on plain http on the LAN
*/
func setRefreshCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		MaxAge:   int(RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
}

/*
Function:	clearRefreshCookie
Purpose:	tell the browser to delete its refresh token cookie
*/
func clearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
}

/*
Function:	refreshTokenFrom
Purpose:	the refresh token a request carries: the cookie, or for browsers signed in before
the cookie existed, a refresh_token in the JSON body (sent once, then the frontend forgets it)
*/
func refreshTokenFrom(r *http.Request) string {
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err == nil {
		return body.RefreshToken
	}
	return ""
}

// whether the browser's connection was HTTPS. Caddy passes X-Forwarded-Proto on from
// Cloudflare and sets it itself otherwise
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}
