package sync

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/locks"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/lib/pq"
)

/*
Function:	syncMusic
Purpose:	add the mp3s under Music/Artist/Album, and fill in what tracks and albums are missing
(track number, length, genres, album year) from the files' ID3 tags. Returns how many tracks
were synced
*/
func (h *Handler) syncMusic() (int, error) {
	musicRoot := filepath.Join(os.Getenv("MEDIA_ROOT"), "Music")
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

		if h.syncKnownTrack(path) || h.addTrack(musicRoot, path, parts[0], parts[1]) {
			synced++
		}
		return nil
	})
	if err != nil {
		return synced, fmt.Errorf("failed to walk music directory: %w", err)
	}

	logger.Info("Music sync complete: %d tracks synced", synced)
	return synced, nil
}

/*
Function:	syncKnownTrack
Purpose:	for a file already in the library, fill in what it's missing from its tags and its
cover. Its title, artist and album stay, an admin may have changed them. Returns false when the
file isn't in the library
Params:
  - path: the track's file
*/
func (h *Handler) syncKnownTrack(path string) bool {
	var mediaItemID string
	var albumID *string
	var missingTags, missingDuration bool
	err := h.db.QueryRow(
		`SELECT mm.media_item_id, mm.album_id,
		mm.track_number IS NULL OR mm.genres IS NULL OR mm.genres = '{}' OR (a.id IS NOT NULL AND a.release_date IS NULL),
		mm.duration_secs IS NULL
		FROM music_metadata mm LEFT JOIN albums a ON a.id = mm.album_id
		WHERE mm.file_path = $1`, path,
	).Scan(&mediaItemID, &albumID, &missingTags, &missingDuration)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logger.Error("failed to look up track %s: %s", path, err.Error())
		}
		return false
	}

	if missingTags || missingDuration {
		tags, _ := musicfiles.ReadTags(path)
		duration := 0
		if missingDuration {
			duration = musicfiles.Duration(path)
		}
		h.fillTrack(mediaItemID, albumID, tags, duration)
	}
	h.syncTrackCover(mediaItemID, path, albumID)
	return true
}

/*
Function:	addTrack
Purpose:	add a file the library doesn't have: to the requested track waiting for it, or as a
new track, on the album named after its folder. Returns false when it couldn't be added
Params:
  - musicRoot: the Music folder
  - path: the track's file
  - artist: its artist folder
  - albumName: its album folder
*/
func (h *Handler) addTrack(musicRoot, path, artist, albumName string) bool {
	fileName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	tags, _ := musicfiles.ReadTags(path)
	title := tags.Title
	if title == "" {
		title = fileName
	}

	// a requested track still waiting for its file, by its tagged title or the file's name,
	// the same artist first. Tracks with a file are never matched, so two songs with the same
	// title stay apart
	var mediaItemID string
	err := h.db.QueryRow(
		`SELECT mi.id FROM media_items mi
		LEFT JOIN music_metadata mm ON mm.media_item_id = mi.id
		WHERE mi.type = 'music_track' AND COALESCE(mm.file_path, '') = ''
		AND LOWER(mi.title) IN (LOWER($1), LOWER($2))
		ORDER BY LOWER(COALESCE(mm.artist, '')) = LOWER($3) DESC
		LIMIT 1`,
		title, fileName, artist,
	).Scan(&mediaItemID)
	if errors.Is(err, sql.ErrNoRows) {
		err = h.db.QueryRow(
			`INSERT INTO media_items (type, title) VALUES ('music_track', $1) RETURNING id`,
			title,
		).Scan(&mediaItemID)
		if err != nil {
			logger.Error("failed to create media item for %s: %s", fileName, err.Error())
			return false
		}
	} else if err != nil {
		logger.Error("failed to query media item for %s: %s", fileName, err.Error())
		return false
	}

	albumID, err := h.albumFor(musicRoot, artist, albumName)
	if err != nil {
		logger.Error("failed to find or create album %s: %s", albumName, err.Error())
		return false
	}

	// a requested track keeps the length and number it was requested with
	if _, err := h.db.Exec(
		`INSERT INTO music_metadata (media_item_id, artist, album_id, file_path, track_number, duration_secs, genres)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (media_item_id) DO UPDATE SET
		artist = EXCLUDED.artist,
		album_id = EXCLUDED.album_id,
		file_path = EXCLUDED.file_path,
		track_number = COALESCE(music_metadata.track_number, EXCLUDED.track_number),
		duration_secs = COALESCE(music_metadata.duration_secs, EXCLUDED.duration_secs),
		genres = COALESCE(NULLIF(music_metadata.genres, '{}'), EXCLUDED.genres)`,
		mediaItemID, artist, albumID, path, positive(tags.Track), positive(musicfiles.Duration(path)), pq.Array(tags.Genres),
	); err != nil {
		logger.Error("failed to upsert music_metadata for %s: %s", fileName, err.Error())
		return false
	}
	h.fillAlbumYear(albumID, tags.Year)
	h.syncTrackCover(mediaItemID, path, &albumID)
	return true
}

/*
Function:	albumFor
Purpose:	the album a folder's tracks are on: the one its other tracks are on (which may have been
renamed since), one named after the folder, or a new one. Singles are an album too
Params:
  - musicRoot: the Music folder
  - artist: the artist folder
  - albumName: the album folder
*/
func (h *Handler) albumFor(musicRoot, artist, albumName string) (string, error) {
	var id string
	err := h.db.QueryRow(
		`SELECT album_id FROM music_metadata WHERE album_id IS NOT NULL AND file_path LIKE $1 ESCAPE '\' LIMIT 1`,
		utils.LikePrefix(filepath.Join(musicRoot, artist, albumName)+string(filepath.Separator)),
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = h.db.QueryRow(`SELECT id FROM albums WHERE title = $1 AND artist = $2`, albumName, artist).Scan(&id)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = h.db.QueryRow(
			`INSERT INTO albums (title, artist) VALUES ($1, $2) RETURNING id`,
			albumName, artist,
		).Scan(&id)
	}
	return id, err
}

/*
Function:	fillTrack
Purpose:	fill a track's missing number, length and genres, and its album's year. Values it
already has, and locked ones, stay
Params:
  - mediaItemID: the track
  - albumID: its album, nil for none
  - tags: its file's tags
  - duration: its length in seconds, 0 for unknown
*/
func (h *Handler) fillTrack(mediaItemID string, albumID *string, tags musicfiles.Tags, duration int) {
	if _, err := h.db.Exec(
		`UPDATE music_metadata SET
		`+locks.KeepMeta(locks.TrackNumber, "music_metadata", "track_number", "COALESCE(music_metadata.track_number, $1)")+`,
		duration_secs = COALESCE(duration_secs, $2),
		`+locks.KeepMeta(locks.Genres, "music_metadata", "genres", "COALESCE(NULLIF(music_metadata.genres, '{}'), $3)")+`
		WHERE media_item_id = $4`,
		positive(tags.Track), positive(duration), pq.Array(tags.Genres), mediaItemID,
	); err != nil {
		logger.Error("failed to fill in track %s from its tags: %s", mediaItemID, err.Error())
	}
	if albumID != nil {
		h.fillAlbumYear(*albumID, tags.Year)
	}
}

// fillAlbumYear sets an album's release date to its tagged year when it has none
func (h *Handler) fillAlbumYear(albumID string, year int) {
	if year <= 0 {
		return
	}
	h.db.Exec(
		`UPDATE albums SET release_date = make_date($1, 1, 1), updated_at = NOW() WHERE id = $2 AND release_date IS NULL`,
		year, albumID,
	)
}

// positive is n, or NULL when it isn't above 0 (a missing tag)
func positive(n int) any {
	if n <= 0 {
		return nil
	}
	return n
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
