package search

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/covers"
	"github.com/Mantie7553/MediaHub/backend/internal/downloader"
	"github.com/Mantie7553/MediaHub/backend/internal/locks"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
	"github.com/lib/pq"
)

type Handler struct {
	db            *sql.DB
	yt            *ytdlp.YtdlpClient
	discovery     *cache.Memo[discoveryResponse]
	searches      *cache.Memo[[]SearchResult]
	musicSearches *cache.Memo[[]ytdlp.SearchResult]
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:            db,
		yt:            ytdlp.NewYtdlpClient("YTDLP_PATH", "MUSIC_DIR"),
		discovery:     newDiscoveryCache(),
		searches:      newSearchCache(),
		musicSearches: newMusicSearchCache(),
	}
}

type SearchResult struct {
	ExternalID     string `json:"external_id"`
	ExternalSource string `json:"external_source"`
	Title          string `json:"title"`
	CoverImageURL  string `json:"cover_image_url"`
	Type           string `json:"type"`
	// episodes or chapters the source says the title has, when it knows
	TotalParts *int `json:"total_parts,omitempty"`
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
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	// check that all required fields are present
	if req.Title == "" || req.Type == "" {
		utils.Error(w, http.StatusBadRequest, " title and type are required")
		return
	}

	// check that action is valid
	if req.Action != "list" && req.Action != "download" && req.Action != "both" {
		utils.Error(w, http.StatusBadRequest, "action must be list, download, or both")
		return
	}

	// anything downloaded by yt-dlp (music, and types without a handler of their own) needs a
	// YouTube video link; any link sent is checked, and kept only in its canonical form
	download := req.Action == "download" || req.Action == "both"
	usesYtdlp := req.Type != "anime" && req.Type != "movie" && req.Type != "manga"
	if req.SourceURL != "" || (download && usesYtdlp) {
		canonical, err := ytdlp.CanonicalURL(req.SourceURL)
		if err != nil {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		req.SourceURL = canonical
	}

	// a track's artist and album become its folders under Music
	if req.Type == "music_track" {
		req.Artist, req.Album = musicfiles.CleanName(req.Artist), musicfiles.CleanName(req.Album)
		if req.Artist == "" {
			utils.Error(w, http.StatusBadRequest, "artist is required")
			return
		}
	}

	// insert the media item if it doesn't exist, otherwise use the existing one. Any user can
	// save, and a title in the library is shared by everyone, so saving never changes one that
	// is already there (only an admin can fill in what it is missing)
	mediaItemID, created, err := h.findOrCreate(req)
	var bad badRequest
	if errors.As(err, &bad) {
		utils.Error(w, http.StatusBadRequest, string(bad))
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	if created {
		covers.LocalizeLater(h.db, mediaItemID, req.CoverImageURL)
	}
	if !created {
		if user.Role == "admin" {
			h.db.Exec(
				`UPDATE media_items SET
				external_id = COALESCE(external_id, $1),
				external_source = COALESCE(external_source, $2),
				cover_image_url = COALESCE(cover_image_url, $3)
				WHERE id = $4`,
				utils.NullString(req.ExternalID), utils.NullString(req.ExternalSource), utils.NullString(req.CoverImageURL), mediaItemID,
			)
			// does nothing unless that filled in a missing cover
			covers.LocalizeLater(h.db, mediaItemID, req.CoverImageURL)
		}
		// metadata is fetched, and downloads dispatched, by the ids the library has for the
		// title, never ones from the request
		var externalID, externalSource sql.NullString
		err := h.db.QueryRow(`SELECT external_id, external_source FROM media_items WHERE id = $1`, mediaItemID).
			Scan(&externalID, &externalSource)
		if utils.InternalError(w, err) {
			return
		}
		req.ExternalID, req.ExternalSource = externalID.String, externalSource.String
	}

	if req.Type == "anime" && req.ExternalSource == "anilist" && req.ExternalID != "" {
		var (
			studio string
			status string
			genres []string
		)

		client := anilist.NewAnilistClient("")
		anilistID, convErr := strconv.Atoi(req.ExternalID)
		if convErr != nil {
			logger.Warn("invalid anilist external_id %s: %v", req.ExternalID, convErr)
		} else {
			anime, fetchErr := client.GetByID(anilistID)
			if fetchErr != nil {
				logger.Warn("failed to fetch anilist metadata for %s: %v", req.ExternalID, fetchErr)
			} else {
				status = anime.LibraryStatus()
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
			`+locks.KeepMeta(locks.Studio, "anime_metadata", "studio", "EXCLUDED.studio")+`,
			`+locks.KeepMeta(locks.Genres, "anime_metadata", "genres", "EXCLUDED.genres")+`,
			`+locks.KeepMeta(locks.Status, "anime_metadata", "status", "EXCLUDED.status"),
			mediaItemID,
			utils.NullString(studio),
			utils.NullString(status),
			pq.Array(genres),
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	if req.Type == "movie" && req.ExternalSource == "anilist" && req.ExternalID != "" {
		var (
			runtime_mins int
			director     string
			genres       []string
		)

		client := anilist.NewAnilistClient("")
		anilistID, convErr := strconv.Atoi(req.ExternalID)
		if convErr != nil {
			logger.Warn("invalid anilist external_id %s: %v", req.ExternalID, convErr)
		} else {
			movie, fetchErr := client.GetByID(anilistID)
			if fetchErr != nil {
				logger.Warn("failed to fetch anilist metadata for %s: %v", req.ExternalID, fetchErr)
			} else {
				genres = append(genres, movie.Genres...)
			}
		}

		_, err = h.db.Exec(
			`INSERT INTO movie_metadata (media_item_id, runtime_mins, director, genres)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO UPDATE SET
			`+locks.KeepMeta(locks.RuntimeMins, "movie_metadata", "runtime_mins", "EXCLUDED.runtime_mins")+`,
			`+locks.KeepMeta(locks.Director, "movie_metadata", "director", "EXCLUDED.director")+`,
			`+locks.KeepMeta(locks.Genres, "movie_metadata", "genres", "EXCLUDED.genres"),
			mediaItemID,
			utils.NullInt(&runtime_mins),
			utils.NullString(director),
			pq.Array(genres),
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	if req.Type == "manga" && req.ExternalSource == "mangadex" && req.ExternalID != "" {
		var (
			status        string
			genres        []string
			totalChapters int
		)

		client := mangadex.NewMangaDexClient("")
		manga, err := client.GetByID(req.ExternalID)
		if err != nil {
			logger.Warn("failed to fetch mangadex metadata for %s: %v", req.ExternalID, err)
		} else {
			status = manga.Attributes.Status
			genres = manga.Genres()
			totalChapters = manga.TotalChapters()
		}

		_, err = h.db.Exec(
			`INSERT INTO manga_metadata (media_item_id, total_chapters, genres, status)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO UPDATE SET
			`+locks.KeepMeta(locks.TotalChapters, "manga_metadata", "total_chapters", "EXCLUDED.total_chapters")+`,
			`+locks.KeepMeta(locks.Genres, "manga_metadata", "genres", "EXCLUDED.genres")+`,
			`+locks.KeepMeta(locks.Status, "manga_metadata", "status", "EXCLUDED.status"),
			mediaItemID,
			utils.NullInt(&totalChapters),
			pq.Array(genres),
			utils.NullString(status),
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	// a track already in the library keeps its artist and album
	if created && req.Type == "music_track" && req.ExternalSource == "ytdlp" && req.ExternalID != "" {
		var albumID *string
		// a track with no album is one of the artist's singles, which the library shows as
		// an album of its own like the Singles folder it downloads into
		if req.Album == "" && req.Artist != "" {
			req.Album = "Singles"
		}
		if req.Album != "" {
			var id string
			err := h.db.QueryRow(
				`INSERT INTO albums (title, artist)
				VALUES ($1, $2)
				ON CONFLICT (LOWER(title), LOWER(artist)) DO NOTHING
				RETURNING id`,
				req.Album, req.Artist,
			).Scan(&id)
			if err == nil {
				albumID = &id
			}

			if errors.Is(err, sql.ErrNoRows) {
				err = h.db.QueryRow(
					`SELECT id FROM albums WHERE LOWER(title) = LOWER($1) AND LOWER(artist) = LOWER($2)`,
					req.Album, req.Artist,
				).Scan(&id)
				if err == nil {
					albumID = &id
				}
			}
		}
		_, err = h.db.Exec(
			`INSERT INTO music_metadata (media_item_id, artist, duration_secs, album_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO NOTHING`,
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
			if utils.InternalError(w, tracking.SetProgressCount(h.db, user.UserID, mediaItemID, *req.Progress)) {
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

		if err != nil && !errors.Is(err, sql.ErrNoRows) {
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

// badRequest is a problem with what was sent, answered with a 400
type badRequest string

func (b badRequest) Error() string { return string(b) }

/*
Function:	findOrCreate
Purpose:	the media item a save is for: the one with this title and type, or the one with this
external id, or a new one. Never changes an item that exists
Returns:	its id, and whether it was just created
*/
func (h *Handler) findOrCreate(req saveRequest) (string, bool, error) {
	var id string
	err := h.db.QueryRow(`SELECT id FROM media_items WHERE title = $1 AND type = $2`, req.Title, req.Type).Scan(&id)
	if !errors.Is(err, sql.ErrNoRows) {
		return id, false, err
	}
	if req.ExternalID == "" || req.ExternalSource == "" {
		return "", false, badRequest("external id and external source are required")
	}

	err = h.db.QueryRow(
		`INSERT INTO media_items (type, title, cover_image_url, external_id, external_source)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (external_id, external_source) DO NOTHING
		RETURNING id`,
		req.Type, req.Title, utils.NullString(req.CoverImageURL), req.ExternalID, req.ExternalSource,
	).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	// already in the library under another title
	err = h.db.QueryRow(
		`SELECT id FROM media_items WHERE external_id = $1 AND external_source = $2`,
		req.ExternalID, req.ExternalSource,
	).Scan(&id)
	return id, false, err
}
