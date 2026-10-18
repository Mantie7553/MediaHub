package stream

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

const (
	manifestFile = "stream.json"
	playlistFile = "playlist.m3u8"
	logFile      = "transcode.log"

	segmentSecs = 4
	// how much video past the start position must exist before the player is pointed at it
	readyAheadSecs = 3 * segmentSecs
	readyTimeout   = 90 * time.Second
	readyPoll      = 200 * time.Millisecond
)

// a running ffmpeg transcode, shared by every request for the same media
type transcode struct {
	dir      string
	manifest streamManifest
	done     chan struct{} // closed when ffmpeg exits
	err      error         // set before done is closed
}

/*
Function:	getOrStartTranscode
Purpose:	reuse a running or finished transcode, otherwise clear any leftovers and start one
Params:
  - key: type-id-a<audio> of the media and audio track
  - dir: the transcode folder for this media and audio track
  - filePath: the source file
  - manifest: the tracks this transcode should contain
*/
func (h *Handler) getOrStartTranscode(key, dir, filePath string, manifest streamManifest) (*transcode, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if tc, ok := h.transcodes[key]; ok {
		return tc, nil
	}

	if done, ok := completedTranscode(dir); ok {
		closed := make(chan struct{})
		close(closed)
		return &transcode{dir: dir, manifest: done, done: closed}, nil
	}

	// nothing running and not complete: a failed or interrupted run, or one made before
	// stream.json existed. Start over
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	if err := writeJSONAtomic(filepath.Join(dir, manifestFile), manifest); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	log, err := os.Create(filepath.Join(dir, logFile))
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	cmd := exec.Command("ffmpeg", transcodeArgs(filePath, dir, manifest)...)
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		log.Close()
		os.RemoveAll(dir)
		return nil, err
	}

	tc := &transcode{dir: dir, manifest: manifest, done: make(chan struct{})}
	h.transcodes[key] = tc

	go extractSubtitles(filePath, dir, manifest.Subtitles)

	go func() {
		err := cmd.Wait()
		log.Close()

		h.mu.Lock()
		tc.err = err
		delete(h.transcodes, key)
		if err != nil {
			logger.Error("transcode %s failed: %v\n%s", key, err, tailFile(filepath.Join(dir, logFile), 2048))
			os.RemoveAll(dir)
		}
		h.mu.Unlock()
		close(tc.done)
	}()

	return tc, nil
}

/*
Function:	transcodeArgs
Purpose:	ffmpeg arguments for an HLS transcode of the chosen video and audio tracks
Params:
  - filePath: the source file
  - dir: the transcode folder
  - manifest: the chosen tracks
*/
func transcodeArgs(filePath, dir string, manifest streamManifest) []string {
	encoder := transcodeEncoder()
	args := []string{"-v", "warning"}
	if encoder == "vaapi" {
		args = append(args, "-vaapi_device", vaapiDevice())
	}
	if accel := hwaccelFor(manifest.VideoCodec); accel != "" {
		args = append(args, "-hwaccel", accel)
	}
	args = append(args, "-i", filePath, "-map", "0:v:0")
	if manifest.AudioIndex >= 0 {
		args = append(args, "-map", fmt.Sprintf("0:%d", manifest.AudioIndex))
	}
	args = append(args, encoderArgs(encoder)...)
	args = append(args,
		// keyframe every segment so HLS can cut there
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentSecs),
		"-c:a", "aac", "-b:a", "192k", "-ac", "2",
		"-hls_time", strconv.Itoa(segmentSecs),
		"-hls_list_size", "0",
		// EVENT tells the player the playlist only grows, ENDLIST is added when done
		"-hls_playlist_type", "event",
		"-f", "hls",
		filepath.Join(dir, playlistFile),
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
Purpose:	ffmpeg filter and video encoder arguments for an encoder from transcodeEncoder. All of
            them cap the bitrate, uncapped AMF output was 9-22 Mbps which stalls over the tunnel
Params:
  - encoder: "amf", "vaapi", "nvenc" or "libx264"
*/
func encoderArgs(encoder string) []string {
	rate := []string{"-b:v", "6M", "-maxrate", "8M", "-bufsize", "12M"}
	switch encoder {
	case "amf":
		// forced_idr makes AMF's forced keyframes real cut points, libx264 and VAAPI already do
		return append([]string{"-vf", "format=yuv420p", "-c:v", "h264_amf", "-rc", "vbr_peak", "-forced_idr", "1"}, rate...)
	case "vaapi":
		// frames are uploaded to the GPU after converting, the encoder only takes GPU surfaces
		return append([]string{"-vf", "format=nv12,hwupload", "-c:v", "h264_vaapi"}, rate...)
	case "nvenc":
		// 8-bit first, h264_nvenc has no 10-bit mode. forced-idr makes the forced keyframes cut points
		return append([]string{"-vf", "format=yuv420p", "-c:v", "h264_nvenc", "-preset", "p4", "-rc", "vbr", "-forced-idr", "1"}, rate...)
	default:
		return append([]string{"-vf", "format=yuv420p", "-c:v", "libx264", "-preset", "veryfast"}, rate...)
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
Function:	waitForPlaylist
Purpose:	block until the playlist covers start + a few segments, the transcode ends, or it times out
Params:
  - tc: the transcode to wait on
  - start: the position (seconds) the player will start from
*/
func waitForPlaylist(tc *transcode, start float64) error {
	deadline := time.Now().Add(readyTimeout)
	path := filepath.Join(tc.dir, playlistFile)

	for {
		duration, ended := playlistDuration(path)
		if ended || duration >= start+readyAheadSecs {
			return nil
		}

		select {
		case <-tc.done:
			if tc.err != nil {
				return tc.err
			}
			if _, ended := playlistDuration(path); ended {
				return nil
			}
			return errors.New("transcode exited without finishing the playlist")
		case <-time.After(readyPoll):
		}

		if time.Now().After(deadline) {
			// slow machine or far resume point: hand over what exists and let the player wait
			if duration > 0 {
				return nil
			}
			return errors.New("timed out waiting for transcode")
		}
	}
}

/*
Function:	playlistDuration
Purpose:	total length of the segments listed so far, and whether the playlist is complete
Params:
  - path: the playlist file
*/
func playlistDuration(path string) (float64, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()

	var total float64
	ended := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "#EXTINF:"):
			value := strings.SplitN(strings.TrimPrefix(line, "#EXTINF:"), ",", 2)[0]
			if d, err := strconv.ParseFloat(value, 64); err == nil {
				total += d
			}
		case line == "#EXT-X-ENDLIST":
			ended = true
		}
	}
	return total, ended
}

/*
Function:	completedTranscode
Purpose:	load the manifest of a transcode that already finished
Params:
  - dir: the transcode folder
*/
func completedTranscode(dir string) (streamManifest, bool) {
	var manifest streamManifest
	data, err := os.ReadFile(filepath.Join(dir, manifestFile))
	if err != nil || json.Unmarshal(data, &manifest) != nil {
		return manifest, false
	}
	if _, ended := playlistDuration(filepath.Join(dir, playlistFile)); !ended {
		return manifest, false
	}
	return manifest, true
}

/*
Function:	extractSubtitles
Purpose:	convert the chosen subtitle tracks to WebVTT in one pass over the file. Each is written
            to a .tmp file and renamed when complete so the player never loads half a file
Params:
  - filePath: the source file
  - dir: the transcode folder
  - subs: the tracks to extract
*/
func extractSubtitles(filePath, dir string, subs []subtitleTrack) {
	if len(subs) == 0 {
		return
	}

	run := func(tracks []subtitleTrack) error {
		args := []string{"-v", "error", "-y", "-i", filePath}
		for _, s := range tracks {
			args = append(args, "-map", fmt.Sprintf("0:%d", s.Index), "-c:s", "webvtt", "-f", "webvtt",
				filepath.Join(dir, s.File+".tmp"))
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
