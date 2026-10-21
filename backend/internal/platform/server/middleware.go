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
(chapters, covers) and the raw chunks of large uploads are left to their handlers, which
allow more and stream them to disk
*/
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if r.Body != nil && !strings.HasPrefix(ct, "multipart/form-data") && ct != "application/octet-stream" {
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
