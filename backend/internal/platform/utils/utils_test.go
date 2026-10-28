package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

func TestSafeFileName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"playlist.m3u8", true},
		{"subs.vtt", true},
		{"playlist12.ts", true},
		{"cover image.jpg", true},
		{"", false},
		{".", false},
		{"..", false},
		{"../secret", false},
		{"../../etc/passwd", false},
		{"a/b", false},
		{`..\..\secret`, false},
		{`a\b`, false},
		{"/etc/passwd", false},
		{"%2e%2e%2fsecret", true}, // never decoded again, stays a literal name
		{"a\x00.ts", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeFileName(tt.name); got != tt.ok {
				t.Fatalf("SafeFileName(%q) = %v, want %v", tt.name, got, tt.ok)
			}
		})
	}
}

func TestIsWithin(t *testing.T) {
	dir := filepath.Join("media", "temp", "episode-1")
	tests := []struct {
		path string
		ok   bool
	}{
		{filepath.Join(dir, "playlist.m3u8"), true},
		{dir, false},
		{filepath.Join(dir, ".."), false},
		{filepath.Join(dir, "..", "episode-12", "playlist.m3u8"), false},
		{filepath.Join("media", "temp", "episode-12", "playlist.m3u8"), false},
	}
	for _, tt := range tests {
		if got := IsWithin(dir, tt.path); got != tt.ok {
			t.Errorf("IsWithin(%q) = %v, want %v", tt.path, got, tt.ok)
		}
	}
}

func TestLikePrefixEscapesWildcards(t *testing.T) {
	if got := LikePrefix(`/m/100%_done\`); got != `/m/100\%\_done\\%` {
		t.Errorf("LikePrefix = %q", got)
	}
}

func TestNullIntTreatsZeroAsUnknown(t *testing.T) {
	zero, five := 0, 5
	if NullInt(nil) != nil || NullInt(&zero) != nil {
		t.Error("nil and 0 should both be NULL")
	}
	if got := NullInt(&five); got == nil || *got != 5 {
		t.Errorf("NullInt(5) = %v", got)
	}
}

func TestDecodeJSONRejectsBadBodies(t *testing.T) {
	var v struct{ Name string }
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a"}`))
	if !DecodeJSON(w, r, &v) || v.Name != "a" {
		t.Fatalf("good body: ok=false or name=%q", v.Name)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`))
	if DecodeJSON(w, r, &v) || w.Code != http.StatusBadRequest {
		t.Errorf("bad body: got status %d", w.Code)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	dup := &pq.Error{Code: "23505"}
	if !IsUniqueViolation(dup) || !IsUniqueViolation(fmt.Errorf("insert: %w", dup)) {
		t.Error("23505 should count, wrapped or not")
	}
	if IsUniqueViolation(&pq.Error{Code: "23503"}) || IsUniqueViolation(errors.New("x")) || IsUniqueViolation(nil) {
		t.Error("other errors should not count")
	}
}

func TestUUIDParam(t *testing.T) {
	param := func(v string) *http.Request {
		rc := chi.NewRouteContext()
		rc.URLParams.Add("id", v)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
	}

	const id = "0b6f1b8e-4a4c-4d5e-9f00-123456789abc"
	w := httptest.NewRecorder()
	if got, ok := UUIDParam(w, param(id), "id"); !ok || got != id {
		t.Fatalf("UUIDParam(%q) = %q, %v", id, got, ok)
	}

	w = httptest.NewRecorder()
	if _, ok := UUIDParam(w, param("not-a-uuid"), "id"); ok || w.Code != http.StatusNotFound {
		t.Errorf("non-UUID: ok=%v status=%d", ok, w.Code)
	}
}
