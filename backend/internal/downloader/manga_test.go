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

func noRetryWaits(t *testing.T) {
	saved := imageRetryDelays
	imageRetryDelays = []time.Duration{0, 0}
	t.Cleanup(func() { imageRetryDelays = saved })
}

func TestDownloadChapterWritesPagesInOrder(t *testing.T) {
	noRetryWaits(t)
	src := &fakePages{
		urls:     []string{"https://x/h/1-a.png", "https://x/h/2-b.jpg", "https://x/h/3-c"},
		images:   map[string][]byte{"https://x/h/1-a.png": []byte("one"), "https://x/h/2-b.jpg": []byte("two"), "https://x/h/3-c": []byte("three")},
		failures: map[string]int{"https://x/h/2-b.jpg": 1}, // works on the retry
	}
	path := filepath.Join(t.TempDir(), "[0001] Chapter 1.cbz")

	pages, err := downloadChapter(src, mangadex.Chapter{ID: "c1", Number: "1"}, path)
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
	if got := strings.Join(names, ","); got != "001.png,002.jpg,003.jpg" {
		t.Errorf("entries = %s, want 001.png,002.jpg,003.jpg", got)
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
		images: map[string][]byte{"https://x/h/1.png": []byte("one")}, // page 2 never loads
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "[0002] Chapter 2.cbz")

	if _, err := downloadChapter(src, mangadex.Chapter{ID: "c2", Number: "2"}, path); err == nil {
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
