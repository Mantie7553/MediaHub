package downloader

import (
	"database/sql"
	"errors"

	"github.com/Mantie7553/MediaHub/backend/internal/notify"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

/*
Function:	WishlistDownloads
Purpose:	whether putting a title of this type on a wishlist asks for it to be downloaded.
Music is requested from Discover, so for it the wishlist is only a list
Params:
  - mediaType: the title's type
*/
func WishlistDownloads(mediaType string) bool {
	switch mediaType {
	case "anime", "movie", "manga", "light_novel":
		return true
	}
	return false
}

/*
Function:	ByHand
Purpose:	whether an admin downloads titles of this type themselves. Light novels come from a
site behind Cloudflare's browser check, so the server can't fetch them: their requests are
approved like any other, start no job, and are complete once volumes are added
Params:
  - mediaType: the title's type
*/
func ByHand(mediaType string) bool {
	return mediaType == "light_novel"
}

/*
Function:	CompleteByHand
Purpose:	mark the approved requests for a title an admin downloads by hand complete, once it has
volumes, and tell their requesters it's ready. For the library sync, which can't tell new
volumes apart: an upload tells everyone following the title with notify.TitleArrived instead
Params:
  - db: the database
  - mediaItemID: the title
*/
func CompleteByHand(db *sql.DB, mediaItemID string) {
	var hasVolumes bool
	if err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM light_novel_volumes WHERE media_item_id = $1)`, mediaItemID,
	).Scan(&hasVolumes); err != nil {
		logger.Warn("completing requests for %s: %s", mediaItemID, err.Error())
		return
	}
	if hasVolumes {
		notify.RequestsFulfilled(db, mediaItemID)
	}
}

// what wishlisting a title did about downloading it, sent back so the user is told
const (
	// a request was made and its download started
	WishlistRequested = "requested"
	// a request was made and waits for an admin
	WishlistPending = "pending"
	// the library already has it, or an admin linked it to Sonarr, Radarr or Suwayomi
	WishlistHave = "have"
	// someone asked for it before and that request is still going or done
	WishlistAlreadyRequested = "already_requested"
	// it was asked for before and its last download failed; a new one would fail the same way
	WishlistFailed = "failed"
	// this user asked for it before and an admin turned it down
	WishlistRejected = "rejected"
	// the wishlist doesn't download titles of this type
	WishlistNotDownloadable = "not_downloadable"
	// a request was made and approved, and an admin downloads it by hand (ByHand)
	WishlistByHand = "by_hand"
)

// WishlistOutcome is what wishlisting a title did about downloading it
type WishlistOutcome struct {
	Download string `json:"download"`
	// for a failed one, why its last download failed
	Reason string `json:"reason,omitempty"`
}

/*
wishlistFacts: what the library already has of a title, from RequestWishlisted's lookup
  - have: linked in Sonarr, Radarr or Suwayomi, or has chapters on disk
  - requested: someone has a request for it that wasn't rejected (archived ones count)
  - lastJobStatus, lastJobError: its newest download job, empty when it has none
*/
type wishlistFacts struct {
	mediaType     string
	have          bool
	requested     bool
	lastJobStatus string
	lastJobError  string
}

/*
Function:	existingOutcome
Purpose:	why a title needs no new request, or ok=false when it should be requested
*/
func existingOutcome(f wishlistFacts) (WishlistOutcome, bool) {
	switch {
	case !WishlistDownloads(f.mediaType):
		return WishlistOutcome{Download: WishlistNotDownloadable}, true
	case f.have:
		return WishlistOutcome{Download: WishlistHave}, true
	case f.requested && f.lastJobStatus == "failed":
		return WishlistOutcome{Download: WishlistFailed, Reason: f.lastJobError}, true
	case f.requested:
		return WishlistOutcome{Download: WishlistAlreadyRequested}, true
	}
	return WishlistOutcome{}, false
}

/*
Function:	RequestWishlisted
Purpose:	a user put a title on their wishlist: ask for it to be downloaded, the same way a
download request does (approved and started at once for users allowed to, waiting for an admin
otherwise), and say what happened. Nothing is requested for a title that is already downloaded
or requested, or one of a type the wishlist doesn't download. The download itself starts in
the background, since Sonarr / Radarr can take a few seconds. The wishlist is kept apart from
list statuses, and a title's list status never downloads it
Params:
  - db: the database
  - userID: who wishlisted it
  - mediaItemID: the title
*/
func RequestWishlisted(db *sql.DB, userID, mediaItemID string) (WishlistOutcome, error) {
	var f wishlistFacts
	var externalID *string
	var jobStatus, jobError sql.NullString
	err := db.QueryRow(
		`SELECT mi.type, mi.external_id,
			EXISTS (SELECT 1 FROM sonarr_items WHERE media_item_id = mi.id)
			OR EXISTS (SELECT 1 FROM radarr_items WHERE media_item_id = mi.id)
			OR EXISTS (SELECT 1 FROM suwayomi_items WHERE media_item_id = mi.id)
			OR EXISTS (SELECT 1 FROM manga_chapters WHERE media_item_id = mi.id AND file_path IS NOT NULL)
			OR EXISTS (SELECT 1 FROM light_novel_volumes WHERE media_item_id = mi.id),
			EXISTS (SELECT 1 FROM download_requests WHERE media_item_id = mi.id AND status <> 'rejected'),
			j.status::text, j.error_message
		FROM media_items mi
		LEFT JOIN LATERAL (
			SELECT status, error_message FROM download_jobs
			WHERE media_item_id = mi.id ORDER BY created_at DESC LIMIT 1
		) j ON TRUE
		WHERE mi.id = $1`,
		mediaItemID,
	).Scan(&f.mediaType, &externalID, &f.have, &f.requested, &jobStatus, &jobError)
	if err != nil {
		return WishlistOutcome{}, err
	}
	f.lastJobStatus, f.lastJobError = jobStatus.String, jobError.String
	if outcome, done := existingOutcome(f); done {
		logger.Info("wishlisted %s not requested: %s", mediaItemID, outcome.Download)
		return outcome, nil
	}

	var permission string
	if err := db.QueryRow(`SELECT download_permission FROM users WHERE id = $1`, userID).Scan(&permission); err != nil {
		return WishlistOutcome{}, err
	}
	status := "pending"
	autoApproved := permission == "auto_approved"
	if autoApproved {
		status = "approved"
	}

	var requestID string
	err = db.QueryRow(
		`INSERT INTO download_requests (requested_by, media_item_id, status, auto_approved, source_url)
		VALUES ($1, $2, $3, $4, '')
		ON CONFLICT (requested_by, media_item_id) DO NOTHING
		RETURNING id`,
		userID, mediaItemID, status, autoApproved,
	).Scan(&requestID)
	if errors.Is(err, sql.ErrNoRows) {
		// this user asked before and it was turned down
		logger.Info("wishlisted %s not requested: %s", mediaItemID, WishlistRejected)
		return WishlistOutcome{Download: WishlistRejected}, nil
	}
	if err != nil {
		return WishlistOutcome{}, err
	}

	if !autoApproved {
		notify.RequestPending(db, requestID)
		return WishlistOutcome{Download: WishlistPending}, nil
	}
	if ByHand(f.mediaType) {
		notify.ByHand(db, requestID)
		return WishlistOutcome{Download: WishlistByHand}, nil
	}
	go func() {
		if _, err := Dispatch(db, requestID, mediaItemID, "", f.mediaType, externalID); err != nil {
			logger.Warn("wishlist download of %s failed to start: %s", mediaItemID, err.Error())
		}
	}()
	return WishlistOutcome{Download: WishlistRequested}, nil
}
