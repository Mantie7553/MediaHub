package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/downloader"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/lib/pq"
)

type Handler struct {
	db *sql.DB
	yt *ytdlp.YtdlpClient
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db: db,
		yt: ytdlp.NewYtdlpClient("YTDLP_PATH", "MUSIC_DIR"),
	}
}

type SearchResult struct {
	ExternalID     string `json:"external_id"`
	ExternalSource string `json:"external_source"`
	Title          string `json:"title"`
	CoverImageURL  string `json:"cover_image_url"`
	Type           string `json:"type"`
}

type saveRequest struct {
	ExternalID     string `json:"external_id"`
	ExternalSource string `json:"external_source"`
	Title          string `json:"title"`
	CoverImageURL  string `json:"cover_image_url"`
	Type           string `json:"type"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	Score          *int   `json:"rating"`
	Progress       *int   `json:"progress"`
	Artist         string `json:"artist"`
	Album          string `json:"album"`
	DurationSecs   *int   `json:"duration_secs"`
	SourceURL      string `json:"source_url"`
}

/*
Function:	Search
Purpose:	Search Anilist or MangaDex for media of a given type
Params:
  - w: http response writer to respond to the front endconst
  - r: http request coming from the front end
*/
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	mediaType := r.URL.Query().Get("type")
	q := r.URL.Query().Get("q")

	switch mediaType {
	case "anime":
		// query anilist
		var results []anilist.Media
		var err error
		client := anilist.NewAnilistClient("")
		if q == "" {
			results, err = client.Trending("ANIME", 20, "TV")
		} else {
			results, err = client.Search("ANIME", q, 20, "TV")
		}
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "search failed")
			return
		}
		// go through the items returned
		out := make([]SearchResult, 0, len(results))
		for _, m := range results {
			title := m.Title.English
			if title == "" {
				title = m.Title.Romaji
			}
			// format entries to return
			out = append(out, SearchResult{
				ExternalID:     strconv.Itoa(m.ID),
				ExternalSource: "anilist",
				Title:          title,
				CoverImageURL:  m.CoverImage.Large,
				Type:           "anime",
			})
		}
		utils.JSON(w, out)

	case "movie":
		var results []anilist.Media
		var err error
		client := anilist.NewAnilistClient("")
		if q == "" {
			results, err = client.Trending("ANIME", 20, "MOVIE")
		} else {
			results, err = client.Search("ANIME", q, 20, "MOVIE")
		}
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "search failed")
			return
		}
		// go through the items returned
		out := make([]SearchResult, 0, len(results))
		for _, m := range results {
			title := m.Title.English
			if title == "" {
				title = m.Title.Romaji
			}
			// format entries to return
			out = append(out, SearchResult{
				ExternalID:     strconv.Itoa(m.ID),
				ExternalSource: "anilist",
				Title:          title,
				CoverImageURL:  m.CoverImage.Large,
				Type:           "movie",
			})
		}
		utils.JSON(w, out)
	case "manga":
		// query mangadex
		client := mangadex.NewMangaDexClient("")
		var results []mangadex.Manga
		var err error
		if q == "" {
			results, err = client.Trending()
		} else {
			results, err = client.Search(q)
		}
		if utils.InternalError(w, err) {
			return
		}
		// go through the items returned
		out := make([]SearchResult, 0, len(results))
		for _, m := range results {
			var fileName string
			for _, rel := range m.Relationships {
				if rel.Type == "cover_art" {
					fileName = rel.Attributes.FileName
					break
				}
			}
			// format entries to return
			coverURL := fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", m.ID, fileName)
			title := m.Attributes.Title.En
			if title == "" {
				title = m.Attributes.Title.JaRo
			}
			if title == "" {
				title = m.Attributes.Title.Ja
			}
			out = append(out, SearchResult{
				ExternalID:     m.ID,
				ExternalSource: "mangadex",
				Title:          title,
				CoverImageURL:  coverURL,
				Type:           "manga",
			})
		}
		utils.JSON(w, out)

	default:
		utils.Error(w, http.StatusBadRequest, "type must be anime or manga")
	}
}

/*
Function:	Save
Purpose:	Save a media item to the database and add it to the user's list,

	create a download request, or both

Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the front end
*/
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	var req saveRequest
	user := auth.GetUser(r)

	// decode the incoming request, check that the structure is correct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// check that all required fields are present
	if req.ExternalID == "" || req.ExternalSource == "" || req.Title == "" || req.Type == "" {
		utils.Error(w, http.StatusBadRequest, "external_id, external_source, title and type are required")
		return
	}

	// check that action is valid
	if req.Action != "list" && req.Action != "download" && req.Action != "both" {
		utils.Error(w, http.StatusBadRequest, "action must be list, download, or both")
		return
	}

	// insert the media item if it doesn't exist, otherwise return the existing id
	var mediaItemID string
	err := h.db.QueryRow(
		`SELECT id FROM media_items WHERE title = $1 AND type = $2`,
		req.Title, req.Type,
	).Scan(&mediaItemID)

	if err == sql.ErrNoRows {
		err := h.db.QueryRow(
			`INSERT INTO media_items (type, title, cover_image_url, external_id, external_source)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (external_id, external_source) DO UPDATE SET title = EXCLUDED.title
			RETURNING id`,
			req.Type, req.Title, utils.NullString(req.CoverImageURL), req.ExternalID, req.ExternalSource,
		).Scan(&mediaItemID)
		if utils.InternalError(w, err) {
			return
		}
	} else if err != nil {
		utils.InternalError(w, err)
		return
	} else {
		h.db.Exec(
			`UPDATE media_items SET 
			external_id = COALESCE(external_id, $1),
			external_source = COALESCE(external_source, $2),
			cover_image_url = COALESCE(cover_image_url, $3)
			WHERE id = $4`,
			req.ExternalID, req.ExternalSource, utils.NullString(req.CoverImageURL), mediaItemID,
		)
	}

	if req.Type == "anime" && req.ExternalSource == "anilist" {
		var (
			studio string
			status string
			genres []string
		)

		client := anilist.NewAnilistClient("")
		anilistID, convErr := strconv.Atoi(req.ExternalID)
		if convErr != nil {
			log.Printf("invalid anilist external_id %s: %v", req.ExternalID, convErr)
		} else {
			anime, fetchErr := client.GetByID(anilistID)
			if fetchErr != nil {
				log.Printf("failed to fetch anilist metadata for %s: %v", req.ExternalID, fetchErr)
			} else {
				switch anime.Status {
				case "RELEASING":
					status = "airing"
				case "FINISHED":
					status = "finished"
				case "NOT_YET_RELEASED":
					status = "upcoming"
				default:
					status = ""
				}
				genres = append(genres, anime.Genres...)
				if len(anime.Studios) > 0 {
					studio = anime.Studios[0]
				}
			}
		}

		_, err = h.db.Exec(
			`INSERT INTO anime_metadata (media_item_id, studio, status, genres)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO UPDATE SET
			studio = EXCLUDED.studio,
			genres = EXCLUDED.genres,
			status = EXCLUDED.status`,
			mediaItemID,
			utils.NullString(studio),
			utils.NullString(status),
			pq.Array(genres),
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	if req.Type == "movie" && req.ExternalSource == "anilist" {
		var (
			runtime_mins int
			director     string
			genres       []string
		)

		client := anilist.NewAnilistClient("")
		anilistID, convErr := strconv.Atoi(req.ExternalID)
		if convErr != nil {
			log.Printf("invalid anilist external_id %s: %v", req.ExternalID, convErr)
		} else {
			movie, fetchErr := client.GetByID(anilistID)
			if fetchErr != nil {
				log.Printf("failed to fetch anilist metadata for %s: %v", req.ExternalID, fetchErr)
			} else {
				genres = append(genres, movie.Genres...)
			}
		}

		_, err = h.db.Exec(
			`INSERT INTO movie_metadata (media_item_id, runtime_mins, director, genres)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO UPDATE SET
			runtime_mins = EXCLUDED.runtime_mins,
			director = EXCLUDED.director,
			genres = EXCLUDED.genres`,
			mediaItemID,
			utils.NullInt(&runtime_mins),
			utils.NullString(director),
			pq.Array(genres),
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	if req.Type == "manga" && req.ExternalSource == "mangadex" {
		var (
			status        string
			genres        []string
			totalChapters int
		)

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

		_, err = h.db.Exec(
			`INSERT INTO manga_metadata (media_item_id, total_chapters, genres, status)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO UPDATE SET
			total_chapters = EXCLUDED.total_chapters,
			genres = EXCLUDED.genres,
			status = EXCLUDED.status`,
			mediaItemID,
			utils.NullInt(&totalChapters),
			pq.Array(genres),
			utils.NullString(status),
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	if req.Type == "music_track" && req.ExternalSource == "ytdlp" {
		var albumID *string
		if req.Album != "" {
			var id string
			err := h.db.QueryRow(
				`INSERT INTO albums (title, artist)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
				RETURNING id`,
				req.Album, req.Artist,
			).Scan(&id)
			if err == nil {
				albumID = &id
			}
		}
		_, err = h.db.Exec(
			`INSERT INTO music_metadata (media_item_id, artist, duration_secs, album_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO UPDATE SET
			artist = EXCLUDED.artist,
			duration_secs = EXCLUDED.duration_secs,
			album_id = EXCLUDED.album_id`,
			mediaItemID,
			req.Artist,
			req.DurationSecs,
			albumID,
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	// add to the user's list if requested
	if req.Action == "list" || req.Action == "both" {
		_, err = h.db.Exec(
			`INSERT INTO user_media_status (user_id, media_item_id, status, rating)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, media_item_id) DO NOTHING`,
			user.UserID, mediaItemID, req.Status, req.Score,
		)
		if utils.InternalError(w, err) {
			return
		}
		if req.Progress != nil && *req.Progress > 0 {
			switch req.Type {
			case "anime":
				_, err = h.db.Exec(
					`INSERT INTO user_anime_progress (user_id, media_item_id, episodes_watched)
					VALUES ($1, $2, $3)
					ON CONFLICT (user_id, media_item_id) DO UPDATE SET episodes_watched = EXCLUDED.episodes_watched`,
					user.UserID, mediaItemID, req.Progress,
				)
			case "manga":
				_, err = h.db.Exec(
					`INSERT INTO user_manga_progress (user_id, media_item_id, chapters_read)
					VALUES ($1, $2, $3)
					ON CONFLICT (user_id, media_item_id) DO UPDATE SET chapters_read = EXCLUDED.chapters_read`,
					user.UserID, mediaItemID, req.Progress,
				)
			}
			if utils.InternalError(w, err) {
				return
			}
		}
	}

	// create a download request if requested
	if req.Action == "download" || req.Action == "both" {
		// check the users download permissions
		var downloadPermissions string
		err := h.db.QueryRow(
			`SELECT download_permission FROM users WHERE id = $1`,
			user.UserID,
		).Scan(&downloadPermissions)
		if utils.InternalError(w, err) {
			return
		}
		status := "pending"
		autoApproved := false
		if downloadPermissions == "auto_approved" {
			status = "approved"
			autoApproved = true
		}

		var requestID string
		err = h.db.QueryRow(
			`INSERT INTO download_requests (requested_by, media_item_id, status, auto_approved, source_url)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (requested_by, media_item_id) DO NOTHING
			RETURNING id`,
			user.UserID, mediaItemID, status, autoApproved, req.SourceURL,
		).Scan(&requestID)

		if err != nil && err != sql.ErrNoRows {
			utils.InternalError(w, err)
			return
		}

		if autoApproved && requestID != "" {
			if _, err := downloader.Dispatch(h.db, requestID, mediaItemID, req.SourceURL, req.Type, &req.ExternalID); err != nil {
				logger.Warn("auto-dispatch failed: %s", err.Error())
			}
		}
	}

	utils.JSON(w, map[string]string{"id": mediaItemID}, http.StatusCreated)
}
