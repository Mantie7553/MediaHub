package utils

import "testing"

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
