package server

import (
	"net/http"
	"strings"
)

// largest request body accepted outside file uploads, which set their own limits
const maxJSONBody = 1 << 20

/*
Function:	limitBody
Purpose:	cap request bodies, so a huge JSON body can't use up memory. Multipart uploads
(chapters, covers) are left to their handlers, which allow more
*/
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
		}
		next.ServeHTTP(w, r)
	})
}

/*
Function:	secureHeaders
Purpose:	headers every API response carries: browsers must not guess content types (an
uploaded file served as something else could run as HTML), and API responses are never framed
or cached by shared caches unless a handler says otherwise
*/
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
