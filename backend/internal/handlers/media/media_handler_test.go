package media

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseCoverURL(t *testing.T) {
	tests := []struct {
		raw string
		ok  bool
	}{
		{"https://uploads.mangadex.org/covers/abc/def.jpg", true},
		{"http://uploads.mangadex.org/covers/abc/def.jpg", false},
		{"https://uploads.mangadex.org.evil.com/x.jpg", false},
		{"https://evil.com/uploads.mangadex.org/x.jpg", false},
		{"https://uploads.mangadex.org@evil.com/x.jpg", false},
		{"https://user@uploads.mangadex.org/x.jpg", false},
		{"https://uploads.mangadex.org:8443/x.jpg", false},
		{"https://127.0.0.1/x.jpg", false},
		{"//uploads.mangadex.org/x.jpg", false},
		{"file:///etc/passwd", false},
		{"not a url", false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if _, ok := parseCoverURL(tt.raw); ok != tt.ok {
				t.Fatalf("parseCoverURL(%q) = %v, want %v", tt.raw, ok, tt.ok)
			}
		})
	}
}

func TestCoverClientRedirects(t *testing.T) {
	evilHit := false
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		evilHit = true
		w.Write([]byte("internal"))
	}))
	defer evil.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/offhost":
			http.Redirect(w, r, evil.URL+"/admin", http.StatusFound)
		case "/onhost":
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/loop":
			http.Redirect(w, r, "/loop", http.StatusFound)
		default:
			w.Write([]byte("ok"))
		}
	}))
	defer origin.Close()

	client := &http.Client{CheckRedirect: sameHostRedirects}

	if _, err := client.Get(origin.URL + "/offhost"); err == nil {
		t.Fatal("off-host redirect was followed")
	}
	if evilHit {
		t.Fatal("request reached the off-host server")
	}

	resp, err := client.Get(origin.URL + "/onhost")
	if err != nil {
		t.Fatalf("same-host redirect failed: %v", err)
	}
	resp.Body.Close()
	if resp.Request.URL.Path != "/final" {
		t.Fatalf("ended at %s, want /final", resp.Request.URL.Path)
	}

	if _, err := client.Get(origin.URL + "/loop"); err == nil {
		t.Fatal("redirect loop was not stopped")
	}

	if coverClient.CheckRedirect == nil || coverClient.Timeout == 0 {
		t.Fatal("coverClient must set a redirect policy and timeout")
	}
}

func TestWatchedSeconds(t *testing.T) {
	for _, tt := range []struct{ in, want float64 }{
		{4.75, 4.75},
		{0, 0},
		{-3, 0},
		{math.NaN(), 0},
		{math.Inf(1), maxWatchedPerSave},
		{600, maxWatchedPerSave},
	} {
		if got := watchedSeconds(tt.in); got != tt.want {
			t.Errorf("watchedSeconds(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestReachedEnd(t *testing.T) {
	tests := []struct {
		position, duration float64
		want               bool
	}{
		{1420, 1420, true},
		{1412, 1420, true},
		{1410, 1420, true},
		{1409, 1420, false},
		{600, 1420, false},
		{0, 0, false},
		{100, 0, false},
		{0, 5, false},
		{math.NaN(), 1420, false},
	}
	for _, tt := range tests {
		if got := reachedEnd(tt.position, tt.duration); got != tt.want {
			t.Errorf("reachedEnd(%v, %v) = %v, want %v", tt.position, tt.duration, got, tt.want)
		}
	}
}
