package deletion

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

/*
Function:	libraryFolder
Purpose:	the folder under MEDIA_ROOT a media type lives in, and how many levels below it a
title's own folder sits (TV Shows/<Series>, Music/<Artist>/<Album>). 0 means the
title has no folder of its own, only files (a single track in Music/<Artist>/Singles)
Params:
  - mediaType: the media_items type, or "music_album" for a whole album
*/
func libraryFolder(mediaType string) (string, int) {
	switch mediaType {
	case "anime":
		return "TV Shows", 1
	case "movie":
		return "Movies", 1
	case "manga":
		return "Manga", 1
	case "light_novel":
		return "Light Novels", 1
	case "music_album":
		return "Music", 2
	default:
		return "Music", 0
	}
}

/*
Function:	ownedFolders
Purpose:	the title folders (depth levels below root) that hold the given files. Files outside
root, or not deep enough to have a title folder, are left out
Params:
  - root: the media type's folder, e.g. MEDIA_ROOT/Manga
  - paths: the title's files
  - depth: levels below root that make up a title folder
*/
func ownedFolders(root string, paths []string, depth int) []string {
	if depth <= 0 {
		return nil
	}
	seen := map[string]bool{}
	var folders []string
	for _, p := range paths {
		if !utils.IsWithin(root, p) {
			continue
		}
		rel, err := filepath.Rel(root, filepath.Clean(p))
		if err != nil {
			continue
		}
		parts := strings.Split(rel, string(filepath.Separator))
		// the file itself is the last part, so a title folder needs depth parts before it
		if len(parts) <= depth {
			continue
		}
		folder := filepath.Join(append([]string{root}, parts[:depth]...)...)
		if !seen[folder] {
			seen[folder] = true
			folders = append(folders, folder)
		}
	}
	return folders
}

/*
Function:	deleteFiles
Purpose:	remove a title's files. A title folder no other title uses is removed whole (with the
covers and metadata the download tools leave next to the files, which a sync would
otherwise import again); anything else is removed file by file, then emptied parent
folders up to root. Every path must be inside mediaRoot
Params:
  - mediaRoot: MEDIA_ROOT
  - root: the media type's folder
  - paths: the title's files
  - depth: levels below root that make up a title folder, see libraryFolder
  - shared: reports whether another title has files in a folder

Returns: the paths that could not be removed
*/
func deleteFiles(mediaRoot, root string, paths []string, depth int, shared func(folder string) (bool, error)) []string {
	var failed []string
	removedFolders := map[string]bool{}

	for _, folder := range ownedFolders(root, paths, depth) {
		isShared, err := shared(folder)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", folder, err))
			continue
		}
		if isShared {
			continue
		}
		if err := os.RemoveAll(folder); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", folder, err))
			continue
		}
		removedFolders[folder] = true
		pruneEmptyDirs(root, filepath.Dir(folder))
	}

	for _, p := range paths {
		if insideAny(p, removedFolders) {
			continue
		}
		if !utils.IsWithin(mediaRoot, p) {
			failed = append(failed, fmt.Sprintf("%s (outside the media folder)", p))
			continue
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			failed = append(failed, fmt.Sprintf("%s (%v)", p, err))
			continue
		}
		pruneEmptyDirs(root, filepath.Dir(p))
	}
	return failed
}

/*
Function:	pruneEmptyDirs
Purpose:	remove dir and its parents while they are empty, stopping at root (never removed)
Params:
  - root: the folder to stop at
  - dir: the first folder to check
*/
func pruneEmptyDirs(root, dir string) {
	for utils.IsWithin(root, dir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

/*
Function:	removeTranscodes
Purpose:	best effort removal of finished or abandoned HLS transcodes in MEDIA_ROOT/temp
Params:
  - mediaRoot: MEDIA_ROOT
  - prefixes: transcode folder prefixes, e.g. "episode-<id>" (one folder per audio track follows)
*/
func removeTranscodes(mediaRoot string, prefixes []string) {
	tempDir := filepath.Join(mediaRoot, "temp")
	for _, prefix := range prefixes {
		matches, _ := filepath.Glob(filepath.Join(tempDir, prefix+"-*"))
		for _, m := range matches {
			if utils.IsWithin(tempDir, m) {
				os.RemoveAll(m)
			}
		}
	}
}

func insideAny(path string, folders map[string]bool) bool {
	for folder := range folders {
		if utils.IsWithin(folder, path) {
			return true
		}
	}
	return false
}
