package downloader

import "testing"

func TestWishlistDownloads(t *testing.T) {
	for typ, want := range map[string]bool{
		"anime": true, "movie": true, "manga": true,
		// no downloader of their own (light novels) or requested from Discover (music)
		"light_novel": false, "music_track": false, "music_album": false, "": false,
	} {
		if got := WishlistDownloads(typ); got != want {
			t.Errorf("WishlistDownloads(%q) = %v, want %v", typ, got, want)
		}
	}
}

func TestExistingOutcome(t *testing.T) {
	cases := []struct {
		name  string
		facts wishlistFacts
		want  WishlistOutcome
		done  bool
	}{
		{"new manga", wishlistFacts{mediaType: "manga"}, WishlistOutcome{}, false},
		{"music isn't downloaded", wishlistFacts{mediaType: "music_track"}, WishlistOutcome{Download: WishlistNotDownloadable}, true},
		{"already in the library", wishlistFacts{mediaType: "anime", have: true, requested: true}, WishlistOutcome{Download: WishlistHave}, true},
		{"requested, still going", wishlistFacts{mediaType: "movie", requested: true, lastJobStatus: "downloading"}, WishlistOutcome{Download: WishlistAlreadyRequested}, true},
		{"requested, not dispatched yet", wishlistFacts{mediaType: "movie", requested: true}, WishlistOutcome{Download: WishlistAlreadyRequested}, true},
		{"requested, last download failed", wishlistFacts{mediaType: "manga", requested: true, lastJobStatus: "failed", lastJobError: "MangaDex hosts no English chapters"},
			WishlistOutcome{Download: WishlistFailed, Reason: "MangaDex hosts no English chapters"}, true},
		// a failed job with a Suwayomi link added since: the nightly check gets it
		{"failed, then linked", wishlistFacts{mediaType: "manga", have: true, requested: true, lastJobStatus: "failed"}, WishlistOutcome{Download: WishlistHave}, true},
	}
	for _, c := range cases {
		got, done := existingOutcome(c.facts)
		if got != c.want || done != c.done {
			t.Errorf("%s: existingOutcome = %+v, %v, want %+v, %v", c.name, got, done, c.want, c.done)
		}
	}
}
