package insights

import (
	"database/sql"
	"encoding/json"
	"time"
)

// how many days the activity chart covers, today included
const activityDays = 30

// how far back the storage chart goes
const storageHistoryDays = 90

/*
Function:	topPlayed
Purpose:	the tracks played the most in the window, with how many people played them
*/
func topPlayed(db *sql.DB) ([]topTrack, error) {
	rows, err := db.Query(`
		SELECT mi.id, mi.title, mm.artist, mm.album_id, COUNT(DISTINCT p.user_id), COUNT(*)
		FROM music_plays p
		JOIN media_items mi ON mi.id = p.media_item_id
		JOIN music_metadata mm ON mm.media_item_id = p.media_item_id
		WHERE p.played_at > NOW() - INTERVAL '` + window + `'
		GROUP BY mi.id, mi.title, mm.artist, mm.album_id
		ORDER BY 6 DESC, 5 DESC
		LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracks := []topTrack{}
	for rows.Next() {
		var t topTrack
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.AlbumID, &t.Users, &t.Plays); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

/*
Function:	activity
Purpose:	for each of the last activityDays days, how many people were around and how many
tracks were played. Days with nothing are included as zeros
*/
func activity(db *sql.DB) ([]activityDay, error) {
	rows, err := db.Query(`
		SELECT d::date,
			(SELECT COUNT(*) FROM user_active_days a WHERE a.day = d::date),
			(SELECT COUNT(*) FROM music_plays p WHERE p.played_at >= d AND p.played_at < d + INTERVAL '1 day')
		FROM generate_series(CURRENT_DATE - $1::int, CURRENT_DATE, INTERVAL '1 day') d
		ORDER BY 1`, activityDays-1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	days := []activityDay{}
	for rows.Next() {
		var day time.Time
		var a activityDay
		if err := rows.Scan(&day, &a.Users, &a.Plays); err != nil {
			return nil, err
		}
		a.Day = day.Format(time.DateOnly)
		days = append(days, a)
	}
	return days, rows.Err()
}

/*
Function:	storageHistory
Purpose:	the daily storage snapshots of the last storageHistoryDays days, oldest first
*/
func storageHistory(db *sql.DB) ([]storagePoint, error) {
	rows, err := db.Query(`
		SELECT day, library_bytes, disk_used, disk_total
		FROM storage_snapshots
		WHERE day > CURRENT_DATE - $1::int
		ORDER BY day`, storageHistoryDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := []storagePoint{}
	for rows.Next() {
		var day time.Time
		var p storagePoint
		if err := rows.Scan(&day, &p.LibraryBytes, &p.DiskUsed, &p.DiskTotal); err != nil {
			return nil, err
		}
		p.Day = day.Format(time.DateOnly)
		points = append(points, p)
	}
	return points, rows.Err()
}

/*
Function:	snapshotOf
Purpose:	what a scan adds up to, as stored in a day's snapshot: the library's total, bytes per
media type, and the disk's used and total space when they could be read
*/
func snapshotOf(u storageUsage) (library int64, folders map[string]int64, diskUsed, diskTotal *int64) {
	folders = map[string]int64{}
	for _, f := range u.Folders {
		folders[f.Type] = f.Bytes
		library += f.Bytes
	}
	if u.DiskTotal != nil && u.DiskFree != nil {
		used := *u.DiskTotal - *u.DiskFree
		diskUsed, diskTotal = &used, u.DiskTotal
	}
	return library, folders, diskUsed, diskTotal
}

/*
Function:	recordSnapshot
Purpose:	save a scan as today's storage snapshot, replacing an earlier one from today
*/
func recordSnapshot(db *sql.DB, u storageUsage) error {
	library, folders, diskUsed, diskTotal := snapshotOf(u)
	perType, err := json.Marshal(folders)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO storage_snapshots (day, library_bytes, folders, disk_used, disk_total)
		VALUES (CURRENT_DATE, $1, $2, $3, $4)
		ON CONFLICT (day) DO UPDATE SET
			library_bytes = EXCLUDED.library_bytes, folders = EXCLUDED.folders,
			disk_used = EXCLUDED.disk_used, disk_total = EXCLUDED.disk_total`,
		library, perType, diskUsed, diskTotal)
	return err
}
