package utils

import (
	"path/filepath"
	"testing"
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
