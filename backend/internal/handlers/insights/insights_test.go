package insights

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", size)), 0644); err != nil {
		t.Fatal(err)
	}
}

func folder(u storageUsage, mediaType string) folderUsage {
	for _, f := range u.Folders {
		if f.Type == mediaType {
			return f
		}
	}
	return folderUsage{}
}

func TestScanAddsUpEachLibraryFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "TV Shows", "Show", "Season 1", "e1.mkv"), 100)
	writeFile(t, filepath.Join(root, "TV Shows", "Show", "Season 1", "e2.mkv"), 50)
	writeFile(t, filepath.Join(root, "Manga", "Title", "c1.cbz"), 30)
	writeFile(t, filepath.Join(root, "temp", "anime-1-a0", "seg0.ts"), 7)
	// Movies, Music and Light Novels don't exist and count as empty

	u := newStorageScanner(root).Get(false)
	if len(u.Folders) != len(libraryFolders) {
		t.Fatalf("got %d folders, want %d", len(u.Folders), len(libraryFolders))
	}
	if f := folder(u, "anime"); f.Bytes != 150 || f.Files != 2 || f.Name != "TV Shows" {
		t.Errorf("anime = %+v, want 150 bytes in 2 files", f)
	}
	if f := folder(u, "manga"); f.Bytes != 30 || f.Files != 1 {
		t.Errorf("manga = %+v, want 30 bytes in 1 file", f)
	}
	if f := folder(u, "movie"); f.Bytes != 0 || f.Files != 0 {
		t.Errorf("missing Movies folder = %+v, want empty", f)
	}
	if u.Temp.Bytes != 7 || u.Temp.Files != 1 {
		t.Errorf("temp = %+v, want 7 bytes in 1 file", u.Temp)
	}
}

func TestScanIsReusedUntilStaleOrRefreshed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Movies", "a.mkv"), 10)
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	s := newStorageScanner(root)
	s.now = func() time.Time { return now }

	s.Get(false)
	writeFile(t, filepath.Join(root, "Movies", "b.mkv"), 10)
	if f := folder(s.Get(false), "movie"); f.Bytes != 10 {
		t.Errorf("recent scan should be reused, got %d bytes", f.Bytes)
	}
	if f := folder(s.Get(true), "movie"); f.Bytes != 20 {
		t.Errorf("refresh should scan again, got %d bytes", f.Bytes)
	}

	writeFile(t, filepath.Join(root, "Movies", "c.mkv"), 10)
	now = now.Add(scanTTL + time.Minute)
	if f := folder(s.Get(false), "movie"); f.Bytes != 30 {
		t.Errorf("stale scan should be redone, got %d bytes", f.Bytes)
	}
}

func TestSummarizeUsersCountsRecentlySeen(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	at := func(days int) *time.Time { t := now.AddDate(0, 0, -days); return &t }
	list := []userActivity{
		{Username: "today", LastSeen: at(0)},
		{Username: "last week", LastSeen: at(7)},
		{Username: "last month", LastSeen: at(20)},
		{Username: "long ago", LastSeen: at(90)},
		{Username: "never"},
	}

	s := summarizeUsers(list, now)
	if s.Total != 5 || s.Active7 != 2 || s.Active30 != 3 {
		t.Errorf("total %d, 7d %d, 30d %d; want 5, 2, 3", s.Total, s.Active7, s.Active30)
	}
}
