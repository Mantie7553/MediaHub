package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSignPlaylist(t *testing.T) {
	playlist := "#EXTM3U\n#EXT-X-VERSION:3\n#EXTINF:10.0,\nplaylist0.ts\n#EXTINF:10.0,\nplaylist1.ts?x=1\n\n"
	got := signPlaylist(playlist, "a.b+c")

	want := "#EXTM3U\n#EXT-X-VERSION:3\n#EXTINF:10.0,\nplaylist0.ts?media_token=a.b%2Bc\n#EXTINF:10.0,\nplaylist1.ts?x=1&media_token=a.b%2Bc\n\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "#EXTM3U?") {
		t.Fatal("tag lines must not be signed")
	}
}

func TestServeSegmentRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEDIA_ROOT", root)

	tempDir := filepath.Join(root, "temp", "episode-1-a1")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "playlist0.ts"), []byte("segment"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(nil)
	tests := []struct {
		name      string
		mediaType string
		id        string
		variant   string
		file      string
		status    int
	}{
		{"valid", "episode", "1", "a1", "playlist0.ts", http.StatusOK},
		{"dot dot", "episode", "1", "a1", "..", http.StatusBadRequest},
		{"parent file", "episode", "1", "a1", "../../secret.txt", http.StatusBadRequest},
		{"backslash", "episode", "1", "a1", `..\..\secret.txt`, http.StatusBadRequest},
		{"id traversal", "episode", `1\..\..`, "a1", "secret.txt", http.StatusBadRequest},
		{"bad type", "../temp", "1", "a1", "playlist0.ts", http.StatusBadRequest},
		{"variant traversal", "episode", "1", "..", "playlist0.ts", http.StatusBadRequest},
		{"variant not a track", "episode", "1", "x1", "playlist0.ts", http.StatusBadRequest},
		{"internal file", "episode", "1", "a1", "stream.json", http.StatusBadRequest},
		{"ffmpeg log", "episode", "1", "a1", "transcode.log", http.StatusBadRequest},
		{"ffmpeg playlist", "episode", "1", "a1", "ffmpeg.m3u8", http.StatusBadRequest},
		{"half written subtitle", "episode", "1", "a1", "subs_3.ass.tmp", http.StatusBadRequest},
		{"font not extracted yet", "episode", "1", "a1", "font_5.ttf", http.StatusNotFound},
		{"burned in variant", "episode", "1", "a1-b4", "playlist9.ts", http.StatusNotFound},
		{"unwritten segment without a transcode", "episode", "1", "a1", "playlist5.ts", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("type", tt.mediaType)
			rctx.URLParams.Add("id", tt.id)
			rctx.URLParams.Add("variant", tt.variant)
			rctx.URLParams.Add("file", tt.file)
			req := httptest.NewRequest(http.MethodGet, "/segment", nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			h.ServeSegment(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if strings.Contains(rec.Body.String(), "secret") && !strings.Contains(rec.Body.String(), "invalid") {
				t.Fatal("served a file outside the temp dir")
			}
			// a text type here makes the proxy compress every segment
			if tt.status == http.StatusOK && rec.Header().Get("Content-Type") != "video/mp2t" {
				t.Fatalf("Content-Type = %q, want video/mp2t", rec.Header().Get("Content-Type"))
			}
		})
	}
}

