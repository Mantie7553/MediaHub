package downloader

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
)

func TestYtdlpArgsEndWithTheSourceAfterDoubleDash(t *testing.T) {
	args := ytdlpArgs("/media/Music/A/B/%(title)s.%(ext)s", "/tmp/path.txt", "/tmp/cookies.txt", "--exec=touch /tmp/pwned")
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

func TestYtdlpArgsSignInOnlyWithACookiesFile(t *testing.T) {
	for _, cookies := range []string{"", "/tmp/ytdlp-cookies-1.txt"} {
		args := ytdlpArgs("/media/Music/A/B/%(title)s.%(ext)s", "/tmp/path.txt", cookies, "https://youtu.be/x")
		n := len(args)
		if args[n-2] != "--" || args[n-1] != "https://youtu.be/x" {
			t.Errorf("cookies %q: args end %q, want the source last after --", cookies, args[n-2:])
		}
		got := ""
		for i, a := range args {
			if a == "--cookies" && i+1 < n {
				got = args[i+1]
			}
		}
		if got != cookies {
			t.Errorf("cookies %q: --cookies is %q", cookies, got)
		}
	}
}

func TestCookiesCopyLeavesTheOriginalAlone(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YTDLP_TMPDIR", dir)

	t.Setenv("YTDLP_COOKIES", "")
	if got, err := cookiesCopy(); got != "" || err != nil {
		t.Errorf("no cookies set: got %q, %v, want nothing", got, err)
	}

	t.Setenv("YTDLP_COOKIES", filepath.Join(dir, "missing.txt"))
	if got, err := cookiesCopy(); got != "" || err != nil {
		t.Errorf("missing cookies file: got %q, %v, want nothing", got, err)
	}

	src := filepath.Join(dir, "cookies.txt")
	if err := os.WriteFile(src, []byte("# Netscape HTTP Cookie File\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YTDLP_COOKIES", src)
	got, err := cookiesCopy()
	if err != nil || got == "" || got == src {
		t.Fatalf("got %q, %v, want a copy apart from %s", got, err, src)
	}
	if b, _ := os.ReadFile(got); string(b) != "# Netscape HTTP Cookie File\n" {
		t.Errorf("copy holds %q", b)
	}
}

func TestFailureMessageExplainsAgeRestrictions(t *testing.T) {
	ageErr := "ERROR: [youtube] abc: Sign in to confirm your age."
	if got := failureMessage("ERROR: something else", false); got != "ERROR: something else" {
		t.Errorf("other errors should pass through, got %q", got)
	}
	if got := failureMessage(ageErr, false); !strings.Contains(got, "cookies.txt") || !strings.HasSuffix(got, ageErr) {
		t.Errorf("no cookies: got %q", got)
	}
	if got := failureMessage(ageErr, true); !strings.Contains(got, "expired") || !strings.HasSuffix(got, ageErr) {
		t.Errorf("refused cookies: got %q", got)
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
