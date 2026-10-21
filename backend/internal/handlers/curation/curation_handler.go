/*
Package curation lets admins fix the library by hand where the metadata sources fall short:
rename titles and the episodes, chapters, volumes and tracks in them, write descriptions,
replace covers and add files. Fields an admin edits are locked (see package locks) so the
nightly library sync doesn't put the old values back
*/
package curation

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Mantie7553/MediaHub/backend/internal/covers"
	"github.com/Mantie7553/MediaHub/backend/internal/locks"
	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/Mantie7553/MediaHub/backend/internal/uploads"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

type Handler struct {
	db      *sql.DB
	uploads *uploads.Store
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db, uploads: uploads.Default()}
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// longest values kept, well past real titles and descriptions
const (
	maxTitleRunes       = 300
	maxDescriptionRunes = 20000
	maxGenres           = 30
	maxGenreRunes       = 50
	maxCount            = 100000
)

// the metadata table each type keeps its own details in, and the fields an admin can set there
var metaFields = map[string]struct {
	table  string
	fields []string
}{
	"anime":       {"anime_metadata", []string{locks.Genres, locks.Studio, locks.Status}},
	"movie":       {"movie_metadata", []string{locks.Genres, locks.Director, locks.RuntimeMins}},
	"manga":       {"manga_metadata", []string{locks.Genres, locks.Status, locks.TotalChapters}},
	"light_novel": {"light_novel_metadata", []string{locks.Genres, locks.Author, locks.TotalVolumes}},
	"music_track": {"music_metadata", []string{locks.Genres, locks.Artist, locks.TrackNumber}},
}

// the statuses each type's column (an enum) takes
var statuses = map[string][]string{
	"anime": {"airing", "finished", "upcoming"},
	"manga": {"ongoing", "completed", "hiatus"},
}

// a column to set and its new value
type assignment struct {
	column string
	value  any
}

// badRequest is a problem with what the admin entered, sent back as a 400
type badRequest string

func (b badRequest) Error() string { return string(b) }

/*
Function:	EditMedia
Purpose:	Admin: change a title's details: its name, description, release date, genres and
the fields its type has (studio, director, author, ...). Every field changed is locked so the
library syncs keep it; fields named in "unlock" are handed back to them. Handing back the
cover clears it, and the next sync fills in the source's cover again

Params:
  - w: http response writer
  - r: JSON editMediaRequest
*/
func (h *Handler) EditMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !uuidRe.MatchString(id) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	var req editMediaRequest
	if !decode(w, r, &req) {
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	var mediaType string
	var locked []string
	err = tx.QueryRow(`SELECT type, locked_fields FROM media_items WHERE id = $1 FOR UPDATE`, id).
		Scan(&mediaType, pq.Array(&locked))
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	item, meta, edited, err := mediaChanges(req, mediaType)
	if respondBadRequest(w, err) {
		return
	}
	for _, field := range req.Unlock {
		if !locks.Known(field) {
			utils.Error(w, http.StatusBadRequest, fmt.Sprintf("%q isn't a field that can be handed back", field))
			return
		}
	}

	// handing the cover back clears it, so the next sync puts the source's cover in its place
	clearCover := slices.Contains(req.Unlock, locks.Cover) && slices.Contains(locked, locks.Cover)
	if clearCover {
		item = append(item, assignment{"cover_image_url", nil})
	}
	item = append(item, assignment{"locked_fields", pq.Array(nextLocks(locked, edited, req.Unlock))})

	query, args := updateQuery("media_items", id, item)
	if _, err := tx.Exec(query+", updated_at = NOW() WHERE id = $1", args...); utils.InternalError(w, err) {
		return
	}

	if len(meta) > 0 {
		table := metaFields[mediaType].table
		// a title saved before its metadata was fetched may have no row yet. Tracks always
		// have one (it holds the file), and can't be made without an artist
		if mediaType != "music_track" {
			if _, err := tx.Exec(`INSERT INTO `+table+` (media_item_id) VALUES ($1) ON CONFLICT DO NOTHING`, id); utils.InternalError(w, err) {
				return
			}
		}
		query, args := updateQuery(table, id, meta)
		if _, err := tx.Exec(query+" WHERE media_item_id = $1", args...); utils.InternalError(w, err) {
			return
		}
	}

	if utils.InternalError(w, tx.Commit()) {
		return
	}
	if clearCover {
		if err := covers.Remove(id); err != nil {
			logger.Warn("removing the uploaded cover of %s: %v", id, err)
		}
	}

	logger.Info("user %s edited %s (%s)", auth.GetUser(r).UserID, id, strings.Join(edited, ", "))
	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	mediaChanges
Purpose:	check an edit and turn it into the columns to set on media_items and on the type's
metadata table
Returns:	those, the locks.* names of the fields edited, and a badRequest for a value that
can't be saved
*/
func mediaChanges(req editMediaRequest, mediaType string) (item, meta []assignment, edited []string, err error) {
	allowed := metaFields[mediaType].fields
	// a type's own field, refused for types without it
	own := func(field string, set bool) bool {
		if !set {
			return false
		}
		if !slices.Contains(allowed, field) {
			err = badRequest(fmt.Sprintf("%s has no %s", withArticle(mediaType), strings.ReplaceAll(field, "_", " ")))
			return false
		}
		edited = append(edited, field)
		return true
	}

	if req.Title.Set {
		title, ok := cleanTitle(req.Title.Value)
		if !ok {
			return nil, nil, nil, badRequest(fmt.Sprintf("the title must be 1 to %d characters", maxTitleRunes))
		}
		item = append(item, assignment{"title", title})
		edited = append(edited, locks.Title)
	}
	if req.Description.Set {
		description, ok := cleanText(req.Description.Value, maxDescriptionRunes)
		if !ok {
			return nil, nil, nil, badRequest(fmt.Sprintf("the description can be at most %d characters", maxDescriptionRunes))
		}
		item = append(item, assignment{"description", description})
		edited = append(edited, locks.Description)
	}
	if req.ReleaseDate.Set {
		date, ok := cleanDate(req.ReleaseDate.Value)
		if !ok {
			return nil, nil, nil, badRequest("the release date must be YYYY-MM-DD")
		}
		item = append(item, assignment{"release_date", date})
		edited = append(edited, locks.ReleaseDate)
	}

	if own(locks.Genres, req.Genres.Set) {
		genres, ok := cleanGenres(req.Genres.Value)
		if !ok {
			return nil, nil, nil, badRequest(fmt.Sprintf("at most %d genres of up to %d characters each", maxGenres, maxGenreRunes))
		}
		meta = append(meta, assignment{"genres", pq.Array(genres)})
	}
	if own(locks.Status, req.Status.Set) {
		status, _ := cleanText(req.Status.Value, 50)
		if status != nil && !slices.Contains(statuses[mediaType], *status) {
			return nil, nil, nil, badRequest("the status must be one of " + strings.Join(statuses[mediaType], ", "))
		}
		meta = append(meta, assignment{"status", status})
	}
	for _, f := range []struct {
		field string
		value optional[string]
	}{{locks.Studio, req.Studio}, {locks.Director, req.Director}, {locks.Author, req.Author}} {
		if own(f.field, f.value.Set) {
			text, ok := cleanText(f.value.Value, maxTitleRunes)
			if !ok {
				return nil, nil, nil, badRequest(fmt.Sprintf("the %s can be at most %d characters", f.field, maxTitleRunes))
			}
			meta = append(meta, assignment{f.field, text})
		}
	}
	if own(locks.Artist, req.Artist.Set) {
		artist, ok := cleanTitle(req.Artist.Value)
		if !ok {
			return nil, nil, nil, badRequest("a track needs an artist")
		}
		meta = append(meta, assignment{"artist", artist})
	}
	for _, f := range []struct {
		field string
		value optional[int]
	}{{locks.RuntimeMins, req.RuntimeMins}, {locks.TotalChapters, req.TotalChapters}, {locks.TotalVolumes, req.TotalVolumes}, {locks.TrackNumber, req.TrackNumber}} {
		if own(f.field, f.value.Set) {
			if v := f.value.Value; v != nil && (*v < 0 || *v > maxCount) {
				return nil, nil, nil, badRequest(fmt.Sprintf("the %s must be between 0 and %d", strings.ReplaceAll(f.field, "_", " "), maxCount))
			}
			meta = append(meta, assignment{f.field, f.value.Value})
		}
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if len(item) == 0 && len(meta) == 0 && len(req.Unlock) == 0 {
		return nil, nil, nil, badRequest("nothing to change")
	}
	return item, meta, edited, nil
}

/*
Function:	nextLocks
Purpose:	the locked fields after an edit: the ones handed back are dropped, the ones edited
added. Sorted, so the stored list doesn't depend on the order of edits
*/
func nextLocks(locked, edited, unlock []string) []string {
	out := []string{}
	for _, f := range locked {
		if !slices.Contains(unlock, f) && !slices.Contains(out, f) {
			out = append(out, f)
		}
	}
	for _, f := range edited {
		if !slices.Contains(out, f) {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}

/*
Function:	updateQuery
Purpose:	"UPDATE table SET a = $2, b = $3" for the assignments, with $1 left for the key
Params:
  - table: the table, never user input
  - key: the row's key, the first argument, for the WHERE the caller writes
  - sets: the columns (never user input) and values
*/
func updateQuery(table, key string, sets []assignment) (string, []any) {
	parts := make([]string, len(sets))
	args := []any{key}
	for i, s := range sets {
		args = append(args, s.value)
		parts[i] = fmt.Sprintf("%s = $%d", s.column, len(args))
	}
	return "UPDATE " + table + " SET " + strings.Join(parts, ", "), args
}

/*
Function:	EditAlbum
Purpose:	Admin: rename an album, change its artist, release date or description. The
album's tracks that were credited to the old artist move to the new one. Two albums can't
share a title and artist
Params:
  - w: http response writer
  - r: JSON editAlbumRequest
*/
func (h *Handler) EditAlbum(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !uuidRe.MatchString(id) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	var req editAlbumRequest
	if !decode(w, r, &req) {
		return
	}

	var sets []assignment
	var newArtist *string
	if req.Title.Set {
		title, ok := cleanTitle(req.Title.Value)
		if !ok {
			utils.Error(w, http.StatusBadRequest, fmt.Sprintf("the title must be 1 to %d characters", maxTitleRunes))
			return
		}
		sets = append(sets, assignment{"title", title})
	}
	if req.Artist.Set {
		artist, ok := cleanTitle(req.Artist.Value)
		if !ok {
			utils.Error(w, http.StatusBadRequest, "an album needs an artist")
			return
		}
		sets = append(sets, assignment{"artist", artist})
		newArtist = &artist
	}
	if req.ReleaseDate.Set {
		date, ok := cleanDate(req.ReleaseDate.Value)
		if !ok {
			utils.Error(w, http.StatusBadRequest, "the release date must be YYYY-MM-DD")
			return
		}
		sets = append(sets, assignment{"release_date", date})
	}
	if req.Description.Set {
		description, ok := cleanText(req.Description.Value, maxDescriptionRunes)
		if !ok {
			utils.Error(w, http.StatusBadRequest, fmt.Sprintf("the description can be at most %d characters", maxDescriptionRunes))
			return
		}
		sets = append(sets, assignment{"description", description})
	}
	if len(sets) == 0 {
		utils.Error(w, http.StatusBadRequest, "nothing to change")
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()

	var oldArtist string
	err = tx.QueryRow(`SELECT artist FROM albums WHERE id = $1 FOR UPDATE`, id).Scan(&oldArtist)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	query, args := updateQuery("albums", id, sets)
	if _, err := tx.Exec(query+", updated_at = NOW() WHERE id = $1", args...); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			utils.Error(w, http.StatusConflict, "there's already an album with this title by this artist")
			return
		}
		utils.InternalError(w, err)
		return
	}
	// the music sync leaves tracks it already knows alone, so this sticks
	if newArtist != nil && *newArtist != oldArtist {
		if _, err := tx.Exec(
			`UPDATE music_metadata SET artist = $1 WHERE album_id = $2 AND artist = $3`,
			*newArtist, id, oldArtist,
		); utils.InternalError(w, err) {
			return
		}
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// largest cover image accepted, well above a high resolution poster scan
const maxCoverBytes = 10 << 20

/*
Function:	UploadCover
Purpose:	Admin: replace a title's cover (anime, movie, manga, light novel or track; albums
have their own). It is kept in MEDIA_ROOT/Covers and locked, so syncs don't replace it
Params:
  - w: http response writer
  - r: multipart form with "file", a JPEG, PNG, WebP or GIF image
*/
func (h *Handler) UploadCover(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path, ok := covers.Path(id)
	if !ok {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	var exists bool
	if err := h.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM media_items WHERE id = $1)`, id).Scan(&exists); utils.InternalError(w, err) {
		return
	}
	if !exists {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}

	data, ok := readImage(w, r)
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); utils.InternalError(w, err) {
		return
	}
	if err := musicfiles.WriteCover(path, data); err != nil {
		logger.Error("saving the cover for %s: %s", id, err.Error())
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	cache.Disk().Forget(path)

	coverURL := covers.URL(id)
	_, err := h.db.Exec(
		`UPDATE media_items SET cover_image_url = $1,
		locked_fields = array_append(array_remove(locked_fields, $2), $2),
		updated_at = NOW()
		WHERE id = $3`,
		coverURL, locks.Cover, id,
	)
	if utils.InternalError(w, err) {
		return
	}
	if utils.InternalError(w, auth.SignCoverURL(&coverURL, auth.GetUser(r).UserID)) {
		return
	}
	utils.JSON(w, map[string]string{"cover_image_url": coverURL})
}

/*
Function:	readImage
Purpose:	read the "file" of a multipart upload and check it is an image a browser shows.
Responds with the problem and returns false when it isn't
*/
func readImage(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCoverBytes+(1<<20))
	if err := r.ParseMultipartForm(maxCoverBytes); err != nil {
		utils.Error(w, http.StatusRequestEntityTooLarge, "the image is too large or the upload was cut off")
		return nil, false
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "no image in the upload")
		return nil, false
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxCoverBytes+1))
	if utils.InternalError(w, err) {
		return nil, false
	}
	if len(data) > maxCoverBytes {
		utils.Error(w, http.StatusRequestEntityTooLarge, "the image is too large")
		return nil, false
	}
	if musicfiles.ImageType(data) == "" {
		utils.Error(w, http.StatusBadRequest, "covers must be JPEG, PNG, WebP or GIF images")
		return nil, false
	}
	return data, true
}

/*
Function:	ServeCover
Purpose:	an uploaded cover, loaded by <img> tags with a media token. Served with its real
image type and no-cache, since a replaced cover keeps its URL
*/
func (h *Handler) ServeCover(w http.ResponseWriter, r *http.Request) {
	path, ok := covers.Path(chi.URLParam(r, "id"))
	if !ok {
		utils.Error(w, http.StatusNotFound, "cover not found")
		return
	}
	f, err := os.Open(cache.Disk().Local(path))
	if os.IsNotExist(err) {
		utils.Error(w, http.StatusNotFound, "cover not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if utils.InternalError(w, err) {
		return
	}

	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	contentType := musicfiles.ImageType(head[:n])
	if contentType == "" {
		utils.Error(w, http.StatusNotFound, "cover not found")
		return
	}
	if _, err := f.Seek(0, io.SeekStart); utils.InternalError(w, err) {
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, "", info.ModTime(), f)
}

// withArticle names a media type for a message: "an anime", "a light novel"
func withArticle(mediaType string) string {
	name := strings.ReplaceAll(mediaType, "_", " ")
	if strings.ContainsRune("aeiou", rune(name[0])) {
		return "an " + name
	}
	return "a " + name
}

// cleanTitle trims a required name, false when it is empty or too long
func cleanTitle(s *string) (string, bool) {
	if s == nil {
		return "", false
	}
	t := strings.TrimSpace(*s)
	return t, t != "" && utf8.RuneCountInString(t) <= maxTitleRunes
}

// cleanText trims optional text: nil when empty (stored as NULL), false when too long
func cleanText(s *string, max int) (*string, bool) {
	if s == nil {
		return nil, true
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil, true
	}
	return &t, utf8.RuneCountInString(t) <= max
}

// cleanDate checks an optional YYYY-MM-DD date: nil when empty, false when it isn't a date
func cleanDate(s *string) (*string, bool) {
	t, _ := cleanText(s, 10)
	if t == nil {
		return nil, true
	}
	if _, err := time.Parse("2006-01-02", *t); err != nil {
		return nil, false
	}
	return t, true
}

// cleanGenres trims genres and drops empty and repeated ones (ignoring case)
func cleanGenres(list *[]string) ([]string, bool) {
	out := []string{}
	if list == nil {
		return out, true
	}
	seen := map[string]bool{}
	for _, g := range *list {
		g = strings.TrimSpace(g)
		key := strings.ToLower(g)
		if g == "" || seen[key] {
			continue
		}
		if utf8.RuneCountInString(g) > maxGenreRunes {
			return nil, false
		}
		seen[key] = true
		out = append(out, g)
	}
	return out, len(out) <= maxGenres
}

// respondBadRequest sends a badRequest as a 400 and anything else as a 500
func respondBadRequest(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var bad badRequest
	if errors.As(err, &bad) {
		utils.Error(w, http.StatusBadRequest, string(bad))
		return true
	}
	return utils.InternalError(w, err)
}
