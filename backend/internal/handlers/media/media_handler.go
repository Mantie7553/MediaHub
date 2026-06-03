package media

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
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
				log.Printf("failed to fetch mangadex metadata for %s: %v", req.ExternalID, err)
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
		OR EXISTS (SELECT 1 FROM radarr_items WHERE media_item_id = mi.id))
		OR EXISTS (SELECT 1 FROM manga_chapters WHERE media_item_id = mi.id AND file_path IS NOT NULL)`)
	}

	queryString := `SELECT id, type, title, description, cover_image_url, release_date, external_id, external_source, created_at FROM media_items mi`

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
			&item.ExternalSource, &item.CreatedAt,
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
			`SELECT runtime_mins, director, genres FROM movie_metadata WHERE media_item_id = $1`,
			item.ID,
		).Scan(&meta.RuntimeMins, &meta.Director, pq.Array(&meta.Genres))
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

func (h *Handler) GetEpisodes(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "id")
	episodes := []Episode{}

	rows, err := h.db.Query(
		`SELECT id, season_number, episode_number, title 
        FROM episodes 
        WHERE media_item_id = $1 
        ORDER BY season_number, episode_number`,
		mediaID,
	)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ep Episode
		if err := rows.Scan(&ep.ID, &ep.SeasonNumber, &ep.EpisodeNumber, &ep.Title); err != nil {
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
