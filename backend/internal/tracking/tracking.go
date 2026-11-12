// Package tracking keeps a user's list status for a title and their progress through its
// episodes, chapters or volumes in step: completing the title marks every part done, and marking
// parts done or not works the status out again
package tracking

import (
	"database/sql"
	"errors"
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

	_, err = q.Exec(query, userID, mediaItemID)
	return err
}

/*
Function:	SyncStatus
Purpose:	Work out a user's list status for a title from how many of its parts they have done, and
save it when it changes. The rating is never touched
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

	var query string
	switch t {
	case "anime":
		query = `SELECT COUNT(*) FILTER (WHERE p.watched), COUNT(*)
			FROM episodes e
			LEFT JOIN user_anime_progress p ON p.episode_id = e.id AND p.user_id = $1
			WHERE e.media_item_id = $2`
	case "manga":
		query = `SELECT COUNT(*) FILTER (WHERE p.completed), COUNT(*)
			FROM manga_chapters c
			LEFT JOIN manga_progress p ON p.chapter_id = c.id AND p.user_id = $1
			WHERE c.media_item_id = $2`
	case "light_novel":
		query = `SELECT COUNT(*) FILTER (WHERE p.completed), COUNT(*)
			FROM light_novel_volumes v
			LEFT JOIN light_novel_progress p ON p.volume_id = v.id AND p.user_id = $1
			WHERE v.media_item_id = $2`
	default:
		return nil
	}

	var done, total int
	if err := q.QueryRow(query, userID, mediaItemID).Scan(&done, &total); err != nil {
		return err
	}

	var current *string
	var s string
	err = q.QueryRow(
		`SELECT status FROM user_media_status WHERE user_id = $1 AND media_item_id = $2`,
		userID, mediaItemID,
	).Scan(&s)
	switch {
	case err == nil:
		current = &s
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}

	status, change := nextStatus(done, total, current)
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
