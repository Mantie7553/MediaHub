package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

const (
	manifestFile = "stream.json"
	playlistFile = "playlist.m3u8"
	logFile      = "transcode.log"

	segmentSecs = 4
	// segments from the start position that must exist before the player is pointed at it
	readySegments = 3
	readyTimeout  = 90 * time.Second
	// how long a segment request waits for ffmpeg, under the player's 40s first-byte timeout
	segmentTimeout = 30 * time.Second
	readyPoll      = 200 * time.Millisecond
	// a request this many segments past what a run has written starts a new run there instead of waiting
	restartAheadSegments = 6
	// ffmpeg runs one transcode may have at once, for viewers far apart in the same video.
	// Consumer NVIDIA cards limit how many encodes run at the same time
	maxRunsPerTranscode = 3
	// a run no viewer has asked for segments from in this long is stopped
	runIdleTimeout = 2 * time.Minute
	// how often idle and caught up runs are looked for while no segment is requested
	reapInterval = 15 * time.Second
)

var errSegmentMissing = errors.New("segment was not written")

// a transcode folder for one media and audio track, shared by every request for it. Viewers at
// different places in the video each get an ffmpeg run writing from where they are
type transcode struct {
	dir      string
	manifest streamManifest

	mu      sync.Mutex
	runs    []*ffmpegRun // running runs, and runs that wrote to the end by themselves
	reaping bool         // reapIdle is running
	err     error        // a run failed and the folder was removed, the next play starts over
}

// one ffmpeg process writing segments from start onwards
type ffmpegRun struct {
	cmd      *exec.Cmd
	start    int
	head     int // last segment written without a gap from start
	limit    int // the first segment after start that is on disk or another run writes, segmentCount if none
	lastUsed time.Time
	stopped  bool // killed, so exiting isn't a failure
	done     chan struct{}
}

/*
Function:	getTranscode
Purpose:	the transcode for a media and audio track, reusing its folder (and the segments in it)
when it was made from the same file, otherwise clearing it and starting a new one.
ffmpeg isn't started here, waitForSegment starts it where it's needed
Params:
  - key: type-id-a<audio> of the media and audio track
  - dir: the transcode folder for this media and audio track
  - manifest: what the transcode should contain
*/
func (h *Handler) getTranscode(key, dir string, manifest streamManifest) (*transcode, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if tc, ok := h.transcodes[key]; ok {
		if tc.usable() && sameSource(tc.manifest, manifest) {
			return tc, nil
		}
		// the file changed under a running transcode, which must not write into the new folder
		tc.stop()
	}

	tc := &transcode{dir: dir, manifest: manifest}
	if existing, ok := loadManifest(dir); !ok || !sameSource(existing, manifest) {
		// a failed run, a changed file, or a folder from before playlists were written up front
		if err := tc.create(); err != nil {
			return nil, err
		}
	} else {
		tc.manifest = existing
	}

	if tc.missingSubtitles() {
		go extractSubtitles(dir, tc.manifest)
	}
	h.transcodes[key] = tc
	return tc, nil
}

/*
Function:	touchSegment
Purpose:	note that a segment on disk was served, so the run writing ahead of that viewer isn't
stopped as idle. Transcodes not in use are left alone
Params:
  - key: type-id-a<audio> of the media and audio track
  - seg: the segment served
*/
func (h *Handler) touchSegment(key string, seg int) {
	h.mu.Lock()
	tc := h.transcodes[key]
	h.mu.Unlock()
	if tc != nil {
		tc.touch(seg)
	}
}

/*
Function:	loadTranscode
Purpose:	the transcode for a segment request: the one in use, or one left on disk from before a
restart of the server. False if the folder has no transcode
Params:
  - key: type-id-a<audio> of the media and audio track
  - dir: the transcode folder
*/
func (h *Handler) loadTranscode(key, dir string) (*transcode, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if tc, ok := h.transcodes[key]; ok && tc.usable() {
		return tc, true
	}
	manifest, ok := loadManifest(dir)
	if !ok || manifest.Source == "" || manifest.Duration <= 0 {
		return nil, false
	}
	tc := &transcode{dir: dir, manifest: manifest}
	h.transcodes[key] = tc
	return tc, true
}

/*
Function:	usable
Purpose:	whether a transcode can still be used: no run failed and the folder hasn't been
swept away since
*/
func (tc *transcode) usable() bool {
	tc.mu.Lock()
	failed := tc.err != nil
	tc.mu.Unlock()
	if failed {
		return false
	}
	return fileExists(filepath.Join(tc.dir, manifestFile))
}

/*
Function:	create
Purpose:	start the folder over with a playlist listing every segment and the manifest
*/
func (tc *transcode) create() error {
	if tc.manifest.Duration <= 0 {
		return errors.New("ffprobe found no duration")
	}
	if err := os.RemoveAll(tc.dir); err != nil {
		return err
	}
	if err := os.MkdirAll(tc.dir, 0755); err != nil {
		return err
	}
	err := os.WriteFile(filepath.Join(tc.dir, playlistFile), []byte(vodPlaylist(tc.manifest.Duration)), 0644)
	if err == nil {
		// written last, so a folder with stream.json is ready to use
		err = writeJSONAtomic(filepath.Join(tc.dir, manifestFile), tc.manifest)
	}
	if err != nil {
		os.RemoveAll(tc.dir)
	}
	return err
}

/*
Function:	missingSubtitles
Purpose:	whether any subtitle or font file hasn't been extracted yet, e.g. after a restart of the server
*/
func (tc *transcode) missingSubtitles() bool {
	for _, s := range tc.manifest.Subtitles {
		if !s.Image && !fileExists(filepath.Join(tc.dir, s.File)) {
			return true
		}
		if s.AssFile != "" && !fileExists(filepath.Join(tc.dir, s.AssFile)) {
			return true
		}
	}
	for _, f := range tc.manifest.Fonts {
		if !fileExists(filepath.Join(tc.dir, f.File)) {
			return true
		}
	}
	return false
}

/*
Function:	loadManifest
Purpose:	the stream.json of a transcode folder, if it has one
Params:
  - dir: the transcode folder
*/
func loadManifest(dir string) (streamManifest, bool) {
	var manifest streamManifest
	data, err := os.ReadFile(filepath.Join(dir, manifestFile))
	if err != nil || json.Unmarshal(data, &manifest) != nil {
		return manifest, false
	}
	return manifest, true
}

/*
Function:	sameSource
Purpose:	whether a transcode folder was made from the same file and tracks, so its segments can be reused
Params:
  - existing: the manifest in the folder
  - wanted: the manifest for this play
*/
func sameSource(existing, wanted streamManifest) bool {
	return existing.Source != "" && existing.Source == wanted.Source &&
		existing.Duration > 0 && existing.Duration == wanted.Duration &&
		existing.AudioIndex == wanted.AudioIndex && existing.BurnIndex == wanted.BurnIndex
}

/*
Function:	segmentCount
Purpose:	how many segments a video of this length is cut into
Params:
  - duration: length in seconds
*/
func segmentCount(duration float64) int {
	return int(math.Ceil(duration/segmentSecs - 1e-9))
}

/*
Function:	segmentName
Purpose:	file name of a segment, as ffmpeg writes it
Params:
  - n: the segment number
*/
func segmentName(n int) string {
	return fmt.Sprintf("playlist%d.ts", n)
}

var segmentNameRe = regexp.MustCompile(`^playlist(\d+)\.ts$`)

/*
Function:	vodPlaylist
Purpose:	a playlist listing every segment of the video before any exist, so the player knows the
whole length and can seek anywhere. Segments are cut every segmentSecs, the last one
holds what's left
Params:
  - duration: length in seconds
*/
func vodPlaylist(duration float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n#EXT-X-MEDIA-SEQUENCE:0\n#EXT-X-PLAYLIST-TYPE:VOD\n", segmentSecs)
	count := segmentCount(duration)
	for i := 0; i < count; i++ {
		length := math.Min(segmentSecs, duration-float64(i*segmentSecs))
		fmt.Fprintf(&b, "#EXTINF:%.6f,\n%s\n", length, segmentName(i))
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	return b.String()
}

type action int

const (
	actionWait action = iota
	actionStart
	actionMissing
)

// what waitForSegment knows about an ffmpeg run
type runState struct {
	running  bool
	finished bool // exited by itself after writing from start to the end
	start    int
	head     int
	limit    int
}

/*
Function:	segmentAction
Purpose:	what to do about a segment that doesn't exist yet: wait for the run that will write it
soon, start a new run there (no run is close before it), or give up because a finished run
passed it without writing it. Returns the index of the run to wait on
Params:
  - seg: the segment wanted
  - runs: the transcode's runs
*/
func segmentAction(seg int, runs []runState) (action, int) {
	wait := -1
	for i, run := range runs {
		covers := run.running && seg >= run.start && seg < run.limit && seg <= run.head+restartAheadSegments
		if covers && (wait < 0 || run.head > runs[wait].head) {
			wait = i
		}
	}
	if wait >= 0 {
		return actionWait, wait
	}
	for _, run := range runs {
		if run.finished && seg >= run.start {
			return actionMissing, -1
		}
	}
	return actionStart, -1
}

/*
Function:	waitForSegment
Purpose:	block until a segment exists, starting an ffmpeg run at it when no run would get there soon
Params:
  - seg: the segment wanted
  - deadline: when to give up
*/
func (tc *transcode) waitForSegment(seg int, deadline time.Time) error {
	if seg < 0 || seg >= segmentCount(tc.manifest.Duration) {
		return errSegmentMissing
	}
	path := filepath.Join(tc.dir, segmentName(seg))

	for {
		if fileExists(path) {
			return nil
		}

		tc.mu.Lock()
		if tc.err != nil {
			err := tc.err
			tc.mu.Unlock()
			return err
		}
		now := time.Now()
		var run *ffmpegRun
		switch act, i := segmentAction(seg, tc.refresh(now)); act {
		case actionMissing:
			tc.mu.Unlock()
			return errSegmentMissing
		case actionStart:
			var err error
			if run, err = tc.startRun(seg, now); err != nil {
				tc.mu.Unlock()
				return err
			}
		default:
			run = tc.runs[i]
			run.lastUsed = now
		}
		done := run.done
		tc.mu.Unlock()

		select {
		case <-done:
		case <-time.After(readyPoll):
		}
		if time.Now().After(deadline) {
			return errors.New("timed out waiting for transcode")
		}
	}
}

/*
Function:	touch
Purpose:	note that a viewer asked for a segment already on disk, so the run writing ahead of them
isn't stopped as idle
Params:
  - seg: the segment served
*/
func (tc *transcode) touch(seg int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	now := time.Now()
	for _, run := range tc.runs {
		if seg >= run.start && seg < run.limit {
			run.lastUsed = now
		}
	}
}

/*
Function:	refresh
Purpose:	move each run's head past segments written since, stop runs that reached segments
already on disk or written by another run, or that no viewer has used in a while, and
drop runs that were stopped or failed. Returns the state of the runs left. tc.mu must be held
Params:
  - now: the current time
*/
func (tc *transcode) refresh(now time.Time) []runState {
	count := segmentCount(tc.manifest.Duration)
	for _, run := range tc.runs {
		for run.head+1 < run.limit && fileExists(filepath.Join(tc.dir, segmentName(run.head+1))) {
			run.head++
		}
		if run.live() && (caughtUp(run, count) || now.Sub(run.lastUsed) > runIdleTimeout) {
			tc.killRun(run)
		}
	}
	tc.prune()

	states := make([]runState, len(tc.runs))
	for i, run := range tc.runs {
		states[i] = runState{start: run.start, head: run.head, limit: run.limit}
		if run.live() {
			states[i].running = true
		} else {
			states[i].finished = true
		}
	}
	return states
}

/*
Function:	caughtUp
Purpose:	whether a run has written up to segments that are on disk or written by another run,
so going on would only write them again
Params:
  - run: the run
  - count: segments in the video
*/
func caughtUp(run *ffmpegRun, count int) bool {
	return run.limit < count && run.head+1 >= run.limit
}

/*
Function:	prune
Purpose:	drop runs that were stopped or failed, keeping running ones and ones that wrote to the
end by themselves. tc.mu must be held
*/
func (tc *transcode) prune() {
	kept := tc.runs[:0]
	for _, run := range tc.runs {
		if run.live() || run.finished() {
			kept = append(kept, run)
		}
	}
	clear(tc.runs[len(kept):])
	tc.runs = kept
}

/*
Function:	live
Purpose:	whether ffmpeg is still running
*/
func (run *ffmpegRun) live() bool {
	select {
	case <-run.done:
		return false
	default:
		return true
	}
}

/*
Function:	finished
Purpose:	whether ffmpeg exited by itself without an error
*/
func (run *ffmpegRun) finished() bool {
	return !run.live() && !run.stopped && run.cmd.ProcessState != nil && run.cmd.ProcessState.Success()
}

/*
Function:	leastRecentlyUsed
Purpose:	the running run no viewer has used for the longest, nil if none is running
Params:
  - runs: the transcode's runs
*/
func leastRecentlyUsed(runs []*ffmpegRun) *ffmpegRun {
	var oldest *ffmpegRun
	for _, run := range runs {
		if run.live() && (oldest == nil || run.lastUsed.Before(oldest.lastUsed)) {
			oldest = run
		}
	}
	return oldest
}

/*
Function:	runLimit
Purpose:	the first segment after seg a new run would only write again: one on disk or the start
of a later run, segmentCount if there is none
Params:
  - dir: the transcode folder
  - seg: where the new run starts
  - count: segments in the video
  - runs: the transcode's runs
*/
func runLimit(dir string, seg, count int, runs []*ffmpegRun) int {
	limit := count
	// one listing instead of a stat per segment, the folder may be on the NAS
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if match := segmentNameRe.FindStringSubmatch(entry.Name()); match != nil {
			if n, _ := strconv.Atoi(match[1]); n > seg && n < limit {
				limit = n
			}
		}
	}
	for _, run := range runs {
		if run.live() && run.start > seg && run.start < limit {
			limit = run.start
		}
	}
	return limit
}

/*
Function:	stop
Purpose:	kill every ffmpeg run
*/
func (tc *transcode) stop() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	for _, run := range tc.runs {
		tc.killRun(run)
	}
	tc.prune()
}

/*
Function:	killRun
Purpose:	kill a run if it's running and wait for it to exit. tc.mu must be held
Params:
  - run: the run
*/
func (tc *transcode) killRun(run *ffmpegRun) {
	if !run.live() {
		return
	}
	run.stopped = true
	run.cmd.Process.Kill()
	// done closes before the exit handler locks tc.mu, so this can't deadlock
	<-run.done
}

/*
Function:	startRun
Purpose:	start an ffmpeg run writing segments from seg, stopping the least recently used run
first when there are already maxRunsPerTranscode. Other runs keep going, a run before seg
now stops when it gets there. tc.mu must be held
Params:
  - seg: the first segment to write
  - now: the current time
*/
func (tc *transcode) startRun(seg int, now time.Time) (*ffmpegRun, error) {
	live := 0
	for _, run := range tc.runs {
		if run.live() {
			live++
		}
	}
	if live >= maxRunsPerTranscode {
		tc.killRun(leastRecentlyUsed(tc.runs))
		tc.prune()
	}

	// every run appends, so the log shows each start. Runs at the same time interleave their warnings
	log, err := os.OpenFile(filepath.Join(tc.dir, logFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(log, "--- %s starting at segment %d\n", now.Format(time.RFC3339), seg)

	cmd := exec.Command("ffmpeg", transcodeArgs(tc.dir, tc.manifest, seg)...)
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		log.Close()
		return nil, err
	}

	count := segmentCount(tc.manifest.Duration)
	run := &ffmpegRun{cmd: cmd, start: seg, head: seg - 1, limit: runLimit(tc.dir, seg, count, tc.runs),
		lastUsed: now, done: make(chan struct{})}
	for _, other := range tc.runs {
		if other.live() && other.start < seg && seg < other.limit {
			other.limit = seg
		}
	}
	tc.runs = append(tc.runs, run)
	if !tc.reaping {
		tc.reaping = true
		go tc.reapIdle()
	}

	go func() {
		err := cmd.Wait()
		log.Close()
		close(run.done)

		tc.mu.Lock()
		defer tc.mu.Unlock()
		if err != nil && !run.stopped {
			logger.Error("transcode %s failed: %v\n%s", tc.dir, err, tailFile(filepath.Join(tc.dir, logFile), 2048))
			tc.err = err
			// the others must not write into the folder once it's gone
			for _, other := range tc.runs {
				tc.killRun(other)
			}
			tc.prune()
			os.RemoveAll(tc.dir)
		}
	}()
	return run, nil
}

/*
Function:	reapIdle
Purpose:	while runs are going, stop the ones no viewer uses any more (the tab was closed) or that
caught up with segments already written, even when no segment is requested
*/
func (tc *transcode) reapIdle() {
	ticker := time.NewTicker(reapInterval)
	defer ticker.Stop()
	for range ticker.C {
		tc.mu.Lock()
		tc.refresh(time.Now())
		running := false
		for _, run := range tc.runs {
			running = running || run.live()
		}
		if !running {
			tc.reaping = false
			tc.mu.Unlock()
			return
		}
		tc.mu.Unlock()
	}
}

/*
Function:	fileExists
Purpose:	whether a file exists
Params:
  - path: the file
*/
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

/*
Function:	transcodeArgs
Purpose:	ffmpeg arguments for an HLS transcode of the chosen video and audio tracks from a segment
onwards. Timestamps stay those of the whole video, so segments from different runs
and the subtitles line up
Params:
  - dir: the transcode folder
  - manifest: the source file and chosen tracks
  - startSeg: the first segment to write
*/
func transcodeArgs(dir string, manifest streamManifest, startSeg int) []string {
	encoder := transcodeEncoder()
	offset := strconv.Itoa(startSeg * segmentSecs)
	args := []string{"-y", "-v", "warning"}
	if encoder == "vaapi" {
		args = append(args, "-vaapi_device", vaapiDevice())
	}
	if accel := hwaccelFor(manifest.VideoCodec); accel != "" {
		args = append(args, "-hwaccel", accel)
	}
	if startSeg > 0 {
		args = append(args, "-ss", offset)
	}
	args = append(args, inputOnlyLocal...)
	args = append(args, "-i", manifest.Source)

	filter, codec := encoderArgs(encoder)
	if manifest.BurnIndex >= 0 {
		// the subtitle is scaled to the video first, DVD subtitles are often a smaller canvas.
		// The overlay runs on the CPU before any upload to the GPU
		args = append(args, "-filter_complex", fmt.Sprintf(
			"[0:%d][0:v:0]scale2ref[sub][video];[video][sub]overlay=eof_action=pass,%s[out]",
			manifest.BurnIndex, filter), "-map", "[out]")
	} else {
		args = append(args, "-map", "0:v:0", "-vf", filter)
	}
	if manifest.AudioIndex >= 0 {
		args = append(args, "-map", fmt.Sprintf("0:%d", manifest.AudioIndex))
	}
	args = append(args, codec...)
	args = append(args,
		// keyframe every segment so HLS can cut there
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentSecs),
		"-c:a", "aac", "-b:a", "192k", "-ac", "2",
	)
	if startSeg > 0 {
		args = append(args, "-output_ts_offset", offset)
	}
	args = append(args,
		"-hls_time", strconv.Itoa(segmentSecs),
		"-hls_list_size", "0",
		"-start_number", strconv.Itoa(startSeg),
		"-hls_segment_filename", filepath.Join(dir, "playlist%d.ts"),
		// segments are written to .tmp and renamed, so one that exists is complete
		"-hls_flags", "temp_file",
		"-f", "hls",
		// ffmpeg's own playlist, never served (playlist.m3u8 lists every segment up front).
		// One per run, so runs at the same time don't write the same file
		filepath.Join(dir, fmt.Sprintf("ffmpeg-%d.m3u8", startSeg)),
	)
	return args
}

/*
Function:	transcodeEncoder
Purpose:	H.264 encoder to use: "amf" (AMD on Windows), "vaapi" (AMD/Intel GPU on Linux), "nvenc"
(NVIDIA GPU) or "libx264" (CPU). TRANSCODE_ENCODER overrides the platform default
*/
func transcodeEncoder() string {
	switch enc := os.Getenv("TRANSCODE_ENCODER"); enc {
	case "amf", "vaapi", "nvenc", "libx264":
		return enc
	case "":
	default:
		logger.Warn("unknown TRANSCODE_ENCODER %q, using the platform default", enc)
	}
	if runtime.GOOS == "windows" {
		return "amf"
	}
	return "libx264"
}

/*
Function:	vaapiDevice
Purpose:	render node for VAAPI encoding, TRANSCODE_VAAPI_DEVICE overrides the usual first GPU
*/
func vaapiDevice() string {
	if dev := os.Getenv("TRANSCODE_VAAPI_DEVICE"); dev != "" {
		return dev
	}
	return "/dev/dri/renderD128"
}

/*
Function:	encoderArgs
Purpose:	the video filter an encoder from transcodeEncoder needs, and its encoder arguments.
All of them cap the bitrate, uncapped AMF output was 9-22 Mbps which stalls over the tunnel
Params:
  - encoder: "amf", "vaapi", "nvenc" or "libx264"
*/
func encoderArgs(encoder string) (string, []string) {
	rate := []string{"-b:v", "6M", "-maxrate", "8M", "-bufsize", "12M"}
	switch encoder {
	case "amf":
		// forced_idr makes AMF's forced keyframes real cut points, libx264 and VAAPI already do
		return "format=yuv420p", append([]string{"-c:v", "h264_amf", "-rc", "vbr_peak", "-forced_idr", "1"}, rate...)
	case "vaapi":
		// frames are uploaded to the GPU after converting, the encoder only takes GPU surfaces
		return "format=nv12,hwupload", append([]string{"-c:v", "h264_vaapi"}, rate...)
	case "nvenc":
		// 8-bit first, h264_nvenc has no 10-bit mode. forced-idr makes the forced keyframes cut points
		return "format=yuv420p", append([]string{"-c:v", "h264_nvenc", "-preset", "p4", "-rc", "vbr", "-forced-idr", "1"}, rate...)
	default:
		return "format=yuv420p", append([]string{"-c:v", "libx264", "-preset", "veryfast"}, rate...)
	}
}

/*
Function:	hwaccelFor
Purpose:	GPU decoder to use for a source codec. Only AV1 gets one: this ffmpeg decodes it with
libaom at ~3.5x realtime vs ~15x on the GPU, while HEVC/H.264 are already fast on the
CPU (and slower with -hwaccel auto). TRANSCODE_HWACCEL overrides the platform
default, "none" turns it off
Params:
  - codec: the source video codec from ffprobe
*/
func hwaccelFor(codec string) string {
	if codec != "av1" {
		return ""
	}
	accel := os.Getenv("TRANSCODE_HWACCEL")
	if accel == "" && runtime.GOOS == "windows" {
		accel = "d3d11va"
	}
	if accel == "none" {
		return ""
	}
	return accel
}

/*
Function:	extractSubtitles
Purpose:	extract the attached fonts, then convert the text subtitle tracks to WebVTT (and keep
ASS tracks as ASS too) in one pass over the file. Each file is written to .tmp and
renamed when complete so the player never loads half a file
Params:
  - dir: the transcode folder
  - manifest: the source file, its subtitle tracks and fonts
*/
func extractSubtitles(dir string, manifest streamManifest) {
	filePath := manifest.Source
	extractFonts(filePath, dir, manifest.Fonts)

	var subs []subtitleTrack
	for _, s := range manifest.Subtitles {
		if !s.Image {
			subs = append(subs, s)
		}
	}
	if len(subs) == 0 {
		return
	}

	run := func(tracks []subtitleTrack) error {
		args := append([]string{"-v", "error", "-y"}, inputOnlyLocal...)
		args = append(args, "-i", filePath)
		for _, s := range tracks {
			args = append(args, "-map", fmt.Sprintf("0:%d", s.Index), "-c:s", "webvtt", "-f", "webvtt",
				filepath.Join(dir, s.File+".tmp"))
			if s.AssFile != "" {
				args = append(args, "-map", fmt.Sprintf("0:%d", s.Index), "-c:s", "ass", "-f", "ass",
					filepath.Join(dir, s.AssFile+".tmp"))
			}
		}
		out, err := exec.Command("ffmpeg", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	if err := run(subs); err != nil {
		// one bad track fails the whole pass, retry individually so the others still work
		logger.Warn("subtitle extraction in %s failed, retrying per track: %v", dir, err)
		for _, s := range subs {
			if err := run([]subtitleTrack{s}); err != nil {
				logger.Warn("subtitle track %d in %s failed: %v", s.Index, dir, err)
			}
		}
	}

	for _, s := range subs {
		if s.AssFile != "" {
			os.Rename(filepath.Join(dir, s.AssFile+".tmp"), filepath.Join(dir, s.AssFile))
		}
		tmp := filepath.Join(dir, s.File+".tmp")
		data, err := os.ReadFile(tmp)
		if err != nil {
			continue
		}
		if err := os.WriteFile(tmp, []byte(cleanVTT(string(data))), 0644); err != nil {
			continue
		}
		os.Rename(tmp, filepath.Join(dir, s.File))
	}
}

/*
Function:	extractFonts
Purpose:	write the fonts attached to a file into the transcode folder, for rendering ASS subtitles
Params:
  - filePath: the source file
  - dir: the transcode folder
  - fonts: the attachments to extract
*/
func extractFonts(filePath, dir string, fonts []fontFile) {
	if len(fonts) == 0 {
		return
	}
	args := []string{"-v", "error", "-y"}
	for _, f := range fonts {
		args = append(args, fmt.Sprintf("-dump_attachment:%d", f.Index), filepath.Join(dir, f.File+".tmp"))
	}
	args = append(args, inputOnlyLocal...)
	// dumping happens on opening the input, nothing needs decoding
	args = append(args, "-i", filePath, "-t", "0", "-f", "null", "-")
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		logger.Warn("font extraction in %s failed: %v: %s", dir, err, strings.TrimSpace(string(out)))
	}
	for _, f := range fonts {
		os.Rename(filepath.Join(dir, f.File+".tmp"), filepath.Join(dir, f.File))
	}
}

var (
	// ASS override blocks ffmpeg leaves in the text, e.g. {=74} or {\an8}
	assTagRe  = regexp.MustCompile(`\{[^}]*\}`)
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
	// ASS vector drawings used for typesetting ("m 735.73 28.24 b 735.73 ..."), which
	// WebVTT can't draw and would otherwise show as walls of numbers
	drawingRe = regexp.MustCompile(`^[mnlbspc]\s+-?\d[\d.\s\-mnlbspc]*$`)
)

var numberRe = regexp.MustCompile(`-?\d+(?:\.\d+)?`)

/*
Function:	isDrawing
Purpose:	whether cue text is an ASS vector drawing. Drawings are only drawing commands and
coordinates, and always have several points, so short lines like "b 2" are kept
Params:
  - text: the visible cue text
*/
func isDrawing(text string) bool {
	return drawingRe.MatchString(text) && len(numberRe.FindAllString(text, 5)) >= 4
}

/*
Function:	cleanVTT
Purpose:	tidy ffmpeg's ASS to WebVTT output: strip leftover ASS tags, drop cues that are
vector drawings or empty, and drop repeated cues (ASS often draws a line several
times for styling layers)
Params:
  - vtt: the WebVTT file contents
*/
func cleanVTT(vtt string) string {
	blocks := strings.Split(strings.ReplaceAll(vtt, "\r\n", "\n"), "\n\n")
	seen := map[string]bool{}
	kept := make([]string, 0, len(blocks))
	for _, b := range blocks {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}

		if !strings.HasPrefix(b, "WEBVTT") {
			var out []string
			visible := ""
			for _, line := range strings.Split(b, "\n") {
				if strings.Contains(line, "-->") {
					out = append(out, line)
					continue
				}
				line = strings.TrimSpace(assTagRe.ReplaceAllString(line, ""))
				text := strings.TrimSpace(htmlTagRe.ReplaceAllString(line, ""))
				if text == "" || isDrawing(text) {
					continue
				}
				visible += text
				out = append(out, line)
			}
			if visible == "" {
				continue
			}
			b = strings.Join(out, "\n")
		}

		if seen[b] {
			continue
		}
		seen[b] = true
		kept = append(kept, b)
	}
	return strings.Join(kept, "\n\n") + "\n"
}

/*
Function:	writeJSONAtomic
Purpose:	write JSON to a temp file then rename so readers never see a partial file
Params:
  - path: destination file
  - v: the value to encode
*/
func writeJSONAtomic(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

/*
Function:	tailFile
Purpose:	last n bytes of a file, for logging ffmpeg errors
Params:
  - path: the file
  - n: how many bytes
*/
func tailFile(path string, n int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > n {
		f.Seek(-n, io.SeekEnd)
	}
	data, _ := io.ReadAll(f)
	return strings.TrimSpace(string(data))
}
