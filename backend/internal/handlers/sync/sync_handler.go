package sync

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/locks"
	"github.com/Mantie7553/MediaHub/backend/internal/mangafiles"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/novelfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/lib/pq"
)

type Handler struct {
	db       *sql.DB
	sonarr   *arr.ArrClient
	radarr   *arr.ArrClient
	mangadex *mangadex.MangaDexClient
	anilist  *anilist.AnilistClient
	runner   runner
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

func (h *Handler) syncSonarr() error {
	series, err := h.sonarr.GetAllSeries()
	if err != nil {
		return err
	}

	tvdbToAnilist, _, seasonToMal, err := arr.FetchAnimeMaps()
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

		if errors.Is(err, sql.ErrNoRows) {
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
				`+locks.Keep(locks.Title, "title", "$1")+`,
				`+locks.Keep(locks.Cover, "cover_image_url", "$2")+`,
				`+locks.Keep(locks.Description, "description", "COALESCE(description, $3)")+`,
				`+locks.Keep(locks.ReleaseDate, "release_date", "COALESCE(release_date, $4)")+`
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
					utils.DerefOr(aniResult.StartDate.Month, 1),
					utils.DerefOr(aniResult.StartDate.Day, 1),
				)
				h.db.Exec(
					`UPDATE media_items SET `+locks.Keep(locks.ReleaseDate, "release_date", "$1")+` WHERE id = $2`,
					date, mediaItemID,
				)
			}

			status := aniResult.LibraryStatus()

			var studio string
			if len(aniResult.Studios) > 0 {
				studio = aniResult.Studios[0]
			}

			_, err = h.db.Exec(
				`INSERT INTO anime_metadata (media_item_id, studio, status, genres)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (media_item_id) DO UPDATE SET
				`+locks.KeepMeta(locks.Studio, "anime_metadata", "studio", "EXCLUDED.studio")+`,
				`+locks.KeepMeta(locks.Status, "anime_metadata", "status", "EXCLUDED.status")+`,
				`+locks.KeepMeta(locks.Genres, "anime_metadata", "genres", "EXCLUDED.genres"),
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
				`+locks.KeepMeta(locks.Genres, "anime_metadata", "genres", "EXCLUDED.genres"),
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
				title = CASE WHEN episodes.title_locked THEN episodes.title ELSE EXCLUDED.title END,
				sonarr_id = EXCLUDED.sonarr_id,
				updated_at = NOW()`,
				mediaItemID, ep.SeasonNumber, ep.EpisodeNumber, ep.Title, filePath, ep.ID,
			)
			if err != nil {
				logger.Error("failed to update or insert episode %d: %s", ep.ID, err.Error())
			}
		}

		h.saveSeasonMal(mediaItemID, s.TvdbID, episodes, seasonToMal)
	}
	return nil
}

/*
Function:	saveSeasonMal
Purpose:	save the MyAnimeList entry of each of the series' seasons, used to look up skip times
Params:
  - mediaItemID: the series
  - tvdbID: the series' TVDB id
  - episodes: the series' episodes from Sonarr
  - seasonToMal: TVDB season to MyAnimeList id, from Fribb's list
*/
func (h *Handler) saveSeasonMal(mediaItemID string, tvdbID int, episodes []arr.SonarrEpisode, seasonToMal map[arr.TvdbSeason]int) {
	if tvdbID <= 0 {
		return
	}
	seen := map[int]bool{}
	for _, ep := range episodes {
		if seen[ep.SeasonNumber] {
			continue
		}
		seen[ep.SeasonNumber] = true
		malID, ok := seasonToMal[arr.TvdbSeason{TvdbID: tvdbID, Season: ep.SeasonNumber}]
		if !ok {
			continue
		}
		if _, err := h.db.Exec(
			`INSERT INTO anime_season_mal (media_item_id, season_number, mal_id) VALUES ($1, $2, $3)
			ON CONFLICT (media_item_id, season_number) DO UPDATE SET mal_id = EXCLUDED.mal_id`,
			mediaItemID, ep.SeasonNumber, malID,
		); err != nil {
			logger.Error("failed to save MyAnimeList id of season %d of %s: %s", ep.SeasonNumber, mediaItemID, err.Error())
		}
	}
}

func (h *Handler) syncRadarr() error {
	movies, err := h.radarr.GetAllMovies()
	if err != nil {
		return err
	}

	_, tmdbToAnilist, _, err := arr.FetchAnimeMaps()
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

		if errors.Is(err, sql.ErrNoRows) {
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
				`+locks.Keep(locks.Title, "title", "$1")+`,
				`+locks.Keep(locks.Cover, "cover_image_url", "$2")+`,
				`+locks.Keep(locks.Description, "description", "COALESCE(description, $3)")+`,
				`+locks.Keep(locks.ReleaseDate, "release_date", "COALESCE(release_date, $4)")+`
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
			`+locks.KeepMeta(locks.Genres, "movie_metadata", "genres", "EXCLUDED.genres"),
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
						`+locks.KeepMeta(locks.Genres, "movie_metadata", "genres", "EXCLUDED.genres"),
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
	return nil
}

func (h *Handler) syncManga() error {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	mangaRoot := filepath.Join(mediaRoot, "Manga")

	dirs, err := os.ReadDir(mangaRoot)
	if err != nil {
		return err
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

		genres := manga.Genres()
		status := manga.Attributes.Status
		totalChapters := manga.TotalChapters()

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

			// a cover an admin handed back to the sync is NULL, and filled in again here
			_, err = h.db.Exec(
				`UPDATE media_items SET
				`+locks.Keep(locks.Description, "description", "COALESCE(description, $1)")+`,
				`+locks.Keep(locks.Cover, "cover_image_url", "COALESCE(cover_image_url, $3)")+`
				WHERE id = $2`,
				description, mediaItemID, utils.NullString(coverURL),
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

		// Walk the chapter archives and add the ones the library doesn't have
		mangaDir := filepath.Join(mangaRoot, dir.Name())
		filepath.WalkDir(mangaDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !mangafiles.IsArchive(path) {
				return nil
			}

			chapter, ok := mangafiles.ParseChapter(d.Name(), title)
			if !ok {
				// whole volumes can't be told apart from chapters by number
				logger.Warn("no chapter number in %s, skipping it", path)
				return nil
			}

			// a file already in the library keeps the number it has there, which an admin may
			// have corrected
			var known bool
			h.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM manga_chapters WHERE file_path = $1)`, path).Scan(&known)
			if known {
				return nil
			}
			h.db.Exec(
				`INSERT INTO manga_chapters (media_item_id, chapter_number, title, file_path, page_count, created_at)
				VALUES ($1, $2, NULLIF($3, ''), $4, $5, NOW())
				ON CONFLICT (media_item_id, chapter_number) DO NOTHING`,
				mediaItemID, chapter.Value, chapter.Title, path, mangafiles.PageCount(path),
			)
			return nil
		})
	}

	return nil
}

func (h *Handler) syncLightNovels() error {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	dirListing, err := os.ReadDir(filepath.Join(mediaRoot, "Light Novels"))
	if err != nil {
		return err
	}

	for _, entry := range dirListing {
		if !entry.IsDir() {
			continue
		}

		files, err := os.ReadDir(filepath.Join(mediaRoot, "Light Novels", entry.Name()))
		if err != nil {
			return err
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

		// the title that owns volumes in this folder, which may have been renamed since, or one
		// named after the folder
		folder := filepath.Join(mediaRoot, "Light Novels", entry.Name())
		var mediaItemID string
		err = h.db.QueryRow(
			`SELECT media_item_id FROM light_novel_volumes WHERE file_path LIKE $1 ESCAPE '\' LIMIT 1`,
			utils.LikePrefix(folder+string(filepath.Separator)),
		).Scan(&mediaItemID)
		if errors.Is(err, sql.ErrNoRows) {
			err = h.db.QueryRow(
				`SELECT id FROM media_items WHERE type = 'light_novel' AND title = $1`,
				entry.Name(),
			).Scan(&mediaItemID)
		}

		if errors.Is(err, sql.ErrNoRows) {
			var startDate *string

			if lightNovel.StartDate.Year != nil {
				date := fmt.Sprintf("%04d-%02d-%02d",
					*lightNovel.StartDate.Year,
					utils.DerefOr(lightNovel.StartDate.Month, 1),
					utils.DerefOr(lightNovel.StartDate.Day, 1),
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
					utils.DerefOr(lightNovel.StartDate.Month, 1),
					utils.DerefOr(lightNovel.StartDate.Day, 1),
				)
				releaseDate = &date
			}
			h.db.Exec(
				`UPDATE media_items SET
				external_id = COALESCE(external_id, $1),
				external_source = COALESCE(external_source, $2),
				`+locks.Keep(locks.Description, "description", "COALESCE(description, $3)")+`,
				`+locks.Keep(locks.Cover, "cover_image_url", "COALESCE(cover_image_url, $4)")+`,
				`+locks.Keep(locks.ReleaseDate, "release_date", "COALESCE(release_date, $5)")+`
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
			volumeNum, ok := novelfiles.VolumeNumber(file.Name())
			if !ok {
				logger.Warn("File %s does not include volume number in the name", file.Name())
				continue
			}

			filePath := filepath.Join(folder, file.Name())
			// a file already in the library keeps the number it has there, which an admin may
			// have corrected
			var known bool
			h.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM light_novel_volumes WHERE file_path = $1)`, filePath).Scan(&known)
			if known {
				continue
			}
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
				coverFile, err := novelfiles.CoverHref(filePath)
				if err != nil {
					logger.Warn("failed to extract cover for %s: %v", file.Name(), err)
				} else if coverFile != "" {
					coverURL := fmt.Sprintf("/api/light-novels/%s/volumes/%s/images/%s",
						mediaItemID, volumeID, coverFile)
					h.db.Exec(
						`UPDATE media_items SET cover_image_url = $1 WHERE id = $2 AND cover_image_url IS NULL`,
						coverURL, mediaItemID,
					)
				}
			}

		}

	}

	return nil
}

func (h *Handler) syncMusic() (int, error) {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	musicRoot := filepath.Join(mediaRoot, "Music")

	if _, err := os.Stat(musicRoot); os.IsNotExist(err) {
		return 0, fmt.Errorf("music directory not found: %s", musicRoot)
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

		// a track already in the library by this file keeps its title, artist and album, which
		// an admin may have changed: only its cover is looked at below
		var mediaItemID string
		var knownAlbum *string
		err = h.db.QueryRow(
			`SELECT media_item_id, album_id FROM music_metadata WHERE file_path = $1`, path,
		).Scan(&mediaItemID, &knownAlbum)
		if err == nil {
			synced++
			h.syncTrackCover(mediaItemID, path, knownAlbum)
			return nil
		}

		// upsert media_item
		err = h.db.QueryRow(
			`SELECT id FROM media_items WHERE title = $1 AND type = 'music_track'`,
			fileName,
		).Scan(&mediaItemID)

		if errors.Is(err, sql.ErrNoRows) {
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

		// upsert the album, Singles included: an artist's singles are listed as an album too
		var albumID *string
		// the album the folder's other tracks are on, which may have been renamed since, or
		// one named after the folder
		var id string
		err = h.db.QueryRow(
			`SELECT album_id FROM music_metadata WHERE album_id IS NOT NULL AND file_path LIKE $1 ESCAPE '\' LIMIT 1`,
			utils.LikePrefix(filepath.Join(musicRoot, artist, albumName)+string(filepath.Separator)),
		).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			err = h.db.QueryRow(
				`SELECT id FROM albums WHERE title = $1 AND artist = $2`,
				albumName, artist,
			).Scan(&id)
		}

		if errors.Is(err, sql.ErrNoRows) {
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
		h.syncTrackCover(mediaItemID, path, albumID)
		return nil
	})

	if err != nil {
		return synced, fmt.Errorf("failed to walk music directory: %w", err)
	}

	logger.Info("Music sync complete: %d tracks synced", synced)
	return synced, nil
}

/*
Function:	syncTrackCover
Purpose:	give a track, and its album, the cover.jpg in the track's folder when they have no
cover yet, taking it from the track's embedded art if the folder has none
Params:
  - mediaItemID: the track
  - path: its file
  - albumID: its album, nil for a single
*/
func (h *Handler) syncTrackCover(mediaItemID, path string, albumID *string) {
	coverPath := musicfiles.CoverPath(path)
	_, coverErr := os.Stat(coverPath)
	if os.IsNotExist(coverErr) && musicfiles.ExtractCover(path, coverPath) {
		coverErr = nil
	}
	if coverErr != nil {
		return
	}

	// tracks and albums keep a cover they already have, such as an uploaded one
	coverURL := musicfiles.CoverURL(mediaItemID)
	h.db.Exec(
		`UPDATE media_items SET cover_image_url = $1 WHERE id = $2 AND cover_image_url IS NULL`,
		coverURL, mediaItemID,
	)
	if albumID != nil {
		h.db.Exec(
			`UPDATE albums SET cover_image_url = $1 WHERE id = $2 AND cover_image_url IS NULL`,
			coverURL, *albumID,
		)
	}
}
