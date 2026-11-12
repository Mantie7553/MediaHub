// Package tracking keeps a user's list status for a title and their progress through its
// episodes, chapters or volumes in step: completing the title marks every part done, and marking
// parts done or not works the status out again
package tracking

import (
	"database/sql"
	"errors"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

// Querier is a *sql.DB or a *sql.Tx
type Querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

func mediaType(q Querier, mediaItemID string) (string, error) {
	var t string
	err := q.QueryRow(`SELECT type FROM media_items WHERE id = $1`, mediaItemID).Scan(&t)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return t, err
}

/*
Function:	CompleteAll
Purpose:	Mark every episode, chapter or volume of a title done for a user, as if they had finished
each one. Titles without parts (movies, music) are left as they are
Params:
  - q: the database or a transaction
  - userID: the user
  - mediaItemID: the title
*/
func CompleteAll(q Querier, userID, mediaItemID string) error {
	t, err := mediaType(q, mediaItemID)
	if err != nil {
		return err
	}

	var query string
	switch t {
	case "anime":
		// one already watched keeps when it was
		query = `INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at)
			SELECT $1, e.id, true, NOW() FROM episodes e WHERE e.media_item_id = $2
			ON CONFLICT (user_id, episode_id) DO UPDATE SET watched = true, watched_at = NOW()
			WHERE user_anime_progress.watched IS NOT TRUE`
	case "manga":
		query = `INSERT INTO manga_progress (user_id, chapter_id, media_item_id, last_page_read, completed, updated_at)
			SELECT $1, c.id, c.media_item_id, GREATEST(COALESCE(c.page_count, 1) - 1, 0), true, NOW()
			FROM manga_chapters c WHERE c.media_item_id = $2
			ON CONFLICT (user_id, chapter_id) DO UPDATE SET
			last_page_read = EXCLUDED.last_page_read, completed = true, updated_at = NOW()
			WHERE manga_progress.completed IS NOT TRUE`
	case "light_novel":
		// the jump to the end clears the old anchor and bookmark, like marking a volume read does
		query = `INSERT INTO light_novel_progress (user_id, volume_id, media_item_id, scroll_position, completed, updated_at)
			SELECT $1, v.id, v.media_item_id, 1, true, NOW()
			FROM light_novel_volumes v WHERE v.media_item_id = $2
			ON CONFLICT (user_id, volume_id) DO UPDATE SET
			scroll_position = 1, completed = true, anchor_block = NULL, anchor_char = NULL,
			bookmark_block = NULL, updated_at = NOW()
			WHERE light_novel_progress.completed IS NOT TRUE`
	default:
		return nil
	}

	if _, err = q.Exec(query, userID, mediaItemID); err != nil {
		return err
	}

	// the count covers what the source has that the library doesn't
	parts, err := partsOf(q, t, userID, mediaItemID)
	if err != nil {
		return err
	}
	_, err = q.Exec(`UPDATE user_media_status SET progress_count = $3 WHERE user_id = $1 AND media_item_id = $2`,
		userID, mediaItemID, parts.Total())
	return err
}

// Parts is how far a user is through a title's episodes, chapters or volumes
type Parts struct {
	// Done is how many of the library's files they've finished
	Done int
	// Files is how many the library has
	Files int
	// Source is how many the title has according to its metadata, 0 when unknown
	Source int
}

// Total is how many parts the title has: the source's count, or the files when there are more
func (p Parts) Total() int {
	return max(p.Files, p.Source)
}

// CountHolds: a stored count is believable only while every file inside it is done
func (p Parts) CountHolds(count int) bool {
	return p.Done >= min(count, p.Files)
}

// partsQuery counts a user's done parts, the files and the source's total ($1 user, $2 title),
// "" for types without parts
func partsQuery(mediaType string) string {
	switch mediaType {
	case "anime":
		// Sonarr lists every episode, downloaded or not, so its rows are the source's count
		return `SELECT COUNT(*) FILTER (WHERE p.watched), COUNT(*), COUNT(*)
			FROM episodes e
			LEFT JOIN user_anime_progress p ON p.episode_id = e.id AND p.user_id = $1
			WHERE e.media_item_id = $2`
	case "manga":
		return `SELECT COUNT(*) FILTER (WHERE p.completed), COUNT(*),
			COALESCE((SELECT total_chapters FROM manga_metadata WHERE media_item_id = $2), 0)
			FROM manga_chapters c
			LEFT JOIN manga_progress p ON p.chapter_id = c.id AND p.user_id = $1
			WHERE c.media_item_id = $2`
	case "light_novel":
		return `SELECT COUNT(*) FILTER (WHERE p.completed), COUNT(*),
			COALESCE((SELECT total_volumes FROM light_novel_metadata WHERE media_item_id = $2), 0)
			FROM light_novel_volumes v
			LEFT JOIN light_novel_progress p ON p.volume_id = v.id AND p.user_id = $1
			WHERE v.media_item_id = $2`
	default:
		return ""
	}
}

func partsOf(q Querier, mediaType, userID, mediaItemID string) (Parts, error) {
	var p Parts
	query := partsQuery(mediaType)
	if query == "" {
		return p, nil
	}
	err := q.QueryRow(query, userID, mediaItemID).Scan(&p.Done, &p.Files, &p.Source)
	return p, err
}

/*
Function:	SyncStatus
Purpose:	Work out a user's list status for a title from how many of its parts they have done (or
the count they gave, while it holds) against how many it has, and save it when it changes. The
rating is never touched
Params:
  - q: the database or a transaction
  - userID: the user
  - mediaItemID: the title
*/
func SyncStatus(q Querier, userID, mediaItemID string) error {
	t, err := mediaType(q, mediaItemID)
	if err != nil {
		return err
	}
	if partsQuery(t) == "" {
		return nil
	}
	parts, err := partsOf(q, t, userID, mediaItemID)
	if err != nil {
		return err
	}

	var current *string
	var s string
	var count sql.NullInt64
	err = q.QueryRow(
		`SELECT status, progress_count FROM user_media_status WHERE user_id = $1 AND media_item_id = $2`,
		userID, mediaItemID,
	).Scan(&s, &count)
	switch {
	case err == nil:
		current = &s
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}

	// a count stands only while every file inside it is done; marking one of those not done
	// goes back to what the files say
	if count.Valid && !parts.CountHolds(int(count.Int64)) {
		if _, err := q.Exec(`UPDATE user_media_status SET progress_count = NULL WHERE user_id = $1 AND media_item_id = $2`,
			userID, mediaItemID); err != nil {
			return err
		}
		count.Valid = false
	}
	done := parts.Done
	if count.Valid {
		done = max(done, int(count.Int64))
	}

	status, change := nextStatus(done, parts.Total(), current)
	if !change {
		return nil
	}

	_, err = q.Exec(
		`INSERT INTO user_media_status (user_id, media_item_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, media_item_id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()`,
		userID, mediaItemID, status,
	)
	return err
}

/*
Function:	SyncStatusOf
Purpose:	SyncStatus for the title a part (episode, chapter, volume) belongs to
Params:
  - q: the database or a transaction
  - userID: the user
  - lookup: a query taking the part's id as $1 and giving its media_item_id
  - partID: the part
*/
func SyncStatusOf(q Querier, userID, lookup, partID string) error {
	var mediaItemID string
	err := q.QueryRow(lookup, partID).Scan(&mediaItemID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return SyncStatus(q, userID, mediaItemID)
}

// the media_item_id lookups for SyncStatusOf
const (
	EpisodeTitle = `SELECT media_item_id FROM episodes WHERE id = $1`
	ChapterTitle = `SELECT media_item_id FROM manga_chapters WHERE id = $1`
	VolumeTitle  = `SELECT media_item_id FROM light_novel_volumes WHERE id = $1`
)

/*
Function:	nextStatus
Purpose:	The list status a title's progress calls for: everything done is completed, some done is
current, and nothing done takes a title the user was on (current or completed) back to planned.
Nothing done otherwise leaves the entry alone, and never adds one
Params:
  - done: the parts the user has done
  - total: the parts the title has
  - current: the user's status now, nil when the title isn't on their list

Returns: the status, and whether it differs from the current one
*/
func nextStatus(done, total int, current *string) (string, bool) {
	var want string
	switch {
	case total == 0:
		return "", false
	case done >= total:
		want = "completed"
	case done > 0:
		want = "current"
	case current != nil && (*current == "current" || *current == "completed"):
		want = "planned"
	default:
		return "", false
	}
	if current != nil && *current == want {
		return want, false
	}
	return want, true
}

// marks the first $3 chapters by number read (a chapter already read keeps its page and time)
// and the ones after them unread, for a progress count picked by hand
const (
	setChaptersRead = `WITH ranked AS (
		SELECT id, media_item_id, page_count, ROW_NUMBER() OVER (ORDER BY chapter_number) AS n
		FROM manga_chapters WHERE media_item_id = $2
	)
	INSERT INTO manga_progress (user_id, chapter_id, media_item_id, last_page_read, completed, updated_at)
	SELECT $1, r.id, r.media_item_id, GREATEST(COALESCE(r.page_count, 1) - 1, 0), true, NOW()
	FROM ranked r WHERE r.n <= $3
	ON CONFLICT (user_id, chapter_id) DO UPDATE SET
	last_page_read = EXCLUDED.last_page_read, completed = true, updated_at = NOW()
	WHERE manga_progress.completed IS NOT TRUE`
	setChaptersUnread = `WITH ranked AS (
		SELECT id, ROW_NUMBER() OVER (ORDER BY chapter_number) AS n
		FROM manga_chapters WHERE media_item_id = $2
	)
	UPDATE manga_progress p SET completed = false, updated_at = NOW()
	FROM ranked r
	WHERE p.chapter_id = r.id AND p.user_id = $1 AND r.n > $3 AND p.completed`

	setVolumesRead = `WITH ranked AS (
		SELECT id, media_item_id, ROW_NUMBER() OVER (ORDER BY volume_number) AS n
		FROM light_novel_volumes WHERE media_item_id = $2
	)
	INSERT INTO light_novel_progress (user_id, volume_id, media_item_id, scroll_position, completed, updated_at)
	SELECT $1, r.id, r.media_item_id, 1, true, NOW()
	FROM ranked r WHERE r.n <= $3
	ON CONFLICT (user_id, volume_id) DO UPDATE SET
	scroll_position = 1, completed = true, anchor_block = NULL, anchor_char = NULL,
	bookmark_block = NULL, updated_at = NOW()
	WHERE light_novel_progress.completed IS NOT TRUE`
	setVolumesUnread = `WITH ranked AS (
		SELECT id, ROW_NUMBER() OVER (ORDER BY volume_number) AS n
		FROM light_novel_volumes WHERE media_item_id = $2
	)
	UPDATE light_novel_progress p SET completed = false, updated_at = NOW()
	FROM ranked r
	WHERE p.volume_id = r.id AND p.user_id = $1 AND r.n > $3 AND p.completed`
)

// marks the first $3 episodes watched, specials last (one already watched keeps when it was),
// and the ones after them unwatched
const (
	setEpisodesWatched = `WITH ranked AS (
		SELECT id, ROW_NUMBER() OVER (ORDER BY season_number = 0, season_number, episode_number) AS n
		FROM episodes WHERE media_item_id = $2
	)
	INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at)
	SELECT $1, r.id, true, NOW() FROM ranked r WHERE r.n <= $3
	ON CONFLICT (user_id, episode_id) DO UPDATE SET watched = true, watched_at = NOW()
	WHERE user_anime_progress.watched IS NOT TRUE`
	setEpisodesUnwatched = `WITH ranked AS (
		SELECT id, ROW_NUMBER() OVER (ORDER BY season_number = 0, season_number, episode_number) AS n
		FROM episodes WHERE media_item_id = $2
	)
	UPDATE user_anime_progress p SET watched = false
	FROM ranked r
	WHERE p.episode_id = r.id AND p.user_id = $1 AND r.n > $3 AND p.watched`
)

// progressCountQueries are the statements SetProgressCount runs for a media type, none for
// types whose progress isn't a count of parts
func progressCountQueries(mediaType string) []string {
	switch mediaType {
	case "anime":
		return []string{setEpisodesWatched, setEpisodesUnwatched}
	case "manga":
		return []string{setChaptersRead, setChaptersUnread}
	case "light_novel":
		return []string{setVolumesRead, setVolumesUnread}
	default:
		return nil
	}
}

/*
Function:	SetProgressCount
Purpose:	Make a user's progress through a title exactly count episodes, chapters or volumes: the
first count files done, the rest not, and the count kept on their list entry for the parts past
the library's files. The list status is left to the caller
Params:
  - q: the database or a transaction
  - userID: the user, whose list entry for the title must exist
  - mediaItemID: the title
  - count: how many parts, from the first
*/
func SetProgressCount(q Querier, userID, mediaItemID string, count int) error {
	t, err := mediaType(q, mediaItemID)
	if err != nil {
		return err
	}
	queries := progressCountQueries(t)
	if len(queries) == 0 {
		return nil
	}
	count = max(count, 0)
	for _, query := range queries {
		if _, err := q.Exec(query, userID, mediaItemID, count); err != nil {
			return err
		}
	}
	_, err = q.Exec(`UPDATE user_media_status SET progress_count = $3 WHERE user_id = $1 AND media_item_id = $2`,
		userID, mediaItemID, count)
	return err
}

// mark the first progress_count parts done for everyone with a count on the title ($1), for
// parts that arrived after the count was given. Nothing is marked not done
const (
	applyCountedEpisodes = `WITH ranked AS (
		SELECT id, ROW_NUMBER() OVER (ORDER BY season_number = 0, season_number, episode_number) AS n
		FROM episodes WHERE media_item_id = $1
	)
	INSERT INTO user_anime_progress (user_id, episode_id, watched, watched_at)
	SELECT ums.user_id, r.id, true, NOW()
	FROM user_media_status ums JOIN ranked r ON r.n <= ums.progress_count
	WHERE ums.media_item_id = $1 AND ums.progress_count > 0
	ON CONFLICT (user_id, episode_id) DO UPDATE SET watched = true, watched_at = NOW()
	WHERE user_anime_progress.watched IS NOT TRUE`
	applyCountedChapters = `WITH ranked AS (
		SELECT id, media_item_id, page_count, ROW_NUMBER() OVER (ORDER BY chapter_number) AS n
		FROM manga_chapters WHERE media_item_id = $1
	)
	INSERT INTO manga_progress (user_id, chapter_id, media_item_id, last_page_read, completed, updated_at)
	SELECT ums.user_id, r.id, r.media_item_id, GREATEST(COALESCE(r.page_count, 1) - 1, 0), true, NOW()
	FROM user_media_status ums JOIN ranked r ON r.n <= ums.progress_count
	WHERE ums.media_item_id = $1 AND ums.progress_count > 0
	ON CONFLICT (user_id, chapter_id) DO UPDATE SET
	last_page_read = EXCLUDED.last_page_read, completed = true, updated_at = NOW()
	WHERE manga_progress.completed IS NOT TRUE`
	applyCountedVolumes = `WITH ranked AS (
		SELECT id, media_item_id, ROW_NUMBER() OVER (ORDER BY volume_number) AS n
		FROM light_novel_volumes WHERE media_item_id = $1
	)
	INSERT INTO light_novel_progress (user_id, volume_id, media_item_id, scroll_position, completed, updated_at)
	SELECT ums.user_id, r.id, r.media_item_id, 1, true, NOW()
	FROM user_media_status ums JOIN ranked r ON r.n <= ums.progress_count
	WHERE ums.media_item_id = $1 AND ums.progress_count > 0
	ON CONFLICT (user_id, volume_id) DO UPDATE SET
	scroll_position = 1, completed = true, anchor_block = NULL, anchor_char = NULL,
	bookmark_block = NULL, updated_at = NOW()
	WHERE light_novel_progress.completed IS NOT TRUE`
)

/*
Function:	ApplyCountedProgress
Purpose:	After episodes, chapters or volumes are added to a title, mark the ones inside each
user's progress count done, so a chapter read elsewhere doesn't show as unread once it downloads
Params:
  - q: the database or a transaction
  - mediaItemID: the title the parts were added to
*/
func ApplyCountedProgress(q Querier, mediaItemID string) error {
	t, err := mediaType(q, mediaItemID)
	if err != nil {
		return err
	}
	var query string
	switch t {
	case "anime":
		query = applyCountedEpisodes
	case "manga":
		query = applyCountedChapters
	case "light_novel":
		query = applyCountedVolumes
	default:
		return nil
	}
	_, err = q.Exec(query, mediaItemID)
	return err
}

// PartsAdded runs ApplyCountedProgress after parts were added, logging a failure instead of
// failing what added them: the parts are there either way
func PartsAdded(q Querier, mediaItemID string) {
	if err := ApplyCountedProgress(q, mediaItemID); err != nil {
		logger.Warn("marking new parts of %s inside users' progress counts: %v", mediaItemID, err)
	}
}

/*
Function:	ProgressOf
Purpose:	How far a user is through a title, whether or not it's on their list: the parts done (or
the count they gave, when that's further) and how many it has
Params:
  - q: the database or a transaction
  - userID: the user
  - mediaItemID: the title

Returns:	count, the parts, and whether the title has parts at all (anime, manga, light novels)
*/
func ProgressOf(q Querier, userID, mediaItemID string) (int, Parts, bool, error) {
	t, err := mediaType(q, mediaItemID)
	if err != nil || partsQuery(t) == "" {
		return 0, Parts{}, false, err
	}
	parts, err := partsOf(q, t, userID, mediaItemID)
	if err != nil {
		return 0, parts, true, err
	}
	var count sql.NullInt64
	err = q.QueryRow(`SELECT progress_count FROM user_media_status WHERE user_id = $1 AND media_item_id = $2`,
		userID, mediaItemID).Scan(&count)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, parts, true, err
	}
	return max(parts.Done, int(count.Int64)), parts, true, nil
}
