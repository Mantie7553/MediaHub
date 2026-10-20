package stream

import (
	"database/sql"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db *sql.DB

	mu         sync.Mutex
	transcodes map[string]*transcode // running transcodes by type-id-a<audio>
	probes     probeCache
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db, transcodes: map[string]*transcode{}}
}

// one transcode folder per media and audio track, e.g. a2
var variantRe = regexp.MustCompile(`^a(-1|\d+)$`)

/*
Function:	StreamMedia
Purpose:	Start (or reuse) the transcode of an episode or movie and return signed playlist and
            subtitle URLs once enough of it exists to play from the optional ?start= position.
            ?audio= picks an audio stream, otherwise the user's preferred language is used
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) StreamMedia(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "type")
	mediaID := chi.URLParam(r, "id")
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
		return
	}

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "episode not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	start, _ := strconv.ParseFloat(r.URL.Query().Get("start"), 64)
	if start < 0 {
		start = 0
	}

	streams, err := h.probes.get(filePath)
	if utils.InternalError(w, err) {
		return
	}
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

	variant := "a" + strconv.Itoa(audioIndex)
	key := mediaType + "-" + mediaID + "-" + variant
	tempDir := filepath.Join(os.Getenv("MEDIA_ROOT"), "temp", key)

	tc, err := h.getOrStartTranscode(key, tempDir, filePath, buildManifest(streams, audioIndex))
	if utils.InternalError(w, err) {
		return
	}

	if err := waitForPlaylist(tc, start); err != nil {
		logger.Error("transcode %s not ready: %v", key, err)
		utils.Error(w, http.StatusInternalServerError, "transcode failed to start")
		return
	}

	writeStreamURLs(w, r, mediaType, mediaID, variant, tc.manifest, audioTracks(streams), prefs)
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
  - variant: the transcode folder suffix for the chosen audio, e.g. a2
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
		resp.Subtitles = append(resp.Subtitles, subtitleResponse{
			URL:      base + "/" + s.File + query,
			Language: s.Language,
			Label:    s.Label,
			Default:  s.Index == defaultIndex,
		})
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
	id := chi.URLParam(r, "id")

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

	// only playlist, segments and finished subtitles; not stream.json, transcode.log or .tmp files.
	// Types are set here because the system MIME table (Debian's) calls .ts a text file, and the
	// proxy then compresses every segment
	contentTypes := map[string]string{
		".m3u8": "application/vnd.apple.mpegurl",
		".ts":   "video/mp2t",
		".vtt":  "text/vtt; charset=utf-8",
	}
	contentType, servable := contentTypes[filepath.Ext(file)]
	if (mediaType != "episode" && mediaType != "movie") || !utils.SafeFileName(mediaID) ||
		!variantRe.MatchString(variant) || !utils.SafeFileName(file) || !servable {
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

	// the playlist grows while ffmpeg runs, so rewrite it on every request
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

	// ServeFile keeps a Content-Type that is already set
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, filePath)
}

func (h *Handler) ServeTrack(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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
	http.ServeFile(w, r, filePath)
}

func (h *Handler) ServeCover(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var filePath string
	err := h.db.QueryRow(
		`SELECT file_path FROM music_metadata WHERE media_item_id = $1`,
		id,
	).Scan(&filePath)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "track not found")
		return
	}

	f, err := os.Open(musicfiles.CoverPath(filePath))
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
