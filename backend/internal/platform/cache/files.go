package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

const (
	// how long a NAS file's size and modified time are trusted before checking again, so
	// serving a cached copy doesn't wait on the NAS every time
	statTTL = time.Minute
	// copies made at once, so a burst of new files doesn't saturate the NAS link
	maxCopies = 2
	// default size of the cache in GB, CACHE_MAX_GB in the environment overrides it
	defaultMaxGB = 5
)

/*
Files keeps local copies of files that live on slow storage (the NAS) and of images fetched
from other sites. A NAS file is served from the NAS the first time it is read while a copy is
made in the background; later reads use the copy. A copy is named after the file's path, size
and modified time, so a changed file is copied again and its old copy is never served. The
least recently used copies are deleted once the cache is over its size
*/
type Files struct {
	dir string
	max int64
	now func() time.Time

	mu      sync.Mutex
	entries map[string]*fileEntry // by file name in dir
	total   int64
	stats   map[string]statEntry // by source path
	copying map[string]bool      // file names being copied
	sem     chan struct{}
	copies  sync.WaitGroup // background copies still running
}

type fileEntry struct {
	size int64
	used time.Time
}

type statEntry struct {
	info os.FileInfo
	err  error
	at   time.Time
}

var (
	diskOnce sync.Once
	disk     *Files
)

/*
Function:	Disk
Purpose:	the shared file cache, set up from CACHE_DIR (default: the user cache folder, so
/data/.cache/mediahub in the container) and CACHE_MAX_GB (default 5, 0 turns it off). nil when
off or it can't be set up; a nil *Files hands every path straight back
*/
func Disk() *Files {
	diskOnce.Do(func() {
		maxGB := float64(defaultMaxGB)
		if v := os.Getenv("CACHE_MAX_GB"); v != "" {
			n, err := strconv.ParseFloat(v, 64)
			if err != nil || n < 0 {
				logger.Warn("CACHE_MAX_GB %q is not a size in GB, using %d", v, defaultMaxGB)
			} else {
				maxGB = n
			}
		}
		if maxGB == 0 {
			logger.Info("file cache is off (CACHE_MAX_GB=0)")
			return
		}

		dir := os.Getenv("CACHE_DIR")
		if dir == "" {
			base, err := os.UserCacheDir()
			if err != nil {
				logger.Warn("file cache is off: no CACHE_DIR and no user cache folder: %s", err.Error())
				return
			}
			dir = filepath.Join(base, "mediahub")
		}

		f, err := NewFiles(dir, int64(maxGB*(1<<30)))
		if err != nil {
			logger.Warn("file cache is off: %s", err.Error())
			return
		}
		disk = f
		logger.Info("file cache in %s, up to %.1f GB (%.2f GB used)", dir, maxGB, float64(f.total)/(1<<30))
	})
	return disk
}

/*
Function:	NewFiles
Purpose:	a file cache in dir holding up to max bytes. Copies already in dir are kept, oldest
modified counted as least recently used; unfinished copies are deleted
*/
func NewFiles(dir string, max int64) (*Files, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f := &Files{
		dir:     dir,
		max:     max,
		entries: map[string]*fileEntry{},
		stats:   map[string]statEntry{},
		copying: map[string]bool{},
		sem:     make(chan struct{}, maxCopies),
	}

	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		if strings.HasSuffix(item.Name(), ".tmp") {
			os.Remove(filepath.Join(dir, item.Name()))
			continue
		}
		info, err := item.Info()
		if err != nil {
			continue
		}
		f.entries[item.Name()] = &fileEntry{size: info.Size(), used: info.ModTime()}
		f.total += info.Size()
	}
	f.mu.Lock()
	f.evictLocked()
	f.mu.Unlock()
	return f, nil
}

/*
Function:	Local
Purpose:	the path to read src from: its cached copy when there is one, otherwise src itself
(and a copy is started for next time). Any problem reading src is left for the caller to
find when it opens the path
Params:
  - src: a media file on the NAS
*/
func (f *Files) Local(src string) string {
	if f == nil {
		return src
	}
	info, err := f.stat(src)
	if err != nil || info.IsDir() {
		return src
	}

	name := hashName(fmt.Sprintf("%s|%d|%d", src, info.Size(), info.ModTime().UnixNano()), filepath.Ext(src))
	f.mu.Lock()
	defer f.mu.Unlock()
	if e, ok := f.entries[name]; ok {
		e.used = f.clock()
		return filepath.Join(f.dir, name)
	}
	// one file taking a big share of the cache would push out everything else
	if info.Size() > f.max/4 || f.copying[name] {
		return src
	}
	f.copying[name] = true
	f.copies.Add(1)
	go f.copy(src, name)
	return src
}

/*
Function:	Stats
Purpose:	bytes the cache holds, the most it may hold and its folder. All zero when it is off
*/
func (f *Files) Stats() (used, max int64, dir string) {
	if f == nil {
		return 0, 0, ""
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.total, f.max, f.dir
}

/*
Function:	Forget
Purpose:	check src on the NAS again on its next read, for a file that was just replaced
*/
func (f *Files) Forget(src string) {
	if f == nil {
		return
	}
	f.mu.Lock()
	delete(f.stats, src)
	f.mu.Unlock()
}

/*
Function:	Remote
Purpose:	the bytes for key from the cache, or from fetch, which are then kept
Params:
  - key: identifies the content, such as its URL
  - fetch: gets the content when it isn't cached; an error is returned and nothing is kept
*/
func (f *Files) Remote(key string, fetch func() ([]byte, error)) ([]byte, error) {
	if f == nil {
		return fetch()
	}
	name := hashName(key, "")
	path := filepath.Join(f.dir, name)

	f.mu.Lock()
	e, ok := f.entries[name]
	if ok {
		e.used = f.clock()
	}
	f.mu.Unlock()
	if ok {
		if data, err := os.ReadFile(path); err == nil {
			return data, nil
		}
	}

	data, err := fetch()
	if err != nil {
		return nil, err
	}
	if int64(len(data)) <= f.max/4 {
		if err := writeAtomic(path, data); err == nil {
			f.add(name, int64(len(data)))
		}
	}
	return data, nil
}

// src's size and modified time, from memory when checked within statTTL
func (f *Files) stat(src string) (os.FileInfo, error) {
	f.mu.Lock()
	s, ok := f.stats[src]
	f.mu.Unlock()
	if ok && f.clock().Sub(s.at) < statTTL {
		return s.info, s.err
	}

	info, err := os.Stat(src)
	f.mu.Lock()
	f.stats[src] = statEntry{info: info, err: err, at: f.clock()}
	f.mu.Unlock()
	return info, err
}

// copies src into the cache as name, at most maxCopies at once
func (f *Files) copy(src, name string) {
	defer f.copies.Done()
	f.sem <- struct{}{}
	defer func() { <-f.sem }()
	defer func() {
		f.mu.Lock()
		delete(f.copying, name)
		f.mu.Unlock()
	}()

	path := filepath.Join(f.dir, name)
	size, err := copyAtomic(src, path)
	if err != nil {
		logger.Warn("caching %s: %s", src, err.Error())
		return
	}
	f.add(name, size)
}

// counts a new copy and makes room for it
func (f *Files) add(name string, size int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.entries[name]; ok {
		f.total -= old.size
	}
	f.entries[name] = &fileEntry{size: size, used: f.clock()}
	f.total += size
	f.evictLocked()
}

// deletes least recently used copies until the cache fits; f.mu must be held
func (f *Files) evictLocked() {
	for f.total > f.max && len(f.entries) > 0 {
		var oldest string
		var oldestUsed time.Time
		for name, e := range f.entries {
			if oldest == "" || e.used.Before(oldestUsed) {
				oldest, oldestUsed = name, e.used
			}
		}
		os.Remove(filepath.Join(f.dir, oldest))
		f.total -= f.entries[oldest].size
		delete(f.entries, oldest)
	}
}

func (f *Files) clock() time.Time {
	if f.now != nil {
		return f.now()
	}
	return time.Now()
}

// a file name for key that is safe on any file system, keeping ext so types stay readable
func hashName(key, ext string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:16]) + strings.ToLower(ext)
}

// copies src to dest through a .tmp file, so a half copied file is never served
func copyAtomic(src, dest string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	tmp := dest + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp, dest)
	}
	if err != nil {
		os.Remove(tmp)
		return 0, err
	}
	return n, nil
}

func writeAtomic(dest string, data []byte) error {
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
