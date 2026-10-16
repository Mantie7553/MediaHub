package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testAccessSecret = "access-secret-for-tests"
	testMediaSecret  = "media-secret-for-tests"
)

func setSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", testAccessSecret)
	t.Setenv("MEDIA_SIGNING_SECRET", testMediaSecret)
}

func signWith(t *testing.T, secret string, claims jwt.Claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mediaClaims(typ string, scope MediaScope, exp time.Time) MediaClaims {
	return MediaClaims{
		Type:       typ,
		MediaScope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
}

func TestValidateMediaToken(t *testing.T) {
	setSecrets(t)

	ep1 := MediaScope{MediaType: "episode", MediaID: "1"}
	valid, err := GenerateMediaToken("user-1", ep1)
	if err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name  string
		token string
		want  MediaScope
		ok    bool
	}{
		{"valid", valid, ep1, true},
		{"sibling id 12", valid, MediaScope{MediaType: "episode", MediaID: "12"}, false},
		{"prefix id", valid, MediaScope{MediaType: "episode", MediaID: ""}, false},
		{"other type", valid, MediaScope{MediaType: "movie", MediaID: "1"}, false},
		{"extra volume", valid, MediaScope{MediaType: "episode", MediaID: "1", VolumeID: "1"}, false},
		{"signed with access secret", signWith(t, testAccessSecret, mediaClaims(TokenTypeMedia, ep1, future)), ep1, false},
		{"typ access", signWith(t, testMediaSecret, mediaClaims(TokenTypeAccess, ep1, future)), ep1, false},
		{"typ missing", signWith(t, testMediaSecret, mediaClaims("", ep1, future)), ep1, false},
		{"expired", signWith(t, testMediaSecret, mediaClaims(TokenTypeMedia, ep1, time.Now().Add(-time.Minute))), ep1, false},
		{"garbage", "not-a-token", ep1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateMediaToken(tt.token, tt.want)
			if (err == nil) != tt.ok {
				t.Fatalf("ok = %v, want %v (err: %v)", err == nil, tt.ok, err)
			}
		})
	}
}

func TestValidateMediaTokenRequiresSecret(t *testing.T) {
	setSecrets(t)
	scope := MediaScope{MediaType: "music", MediaID: "1"}
	token, err := GenerateMediaToken("user-1", scope)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("MEDIA_SIGNING_SECRET", "")
	if _, err := GenerateMediaToken("user-1", scope); err == nil {
		t.Fatal("expected mint to fail without MEDIA_SIGNING_SECRET")
	}
	if _, err := ValidateMediaToken(token, scope); err == nil {
		t.Fatal("expected validate to fail without MEDIA_SIGNING_SECRET")
	}
}

func TestTokenTypesAreNotInterchangeable(t *testing.T) {
	setSecrets(t)

	access, err := GenerateToken("user-1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(access); err != nil {
		t.Fatalf("access token should validate: %v", err)
	}
	if _, err := ValidateMediaToken(access, MediaScope{}); err == nil {
		t.Fatal("access token accepted as media token")
	}

	media, err := GenerateMediaToken("user-1", MediaScope{MediaType: "music", MediaID: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(media); err == nil {
		t.Fatal("media token accepted as access token")
	}

	// a media-typed token signed with the access secret must still fail the access check
	forged := signWith(t, testAccessSecret, mediaClaims(TokenTypeMedia, MediaScope{}, time.Now().Add(time.Hour)))
	if _, err := ValidateToken(forged); err == nil {
		t.Fatal("media typ accepted by access validator")
	}

	// legacy access token with no typ
	legacy := signWith(t, testAccessSecret, Claims{
		UserID:           "user-1",
		Role:             "admin",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	if _, err := ValidateToken(legacy); err == nil {
		t.Fatal("access token without typ accepted")
	}
}

func TestRequireMediaToken(t *testing.T) {
	setSecrets(t)

	scope := MediaScope{MediaType: "music", MediaID: "1"}
	token, err := GenerateMediaToken("user-1", scope)
	if err != nil {
		t.Fatal(err)
	}
	access, err := GenerateToken("user-1", "admin")
	if err != nil {
		t.Fatal(err)
	}

	handler := RequireMediaToken(func(r *http.Request) MediaScope {
		return MediaScope{MediaType: "music", MediaID: r.URL.Query().Get("id")}
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		id     string
		token  string
		status int
	}{
		{"valid", "1", token, http.StatusOK},
		{"missing", "1", "", http.StatusUnauthorized},
		{"wrong id", "12", token, http.StatusUnauthorized},
		{"access token", "1", access, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := url.Values{"id": {tt.id}}
			if tt.token != "" {
				q.Set(MediaTokenParam, tt.token)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil))
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
		})
	}
}

func TestSignMediaPath(t *testing.T) {
	setSecrets(t)
	scope := MediaScope{MediaType: "episode", MediaID: "1"}

	signed, err := SignMediaPath("/stream/segments/episode/1/playlist.m3u8", "user-1", scope)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/stream/segments/episode/1/playlist.m3u8" {
		t.Fatalf("path = %q", u.Path)
	}
	if _, err := ValidateMediaToken(u.Query().Get(MediaTokenParam), scope); err != nil {
		t.Fatalf("signed token invalid: %v", err)
	}
}
