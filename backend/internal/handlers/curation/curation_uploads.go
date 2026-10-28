package curation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/mangafiles"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/novelfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/Mantie7553/MediaHub/backend/internal/uploads"
	"github.com/go-chi/chi/v5"
)

/*
Adding files: the browser sends a file to the uploads store in chunks, then names the finished
upload here with the details (episode number, track title, ...). The file is checked, moved to
where the library keeps that kind of file, and recorded. When anything is refused the upload
is handed back, so the admin can correct the details without sending the file again
*/

/*
Function:	AddEpisode
Purpose:	Admin: add an episode file to an anime, saved as
"TV Shows/<Series>/Season 01/<Series> - S01E05 - Title.mkv" beside the series' other
episodes, the way Sonarr names them
Params:
  - w: http response writer
  - r: JSON addEpisodeRequest
*/
func (h *Handler) AddEpisode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req addEpisodeRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if req.SeasonNumber < 0 || req.SeasonNumber > 1000 || req.EpisodeNumber < 0 || req.EpisodeNumber > maxCount {
		utils.Error(w, http.StatusBadRequest, "enter the season and episode numbers")
		return
	}
	title, ok := cleanText(&req.Title, maxTitleRunes)
	if !ok {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("the title can be at most %d characters", maxTitleRunes))
		return
	}

	series, ok := h.titleOf(w, id, "anime")
	if !ok {
		return
	}
	file, ok := h.claim(w, r, req.UploadID, uploads.Video)
	if !ok {
		return
	}
	added := false
	defer h.finish(file, &added)

	if respondFileError(w, checkVideo(file.Path)) {
		return
	}
	var taken bool
	err := h.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM episodes WHERE media_item_id = $1 AND season_number = $2
		AND episode_number = $3 AND file_path IS NOT NULL AND file_path <> '')`,
		id, req.SeasonNumber, req.EpisodeNumber,
	).Scan(&taken)
	if utils.InternalError(w, err) {
		return
	}
	if taken {
		utils.Error(w, http.StatusConflict, fmt.Sprintf("S%02dE%02d already has a file", req.SeasonNumber, req.EpisodeNumber))
		return
	}

	tvRoot := filepath.Join(os.Getenv("MEDIA_ROOT"), "TV Shows")
	existing, err := h.strings(`SELECT file_path FROM episodes WHERE media_item_id = $1 AND file_path IS NOT NULL AND file_path <> ''
		ORDER BY season_number = $2 DESC, season_number, episode_number`, id, req.SeasonNumber)
	if utils.InternalError(w, err) {
		return
	}
	seriesDir := seriesFolder(tvRoot, series, existing)
	dest := filepath.Join(
		seasonFolder(seriesDir, req.SeasonNumber, existing),
		episodeFileName(filepath.Base(seriesDir), req.SeasonNumber, req.EpisodeNumber, deref(title), file.Ext()),
	)
	if !h.place(w, file, tvRoot, dest) {
		return
	}

	_, err = h.db.Exec(
		`INSERT INTO episodes (media_item_id, season_number, episode_number, title, title_locked, file_path)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (media_item_id, season_number, episode_number) DO UPDATE SET
		file_path = EXCLUDED.file_path,
		title = COALESCE(EXCLUDED.title, episodes.title),
		title_locked = episodes.title_locked OR EXCLUDED.title_locked,
		updated_at = NOW()`,
		id, req.SeasonNumber, req.EpisodeNumber, title, title != nil, dest,
	)
	if h.unplaceOnError(w, file, dest, err) {
		return
	}
	added = true
	logger.Info("user %s added %s to %s", auth.GetUser(r).UserID, filepath.Base(dest), series)
	w.WriteHeader(http.StatusCreated)
}

/*
Function:	AddMovieFile
Purpose:	Admin: give a movie that has no file one, saved as "Movies/<Title (Year)>/<Title (Year)>.mkv"
Params:
  - w: http response writer
  - r: JSON { upload_id }
*/
func (h *Handler) AddMovieFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req addMovieFileRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	title, ok := h.titleOf(w, id, "movie")
	if !ok {
		return
	}
	var year sql.NullString
	var current sql.NullString
	err := h.db.QueryRow(
		`SELECT to_char(mi.release_date, 'YYYY'), mm.file_path FROM media_items mi
		LEFT JOIN movie_metadata mm ON mm.media_item_id = mi.id WHERE mi.id = $1`, id,
	).Scan(&year, &current)
	if utils.InternalError(w, err) {
		return
	}
	if current.Valid && current.String != "" {
		utils.Error(w, http.StatusConflict, "this movie already has a file")
		return
	}

	file, ok := h.claim(w, r, req.UploadID, uploads.Video)
	if !ok {
		return
	}
	added := false
	defer h.finish(file, &added)

	if respondFileError(w, checkVideo(file.Path)) {
		return
	}
	moviesRoot := filepath.Join(os.Getenv("MEDIA_ROOT"), "Movies")
	name := mangafiles.SanitizeFileName(title)
	if year.Valid && year.String != "" && year.String != "0001" {
		name += " (" + year.String + ")"
	}
	dest := filepath.Join(moviesRoot, name, name+file.Ext())
	if !h.place(w, file, moviesRoot, dest) {
		return
	}

	_, err = h.db.Exec(
		`INSERT INTO movie_metadata (media_item_id, file_path) VALUES ($1, $2)
		ON CONFLICT (media_item_id) DO UPDATE SET file_path = EXCLUDED.file_path`,
		id, dest,
	)
	if h.unplaceOnError(w, file, dest, err) {
		return
	}
	added = true
	logger.Info("user %s added a file to %s", auth.GetUser(r).UserID, title)
	w.WriteHeader(http.StatusCreated)
}

/*
Function:	AddVolume
Purpose:	Admin: add an EPUB volume to a light novel, saved as
"Light Novels/<Series>/<Series> Volume 3.epub" beside its other volumes. Volume 1 gives a
light novel without a cover the EPUB's cover
Params:
  - w: http response writer
  - r: JSON addVolumeRequest
*/
func (h *Handler) AddVolume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req addVolumeRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if req.VolumeNumber < 0 || req.VolumeNumber > maxCount {
		utils.Error(w, http.StatusBadRequest, "enter the volume number")
		return
	}
	title, ok := cleanText(&req.Title, maxTitleRunes)
	if !ok {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("the title can be at most %d characters", maxTitleRunes))
		return
	}
	series, ok := h.titleOf(w, id, "light_novel")
	if !ok {
		return
	}
	var taken bool
	err := h.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM light_novel_volumes WHERE media_item_id = $1 AND volume_number = $2)`,
		id, req.VolumeNumber,
	).Scan(&taken)
	if utils.InternalError(w, err) {
		return
	}
	if taken {
		utils.Error(w, http.StatusConflict, fmt.Sprintf("volume %d is already in the library", req.VolumeNumber))
		return
	}

	file, ok := h.claim(w, r, req.UploadID, uploads.EPUB)
	if !ok {
		return
	}
	added := false
	defer h.finish(file, &added)

	if err := novelfiles.Check(file.Path); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	lnRoot := filepath.Join(os.Getenv("MEDIA_ROOT"), "Light Novels")
	existing, err := h.strings(`SELECT file_path FROM light_novel_volumes WHERE media_item_id = $1 AND file_path IS NOT NULL AND file_path <> ''`, id)
	if utils.InternalError(w, err) {
		return
	}
	folder := filepath.Join(lnRoot, mangafiles.SanitizeFileName(series))
	if dir, ok := sharedFolder(lnRoot, existing); ok {
		folder = dir
	}
	dest := filepath.Join(folder, novelfiles.FileName(filepath.Base(folder), req.VolumeNumber))
	if !h.place(w, file, lnRoot, dest) {
		return
	}

	var volumeID string
	err = h.db.QueryRow(
		`INSERT INTO light_novel_volumes (media_item_id, volume_number, title, file_path)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		id, req.VolumeNumber, title, dest,
	).Scan(&volumeID)
	if h.unplaceOnError(w, file, dest, err) {
		return
	}
	added = true

	if req.VolumeNumber == 1 {
		if cover, err := novelfiles.CoverHref(dest); err == nil && cover != "" {
			h.db.Exec(
				`UPDATE media_items SET cover_image_url = $1 WHERE id = $2 AND cover_image_url IS NULL`,
				fmt.Sprintf("/api/light-novels/%s/volumes/%s/images/%s", id, volumeID, cover), id,
			)
		}
	}
	logger.Info("user %s added %s to %s", auth.GetUser(r).UserID, filepath.Base(dest), series)
	utils.JSON(w, map[string]string{"id": volumeID}, http.StatusCreated)
}

/*
Function:	AddTrack
Purpose:	Admin: add an mp3 to an album, saved as "Music/<Artist>/<Album>/<Title>.mp3"
beside its other tracks. The folder's cover.jpg becomes its cover, or the track's own art
when the folder has none
Params:
  - w: http response writer
  - r: JSON addTrackRequest
*/
func (h *Handler) AddTrack(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	var req addTrackRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	title, ok := cleanTitle(&req.Title)
	if !ok {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("the title must be 1 to %d characters", maxTitleRunes))
		return
	}
	if req.TrackNumber != nil && (*req.TrackNumber < 0 || *req.TrackNumber > maxCount) {
		utils.Error(w, http.StatusBadRequest, "the track number must be a whole number")
		return
	}
	fileName := musicfiles.CleanName(title)
	if fileName == "" {
		utils.Error(w, http.StatusBadRequest, "the title needs letters or numbers to name the file")
		return
	}

	var albumTitle, artist string
	err := h.db.QueryRow(`SELECT title, artist FROM albums WHERE id = $1`, albumID).Scan(&albumTitle, &artist)
	if errors.Is(err, sql.ErrNoRows) || !utils.IsUUID(albumID) {
		utils.Error(w, http.StatusNotFound, "album not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	file, ok := h.claim(w, r, req.UploadID, uploads.Audio)
	if !ok {
		return
	}
	added := false
	defer h.finish(file, &added)

	duration, err := checkMP3(file.Path)
	if respondFileError(w, err) {
		return
	}

	musicRoot := filepath.Join(os.Getenv("MEDIA_ROOT"), "Music")
	existing, err := h.strings(`SELECT file_path FROM music_metadata WHERE album_id = $1 AND file_path IS NOT NULL AND file_path <> ''`, albumID)
	if utils.InternalError(w, err) {
		return
	}
	folder := filepath.Join(musicRoot, orDefault(musicfiles.CleanName(artist), "Unknown Artist"), orDefault(musicfiles.CleanName(albumTitle), "Unknown Album"))
	if dir, ok := sharedFolder(musicRoot, existing); ok {
		folder = dir
	}
	dest := filepath.Join(folder, fileName+".mp3")
	if !h.place(w, file, musicRoot, dest) {
		return
	}

	tx, err := h.db.Begin()
	if h.unplaceOnError(w, file, dest, err) {
		return
	}
	defer tx.Rollback()
	var trackID string
	err = tx.QueryRow(`INSERT INTO media_items (type, title, external_source) VALUES ('music_track', $1, 'manual') RETURNING id`, title).Scan(&trackID)
	if err == nil {
		_, err = tx.Exec(
			`INSERT INTO music_metadata (media_item_id, artist, album_id, track_number, duration_secs, file_path)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			trackID, artist, albumID, req.TrackNumber, duration, dest,
		)
	}
	if err == nil {
		err = tx.Commit()
	}
	if h.unplaceOnError(w, file, dest, err) {
		return
	}
	added = true

	coverPath := musicfiles.CoverPath(dest)
	if _, err := os.Stat(coverPath); err == nil || musicfiles.ExtractCover(dest, coverPath) {
		coverURL := musicfiles.CoverURL(trackID)
		h.db.Exec(`UPDATE media_items SET cover_image_url = $1 WHERE id = $2`, coverURL, trackID)
		h.db.Exec(`UPDATE albums SET cover_image_url = $1 WHERE id = $2 AND cover_image_url IS NULL`, coverURL, albumID)
	}
	logger.Info("user %s added %s to %s", auth.GetUser(r).UserID, filepath.Base(dest), albumTitle)
	utils.JSON(w, map[string]string{"id": trackID}, http.StatusCreated)
}

/*
Function:	titleOf
Purpose:	a media item's title, checking it is of the type expected. Responds and returns
false when it isn't there or is another type
*/
func (h *Handler) titleOf(w http.ResponseWriter, id, mediaType string) (string, bool) {
	if !utils.IsUUID(id) {
		utils.Error(w, http.StatusNotFound, "not found")
		return "", false
	}
	var title, actual string
	err := h.db.QueryRow(`SELECT title, type FROM media_items WHERE id = $1`, id).Scan(&title, &actual)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "not found")
		return "", false
	}
	if utils.InternalError(w, err) {
		return "", false
	}
	if actual != mediaType {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("this is %s, not %s", withArticle(actual), withArticle(mediaType)))
		return "", false
	}
	return title, true
}

// claim takes a finished upload, answering with why it can't be used
func (h *Handler) claim(w http.ResponseWriter, r *http.Request, id string, kind uploads.Kind) (uploads.File, bool) {
	file, err := h.uploads.Claim(id, auth.GetUser(r).UserID, kind)
	switch {
	case err == nil:
		return file, true
	case errors.Is(err, uploads.ErrNotFound):
		utils.Error(w, http.StatusNotFound, err.Error())
	default:
		utils.Error(w, http.StatusConflict, err.Error())
	}
	return uploads.File{}, false
}

// finish forgets an upload once it is in the library, or hands it back for another try
func (h *Handler) finish(file uploads.File, added *bool) {
	if *added {
		h.uploads.Done(file.ID)
	} else {
		h.uploads.Release(file.ID)
	}
}

/*
Function:	place
Purpose:	move an upload to dest inside root, never replacing a file. Responds with the
problem and returns false when it can't
*/
func (h *Handler) place(w http.ResponseWriter, file uploads.File, root, dest string) bool {
	if os.Getenv("MEDIA_ROOT") == "" || !utils.IsWithin(root, dest) {
		utils.Error(w, http.StatusInternalServerError, "MEDIA_ROOT is not set, so the file can't be saved")
		return false
	}
	if _, err := os.Stat(dest); err == nil {
		utils.Error(w, http.StatusConflict, "there's already a file named "+filepath.Base(dest)+" in "+filepath.Base(filepath.Dir(dest)))
		return false
	}
	if err := uploads.Move(file.Path, dest); err != nil {
		logger.Error("moving upload %s to %s: %v", file.ID, dest, err)
		utils.Error(w, http.StatusInternalServerError, "couldn't save the file in the library")
		return false
	}
	return true
}

/*
Function:	unplaceOnError
Purpose:	after a database error, move the file back to the upload so the admin can try
again, and respond. Returns false when there was no error
*/
func (h *Handler) unplaceOnError(w http.ResponseWriter, file uploads.File, dest string, err error) bool {
	if err == nil {
		return false
	}
	if moveErr := os.Rename(dest, file.Path); moveErr != nil {
		logger.Error("moving %s back to its upload: %v", dest, moveErr)
		os.Remove(dest)
	}
	if utils.IsUniqueViolation(err) {
		utils.Error(w, http.StatusConflict, "that's already in the library")
		return true
	}
	return utils.InternalError(w, err)
}

// strings runs a query that returns one text column
func (h *Handler) strings(query string, args ...any) ([]string, error) {
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// a season's folder, as Sonarr names them: "Season 01", or "Specials" for season 0
var seasonDirRe = regexp.MustCompile(`(?i)^(season\s*\d+|specials)$`)

/*
Function:	seriesFolder
Purpose:	the folder a series keeps its episodes in: the one its episodes are already in
(above their season folders), or one named after the title under the TV Shows folder
Params:
  - tvRoot: MEDIA_ROOT/TV Shows
  - title: the series title
  - existing: its episodes' files
*/
func seriesFolder(tvRoot, title string, existing []string) string {
	for _, p := range existing {
		dir := filepath.Dir(p)
		if seasonDirRe.MatchString(filepath.Base(dir)) {
			dir = filepath.Dir(dir)
		}
		if utils.IsWithin(tvRoot, dir) {
			return dir
		}
	}
	return filepath.Join(tvRoot, mangafiles.SanitizeFileName(title))
}

/*
Function:	seasonFolder
Purpose:	the folder for a season's episodes: the one the season's episodes already use
(existing is sorted with that season first), otherwise "Season 01" ("Specials" for 0)
Params:
  - seriesDir: from seriesFolder
  - season: the season number
  - existing: the series' episode files, the season's own first
*/
func seasonFolder(seriesDir string, season int, existing []string) string {
	want := fmt.Sprintf("Season %02d", season)
	if season == 0 {
		want = "Specials"
	}
	for _, p := range existing {
		dir := filepath.Dir(p)
		if filepath.Dir(dir) != seriesDir || !seasonDirRe.MatchString(filepath.Base(dir)) {
			continue
		}
		if n, ok := seasonNumber(filepath.Base(dir)); ok && n == season {
			return dir
		}
	}
	return filepath.Join(seriesDir, want)
}

func seasonNumber(folder string) (int, bool) {
	if strings.EqualFold(folder, "specials") {
		return 0, true
	}
	n, err := strconv.Atoi(strings.TrimSpace(folder[len("season"):]))
	return n, err == nil
}

/*
Function:	episodeFileName
Purpose:	"Series - S01E05 - Title.mkv", the way Sonarr names episodes, so it can recognise
the file
*/
func episodeFileName(series string, season, episode int, title, ext string) string {
	name := fmt.Sprintf("%s - S%02dE%02d", series, season, episode)
	if title != "" {
		name += " - " + title
	}
	return mangafiles.SanitizeFileName(name) + ext
}

/*
Function:	sharedFolder
Purpose:	the folder a title's files are in, when they are inside root
*/
func sharedFolder(root string, existing []string) (string, bool) {
	for _, p := range existing {
		if dir := filepath.Dir(p); utils.IsWithin(root, dir) {
			return dir, true
		}
	}
	return "", false
}

// a problem with an uploaded file, shown to the admin as a 400
type fileError string

func (e fileError) Error() string { return string(e) }

func respondFileError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var fe fileError
	if errors.As(err, &fe) {
		utils.Error(w, http.StatusBadRequest, string(fe))
		return true
	}
	logger.Error("checking an uploaded file: %v", err)
	utils.Error(w, http.StatusInternalServerError, "couldn't check the file")
	return true
}

// the formats ffprobe may read an upload as: what the allowed video extensions and mp3 are
const uploadFormats = "matroska,mov,avi,mpegts,mp3"

type probeResult struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
	} `json:"format"`
}

// probe asks ffprobe what a file holds. A file ffprobe can't read is a fileError
func probe(path string) (probeResult, error) {
	var res probeResult
	out, err := exec.Command("ffprobe", "-v", "error",
		// an upload is read as a local file in one of the formats allowed for it and nothing
		// else, so a playlist or reference posing as a video can't reach other files or hosts
		"-protocol_whitelist", "file", "-format_whitelist", uploadFormats,
		"-show_entries", "stream=codec_type:format=format_name,duration", "-of", "json", path).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return res, fileError("the file couldn't be read as audio or video, it may be damaged")
		}
		return res, err
	}
	return res, json.Unmarshal(out, &res)
}

// checkVideo makes sure a file has a video stream the player can transcode
func checkVideo(path string) error {
	res, err := probe(path)
	if err != nil {
		return err
	}
	for _, s := range res.Streams {
		if s.CodecType == "video" {
			return nil
		}
	}
	return fileError("the file has no video in it")
}

// checkMP3 makes sure a file is an mp3 and returns its length in seconds
func checkMP3(path string) (*int, error) {
	res, err := probe(path)
	if err != nil {
		return nil, err
	}
	if res.Format.FormatName != "mp3" {
		return nil, fileError("the file isn't an mp3")
	}
	if secs, err := strconv.ParseFloat(res.Format.Duration, 64); err == nil && secs > 0 {
		n := int(secs + 0.5)
		return &n, nil
	}
	return nil, nil
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
