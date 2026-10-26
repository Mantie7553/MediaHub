/*
Package covers holds the cover images admins upload for a title, kept in MEDIA_ROOT/Covers
under the media item's id, and the images they upload for a playlist or collection, under an
image id of their own. Album covers aren't kept here: they are the cover.jpg in the
album's folder (see musicfiles)
*/
package covers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

/*
Function:	Path
Purpose:	where an uploaded cover for a media item is kept. The image type is read from the
file when it is served, so the name has no extension
Params:
  - mediaItemID: the media item's id, a UUID
Returns:	the path, and false when the id isn't a UUID or MEDIA_ROOT isn't set
*/
func Path(mediaItemID string) (string, bool) {
	root := os.Getenv("MEDIA_ROOT")
	if root == "" || !utils.IsUUID(mediaItemID) {
		return "", false
	}
	return filepath.Join(root, "Covers", mediaItemID), true
}

/*
Function:	URL
Purpose:	the stored cover_image_url for an uploaded cover, signed with a media token when it
is sent out (auth.SignCoverURL)
Params:
  - mediaItemID: the media item's id
*/
func URL(mediaItemID string) string {
	return fmt.Sprintf("/api/media/%s/cover", mediaItemID)
}

/*
Function:	Remove
Purpose:	delete a media item's uploaded cover, if it has one
Params:
  - mediaItemID: the media item's id
*/
func Remove(mediaItemID string) error {
	path, ok := Path(mediaItemID)
	if !ok {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
