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
	// ffmpeg's own playlist, never served: playlist.m3u8 lists every segment up front instead
	ffmpegPlaylistFile = "ffmpeg.m3u8"
	logFile            = "transcode.log"

	segmentSecs = 4
	// segments from the start position that must exist before the player is pointed at it
	readySegments = 3
	readyTimeout  = 90 * time.Second
	// how long a segment request waits for ffmpeg, under the player's 40s first-byte timeout
	segmentTimeout = 30 * time.Second
	readyPoll      = 200 * time.Millisecond
	// a request this many segments past what ffmpeg has written restarts it there instead of waiting
	restartAheadSegments = 6
)

var errSegmentMissing = errors.New("segment was not written")

// a transcode folder for one media and audio track, shared by every request for it. ffmpeg
// may be running, finished, or restarted at whichever segment the player asks for next
type transcode struct {
	dir      string
	manifest streamManifest

	mu  sync.Mutex
	run *ffmpegRun // the current or last ffmpeg run, nil before the first
	err error      // a run failed and the folder was removed, the next play starts over
}

// one ffmpeg process writing segments from start onwards
type ffmpegRun struct {
	cmd     *exec.Cmd
	start   int
	head    int  // last segment written without a gap from start
	stopped bool // killed for a restart, so exiting isn't a failure
	done    chan struct{}
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
	actionRestart
	actionMissing
)

// what waitForSegment knows about the latest ffmpeg run
type runState struct {
	exists   bool // ffmpeg has run since the transcode was loaded
	running  bool
	finished bool // exited by itself after writing from start to the end
	start    int
	head     int
}

/*
Function:	segmentAction
Purpose:	what to do about a segment that doesn't exist yet: wait for ffmpeg to reach it, restart
            ffmpeg there (it isn't running, or is behind or too far before the segment), or give
            up because a finished run passed it without writing it
Params:
  - seg: the segment wanted
  - run: the latest ffmpeg run
*/
func segmentAction(seg int, run runState) action {
	switch {
	case !run.exists:
		return actionRestart
	case run.running:
		if seg < run.start || seg > run.head+restartAheadSegments {
			return actionRestart
		}
		return actionWait
	case run.finished && seg >= run.start:
		return actionMissing
	default:
		return actionRestart
	}
}

/*
Function:	waitForSegment
Purpose:	block until a segment exists, starting or restarting ffmpeg at it when waiting
            wouldn't get there soon
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
		switch segmentAction(seg, tc.state()) {
		case actionMissing:
			tc.mu.Unlock()
			return errSegmentMissing
		case actionRestart:
			if err := tc.startRun(seg); err != nil {
				tc.mu.Unlock()
				return err
			}
		}
		done := tc.run.done
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
Function:	state
Purpose:	the latest run's progress, moving its head past any segments written since. tc.mu must be held
*/
func (tc *transcode) state() runState {
	run := tc.run
	if run == nil {
		return runState{}
	}
	for fileExists(filepath.Join(tc.dir, segmentName(run.head+1))) {
		run.head++
	}
	state := runState{exists: true, start: run.start, head: run.head}
	select {
	case <-run.done:
		state.finished = !run.stopped && run.cmd.ProcessState.Success()
	default:
		state.running = true
	}
	return state
}

/*
Function:	stop
Purpose:	kill ffmpeg if it's running
*/
func (tc *transcode) stop() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.killRun()
}

/*
Function:	killRun
Purpose:	kill ffmpeg if it's running and wait for it to exit. tc.mu must be held
*/
func (tc *transcode) killRun() {
	run := tc.run
	if run == nil {
		return
	}
	select {
	case <-run.done:
	default:
		run.stopped = true
		run.cmd.Process.Kill()
		// done closes before the exit handler locks tc.mu, so this can't deadlock
		<-run.done
	}
}

/*
Function:	startRun
Purpose:	stop any running ffmpeg and start one writing segments from seg. Segments already on disk
            are kept. tc.mu must be held
Params:
  - seg: the first segment to write
*/
func (tc *transcode) startRun(seg int) error {
	tc.killRun()

	// every run appends, so the log shows each restart
	log, err := os.OpenFile(filepath.Join(tc.dir, logFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	fmt.Fprintf(log, "--- %s starting at segment %d\n", time.Now().Format(time.RFC3339), seg)

	cmd := exec.Command("ffmpeg", transcodeArgs(tc.dir, tc.manifest, seg)...)
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		log.Close()
		return err
	}

	run := &ffmpegRun{cmd: cmd, start: seg, head: seg - 1, done: make(chan struct{})}
	tc.run = run
	go func() {
		err := cmd.Wait()
		log.Close()
		close(run.done)

		tc.mu.Lock()
		defer tc.mu.Unlock()
		if err != nil && !run.stopped {
			logger.Error("transcode %s failed: %v\n%s", tc.dir, err, tailFile(filepath.Join(tc.dir, logFile), 2048))
			tc.err = err
			os.RemoveAll(tc.dir)
		}
	}()
	return nil
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
		filepath.Join(dir, ffmpegPlaylistFile),
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
