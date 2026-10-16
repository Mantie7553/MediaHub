package stream

import (
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"sync"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db *sql.DB
	procs    map[string]*exec.Cmd
    procsMu  sync.Mutex
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	StreamMedia
Purpose:	Start transcoding of an episode
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

	mediaRoot := os.Getenv("MEDIA_ROOT")
	tempDir := filepath.Join(mediaRoot, "temp", mediaType+"-"+mediaID)

	if _, err := os.Stat(tempDir); err == nil {
		writeStreamURLs(w, r, mediaType, mediaID)
		return
	}

	if err := os.Mkdir(tempDir, 0755); err != nil {
		utils.InternalError(w, err)
		return
	}

	playlistPath := filepath.Join(tempDir, "playlist.m3u8")

	cmd := exec.Command("ffmpeg",
		"-i", filePath,
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-vf", "format=yuv420p",
		"-c:v", "h264_amf",
		"-c:a", "aac",
		"-b:a", "192k",
		"-hls_time", "10",
		"-hls_list_size", "0",
		"-f", "hls",
		playlistPath,
	)

	if err := cmd.Start(); err != nil {
		utils.InternalError(w, err)
		return
	}
	go cmd.Wait()

	subsPath := filepath.Join(tempDir, "subs.vtt")
	subCmd := exec.Command("ffmpeg",
		"-i", filePath,
		"-map", "0:s:0",
		"-c:s", "webvtt",
		subsPath,
	)

	subCmd.Start()
	go subCmd.Wait()

	for i := 0; i < 30; i++ {
		if _, err := os.Stat(playlistPath); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if _, err := os.Stat(playlistPath); err != nil {
		utils.Error(w, http.StatusInternalServerError, "transcode failed to start")
		return
	}

	writeStreamURLs(w, r, mediaType, mediaID)
}

/*
Function:	writeStreamURLs
Purpose:	respond with playlist and subtitle URLs signed with a media token for this episode/movie
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
  - mediaType: episode or movie
  - mediaID: the id of the episode or movie
*/
func writeStreamURLs(w http.ResponseWriter, r *http.Request, mediaType, mediaID string) {
	token, err := auth.GenerateMediaToken(auth.GetUser(r).UserID, auth.MediaScope{MediaType: mediaType, MediaID: mediaID})
	if utils.InternalError(w, err) {
		return
	}
	query := "?" + auth.MediaTokenParam + "=" + url.QueryEscape(token)
	base := "/stream/segments/" + mediaType + "/" + mediaID

	utils.JSON(w, map[string]string{
		"playlist":  base + "/playlist.m3u8" + query,
		"subtitles": base + "/subs.vtt" + query,
	})
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
	file := chi.URLParam(r, "file")

	mediaRoot := os.Getenv("MEDIA_ROOT")
	filePath := filepath.Join(mediaRoot, "temp", mediaType+"-"+mediaID, file)

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
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte(signPlaylist(string(content), r.URL.Query().Get(auth.MediaTokenParam))))
		return
	}

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

	coverPath := filepath.Join(filepath.Dir(filePath), "cover.jpg")
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		utils.Error(w, http.StatusNotFound, "cover not found")
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, coverPath)
}
