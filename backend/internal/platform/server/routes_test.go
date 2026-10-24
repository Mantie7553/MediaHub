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
		{"password change under /auth needs auth", http.MethodPut, "/api/auth/password", `{}`, "", http.StatusUnauthorized},
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

// Exercises routing and auth only; the admin requests stop before the database.
func TestCurationRoutes(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost")
	t.Setenv("JWT_SECRET", "access-secret-for-tests")
	t.Setenv("MEDIA_SIGNING_SECRET", "media-secret-for-tests")
	t.Setenv("MEDIA_ROOT", t.TempDir())

	s := New(nil)
	userToken, err := auth.GenerateToken("user-1", "user")
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := auth.GenerateToken("admin-1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	const id = "6f1c2b9e-1d2a-4b3c-8d4e-5f6a7b8c9d0e"
	coverToken, err := auth.GenerateMediaToken("user-1", auth.MediaScope{MediaType: "cover", MediaID: id})
	if err != nil {
		t.Fatal(err)
	}
	musicToken, err := auth.GenerateMediaToken("user-1", auth.MediaScope{MediaType: "music", MediaID: id})
	if err != nil {
		t.Fatal(err)
	}

	adminOnly := []struct{ method, path string }{
		{http.MethodPut, "/api/admin/media/" + id},
		{http.MethodPost, "/api/admin/media/" + id + "/cover"},
		{http.MethodPut, "/api/admin/albums/" + id},
		{http.MethodPut, "/api/admin/episodes/" + id},
		{http.MethodPut, "/api/admin/chapters/" + id},
		{http.MethodPut, "/api/admin/volumes/" + id},
		{http.MethodDelete, "/api/admin/episodes/" + id},
		{http.MethodDelete, "/api/admin/chapters/" + id},
		{http.MethodDelete, "/api/admin/volumes/" + id},
		{http.MethodPost, "/api/admin/uploads"},
		{http.MethodPut, "/api/admin/uploads/abc"},
		{http.MethodDelete, "/api/admin/uploads/abc"},
		{http.MethodPost, "/api/admin/anime/" + id + "/episodes"},
		{http.MethodPost, "/api/admin/movies/" + id + "/file"},
		{http.MethodPost, "/api/admin/light-novels/" + id + "/volumes"},
		{http.MethodPost, "/api/admin/albums/" + id + "/tracks"},
	}
	for _, rt := range adminOnly {
		for _, c := range []struct {
			bearer string
			status int
		}{{"", http.StatusUnauthorized}, {userToken, http.StatusForbidden}} {
			req := httptest.NewRequest(rt.method, rt.path, strings.NewReader("{}"))
			if c.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+c.bearer)
			}
			rec := httptest.NewRecorder()
			s.router.ServeHTTP(rec, req)
			if rec.Code != c.status {
				t.Errorf("%s %s with bearer=%t: status = %d, want %d", rt.method, rt.path, c.bearer != "", rec.Code, c.status)
			}
		}
	}

	covers := []struct {
		name   string
		query  string
		status int
	}{
		{"cover without token", "", http.StatusUnauthorized},
		{"cover with a music token", "?" + auth.MediaTokenParam + "=" + url.QueryEscape(musicToken), http.StatusUnauthorized},
		// no file was uploaded for it
		{"cover with its token", "?" + auth.MediaTokenParam + "=" + url.QueryEscape(coverToken), http.StatusNotFound},
	}
	for _, c := range covers {
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/media/"+id+"/cover"+c.query, nil))
		if rec.Code != c.status {
			t.Errorf("%s: status = %d, want %d", c.name, rec.Code, c.status)
		}
	}

	// an admin can start an upload; nothing here needs the database
	req := httptest.NewRequest(http.MethodPost, "/api/admin/uploads", strings.NewReader(`{"name":"a.mkv","size":4,"kind":"video"}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("admin upload start: status = %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestCollectionRoutesNeedSignIn(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost")
	t.Setenv("JWT_SECRET", "access-secret-for-tests")
	t.Setenv("MEDIA_SIGNING_SECRET", "media-secret-for-tests")
	t.Setenv("MEDIA_ROOT", t.TempDir())

	s := New(nil)
	const id = "6f1c2b9e-1d2a-4b3c-8d4e-5f6a7b8c9d0e"
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/me/collections"},
		{http.MethodPost, "/api/me/collections"},
		{http.MethodGet, "/api/collections/" + id},
		{http.MethodGet, "/api/collections/listened"},
		{http.MethodPut, "/api/collections/" + id},
		{http.MethodDelete, "/api/collections/" + id},
		{http.MethodPost, "/api/collections/" + id + "/items"},
		{http.MethodDelete, "/api/collections/" + id + "/items/" + id},
		{http.MethodPut, "/api/collections/" + id + "/items/" + id + "/position"},
	}
	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, strings.NewReader("{}"))
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session: status = %d, want %d", rt.method, rt.path, rec.Code, http.StatusUnauthorized)
		}
	}
}

func TestInsightsRouteIsAdminOnly(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost")
	t.Setenv("JWT_SECRET", "access-secret-for-tests")
	t.Setenv("MEDIA_SIGNING_SECRET", "media-secret-for-tests")
	t.Setenv("MEDIA_ROOT", t.TempDir())

	s := New(nil)
	userToken, err := auth.GenerateToken("user-1", "user")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		bearer string
		status int
	}{{"", http.StatusUnauthorized}, {userToken, http.StatusForbidden}} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/insights", nil)
		if c.bearer != "" {
			req.Header.Set("Authorization", "Bearer "+c.bearer)
		}
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)
		if rec.Code != c.status {
			t.Errorf("GET /api/admin/insights with bearer=%t: status = %d, want %d", c.bearer != "", rec.Code, c.status)
		}
	}
}

func TestRecordPlayNeedsSignIn(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost")
	t.Setenv("JWT_SECRET", "access-secret-for-tests")
	t.Setenv("MEDIA_SIGNING_SECRET", "media-secret-for-tests")
	t.Setenv("MEDIA_ROOT", t.TempDir())

	s := New(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/music/6f1c2b9e-1d2a-4b3c-8d4e-5f6a7b8c9d0e/plays", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /api/music/{id}/plays without a token: status = %d, want 401", rec.Code)
	}
}

func TestFeaturingRoutes(t *testing.T) {
	t.Setenv("CORS_ORIGIN", "http://localhost")
	t.Setenv("JWT_SECRET", "access-secret-for-tests")
	t.Setenv("MEDIA_SIGNING_SECRET", "media-secret-for-tests")
	t.Setenv("MEDIA_ROOT", t.TempDir())

	s := New(nil)
	userToken, err := auth.GenerateToken("user-1", "user")
	if err != nil {
		t.Fatal(err)
	}
	const id = "6f1c2b9e-1d2a-4b3c-8d4e-5f6a7b8c9d0e"
	serve := func(method, path, bearer string) int {
		req := httptest.NewRequest(method, path, strings.NewReader("{}"))
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)
		return rec.Code
	}

	signedIn := []struct{ method, path string }{
		{http.MethodGet, "/api/featured"},
		{http.MethodPost, "/api/collections/" + id + "/submission"},
		{http.MethodDelete, "/api/collections/" + id + "/submission"},
		{http.MethodPost, "/api/collections/" + id + "/copy"},
	}
	for _, rt := range signedIn {
		if code := serve(rt.method, rt.path, ""); code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token: status = %d, want 401", rt.method, rt.path, code)
		}
	}

	adminOnly := []struct{ method, path string }{
		{http.MethodGet, "/api/admin/collection-submissions"},
		{http.MethodPost, "/api/admin/collection-submissions/" + id + "/approve"},
		{http.MethodPost, "/api/admin/collection-submissions/" + id + "/reject"},
		{http.MethodPost, "/api/admin/collections/" + id + "/feature"},
	}
	for _, rt := range adminOnly {
		if code := serve(rt.method, rt.path, ""); code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token: status = %d, want 401", rt.method, rt.path, code)
		}
		if code := serve(rt.method, rt.path, userToken); code != http.StatusForbidden {
			t.Errorf("%s %s as a user: status = %d, want 403", rt.method, rt.path, code)
		}
	}
}
