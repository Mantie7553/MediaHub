/*
Package musicfiles handles the files beside music tracks: the cover image kept in each
album folder as cover.jpg, which every track in that folder serves as its cover
*/
package musicfiles

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/dhowden/tag"
)

// the cover's file name in an album folder. Uploaded covers keep this name whatever their
// format, so the image type is read from the file itself when it is served
const CoverName = "cover.jpg"

/*
Function:	CoverPath
Purpose:	where the cover for a track's folder is kept
Params:
  - trackPath: the track's file
*/
func CoverPath(trackPath string) string {
	return filepath.Join(filepath.Dir(trackPath), CoverName)
}

/*
Function:	CoverURL
Purpose:	the stored cover URL for a track, signed with a media token when it is sent out
Params:
  - mediaItemID: the track's media item id
*/
func CoverURL(mediaItemID string) string {
	return fmt.Sprintf("/api/stream/music/%s/cover", mediaItemID)
}

/*
Function:	ImageType
Purpose:	the type of an image from its first bytes, "" when it isn't one a browser shows
Params:
  - data: the start of the file, 512 bytes is enough
*/
func ImageType(data []byte) string {
	switch ct := http.DetectContentType(data); ct {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return ct
	default:
		return ""
	}
}

/*
Function:	ExtractCover
Purpose:	write a track's embedded album art to destPath. Returns false when the track has
none or it couldn't be written
Params:
  - trackPath: the mp3 to read
  - destPath: where to write the image
*/
func ExtractCover(trackPath string, destPath string) bool {
	f, err := os.Open(trackPath)
	if err != nil {
		return false
	}
	defer f.Close()

	meta, err := tag.ReadFrom(f)
	if err != nil {
		return false
	}

	pic := meta.Picture()
	if pic == nil || len(pic.Data) == 0 {
		return false
	}

	return WriteCover(destPath, pic.Data) == nil
}

/*
Function:	WriteCover
Purpose:	replace a cover image without a reader ever seeing a half written file
Params:
  - destPath: the cover's path
  - data: the image
*/
func WriteCover(destPath string, data []byte) error {
	tmp := destPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, destPath); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// longest artist or album name kept, well past real names but short of file system limits
const maxNameRunes = 100

// characters that can't be part of a folder name: path separators, and what Windows and SMB
// shares refuse
var unsafeNameChars = strings.NewReplacer(
	"/", "-", "\\", "-", ":", "-", "*", "", "?", "", "\"", "'", "<", "", ">", "", "|", "-",
	// yt-dlp reads %(...)s in its output path as a field of the video, so a % can't reach it
	"%", "",
)

/*
Function:	CleanName
Purpose:	an artist or album name made safe to use as one folder under Music: no path
separators or ".." (which would climb out of the library), no characters file systems refuse,
no % for yt-dlp to read as a template, no control characters or leading dots. "" when nothing
usable is left
Params:
  - name: the name a user typed
*/
func CleanName(name string) string {
	name = unsafeNameChars.Replace(name)
	name = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' '
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, name)
	name = strings.Join(strings.Fields(name), " ")
	// a folder can't be "." or "..", and a leading dot hides it
	name = strings.TrimLeft(name, ". ")
	name = strings.TrimRight(name, ". ")
	if r := []rune(name); len(r) > maxNameRunes {
		name = strings.TrimSpace(string(r[:maxNameRunes]))
	}
	return name
}
