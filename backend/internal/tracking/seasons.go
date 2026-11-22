package tracking

import (
	"database/sql"
	"errors"
)

// ShowEntry is one AniList/MyAnimeList entry of a series, as Sonarr's one title holds them:
// a season, a cour or a special, with how many episodes it has (0 when unknown)
type ShowEntry struct {
	AnilistID int
	Season    int
	Episodes  int
}

/*
Function:	SeasonProgress
Purpose:	What a status and progress given for one entry of a series (AniList and MyAnimeList
keep "Nisemonogatari" or "Season 2" apart) mean for the series as one title: every entry before
it counts as watched, then the episodes given of this one. Finishing an entry finishes the series
only when it is the last regular one. A special (season 0) moves nothing, the series' count puts
specials last so its episodes can't be counted on their own
Params:
  - entries: the series' entries in watch order, specials last
  - anilistID: the entry the status is for
  - status: the entry's list status
  - progress: episodes watched of the entry

Returns: the episodes watched of the series, and the series' status
*/
func SeasonProgress(entries []ShowEntry, anilistID int, status string, progress int) (int, string) {
	at := -1
	lastRegular := -1
	for i, e := range entries {
		if e.AnilistID == anilistID {
			at = i
		}
		if e.Season != 0 {
			lastRegular = i
		}
	}
	if at < 0 || entries[at].Season == 0 {
		if status == "completed" {
			status = "current"
		}
		if at < 0 {
			return max(progress, 0), status
		}
		return 0, status
	}

	before := 0
	for _, e := range entries[:at] {
		before += e.Episodes
	}
	progress = max(progress, 0)
	if e := entries[at].Episodes; e > 0 {
		progress = min(progress, e)
	}

	switch status {
	case "completed":
		watched := progress
		if e := entries[at].Episodes; e > 0 {
			watched = e
		}
		if at == lastRegular {
			return before + watched, "completed"
		}
		return before + watched, "current"
	case "planned":
		// planning a later entry says nothing about the earlier ones
		return 0, status
	default:
		if at > 0 && status == "current" && progress == 0 {
			// started on it: everything before is watched
			return before, status
		}
		return before + progress, status
	}
}

// how far along a status says someone is, for picking between two saves of the same series
func statusRank(status string) int {
	switch status {
	case "completed":
		return 4
	case "current", "dropped":
		return 3
	case "planned":
		return 2
	}
	return 0
}

/*
Function:	MergeStatus
Purpose:	Combine a series' list entry with another entry's save, so saving entries in any order
(an import going season by season) never moves the user backwards: the larger episode count
wins, and the status that came with it. The rating already there stays
Params:
  - oldCount, oldStatus: what the entry has now
  - newCount, newStatus: what the save says

Returns: the count and status to keep
*/
func MergeStatus(oldCount int, oldStatus string, newCount int, newStatus string) (int, string) {
	switch {
	case newCount > oldCount:
		return newCount, newStatus
	case newCount < oldCount:
		return oldCount, oldStatus
	case statusRank(newStatus) > statusRank(oldStatus):
		return oldCount, newStatus
	default:
		return oldCount, oldStatus
	}
}

/*
Function:	SaveSeriesEntry
Purpose:	Put a series on a user's list from one of its entries' saves (see SeasonProgress),
merged with what their list already has for it (see MergeStatus)
Params:
  - q: the database or a transaction
  - userID: the user
  - mediaItemID: the series' title
  - status: the series' status from SeasonProgress
  - rating: the rating given for the whole series, kept unless the user already picked one
  - count: the series' episodes watched from SeasonProgress
*/
func SaveSeriesEntry(q Querier, userID, mediaItemID, status string, rating *int, count int) error {
	var oldStatus string
	var oldCount sql.NullInt64
	var watched int
	err := q.QueryRow(
		`SELECT s.status, s.progress_count,
		(SELECT COUNT(*) FROM user_anime_progress p JOIN episodes e ON e.id = p.episode_id
		 WHERE p.user_id = s.user_id AND e.media_item_id = s.media_item_id AND p.watched)
		FROM user_media_status s WHERE s.user_id = $1 AND s.media_item_id = $2`,
		userID, mediaItemID,
	).Scan(&oldStatus, &oldCount, &watched)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := q.Exec(
			`INSERT INTO user_media_status (user_id, media_item_id, status, rating) VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, media_item_id) DO NOTHING`,
			userID, mediaItemID, status, rating,
		); err != nil {
			return err
		}
		if count > 0 {
			return SetProgressCount(q, userID, mediaItemID, count)
		}
		return nil
	}
	if err != nil {
		return err
	}

	had := max(int(oldCount.Int64), watched)
	keepCount, keepStatus := MergeStatus(had, oldStatus, count, status)
	if _, err := q.Exec(
		`UPDATE user_media_status SET status = $3,
		rating = CASE WHEN $4::int IS NOT NULL AND (rating IS NULL OR NOT rating_is_set) THEN $4 ELSE rating END,
		rating_is_set = rating_is_set OR $4::int IS NOT NULL,
		updated_at = NOW()
		WHERE user_id = $1 AND media_item_id = $2`,
		userID, mediaItemID, keepStatus, rating,
	); err != nil {
		return err
	}
	if keepCount > had {
		return SetProgressCount(q, userID, mediaItemID, keepCount)
	}
	return nil
}
