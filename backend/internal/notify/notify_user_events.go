package notify

import (
	"database/sql"
	"fmt"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

// what a request is for and who made it, for the requester's notifications
type requestFacts struct {
	requester   string
	title       string
	mediaType   string
	mediaItemID string
	albumID     string
}

func lookupMyRequest(db *sql.DB, requestID string) (requestFacts, bool) {
	var f requestFacts
	var by, mediaType, mediaItemID, albumID sql.NullString
	err := db.QueryRow(
		`SELECT r.requested_by::text, COALESCE(mi.title, a.title, r.title_override, 'Untitled'),
			mi.type::text, mi.id::text, r.album_id::text
		FROM download_requests r
		LEFT JOIN media_items mi ON mi.id = r.media_item_id
		LEFT JOIN albums a ON a.id = r.album_id
		WHERE r.id = $1`,
		requestID,
	).Scan(&by, &f.title, &mediaType, &mediaItemID, &albumID)
	if err != nil {
		logger.Warn("looking up request %s to notify its requester: %s", requestID, err.Error())
		return f, false
	}
	f.requester, f.mediaType, f.mediaItemID, f.albumID = by.String, mediaType.String, mediaItemID.String, albumID.String
	return f, true
}

// the page a request opens: its title's, or its album's
func (f requestFacts) link() string {
	if f.albumID != "" {
		return "/albums/" + f.albumID
	}
	return DetailsLink(f.mediaType, f.mediaItemID, "")
}

/*
Function:	RequestApproved
Purpose:	tell the requester an admin approved their request. Nobody is told about a request they
approved themselves
Params:
  - db: the database
  - requestID: the request
  - approverID: the admin who approved it
*/
func RequestApproved(db *sql.DB, requestID, approverID string) {
	f, ok := lookupMyRequest(db, requestID)
	if !ok || f.requester == "" || f.requester == approverID {
		return
	}
	Users(db, []string{f.requester}, Notification{
		Kind:  KindRequestApproved,
		Title: f.title + " was approved",
		Body:  "It will download soon",
		Link:  f.link(),
		Key:   MyRequestKey(requestID),
	})
}

/*
Function:	RequestRejected
Purpose:	tell the requester an admin turned their request down, and why when the admin said
Params:
  - db: the database
  - requestID: the request
  - approverID: the admin who rejected it
  - note: the admin's reason, "" for none
*/
func RequestRejected(db *sql.DB, requestID, approverID, note string) {
	f, ok := lookupMyRequest(db, requestID)
	if !ok || f.requester == "" || f.requester == approverID {
		return
	}
	Users(db, []string{f.requester}, Notification{
		Kind:  KindRequestRejected,
		Title: f.title + " was not approved",
		Body:  note,
		Link:  f.link(),
		Key:   MyRequestKey(requestID),
	})
}

/*
Function:	TitleArrived
Purpose:	tell users new content of a title arrived. The open requests for it are complete now and
their requesters are told it's ready; everyone else following it (an older, finished request,
the wishlist, or watching / reading it) is told how many parts are new. A title with no parts
(a movie, a track) is ready for all of them. One unread notification a title, refreshed by the
next arrival
Params:
  - db: the database
  - mediaItemID: the title
  - added: how many parts arrived (episodes, chapters, volumes); 1 when unknown
*/
func TitleArrived(db *sql.DB, mediaItemID string, added int) {
	arrived(db, mediaItemID, added, true)
}

/*
Function:	RequestsFulfilled
Purpose:	complete a title's open requests now it has content, and tell only their requesters it's
ready: for a library sync, which can't tell what's new
Params:
  - db: the database
  - mediaItemID: the title
*/
func RequestsFulfilled(db *sql.DB, mediaItemID string) {
	arrived(db, mediaItemID, 0, false)
}

func arrived(db *sql.DB, mediaItemID string, added int, tellFollowers bool) {
	ready := completeRequests(db, "media_item_id", mediaItemID)
	if len(ready) == 0 && !tellFollowers {
		return
	}
	var title, mediaType string
	if err := db.QueryRow(`SELECT title, type::text FROM media_items WHERE id = $1`, mediaItemID).
		Scan(&title, &mediaType); err != nil {
		logger.Warn("looking up %s to notify about its new content: %s", mediaItemID, err.Error())
		return
	}

	var following []string
	if tellFollowers {
		var err error
		following, err = scanIDs(db.Query(
			`SELECT requested_by::text FROM download_requests
				WHERE media_item_id = $1 AND status = 'complete' AND requested_by IS NOT NULL
			UNION SELECT user_id::text FROM user_wishlist WHERE media_item_id = $1
			UNION SELECT user_id::text FROM user_media_status WHERE media_item_id = $1 AND status = 'current'`,
			mediaItemID,
		))
		if err != nil {
			logger.Warn("finding who follows %s: %s", mediaItemID, err.Error())
		}
	}
	ready, others := splitRecipients(ready, following)

	link := DetailsLink(mediaType, mediaItemID, "")
	readyNote := Notification{
		Kind: KindTitleReady, Title: title + " is ready", Body: readyBody(mediaType), Link: link, Key: TitleKey(mediaItemID),
	}
	part := partName(mediaType)
	if part == "" {
		Users(db, append(ready, others...), readyNote)
		return
	}
	Users(db, ready, readyNote)
	Users(db, others, Notification{
		Kind:  KindNewParts,
		Title: newPartsTitle(title, part, added),
		Body:  readyBody(mediaType),
		Link:  link,
		Key:   TitleKey(mediaItemID),
	})
}

/*
Function:	AlbumArrived
Purpose:	tell the requesters of an album it's ready, once all its tracks have downloaded: its
open requests are complete then
Params:
  - db: the database
  - albumID: the album
*/
func AlbumArrived(db *sql.DB, albumID string) {
	var title string
	var pending bool
	if err := db.QueryRow(
		`SELECT a.title, EXISTS (
			SELECT 1 FROM download_jobs j JOIN download_requests r ON r.id = j.request_id
			WHERE r.album_id = a.id AND j.status IN ('queued', 'downloading'))
		FROM albums a WHERE a.id = $1`,
		albumID,
	).Scan(&title, &pending); err != nil {
		logger.Warn("looking up album %s to notify about it: %s", albumID, err.Error())
		return
	}
	if pending {
		return
	}
	ready, _ := splitRecipients(completeRequests(db, "album_id", albumID), nil)
	Users(db, ready, Notification{
		Kind: KindTitleReady, Title: title + " is ready", Body: readyBody("music_track"),
		Link: "/albums/" + albumID, Key: AlbumKey(albumID),
	})
}

/*
Function:	JobArrived
Purpose:	tell users a download job's content arrived: its album's when the job's request was for
an album, otherwise its title's
Params:
  - db: the database
  - jobID: the finished job
  - added: how many parts it brought
*/
func JobArrived(db *sql.DB, jobID string, added int) {
	var mediaItemID string
	var albumID sql.NullString
	if err := db.QueryRow(
		`SELECT j.media_item_id::text, r.album_id::text
		FROM download_jobs j JOIN download_requests r ON r.id = j.request_id WHERE j.id = $1`,
		jobID,
	).Scan(&mediaItemID, &albumID); err != nil {
		logger.Warn("looking up job %s to notify about it: %s", jobID, err.Error())
		return
	}
	if albumID.Valid {
		AlbumArrived(db, albumID.String)
		return
	}
	TitleArrived(db, mediaItemID, added)
}

/*
Function:	completeRequests
Purpose:	mark the open requests for a title or album complete, now its content arrived, and the
admins' notifications about them read. Returns who made them
Params:
  - db: the database
  - column: media_item_id or album_id, never user input
  - id: the title or album
*/
func completeRequests(db *sql.DB, column, id string) []string {
	rows, err := db.Query(
		`UPDATE download_requests SET status = 'complete', resolved_at = COALESCE(resolved_at, NOW())
		WHERE `+column+` = $1 AND status IN ('approved', 'queued', 'downloading')
		RETURNING id::text, requested_by::text`,
		id,
	)
	if err != nil {
		logger.Warn("completing the requests of %s: %s", id, err.Error())
		return nil
	}
	var requestIDs, requesters []string
	for rows.Next() {
		var requestID string
		var by sql.NullString
		if rows.Scan(&requestID, &by) == nil {
			requestIDs = append(requestIDs, requestID)
			requesters = append(requesters, by.String)
		}
	}
	rows.Close()
	for _, requestID := range requestIDs {
		Resolve(db, RequestKey(requestID))
	}
	return requesters
}

/*
Function:	splitRecipients
Purpose:	who is told a title is ready and who that it has new parts: everyone ready once, and
everyone else following it once. Empty ids (a deleted requester) are dropped
*/
func splitRecipients(ready, following []string) (readyOut, others []string) {
	seen := map[string]bool{"": true}
	for _, id := range ready {
		if !seen[id] {
			seen[id] = true
			readyOut = append(readyOut, id)
		}
	}
	for _, id := range following {
		if !seen[id] {
			seen[id] = true
			others = append(others, id)
		}
	}
	return readyOut, others
}

// what a title's parts are called, "" for titles with none
func partName(mediaType string) string {
	switch mediaType {
	case "anime":
		return "episode"
	case "manga":
		return "chapter"
	case "light_novel":
		return "volume"
	}
	return ""
}

// "3 new episodes of Frieren", "A new chapter of Berserk"
func newPartsTitle(title, part string, added int) string {
	if added <= 1 {
		return fmt.Sprintf("A new %s of %s", part, title)
	}
	return fmt.Sprintf("%d new %ss of %s", added, part, title)
}

func readyBody(mediaType string) string {
	switch mediaType {
	case "manga", "light_novel":
		return "Ready to read"
	case "music_track":
		return "Ready to listen to"
	}
	return "Ready to watch"
}

// the ids a query returned, NULLs as ""
func scanIDs(rows *sql.Rows, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return ids, err
		}
		ids = append(ids, id.String)
	}
	return ids, rows.Err()
}
