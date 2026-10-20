package downloader

import (
	"errors"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
)

func TestYtdlpArgsEndWithTheSourceAfterDoubleDash(t *testing.T) {
	args := ytdlpArgs("/media/Music/A/B/%(title)s.%(ext)s", "/tmp/path.txt", "--exec=touch /tmp/pwned")
	n := len(args)
	if args[n-2] != "--" || args[n-1] != "--exec=touch /tmp/pwned" {
		t.Errorf("args end %q, want the source last after --, so yt-dlp can't read it as an option", args[n-2:])
	}
	for _, a := range args[:n-2] {
		if a == "--" {
			t.Error("-- should appear once, right before the source")
		}
	}
}

// Exercises validation only; a refused source never reaches the database.
func TestDispatchRefusesNonYouTubeSourcesForYtdlp(t *testing.T) {
	for _, mediaType := range []string{"music_track", "light_novel"} {
		for _, source := range []string{"--exec=touch /tmp/pwned", "", "http://sonarr:8989/api", "file:///etc/passwd"} {
			_, err := Dispatch(nil, "request-1", "media-1", source, mediaType, nil)
			if !errors.Is(err, ytdlp.ErrNotYouTubeVideo) {
				t.Errorf("Dispatch(%s, %q) error = %v, want it refused before a job is made", mediaType, source, err)
			}
		}
	}
}
