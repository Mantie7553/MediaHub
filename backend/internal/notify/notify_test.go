package notify

import (
	"strings"
	"testing"
)

func TestDetailsLink(t *testing.T) {
	cases := []struct {
		mediaType, id, album, want string
	}{
		{"anime", "a1", "", "/anime/a1"},
		{"movie", "m1", "", "/movies/m1"},
		{"manga", "g1", "", "/manga/g1"},
		{"light_novel", "l1", "", "/light-novels/l1"},
		{"music_track", "t1", "al1", "/albums/al1"},
		{"music_track", "t1", "", "/downloads"},
		{"other", "x", "", "/downloads"},
	}
	for _, c := range cases {
		if got := DetailsLink(c.mediaType, c.id, c.album); got != c.want {
			t.Errorf("DetailsLink(%q, %q, %q) = %q, want %q", c.mediaType, c.id, c.album, got, c.want)
		}
	}
}

func TestLinksAndKeys(t *testing.T) {
	if got := RequestLink("r1"); got != "/downloads?request=r1" {
		t.Errorf("RequestLink = %q", got)
	}
	if got := JobLink("j1"); got != "/downloads?job=j1" {
		t.Errorf("JobLink = %q", got)
	}
	// a request's approval and its by-hand note share a key, so approving resolves either
	if RequestKey("r1") == JobKey("r1") || DownloadingKey("r1") == JobKey("r1") {
		t.Error("keys of different things must differ")
	}
	if got := StuckKey("sonarr", 12); got != "stuck:sonarr:12" {
		t.Errorf("StuckKey = %q", got)
	}
	if StuckKey("sonarr", 12) == StuckKey("radarr", 12) || NoReleaseKey("r1") == RequestKey("r1") {
		t.Error("stuck keys must differ by arr, and from a request's own key")
	}
}

func TestShorten(t *testing.T) {
	if got := Shorten("\n\n  ERROR: video unavailable  \nmore output", 100); got != "ERROR: video unavailable" {
		t.Errorf("first line: got %q", got)
	}
	long := strings.Repeat("é", 300)
	got := Shorten(long, 10)
	if len([]rune(got)) != 10 || !strings.HasSuffix(got, "…") {
		t.Errorf("cut: got %q", got)
	}
	if got := Shorten("", 10); got != "" {
		t.Errorf("empty: got %q", got)
	}
}

func TestHandlerNote(t *testing.T) {
	if handlerNote("sonarr", "anime") != "Sent to Sonarr" || handlerNote("mangadex", "manga") != "Fetching new chapters" {
		t.Error("unexpected handler notes")
	}
	if handlerNote("unknown", "anime") != "" {
		t.Error("unknown handler should have no note")
	}
}
