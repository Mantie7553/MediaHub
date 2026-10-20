package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLimiterAllowsBurstThenRefills(t *testing.T) {
	now := time.Now()
	l := NewLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("attempt %d refused, want the burst allowed", i+1)
		}
	}
	if l.Allow("a") {
		t.Fatal("attempt past the burst allowed")
	}
	if !l.Allow("b") {
		t.Error("another key has its own allowance")
	}

	now = now.Add(time.Minute)
	if !l.Allow("a") {
		t.Error("one attempt should be back after an interval")
	}
	if l.Allow("a") {
		t.Error("only one attempt should be back after an interval")
	}

	l.Reset("a")
	if !l.Allow("a") {
		t.Error("Reset should give the allowance back")
	}
}

func TestLimiterForgetsIdleKeys(t *testing.T) {
	now := time.Now()
	l := NewLimiter(2, time.Second)
	l.now = func() time.Time { return now }
	l.Allow("old")

	now = now.Add(time.Hour)
	l.Allow("new")
	if _, ok := l.buckets["old"]; ok {
		t.Error("a key idle long enough to be full again should be forgotten")
	}
}

func TestClientIPIgnoresForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	r.RemoteAddr = "172.18.0.5:41000"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := ClientIP(r); got != "172.18.0.5" {
		t.Errorf("ClientIP = %s, want the connection's address, not a header the client chose", got)
	}

	r.Header.Set("X-Real-IP", "203.0.113.9")
	if got := ClientIP(r); got != "203.0.113.9" {
		t.Errorf("ClientIP = %s, want Caddy's X-Real-IP", got)
	}
}

func TestEmailsAndUsernames(t *testing.T) {
	if got := NormalizeEmail("  Me@Example.COM "); got != "me@example.com" {
		t.Errorf("NormalizeEmail = %q", got)
	}
	for _, ok := range []string{"me@example.com", "first.last+tag@mail.example.co.uk"} {
		if err := ValidateEmail(ok); err != nil {
			t.Errorf("ValidateEmail(%q) = %v, want ok", ok, err)
		}
	}
	for _, bad := range []string{"", "me", "me@localhost", "Me <me@example.com>", "a@b.c\r\nBcc: x@y.z", strings.Repeat("a", 250) + "@x.io"} {
		if ValidateEmail(bad) == nil {
			t.Errorf("ValidateEmail(%q) passed, want an error", bad)
		}
	}

	if err := ValidateUsername("Jaden M"); err != nil {
		t.Errorf("ValidateUsername = %v", err)
	}
	for _, bad := range []string{"ab", " padded ", strings.Repeat("x", 33), "tab\there"} {
		if ValidateUsername(bad) == nil {
			t.Errorf("ValidateUsername(%q) passed, want an error", bad)
		}
	}
}

func TestPasswordsOverBcryptsLimitAreRefused(t *testing.T) {
	long := "Aa1!" + strings.Repeat("x", 69)
	if ValidatePassword(long) == nil {
		t.Error("a 73 byte password should be refused, bcrypt can't hash it")
	}
	if err := ValidatePassword(long[:72]); err != nil {
		t.Errorf("a 72 byte password = %v, want ok", err)
	}
}

func TestRefreshCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	setRefreshCookie(rec, r, "secret-token")

	c := rec.Result().Cookies()[0]
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/auth" {
		t.Errorf("cookie = %+v, want HttpOnly, Secure, SameSite=Strict on /api/auth", c)
	}

	rec = httptest.NewRecorder()
	setRefreshCookie(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil), "t")
	if rec.Result().Cookies()[0].Secure {
		t.Error("over plain http (the LAN) the cookie can't be Secure or the browser drops it")
	}
}

func TestRefreshTokenFrom(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"from-body"}`))
	if got := refreshTokenFrom(r); got != "from-body" {
		t.Errorf("token = %q, want the body's for browsers signed in before the cookie", got)
	}

	r = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"from-body"}`))
	r.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "from-cookie"})
	if got := refreshTokenFrom(r); got != "from-cookie" {
		t.Errorf("token = %q, want the cookie first", got)
	}

	if got := refreshTokenFrom(httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)); got != "" {
		t.Errorf("token = %q, want none", got)
	}
}

func TestAccessTokensAreShortLived(t *testing.T) {
	t.Setenv("JWT_SECRET", "access-secret-for-tests")
	token, err := GenerateToken("user-1", "user")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if ttl := time.Until(claims.ExpiresAt.Time); ttl > AccessTokenTTL || ttl < AccessTokenTTL-time.Minute {
		t.Errorf("token lasts %s, want about %s", ttl, AccessTokenTTL)
	}
}
