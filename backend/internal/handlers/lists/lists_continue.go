package lists

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// how many titles the Continue row shows
const continueLimit = 20

/*
continueQuery finds every title the user has started and not finished, newest activity first.
Each branch reads the progress tables directly, so a title doesn't have to be on the user's list,
but one they marked completed or dropped is left out. The rules match the details pages: a video
counts as started after 30 seconds and as finished within its last 2 minutes (credits), and an
episode, chapter or volume without a file can't be next
*/
const continueQuery = `
WITH finished_episodes AS (
	SELECT e.id, e.media_item_id
	FROM episodes e
	LEFT JOIN user_anime_progress uap ON uap.episode_id = e.id AND uap.user_id = $1
	LEFT JOIN user_episode_progress uep ON uep.episode_id = e.id AND uep.user_id = $1
	WHERE uap.watched IS TRUE
		OR (uep.duration_secs > 0 AND uep.position_secs >= uep.duration_secs - 120)
),
progress AS (
	-- anime: the first episode not finished yet
	SELECT 'anime' AS kind, a.media_item_id, a.last_activity,
		NULL::float8 AS position_secs, NULL::float8 AS duration_secs,
		a.done, a.total, nxt.season_number AS season, a.seasons, nxt.episode_number::text AS number,
		NULL::int AS page, NULL::float8 AS scroll, NULL::bool AS completed, NULL::text AS next_number
	FROM (
		SELECT e.media_item_id,
			MAX(GREATEST(
				CASE WHEN uap.watched THEN uap.watched_at END,
				CASE WHEN uep.position_secs > 30 THEN uep.updated_at END)) AS last_activity,
			COUNT(fe.id) FILTER (WHERE e.file_path IS NOT NULL) AS done,
			COUNT(*) FILTER (WHERE e.file_path IS NOT NULL) AS total,
			COUNT(DISTINCT e.season_number) AS seasons
		FROM episodes e
		LEFT JOIN user_anime_progress uap ON uap.episode_id = e.id AND uap.user_id = $1
		LEFT JOIN user_episode_progress uep ON uep.episode_id = e.id AND uep.user_id = $1
		LEFT JOIN finished_episodes fe ON fe.id = e.id
		WHERE e.media_item_id IN (
			SELECT e2.media_item_id FROM episodes e2
			JOIN user_anime_progress uap2 ON uap2.episode_id = e2.id AND uap2.user_id = $1 AND uap2.watched
			UNION
			SELECT e2.media_item_id FROM episodes e2
			JOIN user_episode_progress uep2 ON uep2.episode_id = e2.id AND uep2.user_id = $1 AND uep2.position_secs > 30
		)
		GROUP BY e.media_item_id
	) a
	JOIN LATERAL (
		SELECT n.season_number, n.episode_number
		FROM episodes n
		WHERE n.media_item_id = a.media_item_id AND n.file_path IS NOT NULL
			AND n.id NOT IN (SELECT id FROM finished_episodes)
		ORDER BY n.season_number, n.episode_number
		LIMIT 1
	) nxt ON true

	UNION ALL

	-- movies: started and not into the credits
	SELECT 'movie', ump.media_item_id, ump.updated_at,
		ump.position_secs, ump.duration_secs,
		NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL
	FROM user_movie_progress ump
	WHERE ump.user_id = $1 AND ump.position_secs > 30
		AND (ump.duration_secs = 0 OR ump.position_secs < ump.duration_secs - 120)

	UNION ALL

	-- manga: the chapter read last, while a chapter with a file is still unread
	SELECT 'manga', last.media_item_id, last.updated_at,
		NULL, NULL, counts.done, counts.total, NULL, NULL,
		last.chapter_number::text, last.last_page_read, NULL, last.completed,
		(SELECT mc.chapter_number::text FROM manga_chapters mc
			LEFT JOIN manga_progress mp ON mp.chapter_id = mc.id AND mp.user_id = $1
			WHERE mc.media_item_id = last.media_item_id AND mc.file_path IS NOT NULL
				AND mp.completed IS NOT TRUE AND mc.chapter_number > last.chapter_number
			ORDER BY mc.chapter_number LIMIT 1)
	FROM (
		SELECT DISTINCT ON (mp.media_item_id) mp.media_item_id, mp.updated_at, mp.last_page_read,
			COALESCE(mp.completed, false) AS completed, mc.chapter_number
		FROM manga_progress mp
		JOIN manga_chapters mc ON mc.id = mp.chapter_id
		WHERE mp.user_id = $1 AND mp.updated_at IS NOT NULL
		ORDER BY mp.media_item_id, mp.updated_at DESC
	) last
	JOIN LATERAL (
		SELECT COUNT(*) FILTER (WHERE mp.completed) AS done, COUNT(*) AS total
		FROM manga_chapters mc
		LEFT JOIN manga_progress mp ON mp.chapter_id = mc.id AND mp.user_id = $1
		WHERE mc.media_item_id = last.media_item_id AND mc.file_path IS NOT NULL
	) counts ON counts.done < counts.total

	UNION ALL

	-- light novels: the volume read last, while a volume is still unread
	SELECT 'light_novel', last.media_item_id, last.updated_at,
		NULL, NULL, counts.done, counts.total, NULL, NULL,
		last.volume_number::text, NULL, last.scroll_position, last.completed,
		(SELECT lv.volume_number::text FROM light_novel_volumes lv
			LEFT JOIN light_novel_progress lp ON lp.volume_id = lv.id AND lp.user_id = $1
			WHERE lv.media_item_id = last.media_item_id AND lv.file_path IS NOT NULL
				AND lp.completed IS NOT TRUE AND lv.volume_number > last.volume_number
			ORDER BY lv.volume_number LIMIT 1)
	FROM (
		SELECT DISTINCT ON (lp.media_item_id) lp.media_item_id, lp.updated_at, lp.scroll_position,
			lp.completed, lv.volume_number
		FROM light_novel_progress lp
		JOIN light_novel_volumes lv ON lv.id = lp.volume_id
		WHERE lp.user_id = $1
		ORDER BY lp.media_item_id, lp.updated_at DESC
	) last
	JOIN LATERAL (
		SELECT COUNT(*) FILTER (WHERE lp.completed) AS done, COUNT(*) AS total
		FROM light_novel_volumes lv
		LEFT JOIN light_novel_progress lp ON lp.volume_id = lv.id AND lp.user_id = $1
		WHERE lv.media_item_id = last.media_item_id AND lv.file_path IS NOT NULL
	) counts ON counts.done < counts.total
)
SELECT p.kind, mi.id, mi.title, mi.cover_image_url, mi.release_date, mi.external_id, mi.external_source, p.last_activity,
	p.position_secs, p.duration_secs, p.done, p.total, p.season, p.seasons, p.number,
	p.page, p.scroll, p.completed, p.next_number
FROM progress p
JOIN media_items mi ON mi.id = p.media_item_id
WHERE p.last_activity IS NOT NULL
	AND NOT EXISTS (
		SELECT 1 FROM user_media_status ums
		WHERE ums.user_id = $1 AND ums.media_item_id = p.media_item_id
			AND ums.status IN ('completed', 'dropped'))
ORDER BY p.last_activity DESC
LIMIT $2`

/*
continueProgress is how far into a title the user is, as the query found it. Which fields are set
depends on the kind:
  - anime: Done/Total episodes, and Season/Number of the next one (Seasons tells whether to name the season)
  - movie: PositionSecs/DurationSecs
  - manga: Done/Total chapters, Number and Page of the one read last, and NextNumber after it
  - light_novel: Done/Total volumes, Number and Scroll of the one read last, and NextNumber after it
*/
type continueProgress struct {
	Kind         string
	PositionSecs sql.NullFloat64
	DurationSecs sql.NullFloat64
	Done         sql.NullInt64
	Total        sql.NullInt64
	Season       sql.NullInt64
	Seasons      sql.NullInt64
	Number       sql.NullString
	Page         sql.NullInt64
	Scroll       sql.NullFloat64
	Completed    sql.NullBool
	NextNumber   sql.NullString
}

/*
Function:	pct
Purpose:	How much of the title is done, as a whole percentage for the card's progress bar
Returns:	nil when there's nothing to measure against
*/
func (p continueProgress) pct() *int {
	var part, whole float64
	if p.Kind == "movie" {
		part, whole = p.PositionSecs.Float64, p.DurationSecs.Float64
	} else {
		part, whole = float64(p.Done.Int64), float64(p.Total.Int64)
	}
	if whole <= 0 {
		return nil
	}
	pct := int(math.Round(part / whole * 100))
	return &pct
}

/*
Function:	label
Purpose:	The line under the card's title saying where the user is up to, like "S2 · E5",
"42 min left", "Ch 12 · p. 6" or "Next: Vol 3"
Returns:	nil when the query didn't find enough to say
*/
func (p continueProgress) label() *string {
	var s string
	switch p.Kind {
	case "anime":
		if !p.Number.Valid {
			return nil
		}
		s = "E" + number(p.Number.String)
		if p.Seasons.Int64 > 1 {
			s = fmt.Sprintf("S%d · %s", p.Season.Int64, s)
		}
	case "movie":
		if p.DurationSecs.Float64 <= 0 {
			return nil
		}
		s = fmt.Sprintf("%d min left", int(math.Ceil((p.DurationSecs.Float64-p.PositionSecs.Float64)/60)))
	case "manga", "light_novel":
		unit := "Ch"
		if p.Kind == "light_novel" {
			unit = "Vol"
		}
		switch {
		case p.Completed.Bool && p.NextNumber.Valid:
			s = fmt.Sprintf("Next: %s %s", unit, number(p.NextNumber.String))
		case p.Completed.Bool || !p.Number.Valid:
			// finished the one read last, and the next unread one is further back
			return nil
		case p.Kind == "manga" && p.Page.Valid:
			s = fmt.Sprintf("Ch %s · p. %d", number(p.Number.String), p.Page.Int64+1)
		case p.Kind == "light_novel" && p.Scroll.Float64 > 0:
			s = fmt.Sprintf("Vol %s · %d%%", number(p.Number.String), int(math.Round(p.Scroll.Float64*100)))
		default:
			s = unit + " " + number(p.Number.String)
		}
	default:
		return nil
	}
	return &s
}

// a chapter or volume number without trailing zeros, so 12.0 reads 12 and 10.50 reads 10.5
func number(raw string) string {
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

/*
Function:	GetContinue
Purpose:	Get the titles the user started and hasn't finished, newest activity first, for the
dashboard's Continue row
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetContinue(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)

	items, err := loadContinue(h.db, user.UserID)
	if utils.InternalError(w, err) {
		return
	}
	for i := range items {
		if utils.InternalError(w, auth.SignCoverURL(items[i].CoverImageURL, user.UserID)) {
			return
		}
	}

	utils.JSON(w, items)
}

// a *sql.DB, or a *sql.Tx in tests
type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

/*
Function:	loadContinue
Purpose:	Run continueQuery for a user and turn each row into a card
Params:
  - db: where to run it
  - userID: whose progress
*/
func loadContinue(db querier, userID string) ([]ContinueEntry, error) {
	items := []ContinueEntry{}

	rows, err := db.Query(continueQuery, userID, continueLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item ContinueEntry
		var p continueProgress
		err := rows.Scan(
			&p.Kind, &item.MediaItemID, &item.MediaTitle, &item.CoverImageURL, &item.ReleaseDate,
			&item.ExternalID, &item.ExternalSource, &item.LastActivityAt,
			&p.PositionSecs, &p.DurationSecs, &p.Done, &p.Total, &p.Season, &p.Seasons, &p.Number,
			&p.Page, &p.Scroll, &p.Completed, &p.NextNumber,
		)
		if err != nil {
			return nil, err
		}
		item.ID = item.MediaItemID
		item.MediaType = p.Kind
		item.ProgressPct = p.pct()
		item.ProgressLabel = p.label()
		items = append(items, item)
	}

	return items, rows.Err()
}
