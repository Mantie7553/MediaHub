package sync

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/dhowden/tag"
	"github.com/lib/pq"
)

type Handler struct {
	db       *sql.DB
	sonarr   *arr.ArrClient
	radarr   *arr.ArrClient
	mangadex *mangadex.MangaDexClient
	anilist  *anilist.AnilistClient
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:       db,
		sonarr:   arr.NewArrClient("SONARR_URL", "SONARR_API_KEY"),
		radarr:   arr.NewArrClient("RADARR_URL", "RADARR_API_KEY"),
		mangadex: mangadex.NewMangaDexClient("MANGADEX_URL"),
		anilist:  anilist.NewAnilistClient(""),
	}
}

func (h *Handler) SyncSonar(w http.ResponseWriter, r *http.Request) {
	series, err := h.sonarr.GetAllSeries()
	if utils.InternalError(w, err) {
		return
	}

	tvdbToAnilist, _, err := arr.FetchAnimeMaps()
	if err != nil {
		logger.Error("failed to fetch fribb anime-list map: %s", err.Error())
		tvdbToAnilist = map[int]int{}
	}

	for _, s := range series {
		// Look up existing record by sonarr_series_id first
		var mediaItemID string
		err := h.db.QueryRow(
			`SELECT media_item_id FROM sonarr_items WHERE sonarr_series_id = $1`,
			s.ID,
		).Scan(&mediaItemID)

		if err == sql.ErrNoRows {
			// New series — insert media_item
			var releaseDate *string
			if s.Year > 0 {
				date := fmt.Sprintf("%04d-01-01", s.Year)
				releaseDate = &date
			}

			err = h.db.QueryRow(
				`INSERT INTO media_items (type, title, cover_image_url,description, release_date)
				VALUES ('anime', $1, $2, $3, $4)
				RETURNING id`,
				s.Title, utils.NullString(s.PosterURL()), utils.NullString(s.Overview), releaseDate,
			).Scan(&mediaItemID)
			if err != nil {
				logger.Error("failed to create media item for %s: %s", s.Title, err.Error())
				continue
			}
		} else if err == nil {
			// Existing — update native fields
			var releaseDate *string
			if s.Year > 0 {
				date := fmt.Sprintf("%04d-01-01", s.Year)
				releaseDate = &date
			}
			h.db.Exec(
				`UPDATE media_items SET
				title = $1,
				cover_image_url = $2,
				description = COALESCE(description, $3),
				release_date = COALESCE(release_date, $4)
				WHERE id = $5`,
				s.Title, utils.NullString(s.PosterURL()), utils.NullString(s.Overview), releaseDate, mediaItemID,
			)
		} else {
			logger.Error("failed to query sonarr_items for %s: %s", s.Title, err.Error())
			continue
		}

		var aniResult *anilist.Media
		if s.TvdbID > 0 {
			if anilistID, ok := tvdbToAnilist[s.TvdbID]; ok {
				full, fetchErr := h.anilist.GetByID(anilistID)
				time.Sleep(2 * time.Second)
				if fetchErr != nil {
					logger.Error("anilist fetch failed for %s (anilist %d): %s", s.Title, anilistID, fetchErr.Error())
				} else {
					aniResult = full
				}
			}
		}

		if aniResult != nil {
			id := strconv.Itoa(aniResult.ID)
			h.db.Exec(
				`UPDATE media_items SET
				external_id = COALESCE(external_id, $1),
				external_source = COALESCE(external_source, $2)
				WHERE id = $3`,
				id, "anilist", mediaItemID,
			)

			if aniResult.StartDate.Year != nil {
				date := fmt.Sprintf("%04d-%02d-%02d",
					*aniResult.StartDate.Year,
					utils.SafeMonth(aniResult.StartDate.Month),
					utils.SafeDay(aniResult.StartDate.Day),
				)
				h.db.Exec(
					`UPDATE media_items SET release_date = $1 WHERE id = $2`,
					date, mediaItemID,
				)
			}

			var status string
			switch aniResult.Status {
			case "RELEASING":
				status = "airing"
			case "FINISHED":
				status = "finished"
			case "NOT_YET_RELEASED":
				status = "upcoming"
			}

			var studio string
			if len(aniResult.Studios) > 0 {
				studio = aniResult.Studios[0]
			}

			_, err = h.db.Exec(
				`INSERT INTO anime_metadata (media_item_id, studio, status, genres)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (media_item_id) DO UPDATE SET
				studio = EXCLUDED.studio,
				status = EXCLUDED.status,
				genres = EXCLUDED.genres`,
				mediaItemID, utils.NullString(studio), utils.NullString(status), pq.Array(aniResult.Genres),
			)
			if err != nil {
				logger.Error("failed to upsert anime metadata for %s: %s", s.Title, err.Error())
			}
		} else {
			// No AniList match — still upsert anime_metadata with Sonarr genres
			_, err = h.db.Exec(
				`INSERT INTO anime_metadata (media_item_id, genres)
				VALUES ($1, $2)
				ON CONFLICT (media_item_id) DO UPDATE SET
				genres = EXCLUDED.genres`,
				mediaItemID, pq.Array(s.Genres),
			)
			if err != nil {
				logger.Error("failed to update or insert anime metadata for %s: %s", s.Title, err.Error())
			}
		}

		_, err = h.db.Exec(
			`INSERT INTO sonarr_items (media_item_id, sonarr_series_id)
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO UPDATE SET
			sonarr_series_id = EXCLUDED.sonarr_series_id,
			last_synced_at = NOW()`,
			mediaItemID, s.ID,
		)
		if err != nil {
			logger.Error("failed to update or insert sonarr_items for %s : %s", s.Title, err.Error())
			continue
		}

		episodes, err := h.sonarr.GetEpisodes(s.ID)
		if err != nil {
			logger.Error("failed to get episodes for %s: %s", s.Title, err.Error())
			continue
		}

		for _, ep := range episodes {
			if !ep.HasFile {
				continue
			}

			filePath, err := h.sonarr.GetEpisodeFilePath(ep.EpisodeFileID)
			if err != nil {
				logger.Error("failed to get file path for episode %d: %s", ep.ID, err.Error())
				continue
			}

			_, err = h.db.Exec(
				`INSERT INTO episodes (media_item_id, season_number, episode_number, title, file_path, sonarr_id)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (media_item_id, season_number, episode_number) DO UPDATE SET
				file_path = EXCLUDED.file_path,
				title = EXCLUDED.title,
				updated_at = NOW()`,
				mediaItemID, ep.SeasonNumber, ep.EpisodeNumber, ep.Title, filePath, ep.ID,
			)
			if err != nil {
				logger.Error("failed to update or insert episode %d: %s", ep.ID, err.Error())
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SyncRadarr(w http.ResponseWriter, r *http.Request) {
	movies, err := h.radarr.GetAllMovies()
	if utils.InternalError(w, err) {
		return
	}

	_, tmdbToAnilist, err := arr.FetchAnimeMaps()
	if err != nil {
		logger.Error("failed to fetch fribb anime-list map: %s", err.Error())
		tmdbToAnilist = map[int]int{}
	}

	for _, m := range movies {
		// Look up existing record by radarr_movie_id first
		var mediaItemID string
		err := h.db.QueryRow(
			`SELECT media_item_id FROM radarr_items WHERE radarr_movie_id = $1`,
			m.ID,
		).Scan(&mediaItemID)

		if err == sql.ErrNoRows {
			// New movie - insert media_item using Radarr's native data
			err = h.db.QueryRow(
				`INSERT INTO media_items (type, title, cover_image_url, description, release_date)
				VALUES ('movie', $1, $2, $3, $4)
				RETURNING id`,
				m.Title, utils.NullString(m.PosterURL()), utils.NullString(m.Overview), utils.NullString(m.ReleaseDate()),
			).Scan(&mediaItemID)
			if err != nil {
				logger.Error("failed to create media item for %s: %s", m.Title, err.Error())
				continue
			}
		} else if err == nil {
			// Existing - update native fields
			h.db.Exec(
				`UPDATE media_items SET
				title = $1,
				cover_image_url = $2,
				description = COALESCE(description, $3),
				release_date = COALESCE(release_date, $4)
				WHERE id = $5`,
				m.Title, utils.NullString(m.PosterURL()), utils.NullString(m.Overview), utils.NullString(m.ReleaseDate()), mediaItemID,
			)
		} else {
			logger.Error("failed to query radarr_items for %s: %s", m.Title, err.Error())
			continue
		}

		// Upsert movie_metadata using Radarr's genres as baseline
		_, err = h.db.Exec(
			`INSERT INTO movie_metadata (media_item_id, genres)
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO UPDATE SET
			genres = EXCLUDED.genres`,
			mediaItemID, pq.Array(m.Genres),
		)
		if err != nil {
			logger.Error("failed to udpate or insert movie metadata for %s: %s", m.Title, err.Error())
		}

		// AniList enrichment via Fribb TMDB mapping
		if m.TmdbID > 0 {
			if anilistID, ok := tmdbToAnilist[m.TmdbID]; ok {
				full, fetchErr := h.anilist.GetByID(anilistID)
				time.Sleep(1 * time.Second)
				if fetchErr != nil {
					logger.Error("anilist fetch failed for %s (anilist %d): %s", m.Title, anilistID, fetchErr.Error())
				} else {
					id := strconv.Itoa(full.ID)
					h.db.Exec(
						`UPDATE media_items SET
						external_id = COALESCE(external_id, $1),
						external_source = COALESCE(external_source, $2)
						WHERE id = $3`,
						id, "anilist", mediaItemID,
					)
					_, err = h.db.Exec(
						`INSERT INTO movie_metadata (media_item_id, genres)
						VALUES ($1, $2)
						ON CONFLICT (media_item_id) DO UPDATE SET
						genres = EXCLUDED.genres`,
						mediaItemID, pq.Array(full.Genres),
					)
					if err != nil {
						logger.Error("failed to upsert movie metadata for %s: %s", m.Title, err.Error())
					}
				}
			}
		}

		// Always upsert radarr_items regardless of file status
		_, err = h.db.Exec(
			`INSERT INTO radarr_items (media_item_id, radarr_movie_id)
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO UPDATE SET
			radarr_movie_id = EXCLUDED.radarr_movie_id,
			last_synced_at = NOW()`,
			mediaItemID, m.ID,
		)
		if err != nil {
			logger.Error("failed to update or insert radarr_items for %s: %s", m.Title, err.Error())
			continue
		}

		if !m.HasFile {
			continue
		}

		_, err = h.db.Exec(
			`UPDATE movie_metadata SET file_path = $1 WHERE media_item_id = $2`,
			m.MovieFile.Path, mediaItemID,
		)
		if err != nil {
			logger.Error("failed to update file path for %s: %s", m.Title, err.Error())
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SyncManga(w http.ResponseWriter, r *http.Request) {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	mangaRoot := filepath.Join(mediaRoot, "Manga")

	dirs, err := os.ReadDir(mangaRoot)
	if utils.InternalError(w, err) {
		return
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		// Convert folder name to a search query
		query := strings.ReplaceAll(dir.Name(), "_", " ")
		results, err := h.mangadex.Search(query)
		time.Sleep(500 * time.Millisecond)
		if err != nil {
			logger.Error("mangadex search failed for %s: %s", query, err.Error())
			continue
		}
		if len(results) == 0 {
			logger.Warn("no mangadex results for %s, skipping", query)
			continue
		}

		manga := results[0]

		// Extract title with fallback
		title := manga.Attributes.Title.En
		if title == "" {
			title = manga.Attributes.Title.JaRo
		}
		if title == "" {
			title = manga.Attributes.Title.Ja
		}

		description := manga.Attributes.Description["en"]

		// Extract cover URL
		var coverURL string
		for _, rel := range manga.Relationships {
			if rel.Type == "cover_art" {
				coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", manga.ID, rel.Attributes.FileName)
				break
			}
		}

		// Extract genres and status
		var genres []string
		for _, tag := range manga.Attributes.Tags {
			if tag.Attributes.Group == "genre" {
				genres = append(genres, tag.Attributes.Name.En)
			}
		}
		status := manga.Attributes.Status

		var totalChapters int
		if n, convErr := strconv.Atoi(manga.Attributes.LastChapter); convErr == nil {
			totalChapters = n
		}

		// Insert or find the media item
		var mediaItemID string
		err = h.db.QueryRow(
			`INSERT INTO media_items (type, title, cover_image_url, external_id, external_source, description)
			VALUES ('manga', $1, $2, $3, 'mangadex', $4)
			ON CONFLICT (external_id, external_source) DO NOTHING
			RETURNING id`,
			title, utils.NullString(coverURL), manga.ID, description,
		).Scan(&mediaItemID)

		if err != nil {
			// Row already existed, fetch the ID
			err = h.db.QueryRow(
				`SELECT id FROM media_items WHERE external_id = $1 AND external_source = 'mangadex'`,
				manga.ID,
			).Scan(&mediaItemID)
			if err != nil {
				logger.Error("failed to find existing media item for %s: %s", title, err.Error())
				continue
			}

			_, err = h.db.Exec(
				`UPDATE media_items SET description = COALESCE(description, $1) WHERE id = $2`,
				description, mediaItemID,
			)
			if err != nil {
				logger.Error("unable to add description for %s: %s", title, err.Error())
				continue
			}
		}

		// Insert manga metadata
		_, err = h.db.Exec(
			`INSERT INTO manga_metadata (media_item_id, total_chapters, genres, status)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO NOTHING`,
			mediaItemID, utils.NullInt(&totalChapters), pq.Array(genres), utils.NullString(status),
		)
		if err != nil {
			logger.Error("failed to insert manga metadata for %s: %s", title, err.Error())
		}

		// Walk CBZ files and insert chapters
		mangaDir := filepath.Join(mangaRoot, dir.Name())
		filepath.WalkDir(mangaDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".cbz" {
				return nil
			}

			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			chapterNum := 0.0
			if strings.HasPrefix(base, "[") {
				end := strings.Index(base, "]")
				if end > 1 {
					chapterNum, _ = strconv.ParseFloat(strings.TrimSpace(base[1:end]), 64)
				}
			}

			zr, zErr := zip.OpenReader(path)
			pageCount := 0
			if zErr == nil {
				for _, f := range zr.File {
					ext := strings.ToLower(filepath.Ext(f.Name))
					if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
						pageCount++
					}
				}
				zr.Close()
			}

			h.db.Exec(
				`INSERT INTO manga_chapters (media_item_id, chapter_number, file_path, page_count)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (media_item_id, chapter_number) DO NOTHING`,
				mediaItemID, chapterNum, path, pageCount,
			)
			return nil
		})
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SyncLightNovel(w http.ResponseWriter, r *http.Request) {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	regex := regexp.MustCompile(`(?i)volume[_\s](\d+)`)
	dirListing, err := os.ReadDir(filepath.Join(mediaRoot, "Light Novels"))
	if utils.InternalError(w, err) {
		return
	}

	for _, entry := range dirListing {
		if !entry.IsDir() {
			continue
		}

		files, err := os.ReadDir(filepath.Join(mediaRoot, "Light Novels", entry.Name()))
		if utils.InternalError(w, err) {
			return
		}

		// Convert folder name to a search query
		query := strings.ReplaceAll(entry.Name(), "_", " ")
		results, err := h.anilist.Search("MANGA", query, 1, "NOVEL")
		time.Sleep(500 * time.Millisecond)
		if err != nil {
			logger.Error("anilist search failed for %s: %s", query, err.Error())
			continue
		}
		if len(results) == 0 {
			logger.Warn("no anilist results for %s, skipping", query)
			continue
		}

		lightNovel, err := h.anilist.GetByID(results[0].ID)
		time.Sleep(500 * time.Millisecond)
		if err != nil {
			logger.Error("anilist search failed for %s: %s", query, err.Error())
			continue
		}
		if lightNovel == nil {
			logger.Warn("no anilist results for %s, skipping", query)
			continue
		}

		var mediaItemID string
		err = h.db.QueryRow(
			`SELECT id FROM media_items WHERE type = 'light_novel' AND title = $1`,
			entry.Name(),
		).Scan(&mediaItemID)

		if err == sql.ErrNoRows {
			var startDate *string

			if lightNovel.StartDate.Year != nil {
				date := fmt.Sprintf("%04d-%02d-%02d",
					*lightNovel.StartDate.Year,
					utils.SafeMonth(lightNovel.StartDate.Month),
					utils.SafeDay(lightNovel.StartDate.Day),
				)
				startDate = &date
			}

			err = h.db.QueryRow(
				`INSERT INTO media_items (type, title, description, cover_image_url, external_id, external_source, release_date)
				 VALUES ('light_novel', $1, $2, $3, $4, $5, $6) RETURNING id`,
				entry.Name(), lightNovel.Description, lightNovel.CoverImage.Large, strconv.Itoa(lightNovel.ID), "anilist", startDate,
			).Scan(&mediaItemID)
			if err != nil {
				logger.Warn("failed to insert media item for %s: %v", entry.Name(), err)
				continue
			}
		} else if err == nil {
			id := strconv.Itoa(lightNovel.ID)
			var releaseDate *string
			if lightNovel.StartDate.Year != nil {
				date := fmt.Sprintf("%04d-%02d-%02d",
					*lightNovel.StartDate.Year,
					utils.SafeMonth(lightNovel.StartDate.Month),
					utils.SafeDay(lightNovel.StartDate.Day),
				)
				releaseDate = &date
			}
			h.db.Exec(
				`UPDATE media_items SET
				external_id = COALESCE(external_id, $1),
				external_source = COALESCE(external_source, $2),
				description = COALESCE(description, $3),
				cover_image_url = COALESCE(cover_image_url, $4),
				release_date = COALESCE(release_date, $5)
				WHERE id = $6`,
				id, "anilist", lightNovel.Description, lightNovel.CoverImage.Large, releaseDate, mediaItemID,
			)
		} else if err != nil {
			logger.Warn("failed to query media item for %s: %v", entry.Name(), err)
			continue
		}

		_, err = h.db.Exec(
			`INSERT INTO light_novel_metadata (media_item_id, genres)
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO NOTHING`,
			mediaItemID, pq.Array(lightNovel.Genres),
		)
		if err != nil {
			logger.Warn("failed to insert or update metadata for %s: %v", entry.Name(), err)
			continue
		}

		for _, file := range files {
			if filepath.Ext(file.Name()) != ".epub" {
				continue
			}
			match := regex.FindStringSubmatch(file.Name())
			if match == nil {
				logger.Warn("File %s does not include volume number in the name", file.Name())
				continue
			}

			volumeNum, _ := strconv.Atoi(match[1])

			filePath := filepath.Join(mediaRoot, "Light Novels", entry.Name(), file.Name())
			var volumeID string
			err = h.db.QueryRow(
				`INSERT INTO light_novel_volumes (media_item_id, volume_number, title, file_path)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (media_item_id, volume_number) DO UPDATE SET file_path = EXCLUDED.file_path
				RETURNING id`,
				mediaItemID, volumeNum, file.Name(), filePath,
			).Scan(&volumeID)
			if err != nil {
				logger.Warn("failed to insert or update volume for %s: %v", file.Name(), err)
				continue
			}

			if volumeNum == 1 {
				coverFile, err := extractCoverHref(filePath)
				if err != nil {
					logger.Warn("failed to extract cover for %s: %v", file.Name(), err)
				} else if coverFile != "" {
					coverURL := fmt.Sprintf("%s/light-novels/%s/volumes/%s/images/%s",
						os.Getenv("API_URL"), mediaItemID, volumeID, coverFile)
					h.db.Exec(
						`UPDATE media_items SET cover_image_url = $1 WHERE id = $2 AND cover_image_url IS NULL`,
						coverURL, mediaItemID,
					)
				}
			}

		}

	}

	w.WriteHeader(http.StatusOK)
}

func extractCoverHref(filePath string) (string, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	// step 1: find OPF path from container.xml
	opfPath := ""
	for _, f := range zr.File {
		if f.Name != "META-INF/container.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		pathRe := regexp.MustCompile(`full-path="([^"]+)"`)
		m := pathRe.FindSubmatch(content)
		if m != nil {
			opfPath = string(m[1])
		}
		break
	}
	if opfPath == "" {
		return "", nil
	}

	// step 2: read the OPF
	for _, f := range zr.File {
		if f.Name != opfPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}

		opf := string(content)

		// try properties="cover-image" first (EPUB3 standard)
		coverImageRe := regexp.MustCompile(`<item[^>]+properties="cover-image"[^>]+href="([^"]+)"`)
		if m := coverImageRe.FindStringSubmatch(opf); m != nil {
			return filepath.Base(m[1]), nil
		}
		// also try href before properties
		coverImageRe2 := regexp.MustCompile(`<item[^>]+href="([^"]+)"[^>]+properties="cover-image"`)
		if m := coverImageRe2.FindStringSubmatch(opf); m != nil {
			return filepath.Base(m[1]), nil
		}

		// try name first then content
		metaRe2 := regexp.MustCompile(`(?i)<meta\s+name="cover"\s+content="([^"]+)"`)
		metaRe3 := regexp.MustCompile(`(?i)<meta\s+content="([^"]+)"\s+name="cover"`)

		var coverID string
		if m := metaRe2.FindStringSubmatch(opf); m != nil {
			coverID = m[1]
		} else if m := metaRe3.FindStringSubmatch(opf); m != nil {
			coverID = m[1]
		}

		if coverID != "" {
			itemRe := regexp.MustCompile(`<item\s+[^>]*id="` + regexp.QuoteMeta(coverID) + `"[^>]+href="([^"]+)"`)
			itemRe2 := regexp.MustCompile(`<item\s+[^>]*href="([^"]+)"[^>]+id="` + regexp.QuoteMeta(coverID) + `"`)
			if m := itemRe.FindStringSubmatch(opf); m != nil {
				return filepath.Base(m[1]), nil
			}
			if m := itemRe2.FindStringSubmatch(opf); m != nil {
				return filepath.Base(m[1]), nil
			}
		}

		return "", nil
	}

	return "", nil
}

func (h *Handler) SyncMusic(w http.ResponseWriter, r *http.Request) {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	musicRoot := filepath.Join(mediaRoot, "Music")

	if _, err := os.Stat(musicRoot); os.IsNotExist(err) {
		utils.Error(w, http.StatusNotFound, "music directory not found")
		return
	}

	synced := 0

	err := filepath.WalkDir(musicRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".mp3" {
			return nil
		}

		// derive artist and album from directory structure: Music/Artist/Album/track.mp3
		rel, err := filepath.Rel(musicRoot, path)
		if err != nil {
			logger.Warn("failed to get relative path for %s: %s", path, err.Error())
			return nil
		}

		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 3 {
			logger.Warn("unexpected path structure for %s, skipping", path)
			return nil
		}

		artist := parts[0]
		albumName := parts[1]
		fileName := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))

		// upsert media_item
		var mediaItemID string
		err = h.db.QueryRow(
			`SELECT id FROM media_items WHERE title = $1 AND type = 'music_track'`,
			fileName,
		).Scan(&mediaItemID)

		if err == sql.ErrNoRows {
			err = h.db.QueryRow(
				`INSERT INTO media_items (type, title)
				VALUES ('music_track', $1)
				RETURNING id`,
				fileName,
			).Scan(&mediaItemID)
			if err != nil {
				logger.Error("failed to create media item for %s: %s", fileName, err.Error())
				return nil
			}
		} else if err != nil {
			logger.Error("failed to query media item for %s: %s", fileName, err.Error())
			return nil
		}

		// upsert album if not "Singles"
		var albumID *string
		if albumName != "Singles" {
			var id string
			err = h.db.QueryRow(
				`SELECT id FROM albums WHERE title = $1 AND artist = $2`,
				albumName, artist,
			).Scan(&id)

			if err == sql.ErrNoRows {
				err = h.db.QueryRow(
					`INSERT INTO albums (title, artist)
					VALUES ($1, $2)
					RETURNING id`,
					albumName, artist,
				).Scan(&id)
				if err != nil {
					logger.Error("failed to create album %s: %s", albumName, err.Error())
					return nil
				}
			} else if err != nil {
				logger.Error("failed to query album %s: %s", albumName, err.Error())
				return nil
			}
			albumID = &id
		}

		// upsert music_metadata with file_path
		_, err = h.db.Exec(
			`INSERT INTO music_metadata (media_item_id, artist, album_id, file_path)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id) DO UPDATE SET
			artist = EXCLUDED.artist,
			album_id = EXCLUDED.album_id,
			file_path = EXCLUDED.file_path`,
			mediaItemID, artist, albumID, path,
		)
		if err != nil {
			logger.Error("failed to upsert music_metadata for %s: %s", fileName, err.Error())
			return nil
		}

		synced++

		coverPath := filepath.Join(filepath.Dir(path), "cover.jpg")
		if _, coverErr := os.Stat(coverPath); os.IsNotExist(coverErr) {
			extractCoverArt(path, coverPath)
		}

		// Set cover_image_url if cover exists
		if _, coverErr := os.Stat(coverPath); coverErr == nil {
			apiURL := os.Getenv("API_URL")
			coverURL := fmt.Sprintf("%s/stream/music/%s/cover", apiURL, mediaItemID)
			h.db.Exec(
				`UPDATE media_items SET cover_image_url = $1 WHERE id = $2 AND cover_image_url IS NULL`,
				coverURL, mediaItemID,
			)
		}

		if albumID != nil {
			if _, coverErr := os.Stat(coverPath); coverErr == nil {
				apiURL := os.Getenv("API_URL")
				coverURL := fmt.Sprintf("%s/stream/music/%s/cover", apiURL, mediaItemID)
				h.db.Exec(
					`UPDATE albums SET cover_image_url = $1 WHERE id = $2 AND cover_image_url IS NULL`,
					coverURL, *albumID,
				)
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("failed to walk music directory: %s", err.Error())
		utils.Error(w, http.StatusInternalServerError, "sync failed")
		return
	}

	logger.Info("Music sync complete: %d tracks synced", synced)
	utils.JSON(w, map[string]int{"synced": synced})
}

func extractCoverArt(mp3Path string, destPath string) {
	f, err := os.Open(mp3Path)
	if err != nil {
		return
	}
	defer f.Close()

	meta, err := tag.ReadFrom(f)
	if err != nil {
		return
	}

	pic := meta.Picture()
	if pic == nil || len(pic.Data) == 0 {
		return
	}

	os.WriteFile(destPath, pic.Data, 0644)
}
