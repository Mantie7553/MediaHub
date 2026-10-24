package insights

import (
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

// how long a folder scan is reused; walking the NAS takes a while on a big library
const scanTTL = time.Hour

// the folders under MEDIA_ROOT each media type lives in
var libraryFolders = []struct{ name, mediaType string }{
	{"TV Shows", "anime"},
	{"Movies", "movie"},
	{"Music", "music"},
	{"Manga", "manga"},
	{"Light Novels", "light_novel"},
}

/*
storageScanner adds up how much each library folder holds, keeping the answer for scanTTL.
One scan runs at a time; requests that arrive during it wait and get its result
*/
type storageScanner struct {
	root string
	ttl  time.Duration
	now  func() time.Time
	// called with each new scan, to keep a daily snapshot
	onScan func(storageUsage)

	mu   sync.Mutex
	last *storageUsage
}

func newStorageScanner(root string, onScan func(storageUsage)) *storageScanner {
	return &storageScanner{root: root, ttl: scanTTL, now: time.Now, onScan: onScan}
}

/*
Function:	Get
Purpose:	the last scan when it is recent, otherwise a new one
Params:
  - refresh: scan again even if the last scan is recent
*/
func (s *storageScanner) Get(refresh bool) storageUsage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last != nil && !refresh && s.now().Sub(s.last.ScannedAt) < s.ttl {
		return *s.last
	}

	usage := storageUsage{Folders: []folderUsage{}}
	for _, f := range libraryFolders {
		u := folderSize(filepath.Join(s.root, f.name))
		u.Name, u.Type = f.name, f.mediaType
		usage.Folders = append(usage.Folders, u)
	}
	usage.Temp = folderSize(filepath.Join(s.root, "temp"))
	usage.Temp.Name, usage.Temp.Type = "temp", "temp"
	if total, free, ok := diskSpace(s.root); ok {
		usage.DiskTotal, usage.DiskFree = &total, &free
	}
	usage.ScannedAt = s.now()
	s.last = &usage
	if s.onScan != nil {
		s.onScan(usage)
	}
	return usage
}

/*
Function:	folderSize
Purpose:	the bytes and number of files under dir. A missing folder, or files that can't be
read, count as nothing rather than failing the scan
*/
func folderSize(dir string) folderUsage {
	var u folderUsage
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		u.Bytes += info.Size()
		u.Files++
		return nil
	})
	return u
}
