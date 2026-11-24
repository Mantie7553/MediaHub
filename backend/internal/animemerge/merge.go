package animemerge

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/locks"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
)

// Series is where a merge gets series from: arr.Shows, or a stand-in in tests
type Series interface {
	Maps() arr.AnimeMaps
	Load(tvdbIDs []int) map[int]arr.Show
}

/*
Function:	Find
Purpose:	the series that have more than one title in the library, and which title each keeps.
Changes nothing
Params:
  - db: the database
  - series: Fribb's list and the series' names
*/
func Find(db *sql.DB, series Series) ([]Group, error) {
	rows, err := db.Query(
		`SELECT mi.id, mi.title, COALESCE(mi.external_id, ''), COALESCE(mi.external_source::text, ''),
		COALESCE(mi.tvdb_id, 0), EXISTS (SELECT 1 FROM sonarr_items si WHERE si.media_item_id = mi.id)
		FROM media_items mi WHERE mi.type = 'anime'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.ID, &r.Title, &r.ExternalID, &r.ExternalSource, &r.TvdbID, &r.Sonarr); err != nil {
			return nil, err
		}
		all = append(all, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	maps := series.Maps()
	byTvdb := groupRows(all, maps)
	ids := make([]int, 0, len(byTvdb))
	for tvdb := range byTvdb {
		ids = append(ids, tvdb)
	}
	names := series.Load(ids)

	groups := make([]Group, 0, len(byTvdb))
	for tvdb, g := range byTvdb {
		group := planGroup(tvdb, g, maps)
		if s, ok := names[tvdb]; ok && s.Title != "" {
			group.Series = s.Title
		}
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Series < groups[j].Series })
	return groups, nil
}

// Result is how merging one series went
type Result struct {
	TvdbID int    `json:"tvdb_id"`
	Series string `json:"series"`
	Merged int    `json:"merged"`
	// the title the series' others were folded into
	KeepID string `json:"keep_id"`
	Error  string `json:"error,omitempty"`
}

/*
Function:	Merge
Purpose:	merge the series asked for, each in a transaction of its own so one failing leaves the
others merged. Groups are worked out again, so a merge only ever does what Find shows now
Params:
  - db: the database
  - series: Fribb's list, the series' names and their entries' episode counts
  - tvdbIDs: the series to merge
*/
func Merge(db *sql.DB, series Series, tvdbIDs []int) ([]Result, error) {
	groups, err := Find(db, series)
	if err != nil {
		return nil, err
	}
	want := map[int]bool{}
	for _, id := range tvdbIDs {
		want[id] = true
	}
	var picked []Group
	var ids []int
	for _, g := range groups {
		if want[g.TvdbID] {
			picked = append(picked, g)
			ids = append(ids, g.TvdbID)
		}
	}
	shows := series.Load(ids)

	results := make([]Result, 0, len(picked))
	for _, g := range picked {
		res := Result{TvdbID: g.TvdbID, Series: g.Series, KeepID: g.Keep.ID}
		show, ok := shows[g.TvdbID]
		if !ok {
			res.Error = "couldn't look up the series' seasons on AniList, try again"
		} else if err := mergeGroup(db, g, show); err != nil {
			logger.Error("merging the titles of %s: %s", g.Series, err.Error())
			res.Error = err.Error()
		} else {
			res.Merged = len(g.Drop)
			logger.Info("merged %d titles into %s", len(g.Drop), g.Series)
		}
		results = append(results, res)
	}
	return results, nil
}

// mergeGroup folds a series' titles into the one it keeps, in one transaction
func mergeGroup(db *sql.DB, g Group, show arr.Show) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	entries := make([]tracking.ShowEntry, 0, len(show.Entries))
	for _, e := range show.Entries {
		entries = append(entries, tracking.ShowEntry{AnilistID: e.AnilistID, Season: e.Season, Episodes: e.Episodes})
	}
	all := append([]Title{g.Keep}, g.Drop...)

	// everyone's entries are read before anything moves, each title's progress as it stands
	byUser, err := readEntries(tx, all)
	if err != nil {
		return fmt.Errorf("reading list entries: %w", err)
	}

	for _, d := range g.Drop {
		if err := moveTitle(tx, d, g.Keep.ID); err != nil {
			return fmt.Errorf("moving %s: %w", d.Title, err)
		}
	}
	for user, list := range byUser {
		if err := writeEntry(tx, user, g.Keep, combine(list, entries)); err != nil {
			return fmt.Errorf("saving the list entry of user %s: %w", user, err)
		}
	}
	if err := finishKeep(tx, g, show); err != nil {
		return fmt.Errorf("updating %s: %w", g.Keep.Title, err)
	}
	return tx.Commit()
}

// every user's list entries for the titles, with how far each got
func readEntries(tx *sql.Tx, titles []Title) (map[string][]Entry, error) {
	out := map[string][]Entry{}
	for _, t := range titles {
		rows, err := tx.Query(
			`SELECT s.user_id, s.status, s.rating, s.rating_is_set, GREATEST(COALESCE(s.progress_count, 0),
				(SELECT COUNT(*) FROM user_anime_progress p JOIN episodes e ON e.id = p.episode_id
				 WHERE p.user_id = s.user_id AND e.media_item_id = s.media_item_id AND p.watched))
			FROM user_media_status s WHERE s.media_item_id = $1`,
			t.ID,
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var user string
			var rating sql.NullInt64
			e := Entry{Title: t}
			if err := rows.Scan(&user, &e.Status, &rating, &e.RatingIsSet, &e.Done); err != nil {
				rows.Close()
				return nil, err
			}
			if rating.Valid {
				r := int(rating.Int64)
				e.Rating = &r
			}
			out[user] = append(out[user], e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

/*
Function:	moveTitle
Purpose:	point everything that points at a title being merged at the kept one instead, then
delete it. Where the kept title already has the same thing (an episode, a collection entry, a
day of watch time) the two are combined or the merged one's is dropped
*/
func moveTitle(tx *sql.Tx, from Title, to string) error {
	steps := []struct {
		what  string
		query string
	}{
		// episodes the kept title has too: their progress moves over, then they go
		{"episode progress", `INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at)
			SELECT p.user_id, k.id, p.watched, p.watched_at FROM user_anime_progress p
			JOIN episodes e ON e.id = p.episode_id
			JOIN episodes k ON k.media_item_id = $2 AND k.season_number = e.season_number AND k.episode_number = e.episode_number
			WHERE e.media_item_id = $1
			ON CONFLICT (user_id, episode_id) DO UPDATE SET
			watched = user_anime_progress.watched OR EXCLUDED.watched,
			watched_at = GREATEST(user_anime_progress.watched_at, EXCLUDED.watched_at)`},
		{"playback positions", `INSERT INTO user_episode_progress (user_id, episode_id, position_secs, duration_secs, updated_at)
			SELECT p.user_id, k.id, p.position_secs, p.duration_secs, p.updated_at FROM user_episode_progress p
			JOIN episodes e ON e.id = p.episode_id
			JOIN episodes k ON k.media_item_id = $2 AND k.season_number = e.season_number AND k.episode_number = e.episode_number
			WHERE e.media_item_id = $1
			ON CONFLICT (user_id, episode_id) DO NOTHING`},
		{"duplicate episodes", `DELETE FROM episodes e USING episodes k
			WHERE e.media_item_id = $1 AND k.media_item_id = $2
			AND k.season_number = e.season_number AND k.episode_number = e.episode_number`},
		{"episodes", `UPDATE episodes SET media_item_id = $2 WHERE media_item_id = $1`},

		{"download requests", `UPDATE download_requests r SET media_item_id = $2 WHERE r.media_item_id = $1
			AND NOT EXISTS (SELECT 1 FROM download_requests o WHERE o.media_item_id = $2 AND o.requested_by = r.requested_by)`},
		{"download jobs", `UPDATE download_jobs SET media_item_id = $2 WHERE media_item_id = $1`},
		{"watch time", `INSERT INTO watch_time (user_id, media_item_id, day, seconds)
			SELECT user_id, $2, day, seconds FROM watch_time WHERE media_item_id = $1
			ON CONFLICT (user_id, media_item_id, day) DO UPDATE SET seconds = watch_time.seconds + EXCLUDED.seconds`},
		{"season ratings", `INSERT INTO user_season_ratings (user_id, media_item_id, season_number, anilist_id, rating, updated_at)
			SELECT user_id, $2, season_number, anilist_id, rating, updated_at FROM user_season_ratings WHERE media_item_id = $1
			ON CONFLICT DO NOTHING`},
		{"season MyAnimeList ids", `INSERT INTO anime_season_mal (media_item_id, season_number, mal_id)
			SELECT $2, season_number, mal_id FROM anime_season_mal WHERE media_item_id = $1
			ON CONFLICT DO NOTHING`},
		{"wishlist", `INSERT INTO user_wishlist (user_id, media_item_id, created_at)
			SELECT user_id, $2, created_at FROM user_wishlist WHERE media_item_id = $1
			ON CONFLICT DO NOTHING`},
		{"Sonarr link", `UPDATE sonarr_items SET media_item_id = $2 WHERE media_item_id = $1
			AND NOT EXISTS (SELECT 1 FROM sonarr_items WHERE media_item_id = $2)`},
	}
	for _, s := range steps {
		if _, err := tx.Exec(s.query, from.ID, to); err != nil {
			return fmt.Errorf("%s: %w", s.what, err)
		}
	}
	if err := moveCollectionItems(tx, from, to); err != nil {
		return fmt.Errorf("collections: %w", err)
	}
	_, err := tx.Exec(`DELETE FROM media_items WHERE id = $1`, from.ID)
	return err
}

/*
Function:	moveCollectionItems
Purpose:	a collection entry for a title that stood for one season becomes an entry for that
season of the kept series; one for a whole title becomes the whole series. An entry the
collection already has for it is dropped
*/
func moveCollectionItems(tx *sql.Tx, from Title, to string) error {
	rows, err := tx.Query(`SELECT id, collection_id, season_number FROM collection_items WHERE media_item_id = $1`, from.ID)
	if err != nil {
		return err
	}
	type item struct {
		id, collection string
		season         sql.NullInt64
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.collection, &it.season); err != nil {
			rows.Close()
			return err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		season := it.season
		if !season.Valid && from.Relative() && from.Season > 0 {
			season = sql.NullInt64{Int64: int64(from.Season), Valid: true}
		}
		// a collection holds either the whole series or some of its seasons, never both
		var taken bool
		if err := tx.QueryRow(
			`SELECT EXISTS (SELECT 1 FROM collection_items WHERE collection_id = $1 AND media_item_id = $2
			AND ($3::int IS NULL OR season_number IS NULL OR season_number = $3))`,
			it.collection, to, season,
		).Scan(&taken); err != nil {
			return err
		}
		if taken {
			if _, err := tx.Exec(`DELETE FROM collection_items WHERE id = $1`, it.id); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(`UPDATE collection_items SET media_item_id = $2, season_number = $3 WHERE id = $1`,
			it.id, to, season); err != nil {
			return err
		}
	}
	return nil
}

/*
Function:	writeEntry
Purpose:	save one user's combined entry on the kept title: its status, the series' progress,
their season ratings, and the series rating (picked, or the seasons' average)
*/
func writeEntry(tx *sql.Tx, user string, keep Title, o Outcome) error {
	if _, err := tx.Exec(
		`INSERT INTO user_media_status (user_id, media_item_id, status) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, media_item_id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()`,
		user, keep.ID, o.Status,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE user_media_status SET rating = $3, rating_is_set = $3::int IS NOT NULL
		WHERE user_id = $1 AND media_item_id = $2`,
		user, keep.ID, o.Rating,
	); err != nil {
		return err
	}
	for _, s := range o.Seasons {
		r := s.Rating
		if err := tracking.SetSeasonRating(tx, user, keep.ID, s.Season, s.AnilistID, &r); err != nil {
			return err
		}
	}
	if err := tracking.DeriveSeriesRating(tx, user, keep.ID); err != nil {
		return err
	}

	// progress only moves forward: setting a count re-marks episodes from the first, which
	// would undo a series watched out of order. A kept title that stood for one season counted
	// from that season, so its count is always replaced
	var done int
	if err := tx.QueryRow(
		`SELECT GREATEST(COALESCE(s.progress_count, 0),
			(SELECT COUNT(*) FROM user_anime_progress p JOIN episodes e ON e.id = p.episode_id
			 WHERE p.user_id = s.user_id AND e.media_item_id = s.media_item_id AND p.watched))
		FROM user_media_status s WHERE s.user_id = $1 AND s.media_item_id = $2`,
		user, keep.ID,
	).Scan(&done); err != nil {
		return err
	}
	if (keep.Relative() && o.Done > 0) || o.Done > done {
		return tracking.SetProgressCount(tx, user, keep.ID, o.Done)
	}
	return nil
}

// finishKeep makes the kept title the series: its TVDB id, its first entry's AniList id and,
// unless an admin locked it or Sonarr names it, the series' name
func finishKeep(tx *sql.Tx, g Group, show arr.Show) error {
	if _, err := tx.Exec(
		`UPDATE media_items SET tvdb_id = $2 WHERE id = $1
		AND NOT EXISTS (SELECT 1 FROM media_items o WHERE o.type = 'anime' AND o.tvdb_id = $2 AND o.id <> $1)`,
		g.Keep.ID, g.TvdbID,
	); err != nil {
		return err
	}
	canonical := strconv.Itoa(show.AnilistID)
	var taken bool
	err := tx.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM media_items WHERE external_source = 'anilist' AND external_id = $1 AND id <> $2)`,
		canonical, g.Keep.ID,
	).Scan(&taken)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if !taken && show.AnilistID > 0 {
		if _, err := tx.Exec(
			`UPDATE media_items SET external_id = $2, external_source = 'anilist' WHERE id = $1`,
			g.Keep.ID, canonical,
		); err != nil {
			return err
		}
	}
	if !g.Keep.Sonarr && g.Series != "" {
		if _, err := tx.Exec(
			`UPDATE media_items SET `+locks.Keep(locks.Title, "title", "$2")+` WHERE id = $1`,
			g.Keep.ID, g.Series,
		); err != nil {
			return err
		}
	}
	return nil
}
