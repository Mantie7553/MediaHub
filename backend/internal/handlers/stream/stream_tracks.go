package stream

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultAudioLanguage    = "jpn"
	defaultSubtitleLanguage = "eng"
	// audio preference meaning "whatever the file marks as default"
	audioFileDefault = "default"
	// subtitle preference meaning no subtitles
	subtitlesOff = "off"
)

// subtitle codecs ffmpeg can turn into WebVTT; image based ones (PGS, VobSub) can't be
var textSubtitleCodecs = map[string]bool{
	"ass": true, "ssa": true, "subrip": true, "srt": true,
	"mov_text": true, "webvtt": true, "text": true,
}

var (
	commentaryRe = regexp.MustCompile(`(?i)comment`)
	signsRe      = regexp.MustCompile(`(?i)\bsigns?\b`)
	dialogueRe   = regexp.MustCompile(`(?i)dialog|full`)
	lessUsefulRe = regexp.MustCompile(`(?i)\bsdh\b|dubtitle|\bcc\b`)
)

// inputOnlyLocal goes before every input ffmpeg and ffprobe open: a media file can only read
// local files, so a file that is really a playlist or a reference to somewhere else (a
// torrented "episode", say) can't make the server send requests
var inputOnlyLocal = []string{"-protocol_whitelist", "file"}

/*
Function:	probeStreams
Purpose:	list the streams in a media file and its length with ffprobe
Params:
  - filePath: the file to inspect
*/
func probeStreams(filePath string) (probeResult, error) {
	args := append([]string{"-v", "error"}, inputOnlyLocal...)
	out, err := exec.Command("ffprobe", append(args,
		"-show_entries", "stream=index,codec_type,codec_name:stream_tags=language,title:stream_disposition=default,forced:format=duration",
		"-of", "json",
		filePath,
	)...).Output()
	if err != nil {
		return probeResult{}, fmt.Errorf("ffprobe: %w", err)
	}

	var parsed probeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return probeResult{}, fmt.Errorf("ffprobe output: %w", err)
	}
	duration, _ := strconv.ParseFloat(parsed.Format.Duration, 64)
	return probeResult{Streams: parsed.Streams, Duration: duration}, nil
}

// probe results by file, so picking tracks doesn't run ffprobe on every play
type probeCache struct {
	mu      sync.Mutex
	entries map[string]probeCacheEntry
}

type probeCacheEntry struct {
	size    int64
	modTime time.Time
	result  probeResult
}

/*
Function:	get
Purpose:	cached ffprobe result for a file, re-probed if the file changed
Params:
  - filePath: the file to inspect
*/
func (c *probeCache) get(filePath string) (probeResult, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return probeResult{}, err
	}

	c.mu.Lock()
	entry, ok := c.entries[filePath]
	c.mu.Unlock()
	if ok && entry.size == info.Size() && entry.modTime.Equal(info.ModTime()) {
		return entry.result, nil
	}

	result, err := probeStreams(filePath)
	if err != nil {
		return probeResult{}, err
	}

	c.mu.Lock()
	if c.entries == nil {
		c.entries = map[string]probeCacheEntry{}
	}
	c.entries[filePath] = probeCacheEntry{size: info.Size(), modTime: info.ModTime(), result: result}
	c.mu.Unlock()
	return result, nil
}

/*
Function:	audioTracks
Purpose:	the audio tracks a user can pick from
Params:
  - streams: the file's streams from ffprobe
*/
func audioTracks(streams []probeStream) []audioTrack {
	tracks := []audioTrack{}
	for _, s := range streams {
		if s.CodecType == "audio" {
			tracks = append(tracks, audioTrack{Index: s.Index, Language: s.Tags.Language, Label: trackLabel(s)})
		}
	}
	return tracks
}

/*
Function:	selectAudio
Purpose:	pick the audio track for a language preference: that language, then the file's
            default track, then the first one. Commentary is only used if nothing else exists
Params:
  - streams: the file's streams from ffprobe
  - preferred: a language code, or "default" for the file's default track
*/
func selectAudio(streams []probeStream, preferred string) *probeStream {
	var audio []probeStream
	for _, s := range streams {
		if s.CodecType == "audio" && !commentaryRe.MatchString(s.Tags.Title) {
			audio = append(audio, s)
		}
	}
	if len(audio) == 0 {
		for _, s := range streams {
			if s.CodecType == "audio" {
				audio = append(audio, s)
			}
		}
	}
	if len(audio) == 0 {
		return nil
	}

	pick := func(match func(s probeStream) bool) *probeStream {
		// default-flagged tracks first, then file order
		for _, s := range audio {
			if match(s) && s.Disposition.Default == 1 {
				return &s
			}
		}
		for _, s := range audio {
			if match(s) {
				return &s
			}
		}
		return nil
	}

	if preferred != audioFileDefault {
		if s := pick(func(s probeStream) bool { return s.Tags.Language == preferred }); s != nil {
			return s
		}
	}
	return pick(func(probeStream) bool { return true })
}

/*
Function:	subtitleTracks
Purpose:	the text subtitle tracks to extract, with what's needed to choose a default later
Params:
  - streams: the file's streams from ffprobe
*/
func subtitleTracks(streams []probeStream) []subtitleTrack {
	tracks := []subtitleTrack{}
	for _, s := range streams {
		if s.CodecType != "subtitle" || !textSubtitleCodecs[s.CodecName] {
			continue
		}
		tracks = append(tracks, subtitleTrack{
			Index:       s.Index,
			File:        fmt.Sprintf("subs_%d.vtt", s.Index),
			Language:    s.Tags.Language,
			Label:       trackLabel(s),
			Title:       s.Tags.Title,
			Forced:      s.Disposition.Forced == 1,
			FileDefault: s.Disposition.Default == 1,
		})
	}
	return tracks
}

/*
Function:	buildManifest
Purpose:	what a transcode with this audio track contains
Params:
  - filePath: the source file
  - probe: the file's streams and length from ffprobe
  - audioIndex: the chosen audio stream, -1 for none
*/
func buildManifest(filePath string, probe probeResult, audioIndex int) streamManifest {
	manifest := streamManifest{
		Source:     filePath,
		Duration:   probe.Duration,
		AudioIndex: audioIndex,
		Subtitles:  subtitleTracks(probe.Streams),
	}
	for _, s := range probe.Streams {
		if s.CodecType == "video" && manifest.VideoCodec == "" {
			manifest.VideoCodec = s.CodecName
		}
		if s.Index == audioIndex {
			manifest.AudioLanguage = s.Tags.Language
		}
	}
	return manifest
}

/*
Function:	matchesLanguage
Purpose:	whether a track is in the preferred language. English with honorifics (enm) counts as English
Params:
  - language: the track's language
  - preferred: the preferred language
*/
func matchesLanguage(language, preferred string) bool {
	return language == preferred || (preferred == "eng" && language == "enm")
}

/*
Function:	defaultSubtitle
Purpose:	pick the subtitle track to show. If the audio is already in the preferred subtitle
            language only a signs/songs track is shown, otherwise full dialogue, avoiding
            signs-only, SDH and dubtitle tracks. Returns the stream index, or -1 for none
Params:
  - tracks: the extracted subtitle tracks
  - audioLanguage: language of the chosen audio track
  - preferred: the preferred subtitle language, or "off"
*/
func defaultSubtitle(tracks []subtitleTrack, audioLanguage, preferred string) int {
	if preferred == subtitlesOff {
		return -1
	}
	sameLanguageAudio := matchesLanguage(audioLanguage, preferred)

	best, bestScore := -1, 0
	for _, t := range tracks {
		if !matchesLanguage(t.Language, preferred) {
			continue
		}

		signsOnly := t.Forced || (signsRe.MatchString(t.Title) && !dialogueRe.MatchString(t.Title))
		// audio the viewer understands only needs signs translated, never full dialogue
		if sameLanguageAudio && !signsOnly {
			continue
		}

		score := 100
		if t.Language != preferred { // e.g. English with honorifics
			score -= 10
		}
		if signsOnly && !sameLanguageAudio {
			score -= 80
		}
		if lessUsefulRe.MatchString(t.Title) {
			score -= 20
		}
		if t.FileDefault {
			score += 5
		}

		if score > bestScore {
			best, bestScore = t.Index, score
		}
	}
	return best
}

/*
Function:	trackLabel
Purpose:	readable name for an audio or subtitle track in the player's menus
Params:
  - s: the stream
*/
func trackLabel(s probeStream) string {
	lang, known := languageNames[s.Tags.Language]
	if !known {
		lang = strings.ToUpper(s.Tags.Language)
	}
	title := strings.TrimSpace(s.Tags.Title)

	switch {
	case title != "" && lang != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(lang)):
		return lang + " - " + title
	case title != "":
		return title
	case lang != "":
		return lang
	default:
		return fmt.Sprintf("Track %d", s.Index)
	}
}

var languageNames = map[string]string{
	"eng": "English", "enm": "English (Honorifics)", "jpn": "Japanese", "spa": "Spanish",
	"fre": "French", "fra": "French", "ger": "German", "deu": "German", "ita": "Italian",
	"por": "Portuguese", "rus": "Russian", "ara": "Arabic", "chi": "Chinese", "zho": "Chinese",
	"kor": "Korean", "ind": "Indonesian", "hin": "Hindi", "zxx": "", "und": "",
}
