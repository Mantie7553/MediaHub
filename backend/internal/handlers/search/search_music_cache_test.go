package search

import (
	"fmt"
	"testing"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
)

func TestMusicSearchCacheIgnoresCaseAndSpacing(t *testing.T) {
	var c musicSearchCache
	c.put("YOASOBI  Idol", 15, []ytdlp.SearchResult{{ID: "a"}})

	got, ok := c.get(" yoasobi idol ", 15)
	if !ok || len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("get = %v, %v; want the cached result", got, ok)
	}
	if _, ok := c.get("yoasobi idol", 5); ok {
		t.Error("a different limit is a different search")
	}
}

func TestMusicSearchCacheExpires(t *testing.T) {
	now := time.Now()
	c := musicSearchCache{now: func() time.Time { return now }}
	c.put("idol", 15, []ytdlp.SearchResult{{ID: "a"}})

	now = now.Add(musicSearchTTL + time.Second)
	if _, ok := c.get("idol", 15); ok {
		t.Error("results older than the TTL should be searched again")
	}
}

func TestMusicSearchCacheDropsOldestWhenFull(t *testing.T) {
	now := time.Now()
	c := musicSearchCache{now: func() time.Time { return now }}
	for i := 0; i < musicSearchMax; i++ {
		c.put(fmt.Sprint("q", i), 15, nil)
		now = now.Add(time.Second)
	}
	c.put("new", 15, nil)

	if len(c.entries) != musicSearchMax {
		t.Fatalf("entries = %d, want %d", len(c.entries), musicSearchMax)
	}
	if _, ok := c.get("q0", 15); ok {
		t.Error("the oldest search should have been dropped")
	}
	if _, ok := c.get("new", 15); !ok {
		t.Error("the newest search should be kept")
	}
}
