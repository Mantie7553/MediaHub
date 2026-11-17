package ytmusic

import (
	"os"
	"testing"
)

// TestLiveSearch calls the real YouTube Music API, to check it still answers the way the
// parser expects. Run it with YTMUSIC_LIVE=1
func TestLiveSearch(t *testing.T) {
	if os.Getenv("YTMUSIC_LIVE") == "" {
		t.Skip("set YTMUSIC_LIVE=1 to search YouTube Music")
	}
	c := NewYTMusicClient()
	for _, filter := range []string{FilterSongs, FilterVideos} {
		query := os.Getenv("YTMUSIC_QUERY")
		if query == "" {
			query = "kendrick lamar not like us"
		}
		tracks, err := c.Search(query, filter, 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(tracks) == 0 {
			t.Fatal("no results")
		}
		for _, tr := range tracks {
			t.Logf("%s | %q by %q (%v) album %q %ds explicit=%v official=%v", tr.Kind, tr.Title, tr.Artist, tr.Artists, tr.Album, tr.DurationSecs, tr.Explicit, tr.Official)
			if tr.Artist == "" || tr.DurationSecs == 0 {
				t.Errorf("missing artist or duration: %+v", tr)
			}
		}
	}
}
