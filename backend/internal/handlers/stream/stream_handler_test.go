package stream

import (
	"strings"
	"testing"
)

func TestSignPlaylist(t *testing.T) {
	playlist := "#EXTM3U\n#EXT-X-VERSION:3\n#EXTINF:10.0,\nplaylist0.ts\n#EXTINF:10.0,\nplaylist1.ts?x=1\n\n"
	got := signPlaylist(playlist, "a.b+c")

	want := "#EXTM3U\n#EXT-X-VERSION:3\n#EXTINF:10.0,\nplaylist0.ts?media_token=a.b%2Bc\n#EXTINF:10.0,\nplaylist1.ts?x=1&media_token=a.b%2Bc\n\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "#EXTM3U?") {
		t.Fatal("tag lines must not be signed")
	}
}
