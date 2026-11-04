package stream

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/aniskip"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db      *sql.DB
	aniskip *aniskip.AniSkipClient

	mu         sync.Mutex
	transcodes map[string]*transcode // transcodes in use by type-id-a<audio>
	probes     probeCache
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db, aniskip: aniskip.NewAniSkipClient(), transcodes: map[string]*transcode{}}
}

// one transcode folder per media, audio track and burned in subtitle, e.g. a2 or a2-b5
var variantRe = regexp.MustCompile(`^a(-1|\d+)(-b\d+)?$`)

/*
Function:	StreamMedia
Purpose:	Start (or reuse) the transcode of an episode or movie and return signed playlist and
subtitle URLs once enough of it exists to play from the optional ?start= position,
where ffmpeg starts when those segments don't exist yet.
?audio= picks an audio stream, otherwise the user's preferred language is used.
?burn= burns an image subtitle into the video, -1 for none; without it one is burned
in when it's the subtitle the user's preferences pick
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) StreamMedia(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "type")
	mediaID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	filePath, ok := h.videoFile(w, mediaType, mediaID)
	if !ok {
		return
	}

	start, _ := strconv.ParseFloat(r.URL.Query().Get("start"), 64)
	if start < 0 {
		start = 0
	}

	probe, err := h.probes.get(filePath)
	if utils.InternalError(w, err) {
		return
	}
	streams := probe.Streams
	prefs := h.languagePreferences(auth.GetUser(r).UserID)

	audioIndex := -1
	if requested := r.URL.Query().Get("audio"); requested != "" {
		idx, err := strconv.Atoi(requested)
		if err != nil || !hasAudioStream(streams, idx) {
			utils.Error(w, http.StatusBadRequest, "invalid audio track")
			return
		}
		audioIndex = idx
	} else if audio := selectAudio(streams, prefs.Audio); audio != nil {
		audioIndex = audio.Index
	}

	manifest := buildManifest(filePath, probe, audioIndex, -1)
	if requested := r.URL.Query().Get("burn"); requested != "" {
		idx, err := strconv.Atoi(requested)
		if err != nil || (idx != -1 && !imageSubtitle(manifest.Subtitles, idx)) {
			utils.Error(w, http.StatusBadRequest, "invalid subtitle track")
			return
		}
		manifest.BurnIndex = idx
	} else if idx := defaultSubtitle(manifest.Subtitles, manifest.AudioLanguage, prefs.Subtitle); imageSubtitle(manifest.Subtitles, idx) {
		manifest.BurnIndex = idx
	}

	variant := "a" + strconv.Itoa(audioIndex)
	if manifest.BurnIndex >= 0 {
		variant += "-b" + strconv.Itoa(manifest.BurnIndex)
	}
	key := mediaType + "-" + mediaID + "-" + variant
	tempDir := filepath.Join(os.Getenv("MEDIA_ROOT"), "temp", key)

	tc, err := h.getTranscode(key, tempDir, manifest)
	if utils.InternalError(w, err) {
		return
	}

	if err := waitUntilReady(tc, start); err != nil {
		logger.Error("transcode %s not ready: %v", key, err)
		utils.Error(w, http.StatusInternalServerError, "transcode failed to start")
		return
	}

	writeStreamURLs(w, r, mediaType, mediaID, variant, tc.manifest, audioTracks(streams), prefs)
}

/*
Function:	videoFile
Purpose:	the file of an episode or movie, responding with an error when there isn't one
Params:
  - w: http response writer, for the error
  - mediaType: episode or movie
  - mediaID: the id of the episode or movie
*/
func (h *Handler) videoFile(w http.ResponseWriter, mediaType, mediaID string) (string, bool) {
	var filePath string
	var err error
	switch mediaType {
	case "episode":
		err = h.db.QueryRow(
			`SELECT file_path FROM episodes WHERE id = $1`,
			mediaID,
		).Scan(&filePath)
	case "movie":
		err = h.db.QueryRow(
			`SELECT file_path FROM movie_metadata WHERE media_item_id = $1`,
			mediaID,
		).Scan(&filePath)
	default:
		utils.Error(w, http.StatusBadRequest, "invalid media type")
		return "", false
	}

	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, mediaType+" not found")
		return "", false
	}
	if utils.InternalError(w, err) {
		return "", false
	}
	return filePath, true
}

/*
Function:	waitUntilReady
Purpose:	wait for the segment at the start position and a few after it, so playback doesn't
stall straight away. Only the first is required, a slow machine gets what exists
Params:
  - tc: the transcode
  - start: the position (seconds) the player will start from
*/
func waitUntilReady(tc *transcode, start float64) error {
	count := segmentCount(tc.manifest.Duration)
	first := min(max(int(start/segmentSecs), 0), count-1)
	deadline := time.Now().Add(readyTimeout)
	for seg := first; seg < min(first+readySegments, count); seg++ {
		if err := tc.waitForSegment(seg, deadline); err != nil && seg == first {
			return err
		} else if err != nil {
			break
		}
	}
	return nil
}

/*
Function:	languagePreferences
Purpose:	the user's preferred audio and subtitle languages, falling back to Japanese/English
Params:
  - userID: the current user
*/
func (h *Handler) languagePreferences(userID string) languagePreferences {
	prefs := languagePreferences{Audio: defaultAudioLanguage, Subtitle: defaultSubtitleLanguage}
	if err := h.db.QueryRow(
		`SELECT preferred_audio_language, preferred_subtitle_language FROM users WHERE id = $1`,
		userID,
	).Scan(&prefs.Audio, &prefs.Subtitle); err != nil {
		logger.Warn("could not load language preferences for %s: %v", userID, err)
	}
	return prefs
}

/*
Function:	hasAudioStream
Purpose:	whether index is an audio stream in the file
Params:
  - streams: the file's streams from ffprobe
  - index: the requested stream index
*/
func hasAudioStream(streams []probeStream, index int) bool {
	for _, s := range streams {
		if s.Index == index && s.CodecType == "audio" {
			return true
		}
	}
	return false
}

/*
Function:	writeStreamURLs
Purpose:	respond with the playlist URL, the audio tracks to choose from, and the subtitle
tracks, all signed with a media token for this episode/movie
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
  - mediaType: episode or movie
  - mediaID: the id of the episode or movie
  - variant: the transcode folder suffix for the chosen audio and subtitle, e.g. a2-b5
  - manifest: what the transcode contains
  - audio: every audio track in the file
  - prefs: the user's language preferences, used to pick the default subtitle
*/
func writeStreamURLs(w http.ResponseWriter, r *http.Request, mediaType, mediaID, variant string,
	manifest streamManifest, audio []audioTrack, prefs languagePreferences) {
	token, err := auth.GenerateMediaToken(auth.GetUser(r).UserID, auth.MediaScope{MediaType: mediaType, MediaID: mediaID})
	if utils.InternalError(w, err) {
		return
	}
	query := "?" + auth.MediaTokenParam + "=" + url.QueryEscape(token)
	base := "/stream/segments/" + mediaType + "/" + mediaID + "/" + variant

	resp := streamResponse{
		Playlist:  base + "/" + playlistFile + query,
		Audio:     []audioResponse{},
		Subtitles: []subtitleResponse{},
		Burned:    manifest.BurnIndex,
	}
	for _, a := range audio {
		resp.Audio = append(resp.Audio, audioResponse{
			Index:    a.Index,
			Language: a.Language,
			Label:    a.Label,
			Selected: a.Index == manifest.AudioIndex,
		})
	}

	defaultIndex := defaultSubtitle(manifest.Subtitles, manifest.AudioLanguage, prefs.Subtitle)
	for _, s := range manifest.Subtitles {
		sub := subtitleResponse{
			Index:    s.Index,
			Burn:     s.Image,
			Language: s.Language,
			Label:    s.Label,
			Default:  s.Index == defaultIndex,
		}
		if !s.Image {
			sub.URL = base + "/" + s.File + query
		}
		if s.AssFile != "" {
			sub.AssURL = base + "/" + s.AssFile + query
		}
		resp.Subtitles = append(resp.Subtitles, sub)
	}
	resp.Fonts = []string{}
	for _, f := range manifest.Fonts {
		resp.Fonts = append(resp.Fonts, base+"/"+f.File+query)
	}

	utils.JSON(w, resp)
}

/*
Function:	SignTrackURL
Purpose:	return the stream URL for a music track signed with a media token
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) SignTrackURL(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	var exists bool
	err := h.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM music_metadata WHERE media_item_id = $1 AND file_path IS NOT NULL)`,
		id,
	).Scan(&exists)
	if utils.InternalError(w, err) {
		return
	}
	if !exists {
		utils.Error(w, http.StatusNotFound, "track not found")
		return
	}

	token, err := auth.GenerateMediaToken(auth.GetUser(r).UserID, auth.MediaScope{MediaType: "music", MediaID: id})
	if utils.InternalError(w, err) {
		return
	}
	query := "?" + auth.MediaTokenParam + "=" + url.QueryEscape(token)

	utils.JSON(w, map[string]string{
		"url":   "/stream/music/" + id + query,
		"cover": "/stream/music/" + id + "/cover" + query,
	})
}

/*
Function:	signPlaylist
Purpose:	append the media token to every segment URI so HLS.js relative fetches stay authorized
Params:
  - playlist: contents of the m3u8 written by ffmpeg
  - token: the media token the playlist was requested with
*/
func signPlaylist(playlist, token string) string {
	param := auth.MediaTokenParam + "=" + url.QueryEscape(token)
	lines := strings.Split(playlist, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		sep := "?"
		if strings.Contains(trimmed, "?") {
			sep = "&"
		}
		lines[i] = trimmed + sep + param
	}
	return strings.Join(lines, "\n")
}

/*
Function: ServeSegment
Purpose: serve the file the frontend will display
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) ServeSegment(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "type")
	mediaID := chi.URLParam(r, "id")
	variant := chi.URLParam(r, "variant")
	file := chi.URLParam(r, "file")

	// only playlist, segments and finished subtitles; not stream.json, transcode.log, ffmpeg's
	// own playlist or .tmp files.
	// Types are set here because the system MIME table (Debian's) calls .ts a text file, and the
	// proxy then compresses every segment
	contentTypes := map[string]string{
		".m3u8":  "application/vnd.apple.mpegurl",
		".ts":    "video/mp2t",
		".vtt":   "text/vtt; charset=utf-8",
		".ass":   "text/plain; charset=utf-8",
		".ttf":   "font/ttf",
		".otf":   "font/otf",
		".ttc":   "font/collection",
		".woff":  "font/woff",
		".woff2": "font/woff2",
	}
	contentType, servable := contentTypes[filepath.Ext(file)]
	if (mediaType != "episode" && mediaType != "movie") || !utils.SafeFileName(mediaID) ||
		!variantRe.MatchString(variant) || !utils.SafeFileName(file) || !servable ||
		(filepath.Ext(file) == ".m3u8" && file != playlistFile) {
		utils.Error(w, http.StatusBadRequest, "invalid segment path")
		return
	}

	mediaRoot := os.Getenv("MEDIA_ROOT")
	tempDir := filepath.Join(mediaRoot, "temp", mediaType+"-"+mediaID+"-"+variant)
	filePath := filepath.Join(tempDir, file)
	if !utils.IsWithin(tempDir, filePath) {
		utils.Error(w, http.StatusBadRequest, "invalid segment path")
		return
	}

	// signed with the token it was requested with
	if filepath.Ext(file) == ".m3u8" {
		content, err := os.ReadFile(filePath)
		if os.IsNotExist(err) {
			utils.Error(w, http.StatusNotFound, "playlist not found")
			return
		}
		if utils.InternalError(w, err) {
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte(signPlaylist(string(content), r.URL.Query().Get(auth.MediaTokenParam))))
		return
	}

	// the playlist lists every segment, so the player can ask for one ffmpeg hasn't written
	if match := segmentNameRe.FindStringSubmatch(file); match != nil && !fileExists(filePath) {
		tc, ok := h.loadTranscode(mediaType+"-"+mediaID+"-"+variant, tempDir)
		if !ok {
			utils.Error(w, http.StatusNotFound, "segment not found")
			return
		}
		seg, _ := strconv.Atoi(match[1])
		if err := tc.waitForSegment(seg, time.Now().Add(segmentTimeout)); err != nil {
			logger.Warn("segment %s of %s not ready: %v", file, tempDir, err)
			utils.Error(w, http.StatusNotFound, "segment not ready")
			return
		}
	}

	// ServeFile keeps a Content-Type that is already set
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, filePath)
}

func (h *Handler) ServeTrack(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	var filePath string
	err := h.db.QueryRow(
		`SELECT file_path FROM music_metadata WHERE media_item_id = $1`,
		id,
	).Scan(&filePath)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "track not found")
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	http.ServeFile(w, r, cache.Disk().Local(filePath))
}

func (h *Handler) ServeCover(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	var filePath string
	err := h.db.QueryRow(
		`SELECT file_path FROM music_metadata WHERE media_item_id = $1`,
		id,
	).Scan(&filePath)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "track not found")
		return
	}

	f, err := os.Open(cache.Disk().Local(musicfiles.CoverPath(filePath)))
	if os.IsNotExist(err) {
		utils.Error(w, http.StatusNotFound, "cover not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if utils.InternalError(w, err) {
		return
	}

	// an uploaded cover can be a PNG or WebP under the cover.jpg name
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	contentType := musicfiles.ImageType(head[:n])
	if contentType == "" {
		contentType = "image/jpeg"
	}
	if _, err := f.Seek(0, io.SeekStart); utils.InternalError(w, err) {
		return
	}

	// an admin can replace the cover behind the same URL, so check it's unchanged before reuse
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, "", info.ModTime(), f)
}
