package stream

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCleanVTT(t *testing.T) {
	in := "WEBVTT\r\n\r\n" +
		// repeated styling layer
		"00:52.652 --> 00:54.032\r\n<b>Spring</b>\r\n\r\n00:52.652 --> 00:54.032\r\n<b>Spring</b>\r\n\r\n" +
		// same text at a different time is kept
		"00:54.742 --> 00:57.032\r\nHello\r\n\r\n00:58.000 --> 00:59.000\r\nHello\r\n\r\n" +
		// leftover ASS override tag (Blue Box)
		"00:21.710 --> 00:23.500\r\n<b>{=74}<i>There's someone...</i></b>\r\n\r\n" +
		// vector drawings (Chainsaw Man typesetting)
		"01:00.000 --> 01:02.000\r\nm 735.73 28.24 b 735.73 29.09 735.84 29.94 l 740.09 4.26\r\n\r\n" +
		"01:00.500 --> 01:02.000\r\n<b>m 291.73 37.08 b 291.73 36.95 291.89 36.79</b>\r\n\r\n" +
		// tags only, nothing visible
		"01:03.000 --> 01:04.000\r\n<b></b>\r\n\r\n" +
		// real dialogue that starts with a drawing letter must survive
		"01:05.000 --> 01:06.000\r\nm-maybe later\r\n\r\n" +
		"01:07.000 --> 01:08.000\r\nb 2\r\n"
	want := "WEBVTT\n\n" +
		"00:52.652 --> 00:54.032\n<b>Spring</b>\n\n" +
		"00:54.742 --> 00:57.032\nHello\n\n00:58.000 --> 00:59.000\nHello\n\n" +
		"00:21.710 --> 00:23.500\n<b><i>There's someone...</i></b>\n\n" +
		"01:05.000 --> 01:06.000\nm-maybe later\n\n" +
		"01:07.000 --> 01:08.000\nb 2\n"
	if got := cleanVTT(in); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func writePlaylist(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, playlistFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestPlaylistDuration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, playlistFile)

	if d, ended := playlistDuration(path); d != 0 || ended {
		t.Fatalf("missing playlist = %v, %v", d, ended)
	}

	writePlaylist(t, dir, "#EXTM3U\n#EXT-X-PLAYLIST-TYPE:EVENT\n#EXTINF:4.004000,\nplaylist0.ts\n#EXTINF:4.004000,\nplaylist1.ts\n")
	if d, ended := playlistDuration(path); d < 8.0 || d > 8.01 || ended {
		t.Fatalf("growing playlist = %v, %v", d, ended)
	}

	writePlaylist(t, dir, "#EXTM3U\n#EXTINF:4.004000,\nplaylist0.ts\n#EXT-X-ENDLIST\n")
	if _, ended := playlistDuration(path); !ended {
		t.Fatal("ENDLIST not detected")
	}
}

func TestWaitForPlaylist(t *testing.T) {
	segments := func(n int) string {
		var b strings.Builder
		b.WriteString("#EXTM3U\n#EXT-X-PLAYLIST-TYPE:EVENT\n")
		for i := 0; i < n; i++ {
			b.WriteString("#EXTINF:4.000000,\nplaylist.ts\n")
		}
		return b.String()
	}

	t.Run("ready once enough is past the start", func(t *testing.T) {
		dir := t.TempDir()
		tc := &transcode{dir: dir, done: make(chan struct{})}
		writePlaylist(t, dir, segments(1))
		go func() {
			time.Sleep(300 * time.Millisecond)
			writePlaylist(t, dir, segments(3))
		}()
		if err := waitForPlaylist(tc, 0); err != nil {
			t.Fatal(err)
		}
		if d, _ := playlistDuration(filepath.Join(dir, playlistFile)); d < readyAheadSecs {
			t.Fatalf("returned with only %vs ready", d)
		}
	})

	t.Run("resume point needs more", func(t *testing.T) {
		dir := t.TempDir()
		tc := &transcode{dir: dir, done: make(chan struct{})}
		writePlaylist(t, dir, segments(5)) // 20s
		go func() {
			time.Sleep(300 * time.Millisecond)
			writePlaylist(t, dir, segments(10)) // 40s
		}()
		if err := waitForPlaylist(tc, 25); err != nil {
			t.Fatal(err)
		}
		if d, _ := playlistDuration(filepath.Join(dir, playlistFile)); d < 25+readyAheadSecs {
			t.Fatalf("returned with only %vs ready for start 25", d)
		}
	})

	t.Run("finished playlist shorter than start", func(t *testing.T) {
		dir := t.TempDir()
		tc := &transcode{dir: dir, done: make(chan struct{})}
		writePlaylist(t, dir, segments(2)+"#EXT-X-ENDLIST\n")
		if err := waitForPlaylist(tc, 600); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ffmpeg failed", func(t *testing.T) {
		dir := t.TempDir()
		tc := &transcode{dir: dir, done: make(chan struct{}), err: errors.New("exit status 1")}
		close(tc.done)
		if err := waitForPlaylist(tc, 0); err == nil {
			t.Fatal("expected the ffmpeg error")
		}
	})
}

func TestTranscodeArgs(t *testing.T) {
	t.Setenv("TRANSCODE_ENCODER", "amf")
	args := strings.Join(transcodeArgs("in.mkv", "out", streamManifest{AudioIndex: 2}), " ")
	for _, want := range []string{
		"-map 0:v:0", "-map 0:2", "-b:v 6M", "-maxrate 8M", "-forced_idr 1",
		"-hls_time 4", "-hls_playlist_type event", "-ac 2",
		// the input may only be read as a local file
		"-protocol_whitelist file -i in.mkv",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q: %s", want, args)
		}
	}

	noAudio := strings.Join(transcodeArgs("in.mkv", "out", streamManifest{AudioIndex: -1}), " ")
	if strings.Contains(noAudio, "-map 0:-1") {
		t.Errorf("mapped a missing audio track: %s", noAudio)
	}
}

func TestTranscodeEncoder(t *testing.T) {
	t.Setenv("TRANSCODE_ENCODER", "")
	def := "libx264"
	if runtime.GOOS == "windows" {
		def = "amf"
	}
	if got := transcodeEncoder(); got != def {
		t.Errorf("default = %q, want %q", got, def)
	}

	t.Setenv("TRANSCODE_ENCODER", "bogus")
	if got := transcodeEncoder(); got != def {
		t.Errorf("unknown encoder = %q, want the default %q", got, def)
	}

	t.Setenv("TRANSCODE_ENCODER", "vaapi")
	t.Setenv("TRANSCODE_VAAPI_DEVICE", "")
	args := strings.Join(transcodeArgs("in.mkv", "out", streamManifest{AudioIndex: 1}), " ")
	for _, want := range []string{"-vaapi_device /dev/dri/renderD128", "format=nv12,hwupload", "-c:v h264_vaapi", "-maxrate 8M"} {
		if !strings.Contains(args, want) {
			t.Errorf("vaapi args missing %q: %s", want, args)
		}
	}
	// the device has to be set up before the input
	if strings.Index(args, "-vaapi_device") > strings.Index(args, "-i in.mkv") {
		t.Errorf("-vaapi_device after the input: %s", args)
	}

	t.Setenv("TRANSCODE_ENCODER", "nvenc")
	args = strings.Join(transcodeArgs("in.mkv", "out", streamManifest{AudioIndex: 1}), " ")
	for _, want := range []string{"format=yuv420p", "-c:v h264_nvenc", "-forced-idr 1", "-maxrate 8M", "-force_key_frames"} {
		if !strings.Contains(args, want) {
			t.Errorf("nvenc args missing %q: %s", want, args)
		}
	}
	if strings.Contains(args, "-vaapi_device") {
		t.Errorf("nvenc args contain -vaapi_device: %s", args)
	}

	t.Setenv("TRANSCODE_ENCODER", "libx264")
	args = strings.Join(transcodeArgs("in.mkv", "out", streamManifest{AudioIndex: 1}), " ")
	for _, want := range []string{"-c:v libx264", "-preset veryfast", "-maxrate 8M", "-force_key_frames"} {
		if !strings.Contains(args, want) {
			t.Errorf("libx264 args missing %q: %s", want, args)
		}
	}
	for _, bad := range []string{"-vaapi_device", "-forced_idr", "h264_amf"} {
		if strings.Contains(args, bad) {
			t.Errorf("libx264 args contain %q: %s", bad, args)
		}
	}
}

func TestHwaccelFor(t *testing.T) {
	t.Setenv("TRANSCODE_HWACCEL", "")
	if got := hwaccelFor("hevc"); got != "" {
		t.Errorf("hevc should decode on the CPU, got %q", got)
	}

	t.Setenv("TRANSCODE_HWACCEL", "vaapi")
	if got := hwaccelFor("av1"); got != "vaapi" {
		t.Errorf("override not used: %q", got)
	}
	if got := hwaccelFor("h264"); got != "" {
		t.Errorf("override applied to h264: %q", got)
	}

	t.Setenv("TRANSCODE_HWACCEL", "none")
	if got := hwaccelFor("av1"); got != "" {
		t.Errorf("none should disable it, got %q", got)
	}

	args := strings.Join(transcodeArgs("in.mkv", "out", streamManifest{VideoCodec: "av1", AudioIndex: 1}), " ")
	if strings.Contains(args, "-hwaccel") {
		t.Errorf("hwaccel added while disabled: %s", args)
	}
}

func TestCompletedTranscode(t *testing.T) {
	dir := t.TempDir()
	if _, ok := completedTranscode(dir); ok {
		t.Fatal("empty folder treated as complete")
	}

	// folder from the old pipeline: finished playlist but no stream.json
	writePlaylist(t, dir, "#EXTM3U\n#EXTINF:10.0,\nplaylist0.ts\n#EXT-X-ENDLIST\n")
	if _, ok := completedTranscode(dir); ok {
		t.Fatal("old format folder without stream.json treated as complete")
	}

	if err := writeJSONAtomic(filepath.Join(dir, manifestFile), streamManifest{AudioIndex: 1}); err != nil {
		t.Fatal(err)
	}
	m, ok := completedTranscode(dir)
	if !ok || m.AudioIndex != 1 {
		t.Fatalf("complete transcode not loaded: %v %+v", ok, m)
	}

	writePlaylist(t, dir, "#EXTM3U\n#EXTINF:10.0,\nplaylist0.ts\n")
	if _, ok := completedTranscode(dir); ok {
		t.Fatal("unfinished playlist treated as complete")
	}
}
