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
)

func TestChapterFileName(t *testing.T) {
	cases := []struct {
		ch   mangadex.Chapter
		want string
	}{
		{mangadex.Chapter{Number: "1", Title: "A Dog and a Chainsaw"}, "[0001] Chapter 1 - A Dog and a Chainsaw.cbz"},
		{mangadex.Chapter{Number: "52.5"}, "[0052.5] Chapter 52.5.cbz"},
		{mangadex.Chapter{Number: "", Title: "Special"}, "[0000] Oneshot - Special.cbz"},
		{mangadex.Chapter{Number: "7", Title: `Who? / What: "Why"`}, "[0007] Chapter 7 - Who  What Why.cbz"},
	}
	for _, c := range cases {
		if got := chapterFileName(c.ch); got != c.want {
			t.Errorf("chapterFileName(%+v) = %q, want %q", c.ch, got, c.want)
		}
	}
}

func TestSanitizeFileName(t *testing.T) {
	if got := sanitizeFileName(`  Re:Zero?.. `); got != "ReZero" {
		t.Errorf("got %q, want ReZero", got)
	}
	if got := sanitizeFileName(`???`); got != "Untitled" {
		t.Errorf("got %q, want Untitled", got)
	}
}

func TestExistingChapterMatchesTheNumberWhateverTheTitle(t *testing.T) {
	dir := t.TempDir()
	// what mangal left behind, with a title MangaDex may have since changed
	os.WriteFile(filepath.Join(dir, "[0001] Chapter 1 - Old Title.cbz"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "[0010] Chapter 10.cbz.part"), []byte("x"), 0o644)

	if got := existingChapter(dir, mangadex.Chapter{Number: "1", Title: "New Title"}); filepath.Base(got) != "[0001] Chapter 1 - Old Title.cbz" {
		t.Errorf("chapter 1 = %q, want the existing file", got)
	}
	if got := existingChapter(dir, mangadex.Chapter{Number: "10"}); got != "" {
		t.Errorf("an unfinished .part file should not count, got %q", got)
	}
	if got := existingChapter(dir, mangadex.Chapter{Number: "2"}); got != "" {
		t.Errorf("chapter 2 = %q, want none", got)
	}
}

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
	if cbzPageCount(path) != 3 {
		t.Errorf("cbzPageCount = %d, want 3", cbzPageCount(path))
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
	if existingChapter(dir, mangadex.Chapter{Number: "2"}) != "" {
		t.Error("a failed chapter must not look downloaded, or a retry would skip it")
	}
}
