package ytdlp

import "testing"

func TestCanonicalURLAcceptsVideoLinks(t *testing.T) {
	want := "https://www.youtube.com/watch?v=ZRtdQ81jPUQ"
	for _, raw := range []string{
		"https://www.youtube.com/watch?v=ZRtdQ81jPUQ",
		"https://music.youtube.com/watch?v=ZRtdQ81jPUQ&si=abc",
		"https://youtube.com/watch?v=ZRtdQ81jPUQ&list=PL123&index=4",
		"https://m.youtube.com/watch?v=ZRtdQ81jPUQ",
		"https://youtu.be/ZRtdQ81jPUQ?t=30",
		"https://www.youtube.com/shorts/ZRtdQ81jPUQ",
		"  https://WWW.YouTube.com/watch?v=ZRtdQ81jPUQ  ",
	} {
		got, err := CanonicalURL(raw)
		if err != nil || got != want {
			t.Errorf("CanonicalURL(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
}

func TestCanonicalURLRefusesEverythingElse(t *testing.T) {
	for _, raw := range []string{
		"",
		// yt-dlp would read these as options; --exec runs a shell command
		"--exec=touch /tmp/pwned",
		"--exec touch /tmp/pwned",
		"-o /etc/cron.d/x",
		// other sites, schemes and places
		"http://www.youtube.com/watch?v=ZRtdQ81jPUQ",
		"https://evil.example/watch?v=ZRtdQ81jPUQ",
		"https://www.youtube.com.evil.example/watch?v=ZRtdQ81jPUQ",
		"https://user@www.youtube.com/watch?v=ZRtdQ81jPUQ",
		"https://www.youtube.com:8443/watch?v=ZRtdQ81jPUQ",
		"file:///etc/passwd",
		"ytsearch:anything",
		"http://sonarr:8989/api/v3/system/status",
		// not a single video
		"https://www.youtube.com/playlist?list=PL123",
		"https://www.youtube.com/watch?v=short",
		"https://www.youtube.com/watch?v=ZRtdQ81jPUQ--exec",
		"https://youtu.be/ZRtdQ81jPUQ/extra",
		"https://www.youtube.com/@channel",
	} {
		if got, err := CanonicalURL(raw); err == nil {
			t.Errorf("CanonicalURL(%q) = %q, want it refused", raw, got)
		}
	}
}
