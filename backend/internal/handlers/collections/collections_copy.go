package collections

import (
	"database/sql"
)

/*
copyTarget is what a copy becomes: a featured copy that nobody owns, or a private one for a user
who saved a featured playlist or collection to their own
*/
type copyTarget struct {
	featured bool
	// the owner of a private copy
	ownerID string
	// featured copies only: who made the original, and the admin who featured it
	createdBy  *string
	featuredBy string
}

/*
Function:	newCopy
Purpose:	Copy a stored playlist or collection with everything in it. Playlists nested in a
collection are copied too, so the copy never points at something its source's owner can still
change or that the people seeing it can't open
Params:
  - tx: the transaction the copy is made in
  - srcID: what to copy
  - to: what the copy becomes

Returns:	the copy's id
*/
func newCopy(tx querier, srcID string, to copyTarget) (string, error) {
	var id string
	var err error
	if to.featured {
		err = tx.QueryRow(`
			INSERT INTO collections (owner_id, kind, name, description, visibility, source_collection_id, created_by, featured_by, featured_at)
			SELECT NULL, kind, name, description, 'featured', id, $2, $3, NOW() FROM collections WHERE id = $1
			RETURNING id`, srcID, to.createdBy, to.featuredBy).Scan(&id)
	} else {
		err = tx.QueryRow(`
			INSERT INTO collections (owner_id, kind, name, description)
			SELECT $2, kind, name, description FROM collections WHERE id = $1
			RETURNING id`, srcID, to.ownerID).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	return id, copyEntries(tx, srcID, id, to)
}

/*
Function:	replaceFeatured
Purpose:	Swap the contents of a featured copy for its source's as they are now, keeping the copy's
id and its place in the Featured row
*/
func replaceFeatured(tx querier, featuredID, srcID string, to copyTarget) error {
	// the old nested copies go with their entries
	if _, err := tx.Exec(`DELETE FROM collections WHERE part_of = $1`, featuredID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM collection_items WHERE collection_id = $1`, featuredID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE collections f SET name = s.name, description = s.description, created_by = $3,
			featured_by = $4, updated_at = NOW()
		FROM collections s WHERE f.id = $1 AND s.id = $2`,
		featuredID, srcID, to.createdBy, to.featuredBy); err != nil {
		return err
	}
	return copyEntries(tx, srcID, featuredID, to)
}

/*
Function:	copyEntries
Purpose:	Copy srcID's entries into dstID in the same order. Tracks, titles and albums are the
library's own, so the copy points at the same ones; nested playlists are copied first
*/
func copyEntries(tx querier, srcID, dstID string, to copyTarget) error {
	if _, err := tx.Exec(`
		INSERT INTO collection_items (collection_id, media_item_id, album_id, position, added_by, added_at)
		SELECT $2, media_item_id, album_id, position, added_by, added_at
		FROM collection_items WHERE collection_id = $1 AND child_collection_id IS NULL`,
		srcID, dstID); err != nil {
		return err
	}

	type nested struct {
		childID  string
		position int64
	}
	rows, err := tx.Query(`
		SELECT child_collection_id, position FROM collection_items
		WHERE collection_id = $1 AND child_collection_id IS NOT NULL
		ORDER BY position`, srcID)
	if err != nil {
		return err
	}
	var children []nested
	for rows.Next() {
		var n nested
		if err := rows.Scan(&n.childID, &n.position); err != nil {
			rows.Close()
			return err
		}
		children = append(children, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, n := range children {
		childCopy, err := copyChild(tx, n.childID, dstID, to)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`
			INSERT INTO collection_items (collection_id, child_collection_id, position)
			VALUES ($1, $2, $3)`, dstID, childCopy, n.position); err != nil {
			return err
		}
	}
	return nil
}

/*
Function:	copyChild
Purpose:	Copy a playlist nested in a collection being copied. In a featured copy it belongs to
that collection (part_of) and doesn't show in the Featured row; in a private copy it becomes one
of the user's own playlists, as the original was its owner's
*/
func copyChild(tx querier, childID, parentCopyID string, to copyTarget) (string, error) {
	var id string
	var err error
	if to.featured {
		err = tx.QueryRow(`
			INSERT INTO collections (owner_id, kind, name, description, visibility, created_by, part_of)
			SELECT NULL, kind, name, description, 'featured', $2, $3 FROM collections WHERE id = $1
			RETURNING id`, childID, to.createdBy, parentCopyID).Scan(&id)
	} else {
		err = tx.QueryRow(`
			INSERT INTO collections (owner_id, kind, name, description)
			SELECT $2, kind, name, description FROM collections WHERE id = $1
			RETURNING id`, childID, to.ownerID).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	// playlists hold only tracks, so this goes no deeper
	return id, copyEntries(tx, childID, id, to)
}

/*
Function:	feature
Purpose:	Feature a stored playlist or collection as it is now: a new featured copy, or the
existing featured copy of the same original updated in place
Returns:	the featured copy's id
*/
func feature(tx querier, srcID string, to copyTarget) (string, error) {
	existing, err := featuredCopyOf(tx, srcID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		if _, err := tx.Exec(`SELECT 1 FROM collections WHERE id = $1 FOR UPDATE`, *existing); err != nil {
			return "", err
		}
		return *existing, replaceFeatured(tx, *existing, srcID, to)
	}
	return newCopy(tx, srcID, to)
}

/*
Function:	featuredCopyOf
Purpose:	The id of the featured copy made from a playlist or collection, or nil when there isn't
one
*/
func featuredCopyOf(db querier, srcID string) (*string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM collections WHERE source_collection_id = $1 AND featured_at IS NOT NULL`, srcID).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
