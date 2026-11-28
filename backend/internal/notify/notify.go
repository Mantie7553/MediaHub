/*
Package notify tells people about things that need them, in the app. Each recipient gets a
row of their own, read once its read_at is set. Admins are told about requests to approve,
downloads that failed and new content downloading; every user about their own requests and
about titles they follow getting new content. Anyone can turn a kind off (notification_mutes)
*/
package notify

import (
	"database/sql"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

// what a notification is about
const (
	KindApprovalNeeded = "approval_needed"
	KindDownloadFailed = "download_failed"
	KindDownloading    = "downloading"
	KindByHand         = "download_by_hand"

	KindRequestApproved = "request_approved"
	KindRequestRejected = "request_rejected"
	KindTitleReady      = "title_ready"
	KindNewParts        = "new_parts"
)

// AdminKinds are only sent to admins, UserKinds to anyone
var (
	AdminKinds = []string{KindApprovalNeeded, KindDownloadFailed, KindDownloading, KindByHand}
	UserKinds  = []string{KindRequestApproved, KindRequestRejected, KindTitleReady, KindNewParts}
)

/*
Function:	KindsFor
Purpose:	the kinds someone can be sent, so the ones they can turn off
*/
func KindsFor(admin bool) []string {
	if admin {
		return slices.Concat(UserKinds, AdminKinds)
	}
	return slices.Clone(UserKinds)
}

// a recipient who turned the notification's kind off isn't told
const notMuted = `NOT EXISTS (SELECT 1 FROM notification_mutes m WHERE m.user_id = u.id AND m.kind = $1)`

// Notification is one thing to tell people about
type Notification struct {
	Kind  string
	Title string
	Body  string
	// the page to open, where the recipient acts on it
	Link string
	// what it is about: a recipient is told once until they read it
	Key string
}

// the longest body kept, a yt-dlp failure can run to pages
const maxBody = 240

/*
Function:	Admins
Purpose:	tell every admin. An admin who hasn't read the last one with the same key isn't told
again. Failing to save it is logged, never returned: a notification never stops a download
or a request
Params:
  - db: the database
  - n: what to tell them
*/
func Admins(db *sql.DB, n Notification) {
	if _, err := db.Exec(
		`INSERT INTO notifications (user_id, kind, title, body, link, dedupe_key)
		SELECT u.id, $1, $2, NULLIF($3, ''), NULLIF($4, ''), $5 FROM users u
		WHERE u.role = 'admin' AND `+notMuted+`
		ON CONFLICT (user_id, dedupe_key) WHERE read_at IS NULL DO NOTHING`,
		n.Kind, n.Title, Shorten(n.Body, maxBody), n.Link, n.Key,
	); err != nil {
		logger.Warn("notifying admins (%s): %s", n.Key, err.Error())
	}
}

/*
Function:	Users
Purpose:	tell these users. One who hasn't read the last one with the same key has it replaced,
so it comes back to the top with the newer news instead of a second one. Failing to save it is
logged, never returned
Params:
  - db: the database
  - userIDs: who to tell
  - n: what to tell them
*/
func Users(db *sql.DB, userIDs []string, n Notification) {
	if len(userIDs) == 0 {
		return
	}
	if _, err := db.Exec(
		`INSERT INTO notifications (user_id, kind, title, body, link, dedupe_key)
		SELECT u.id, $1, $2, NULLIF($3, ''), NULLIF($4, ''), $5 FROM users u
		WHERE u.id = ANY($6::uuid[]) AND `+notMuted+`
		ON CONFLICT (user_id, dedupe_key) WHERE read_at IS NULL DO UPDATE SET
		kind = EXCLUDED.kind, title = EXCLUDED.title, body = EXCLUDED.body,
		link = EXCLUDED.link, created_at = NOW()`,
		n.Kind, n.Title, Shorten(n.Body, maxBody), n.Link, n.Key, pq.Array(userIDs),
	); err != nil {
		logger.Warn("notifying users (%s): %s", n.Key, err.Error())
	}
}

/*
Function:	Resolve
Purpose:	mark everyone's unread notifications with this key read, once what they asked for is
done (a request someone approved)
Params:
  - db: the database
  - key: what they are about
*/
func Resolve(db *sql.DB, key string) {
	if _, err := db.Exec(
		`UPDATE notifications SET read_at = NOW() WHERE dedupe_key = $1 AND read_at IS NULL`, key,
	); err != nil {
		logger.Warn("resolving notifications (%s): %s", key, err.Error())
	}
}

/*
Function:	PruneRead
Purpose:	delete notifications read more than the given number of days ago
*/
func PruneRead(db *sql.DB, days int) (int64, error) {
	res, err := db.Exec(
		`DELETE FROM notifications WHERE read_at < NOW() - make_interval(days => $1)`, days,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// the keys notifications are told apart by
func RequestKey(requestID string) string       { return "request:" + requestID }
func JobKey(jobID string) string               { return "job:" + jobID }
func DownloadingKey(mediaItemID string) string { return "downloading:" + mediaItemID }

// the keys of what users are told: apart from the admins' so resolving those leaves them be
func MyRequestKey(requestID string) string { return "myrequest:" + requestID }
func TitleKey(mediaItemID string) string   { return "title:" + mediaItemID }
func AlbumKey(albumID string) string       { return "album:" + albumID }

// the Downloads page, showing one request or job
func RequestLink(requestID string) string { return "/downloads?request=" + requestID }
func JobLink(jobID string) string         { return "/downloads?job=" + jobID }

/*
Function:	DetailsLink
Purpose:	a title's page. A track opens its album; anything without a page opens Downloads
Params:
  - mediaType: the title's type
  - mediaItemID: the title
  - albumID: a track's album, "" when it has none
*/
func DetailsLink(mediaType, mediaItemID, albumID string) string {
	switch mediaType {
	case "anime":
		return "/anime/" + mediaItemID
	case "movie":
		return "/movies/" + mediaItemID
	case "manga":
		return "/manga/" + mediaItemID
	case "light_novel":
		return "/light-novels/" + mediaItemID
	case "music_track":
		if albumID != "" {
			return "/albums/" + albumID
		}
	}
	return "/downloads"
}

/*
Function:	Shorten
Purpose:	the first non-empty line of a message, cut to max characters with an ellipsis
*/
func Shorten(msg string, max int) string {
	line := ""
	for _, l := range strings.Split(msg, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			line = l
			break
		}
	}
	if utf8.RuneCountInString(line) <= max {
		return line
	}
	return string([]rune(line)[:max-1]) + "…"
}
