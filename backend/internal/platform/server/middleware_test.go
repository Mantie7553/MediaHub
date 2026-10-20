package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLimitBodyRefusesHugeJSON(t *testing.T) {
	var readErr error
	h := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))

	big := strings.NewReader(strings.Repeat("x", maxJSONBody+1))
	r := httptest.NewRequest(http.MethodPost, "/api/me/media", big)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(httptest.NewRecorder(), r)
	if readErr == nil {
		t.Error("a JSON body over the limit should fail to read")
	}

	big = strings.NewReader(strings.Repeat("x", maxJSONBody+1))
	r = httptest.NewRequest(http.MethodPost, "/api/admin/manga/1/chapters", big)
	r.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	h.ServeHTTP(httptest.NewRecorder(), r)
	if readErr != nil {
		t.Errorf("uploads set their own limit, got %v", readErr)
	}
}

func TestSecureHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	secureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "same-origin",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
