package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
		{"segment without token", "/api/stream/segments/episode/1/a1/playlist0.ts", http.StatusUnauthorized},
		{"segment with token", "/api/stream/segments/episode/1/a1/playlist0.ts" + q, http.StatusNotFound},
		{"segment sibling id", "/api/stream/segments/episode/12/a1/playlist0.ts" + q, http.StatusUnauthorized},
		{"segment other type", "/api/stream/segments/movie/1/a1/playlist0.ts" + q, http.StatusUnauthorized},
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

// Exercises routing and validation only; none of these requests reach the database.
func TestInviteRoutes(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost")
	t.Setenv("JWT_SECRET", "access-secret-for-tests")
	t.Setenv("MEDIA_SIGNING_SECRET", "media-secret-for-tests")

	s := New(nil)

	userToken, err := auth.GenerateToken("user-1", "user")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		bearer string
		status int
	}{
		{"register without invite", http.MethodPost, "/api/auth/register",
			`{"username":"a","email":"a@b.c","password":"Valid-Passw0rd!"}`, "", http.StatusBadRequest},
		{"list invites needs auth", http.MethodGet, "/api/admin/invites", "", "", http.StatusUnauthorized},
		{"list invites needs admin", http.MethodGet, "/api/admin/invites", "", userToken, http.StatusForbidden},
		{"create invite needs admin", http.MethodPost, "/api/admin/invites", "{}", userToken, http.StatusForbidden},
		{"revoke invite needs admin", http.MethodDelete, "/api/admin/invites/1", "", userToken, http.StatusForbidden},
		{"chapter upload needs auth", http.MethodPost, "/api/admin/manga/1/chapters", "", "", http.StatusUnauthorized},
		{"chapter upload needs admin", http.MethodPost, "/api/admin/manga/1/chapters", "", userToken, http.StatusForbidden},
		{"chapter check needs admin", http.MethodPost, "/api/admin/manga/1/check", "", userToken, http.StatusForbidden},
		{"album cover upload needs auth", http.MethodPost, "/api/admin/albums/1/cover", "", "", http.StatusUnauthorized},
		{"album cover upload needs admin", http.MethodPost, "/api/admin/albums/1/cover", "", userToken, http.StatusForbidden},
		{"request archive needs auth", http.MethodPost, "/api/requests/archive", `{"ids":[]}`, "", http.StatusUnauthorized},
		{"request archive needs admin", http.MethodPost, "/api/requests/archive", `{"ids":[]}`, userToken, http.StatusForbidden},
		{"job archive needs admin", http.MethodPost, "/api/admin/jobs/archive", `{"ids":[]}`, userToken, http.StatusForbidden},
		{"account change needs auth", http.MethodPut, "/api/me", `{}`, "", http.StatusUnauthorized},
		{"password change needs auth", http.MethodPut, "/api/me/password", `{}`, "", http.StatusUnauthorized},
		{"sign out everywhere needs auth", http.MethodPost, "/api/auth/logout-all", "", "", http.StatusUnauthorized},
		{"refresh without a session", http.MethodPost, "/api/auth/refresh", "", "", http.StatusUnauthorized},
		{"logout without a session", http.MethodDelete, "/api/auth/logout", "", "", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.RemoteAddr = "192.0.2.1:1234"
			if tt.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tt.bearer)
			}
			rec := httptest.NewRecorder()
			s.router.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.status, rec.Body.String())
			}
		})
	}
}
