package manga

import (
	"archive/zip"
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func archive(t *testing.T, names ...string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("img"))
	}
	zw.Close()
	return buf.Bytes()
}

func TestSaveChapterKeepsGoodArchives(t *testing.T) {
	path := filepath.Join(t.TempDir(), "[0098] Chapter 98.cbz")
	pages, _, err := saveChapter(bytes.NewReader(archive(t, "ComicInfo.xml", "01.jpg", "02.jpg")), path)
	if err != nil {
		t.Fatalf("saveChapter: %v", err)
	}
	if pages != 2 {
		t.Errorf("pages = %d, want 2 (ComicInfo.xml isn't a page)", pages)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("chapter file missing: %v", err)
	}
	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Error("the temporary file should be gone")
	}
}

func TestSaveChapterRejectsBadUploads(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string][]byte{
		"not a zip":   []byte("this is not an archive"),
		"no pictures": archive(t, "ComicInfo.xml", "readme.txt"),
	} {
		path := filepath.Join(dir, "[0001] Chapter 1.cbz")
		_, status, err := saveChapter(bytes.NewReader(body), path)
		if err == nil || status != http.StatusBadRequest {
			t.Errorf("%s: status %d, err %v; want a 400", name, status, err)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("rejected uploads left files behind: %v", entries)
	}
}

func TestChapterFromForm(t *testing.T) {
	cases := []struct {
		number, file, want string
		ok                 bool
	}{
		{"", "Chainsaw Man c098.cbz", "98", true},
		{"099", "whatever.cbz", "99", true},
		{"52.5", "", "52.5", true},
		{"abc", "c098.cbz", "", false},
		{"-1", "c098.cbz", "", false},
		{"", "Chainsaw Man v12.cbz", "", false},
	}
	for _, c := range cases {
		ch, ok := chapterFromForm(c.number, c.file, "Chainsaw Man")
		if ok != c.ok || (ok && ch.Number != c.want) {
			t.Errorf("chapterFromForm(%q, %q) = %q, %v; want %q, %v", c.number, c.file, ch.Number, ok, c.want, c.ok)
		}
		if ok && strings.HasPrefix(ch.Number, "0") && ch.Number != "0" {
			t.Errorf("number %q should have no leading zeros", ch.Number)
		}
	}
}
