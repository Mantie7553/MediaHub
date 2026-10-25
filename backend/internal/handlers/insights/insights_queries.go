package insights

import (
	"database/sql"
	"time"
)

// how far back the watch, read and job figures look
const window = "30 days"

/*
Function:	topWatched
Purpose:	the anime and movies the most people watched in the window, with how many
episodes were started and the hours actually played (counted since 2026-10-24)
*/
func topWatched(db *sql.DB) ([]topTitle, error) {
	rows, err := db.Query(`
		WITH watch AS (
			SELECT e.media_item_id, p.user_id, p.updated_at
			FROM user_episode_progress p
			JOIN episodes e ON e.id = p.episode_id
			UNION ALL
			SELECT media_item_id, user_id, updated_at
			FROM user_movie_progress
		), played AS (
			SELECT media_item_id, SUM(seconds) AS seconds
			FROM watch_time
			WHERE day > CURRENT_DATE - INTERVAL '` + window + `'
			GROUP BY media_item_id
		)
		SELECT mi.id, mi.type, mi.title, COUNT(DISTINCT w.user_id), COUNT(*),
			COALESCE(MAX(p.seconds), 0) / 3600.0
		FROM watch w
		JOIN media_items mi ON mi.id = w.media_item_id
		LEFT JOIN played p ON p.media_item_id = w.media_item_id
		WHERE w.updated_at > NOW() - INTERVAL '` + window + `'
		GROUP BY mi.id, mi.type, mi.title
		ORDER BY 4 DESC, 6 DESC
		LIMIT 10`)
	if err != nil {
		return nil, err
	}
	return scanTop(rows, true)
}

/*
Function:	topRead
Purpose:	the manga and light novels the most people read in the window, with how many
chapters or volumes were opened
*/
func topRead(db *sql.DB) ([]topTitle, error) {
	rows, err := db.Query(`
		WITH reads AS (
			SELECT media_item_id, user_id, updated_at FROM manga_progress
			UNION ALL
			SELECT media_item_id, user_id, updated_at FROM light_novel_progress
		)
		SELECT mi.id, mi.type, mi.title, COUNT(DISTINCT r.user_id), COUNT(*)
		FROM reads r
		JOIN media_items mi ON mi.id = r.media_item_id
		WHERE r.updated_at > NOW() - INTERVAL '` + window + `'
		GROUP BY mi.id, mi.type, mi.title
		ORDER BY 4 DESC, 5 DESC
		LIMIT 10`)
	if err != nil {
		return nil, err
	}
	return scanTop(rows, false)
}

func scanTop(rows *sql.Rows, withHours bool) ([]topTitle, error) {
	defer rows.Close()
	titles := []topTitle{}
	for rows.Next() {
		var t topTitle
		dest := []any{&t.ID, &t.Type, &t.Title, &t.Users, &t.Items}
		if withHours {
			dest = append(dest, &t.Hours)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		titles = append(titles, t)
	}
	return titles, rows.Err()
}

/*
Function:	users
Purpose:	every user with when they were last seen and how many titles they watched or read
in the window. Last seen is the later of their last sign in or refresh and their last saved
progress, so users who haven't signed in since last_seen_at was added still show up
*/
func users(db *sql.DB, now time.Time) (userSummary, error) {
	rows, err := db.Query(`
		WITH activity AS (
			SELECT e.media_item_id, p.user_id, p.updated_at
			FROM user_episode_progress p JOIN episodes e ON e.id = p.episode_id
			UNION ALL SELECT media_item_id, user_id, updated_at FROM user_movie_progress
			UNION ALL SELECT media_item_id, user_id, updated_at FROM manga_progress
			UNION ALL SELECT media_item_id, user_id, updated_at FROM light_novel_progress
		), per_user AS (
			SELECT user_id, MAX(updated_at) AS last_at,
				COUNT(DISTINCT media_item_id) FILTER (
					WHERE updated_at > NOW() - INTERVAL '` + window + `'
				) AS recent
			FROM activity
			GROUP BY user_id
		)
		SELECT u.id, u.username, u.role, GREATEST(u.last_seen_at, a.last_at), COALESCE(a.recent, 0)
		FROM users u
		LEFT JOIN per_user a ON a.user_id = u.id
		ORDER BY 4 DESC NULLS LAST, u.username`)
	if err != nil {
		return userSummary{}, err
	}
	defer rows.Close()

	list := []userActivity{}
	for rows.Next() {
		var u userActivity
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.LastSeen, &u.RecentTitles); err != nil {
			return userSummary{}, err
		}
		list = append(list, u)
	}
	if err := rows.Err(); err != nil {
		return userSummary{}, err
	}
	return summarizeUsers(list, now), nil
}

/*
Function:	summarizeUsers
Purpose:	the user list with how many were seen in the last 7 and 30 days
*/
func summarizeUsers(list []userActivity, now time.Time) userSummary {
	s := userSummary{Total: len(list), List: list}
	for _, u := range list {
		if u.LastSeen == nil {
			continue
		}
		age := now.Sub(*u.LastSeen)
		if age <= 7*24*time.Hour {
			s.Active7++
		}
		if age <= 30*24*time.Hour {
			s.Active30++
		}
	}
	return s
}

/*
Function:	jobs
Purpose:	download jobs by status in the window, requests waiting for approval, and the most
recent failures still on the Downloads page
*/
func jobs(db *sql.DB) (jobSummary, error) {
	s := jobSummary{Counts: map[string]int{}, RecentFailures: []failedJob{}}

	rows, err := db.Query(`
		SELECT status, COUNT(*) FROM download_jobs
		WHERE created_at > NOW() - INTERVAL '` + window + `'
		GROUP BY status`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return s, err
		}
		s.Counts[status] = n
	}
	if err := rows.Err(); err != nil {
		return s, err
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM download_requests WHERE status = 'pending'`).Scan(&s.PendingRequests); err != nil {
		return s, err
	}

	failed, err := db.Query(`
		SELECT j.id, mi.title, j.handler, j.error_message, COALESCE(j.completed_at, j.started_at, j.created_at)
		FROM download_jobs j
		JOIN media_items mi ON mi.id = j.media_item_id
		WHERE j.status = 'failed' AND j.archived_at IS NULL
		ORDER BY 5 DESC
		LIMIT 10`)
	if err != nil {
		return s, err
	}
	defer failed.Close()
	for failed.Next() {
		var f failedJob
		if err := failed.Scan(&f.ID, &f.Title, &f.Handler, &f.Error, &f.FailedAt); err != nil {
			return s, err
		}
		s.RecentFailures = append(s.RecentFailures, f)
	}
	return s, failed.Err()
}
