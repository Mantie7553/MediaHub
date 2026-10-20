package search

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
)

const (
	// YouTube results for a query barely change within the hour
	musicSearchTTL = time.Hour
	// searches remembered at once, oldest dropped first
	musicSearchMax = 200
)

// recent YouTube searches, so searching the same thing again (or going back to the results)
// doesn't wait on yt-dlp again
type musicSearchCache struct {
	mu      sync.Mutex
	entries map[string]musicSearchEntry
	now     func() time.Time
}

type musicSearchEntry struct {
	results []ytdlp.SearchResult
	at      time.Time
}

func musicSearchKey(query string, limit int) string {
	return fmt.Sprintf("%d:%s", limit, strings.ToLower(strings.Join(strings.Fields(query), " ")))
}

/*
Function:	get
Purpose:	the cached results for a search, if it ran within the TTL
Params:
  - query: the search text, case and extra spaces ignored
  - limit: the number of results asked for
*/
func (c *musicSearchCache) get(query string, limit int) ([]ytdlp.SearchResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[musicSearchKey(query, limit)]
	if !ok || c.clock().Sub(e.at) > musicSearchTTL {
		return nil, false
	}
	return e.results, true
}

/*
Function:	put
Purpose:	remember a search's results, dropping the oldest search when full
Params:
  - query: the search text
  - limit: the number of results asked for
  - results: what yt-dlp found
*/
func (c *musicSearchCache) put(query string, limit int, results []ytdlp.SearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]musicSearchEntry{}
	}
	key := musicSearchKey(query, limit)
	if _, ok := c.entries[key]; !ok && len(c.entries) >= musicSearchMax {
		var oldest string
		for k, e := range c.entries {
			if oldest == "" || e.at.Before(c.entries[oldest].at) {
				oldest = k
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[key] = musicSearchEntry{results: results, at: c.clock()}
}

func (c *musicSearchCache) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}
