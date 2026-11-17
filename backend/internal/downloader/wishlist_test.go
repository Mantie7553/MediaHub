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
