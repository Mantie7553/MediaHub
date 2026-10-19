package mangafiles

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseChapter(t *testing.T) {
	cases := []struct {
		name, series string
		number       string
		title        string
		ok           bool
	}{
		{"[0098] Chapter 98 - Katana Man.cbz", "", "98", "Katana Man", true},
		{"[0052.5] Chapter 52.5.cbz", "", "52.5", "", true},
		{"[0000] Oneshot - Special.cbz", "", "0", "", true},
		{"Chainsaw Man c098.cbz", "Chainsaw Man", "98", "", true},
		{"Chainsaw Man v12 c098 (Digital) [Group].cbz", "Chainsaw Man", "98", "", true},
		{"Chainsaw_Man_Ch.98.zip", "Chainsaw Man", "98", "", true},
		{"Chapter 101.5.cbz", "", "101.5", "", true},
		{"ch 7.cbz", "", "7", "", true},
		{"#12.cbz", "", "12", "", true},
		{"Chainsaw Man 098.cbz", "Chainsaw Man", "98", "", true},
		{"Kaiju No. 8 012.cbz", "Kaiju No. 8", "12", "", true},
		{"Kaiju No. 8 c012.cbz", "", "12", "", true},
		{"Chainsaw Man v12.cbz", "Chainsaw Man", "", "", false},
		{"Chainsaw Man Vol. 3 (Digital).cbz", "Chainsaw Man", "", "", false},
		{"cover.cbz", "", "", "", false},
		{"Chapter 52.5", "", "52.5", "", true},
	}
	for _, c := range cases {
		ch, ok := ParseChapter(c.name, c.series)
		if ok != c.ok || ch.Number != c.number || ch.Title != c.title {
			t.Errorf("ParseChapter(%q, %q) = %+v, %v; want number %q title %q ok %v", c.name, c.series, ch, ok, c.number, c.title, c.ok)
		}
	}
}

func TestFileNameRoundTrips(t *testing.T) {
	for _, c := range []struct{ number, title, want string }{
		{"1", "A Dog and a Chainsaw", "[0001] Chapter 1 - A Dog and a Chainsaw.cbz"},
		{"52.5", "", "[0052.5] Chapter 52.5.cbz"},
		{"", "Special", "[0000] Oneshot - Special.cbz"},
		{"7", `Who? / What: "Why"`, "[0007] Chapter 7 - Who  What Why.cbz"},
	} {
		got := FileName(c.number, c.title)
		if got != c.want {
			t.Errorf("FileName(%q, %q) = %q, want %q", c.number, c.title, got, c.want)
		}
		if c.number == "" {
			continue
		}
		if back, ok := ParseChapter(got, ""); !ok || back.Number != c.number {
			t.Errorf("ParseChapter(%q) = %+v, %v; want number %q", got, back, ok, c.number)
		}
	}
}

func TestSanitizeFileName(t *testing.T) {
	if got := SanitizeFileName(`  Re:Zero?.. `); got != "ReZero" {
		t.Errorf("got %q, want ReZero", got)
	}
	if got := SanitizeFileName(`???`); got != "Untitled" {
		t.Errorf("got %q, want Untitled", got)
	}
}

// zipWith builds an archive holding the named (empty) files
func zipWith(t *testing.T, names ...string) *zip.Reader {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		if _, err := zw.Create(n); err != nil {
			t.Fatal(err)
		}
	}
	zw.Close()
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestPagesSkipsExtrasAndSortsNaturally(t *testing.T) {
	r := zipWith(t,
		"ComicInfo.xml", "pages/", "pages/10.jpg", "pages/2.jpg", "pages/1.PNG",
		"__MACOSX/pages/._1.PNG", "pages/._2.jpg", "notes.txt",
	)
	var names []string
	for _, f := range Pages(r) {
		names = append(names, f.Name)
	}
	if got := strings.Join(names, ","); got != "pages/1.PNG,pages/2.jpg,pages/10.jpg" {
		t.Errorf("pages = %s", got)
	}
}

func TestExistingChapterMatchesTheNumberWhateverTheTitle(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "[0001] Chapter 1 - Old Title.cbz"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "[0010] Chapter 10.cbz.part"), []byte("x"), 0o644)

	if got := ExistingChapter(dir, "1"); filepath.Base(got) != "[0001] Chapter 1 - Old Title.cbz" {
		t.Errorf("chapter 1 = %q, want the existing file", got)
	}
	if got := ExistingChapter(dir, "10"); got != "" {
		t.Errorf("an unfinished .part file should not count, got %q", got)
	}
	if got := ExistingChapter(dir, "2"); got != "" {
		t.Errorf("chapter 2 = %q, want none", got)
	}
}
