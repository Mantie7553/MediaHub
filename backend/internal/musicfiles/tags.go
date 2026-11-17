package musicfiles

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
)

// Tags: what a track's ID3 tags say about it. Zero values are missing tags
type Tags struct {
	Title  string
	Artist string
	Album  string
	Track  int
	Year   int
	Genres []string
}

/*
Function:	ReadTags
Purpose:	a track's ID3 tags. Returns false when the file has none or can't be read
Params:
  - trackPath: the mp3 to read
*/
func ReadTags(trackPath string) (Tags, bool) {
	f, err := os.Open(trackPath)
	if err != nil {
		return Tags{}, false
	}
	defer f.Close()

	meta, err := tag.ReadFrom(f)
	if err != nil {
		return Tags{}, false
	}
	track, _ := meta.Track()
	return Tags{
		Title:  clean(meta.Title()),
		Artist: clean(meta.Artist()),
		Album:  clean(meta.Album()),
		Track:  max(track, 0),
		Year:   validYear(meta.Year()),
		Genres: SplitGenres(meta.Genre()),
	}, true
}

// clean trims a tag, dropping the NUL padding some taggers leave
func clean(s string) string {
	return strings.TrimSpace(strings.Trim(s, "\x00"))
}

func validYear(y int) int {
	if y < 1000 || y > 9999 {
		return 0
	}
	return y
}

// an ID3v1 genre number, like "(17)", that tag couldn't turn into a name
var genreCode = regexp.MustCompile(`^\(?\d+\)?$`)

/*
Function:	SplitGenres
Purpose:	a genre tag as a list: split on the separators taggers use (";", "/", ",", NUL),
trimmed, without bare genre numbers or repeats
Params:
  - genre: the tag, e.g. "J-Pop; Anime"
*/
func SplitGenres(genre string) []string {
	parts := strings.FieldsFunc(genre, func(r rune) bool {
		return r == ';' || r == '/' || r == ',' || r == 0
	})
	var out []string
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || genreCode.MatchString(p) || seen[strings.ToLower(p)] {
			continue
		}
		seen[strings.ToLower(p)] = true
		out = append(out, p)
	}
	return out
}

/*
Function:	Duration
Purpose:	a track's length in whole seconds from ffprobe, 0 when it can't be read
Params:
  - trackPath: the mp3 to read
*/
func Duration(trackPath string) int {
	out, err := exec.Command("ffprobe", "-v", "error",
		// read as a local mp3 and nothing else, like uploads are
		"-protocol_whitelist", "file", "-format_whitelist", "mp3",
		"-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", trackPath).Output()
	if err != nil {
		return 0
	}
	return parseDuration(string(out))
}

func parseDuration(s string) int {
	secs, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || secs <= 0 {
		return 0
	}
	return int(secs + 0.5)
}
