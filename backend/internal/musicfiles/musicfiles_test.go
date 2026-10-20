package musicfiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageType(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00"), "image/jpeg"},
		{"png", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), "image/png"},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "image/webp"},
		{"gif", []byte("GIF89a\x01\x00\x01\x00"), "image/gif"},
		{"html is not an image", []byte("<html><body>hi</body></html>"), ""},
		{"svg is not accepted", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), ""},
		{"empty", nil, ""},
	}
	for _, c := range cases {
		if got := ImageType(c.data); got != c.want {
			t.Errorf("%s: ImageType = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestWriteCoverReplacesTheOldCover(t *testing.T) {
	path := filepath.Join(t.TempDir(), CoverName)
	if err := WriteCover(path, []byte("old")); err != nil {
		t.Fatal(err)
	}
	if err := WriteCover(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("cover = %q, want %q", got, "new")
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("the temporary file should be gone")
	}
}

func TestCoverPathIsBesideTheTrack(t *testing.T) {
	track := filepath.Join("Music", "Artist", "Album", "01 Song.mp3")
	want := filepath.Join("Music", "Artist", "Album", "cover.jpg")
	if got := CoverPath(track); got != want {
		t.Errorf("CoverPath = %q, want %q", got, want)
	}
}

func TestCleanName(t *testing.T) {
	cases := map[string]string{
		"YOASOBI":                 "YOASOBI",
		"  Kenshi   Yonezu ":      "Kenshi Yonezu",
		"AC/DC":                   "AC-DC",
		"../../../etc":            "-..-..-etc",
		"..":                      "",
		".hidden":                 "hidden",
		"Album.":                  "Album",
		"%(title)s":               "(title)s",
		"100% Pure":               "100 Pure",
		"What? Why: \"Live\" <1>": "What Why- 'Live' 1",
		"tab\tand\nnewline":       "tab and newline",
		`C:\Windows`:              "C--Windows",
		"アイドル":                    "アイドル",
	}
	for in, want := range cases {
		if got := CleanName(in); got != want {
			t.Errorf("CleanName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := CleanName(strings.Repeat("a", 150)); len([]rune(got)) != maxNameRunes {
		t.Errorf("long name kept %d characters, want %d", len([]rune(got)), maxNameRunes)
	}
}

func TestCleanNameNeverLeavesTheFolder(t *testing.T) {
	root := filepath.Join("media", "Music")
	for _, name := range []string{"..", "../..", `..\..`, "/etc", "a/../../b", " . ", "...."} {
		cleaned := CleanName(name)
		if cleaned == "" {
			continue
		}
		path := filepath.Join(root, cleaned, cleaned)
		if !strings.HasPrefix(path, root+string(filepath.Separator)) || strings.Count(path, string(filepath.Separator)) != 3 {
			t.Errorf("CleanName(%q) = %q gives %s, outside one artist/album folder", name, cleaned, path)
		}
	}
}
