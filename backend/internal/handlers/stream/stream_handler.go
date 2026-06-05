package stream

import (
	"database/sql"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db *sql.DB
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
		utils.JSON(w, map[string]string{
			"playlist":  "/stream/segments/" + mediaType + "/" + mediaID + "/playlist.m3u8",
			"subtitles": "/stream/segments/" + mediaType + "/" + mediaID + "/subs.vtt",
		})
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

	utils.JSON(w, map[string]string{
		"playlist":  "/stream/segments/" + mediaType + "/" + mediaID + "/playlist.m3u8",
		"subtitles": "/stream/segments/" + mediaType + "/" + mediaID + "/subs.vtt",
	})
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
