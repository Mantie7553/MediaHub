package covers

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	pngenc "image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func encoded(t *testing.T, w, h int, encode func(*bytes.Buffer, image.Image) error) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		img.Set(x, x%h, color.RGBA{R: 200, A: 255})
	}
	var buf bytes.Buffer
	if err := encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pngOf(b *bytes.Buffer, img image.Image) error  { return pngenc.Encode(b, img) }
func jpegOf(b *bytes.Buffer, img image.Image) error { return jpeg.Encode(b, img, nil) }
func gifOf(b *bytes.Buffer, img image.Image) error  { return gif.Encode(b, img, nil) }

func TestThumbnailScalesLargeCovers(t *testing.T) {
	data := encoded(t, 2000, 1000, pngOf)
	out, contentType := Thumbnail(data, "image/png")
	if contentType != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", contentType)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != MaxCoverSide || cfg.Height != MaxCoverSide/2 {
		t.Fatalf("size = %dx%d, want %dx%d", cfg.Width, cfg.Height, MaxCoverSide, MaxCoverSide/2)
	}
}

func TestThumbnailKeepsSmallCovers(t *testing.T) {
	data := encoded(t, 300, 300, jpegOf)
	out, contentType := Thumbnail(data, "image/jpeg")
	if contentType != "image/jpeg" || !bytes.Equal(out, data) {
		t.Fatal("a cover already small enough was changed")
	}
}

func TestThumbnailKeepsGIFs(t *testing.T) {
	data := encoded(t, 1200, 1200, gifOf)
	out, contentType := Thumbnail(data, "image/gif")
	if contentType != "image/gif" || !bytes.Equal(out, data) {
		t.Fatal("a GIF was changed")
	}
}

func TestThumbnailKeepsUndecodable(t *testing.T) {
	data := []byte("\xff\xd8\xff not really a jpeg")
	out, contentType := Thumbnail(data, "image/jpeg")
	if contentType != "image/jpeg" || !bytes.Equal(out, data) {
		t.Fatal("an image that can't be decoded was changed")
	}
}

func TestServe(t *testing.T) {
	dir := t.TempDir()
	cover := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(cover, encoded(t, 1600, 1600, pngOf), 0o644); err != nil {
		t.Fatal(err)
	}
	notImage := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notImage, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	Serve(rec, httptest.NewRequest(http.MethodGet, "/", nil), cover)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("content type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=3600" {
		t.Fatalf("cache control = %q", got)
	}
	cfg, err := jpeg.DecodeConfig(rec.Body)
	if err != nil || cfg.Width != MaxCoverSide {
		t.Fatalf("served %dx%d (%v), want %d wide", cfg.Width, cfg.Height, err, MaxCoverSide)
	}

	for _, path := range []string{notImage, filepath.Join(dir, "missing.jpg")} {
		rec := httptest.NewRecorder()
		Serve(rec, httptest.NewRequest(http.MethodGet, "/", nil), path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", filepath.Base(path), rec.Code)
		}
	}
}
