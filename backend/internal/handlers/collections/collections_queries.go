package collections

import (
	"database/sql"
	"strings"

	"github.com/lib/pq"
)

/*
The SQL pieces for showing entries are built with an alias suffix, because cover lookups nest
inside other queries (a nested playlist's mosaic inside a collection's items) and the inner
aliases mustn't shadow the outer ones. With suffix s, an entry's tables are ci{s} (the entry),
mi{s} (a track or title), mm{s} (its music metadata), ta{s} (a track's album) and a{s} (an album
entry)
*/

// an album's cover, or failing that one of its tracks' (an uploaded one first), as the album
// pages show it
func albumCoverOf(alias, s string) string {
	return `COALESCE(` + alias + `.cover_image_url, (
		SELECT mic` + s + `.cover_image_url FROM music_metadata mmc` + s + `
		JOIN media_items mic` + s + ` ON mic` + s + `.id = mmc` + s + `.media_item_id
		WHERE mmc` + s + `.album_id = ` + alias + `.id AND mic` + s + `.cover_image_url IS NOT NULL
		ORDER BY mic` + s + `.cover_image_url LIKE '/api/%' DESC, mmc` + s + `.track_number NULLS LAST
		LIMIT 1))`
}

// the joins that resolve entry ci{s}
func entryJoins(s string) string {
	return strings.NewReplacer("{s}", s).Replace(`
		LEFT JOIN media_items mi{s} ON mi{s}.id = ci{s}.media_item_id
		LEFT JOIN music_metadata mm{s} ON mm{s}.media_item_id = mi{s}.id
		LEFT JOIN albums ta{s} ON ta{s}.id = mm{s}.album_id
		LEFT JOIN albums a{s} ON a{s}.id = ci{s}.album_id`)
}

// entry ci{s}'s cover: a track shows its album's, a title its own, an album its album cover
func entryCover(s string) string {
	return `COALESCE(` + albumCoverOf("ta"+s, s+"t") + `, mi` + s + `.cover_image_url, ` + albumCoverOf("a"+s, s+"a") + `)`
}

// up to 4 different covers from the start of a collection, for its mosaic
func coversOf(collectionID, s string) string {
	return `ARRAY(SELECT cover FROM (
		SELECT ` + entryCover(s) + ` AS cover, MIN(ci` + s + `.position) AS first
		FROM collection_items ci` + s + entryJoins(s) + `
		WHERE ci` + s + `.collection_id = ` + collectionID + `
		GROUP BY 1) covers
	WHERE cover IS NOT NULL ORDER BY first LIMIT 4)`
}

// the user's own playlists and collections, newest change first (featured copies belong to
// nobody, so they never show here). $2 is a kind or ”, $3 an
// item id to mark the ones that hold it, or ”, and $4 the season of it, or ” for the whole thing
var summariesQuery = `
SELECT c.id, c.kind::text, c.name, c.description, c.updated_at,
	(SELECT COUNT(*) FROM collection_items x WHERE x.collection_id = c.id),
	` + coversOf("c.id", "c") + `,
	($3 <> '' AND EXISTS (SELECT 1 FROM collection_items x WHERE x.collection_id = c.id
		AND $3 IN (x.media_item_id::text, x.album_id::text, x.child_collection_id::text)
		AND COALESCE(x.season_number::text, '') = $4))
FROM collections c
WHERE c.owner_id = $1 AND ($2 = '' OR c.kind::text = $2)
ORDER BY c.updated_at DESC`

// everything in collection $1, in order
var entriesQuery = `
SELECT ci.id,
	COALESCE(mi.type::text, CASE WHEN ci.album_id IS NOT NULL THEN 'album' ELSE 'playlist' END),
	COALESCE(ci.media_item_id, ci.album_id, ci.child_collection_id)::text,
	COALESCE(mi.title, a.title, cc.name, ''),
	COALESCE(mm.artist, a.artist),
	` + entryCover("") + `,
	mi.release_date, mi.external_source::text,
	mm.duration_secs, COALESCE(mm.file_path IS NOT NULL, false),
	mm.album_id::text, ta.title, ci.season_number,
	CASE
		WHEN ci.season_number IS NOT NULL THEN
			(SELECT COUNT(*) FROM episodes x WHERE x.media_item_id = ci.media_item_id
				AND x.season_number = ci.season_number AND x.file_path IS NOT NULL)
		WHEN ci.album_id IS NOT NULL THEN
			(SELECT COUNT(*) FROM music_metadata x WHERE x.album_id = ci.album_id AND x.file_path IS NOT NULL)
		WHEN ci.child_collection_id IS NOT NULL THEN
			(SELECT COUNT(*) FROM collection_items x WHERE x.collection_id = ci.child_collection_id)
	END,
	CASE WHEN ci.child_collection_id IS NOT NULL THEN ` + coversOf("ci.child_collection_id", "n") + ` END
FROM collection_items ci` + entryJoins("") + `
LEFT JOIN collections cc ON cc.id = ci.child_collection_id
WHERE ci.collection_id = $1
ORDER BY ci.position, ci.added_at`

/*
smartQuery fills a computed playlist: the downloaded tracks whose status is in $2. A track saved
on its own goes by its own status, otherwise by its album's, so a track marked listened in an
album still to listen to is only in Listened. Newest list entry first, then album order
*/
var smartQuery = `
WITH statuses AS (
	SELECT mm.media_item_id AS track_id, ums.status::text AS status, ums.updated_at, 0 AS rank
	FROM user_media_status ums
	JOIN music_metadata mm ON mm.media_item_id = ums.media_item_id
	WHERE ums.user_id = $1
	UNION ALL
	SELECT mm.media_item_id, ums.status::text, ums.updated_at, 1
	FROM user_media_status ums
	JOIN music_metadata mm ON mm.album_id = ums.album_id
	WHERE ums.user_id = $1
),
best AS (
	SELECT DISTINCT ON (track_id) track_id, status, updated_at
	FROM statuses
	ORDER BY track_id, rank
)
SELECT mi.id, mi.title, mm.artist,
	COALESCE(` + albumCoverOf("ta", "t") + `, mi.cover_image_url),
	mi.release_date, mi.external_source::text, mm.duration_secs, mm.album_id::text, ta.title
FROM best
JOIN music_metadata mm ON mm.media_item_id = best.track_id AND mm.file_path IS NOT NULL
JOIN media_items mi ON mi.id = mm.media_item_id
LEFT JOIN albums ta ON ta.id = mm.album_id
WHERE best.status = ANY($2)
ORDER BY best.updated_at DESC, mm.album_id, mm.track_number NULLS LAST, mi.title`

// the statuses that put a track in each computed playlist. Dropped ones are in neither
var smartStatuses = map[string][]string{
	smartListened:   {"completed"},
	smartUnlistened: {"planned", "wishlist", "current"},
}

// the computed playlists' names, in the order they're listed
var smartNames = []struct{ id, name string }{
	{smartListened, "Listened"},
	{smartUnlistened, "Not listened yet"},
}

// a *sql.DB or *sql.Tx
type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

/*
Function:	loadEntries
Purpose:	Everything in a stored playlist or collection, in order
*/
func loadEntries(db querier, collectionID string) ([]Entry, error) {
	rows, err := db.Query(entriesQuery, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var e Entry
		var count sql.NullInt64
		var covers []string
		if err := rows.Scan(&e.EntryID, &e.Type, &e.ID, &e.Title, &e.Artist, &e.CoverImageURL,
			&e.ReleaseDate, &e.ExternalSource, &e.DurationSecs, &e.Available, &e.AlbumID, &e.AlbumTitle,
			&e.SeasonNumber, &count, pq.Array(&covers)); err != nil {
			return nil, err
		}
		if count.Valid {
			n := int(count.Int64)
			e.ItemCount = &n
		}
		e.Covers = covers
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

/*
Function:	loadSmart
Purpose:	The tracks in one of the computed playlists
Params:
  - db: where to look
  - userID: whose
  - id: smartListened or smartUnlistened
*/
func loadSmart(db querier, userID, id string) ([]Entry, error) {
	rows, err := db.Query(smartQuery, userID, pq.Array(smartStatuses[id]))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		e := Entry{Type: "music_track", Available: true}
		if err := rows.Scan(&e.ID, &e.Title, &e.Artist, &e.CoverImageURL, &e.ReleaseDate,
			&e.ExternalSource, &e.DurationSecs, &e.AlbumID, &e.AlbumTitle); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

/*
Function:	smartSummary
Purpose:	A computed playlist as the rows and picker show it, from its tracks
*/
func smartSummary(id, name string, tracks []Entry) Summary {
	covers := []string{}
	seen := map[string]bool{}
	for _, t := range tracks {
		if len(covers) == 4 {
			break
		}
		if t.CoverImageURL != nil && !seen[*t.CoverImageURL] {
			seen[*t.CoverImageURL] = true
			covers = append(covers, *t.CoverImageURL)
		}
	}
	return Summary{ID: id, Kind: kindPlaylist, Name: name, Smart: true, ItemCount: len(tracks), Covers: covers}
}
