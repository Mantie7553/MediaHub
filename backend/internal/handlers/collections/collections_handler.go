package collections

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// the most a description can hold
const maxDescriptionLength = 1000

// ids are UUIDs; anything else can't be one of ours, and would make Postgres fail the query
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// what the handlers need to know about a stored collection before acting on it
type stored struct {
	id   string
	kind string
	access
}

var errNotFound = errors.New("not found")

/*
Function:	load
Purpose:	Find a stored collection the user may see, locking it in a transaction so changes to
its entries happen one at a time
Returns:	errNotFound when it doesn't exist or isn't theirs to see
*/
func load(db querier, id, userID string, lock bool) (stored, error) {
	c := stored{id: id}
	if !isUUID(id) {
		return c, errNotFound
	}
	query := `SELECT kind::text, owner_id FROM collections WHERE id = $1`
	if lock {
		query += ` FOR UPDATE`
	}
	err := db.QueryRow(query, id).Scan(&c.kind, &c.ownerID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !c.canView(userID)) {
		return c, errNotFound
	}
	return c, err
}

// signs every cover so the browser can load it, as the list endpoints do
func signCovers(userID string, covers []string) error {
	for i := range covers {
		if err := auth.SignCoverURL(&covers[i], userID); err != nil {
			return err
		}
	}
	return nil
}

func signEntries(userID string, entries []Entry) error {
	for i := range entries {
		if err := auth.SignCoverURL(entries[i].CoverImageURL, userID); err != nil {
			return err
		}
		if err := signCovers(userID, entries[i].Covers); err != nil {
			return err
		}
	}
	return nil
}

/*
Function:	cleanName
Purpose:	A name as it's stored, trimmed
Returns:	the name, or a message saying what's wrong with it
*/
func cleanName(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "give it a name"
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return "", "names can be up to 100 characters"
	}
	return name, ""
}

// a description as it's stored: trimmed, and nil when empty
func cleanDescription(d *string) (*string, string) {
	if d == nil {
		return nil, ""
	}
	s := strings.TrimSpace(*d)
	if utf8.RuneCountInString(s) > maxDescriptionLength {
		return nil, "descriptions can be up to 1000 characters"
	}
	if s == "" {
		return nil, ""
	}
	return &s, ""
}

/*
Function:	List
Purpose:	The user's playlists and collections. Playlists start with the two computed ones
Query:
  - kind: "playlist", "collection", or nothing for both
  - contains: an item id; the ones already holding it are marked, for the "Add to…" picker
*/
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	kind := r.URL.Query().Get("kind")
	contains := r.URL.Query().Get("contains")
	if kind != "" && kind != kindPlaylist && kind != kindCollection {
		utils.Error(w, http.StatusBadRequest, "kind is playlist or collection")
		return
	}

	items := []Summary{}
	if kind != kindCollection {
		for _, s := range smartNames {
			tracks, err := loadSmart(h.db, user.UserID, s.id)
			if utils.InternalError(w, err) {
				return
			}
			items = append(items, smartSummary(s.id, s.name, tracks))
		}
	}

	rows, err := h.db.Query(summariesQuery, user.UserID, kind, contains)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var s Summary
		var updated time.Time
		if utils.InternalError(w, rows.Scan(&s.ID, &s.Kind, &s.Name, &s.Description, &updated,
			&s.ItemCount, pq.Array(&s.Covers), &s.Contains)) {
			return
		}
		s.UpdatedAt = &updated
		items = append(items, s)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}

	for i := range items {
		if items[i].Covers == nil {
			items[i].Covers = []string{}
		}
		if utils.InternalError(w, signCovers(user.UserID, items[i].Covers)) {
			return
		}
	}
	utils.JSON(w, items)
}

/*
Function:	Create
Purpose:	Make a new, empty playlist or collection for the user
*/
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Kind != kindPlaylist && req.Kind != kindCollection {
		utils.Error(w, http.StatusBadRequest, "kind is playlist or collection")
		return
	}
	name, problem := cleanName(req.Name)
	if problem == "" {
		req.Description, problem = cleanDescription(req.Description)
	}
	if problem != "" {
		utils.Error(w, http.StatusBadRequest, problem)
		return
	}

	var id string
	err := h.db.QueryRow(
		`INSERT INTO collections (owner_id, kind, name, description) VALUES ($1, $2, $3, $4) RETURNING id`,
		user.UserID, req.Kind, name, req.Description,
	).Scan(&id)
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, map[string]string{"id": id}, http.StatusCreated)
}

/*
Function:	Get
Purpose:	A playlist or collection with everything in it, in order. The computed playlists
answer to their names ("listened", "unlistened")
*/
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")

	var detail Detail
	if isSmart(id) {
		tracks, err := loadSmart(h.db, user.UserID, id)
		if utils.InternalError(w, err) {
			return
		}
		for _, s := range smartNames {
			if s.id == id {
				detail.Summary = smartSummary(s.id, s.name, tracks)
			}
		}
		detail.Items = tracks
	} else {
		c, err := load(h.db, id, user.UserID, false)
		if errors.Is(err, errNotFound) {
			utils.Error(w, http.StatusNotFound, "not found")
			return
		}
		if utils.InternalError(w, err) {
			return
		}
		var updated time.Time
		err = h.db.QueryRow(
			`SELECT c.id, c.kind::text, c.name, c.description, c.updated_at, `+coversOf("c.id", "c")+`
			FROM collections c WHERE c.id = $1`, id,
		).Scan(&detail.ID, &detail.Kind, &detail.Name, &detail.Description, &updated, pq.Array(&detail.Covers))
		if utils.InternalError(w, err) {
			return
		}
		detail.UpdatedAt = &updated
		detail.Items, err = loadEntries(h.db, id)
		if utils.InternalError(w, err) {
			return
		}
		detail.ItemCount = len(detail.Items)
		detail.CanEdit = c.canEdit(user.UserID)
	}

	if detail.Covers == nil {
		detail.Covers = []string{}
	}
	if utils.InternalError(w, signCovers(user.UserID, detail.Covers)) {
		return
	}
	if utils.InternalError(w, signEntries(user.UserID, detail.Items)) {
		return
	}
	utils.JSON(w, detail)
}

// answers for a collection that can't be changed: a computed playlist, or one that isn't there
// or isn't the user's to change. Returns whether it answered
func refuseEdit(w http.ResponseWriter, id string, c stored, err error, userID string) bool {
	if isSmart(id) {
		utils.Error(w, http.StatusBadRequest, "this playlist fills itself from your list and can't be changed")
		return true
	}
	if errors.Is(err, errNotFound) {
		utils.Error(w, http.StatusNotFound, "not found")
		return true
	}
	if utils.InternalError(w, err) {
		return true
	}
	if !c.canEdit(userID) {
		utils.Error(w, http.StatusForbidden, "only its owner can change this")
		return true
	}
	return false
}

/*
Function:	Update
Purpose:	Rename a playlist or collection, or change its description
*/
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var c stored
	var err error
	if !isSmart(id) {
		c, err = load(h.db, id, user.UserID, false)
	}
	if refuseEdit(w, id, c, err, user.UserID) {
		return
	}

	sets := []string{}
	args := []any{id}
	if req.Name != nil {
		name, problem := cleanName(*req.Name)
		if problem != "" {
			utils.Error(w, http.StatusBadRequest, problem)
			return
		}
		args = append(args, name)
		sets = append(sets, "name = $"+strconv.Itoa(len(args)))
	}
	if req.Description != nil {
		description, problem := cleanDescription(req.Description)
		if problem != "" {
			utils.Error(w, http.StatusBadRequest, problem)
			return
		}
		args = append(args, description)
		sets = append(sets, "description = $"+strconv.Itoa(len(args)))
	}
	if len(sets) == 0 {
		utils.Error(w, http.StatusBadRequest, "nothing to change")
		return
	}

	_, err = h.db.Exec(`UPDATE collections SET `+strings.Join(sets, ", ")+`, updated_at = NOW() WHERE id = $1`, args...)
	if utils.InternalError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	Delete
Purpose:	Delete a playlist or collection. Its entries go with it, and so does it wherever it
was nested
*/
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id := chi.URLParam(r, "id")

	var c stored
	var err error
	if !isSmart(id) {
		c, err = load(h.db, id, user.UserID, false)
	}
	if refuseEdit(w, id, c, err, user.UserID) {
		return
	}

	_, err = h.db.Exec(`DELETE FROM collections WHERE id = $1`, id)
	if utils.InternalError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
