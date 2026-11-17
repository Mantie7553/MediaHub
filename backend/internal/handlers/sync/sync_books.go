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
	"github.com/Mantie7553/MediaHub/backend/internal/locks"
	"github.com/Mantie7553/MediaHub/backend/internal/mangafiles"
	"github.com/Mantie7553/MediaHub/backend/internal/novelfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
	"github.com/lib/pq"
)

// pause after each MangaDex request, and after AniList requests of the light novel sync
const bookSearchPause = 500 * time.Millisecond

/*
Function:	syncManga
Purpose:	add the chapters in each Manga folder. A folder already in the library, with its cover
and description, is only walked for new chapters; others are matched on MangaDex by name
*/
func (h *Handler) syncManga() error {
	mangaRoot := filepath.Join(os.Getenv("MEDIA_ROOT"), "Manga")
	dirs, err := os.ReadDir(mangaRoot)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		mangaDir := filepath.Join(mangaRoot, dir.Name())

		// the title the folder's chapters belong to. Its name may have been changed since, so
		// chapter names are read against the folder's name, which they usually start with
		var mediaItemID string
		var complete bool
		title := strings.ReplaceAll(dir.Name(), "_", " ")
		err := h.db.QueryRow(
			`SELECT mi.id, mi.cover_image_url IS NOT NULL AND mi.description IS NOT NULL
			FROM manga_chapters mc JOIN media_items mi ON mi.id = mc.media_item_id
			WHERE mc.file_path LIKE $1 ESCAPE '\' LIMIT 1`,
			utils.LikePrefix(mangaDir+string(filepath.Separator)),
		).Scan(&mediaItemID, &complete)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Error("failed to look up the manga in %s: %s", dir.Name(), err.Error())
			continue
		}

		if !complete {
			mediaItemID, title, err = h.matchManga(dir.Name())
			if err != nil {
				logger.Warn("%s", err.Error())
				continue
			}
		}

		h.addChapters(mediaItemID, title, mangaDir)
		tracking.PartsAdded(h.db, mediaItemID)
	}
	return nil
}

/*
Function:	matchManga
Purpose:	find a Manga folder's title on MangaDex and add it to the library, or fill in the
description and cover of the one already there. Returns its id and MangaDex's title
Params:
  - folder: the folder's name
*/
func (h *Handler) matchManga(folder string) (mediaItemID, title string, err error) {
	query := strings.ReplaceAll(folder, "_", " ")
	results, err := h.mangadex.Search(query)
	time.Sleep(bookSearchPause)
	if err != nil {
		return "", "", fmt.Errorf("mangadex search failed for %s: %w", query, err)
	}
	if len(results) == 0 {
		return "", "", fmt.Errorf("no mangadex results for %s, skipping", query)
	}
	manga := results[0]

	title = manga.Attributes.Title.En
	if title == "" {
		title = manga.Attributes.Title.JaRo
	}
	if title == "" {
		title = manga.Attributes.Title.Ja
	}
	description := manga.Attributes.Description["en"]

	var coverURL string
	for _, rel := range manga.Relationships {
		if rel.Type == "cover_art" {
			coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", manga.ID, rel.Attributes.FileName)
			break
		}
	}
	totalChapters := manga.TotalChapters()

	err = h.db.QueryRow(
		`INSERT INTO media_items (type, title, cover_image_url, external_id, external_source, description)
		VALUES ('manga', $1, $2, $3, 'mangadex', $4)
		ON CONFLICT (external_id, external_source) DO NOTHING
		RETURNING id`,
		title, utils.NullString(coverURL), manga.ID, description,
	).Scan(&mediaItemID)

	if err != nil {
		// already in the library
		err = h.db.QueryRow(
			`SELECT id FROM media_items WHERE external_id = $1 AND external_source = 'mangadex'`,
			manga.ID,
		).Scan(&mediaItemID)
		if err != nil {
			return "", "", fmt.Errorf("failed to find existing media item for %s: %w", title, err)
		}

		// a cover an admin handed back to the sync is NULL, and filled in again here
		if _, err = h.db.Exec(
			`UPDATE media_items SET
			`+locks.Keep(locks.Description, "description", "COALESCE(description, $1)")+`,
			`+locks.Keep(locks.Cover, "cover_image_url", "COALESCE(cover_image_url, $3)")+`
			WHERE id = $2`,
			description, mediaItemID, utils.NullString(coverURL),
		); err != nil {
			return "", "", fmt.Errorf("unable to add description for %s: %w", title, err)
		}
	}

	if _, err := h.db.Exec(
		`INSERT INTO manga_metadata (media_item_id, total_chapters, genres, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (media_item_id) DO NOTHING`,
		mediaItemID, utils.NullInt(&totalChapters), pq.Array(manga.Genres()), utils.NullString(manga.Attributes.Status),
	); err != nil {
		logger.Error("failed to insert manga metadata for %s: %s", title, err.Error())
	}
	return mediaItemID, title, nil
}

/*
Function:	addChapters
Purpose:	walk a manga's folder and add the chapter archives the library doesn't have
Params:
  - mediaItemID: the manga
  - title: its title, which chapter names often start with
  - mangaDir: its folder
*/
func (h *Handler) addChapters(mediaItemID, title, mangaDir string) {
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

/*
Function:	syncLightNovels
Purpose:	add the volumes in each Light Novels folder. A folder already in the library keeps its
AniList match and only asks AniList again once its data is a week old
*/
func (h *Handler) syncLightNovels() error {
	novelsRoot := filepath.Join(os.Getenv("MEDIA_ROOT"), "Light Novels")
	dirListing, err := os.ReadDir(novelsRoot)
	if err != nil {
		return err
	}

	for _, entry := range dirListing {
		if !entry.IsDir() {
			continue
		}
		folder := filepath.Join(novelsRoot, entry.Name())
		files, err := os.ReadDir(folder)
		if err != nil {
			// one unreadable folder shouldn't stop the others
			logger.Error("failed to read %s: %s", folder, err.Error())
			continue
		}

		// the title that owns volumes in this folder, which may have been renamed since, or one
		// named after the folder
		var mediaItemID string
		var externalID, externalSource *string
		var syncedAt *time.Time
		err = h.db.QueryRow(
			`SELECT mi.id, mi.external_id, mi.external_source::text, mi.metadata_synced_at
			FROM light_novel_volumes v JOIN media_items mi ON mi.id = v.media_item_id
			WHERE v.file_path LIKE $1 ESCAPE '\' LIMIT 1`,
			utils.LikePrefix(folder+string(filepath.Separator)),
		).Scan(&mediaItemID, &externalID, &externalSource, &syncedAt)
		if errors.Is(err, sql.ErrNoRows) {
			err = h.db.QueryRow(
				`SELECT id, external_id, external_source::text, metadata_synced_at FROM media_items
				WHERE type = 'light_novel' AND title = $1`,
				entry.Name(),
			).Scan(&mediaItemID, &externalID, &externalSource, &syncedAt)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Warn("failed to query media item for %s: %v", entry.Name(), err)
			continue
		}
		known := err == nil

		anilistID := 0
		if known && externalSource != nil && *externalSource == "anilist" && externalID != nil {
			anilistID, _ = strconv.Atoi(*externalID)
		}

		if !known || needsEnrichment(syncedAt, time.Now()) {
			lightNovel, err := h.findLightNovel(entry.Name(), anilistID)
			if err != nil {
				logger.Warn("%s", err.Error())
				if !known {
					continue
				}
			} else if mediaItemID, err = h.saveLightNovel(entry.Name(), mediaItemID, lightNovel); err != nil {
				logger.Warn("%s", err.Error())
				continue
			}
		}

		h.addVolumes(mediaItemID, folder, files)
		tracking.PartsAdded(h.db, mediaItemID)
	}
	return nil
}

/*
Function:	findLightNovel
Purpose:	a light novel's AniList entry: by the id it's linked to, or by searching the folder name
Params:
  - folder: the folder's name
  - anilistID: the AniList id the title already has, 0 for none
*/
func (h *Handler) findLightNovel(folder string, anilistID int) (*anilist.Media, error) {
	if anilistID == 0 {
		query := strings.ReplaceAll(folder, "_", " ")
		results, err := h.anilist.Search("MANGA", query, 1, "NOVEL")
		time.Sleep(bookSearchPause)
		if err != nil {
			return nil, fmt.Errorf("anilist search failed for %s: %w", query, err)
		}
		if len(results) == 0 {
			return nil, fmt.Errorf("no anilist results for %s, skipping", query)
		}
		anilistID = results[0].ID
	}

	lightNovel, err := h.anilist.GetByID(anilistID)
	time.Sleep(bookSearchPause)
	if err != nil {
		return nil, fmt.Errorf("anilist fetch failed for %s (anilist %d): %w", folder, anilistID, err)
	}
	if lightNovel == nil {
		return nil, fmt.Errorf("no anilist entry %d for %s, skipping", anilistID, folder)
	}
	return lightNovel, nil
}

/*
Function:	saveLightNovel
Purpose:	add a light novel from its AniList entry, or fill in the one already there, and note
when its AniList data was fetched. Returns its id
Params:
  - folder: the folder's name, the title of a new one
  - mediaItemID: the title already in the library, "" for none
  - lightNovel: its AniList entry
*/
func (h *Handler) saveLightNovel(folder, mediaItemID string, lightNovel *anilist.Media) (string, error) {
	var releaseDate *string
	if lightNovel.StartDate.Year != nil {
		date := anilistDate(lightNovel)
		releaseDate = &date
	}

	if mediaItemID == "" {
		if err := h.db.QueryRow(
			`INSERT INTO media_items (type, title, description, cover_image_url, external_id, external_source, release_date)
			VALUES ('light_novel', $1, $2, $3, $4, 'anilist', $5) RETURNING id`,
			folder, lightNovel.Description, lightNovel.CoverImage.Large, strconv.Itoa(lightNovel.ID), releaseDate,
		).Scan(&mediaItemID); err != nil {
			return "", fmt.Errorf("failed to insert media item for %s: %w", folder, err)
		}
	} else {
		h.db.Exec(
			`UPDATE media_items SET
			external_id = COALESCE(external_id, $1),
			external_source = COALESCE(external_source, 'anilist'),
			`+locks.Keep(locks.Description, "description", "COALESCE(description, $2)")+`,
			`+locks.Keep(locks.Cover, "cover_image_url", "COALESCE(cover_image_url, $3)")+`,
			`+locks.Keep(locks.ReleaseDate, "release_date", "COALESCE(release_date, $4)")+`
			WHERE id = $5`,
			strconv.Itoa(lightNovel.ID), lightNovel.Description, lightNovel.CoverImage.Large, releaseDate, mediaItemID,
		)
	}

	// AniList's volume count is the series' length, which the progress counts go up to;
	// running series have none, which keeps the one there
	if _, err := h.db.Exec(
		`INSERT INTO light_novel_metadata (media_item_id, genres, total_volumes)
		VALUES ($1, $2, $3)
		ON CONFLICT (media_item_id) DO UPDATE SET
		`+locks.KeepMeta(locks.TotalVolumes, "light_novel_metadata", "total_volumes",
			"COALESCE(EXCLUDED.total_volumes, light_novel_metadata.total_volumes)"),
		mediaItemID, pq.Array(lightNovel.Genres), lightNovel.Volumes,
	); err != nil {
		return "", fmt.Errorf("failed to insert or update metadata for %s: %w", folder, err)
	}
	h.markEnriched(mediaItemID)
	return mediaItemID, nil
}

/*
Function:	addVolumes
Purpose:	add the epub volumes in a light novel's folder the library doesn't have, taking the
cover from volume 1 when the title has none
Params:
  - mediaItemID: the light novel
  - folder: its folder
  - files: the folder's entries
*/
func (h *Handler) addVolumes(mediaItemID, folder string, files []os.DirEntry) {
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
		if err := h.db.QueryRow(
			`INSERT INTO light_novel_volumes (media_item_id, volume_number, title, file_path)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (media_item_id, volume_number) DO UPDATE SET file_path = EXCLUDED.file_path
			RETURNING id`,
			mediaItemID, volumeNum, file.Name(), filePath,
		).Scan(&volumeID); err != nil {
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
