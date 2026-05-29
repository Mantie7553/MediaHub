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
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/lib/pq"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
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
			results, err = client.Trending("ANIME", 20)
		} else {
			results, err = client.Search("ANIME", q, 20)
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
		`INSERT INTO media_items (type, title, cover_image_url, external_id, external_source)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (external_id, external_source) DO UPDATE SET title = EXCLUDED.title
		RETURNING id`,
		req.Type, req.Title, utils.NullString(req.CoverImageURL), req.ExternalID, req.ExternalSource,
	).Scan(&mediaItemID)
	if utils.InternalError(w, err) {
		return
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

	// add to the user's list if requested
	if req.Action == "list" || req.Action == "both" {
		_, err = h.db.Exec(
			`INSERT INTO user_media_status (user_id, media_item_id, status)
			VALUES ($1, $2, 'plan_to_watch')
			ON CONFLICT (user_id, media_item_id) DO NOTHING`,
			user.UserID, mediaItemID,
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	// create a download request if requested
	if req.Action == "download" || req.Action == "both" {
		_, err = h.db.Exec(
			`INSERT INTO download_requests (requested_by, media_item_id, status, auto_approved)
			VALUES ($1, $2, 'pending', false)
			ON CONFLICT (requested_by, media_item_id) DO NOTHING`,
			user.UserID, mediaItemID,
		)
		if utils.InternalError(w, err) {
			return
		}
	}

	utils.JSON(w, map[string]string{"id": mediaItemID}, http.StatusCreated)
}
