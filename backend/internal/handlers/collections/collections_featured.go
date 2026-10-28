package collections

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

/*
Featuring is how playlists and collections reach everyone. A user submits one of theirs; an admin
looks it over and approves it, which freezes a copy of it as it is then. The copy belongs to
nobody, so it can't be changed afterwards, and it shows in everyone's Featured row. Anyone can
save a featured one to their own, as a private copy they can change. Admins can feature their
own without the review step. A later submission of the same original updates its featured copy
in place
*/

// the most a submission note or a reason for turning one down can hold
const maxNoteLength = 500

// the featured copies, newest first, shaped like the user's own summaries
var featuredQuery = `
SELECT c.id, c.kind::text, c.name, c.description, c.updated_at,
	(SELECT COUNT(*) FROM collection_items x WHERE x.collection_id = c.id),
	` + coversOrImage("c", "c") + `
FROM collections c
WHERE c.featured_at IS NOT NULL
ORDER BY c.featured_at DESC`

// waiting submissions, oldest first, with what they are and whether they update a featured copy
var pendingSubmissionsQuery = `
SELECT s.id, s.note, s.submitted_at, u.username,
	c.id, c.kind::text, c.name, c.description,
	(SELECT COUNT(*) FROM collection_items x WHERE x.collection_id = c.id),
	` + coversOrImage("c", "c") + `,
	f.id, f.name
FROM collection_submissions s
JOIN collections c ON c.id = s.collection_id
JOIN users u ON u.id = s.submitted_by
LEFT JOIN collections f ON f.source_collection_id = c.id AND f.featured_at IS NOT NULL
WHERE s.status = 'pending'
ORDER BY s.submitted_at`

// a note as it's stored: trimmed, and nil when empty
func cleanNote(n *string) (*string, string) {
	note, problem := cleanDescription(n)
	if problem != "" || note == nil {
		return note, ""
	}
	if len([]rune(*note)) > maxNoteLength {
		return nil, "notes can be up to 500 characters"
	}
	return note, ""
}

/*
Function:	latestSubmission
Purpose:	The most recent request to feature a playlist or collection, or nil when there's none
*/
func latestSubmission(db querier, collectionID string) (*SubmissionState, error) {
	var s SubmissionState
	err := db.QueryRow(`
		SELECT id, status::text, admin_note, submitted_at, resolved_at FROM collection_submissions
		WHERE collection_id = $1 ORDER BY submitted_at DESC LIMIT 1`, collectionID,
	).Scan(&s.ID, &s.Status, &s.AdminNote, &s.SubmittedAt, &s.ResolvedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// loads a collection the user owns and may change, answering when they can't. Returns whether
// it answered
func loadOwn(w http.ResponseWriter, db querier, id string, user *auth.UserContext, lock bool) (stored, bool) {
	var c stored
	var err error
	if !isSmart(id) {
		c, err = load(db, id, user, lock)
	}
	return c, refuseEdit(w, id, c, err, user.UserID)
}

/*
Function:	Submit
Purpose:	Ask the admins to feature one of the user's playlists or collections. Only one request
for it can wait at a time
Body:		{ note? } a message for the admins
*/
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	var req submitRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	note, problem := cleanNote(req.Note)
	if problem != "" {
		utils.Error(w, http.StatusBadRequest, problem)
		return
	}
	if _, answered := loadOwn(w, h.db, id, user, false); answered {
		return
	}

	var count int
	if utils.InternalError(w, h.db.QueryRow(`SELECT COUNT(*) FROM collection_items WHERE collection_id = $1`, id).Scan(&count)) {
		return
	}
	if count == 0 {
		utils.Error(w, http.StatusBadRequest, "add something to it first")
		return
	}

	_, err := h.db.Exec(
		`INSERT INTO collection_submissions (collection_id, submitted_by, note) VALUES ($1, $2, $3)`,
		id, user.UserID, note,
	)
	if utils.IsUniqueViolation(err) {
		utils.Error(w, http.StatusConflict, "it's already waiting for review")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	state, err := latestSubmission(h.db, id)
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, state, http.StatusCreated)
}

/*
Function:	Withdraw
Purpose:	Take back a request to feature one of the user's playlists or collections that is still
waiting
*/
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	if _, answered := loadOwn(w, h.db, id, user, false); answered {
		return
	}
	res, err := h.db.Exec(`DELETE FROM collection_submissions WHERE collection_id = $1 AND status = 'pending'`, id)
	if utils.InternalError(w, err) {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		utils.Error(w, http.StatusNotFound, "nothing is waiting for review")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	ListFeatured
Purpose:	The featured playlists and collections, newest first, for everyone's Featured row
*/
func (h *Handler) ListFeatured(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	rows, err := h.db.Query(featuredQuery)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	items := []Summary{}
	for rows.Next() {
		var s Summary
		var updated time.Time
		if utils.InternalError(w, rows.Scan(&s.ID, &s.Kind, &s.Name, &s.Description, &updated,
			&s.ItemCount, pq.Array(&s.Covers))) {
			return
		}
		s.UpdatedAt = &updated
		if s.Covers == nil {
			s.Covers = []string{}
		}
		if utils.InternalError(w, signCovers(user.UserID, s.Covers)) {
			return
		}
		items = append(items, s)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}
	utils.JSON(w, items)
}

/*
Function:	SaveCopy
Purpose:	Save a featured playlist or collection to the user's own, as a private copy they can
change. Playlists nested in a collection become their own playlists too
Returns:	{ id, kind } of the copy
*/
func (h *Handler) SaveCopy(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	c, err := load(h.db, id, user, false)
	if errors.Is(err, errNotFound) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	if !c.featured {
		utils.Error(w, http.StatusBadRequest, "only featured playlists and collections can be saved as a copy")
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()
	copyID, err := newCopy(tx, id, copyTarget{ownerID: user.UserID})
	if utils.InternalError(w, err) {
		return
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}
	utils.JSON(w, map[string]string{"id": copyID, "kind": c.kind}, http.StatusCreated)
}

/*
Function:	ListSubmissions
Purpose:	Admins: the requests waiting for review, oldest first
*/
func (h *Handler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	rows, err := h.db.Query(pendingSubmissionsQuery)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	items := []PendingSubmission{}
	for rows.Next() {
		var p PendingSubmission
		if utils.InternalError(w, rows.Scan(&p.ID, &p.Note, &p.SubmittedAt, &p.SubmittedBy,
			&p.Collection.ID, &p.Collection.Kind, &p.Collection.Name, &p.Collection.Description,
			&p.Collection.ItemCount, pq.Array(&p.Collection.Covers), &p.UpdatesID, &p.UpdatesName)) {
			return
		}
		if p.Collection.Covers == nil {
			p.Collection.Covers = []string{}
		}
		if utils.InternalError(w, signCovers(user.UserID, p.Collection.Covers)) {
			return
		}
		items = append(items, p)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}
	utils.JSON(w, items)
}

/*
Function:	Approve
Purpose:	Admins: feature a submitted playlist or collection as it is now. One already featured
from the same original is updated in place
Returns:	{ featured_id }
*/
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	submissionID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	var collectionID string
	var ownerID *string
	err = tx.QueryRow(`
		SELECT s.collection_id, c.owner_id FROM collection_submissions s
		JOIN collections c ON c.id = s.collection_id
		WHERE s.id = $1 AND s.status = 'pending'
		FOR UPDATE OF s, c`, submissionID).Scan(&collectionID, &ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "that isn't waiting for review any more")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	// the featured copy's images, which it may let go of when it's updated
	images, err := imagesOf(tx, collectionID)
	if utils.InternalError(w, err) {
		return
	}
	featuredID, err := feature(tx, collectionID, copyTarget{featured: true, createdBy: ownerID, featuredBy: user.UserID})
	if utils.InternalError(w, err) {
		return
	}
	if _, err := tx.Exec(`
		UPDATE collection_submissions SET status = 'approved', resolved_at = NOW(), resolved_by = $2
		WHERE id = $1`, submissionID, user.UserID); utils.InternalError(w, err) {
		return
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}
	removeUnusedImages(h.db, images...)
	utils.JSON(w, map[string]string{"featured_id": featuredID})
}

/*
Function:	Reject
Purpose:	Admins: turn a submission down, optionally saying why. The user sees the reason on
their playlist or collection and can submit again
Body:		{ note? }
*/
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	submissionID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req submitRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	note, problem := cleanNote(req.Note)
	if problem != "" {
		utils.Error(w, http.StatusBadRequest, problem)
		return
	}

	res, err := h.db.Exec(`
		UPDATE collection_submissions SET status = 'rejected', admin_note = $2, resolved_at = NOW(), resolved_by = $3
		WHERE id = $1 AND status = 'pending'`, submissionID, note, user.UserID)
	if utils.InternalError(w, err) {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		utils.Error(w, http.StatusNotFound, "that isn't waiting for review any more")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	Feature
Purpose:	Admins: feature one of their own playlists or collections straight away, as it is now.
A waiting submission of it is approved along the way
Returns:	{ featured_id }
*/
func (h *Handler) Feature(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()
	if _, answered := loadOwn(w, tx, id, user, true); answered {
		return
	}

	var count int
	if utils.InternalError(w, tx.QueryRow(`SELECT COUNT(*) FROM collection_items WHERE collection_id = $1`, id).Scan(&count)) {
		return
	}
	if count == 0 {
		utils.Error(w, http.StatusBadRequest, "add something to it first")
		return
	}

	images, err := imagesOf(tx, id)
	if utils.InternalError(w, err) {
		return
	}
	featuredID, err := feature(tx, id, copyTarget{featured: true, createdBy: &user.UserID, featuredBy: user.UserID})
	if utils.InternalError(w, err) {
		return
	}
	if _, err := tx.Exec(`
		UPDATE collection_submissions SET status = 'approved', resolved_at = NOW(), resolved_by = $2
		WHERE collection_id = $1 AND status = 'pending'`, id, user.UserID); utils.InternalError(w, err) {
		return
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}
	removeUnusedImages(h.db, images...)
	utils.JSON(w, map[string]string{"featured_id": featuredID})
}
