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

func TestVodPlaylist(t *testing.T) {
	got := vodPlaylist(9.5)
	want := "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:4\n#EXT-X-MEDIA-SEQUENCE:0\n#EXT-X-PLAYLIST-TYPE:VOD\n" +
		"#EXTINF:4.000000,\nplaylist0.ts\n#EXTINF:4.000000,\nplaylist1.ts\n#EXTINF:1.500000,\nplaylist2.ts\n" +
		"#EXT-X-ENDLIST\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}

	if n := segmentCount(8); n != 2 {
		t.Errorf("an exact multiple of the segment length = %d segments, want 2", n)
	}
	if n := segmentCount(8.01); n != 3 {
		t.Errorf("8.01s = %d segments, want 3", n)
	}
}

func TestSegmentAction(t *testing.T) {
	running := func(start, head, limit int) runState {
		return runState{running: true, start: start, head: head, limit: limit}
	}
	tests := []struct {
		name    string
		seg     int
		runs    []runState
		want    action
		wantRun int
	}{
		{"nothing running yet", 10, nil, actionStart, -1},
		{"next segment", 10, []runState{running(0, 9, 500)}, actionWait, 0},
		{"a little ahead", 15, []runState{running(0, 9, 500)}, actionWait, 0},
		{"seek far ahead", 16, []runState{running(0, 9, 500)}, actionStart, -1},
		{"seek back before the run", 3, []runState{running(50, 60, 500)}, actionStart, -1},
		{"just started there", 50, []runState{running(50, 49, 500)}, actionWait, 0},
		// a second viewer far from the first gets their own run, the first keeps going
		{"waits on the run near it", 205, []runState{running(0, 30, 200), running(200, 203, 500)}, actionWait, 1},
		{"past a run's limit", 201, []runState{running(0, 199, 200)}, actionStart, -1},
		{"picks the run furthest along", 12, []runState{running(0, 8, 500), running(10, 11, 500)}, actionWait, 1},
		{"gap before a finished run", 3, []runState{{finished: true, start: 50, head: 400, limit: 401}}, actionStart, -1},
		{"finished run never wrote it", 401, []runState{{finished: true, start: 50, head: 400, limit: 402}}, actionMissing, -1},
	}
	for _, tt := range tests {
		got, run := segmentAction(tt.seg, tt.runs)
		if got != tt.want || run != tt.wantRun {
			t.Errorf("%s: got %v run %d, want %v run %d", tt.name, got, run, tt.want, tt.wantRun)
		}
	}
}

// a run whose ffmpeg is still going, or has exited when done is closed
func testRun(start, head, limit int, lastUsed time.Time, exited bool) *ffmpegRun {
	run := &ffmpegRun{start: start, head: head, limit: limit, lastUsed: lastUsed, done: make(chan struct{})}
	if exited {
		close(run.done)
	}
	return run
}

func TestLeastRecentlyUsed(t *testing.T) {
	now := time.Now()
	recent := testRun(0, 10, 100, now, false)
	old := testRun(100, 110, 200, now.Add(-time.Minute), false)
	exited := testRun(200, 210, 300, now.Add(-time.Hour), true)
	if got := leastRecentlyUsed([]*ffmpegRun{recent, old, exited}); got != old {
		t.Errorf("got run from %d, want the oldest running one from 100", got.start)
	}
	if got := leastRecentlyUsed([]*ffmpegRun{exited}); got != nil {
		t.Errorf("got a run that already exited")
	}
}

func TestCaughtUp(t *testing.T) {
	if caughtUp(testRun(0, 98, 100, time.Now(), false), 500) {
		t.Error("caught up before reaching its limit")
	}
	if !caughtUp(testRun(0, 99, 100, time.Now(), false), 500) {
		t.Error("not caught up at the segment before its limit")
	}
	// the last segment is the end of the video, not another run's segments
	if caughtUp(testRun(0, 499, 500, time.Now(), false), 500) {
		t.Error("a run at the end of the video counted as caught up")
	}
}

func TestRunLimit(t *testing.T) {
	dir := t.TempDir()
	for _, seg := range []int{3, 40, 41} {
		if err := os.WriteFile(filepath.Join(dir, segmentName(seg)), []byte("ts"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	runs := []*ffmpegRun{testRun(30, 32, 40, now, false), testRun(20, 25, 30, now, true)}

	if got := runLimit(dir, 10, 100, runs); got != 30 {
		t.Errorf("limit from 10 = %d, want 30 where a later run starts", got)
	}
	// an exited run's start doesn't count, segments on disk do
	if got := runLimit(dir, 15, 100, runs[1:]); got != 40 {
		t.Errorf("limit from 15 = %d, want 40 where segments are on disk", got)
	}
	if got := runLimit(dir, 50, 100, runs); got != 100 {
		t.Errorf("limit from 50 = %d, want the segment count", got)
	}
}

func TestRefreshAndTouch(t *testing.T) {
	dir := t.TempDir()
	for _, seg := range []int{10, 11, 12} {
		if err := os.WriteFile(filepath.Join(dir, segmentName(seg)), []byte("ts"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	tc := &transcode{dir: dir, manifest: streamManifest{Duration: 400}}
	run := testRun(10, 9, 50, now.Add(-time.Minute), false)
	tc.runs = []*ffmpegRun{run}

	states := tc.refresh(now)
	if len(states) != 1 || !states[0].running || states[0].head != 12 {
		t.Fatalf("states = %+v, want one running run with head 12", states)
	}

	// a viewer served segments already on disk keeps their run from going idle
	tc.touch(11)
	if !run.lastUsed.After(now.Add(-time.Second)) {
		t.Errorf("touching a segment of the run didn't mark it used")
	}
	before := run.lastUsed
	tc.touch(60)
	if run.lastUsed != before {
		t.Errorf("touching a segment past the run's limit marked it used")
	}

	// exited runs that didn't finish by themselves are dropped
	tc.runs = append(tc.runs, testRun(100, 99, 200, now, true))
	tc.runs[1].stopped = true
	if states := tc.refresh(now); len(states) != 1 {
		t.Errorf("kept a stopped run: %+v", states)
	}
}

func TestWaitForSegment(t *testing.T) {
	dir := t.TempDir()
	tc := &transcode{dir: dir, manifest: streamManifest{Duration: 20}}
	if err := os.WriteFile(filepath.Join(dir, segmentName(2)), []byte("ts"), 0644); err != nil {
		t.Fatal(err)
	}

	// already written: no ffmpeg needed
	if err := tc.waitForSegment(2, time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(tc.runs) != 0 {
		t.Fatal("started ffmpeg for a segment on disk")
	}
	for _, seg := range []int{-1, 5} {
		if err := tc.waitForSegment(seg, time.Now().Add(time.Second)); !errors.Is(err, errSegmentMissing) {
			t.Errorf("segment %d of 5: err = %v, want errSegmentMissing", seg, err)
		}
	}

	tc.err = errors.New("exit status 1")
	if err := tc.waitForSegment(3, time.Now().Add(time.Second)); err != tc.err {
		t.Errorf("failed transcode: err = %v", err)
	}
}

func TestTranscodeArgs(t *testing.T) {
	t.Setenv("TRANSCODE_ENCODER", "amf")
	args := strings.Join(transcodeArgs("out", streamManifest{Source: "in.mkv", BurnIndex: -1, AudioIndex: 2}, 0), " ")
	for _, want := range []string{
		"-map 0:v:0", "-map 0:2", "-b:v 6M", "-maxrate 8M", "-forced_idr 1",
		"-hls_time 4", "-start_number 0", "-hls_flags temp_file", "-ac 2",
		// the input may only be read as a local file
		"-protocol_whitelist file -i in.mkv",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q: %s", want, args)
		}
	}
	if strings.Contains(args, "-ss") || strings.Contains(args, "-output_ts_offset") {
		t.Errorf("a run from the start seeks: %s", args)
	}
	// ffmpeg's playlist must not replace the one listing every segment
	if strings.HasSuffix(args, playlistFile) {
		t.Errorf("ffmpeg writes the served playlist: %s", args)
	}

	resumed := strings.Join(transcodeArgs("out", streamManifest{Source: "in.mkv", BurnIndex: -1, AudioIndex: 2}, 150), " ")
	// each run has its own ffmpeg playlist, so runs at the same time don't share one
	if !strings.HasSuffix(args, "ffmpeg-0.m3u8") || !strings.HasSuffix(resumed, "ffmpeg-150.m3u8") {
		t.Errorf("runs don't write their own ffmpeg playlist: %s / %s", args, resumed)
	}
	for _, want := range []string{"-ss 600 -protocol_whitelist file -i in.mkv", "-output_ts_offset 600", "-start_number 150"} {
		if !strings.Contains(resumed, want) {
			t.Errorf("resumed args missing %q: %s", want, resumed)
		}
	}

	noAudio := strings.Join(transcodeArgs("out", streamManifest{Source: "in.mkv", BurnIndex: -1, AudioIndex: -1}, 0), " ")
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
	args := strings.Join(transcodeArgs("out", streamManifest{Source: "in.mkv", BurnIndex: -1, AudioIndex: 1}, 0), " ")
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
	args = strings.Join(transcodeArgs("out", streamManifest{Source: "in.mkv", BurnIndex: -1, AudioIndex: 1}, 0), " ")
	for _, want := range []string{"format=yuv420p", "-c:v h264_nvenc", "-forced-idr 1", "-maxrate 8M", "-force_key_frames"} {
		if !strings.Contains(args, want) {
			t.Errorf("nvenc args missing %q: %s", want, args)
		}
	}
	if strings.Contains(args, "-vaapi_device") {
		t.Errorf("nvenc args contain -vaapi_device: %s", args)
	}

	t.Setenv("TRANSCODE_ENCODER", "libx264")
	args = strings.Join(transcodeArgs("out", streamManifest{Source: "in.mkv", BurnIndex: -1, AudioIndex: 1}, 0), " ")
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

func TestTranscodeArgsBurnIn(t *testing.T) {
	manifest := streamManifest{Source: "in.mkv", AudioIndex: 1, BurnIndex: 4}
	for encoder, chain := range map[string]string{
		"libx264": "overlay=eof_action=pass,format=yuv420p[out]",
		"nvenc":   "overlay=eof_action=pass,format=yuv420p[out]",
		// the overlay has to happen before the frames go to the GPU
		"vaapi": "overlay=eof_action=pass,format=nv12,hwupload[out]",
	} {
		t.Setenv("TRANSCODE_ENCODER", encoder)
		args := strings.Join(transcodeArgs("out", manifest, 0), " ")
		for _, want := range []string{"-filter_complex [0:4][0:v:0]scale2ref[sub][video];[video][sub]" + chain, "-map [out]", "-map 0:1"} {
			if !strings.Contains(args, want) {
				t.Errorf("%s args missing %q: %s", encoder, want, args)
			}
		}
		if strings.Contains(args, "-vf") || strings.Contains(args, "-map 0:v:0") {
			t.Errorf("%s args use -vf with burn in: %s", encoder, args)
		}
	}
}

func TestVariantRe(t *testing.T) {
	for v, want := range map[string]bool{
		"a1": true, "a-1": true, "a2-b5": true, "a-1-b3": true,
		"a2-b": false, "a2-b-1": false, "b5": false, "a2-b5-b6": false, "a2/b5": false,
	} {
		if got := variantRe.MatchString(v); got != want {
			t.Errorf("variantRe(%q) = %v, want %v", v, got, want)
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

	args := strings.Join(transcodeArgs("out", streamManifest{Source: "in.mkv", BurnIndex: -1, VideoCodec: "av1", AudioIndex: 1}, 0), " ")
	if strings.Contains(args, "-hwaccel") {
		t.Errorf("hwaccel added while disabled: %s", args)
	}
}

func TestGetTranscodeReusesSegments(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "episode-1-a1")
	manifest := streamManifest{Source: "in.mkv", Duration: 30, AudioIndex: 1}

	h := NewHandler(nil)
	if _, err := h.getTranscode("episode-1-a1", dir, manifest); err != nil {
		t.Fatal(err)
	}
	if !fileExists(filepath.Join(dir, playlistFile)) || !fileExists(filepath.Join(dir, manifestFile)) {
		t.Fatal("playlist and manifest not written")
	}
	if err := os.WriteFile(filepath.Join(dir, segmentName(0)), []byte("ts"), 0644); err != nil {
		t.Fatal(err)
	}

	// a restarted server finds the folder and keeps its segments
	if _, err := NewHandler(nil).getTranscode("episode-1-a1", dir, manifest); err != nil {
		t.Fatal(err)
	}
	if !fileExists(filepath.Join(dir, segmentName(0))) {
		t.Fatal("segments from the same file were thrown away")
	}

	// a replaced file starts over
	manifest.Duration = 31
	if _, err := NewHandler(nil).getTranscode("episode-1-a1", dir, manifest); err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(dir, segmentName(0))) {
		t.Fatal("segments from a different file were kept")
	}

	// folders from before playlists were written up front have no source or duration
	if sameSource(streamManifest{AudioIndex: 1}, manifest) {
		t.Fatal("old folder treated as the same source")
	}
	burned := manifest
	burned.BurnIndex = 4
	if sameSource(burned, manifest) {
		t.Fatal("a burned in subtitle treated as the same transcode")
	}
}
