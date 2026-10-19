/*
Package mangafiles holds the rules for manga chapter files on disk: reading a chapter number
out of a file name, naming the files MediaHub writes, and finding the page images inside a
CBZ. The downloader, the library sync, chapter uploads and the reader all share them
*/
package mangafiles

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Chapter is a chapter number and title read from a file name
type Chapter struct {
	// Number as written, "98" or "52.5", the form used in file names
	Number string
	// Value is Number as a number, for the database
	Value float64
	// Title is the chapter's own title when the name has one
	Title string
}

var (
	// [0098] Chapter 98 - Title, what MediaHub (and mangal before it) writes
	bracketed = regexp.MustCompile(`^\[(\d+(?:\.\d+)?)\]\s*(.*)$`)
	// c098, ch98, ch. 98, chapter 98, #98
	marked = regexp.MustCompile(`(?i)(?:^|[^a-z])(?:chapter|chap|ch|c)\.?\s*(\d+(?:\.\d+)?)|#(\d+(?:\.\d+)?)`)
	// v12, vol 12, vol. 12, volume 12
	volume = regexp.MustCompile(`(?i)(?:^|[^a-z])(?:volume|vol|v)\.?\s*\d+`)
	// any number, used when nothing marks the chapter
	number = regexp.MustCompile(`\d+(?:\.\d+)?`)
	// only digits, how a decimal chapter number looks after its point
	digits = regexp.MustCompile(`^\d+$`)
	// a "- Title" after the number
	titleAfter = regexp.MustCompile(`(?i)^\s*(?:chapter|ch)?\.?\s*[\d.]*\s*[-–:]\s*(.+)$`)
	// scanlator tags like [Group] or (Digital)
	tags = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)|\{[^}]*\}`)
)

/*
Function:	ParseChapter
Purpose:	Read the chapter number (and title, when there is one) from a chapter file's name.
Understands "[0098] Chapter 98 - Title.cbz", "Chainsaw Man c098.cbz", "Ch. 98.cbz",
"Chainsaw Man v12 c098 (Digital).cbz", "#98.cbz" and "Chainsaw Man 098.cbz"

Params:
  - fileName: the file's name, with or without the extension
  - series: the series title, removed first so a number in it ("Kaiju No. 8") isn't read
    as the chapter. May be empty

Returns:	the chapter, and false when the name has no chapter number, for example a
whole volume ("Chainsaw Man v12.cbz")
*/
func ParseChapter(fileName string, series string) (Chapter, bool) {
	// "Chapter 52.5" has no extension, its ".5" is part of the number
	base := fileName
	if ext := filepath.Ext(fileName); ext != "" && !digits.MatchString(ext[1:]) {
		base = strings.TrimSuffix(fileName, ext)
	}

	if m := bracketed.FindStringSubmatch(base); m != nil {
		ch, ok := newChapter(m[1])
		if ok {
			if t := titleAfter.FindStringSubmatch(m[2]); t != nil {
				ch.Title = strings.TrimSpace(t[1])
			}
		}
		return ch, ok
	}

	name := tags.ReplaceAllString(base, " ")
	name = strings.ReplaceAll(name, "_", " ")
	if series != "" {
		if i := strings.Index(strings.ToLower(name), strings.ToLower(series)); i >= 0 {
			name = name[:i] + " " + name[i+len(series):]
		}
	}

	if m := marked.FindStringSubmatch(name); m != nil {
		num := m[1]
		if num == "" {
			num = m[2]
		}
		return newChapter(num)
	}

	// without a chapter marker, a volume marker means the file is a whole volume
	rest := volume.ReplaceAllString(name, " ")
	if rest != name && number.FindString(rest) == "" {
		return Chapter{}, false
	}
	if m := number.FindAllString(rest, -1); len(m) > 0 {
		return newChapter(m[len(m)-1])
	}
	return Chapter{}, false
}

func newChapter(text string) (Chapter, bool) {
	v, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return Chapter{}, false
	}
	// "098" and "98" are the same chapter, keep the short form
	whole, frac, hasFrac := strings.Cut(text, ".")
	whole = strings.TrimLeft(whole, "0")
	if whole == "" {
		whole = "0"
	}
	if hasFrac {
		text = whole + "." + frac
	} else {
		text = whole
	}
	return Chapter{Number: text, Value: v}, true
}

/*
Function:	FileName
Purpose:	The name MediaHub gives a chapter file: "[0098] Chapter 98 - Title.cbz", or
"[0000] Oneshot.cbz" for a chapter without a number. ParseChapter reads it back
*/
func FileName(number string, title string) string {
	name := "Oneshot"
	if number != "" {
		name = "Chapter " + number
	}
	if title != "" {
		name += " - " + title
	}
	return SanitizeFileName(fmt.Sprintf("[%s] %s", PaddedNumber(number), name)) + ".cbz"
}

// PaddedNumber writes a chapter number with 4 whole digits: 12 -> 0012, 52.5 -> 0052.5
func PaddedNumber(number string) string {
	whole, frac, _ := strings.Cut(number, ".")
	n, err := strconv.Atoi(whole)
	if err != nil {
		n = 0
	}
	if frac != "" {
		return fmt.Sprintf("%04d.%s", n, frac)
	}
	return fmt.Sprintf("%04d", n)
}

// SanitizeFileName drops characters Windows and SMB shares refuse in file names
func SanitizeFileName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimRight(strings.TrimSpace(name), ".")
	if name == "" {
		return "Untitled"
	}
	return name
}

// IsArchive reports whether a file is a chapter archive MediaHub can read (CBZ or ZIP)
func IsArchive(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".cbz", ".zip":
		return true
	}
	return false
}

// imageTypes are the page formats the reader serves, by extension
var imageTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
	".avif": "image/avif",
}

// ImageType is the content type of a page image, "" when the file isn't one
func ImageType(name string) string {
	return imageTypes[strings.ToLower(filepath.Ext(name))]
}

/*
Function:	Pages
Purpose:	The page images in a chapter archive, in reading order. Other files (ComicInfo.xml,
folders, thumbnails' metadata) are left out, and names are compared naturally so
"2.jpg" comes before "10.jpg" even when they aren't zero padded
*/
func Pages(r *zip.Reader) []*zip.File {
	var pages []*zip.File
	for _, f := range r.File {
		if f.FileInfo().IsDir() || ImageType(f.Name) == "" {
			continue
		}
		// macOS zips carry "._name" resource files next to the real images
		if strings.HasPrefix(filepath.Base(f.Name), "._") || strings.Contains(f.Name, "__MACOSX/") {
			continue
		}
		pages = append(pages, f)
	}
	sort.SliceStable(pages, func(i, j int) bool {
		return naturalLess(pages[i].Name, pages[j].Name)
	})
	return pages
}

// PageCount counts the page images in a chapter archive, 0 when it can't be opened
func PageCount(path string) int {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return 0
	}
	defer zr.Close()
	return len(Pages(&zr.Reader))
}

/*
Function:	ExistingChapter
Purpose:	The path of a file in dir that MediaHub named for this chapter number, whatever its
title says, or "" when there is none
*/
func ExistingChapter(dir string, number string) string {
	prefix := "[" + PaddedNumber(number) + "]"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasPrefix(name, prefix) && strings.EqualFold(filepath.Ext(name), ".cbz") {
			return filepath.Join(dir, name)
		}
	}
	return ""
}

// naturalLess compares names with runs of digits as numbers: page2 < page10
func naturalLess(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	for a != "" && b != "" {
		ra, rb := rune(a[0]), rune(b[0])
		if unicode.IsDigit(ra) && unicode.IsDigit(rb) {
			na, restA := leadingNumber(a)
			nb, restB := leadingNumber(b)
			if na != nb {
				return na < nb
			}
			a, b = restA, restB
			continue
		}
		if ra != rb {
			return ra < rb
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func leadingNumber(s string) (int, string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	n, _ := strconv.Atoi(s[:i])
	return n, s[i:]
}
