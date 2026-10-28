package deletion

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write creates files (and their folders) under root
func write(t *testing.T, root string, rels ...string) []string {
	t.Helper()
	var paths []string
	for _, rel := range rels {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func notShared(string) (bool, error) { return false, nil }

func TestOwnedFolders(t *testing.T) {
	root := filepath.Join("media", "Music")
	paths := []string{
		filepath.Join(root, "Artist", "Album", "01.mp3"),
		filepath.Join(root, "Artist", "Album", "02.mp3"),
		filepath.Join(root, "Other", "Album", "01.mp3"),
		filepath.Join(root, "loose.mp3"),              // not deep enough
		filepath.Join("media", "Manga", "X", "1.cbz"), // outside root
	}

	got := ownedFolders(root, paths, 2)
	want := []string{filepath.Join(root, "Artist", "Album"), filepath.Join(root, "Other", "Album")}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("depth 2 = %v, want %v", got, want)
	}
	if got := ownedFolders(root, paths, 0); got != nil {
		t.Errorf("depth 0 should have no folders, got %v", got)
	}
}

func TestDeleteFilesRemovesTheTitleFolder(t *testing.T) {
	mediaRoot := t.TempDir()
	root := filepath.Join(mediaRoot, "Manga")
	chapters := write(t, root, "Series/[1] One.cbz", "Series/[2] Two.cbz")
	// left by the download tool, a sync would turn the folder back into a title
	leftovers := write(t, root, "Series/cover.jpg", "Series/series.json")
	keep := write(t, root, "Other/[1] One.cbz")

	if failed := deleteFiles(mediaRoot, root, chapters, 1, notShared); len(failed) > 0 {
		t.Fatalf("failed: %v", failed)
	}
	if exists(filepath.Join(root, "Series")) || exists(leftovers[0]) {
		t.Error("the series folder should be gone, leftovers included")
	}
	if !exists(keep[0]) || !exists(root) {
		t.Error("other titles and the Manga folder must stay")
	}
}

func TestDeleteFilesInASharedFolderOnlyRemovesItsFiles(t *testing.T) {
	mediaRoot := t.TempDir()
	root := filepath.Join(mediaRoot, "TV Shows")
	mine := write(t, root, "Show/Season 1/e1.mkv")
	theirs := write(t, root, "Show/Season 2/e1.mkv")

	shared := func(folder string) (bool, error) { return folder == filepath.Join(root, "Show"), nil }
	if failed := deleteFiles(mediaRoot, root, mine, 1, shared); len(failed) > 0 {
		t.Fatalf("failed: %v", failed)
	}
	if exists(mine[0]) || exists(filepath.Join(root, "Show", "Season 1")) {
		t.Error("this title's file and its emptied folder should be gone")
	}
	if !exists(theirs[0]) {
		t.Error("the other title's file must stay")
	}
}

func TestDeleteFilesSingleTrackPrunesUpToRoot(t *testing.T) {
	mediaRoot := t.TempDir()
	root := filepath.Join(mediaRoot, "Music")
	track := write(t, root, "Artist/Singles/song.mp3")

	if failed := deleteFiles(mediaRoot, root, track, 0, notShared); len(failed) > 0 {
		t.Fatalf("failed: %v", failed)
	}
	if exists(filepath.Join(root, "Artist")) {
		t.Error("emptied folders should be pruned")
	}
	if !exists(root) {
		t.Error("the Music folder itself must never be removed")
	}
}

func TestDeleteFilesRefusesPathsOutsideMediaRoot(t *testing.T) {
	mediaRoot := t.TempDir()
	outside := write(t, t.TempDir(), "important.txt")

	failed := deleteFiles(mediaRoot, filepath.Join(mediaRoot, "Music"), outside, 0, notShared)
	if len(failed) != 1 || !exists(outside[0]) {
		t.Errorf("a path outside MEDIA_ROOT must be refused and kept, failed = %v", failed)
	}
}

func TestDeleteFilesAlreadyMissingIsFine(t *testing.T) {
	mediaRoot := t.TempDir()
	root := filepath.Join(mediaRoot, "Movies")
	gone := filepath.Join(root, "Movie (2020)", "movie.mkv")

	if failed := deleteFiles(mediaRoot, root, []string{gone}, 0, notShared); len(failed) > 0 {
		t.Errorf("a file that's already gone should not fail: %v", failed)
	}
}

func TestDeleteFilesReportsSharedCheckErrors(t *testing.T) {
	mediaRoot := t.TempDir()
	root := filepath.Join(mediaRoot, "Manga")
	chapters := write(t, root, "Series/[1] One.cbz")

	failing := func(string) (bool, error) { return false, errors.New("db down") }
	failed := deleteFiles(mediaRoot, root, chapters, 1, failing)
	if len(failed) == 0 {
		t.Error("an error checking the folder should be reported")
	}
}

func TestRemoveTranscodes(t *testing.T) {
	mediaRoot := t.TempDir()
	write(t, filepath.Join(mediaRoot, "temp"), "episode-abc-a1/playlist.m3u8", "episode-abc-a2/playlist.m3u8", "episode-xyz-a1/playlist.m3u8")

	removeTranscodes(mediaRoot, []string{"episode-abc"})
	if exists(filepath.Join(mediaRoot, "temp", "episode-abc-a1")) || exists(filepath.Join(mediaRoot, "temp", "episode-abc-a2")) {
		t.Error("the episode's transcodes should be gone")
	}
	if !exists(filepath.Join(mediaRoot, "temp", "episode-xyz-a1")) {
		t.Error("other transcodes must stay")
	}
}

func TestRespondError(t *testing.T) {
	h := &Handler{}
	rec := httptest.NewRecorder()
	if !h.respondError(rec, &deleteError{http.StatusConflict, "Still downloading"}) || rec.Code != http.StatusConflict {
		t.Errorf("deleteError should be sent with its status, got %d", rec.Code)
	}
	if h.respondError(httptest.NewRecorder(), nil) {
		t.Error("nil should not respond")
	}
}
