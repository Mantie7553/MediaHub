/*
Package novelfiles holds the rules for light novel files on disk: reading a volume number out
of a file name, naming the files MediaHub writes, checking an EPUB opens, and finding its cover.
The library sync and volume uploads share them
*/
package novelfiles

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// "Volume 3", "volume_3", how the library sync has always found a volume's number
var volumeRe = regexp.MustCompile(`(?i)volume[_\s](\d+)`)

/*
Function:	VolumeNumber
Purpose:	the volume number in a file name such as "Mushoku Tensei Volume 3.epub"
Returns:	the number, and false when the name has none
*/
func VolumeNumber(fileName string) (int, bool) {
	m := volumeRe.FindStringSubmatch(fileName)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

/*
Function:	FileName
Purpose:	the name MediaHub gives a volume it saves, which VolumeNumber reads back
Params:
  - series: the light novel's folder name
  - number: the volume number
*/
func FileName(series string, number int) string {
	return fmt.Sprintf("%s Volume %d.epub", series, number)
}

/*
Function:	Renumber
Purpose:	a file name with its volume number changed, "X Volume 3.epub" -> "X Volume 4.epub",
so VolumeNumber reads the new one. A name with no number is returned unchanged
*/
func Renumber(fileName string, number int) string {
	loc := volumeRe.FindStringSubmatchIndex(fileName)
	if loc == nil {
		return fileName
	}
	return fileName[:loc[2]] + strconv.Itoa(number) + fileName[loc[3]:]
}

// IsEPUB reports whether a file name is an EPUB
func IsEPUB(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".epub")
}

/*
Function:	Check
Purpose:	make sure a file is an EPUB the reader can open: a ZIP whose container.xml names a
package (OPF) file that is in the archive
*/
func Check(filePath string) error {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("the file isn't a readable EPUB")
	}
	defer zr.Close()

	opf, err := opfPath(&zr.Reader)
	if err != nil {
		return err
	}
	if opf == "" {
		return fmt.Errorf("the EPUB has no META-INF/container.xml naming its contents")
	}
	for _, f := range zr.File {
		if f.Name == opf {
			return nil
		}
	}
	return fmt.Errorf("the EPUB's contents file %s is missing", opf)
}

var fullPathRe = regexp.MustCompile(`full-path="([^"]+)"`)

// opfPath reads the package file's path from META-INF/container.xml, "" when there is none
func opfPath(zr *zip.Reader) (string, error) {
	for _, f := range zr.File {
		if f.Name != "META-INF/container.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		if m := fullPathRe.FindSubmatch(content); m != nil {
			return string(m[1]), nil
		}
		return "", nil
	}
	return "", nil
}

/*
Function:	CoverHref
Purpose:	the file name of an EPUB's cover image, "" when it doesn't mark one
Params:
  - filePath: the EPUB
*/
func CoverHref(filePath string) (string, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	// step 1: find OPF path from container.xml
	opf, err := opfPath(&zr.Reader)
	if err != nil || opf == "" {
		return "", err
	}

	// step 2: read the OPF
	for _, f := range zr.File {
		if f.Name != opf {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}

		opf := string(content)

		// try properties="cover-image" first (EPUB3 standard)
		coverImageRe := regexp.MustCompile(`<item[^>]+properties="cover-image"[^>]+href="([^"]+)"`)
		if m := coverImageRe.FindStringSubmatch(opf); m != nil {
			return filepath.Base(m[1]), nil
		}
		// also try href before properties
		coverImageRe2 := regexp.MustCompile(`<item[^>]+href="([^"]+)"[^>]+properties="cover-image"`)
		if m := coverImageRe2.FindStringSubmatch(opf); m != nil {
			return filepath.Base(m[1]), nil
		}

		// try name first then content
		metaRe2 := regexp.MustCompile(`(?i)<meta\s+name="cover"\s+content="([^"]+)"`)
		metaRe3 := regexp.MustCompile(`(?i)<meta\s+content="([^"]+)"\s+name="cover"`)

		var coverID string
		if m := metaRe2.FindStringSubmatch(opf); m != nil {
			coverID = m[1]
		} else if m := metaRe3.FindStringSubmatch(opf); m != nil {
			coverID = m[1]
		}

		if coverID != "" {
			itemRe := regexp.MustCompile(`<item\s+[^>]*id="` + regexp.QuoteMeta(coverID) + `"[^>]+href="([^"]+)"`)
			itemRe2 := regexp.MustCompile(`<item\s+[^>]*href="([^"]+)"[^>]+id="` + regexp.QuoteMeta(coverID) + `"`)
			if m := itemRe.FindStringSubmatch(opf); m != nil {
				return filepath.Base(m[1]), nil
			}
			if m := itemRe2.FindStringSubmatch(opf); m != nil {
				return filepath.Base(m[1]), nil
			}
		}

		return "", nil
	}

	return "", nil
}
