package sync

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/covers"
	"github.com/Mantie7553/MediaHub/backend/internal/locks"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
	"github.com/lib/pq"
)

// pause after each AniList request, which allows about 30 a minute
const anilistPause = 2 * time.Second

/*
Function:	syncSonarr
Purpose:	sync every Sonarr series, then clear the files of series Sonarr no longer has
*/
func (h *Handler) syncSonarr() error {
	start := time.Now()
	series, err := h.sonarr.GetAllSeries()
	if err != nil {
		return err
	}
	maps, err := arr.LoadAnimeMaps()
	if err != nil {
		logger.Error("failed to fetch fribb anime-list map: %s", err.Error())
	}

	failed := 0
	ids := make([]int64, 0, len(series))
	for _, s := range series {
		ids = append(ids, int64(s.ID))
		if err := h.syncSeries(s, maps); err != nil {
			logger.Error("sync of %s failed: %s", s.Title, err.Error())
			failed++
		}
	}

	// an empty list is more likely a misconfigured Sonarr than an empty library
	if len(series) > 0 {
		h.forgetMissing("sonarr_items", "sonarr_series_id", ids, start)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d series failed", failed, len(series))
	}
	return nil
}

/*
Function:	syncRadarr
Purpose:	sync every Radarr movie, then clear the files of movies Radarr no longer has
*/
func (h *Handler) syncRadarr() error {
	start := time.Now()
	movies, err := h.radarr.GetAllMovies()
	if err != nil {
		return err
	}
	maps, err := arr.LoadAnimeMaps()
	if err != nil {
		logger.Error("failed to fetch fribb anime-list map: %s", err.Error())
	}

	failed := 0
	ids := make([]int64, 0, len(movies))
	for _, m := range movies {
		ids = append(ids, int64(m.ID))
		if err := h.syncMovie(m, maps); err != nil {
			logger.Error("sync of %s failed: %s", m.Title, err.Error())
			failed++
		}
	}

	if len(movies) > 0 {
		h.forgetMissing("radarr_items", "radarr_movie_id", ids, start)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d movies failed", failed, len(movies))
	}
	return nil
}

/*
Function:	SyncSeriesByID
Purpose:	sync one Sonarr series, creating its title when MediaHub doesn't have it yet. Used by
the Sonarr webhook
Params:
  - seriesID: the Sonarr series id
*/
func (h *Handler) SyncSeriesByID(seriesID int) error {
	s, err := h.sonarr.GetSeries(seriesID)
	if err != nil {
		return fmt.Errorf("getting series %d from Sonarr: %w", seriesID, err)
	}
	maps, err := arr.LoadAnimeMaps()
	if err != nil {
		logger.Error("failed to fetch fribb anime-list map: %s", err.Error())
	}
	return h.syncSeries(s, maps)
}

/*
Function:	SyncMovieByID
Purpose:	sync one Radarr movie, creating its title when MediaHub doesn't have it yet. Used by
the Radarr webhook
Params:
  - movieID: the Radarr movie id
*/
func (h *Handler) SyncMovieByID(movieID int) error {
	m, err := h.radarr.GetMovie(movieID)
	if err != nil {
		return fmt.Errorf("getting movie %d from Radarr: %w", movieID, err)
	}
	maps, err := arr.LoadAnimeMaps()
	if err != nil {
		logger.Error("failed to fetch fribb anime-list map: %s", err.Error())
	}
	return h.syncMovie(m, maps)
}

/*
Function:	syncSeries
Purpose:	bring one Sonarr series into the library: its title, AniList data (when over a week
old), episodes and the MyAnimeList id of each season. Episodes whose file Sonarr no longer has
keep their row with no file
Params:
  - s: the series from Sonarr
  - maps: Fribb's anime list
*/
func (h *Handler) syncSeries(s arr.SonarrSeries, maps arr.AnimeMaps) error {
	defer h.items.lock("sonarr:" + strconv.Itoa(s.ID))()

	var releaseDate *string
	if s.Year > 0 {
		date := fmt.Sprintf("%04d-01-01", s.Year)
		releaseDate = &date
	}

	var mediaItemID string
	var syncedAt *time.Time
	err := h.db.QueryRow(
		`SELECT si.media_item_id, mi.metadata_synced_at FROM sonarr_items si
		JOIN media_items mi ON mi.id = si.media_item_id
		WHERE si.sonarr_series_id = $1`,
		s.ID,
	).Scan(&mediaItemID, &syncedAt)

	if errors.Is(err, sql.ErrNoRows) {
		// a title saved from one of the series' seasons, or one Sonarr forgot and has again
		mediaItemID, syncedAt, err = h.titleOfSeries(s.TvdbID, maps)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = h.db.QueryRow(
			`INSERT INTO media_items (type, title, cover_image_url, description, release_date)
			VALUES ('anime', $1, $2, $3, $4)
			RETURNING id`,
			s.Title, utils.NullString(s.PosterURL()), utils.NullString(s.Overview), releaseDate,
		).Scan(&mediaItemID)
		if err != nil {
			return fmt.Errorf("creating media item: %w", err)
		}
	} else if err == nil {
		h.db.Exec(
			`UPDATE media_items SET
			`+locks.Keep(locks.Title, "title", "$1")+`,
			`+locks.Keep(locks.Cover, "cover_image_url", covers.KeepLocal("$2"))+`,
			`+locks.Keep(locks.Description, "description", "COALESCE(description, $3)")+`,
			`+locks.Keep(locks.ReleaseDate, "release_date", "COALESCE(release_date, $4)")+`
			WHERE id = $5`,
			s.Title, utils.NullString(s.PosterURL()), utils.NullString(s.Overview), releaseDate, mediaItemID,
		)
	} else {
		return fmt.Errorf("looking up sonarr_items: %w", err)
	}

	h.setTvdbID(mediaItemID, s.TvdbID)

	anilistID, mapped := maps.TvdbToAnilist[s.TvdbID]
	if s.TvdbID > 0 && mapped {
		if needsEnrichment(syncedAt, time.Now()) {
			h.enrichSeries(s.Title, mediaItemID, anilistID)
		}
	} else {
		// no AniList match: Sonarr's genres
		if _, err := h.db.Exec(
			`INSERT INTO anime_metadata (media_item_id, genres)
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO UPDATE SET
			`+locks.KeepMeta(locks.Genres, "anime_metadata", "genres", "EXCLUDED.genres"),
			mediaItemID, pq.Array(s.Genres),
		); err != nil {
			logger.Error("failed to update or insert anime metadata for %s: %s", s.Title, err.Error())
		}
	}

	if _, err := h.db.Exec(
		`INSERT INTO sonarr_items (media_item_id, sonarr_series_id)
		VALUES ($1, $2)
		ON CONFLICT (media_item_id) DO UPDATE SET
		sonarr_series_id = EXCLUDED.sonarr_series_id,
		last_synced_at = NOW()`,
		mediaItemID, s.ID,
	); err != nil {
		return fmt.Errorf("saving sonarr_items: %w", err)
	}

	episodes, err := h.sonarr.GetEpisodes(s.ID)
	if err != nil {
		return fmt.Errorf("getting episodes: %w", err)
	}
	files, err := h.sonarr.GetEpisodeFiles(s.ID)
	if err != nil {
		return fmt.Errorf("getting episode files: %w", err)
	}

	// episodes Sonarr has a file for, whether or not saving it below works, so a failed save
	// never clears a file that is still there
	withFile := []int64{}
	now := time.Now()
	for _, ep := range episodes {
		if !ep.HasFile {
			// listed without a file, so every season shows; a file it had is cleared below
			if ep.Listed(now) {
				if _, err := h.db.Exec(
					`INSERT INTO episodes (media_item_id, season_number, episode_number, title, sonarr_id)
					VALUES ($1, $2, $3, $4, $5)
					ON CONFLICT (media_item_id, season_number, episode_number) DO UPDATE SET
					title = CASE WHEN episodes.title_locked THEN episodes.title ELSE EXCLUDED.title END,
					-- an uploaded file in its place isn't Sonarr's, so the clearing below leaves it
					sonarr_id = CASE WHEN episodes.file_path IS NULL OR episodes.sonarr_id IS NOT NULL
						THEN EXCLUDED.sonarr_id ELSE episodes.sonarr_id END`,
					mediaItemID, ep.SeasonNumber, ep.EpisodeNumber, ep.Title, ep.ID,
				); err != nil {
					logger.Error("failed to list episode %d: %s", ep.ID, err.Error())
				}
			}
			continue
		}
		withFile = append(withFile, int64(ep.ID))
		filePath, ok := files[ep.EpisodeFileID]
		if !ok || filePath == "" {
			logger.Warn("Sonarr listed no file %d for episode %d of %s", ep.EpisodeFileID, ep.ID, s.Title)
			continue
		}
		if _, err := h.db.Exec(
			`INSERT INTO episodes (media_item_id, season_number, episode_number, title, file_path, sonarr_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (media_item_id, season_number, episode_number) DO UPDATE SET
			file_path = EXCLUDED.file_path,
			title = CASE WHEN episodes.title_locked THEN episodes.title ELSE EXCLUDED.title END,
			sonarr_id = EXCLUDED.sonarr_id,
			updated_at = NOW()`,
			mediaItemID, ep.SeasonNumber, ep.EpisodeNumber, ep.Title, filePath, ep.ID,
		); err != nil {
			logger.Error("failed to update or insert episode %d: %s", ep.ID, err.Error())
		}
	}

	// Sonarr's episodes whose file is gone keep their row, and users' progress, with no file.
	// Uploaded episodes have no sonarr_id and are left alone
	if _, err := h.db.Exec(
		`UPDATE episodes SET file_path = NULL, updated_at = NOW()
		WHERE media_item_id = $1 AND sonarr_id IS NOT NULL AND file_path IS NOT NULL
		AND NOT (sonarr_id = ANY($2))`,
		mediaItemID, pq.Array(withFile),
	); err != nil {
		logger.Error("failed to clear removed episode files of %s: %s", s.Title, err.Error())
	}

	tracking.PartsAdded(h.db, mediaItemID)
	h.saveSeasonMal(mediaItemID, s.TvdbID, episodes, maps.SeasonToMal)
	return nil
}

/*
Function:	titleOfSeries
Purpose:	the anime title that already stands for a TVDB series: the one with its TVDB id, else
one saved from any of the series' AniList entries, earliest season first
Returns:	sql.ErrNoRows when there is none
*/
func (h *Handler) titleOfSeries(tvdbID int, maps arr.AnimeMaps) (string, *time.Time, error) {
	var id string
	var syncedAt *time.Time
	if tvdbID <= 0 {
		return "", nil, sql.ErrNoRows
	}
	err := h.db.QueryRow(
		`SELECT id, metadata_synced_at FROM media_items WHERE type = 'anime' AND tvdb_id = $1`, tvdbID,
	).Scan(&id, &syncedAt)
	if !errors.Is(err, sql.ErrNoRows) {
		return id, syncedAt, err
	}
	for _, anilistID := range seriesAnilistCandidates(tvdbID, maps) {
		err = h.db.QueryRow(
			`SELECT id, metadata_synced_at FROM media_items
			WHERE type = 'anime' AND external_source = 'anilist' AND external_id = $1
			AND NOT EXISTS (SELECT 1 FROM sonarr_items si WHERE si.media_item_id = media_items.id)`,
			strconv.Itoa(anilistID),
		).Scan(&id, &syncedAt)
		if !errors.Is(err, sql.ErrNoRows) {
			return id, syncedAt, err
		}
	}
	return "", nil, sql.ErrNoRows
}

// seriesAnilistCandidates: the AniList entries a title saved from a TVDB series could be,
// its seasons in watch order and then the entry Fribb maps the series itself to (series Fribb
// gives no seasons for have only that one)
func seriesAnilistCandidates(tvdbID int, maps arr.AnimeMaps) []int {
	ids := []int{}
	seen := map[int]bool{}
	for _, entry := range maps.ShowSeasons[tvdbID] {
		if entry.AnilistID > 0 && !seen[entry.AnilistID] {
			seen[entry.AnilistID] = true
			ids = append(ids, entry.AnilistID)
		}
	}
	if id, ok := maps.TvdbToAnilist[tvdbID]; ok && id > 0 && !seen[id] {
		ids = append(ids, id)
	}
	return ids
}

/*
Function:	titleOfMovie
Purpose:	the movie title that already stands for a TMDB movie: the one with its TMDB id, else
one saved from its AniList entry that no Radarr movie is linked to. A movie deleted in Radarr
and added there again finds its old title this way
Returns:	sql.ErrNoRows when there is none
*/
func (h *Handler) titleOfMovie(tmdbID int, maps arr.AnimeMaps) (string, *time.Time, error) {
	var id string
	var syncedAt *time.Time
	if tmdbID <= 0 {
		return "", nil, sql.ErrNoRows
	}
	err := h.db.QueryRow(
		`SELECT id, metadata_synced_at FROM media_items WHERE type = 'movie' AND tmdb_id = $1`, tmdbID,
	).Scan(&id, &syncedAt)
	if !errors.Is(err, sql.ErrNoRows) {
		return id, syncedAt, err
	}
	anilistID, ok := movieAnilistCandidate(tmdbID, maps)
	if !ok {
		return "", nil, sql.ErrNoRows
	}
	err = h.db.QueryRow(
		`SELECT id, metadata_synced_at FROM media_items
		WHERE type = 'movie' AND external_source = 'anilist' AND external_id = $1
		AND NOT EXISTS (SELECT 1 FROM radarr_items ri WHERE ri.media_item_id = media_items.id)`,
		strconv.Itoa(anilistID),
	).Scan(&id, &syncedAt)
	return id, syncedAt, err
}

// movieAnilistCandidate: the AniList entry Fribb maps a TMDB movie to
func movieAnilistCandidate(tmdbID int, maps arr.AnimeMaps) (int, bool) {
	id, ok := maps.TmdbToAnilist[tmdbID]
	return id, ok && id > 0
}

// setTmdbID notes the TMDB movie a title is, unless another movie title already has it
func (h *Handler) setTmdbID(mediaItemID string, tmdbID int) {
	if tmdbID <= 0 {
		return
	}
	if _, err := h.db.Exec(
		`UPDATE media_items SET tmdb_id = $1 WHERE id = $2 AND tmdb_id IS DISTINCT FROM $1
		AND NOT EXISTS (SELECT 1 FROM media_items o WHERE o.type = 'movie' AND o.tmdb_id = $1 AND o.id <> $2)`,
		tmdbID, mediaItemID,
	); err != nil {
		logger.Error("failed to save TMDB id %d of %s: %s", tmdbID, mediaItemID, err.Error())
	}
}

// setTvdbID notes the TVDB series a title is. When another title already has it (seasons
// saved as titles of their own before this was kept) it is left for the merge
func (h *Handler) setTvdbID(mediaItemID string, tvdbID int) {
	if tvdbID <= 0 {
		return
	}
	if _, err := h.db.Exec(
		`UPDATE media_items SET tvdb_id = $1 WHERE id = $2 AND tvdb_id IS DISTINCT FROM $1
		AND NOT EXISTS (SELECT 1 FROM media_items o WHERE o.type = 'anime' AND o.tvdb_id = $1 AND o.id <> $2)`,
		tvdbID, mediaItemID,
	); err != nil {
		logger.Error("failed to save TVDB id %d of %s: %s", tvdbID, mediaItemID, err.Error())
	}
}

/*
Function:	enrichSeries
Purpose:	fill a series' AniList id, start date, studio, status and genres, and note when
Params:
  - title: for log messages
  - mediaItemID: the series
  - anilistID: its AniList id, from Fribb's list
*/
func (h *Handler) enrichSeries(title, mediaItemID string, anilistID int) {
	full, err := h.anilist.GetByID(anilistID)
	time.Sleep(anilistPause)
	if err != nil {
		logger.Error("anilist fetch failed for %s (anilist %d): %s", title, anilistID, err.Error())
		return
	}

	h.setAnilistID(mediaItemID, full)
	if full.StartDate.Year != nil {
		h.db.Exec(
			`UPDATE media_items SET `+locks.Keep(locks.ReleaseDate, "release_date", "$1")+` WHERE id = $2`,
			anilistDate(full), mediaItemID,
		)
	}

	var studio string
	if len(full.Studios) > 0 {
		studio = full.Studios[0]
	}
	if _, err := h.db.Exec(
		`INSERT INTO anime_metadata (media_item_id, studio, status, genres)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (media_item_id) DO UPDATE SET
		`+locks.KeepMeta(locks.Studio, "anime_metadata", "studio", "EXCLUDED.studio")+`,
		`+locks.KeepMeta(locks.Status, "anime_metadata", "status", "EXCLUDED.status")+`,
		`+locks.KeepMeta(locks.Genres, "anime_metadata", "genres", "EXCLUDED.genres"),
		mediaItemID, utils.NullString(studio), utils.NullString(full.LibraryStatus()), pq.Array(full.Genres),
	); err != nil {
		logger.Error("failed to upsert anime metadata for %s: %s", title, err.Error())
		return
	}
	h.markEnriched(mediaItemID)
}

/*
Function:	syncMovie
Purpose:	bring one Radarr movie into the library: its title, genres, AniList data (when over a
week old) and file. A movie whose file Radarr no longer has keeps its row with no file
Params:
  - m: the movie from Radarr
  - maps: Fribb's anime list
*/
func (h *Handler) syncMovie(m arr.RadarrMovie, maps arr.AnimeMaps) error {
	defer h.items.lock("radarr:" + strconv.Itoa(m.ID))()

	var mediaItemID string
	var syncedAt *time.Time
	err := h.db.QueryRow(
		`SELECT ri.media_item_id, mi.metadata_synced_at FROM radarr_items ri
		JOIN media_items mi ON mi.id = ri.media_item_id
		WHERE ri.radarr_movie_id = $1`,
		m.ID,
	).Scan(&mediaItemID, &syncedAt)

	if errors.Is(err, sql.ErrNoRows) {
		// one Radarr forgot and has again, or one saved from its AniList entry
		mediaItemID, syncedAt, err = h.titleOfMovie(m.TmdbID, maps)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = h.db.QueryRow(
			`INSERT INTO media_items (type, title, cover_image_url, description, release_date)
			VALUES ('movie', $1, $2, $3, $4)
			RETURNING id`,
			m.Title, utils.NullString(m.PosterURL()), utils.NullString(m.Overview), utils.NullString(m.ReleaseDate()),
		).Scan(&mediaItemID)
		if err != nil {
			return fmt.Errorf("creating media item: %w", err)
		}
	} else if err == nil {
		h.db.Exec(
			`UPDATE media_items SET
			`+locks.Keep(locks.Title, "title", "$1")+`,
			`+locks.Keep(locks.Cover, "cover_image_url", covers.KeepLocal("$2"))+`,
			`+locks.Keep(locks.Description, "description", "COALESCE(description, $3)")+`,
			`+locks.Keep(locks.ReleaseDate, "release_date", "COALESCE(release_date, $4)")+`
			WHERE id = $5`,
			m.Title, utils.NullString(m.PosterURL()), utils.NullString(m.Overview), utils.NullString(m.ReleaseDate()), mediaItemID,
		)
	} else {
		return fmt.Errorf("looking up radarr_items: %w", err)
	}

	h.setTmdbID(mediaItemID, m.TmdbID)

	anilistID, mapped := maps.TmdbToAnilist[m.TmdbID]
	enrich := m.TmdbID > 0 && mapped && needsEnrichment(syncedAt, time.Now())
	// Radarr's genres are the baseline; AniList's replace them below, and a movie enriched
	// recently keeps the ones it has
	if !mapped || enrich {
		if _, err := h.db.Exec(
			`INSERT INTO movie_metadata (media_item_id, genres)
			VALUES ($1, $2)
			ON CONFLICT (media_item_id) DO UPDATE SET
			`+locks.KeepMeta(locks.Genres, "movie_metadata", "genres", "EXCLUDED.genres"),
			mediaItemID, pq.Array(m.Genres),
		); err != nil {
			logger.Error("failed to update or insert movie metadata for %s: %s", m.Title, err.Error())
		}
	} else if _, err := h.db.Exec(
		`INSERT INTO movie_metadata (media_item_id, genres) VALUES ($1, $2) ON CONFLICT (media_item_id) DO NOTHING`,
		mediaItemID, pq.Array(m.Genres),
	); err != nil {
		logger.Error("failed to insert movie metadata for %s: %s", m.Title, err.Error())
	}
	if enrich {
		h.enrichMovie(m.Title, mediaItemID, anilistID)
	}

	if _, err := h.db.Exec(
		`INSERT INTO radarr_items (media_item_id, radarr_movie_id)
		VALUES ($1, $2)
		ON CONFLICT (media_item_id) DO UPDATE SET
		radarr_movie_id = EXCLUDED.radarr_movie_id,
		last_synced_at = NOW()`,
		mediaItemID, m.ID,
	); err != nil {
		return fmt.Errorf("saving radarr_items: %w", err)
	}

	// no file: cleared, so a deleted movie shows as not available instead of failing to play
	var filePath *string
	if m.HasFile && m.MovieFile.Path != "" {
		filePath = &m.MovieFile.Path
	}
	if _, err := h.db.Exec(
		`UPDATE movie_metadata SET file_path = $1 WHERE media_item_id = $2`,
		filePath, mediaItemID,
	); err != nil {
		return fmt.Errorf("saving file path: %w", err)
	}
	return nil
}

/*
Function:	enrichMovie
Purpose:	fill a movie's AniList id and genres, and note when
Params:
  - title: for log messages
  - mediaItemID: the movie
  - anilistID: its AniList id, from Fribb's list
*/
func (h *Handler) enrichMovie(title, mediaItemID string, anilistID int) {
	full, err := h.anilist.GetByID(anilistID)
	time.Sleep(anilistPause)
	if err != nil {
		logger.Error("anilist fetch failed for %s (anilist %d): %s", title, anilistID, err.Error())
		return
	}
	h.setAnilistID(mediaItemID, full)
	if _, err := h.db.Exec(
		`UPDATE movie_metadata SET `+locks.KeepMeta(locks.Genres, "movie_metadata", "genres", "$1")+`
		WHERE media_item_id = $2`,
		pq.Array(full.Genres), mediaItemID,
	); err != nil {
		logger.Error("failed to upsert movie metadata for %s: %s", title, err.Error())
		return
	}
	h.markEnriched(mediaItemID)
}

// setAnilistID links a title to its AniList entry, unless it already has an external id
func (h *Handler) setAnilistID(mediaItemID string, media *anilist.Media) {
	h.db.Exec(
		`UPDATE media_items SET
		external_id = COALESCE(external_id, $1),
		external_source = COALESCE(external_source, 'anilist')
		WHERE id = $2`,
		strconv.Itoa(media.ID), mediaItemID,
	)
}

// markEnriched notes that a title's AniList data was just fetched
func (h *Handler) markEnriched(mediaItemID string) {
	h.db.Exec(`UPDATE media_items SET metadata_synced_at = NOW() WHERE id = $1`, mediaItemID)
}

// anilistDate is an AniList start date as YYYY-MM-DD, missing month or day as 1
func anilistDate(m *anilist.Media) string {
	return fmt.Sprintf("%04d-%02d-%02d",
		*m.StartDate.Year,
		utils.DerefOr(m.StartDate.Month, 1),
		utils.DerefOr(m.StartDate.Day, 1),
	)
}

/*
Function:	forgetMissing
Purpose:	after a full sync, clear the files of series or movies Sonarr / Radarr no longer has and
drop their link, so a later request can add them again. Their titles, and users' progress, stay.
Links made after the sync started (a request sent meanwhile) are left alone
Params:
  - table: sonarr_items or radarr_items
  - idColumn: its Sonarr / Radarr id column
  - ids: every id Sonarr / Radarr listed
  - start: when the sync started
*/
func (h *Handler) forgetMissing(table, idColumn string, ids []int64, start time.Time) {
	clear := `UPDATE episodes SET file_path = NULL, updated_at = NOW()
		WHERE sonarr_id IS NOT NULL AND file_path IS NOT NULL AND media_item_id IN (SELECT media_item_id FROM gone)`
	if table == "radarr_items" {
		clear = `UPDATE movie_metadata SET file_path = NULL
		WHERE file_path IS NOT NULL AND media_item_id IN (SELECT media_item_id FROM gone)`
	}
	res, err := h.db.Exec(
		`WITH gone AS (
			DELETE FROM `+table+` WHERE NOT (`+idColumn+` = ANY($1)) AND last_synced_at < $2
			RETURNING media_item_id
		) `+clear,
		pq.Array(ids), start,
	)
	if err != nil {
		logger.Error("failed to clear titles gone from %s: %s", table, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		logger.Info("Cleared %d files of titles no longer in %s", n, table)
	}
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
