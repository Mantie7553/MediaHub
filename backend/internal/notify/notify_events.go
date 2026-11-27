package notify

import (
	"database/sql"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

// what a download job is of, for its notifications
type jobFacts struct {
	mediaItemID string
	title       string
	mediaType   string
	albumID     string
	handler     string
}

func lookupJob(db *sql.DB, jobID string) (jobFacts, bool) {
	var f jobFacts
	var albumID sql.NullString
	err := db.QueryRow(
		`SELECT mi.id, mi.title, mi.type::text, mm.album_id::text, j.handler
		FROM download_jobs j
		JOIN media_items mi ON mi.id = j.media_item_id
		LEFT JOIN music_metadata mm ON mm.media_item_id = mi.id
		WHERE j.id = $1`,
		jobID,
	).Scan(&f.mediaItemID, &f.title, &f.mediaType, &albumID, &f.handler)
	if err != nil {
		logger.Warn("looking up job %s to notify about it: %s", jobID, err.Error())
		return f, false
	}
	f.albumID = albumID.String
	return f, true
}

// a request's title and who made it
func lookupRequest(db *sql.DB, requestID string) (title, requester string, ok bool) {
	var by sql.NullString
	err := db.QueryRow(
		`SELECT COALESCE(mi.title, r.title_override, 'Untitled'), u.username
		FROM download_requests r
		LEFT JOIN media_items mi ON mi.id = r.media_item_id
		LEFT JOIN users u ON u.id = r.requested_by
		WHERE r.id = $1`,
		requestID,
	).Scan(&title, &by)
	if err != nil {
		logger.Warn("looking up request %s to notify about it: %s", requestID, err.Error())
		return "", "", false
	}
	return title, by.String, true
}

func requestedBy(requester string) string {
	if requester == "" {
		return ""
	}
	return "Requested by " + requester
}

/*
Function:	RequestPending
Purpose:	tell the admins a request waits for their approval
*/
func RequestPending(db *sql.DB, requestID string) {
	title, requester, ok := lookupRequest(db, requestID)
	if !ok {
		return
	}
	Admins(db, Notification{
		Kind:  KindApprovalNeeded,
		Title: title + " needs approval",
		Body:  requestedBy(requester),
		Link:  RequestLink(requestID),
		Key:   RequestKey(requestID),
	})
}

/*
Function:	ByHand
Purpose:	tell the admins an approved request is one they download themselves (light novels)
*/
func ByHand(db *sql.DB, requestID string) {
	title, requester, ok := lookupRequest(db, requestID)
	if !ok {
		return
	}
	body := "Get it from elscione, then add its volumes"
	if requester != "" {
		body = requestedBy(requester) + ". " + body
	}
	Admins(db, Notification{
		Kind:  KindByHand,
		Title: title + " needs downloading by hand",
		Body:  body,
		Link:  RequestLink(requestID),
		Key:   RequestKey(requestID),
	})
}

/*
Function:	JobFailed
Purpose:	tell the admins a download failed, and why
*/
func JobFailed(db *sql.DB, jobID, message string) {
	f, ok := lookupJob(db, jobID)
	if !ok {
		return
	}
	Admins(db, Notification{
		Kind:  KindDownloadFailed,
		Title: f.title + " failed to download",
		Body:  message,
		Link:  JobLink(jobID),
		Key:   JobKey(jobID),
	})
}

/*
Function:	JobDownloading
Purpose:	tell the admins new content started downloading. One notification a title until it's
read, however many jobs and Sonarr / Radarr grabs follow
*/
func JobDownloading(db *sql.DB, jobID string) {
	f, ok := lookupJob(db, jobID)
	if !ok {
		return
	}
	Admins(db, Notification{
		Kind:  KindDownloading,
		Title: f.title + " is downloading",
		Body:  handlerNote(f.handler, f.mediaType),
		Link:  DetailsLink(f.mediaType, f.mediaItemID, f.albumID),
		Key:   DownloadingKey(f.mediaItemID),
	})
}

/*
Function:	Grabbed
Purpose:	tell the admins Sonarr or Radarr grabbed something to download
Params:
  - db: the database
  - mediaItemID: the title, "" when MediaHub doesn't have it yet
  - mediaType: its type
  - title: its name
  - key: what it is about, for one MediaHub doesn't have
  - body: what was grabbed
*/
func Grabbed(db *sql.DB, mediaItemID, mediaType, title, key, body string) {
	n := Notification{Kind: KindDownloading, Title: title + " is downloading", Body: body, Link: "/downloads", Key: key}
	if mediaItemID != "" {
		n.Link = DetailsLink(mediaType, mediaItemID, "")
		n.Key = DownloadingKey(mediaItemID)
	}
	Admins(db, n)
}

/*
Function:	handlerNote
Purpose:	what a download job does, for its notification
*/
func handlerNote(handler, mediaType string) string {
	switch handler {
	case "sonarr":
		return "Sent to Sonarr"
	case "radarr":
		return "Sent to Radarr"
	case "mangadex":
		return "Fetching new chapters"
	case "ytdlp":
		if mediaType == "music_track" {
			return "Downloading the track from YouTube"
		}
		return "Downloading from YouTube"
	}
	return ""
}
