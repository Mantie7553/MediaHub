package collections

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

// an album's downloaded tracks, added on the end of a playlist in track order: $4 is the
// playlist's last position and $5 the step between entries
const appendTracksOfAlbum = `
INSERT INTO collection_items (collection_id, media_item_id, position, added_by)
SELECT $1, mm.media_item_id, $4 + (row_number() OVER (ORDER BY mm.track_number NULLS LAST, mi.title)) * $5, $3
FROM music_metadata mm
JOIN media_items mi ON mi.id = mm.media_item_id
WHERE mm.album_id = $2 AND mm.file_path IS NOT NULL
ON CONFLICT (collection_id, media_item_id) WHERE media_item_id IS NOT NULL AND season_number IS NULL DO NOTHING`

/*
Function:	itemKind
Purpose:	What the item in an add request is, for the rules, checking it exists and (for a
nested playlist) that the user may see it
Returns:	the kind, the column the entry stores it in, its id, or errNotFound
*/
func itemKind(db querier, req addRequest, user *auth.UserContext) (kind, column, id string, err error) {
	given := 0
	for _, v := range []string{req.MediaItemID, req.AlbumID, req.CollectionID} {
		if v != "" {
			given++
		}
	}
	if given != 1 {
		return "", "", "", errWrongItem
	}

	switch {
	case req.MediaItemID != "":
		id, column = req.MediaItemID, "media_item_id"
		if !utils.IsUUID(id) {
			return "", "", "", errNotFound
		}
		err = db.QueryRow(`SELECT type::text FROM media_items WHERE id = $1`, id).Scan(&kind)
	case req.AlbumID != "":
		id, column, kind = req.AlbumID, "album_id", "album"
		if !utils.IsUUID(id) {
			return "", "", "", errNotFound
		}
		err = db.QueryRow(`SELECT id FROM albums WHERE id = $1`, id).Scan(&id)
	default:
		id, column = req.CollectionID, "child_collection_id"
		if isSmart(id) {
			// computed playlists fill from the user's statuses, they can't be stored in another
			return "", "", "", errSmartNested
		}
		var c stored
		c, err = load(db, id, user, false)
		kind = c.kind
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = errNotFound
	}
	return kind, column, id, err
}

/*
Function:	AddItem
Purpose:	Add a track, title, album or playlist to a playlist or collection, on the end. An album
added to a playlist adds its downloaded tracks. Something already there stays where it is
Returns:	how many entries were added and how many were already there
*/
func (h *Handler) AddItem(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	var req addRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	var c stored
	if !isSmart(id) {
		c, err = load(tx, id, user, true)
	}
	if refuseEdit(w, id, c, err, user.UserID) {
		return
	}

	kind, column, itemID, err := itemKind(tx, req, user)
	if errors.Is(err, errNotFound) {
		utils.Error(w, http.StatusNotFound, "that isn't in the library")
		return
	}
	if errors.Is(err, errSmartNested) {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, errWrongItem) {
		utils.Error(w, http.StatusBadRequest, "send one of media_item_id, album_id or collection_id")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	if itemID == id {
		utils.Error(w, http.StatusBadRequest, errSelf.Error())
		return
	}
	if err := canHold(c.kind, kind); err != nil {
		utils.Error(w, http.StatusBadRequest, holdMessage(c.kind, err))
		return
	}
	if req.SeasonNumber != nil && kind != "anime" {
		utils.Error(w, http.StatusBadRequest, errSeasonOfWhat.Error())
		return
	}
	if kind == "anime" {
		problem, status, err := checkSeason(tx, id, itemID, req.SeasonNumber)
		if utils.InternalError(w, err) {
			return
		}
		if problem != nil {
			utils.Error(w, status, problem.Error())
			return
		}
	}

	var last int64
	if utils.InternalError(w, tx.QueryRow(`SELECT COALESCE(MAX(position), 0) FROM collection_items WHERE collection_id = $1`, id).Scan(&last)) {
		return
	}

	var result addResponse
	if expandsToTracks(c.kind, kind) {
		var total int
		if utils.InternalError(w, tx.QueryRow(`SELECT COUNT(*) FROM music_metadata WHERE album_id = $1 AND file_path IS NOT NULL`, itemID).Scan(&total)) {
			return
		}
		res, err := tx.Exec(appendTracksOfAlbum, id, itemID, user.UserID, last, positionStep)
		if utils.InternalError(w, err) {
			return
		}
		added, _ := res.RowsAffected()
		result = addResponse{Added: int(added), Already: total - int(added)}
	} else {
		query := `INSERT INTO collection_items (collection_id, ` + column + `, position, added_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (collection_id, ` + column + `) WHERE ` + column + ` IS NOT NULL DO NOTHING`
		args := []any{id, itemID, last + positionStep, user.UserID}
		switch {
		case req.SeasonNumber != nil:
			query = `INSERT INTO collection_items (collection_id, media_item_id, season_number, position, added_by)
				VALUES ($1, $2, $5, $3, $4)
				ON CONFLICT (collection_id, media_item_id, season_number) WHERE season_number IS NOT NULL DO NOTHING`
			args = append(args, *req.SeasonNumber)
		case column == "media_item_id":
			// the whole title, which has its own index apart from its seasons
			query = `INSERT INTO collection_items (collection_id, media_item_id, position, added_by)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (collection_id, media_item_id) WHERE media_item_id IS NOT NULL AND season_number IS NULL DO NOTHING`
		}
		res, err := tx.Exec(query, args...)
		if utils.InternalError(w, err) {
			return
		}
		added, _ := res.RowsAffected()
		result = addResponse{Added: int(added), Already: 1 - int(added)}
	}

	if result.Added > 0 {
		if _, err := tx.Exec(`UPDATE collections SET updated_at = NOW() WHERE id = $1`, id); utils.InternalError(w, err) {
			return
		}
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}
	utils.JSON(w, result)
}

/*
Function:	checkSeason
Purpose:	Before an anime, or one season of it, goes into a collection: the season has to be in
the library, and the collection can't end up with both the whole anime and some of its seasons
Returns:	why not and the status to answer with, or a nil problem
*/
func checkSeason(db querier, collectionID, animeID string, season *int) (problem error, status int, err error) {
	if season != nil {
		var exists bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM episodes WHERE media_item_id = $1 AND season_number = $2)`,
			animeID, *season).Scan(&exists); err != nil {
			return nil, 0, err
		}
		if !exists {
			return errors.New("that season isn't in the library"), http.StatusNotFound, nil
		}
	}
	var hasWhole, hasSeasons bool
	if err := db.QueryRow(`
		SELECT COALESCE(bool_or(season_number IS NULL), false), COALESCE(bool_or(season_number IS NOT NULL), false)
		FROM collection_items WHERE collection_id = $1 AND media_item_id = $2`,
		collectionID, animeID).Scan(&hasWhole, &hasSeasons); err != nil {
		return nil, 0, err
	}
	if problem := seasonConflict(season, hasWhole, hasSeasons); problem != nil {
		return problem, http.StatusConflict, nil
	}
	return nil, 0, nil
}

// what to tell someone adding the wrong kind of thing
func holdMessage(kind string, err error) string {
	if errors.Is(err, errNestedCollection) {
		return err.Error()
	}
	if kind == kindPlaylist {
		return "playlists hold tracks and albums"
	}
	return "collections hold titles, albums, tracks and playlists"
}

/*
Function:	RemoveItem
Purpose:	Take one entry out of a playlist or collection
*/
func (h *Handler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	entryID := chi.URLParam(r, "entryId")

	var c stored
	var err error
	if !isSmart(id) {
		c, err = load(h.db, id, user, false)
	}
	if refuseEdit(w, id, c, err, user.UserID) {
		return
	}
	if !utils.IsUUID(entryID) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	res, err := h.db.Exec(`DELETE FROM collection_items WHERE id = $1 AND collection_id = $2`, entryID, id)
	if utils.InternalError(w, err) {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	if _, err := h.db.Exec(`UPDATE collections SET updated_at = NOW() WHERE id = $1`, id); utils.InternalError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// renumbers a collection's entries step apart in their current order, making room again
const renumber = `
UPDATE collection_items ci SET position = ordered.n * $2
FROM (SELECT id, row_number() OVER (ORDER BY position, added_at) AS n
	FROM collection_items WHERE collection_id = $1) ordered
WHERE ci.id = ordered.id`

/*
Function:	neighbours
Purpose:	The positions an entry moving below `after` (or to the top) will sit between, leaving the
moving entry itself out
*/
func neighbours(db querier, collectionID, entryID string, after *string) (prev, next *int64, err error) {
	if after != nil {
		var p int64
		err = db.QueryRow(`SELECT position FROM collection_items WHERE id = $1 AND collection_id = $2`, *after, collectionID).Scan(&p)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errNotFound
		}
		if err != nil {
			return nil, nil, err
		}
		prev = &p
	}
	var n sql.NullInt64
	query := `SELECT MIN(position) FROM collection_items WHERE collection_id = $1 AND id <> $2`
	args := []any{collectionID, entryID}
	if prev != nil {
		query += ` AND position > $3`
		args = append(args, *prev)
	}
	if err = db.QueryRow(query, args...).Scan(&n); err != nil {
		return nil, nil, err
	}
	if n.Valid {
		next = &n.Int64
	}
	return prev, next, nil
}

/*
Function:	MoveItem
Purpose:	Move an entry below another one, or to the top
*/
func (h *Handler) MoveItem(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	entryID := chi.URLParam(r, "entryId")
	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	var c stored
	if !isSmart(id) {
		c, err = load(tx, id, user, true)
	}
	if refuseEdit(w, id, c, err, user.UserID) {
		return
	}
	if !utils.IsUUID(entryID) || (req.After != nil && (!utils.IsUUID(*req.After) || *req.After == entryID)) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	var exists bool
	if utils.InternalError(w, tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM collection_items WHERE id = $1 AND collection_id = $2)`, entryID, id).Scan(&exists)) {
		return
	}
	if !exists {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	position, ok := int64(0), false
	for attempt := 0; attempt < 2 && !ok; attempt++ {
		prev, next, err := neighbours(tx, id, entryID, req.After)
		if errors.Is(err, errNotFound) {
			utils.Error(w, http.StatusNotFound, "not found")
			return
		}
		if utils.InternalError(w, err) {
			return
		}
		position, ok = positionBetween(prev, next)
		if !ok {
			if _, err := tx.Exec(renumber, id, positionStep); utils.InternalError(w, err) {
				return
			}
		}
	}
	if !ok {
		utils.InternalError(w, errors.New("no room to move the entry after renumbering"))
		return
	}

	if _, err := tx.Exec(`UPDATE collection_items SET position = $1 WHERE id = $2`, position, entryID); utils.InternalError(w, err) {
		return
	}
	if _, err := tx.Exec(`UPDATE collections SET updated_at = NOW() WHERE id = $1`, id); utils.InternalError(w, err) {
		return
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
