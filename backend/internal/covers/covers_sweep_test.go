package covers

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

const (
	itemID   = "11111111-1111-1111-1111-111111111111"
	imageID  = "22222222-2222-2222-2222-222222222222"
	orphanID = "33333333-3333-3333-3333-333333333333"
	freshID  = "44444444-4444-4444-4444-444444444444"
	dirID    = "55555555-5555-5555-5555-555555555555"
)

func TestOrphans(t *testing.T) {
	now := time.Now()
	old := now.Add(-2 * sweepGrace)
	files := []coverFile{
		{name: itemID, modTime: old},
		{name: imageID, modTime: old},
		{name: orphanID, modTime: old},
		{name: freshID, modTime: now.Add(-time.Minute)},
		{name: dirID, dir: true, modTime: old},
		{name: "readme.txt", modTime: old},
	}
	referenced := map[string]bool{itemID: true, imageID: true}

	got := orphans(files, referenced, now)
	if want := []string{orphanID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
}

func TestMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEDIA_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, "Covers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Covers", imageID), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := missing([]string{imageID, orphanID, "not-a-uuid"})
	if want := []string{orphanID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing = %v, want %v", got, want)
	}
}
