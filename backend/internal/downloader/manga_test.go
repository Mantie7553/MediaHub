package downloader

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/suwayomi"
	"github.com/Mantie7553/MediaHub/backend/internal/mangafiles"
)

type fakePages struct {
	urls     []string
	images   map[string][]byte
	failures map[string]int // url -> how many times it fails before working
}

func (f *fakePages) PageURLs(string) ([]string, error) { return f.urls, nil }

func (f *fakePages) Image(url string) ([]byte, error) {
	if f.failures[url] > 0 {
		f.failures[url]--
		return nil, errors.New("server hiccup")
	}
	img, ok := f.images[url]
	if !ok {
		return nil, errors.New("404")
	}
	return img, nil
}

// the first bytes of real image files, enough for the type sniffing
var (
	pngBytes  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpegBytes = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	webpBytes = []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
	avifBytes = []byte("\x00\x00\x00\x1cftypavif\x00\x00\x00\x00")
)

func noRetryWaits(t *testing.T) {
	saved := imageRetryDelays
	imageRetryDelays = []time.Duration{0, 0}
	t.Cleanup(func() { imageRetryDelays = saved })
}

func TestDownloadChapterWritesPagesInOrder(t *testing.T) {
	noRetryWaits(t)
	src := &fakePages{
		// the third URL has no extension and the second lies about it: names come from the bytes
		urls:     []string{"https://x/h/1-a.png", "https://x/h/2-b.jpg", "https://x/h/3-c"},
		images:   map[string][]byte{"https://x/h/1-a.png": pngBytes, "https://x/h/2-b.jpg": webpBytes, "https://x/h/3-c": jpegBytes},
		failures: map[string]int{"https://x/h/2-b.jpg": 1}, // works on the retry
	}
	path := filepath.Join(t.TempDir(), "[0001] Chapter 1.cbz")

	pages, err := downloadChapter(src, "c1", path)
	if err != nil {
		t.Fatalf("downloadChapter: %v", err)
	}
	if pages != 3 {
		t.Errorf("pages = %d, want 3", pages)
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open cbz: %v", err)
	}
	defer r.Close()
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	if got := strings.Join(names, ","); got != "001.png,002.webp,003.jpg" {
		t.Errorf("entries = %s, want 001.png,002.webp,003.jpg", got)
	}
	if n := mangafiles.PageCount(path); n != 3 {
		t.Errorf("PageCount = %d, want 3", n)
	}
	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Error("the temporary .part file should be gone")
	}
}

func TestFailedChapterLeavesNothingBehind(t *testing.T) {
	noRetryWaits(t)
	src := &fakePages{
		urls:   []string{"https://x/h/1.png", "https://x/h/2.png"},
		images: map[string][]byte{"https://x/h/1.png": pngBytes}, // page 2 never loads
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "[0002] Chapter 2.cbz")

	if _, err := downloadChapter(src, "c2", path); err == nil {
		t.Fatal("a chapter with a missing page should fail")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("files left behind: %v", entries)
	}
	if mangafiles.ExistingChapter(dir, "2") != "" {
		t.Error("a failed chapter must not look downloaded, or a retry would skip it")
	}
}

func TestNonImagePageFailsTheChapter(t *testing.T) {
	noRetryWaits(t)
	src := &fakePages{
		urls:   []string{"https://x/h/1.png", "https://x/h/2.png"},
		images: map[string][]byte{"https://x/h/1.png": pngBytes, "https://x/h/2.png": []byte("<html>blocked</html>")},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "[0003] Chapter 3.cbz")

	if _, err := downloadChapter(src, "c3", path); err == nil || !strings.Contains(err.Error(), "not an image") {
		t.Fatalf("err = %v, want a not-an-image failure", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("files left behind: %v", entries)
	}
}

func TestPageExt(t *testing.T) {
	cases := map[string][]byte{".png": pngBytes, ".jpg": jpegBytes, ".webp": webpBytes, ".avif": avifBytes, "": []byte("hello")}
	for want, img := range cases {
		if got := pageExt(img); got != want {
			t.Errorf("pageExt(%q) = %q, want %q", img, got, want)
		}
	}
}

func TestMergeChaptersFillsOnlyWhatMangaDexLacks(t *testing.T) {
	md := []mangadex.Chapter{{ID: "md1", Number: "1"}, {ID: "md2", Number: "2"}, {ID: "md3", Number: "3"}}
	sw := []suwayomi.Chapter{{ID: 11, Number: 1}, {ID: 12, Number: 2}, {ID: 13, Number: 2.5, Title: "Extra"}, {ID: 14, Number: 4}}

	got := mergeChapters(md, sw)
	want := []plannedChapter{
		{ID: "md1", Number: "1", Value: 1},
		{ID: "md2", Number: "2", Value: 2},
		{ID: "13", Number: "2.5", Value: 2.5, Title: "Extra", FromSuwayomi: true},
		{ID: "md3", Number: "3", Value: 3},
		{ID: "14", Number: "4", Value: 4, FromSuwayomi: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("chapter %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	if got := mergeChapters(nil, sw); len(got) != 4 || !got[0].FromSuwayomi {
		t.Errorf("Suwayomi alone = %+v", got)
	}
}
