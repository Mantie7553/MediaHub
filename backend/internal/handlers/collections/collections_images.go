package collections

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Mantie7553/MediaHub/backend/internal/covers"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

/*
Admins can give a playlist or collection an image of its own, shown instead of the mosaic of what's
in it. Each upload gets a new image id, so a copy made earlier (a featured one, or one someone
saved) keeps the image it was made with, and an image's file is only removed once no collection
uses it
*/

/*
Function:	canSetImage
Purpose:	Whether an admin may change a collection's image: one of their own they can still
change, or a featured one, which admins look after
*/
func canSetImage(c stored, user *auth.UserContext) bool {
	return user.Role == "admin" && (c.canEdit(user.UserID) || c.featured)
}

// finds the collection an admin is changing the image of, answering when they can't. Returns
// whether it answered
func loadForImage(w http.ResponseWriter, h *Handler, id string, user *auth.UserContext) (stored, bool) {
	if isSmart(id) {
		utils.Error(w, http.StatusBadRequest, "this playlist fills itself from your list and can't be changed")
		return stored{}, true
	}
	c, err := load(h.db, id, user, false)
	if errors.Is(err, errNotFound) {
		utils.Error(w, http.StatusNotFound, "not found")
		return c, true
	}
	if utils.InternalError(w, err) {
		return c, true
	}
	if !canSetImage(c, user) {
		utils.Error(w, http.StatusForbidden, "only admins can change the image of their own or a featured playlist or collection")
		return c, true
	}
	return c, false
}

/*
Function:	SetImage
Purpose:	Admin: upload an image for a playlist or collection, replacing any it had
Params:
  - r: multipart form with "file", a JPEG, PNG, WebP or GIF image

Returns:	{ cover_image_url } signed for <img>
*/
func (h *Handler) SetImage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	if _, answered := loadForImage(w, h, id, user); answered {
		return
	}
	data, ok := covers.ReadUpload(w, r)
	if !ok {
		return
	}

	var imageID string
	if utils.InternalError(w, h.db.QueryRow(`SELECT gen_random_uuid()::text`).Scan(&imageID)) {
		return
	}
	path, ok := covers.Path(imageID)
	if !ok {
		utils.InternalError(w, errors.New("MEDIA_ROOT isn't set, so there's nowhere to keep the image"))
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); utils.InternalError(w, err) {
		return
	}
	if err := musicfiles.WriteCover(path, data); err != nil {
		logger.Error("saving the image for collection %s: %s", id, err.Error())
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var old *string
	err := h.db.QueryRow(`
		UPDATE collections n SET image_id = $2, updated_at = NOW()
		FROM collections o WHERE n.id = $1 AND o.id = n.id
		RETURNING o.image_id::text`, id, imageID).Scan(&old)
	if err != nil {
		covers.Remove(imageID)
		utils.InternalError(w, err)
		return
	}
	if old != nil {
		removeUnusedImages(h.db, *old)
	}

	coverURL := covers.URL(imageID)
	if utils.InternalError(w, auth.SignCoverURL(&coverURL, user.UserID)) {
		return
	}
	utils.JSON(w, map[string]string{"cover_image_url": coverURL})
}

/*
Function:	RemoveImage
Purpose:	Admin: take a playlist's or collection's image away, so it shows the mosaic again
*/
func (h *Handler) RemoveImage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	if _, answered := loadForImage(w, h, id, user); answered {
		return
	}

	var old *string
	err := h.db.QueryRow(`
		UPDATE collections n SET image_id = NULL, updated_at = NOW()
		FROM collections o WHERE n.id = $1 AND o.id = n.id
		RETURNING o.image_id::text`, id).Scan(&old)
	if utils.InternalError(w, err) {
		return
	}
	if old != nil {
		removeUnusedImages(h.db, *old)
	}
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	imagesOf
Purpose:	The images used by a collection and the playlists copied along with it (part_of), and,
when it's an original, by its featured copy and that copy's playlists. Read before they are
deleted or replaced, so the files can go afterwards if nothing else uses them
*/
func imagesOf(db querier, collectionID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT image_id::text FROM collections
		WHERE image_id IS NOT NULL AND (id = $1 OR part_of = $1
			OR id IN (SELECT f.id FROM collections f WHERE f.source_collection_id = $1 AND f.featured_at IS NOT NULL)
			OR part_of IN (SELECT f.id FROM collections f WHERE f.source_collection_id = $1 AND f.featured_at IS NOT NULL))`,
		collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

/*
Function:	removeUnusedImages
Purpose:	Delete the files of images no collection uses any more. Called after the change that
let go of them is saved; a file that can't be removed is only logged, since it no longer shows
anywhere
*/
func removeUnusedImages(db querier, imageIDs ...string) {
	for _, imageID := range imageIDs {
		var used bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM collections WHERE image_id = $1)`, imageID).Scan(&used); err != nil {
			logger.Error("checking whether collection image %s is still used: %s", imageID, err.Error())
			continue
		}
		if used {
			continue
		}
		if err := covers.Remove(imageID); err != nil {
			logger.Error("removing collection image %s: %s", imageID, err.Error())
		}
	}
}
