package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
)

// Exercises the route wiring only; none of these requests reach the database.
func TestMediaRoutesRequireTokens(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost")
	t.Setenv("JWT_SECRET", "access-secret-for-tests")
	t.Setenv("MEDIA_SIGNING_SECRET", "media-secret-for-tests")
	t.Setenv("MEDIA_ROOT", t.TempDir())

	s := New(nil)

	token, err := auth.GenerateMediaToken("user-1", auth.MediaScope{MediaType: "episode", MediaID: "1"})
	if err != nil {
		t.Fatal(err)
	}
	q := "?" + auth.MediaTokenParam + "=" + url.QueryEscape(token)

	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"segment without token", "/api/stream/segments/episode/1/playlist0.ts", http.StatusUnauthorized},
		{"segment with token", "/api/stream/segments/episode/1/playlist0.ts" + q, http.StatusNotFound},
		{"segment sibling id", "/api/stream/segments/episode/12/playlist0.ts" + q, http.StatusUnauthorized},
		{"segment other type", "/api/stream/segments/movie/1/playlist0.ts" + q, http.StatusUnauthorized},
		{"track with episode token", "/api/stream/music/1" + q, http.StatusUnauthorized},
		{"cover with episode token", "/api/stream/music/1/cover" + q, http.StatusUnauthorized},
		{"ln image with episode token", "/api/light-novels/1/volumes/1/images/a.jpg" + q, http.StatusUnauthorized},
		{"stream media needs bearer", "/api/stream/media/episode/1", http.StatusUnauthorized},
		{"stream media rejects media token", "/api/stream/media/episode/1" + q, http.StatusUnauthorized},
		{"track url needs bearer", "/api/stream/music/1/url", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			s.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.status, rec.Body.String())
			}
		})
	}
}
