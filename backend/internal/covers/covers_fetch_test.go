package covers

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// the start of a real PNG, enough for content sniffing
var png = append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), make([]byte, 64)...)

func TestIsRemote(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://uploads.mangadex.org/covers/a/b.jpg", true},
		{"http://image.tmdb.org/t/p/original/x.jpg", true},
		{"/api/media/abc/cover", false},
		{"file:///etc/passwd", false},
		{"ftp://example.com/x.jpg", false},
		{"https://user@example.com/x.jpg", false},
		{"https:///x.jpg", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsRemote(tt.url); got != tt.want {
			t.Errorf("IsRemote(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestIsPublic(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"8.8.8.8", true},
		{"2606:4700:4700::1111", true},
		{"127.0.0.1", false},
		{"::1", false},
		{"10.0.0.5", false},
		{"172.16.3.4", false},
		{"192.168.1.10", false},
		{"169.254.169.254", false},
		{"100.64.0.1", false},
		{"0.0.0.0", false},
		{"fd00::1", false},
		{"fe80::1", false},
	}
	for _, tt := range tests {
		if got := isPublic(net.ParseIP(tt.ip)); got != tt.want {
			t.Errorf("isPublic(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestFetchRefusesTheLocalNetwork(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Header().Set("Content-Type", "image/png")
		w.Write(png)
	}))
	defer srv.Close()

	if _, err := Fetch(srv.URL + "/cover.png"); err == nil {
		t.Fatal("a cover on 127.0.0.1 was downloaded")
	}
	if hit {
		t.Fatal("the request reached the local server")
	}
}

func TestFetchChecksTheImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(png)
		case "/html":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html></html>"))
		case "/lying":
			// claims to be an image but isn't
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write([]byte("<html></html>"))
		case "/huge":
			w.Header().Set("Content-Type", "image/png")
			w.Write(png)
			w.Write(bytes.Repeat([]byte{0}, MaxRemoteBytes))
		case "/svg":
			// an image, but not one ServeCover sends back
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	// allows the local test server, every other check still applies
	c := newClient(nil)

	data, err := fetchWith(c, srv.URL+"/ok.png")
	if err != nil {
		t.Fatalf("fetching a PNG: %v", err)
	}
	if !bytes.Equal(data, png) {
		t.Fatal("the PNG came back changed")
	}
	for _, path := range []string{"/html", "/lying", "/huge", "/svg", "/missing"} {
		if _, err := fetchWith(c, srv.URL+path); err == nil {
			t.Errorf("%s was accepted as a cover", path)
		}
	}
	if _, err := fetchWith(c, "file:///etc/passwd"); err == nil {
		t.Error("a file address was fetched")
	}
}

func TestKeepLocal(t *testing.T) {
	got := KeepLocal("$2")
	for _, want := range []string{
		"media_items.cover_origin_url = $2",
		"media_items.cover_image_url = '/api/media/' || media_items.id || '/cover'",
		"THEN media_items.cover_image_url ELSE $2 END",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("KeepLocal(\"$2\") = %s, missing %s", got, want)
		}
	}
	// the URL KeepLocal recognises is the one covers are saved under
	if URL("abc") != "/api/media/abc/cover" {
		t.Errorf("URL changed to %s, update KeepLocal to match", URL("abc"))
	}
}
