package downloader

import (
	"database/sql"
	"errors"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

/*
Function:	WishlistDownloads
Purpose:	whether putting a title of this type on a wishlist downloads it. Light novels have no
downloader and music is requested from Discover, so for them the wishlist is only a list
Params:
  - mediaType: the title's type
*/
func WishlistDownloads(mediaType string) bool {
	switch mediaType {
	case "anime", "movie", "manga":
		return true
	}
	return false
}

/*
Function:	RequestWishlisted
Purpose:	a user put a title on their wishlist: ask for it to be downloaded, the same way a
download request does (approved and started at once for users allowed to, waiting for an admin
otherwise). Nothing happens for a title that is already downloaded or requested, or one of a
type the wishlist doesn't download. Adding a title to any other list never downloads it
Params:
  - db: the database
  - userID: who wishlisted it
  - mediaItemID: the title
*/
func RequestWishlisted(db *sql.DB, userID, mediaItemID string) error {
	var mediaType string
	var externalID *string
	var have bool
	err := db.QueryRow(
		`SELECT mi.type, mi.external_id,
			EXISTS (SELECT 1 FROM sonarr_items WHERE media_item_id = mi.id)
			OR EXISTS (SELECT 1 FROM radarr_items WHERE media_item_id = mi.id)
			OR EXISTS (SELECT 1 FROM suwayomi_items WHERE media_item_id = mi.id)
			OR EXISTS (SELECT 1 FROM manga_chapters WHERE media_item_id = mi.id AND file_path IS NOT NULL)
			OR EXISTS (SELECT 1 FROM download_requests WHERE media_item_id = mi.id AND status <> 'rejected')
		FROM media_items mi WHERE mi.id = $1`,
		mediaItemID,
	).Scan(&mediaType, &externalID, &have)
	if err != nil {
		return err
	}
	if have || !WishlistDownloads(mediaType) {
		return nil
	}

	var permission string
	if err := db.QueryRow(`SELECT download_permission FROM users WHERE id = $1`, userID).Scan(&permission); err != nil {
		return err
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
		return nil // this user asked before and it was turned down
	}
	if err != nil {
		return err
	}

	if autoApproved {
		if _, err := Dispatch(db, requestID, mediaItemID, "", mediaType, externalID); err != nil {
			logger.Warn("wishlist download of %s failed to start: %s", mediaItemID, err.Error())
		}
	}
	return nil
}
