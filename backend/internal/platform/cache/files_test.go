package cache

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// a cache in a temp folder that waits for its background copies before the folder is removed
func newTestFiles(t *testing.T, max int64) *Files {
	t.Helper()
	f, err := NewFiles(t.TempDir(), max)
	if err != nil {
		t.Fatal(err)
	}
	// cleanups run newest first, so this runs before TempDir deletes the folder
	t.Cleanup(f.copies.Wait)
	return f
}

// waits for the background copy of src, returning the cached path
func cachedPath(t *testing.T, f *Files, src string) string {
	t.Helper()
	var got string
	waitFor(t, func() bool { got = f.Local(src); return got != src })
	return got
}

func entryCount(f *Files) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.entries)
}

func TestLocalServesTheSourceUntilCopied(t *testing.T) {
	nas := t.TempDir()
	src := filepath.Join(nas, "Chapter 1.cbz")
	writeFile(t, src, "pages")
	f := newTestFiles(t, 1<<20)
	dir := f.dir

	if got := f.Local(src); got != src {
		t.Errorf("first read = %s, want the source while it copies", got)
	}
	local := cachedPath(t, f, src)
	if filepath.Dir(local) != dir || filepath.Ext(local) != ".cbz" {
		t.Errorf("cached path = %s, want a .cbz in the cache folder", local)
	}
	if data, _ := os.ReadFile(local); string(data) != "pages" {
		t.Errorf("cached copy = %q", data)
	}
}

func TestLocalCopiesAgainWhenTheSourceChanges(t *testing.T) {
	nas := t.TempDir()
	src := filepath.Join(nas, "cover.jpg")
	writeFile(t, src, "old")
	f := newTestFiles(t, 1<<20)
	old := cachedPath(t, f, src)

	writeFile(t, src, "new cover")
	later := time.Now().Add(time.Hour)
	os.Chtimes(src, later, later)
	f.Forget(src)
	if got := f.Local(src); got != src {
		t.Errorf("read after a change = %s, want the source, not the old copy", got)
	}
	if fresh := cachedPath(t, f, src); fresh == old {
		t.Error("a changed file should get a new copy")
	}
}

func TestLocalSkipsMissingAndHugeFiles(t *testing.T) {
	nas := t.TempDir()
	f := newTestFiles(t, 100)

	missing := filepath.Join(nas, "gone.mp3")
	if got := f.Local(missing); got != missing {
		t.Errorf("missing file = %s, want the source path back", got)
	}

	big := filepath.Join(nas, "big.epub")
	writeFile(t, big, strings.Repeat("x", 30))
	f.Local(big)
	time.Sleep(50 * time.Millisecond)
	if got := f.Local(big); got != big || entryCount(f) != 0 {
		t.Error("a file over a quarter of the cache shouldn't be copied")
	}
}

func TestEvictsLeastRecentlyUsed(t *testing.T) {
	nas := t.TempDir()
	f := newTestFiles(t, 100)

	var srcs []string
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		src := filepath.Join(nas, name)
		writeFile(t, src, strings.Repeat("x", 25))
		cachedPath(t, f, src)
		srcs = append(srcs, src)
		time.Sleep(5 * time.Millisecond)
	}
	if f.total > 100 || entryCount(f) != 4 {
		t.Errorf("total = %d, entries = %d; want 4 copies within 100 bytes", f.total, entryCount(f))
	}
	f.Forget(srcs[0])
	if got := f.Local(srcs[0]); got != srcs[0] {
		t.Error("the least recently used copy should have been deleted")
	}
}

func TestNewFilesKeepsCopiesAndDropsUnfinishedOnes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "kept.cbz"), "12345")
	writeFile(t, filepath.Join(dir, "half.cbz.tmp"), "12")

	f, err := NewFiles(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if f.total != 5 || entryCount(f) != 1 {
		t.Errorf("total = %d, entries = %d; want the one finished copy", f.total, entryCount(f))
	}
	if _, err := os.Stat(filepath.Join(dir, "half.cbz.tmp")); !os.IsNotExist(err) {
		t.Error("unfinished copies should be deleted")
	}
}

func TestRemoteFetchesOnce(t *testing.T) {
	f := newTestFiles(t, 1<<20)
	calls := 0
	fetch := func() ([]byte, error) { calls++; return []byte("img"), nil }

	for i := 0; i < 2; i++ {
		if data, err := f.Remote("https://example/a.jpg", fetch); err != nil || string(data) != "img" {
			t.Fatalf("Remote = %q, %v", data, err)
		}
	}
	if calls != 1 {
		t.Errorf("fetched %d times, want 1", calls)
	}
	if _, err := f.Remote("https://example/b.jpg", func() ([]byte, error) { return nil, errors.New("down") }); err == nil {
		t.Error("want the fetch error")
	}
}

func TestStats(t *testing.T) {
	nas := t.TempDir()
	f := newTestFiles(t, 1000)
	src := filepath.Join(nas, "a")
	writeFile(t, src, strings.Repeat("x", 40))
	cachedPath(t, f, src)

	used, max, dir := f.Stats()
	if used != 40 || max != 1000 || dir != f.dir {
		t.Errorf("Stats = %d, %d, %s; want 40, 1000, %s", used, max, dir, f.dir)
	}

	var off *Files
	if used, max, dir := off.Stats(); used != 0 || max != 0 || dir != "" {
		t.Errorf("nil Stats = %d, %d, %q; want zeros", used, max, dir)
	}
}

func TestNilFilesPassesThrough(t *testing.T) {
	var f *Files
	if got := f.Local("/nas/x.mp3"); got != "/nas/x.mp3" {
		t.Errorf("Local = %s", got)
	}
	f.Forget("/nas/x.mp3")
	if data, _ := f.Remote("k", func() ([]byte, error) { return []byte("v"), nil }); string(data) != "v" {
		t.Errorf("Remote = %q", data)
	}
}
