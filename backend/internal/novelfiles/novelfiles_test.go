package novelfiles

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestVolumeNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int
		ok   bool
	}{
		{"Mushoku Tensei Volume 3.epub", 3, true},
		{"mushoku_tensei_volume_12.epub", 12, true},
		{FileName("Spice and Wolf", 7), 7, true},
		{"Spice and Wolf v7.epub", 0, false},
	}
	for _, tt := range tests {
		if n, ok := VolumeNumber(tt.name); n != tt.n || ok != tt.ok {
			t.Errorf("VolumeNumber(%q) = %d, %v; want %d, %v", tt.name, n, ok, tt.n, tt.ok)
		}
	}
}

func TestRenumber(t *testing.T) {
	if got := Renumber("Mushoku Tensei Volume 3.epub", 4); got != "Mushoku Tensei Volume 4.epub" {
		t.Errorf("got %q", got)
	}
	if got := Renumber("x_volume_09_extra.epub", 10); got != "x_volume_10_extra.epub" {
		t.Errorf("got %q", got)
	}
	if got := Renumber("no number.epub", 2); got != "no number.epub" {
		t.Errorf("a name without a number must stay, got %q", got)
	}
}

func writeEPUB(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.epub")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	f.Close()
	return path
}

func TestCheckAndCover(t *testing.T) {
	good := writeEPUB(t, map[string]string{
		"META-INF/container.xml": `<rootfile full-path="OEBPS/content.opf"/>`,
		"OEBPS/content.opf":      `<item id="c" href="Images/cover.jpg" properties="cover-image"/>`,
	})
	if err := Check(good); err != nil {
		t.Errorf("a good EPUB: %v", err)
	}
	if cover, err := CoverHref(good); err != nil || cover != "cover.jpg" {
		t.Errorf("CoverHref = %q, %v", cover, err)
	}

	noOPF := writeEPUB(t, map[string]string{"META-INF/container.xml": `<rootfile full-path="OEBPS/content.opf"/>`})
	if err := Check(noOPF); err == nil {
		t.Error("an EPUB missing its contents file must be refused")
	}
	noContainer := writeEPUB(t, map[string]string{"chapter.html": "<p>hi</p>"})
	if err := Check(noContainer); err == nil {
		t.Error("a ZIP without container.xml must be refused")
	}

	notZip := filepath.Join(t.TempDir(), "x.epub")
	os.WriteFile(notZip, []byte("plain text"), 0o644)
	if err := Check(notZip); err == nil {
		t.Error("a file that isn't a ZIP must be refused")
	}
}
