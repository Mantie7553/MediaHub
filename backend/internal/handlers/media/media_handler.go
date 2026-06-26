package media

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	Upload
Purpose:	add a new media item
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// check that the type and title were actually provided
	if req.Type == "" || req.Title == "" {
		utils.Error(w, http.StatusBadRequest, "type and title are required")
		return
	}

	// if adding music make sure an artist name was provided
	if req.Type == "music_track" && req.Artist == "" {
		utils.Error(w, http.StatusBadRequest, "artist is required for music_track")
		return
	}

	// confirm the release date is valid
	var releaseDate *time.Time
	if req.ReleaseDate != "" {
		t, err := time.Parse("2006-01-02", req.ReleaseDate)
		if err != nil {
			utils.Error(w, http.StatusBadRequest, "invalid release_date format, use YYYY-MM-DD")
			return
		}
		releaseDate = &t
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	// add the new item to the database
	var mediaID string
	err = tx.QueryRow(
		`INSERT INTO media_items (type, title, description, cover_image_url, release_date, external_id, external_source)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		req.Type,
		req.Title,
		utils.NullString(req.Description),
		utils.NullString(req.CoverImageURL),
		releaseDate,
		utils.NullString(req.ExternalID),
		utils.NullString(req.ExternalSource),
	).Scan(&mediaID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			utils.Error(w, http.StatusConflict, "media item already exists")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// add in the specific types metadata
	switch req.Type {
	case "anime":
		_, err = tx.Exec(
			`INSERT INTO anime_metadata (media_item_id, studio, status, genres)
			 VALUES ($1, $2, $3, $4)`,
			mediaID,
			utils.NullString(req.Studio),
			utils.NullString(req.Status),
			pq.Array(req.Genres),
		)
	case "movie":
		_, err = tx.Exec(
			`INSERT INTO movie_metadata (media_item_id, runtime_mins, director, genres)
			 VALUES ($1, $2, $3, $4)`,
			mediaID,
			utils.NullInt(req.RuntimeMins),
			utils.NullString(req.Director),
			pq.Array(req.Genres),
		)
	case "music_track":
		_, err = tx.Exec(
			`INSERT INTO music_metadata (media_item_id, artist, track_number, duration_secs, genres)
			 VALUES ($1, $2, $3, $4, $5)`,
			mediaID,
			req.Artist,
			utils.NullInt(req.TrackNumber),
			utils.NullInt(req.DurationSec),
			pq.Array(req.Genres),
		)
	case "manga":
		var (
			status        string
			genres        []string
			totalChapters int
		)

		if req.ExternalID != "" {
			client := mangadex.NewMangaDexClient("")
			manga, err := client.GetByID(req.ExternalID)
			if err != nil {
				logger.Warn("failed to fetch mangadex metadata for %s: %v", req.ExternalID, err)
			} else {
				status = manga.Attributes.Status
				for _, tag := range manga.Attributes.Tags {
					if tag.Attributes.Group == "genre" {
						genres = append(genres, tag.Attributes.Name.En)
					}
				}
				if n, err := strconv.Atoi(manga.Attributes.LastChapter); err == nil {
					totalChapters = n
				}
			}
		}

		_, err = tx.Exec(
			`INSERT INTO manga_metadata (media_item_id, total_chapters, genres, status)
			VALUES ($1, $2, $3, $4)`,
			mediaID,
			utils.NullInt(&totalChapters),
			pq.Array(genres),
			utils.NullString(status),
		)

	default:
		utils.Error(w, http.StatusBadRequest, "invalid media type, must be one of: anime, movie, music_track")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	if utils.InternalError(w, tx.Commit()) {
		return
	}

	// return the id of the new item
	utils.JSON(w, map[string]string{"id": mediaID}, http.StatusCreated)
}

/*
Function:	GetAll
Purpose:	Get all media items from the database
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	// get the medias type from the URL
	mediaType := r.URL.Query().Get("type")
	available := r.URL.Query().Get("available")

	conditions := []string{}
	args := []any{}
	items := []MediaItem{}

	if mediaType != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)+1))
		args = append(args, mediaType)
	}

	if available == "true" {
		conditions = append(conditions, `(EXISTS (SELECT 1 FROM sonarr_items WHERE media_item_id = mi.id)
		OR EXISTS (SELECT 1 FROM radarr_items WHERE media_item_id = mi.id)
		OR EXISTS (SELECT 1 FROM manga_chapters WHERE media_item_id = mi.id AND file_path IS NOT NULL)
		OR EXISTS (SELECT 1 FROM light_novel_volumes WHERE media_item_id = mi.id)
		OR EXISTS (SELECT 1 FROM music_metadata WHERE media_item_id = mi.id AND file_path IS NOT NULL))`)
	}

	queryString := `SELECT mi.id, mi.type, mi.title, mi.description, mi.cover_image_url, mi.release_date, mi.external_id, mi.external_source, mi.created_at, mm.artist
	FROM media_items mi
	LEFT JOIN music_metadata mm ON mm.media_item_id = mi.id`

	if len(conditions) > 0 {
		queryString += " WHERE " + strings.Join(conditions, " AND ")
	}

	rows, err := h.db.Query(queryString, args...)

	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	// map the rows to useable structs
	for rows.Next() {
		var item MediaItem
		err := rows.Scan(
			&item.ID, &item.Type, &item.Title, &item.Description,
			&item.CoverImageURL, &item.ReleaseDate, &item.ExternalID,
			&item.ExternalSource, &item.CreatedAt, &item.Artist,
		)

		if utils.InternalError(w, err) {
			return
		}
		items = append(items, item)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	// return the items
	utils.JSON(w, items)
}

/*
Function:	GetSpecific
Purpose:	Get a specific media item from the database
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetSpecific(w http.ResponseWriter, r *http.Request) {
	// get the id from the URL
	mediaId := chi.URLParam(r, "id")
	item := MediaItem{}

	queryString := `SELECT id, type, title, description, 
		cover_image_url, release_date, external_id, 
		external_source, created_at 
		FROM media_items WHERE id = $1`

	// look for the meida item
	err := h.db.QueryRow(queryString, mediaId).Scan(
		&item.ID, &item.Type, &item.Title, &item.Description,
		&item.CoverImageURL, &item.ReleaseDate, &item.ExternalID,
		&item.ExternalSource, &item.CreatedAt,
	)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	if utils.InternalError(w, err) {
		return
	}

	var metadata any

	// get the appropriate metadata for the media type
	switch item.Type {
	case "anime":
		var meta AnimeMetadata
		err := h.db.QueryRow(
			`SELECT studio, status, genres FROM anime_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.Studio, &meta.Status, pq.Array(&meta.Genres))
		if err != nil && err != sql.ErrNoRows {
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		metadata = meta
	case "movie":
		var meta MovieMetadata
		err := h.db.QueryRow(
			`SELECT runtime_mins, director, genres, file_path FROM movie_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.RuntimeMins, &meta.Director, pq.Array(&meta.Genres), &meta.FilePath)
		if err != nil && err != sql.ErrNoRows {
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		metadata = meta

	case "music_track":
		var meta MusicMetadata
		err := h.db.QueryRow(
			`SELECT artist, track_number, duration_secs, genres FROM music_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.Artist, &meta.TrackNumber, &meta.DurationSecs, pq.Array(&meta.Genres))
		if err != nil && err != sql.ErrNoRows {
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		metadata = meta
	case "manga":
		var meta MangaMetadata
		var chapters []MangaChapter
		err := h.db.QueryRow(
			`SELECT total_chapters, genres, status FROM manga_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.TotalChapters, pq.Array(&meta.Genres), &meta.Status)
		if err != nil && err != sql.ErrNoRows {
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// also get the chapters for the manga
		rows, err := h.db.Query(
			`SELECT id, chapter_number, title, file_path, page_count, created_at FROM manga_chapters
			WHERE media_item_id = $1 ORDER BY chapter_number`,
			item.ID,
		)
		if utils.InternalError(w, err) {
			return
		}
		defer rows.Close()

		// map those chapters to useable structs
		for rows.Next() {
			var chapter MangaChapter
			err := rows.Scan(
				&chapter.ID, &chapter.ChapterNumber, &chapter.Title,
				&chapter.FilePath, &chapter.PageCount, &chapter.CreatedAt,
			)

			if utils.InternalError(w, err) {
				return
			}
			chapters = append(chapters, chapter)
		}
		metadata = MangaDetail{MangaMetadata: meta, Chapters: chapters}

	case "light_novel":
		var meta LightNovelMetadata
		var volumes []LightNovelVolume

		err := h.db.QueryRow(
			`SELECT author, total_volumes, genres FROM light_novel_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.Author, &meta.TotalVolumes, pq.Array(&meta.Genres))
		if err != nil && err != sql.ErrNoRows {
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}

		volumeRows, err := h.db.Query(
			`SELECT id, volume_number, title FROM light_novel_volumes
			WHERE media_item_id = $1 ORDER BY volume_number`,
			item.ID,
		)
		if utils.InternalError(w, err) {
			return
		}
		defer volumeRows.Close()

		for volumeRows.Next() {
			var volume LightNovelVolume
			if err := volumeRows.Scan(&volume.ID, &volume.VolumeNumber, &volume.Title); utils.InternalError(w, err) {
				return
			}
			volumes = append(volumes, volume)
		}

		metadata = LightNovelDetail{LightNovelMetadata: meta, Volumes: volumes}
	}

	// return the item and its metadata
	utils.JSON(w, MediaItemDetail{MediaItem: item, Metadata: metadata})
}

/*
Function:	MangaProgress
Purpose:	Add progress tracking for specific manga chapters
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) MangaProgress(w http.ResponseWriter, r *http.Request) {
	var req progressRequest
	// get the chapter id from the URL parameters
	chapterId := chi.URLParam(r, "chapterId")
	// get the user info from the request
	user := auth.GetUser(r)

	// decode the incoming request, check that the structure is correct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// insert the progress
	_, err := h.db.Exec(
		`INSERT INTO manga_progress (user_id, chapter_id, media_item_id, last_page_read, completed, updated_at)
		VALUES ($1, $2, (SELECT media_item_id FROM manga_chapters WHERE id = $2), $3, $4, NOW())
		ON CONFLICT (user_id, chapter_id) DO UPDATE SET
		last_page_read = EXCLUDED.last_page_read,
		completed = EXCLUDED.completed,
		updated_at = NOW()`,
		user.UserID, chapterId, req.LastPageRead, req.Completed,
	)

	if utils.InternalError(w, err) {
		return
	}

	// return no content
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	ServePage
Purpose:	Get a specific page for a manga chapter
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) ServePage(w http.ResponseWriter, r *http.Request) {
	var fPath string
	// get the chapters id from the URL parameters
	chapterId := chi.URLParam(r, "chapterId")
	// get the page number from the URL parameters
	pageNum, err := strconv.Atoi(chi.URLParam(r, "pageNum"))

	if utils.InternalError(w, err) {
		return
	}

	// find the file path for the page
	err = h.db.QueryRow(
		`SELECT file_path FROM manga_chapters WHERE id = $1`,
		chapterId,
	).Scan(&fPath)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "chapter not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	// open the file for reading
	reader, err := zip.OpenReader(fPath)
	if utils.InternalError(w, err) {
		return
	}
	defer reader.Close()

	sort.Slice(reader.File, func(i, j int) bool {
		return reader.File[i].Name < reader.File[j].Name
	})

	// check that the page number is in bounds
	if pageNum < 0 || pageNum >= len(reader.File) {
		utils.Error(w, http.StatusNotFound, "page not found")
		return
	}

	// read the page
	entry := reader.File[pageNum]

	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".webp": "image/webp",
	}

	// check that the content type of the file is actually allowed
	ct, ok := contentTypes[strings.ToLower(filepath.Ext(entry.Name))]
	if !ok {
		ct = "application/octet-stream"
	}

	f, err := entry.Open()
	if utils.InternalError(w, err) {
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", ct)
	io.Copy(w, f)
}

func (h *Handler) ServeVolume(w http.ResponseWriter, r *http.Request) {
	volumeId := chi.URLParam(r, "volumeId")
	id := chi.URLParam(r, "id")

	var filePath string
	err := h.db.QueryRow(
		`SELECT file_path FROM light_novel_volumes WHERE id = $1`,
		volumeId,
	).Scan(&filePath)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "volume not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	zr, err := zip.OpenReader(filePath)
	if utils.InternalError(w, err) {
		return
	}
	defer zr.Close()

	// find OPF path from container.xml
	opfPath := ""
	for _, f := range zr.File {
		if f.Name != "META-INF/container.xml" {
			continue
		}
		rc, err := f.Open()
		if utils.InternalError(w, err) {
			return
		}
		containerContent, err := io.ReadAll(rc)
		rc.Close()
		if utils.InternalError(w, err) {
			return
		}
		pathRe := regexp.MustCompile(`full-path="([^"]+)"`)
		if m := pathRe.FindSubmatch(containerContent); m != nil {
			opfPath = string(m[1])
		}
		break
	}
	if opfPath == "" {
		utils.Error(w, http.StatusInternalServerError, "could not locate OPF in epub")
		return
	}

	var opfContent []byte
	for _, f := range zr.File {
		if f.Name == opfPath {
			rc, err := f.Open()
			if utils.InternalError(w, err) {
				return
			}
			opfContent, err = io.ReadAll(rc)
			rc.Close()
			if utils.InternalError(w, err) {
				return
			}
			break
		}
	}

	if opfContent == nil {
		utils.Error(w, http.StatusInternalServerError, "content.opf not found in epub")
		return
	}

	// extract href values from manifest by id
	manifestHrefs := map[string]string{}
	opfStr := string(opfContent)
	itemRe := regexp.MustCompile(`<item\s+id="([^"]+)"\s+href="([^"]+)"`)
	for _, match := range itemRe.FindAllStringSubmatch(opfStr, -1) {
		manifestHrefs[match[1]] = match[2]
	}

	// extract spine order
	spineRe := regexp.MustCompile(`<itemref\s+idref="([^"]+)"`)
	var spineIDs []string
	for _, match := range spineRe.FindAllStringSubmatch(opfStr, -1) {
		spineIDs = append(spineIDs, match[1])
	}

	// entries to skip
	skip := map[string]bool{
		"cover": true, "toc": true, "copyright": true, "signup": true,
	}

	// build a lookup from filename to zip entry
	zipEntries := map[string]*zip.File{}
	for _, f := range zr.File {
		zipEntries[f.Name] = f
	}

	apiURL := os.Getenv("API_URL")

	var body strings.Builder
	body.WriteString(`<!DOCTYPE html><html><body style="max-width:720px;margin:0 auto;padding:1rem;font-family:serif;line-height:1.8;">`)

	for _, spineID := range spineIDs {
		href, ok := manifestHrefs[spineID]
		if !ok {
			continue
		}

		base := strings.TrimSuffix(filepath.Base(href), filepath.Ext(href))
		if skip[base] {
			continue
		}

		entryPath := "OEBPS/" + href
		f, ok := zipEntries[entryPath]
		if !ok {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		// extract just the body content
		html := string(content)
		bodyStart := strings.Index(html, "<body")
		bodyEnd := strings.LastIndex(html, "</body>")
		if bodyStart != -1 && bodyEnd != -1 {
			// find the end of the opening body tag
			tagEnd := strings.Index(html[bodyStart:], ">")
			html = html[bodyStart+tagEnd+1 : bodyEnd]
		}

		// rewrite image paths
		html = strings.ReplaceAll(html, `src="../Images/`,
			fmt.Sprintf(`src="%s/light-novels/%s/volumes/%s/images/`, apiURL, id, volumeId))
		html = strings.ReplaceAll(html, `src="../images/`,
			fmt.Sprintf(`src="%s/light-novels/%s/volumes/%s/images/`, apiURL, id, volumeId))

		body.WriteString(html)
		body.WriteString(`<div style="height:1px;background:#333;margin:2rem 0;"></div>`)
	}

	body.WriteString(`</body></html>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(body.String()))
}

func (h *Handler) ServeVolumeImage(w http.ResponseWriter, r *http.Request) {
	volumeId := chi.URLParam(r, "volumeId")
	imageName := chi.URLParam(r, "imageName")

	var filePath string
	err := h.db.QueryRow(
		`SELECT file_path FROM light_novel_volumes WHERE id = $1`,
		volumeId,
	).Scan(&filePath)

	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "volume not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	zr, err := zip.OpenReader(filePath)
	if utils.InternalError(w, err) {
		return
	}
	defer zr.Close()

	for _, f := range zr.File {
		if filepath.Base(f.Name) != imageName {
			continue
		}

		rc, err := f.Open()
		if utils.InternalError(w, err) {
			return
		}
		defer rc.Close()

		contentTypes := map[string]string{
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".png":  "image/png",
			".webp": "image/webp",
			".gif":  "image/gif",
		}
		ct, ok := contentTypes[strings.ToLower(filepath.Ext(imageName))]
		if !ok {
			ct = "application/octet-stream"
		}

		w.Header().Set("Content-Type", ct)
		io.Copy(w, rc)
		return
	}

	utils.Error(w, http.StatusNotFound, "image not found in archive")
}

func (h *Handler) GetEpisodes(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	mediaID := chi.URLParam(r, "id")
	episodes := []Episode{}

	rows, err := h.db.Query(
		`SELECT e.id, e.season_number, e.episode_number, e.title,
		COALESCE(uap.watched, false)
		FROM episodes e
		LEFT JOIN user_anime_progress uap ON uap.episode_id = e.id AND uap.user_id = $2
		WHERE e.media_item_id = $1
		ORDER BY e.season_number, e.episode_number`,
		mediaID, user.UserID,
	)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ep Episode
		if err := rows.Scan(&ep.ID, &ep.SeasonNumber, &ep.EpisodeNumber, &ep.Title, &ep.Watched); err != nil {
			utils.InternalError(w, err)
			return
		}
		episodes = append(episodes, ep)
	}

	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, episodes)
}
