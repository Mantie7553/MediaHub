package ytmusic

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func load(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseSongs(t *testing.T) {
	tracks, err := parseSearch(load(t, "songs.json"), KindSong, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 3 {
		t.Fatalf("got %d tracks, want 3", len(tracks))
	}

	single := tracks[0]
	if single.Title != "Not Like Us" || single.Artist != "Kendrick Lamar" || single.Album != "Not Like Us" {
		t.Errorf("single = %+v", single)
	}
	if !single.Explicit || !single.Official || single.Kind != KindSong {
		t.Errorf("single flags = explicit %v official %v kind %q", single.Explicit, single.Official, single.Kind)
	}
	if single.DurationSecs != 275 {
		t.Errorf("duration = %d, want 275", single.DurationSecs)
	}
	if single.URL != "https://www.youtube.com/watch?v=phLb_SoPBlA" {
		t.Errorf("url = %q", single.URL)
	}
	if single.Thumbnail == "" || strings.Contains(single.Thumbnail, "=w60-h60") {
		t.Errorf("thumbnail not upsized: %q", single.Thumbnail)
	}

	duet := tracks[1]
	if want := []string{"Kendrick Lamar", "Taylour Paige"}; !reflect.DeepEqual(duet.Artists, want) {
		t.Errorf("artists = %v, want %v", duet.Artists, want)
	}
	if duet.Artist != "Kendrick Lamar & Taylour Paige" || duet.Uploader != "Kendrick Lamar" {
		t.Errorf("artist %q uploader %q", duet.Artist, duet.Uploader)
	}
	if duet.Album != "Mr. Morale & The Big Steppers" {
		t.Errorf("album = %q", duet.Album)
	}

	clean := tracks[2]
	if clean.Explicit {
		t.Errorf("%q should not be explicit", clean.Title)
	}
}

func TestParseVideos(t *testing.T) {
	tracks, err := parseSearch(load(t, "videos.json"), KindVideo, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 3 {
		t.Fatalf("got %d tracks, want 3", len(tracks))
	}

	official, fan, collab := tracks[0], tracks[1], tracks[2]
	if official.Uploader != "Kendrick Lamar" || !official.Official || official.Album != "" {
		t.Errorf("official video = %+v", official)
	}
	if official.DurationSecs != 355 {
		t.Errorf("duration = %d, want 355", official.DurationSecs)
	}
	if fan.Uploader != "Final Move" || fan.Official {
		t.Errorf("fan upload = %+v", fan)
	}
	if want := []string{"Future", "Metro Boomin", "Kendrick Lamar"}; !reflect.DeepEqual(collab.Artists, want) {
		t.Errorf("artists = %v, want %v", collab.Artists, want)
	}
}

func TestParseLimit(t *testing.T) {
	tracks, err := parseSearch(load(t, "songs.json"), KindSong, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 {
		t.Errorf("got %d tracks, want 2", len(tracks))
	}
}

func TestParseSkipsBrokenItems(t *testing.T) {
	raw := `{"contents":[
		{"musicResponsiveListItemRenderer":{"playlistItemData":{"videoId":"aaaaaaaaaaa"},"flexColumns":[]}},
		{"musicResponsiveListItemRenderer":{"flexColumns":[{"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"No id"}]}}},{"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"3:00"}]}}}]}},
		{"musicResponsiveListItemRenderer":{"playlistItemData":{"videoId":"bbbbbbbbbbb"},"flexColumns":[{"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Clip"}]}}},{"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Someone"},{"text":" • "},{"text":"0:12"}]}}}]}},
		{"musicResponsiveListItemRenderer":{"playlistItemData":{"videoId":"ccccccccccc"},"flexColumns":[{"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Song"}]}}},{"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Song"},{"text":" • "},{"text":"Band"},{"text":" • "},{"text":"LP"},{"text":" • "},{"text":"1:02:03"}]}}}]}}
	]}`
	tracks, err := parseSearch([]byte(raw), KindSong, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks, want 1: %+v", len(tracks), tracks)
	}
	got := tracks[0]
	if got.Artist != "Band" || got.Album != "LP" || got.DurationSecs != 3723 {
		t.Errorf("got %+v", got)
	}
	if !strings.Contains(got.Thumbnail, "ccccccccccc") {
		t.Errorf("thumbnail fallback = %q", got.Thumbnail)
	}
}

func TestSearchRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/search" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Write(load(t, "videos.json"))
	}))
	defer srv.Close()

	c := NewYTMusicClient()
	c.config.BaseURL = srv.URL
	tracks, err := c.Search("not like us", FilterVideos, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 3 || tracks[0].Kind != KindVideo {
		t.Errorf("got %+v", tracks)
	}
}

func TestSearchStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := NewYTMusicClient()
	c.config.BaseURL = srv.URL
	if _, err := c.Search("x", FilterSongs, 5); err == nil {
		t.Error("expected an error for a 403")
	}
}
