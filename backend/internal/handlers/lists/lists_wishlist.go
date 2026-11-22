package lists

import (
	"net/http"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// WishlistEntry is a title the user wants downloaded, with the ids to match it to cards by
type WishlistEntry struct {
	MediaItemID    string    `json:"media_item_id"`
	MediaType      string    `json:"media_type"`
	ExternalID     *string   `json:"external_id"`
	ExternalSource *string   `json:"external_source"`
	CreatedAt      time.Time `json:"created_at"`
}

/*
Function:	GetWishlist
Purpose:	the titles on the user's wishlist, newest first
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetWishlist(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	rows, err := h.db.Query(
		`SELECT mi.id, mi.type, mi.external_id, mi.external_source, uw.created_at
		FROM user_wishlist uw
		JOIN media_items mi ON mi.id = uw.media_item_id
		WHERE uw.user_id = $1
		ORDER BY uw.created_at DESC`,
		user.UserID,
	)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	items := []WishlistEntry{}
	for rows.Next() {
		var item WishlistEntry
		if utils.InternalError(w, rows.Scan(&item.MediaItemID, &item.MediaType, &item.ExternalID, &item.ExternalSource, &item.CreatedAt)) {
			return
		}
		items = append(items, item)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}
	utils.JSON(w, items)
}

/*
Function:	AddToWishlist
Purpose:	put a title on the user's wishlist and ask for it to be downloaded. Their list status,
if any, is left as it is
Params:
  - w: http response writer to respond to the front end
  - r: http request with the media item id in the URL
*/
func (h *Handler) AddToWishlist(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	res, err := h.db.Exec(
		`INSERT INTO user_wishlist (user_id, media_item_id)
		SELECT $1, id FROM media_items WHERE id = $2
		ON CONFLICT DO NOTHING`,
		user.UserID, id,
	)
	if utils.InternalError(w, err) {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// already wishlisted, or no such title
		var exists bool
		if utils.InternalError(w, h.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM media_items WHERE id = $1)`, id).Scan(&exists)) {
			return
		}
		if !exists {
			utils.Error(w, http.StatusNotFound, "not found")
			return
		}
	}
	h.requestWishlisted(user.UserID, id)
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	RemoveFromWishlist
Purpose:	take a title off the user's wishlist. A download already asked for carries on
Params:
  - w: http response writer to respond to the front end
  - r: http request with the media item id in the URL
*/
func (h *Handler) RemoveFromWishlist(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	if _, err := h.db.Exec(`DELETE FROM user_wishlist WHERE user_id = $1 AND media_item_id = $2`, user.UserID, id); utils.InternalError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
