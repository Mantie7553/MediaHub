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
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

type Handler struct {
	db     *sql.DB
	sonarr *arr.ArrClient
	radarr *arr.ArrClient
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:     db,
		sonarr: arr.NewArrClient("SONARR_URL", "SONARR_API_KEY"),
		radarr: arr.NewArrClient("RADARR_URL", "RADARR_API_KEY"),
	}
}

func (h *Handler) SyncSonar(w http.ResponseWriter, r *http.Request) {
	series, err := h.sonarr.GetAllSeries()
	if utils.InternalError(w, err) {
		return
	}

	for _, s := range series {
		var mediaItemID string
		err := h.db.QueryRow(
			`SELECT id FROM media_items WHERE title = $1 AND type = 'anime'`,
			s.Title,
		).Scan(&mediaItemID)

		if err == sql.ErrNoRows {
			posterURL := s.PosterURL()
			var externalID *string
			aniClient := anilist.NewAnilistClient("")
			results, searchErr := aniClient.Search("ANIME", s.Title, 1, "TV")
			time.Sleep(500 * time.Millisecond)
			if searchErr == nil && len(results) > 0 {
				id := strconv.Itoa(results[0].ID)
				externalID = &id
			}

			err = h.db.QueryRow(
				`INSERT INTO media_items (type, title, cover_image_url, external_id, external_source)
				VALUES ('anime', $1, $2, $3, $4)
				RETURNING id`,
				s.Title, posterURL, externalID, utils.NullString("anilist"),
			).Scan(&mediaItemID)
			if err != nil {
				logger.Error("failed to create media item for %s: %s", s.Title, err.Error())
				continue
			}
		} else if err == nil {
			// backfill external_id if missing
			aniClient := anilist.NewAnilistClient("")
			results, searchErr := aniClient.Search("ANIME", s.Title, 1, "TV")
			time.Sleep(500 * time.Millisecond)
			if searchErr == nil && len(results) > 0 {
				id := strconv.Itoa(results[0].ID)
				h.db.Exec(
					`UPDATE media_items SET external_id = COALESCE(external_id, $1), external_source = COALESCE(external_source, $2) WHERE id = $3`,
					id, "anilist", mediaItemID,
				)
			}
		} else {
			logger.Error("failed to find media item for %s : %s", s.Title, err.Error())
			continue
		}

		_, err = h.db.Exec(
			`UPDATE media_items SET cover_image_url = $1 WHERE id = $2`,
			utils.NullString(s.PosterURL()), mediaItemID,
		)
		if err != nil {
			logger.Error("failed to update cover image for %s: %s", s.Title, err.Error())
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
			logger.Error("failed to get episodes fro %s : %s", s.Title, err.Error())
			continue
		}

		for _, ep := range episodes {
			if !ep.HasFile {
				continue
			}

			filePath, err := h.sonarr.GetEpisodeFilePath(ep.EpisodeFileID)
			if err != nil {
				logger.Error("failed to get file path for episode %d : %s", ep.ID, err.Error())
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
				logger.Error("failed to update or insert episode %d : %s", ep.ID, err.Error())
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

	for _, m := range movies {
		var mediaItemID string
		err := h.db.QueryRow(
			`SELECT id FROM media_items WHERE title = $1 AND type = 'movie'`,
			m.Title,
		).Scan(&mediaItemID)

		if err == sql.ErrNoRows {
			posterURL := m.PosterURL()
			err = h.db.QueryRow(
				`INSERT INTO media_items (type, title, cover_image_url)
    			VALUES ('movie', $1, $2)
    			RETURNING id`,
				m.Title, utils.NullString(posterURL),
			).Scan(&mediaItemID)
			if err != nil {
				logger.Error("failed to create media item for %s: %s", m.Title, err.Error())
				continue
			}
			_, err = h.db.Exec(
				`INSERT INTO movie_metadata (media_item_id) VALUES ($1)`,
				mediaItemID,
			)
			if err != nil {
				logger.Error("failed to create movie metadata for %s: %s", m.Title, err.Error())
				continue
			}
		}
		_, err = h.db.Exec(
			`UPDATE media_items SET cover_image_url = $1 WHERE id = $2`,
			utils.NullString(m.PosterURL()), mediaItemID,
		)
		if err != nil {
			logger.Error("failed to update cover image for %s: %s", m.Title, err.Error())
		}
		if date := m.ReleaseDate(); date != "" {
			_, err = h.db.Exec(
				`UPDATE media_items SET release_date = $1 WHERE id = $2`,
				date, mediaItemID,
			)
			if err != nil {
				logger.Error("failed to update release_date for %s: %s", m.Title, err.Error())
			}
		}
		if !m.HasFile {
			continue
		}

		_, err = h.db.Exec(
			`INSERT INTO radarr_items (media_item_id, radarr_movie_id) 
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO UPDATE SET
			radarr_movie_id = EXCLUDED.radarr_movie_id,
			last_synced_at = NOW()`,
			mediaItemID, m.ID,
		)
		if err != nil {
			logger.Error("failed to update or insert sonarr_items for %s : %s", m.Title, err.Error())
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

	rows, err := h.db.Query(`SELECT id, title FROM media_items WHERE type = 'manga'`)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mediaItemID, title string
		if err := rows.Scan(&mediaItemID, &title); err != nil {
			logger.Error("failed to scan manga row: %s", err.Error())
			continue
		}

		folderName := strings.ReplaceAll(strings.ReplaceAll(title, ": ", "_"), " ", "_")
		mangaDir := filepath.Join(mangaRoot, folderName)
		if _, err := os.Stat(mangaDir); os.IsNotExist(err) {
			logger.Warn("manga directory not found for %s, skipping", title)
			continue
		}

		err = filepath.WalkDir(mangaDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
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

			r, err := zip.OpenReader(path)
			pageCount := 0
			if err == nil {
				for _, f := range r.File {
					ext := strings.ToLower(filepath.Ext(f.Name))
					if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
						pageCount++
					}
				}
				r.Close()
			}

			h.db.Exec(
				`INSERT INTO manga_chapters (media_item_id, chapter_number, file_path, page_count)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT DO NOTHING`,
				mediaItemID, chapterNum, path, pageCount,
			)
			return nil
		})

		if err != nil {
			logger.Error("failed to walk manga directory for %s: %s", title, err.Error())
		}
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

			var mediaItemID string
			err = h.db.QueryRow(
				`SELECT id FROM media_items WHERE type = 'light_novel' AND title = $1`,
				entry.Name(),
			).Scan(&mediaItemID)

			if err == sql.ErrNoRows {
				err = h.db.QueryRow(
					`INSERT INTO media_items (type, title) VALUES ('light_novel', $1) RETURNING id`,
					entry.Name(),
				).Scan(&mediaItemID)
				if err != nil {
					logger.Warn("failed to insert media item for %s: %v", entry.Name(), err)
					continue
				}
			} else if err != nil {
				logger.Warn("failed to query media item for %s: %v", entry.Name(), err)
				continue
			}

			_, err = h.db.Exec(
				`INSERT INTO light_novel_metadata (media_item_id)
				VALUES ($1)
				ON CONFLICT (media_item_id) DO NOTHING`,
				mediaItemID,
			)
			if err != nil {
				logger.Warn("failed to insert or update metadata for %s: %v", entry.Name(), err)
				continue
			}

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
						`UPDATE media_items SET cover_image_url = $1 WHERE id = $2`,
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
