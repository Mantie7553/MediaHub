/*
Package uploads receives large files from admins in pieces. Cloudflare refuses requests over
100 MB, so an episode or movie can't arrive in one: the browser starts an upload with the
file's size, sends it in chunks of at most MaxChunk, then names the finished upload in the
request that adds it to the library, which claims the file and moves it into place.

Uploads are written to MEDIA_ROOT/temp/upload-<id>/, on the same disk as the library so the
move is a rename. They are kept in memory, so a restart forgets them; the temp folder cleanup
removes their files after 48 hours
*/
package uploads

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

// Kind is what an upload will become, which decides the file types and size allowed
type Kind string

const (
	Video Kind = "video" // an episode or a movie
	Audio Kind = "audio" // a music track
	EPUB  Kind = "epub"  // a light novel volume
)

type kindRules struct {
	maxBytes   int64
	extensions []string
}

var rules = map[Kind]kindRules{
	// a 4K remux runs past 60 GB; this is well above an anime episode or a normal movie
	Video: {80 << 30, []string{".mkv", ".mp4", ".m4v", ".webm", ".avi", ".mov", ".ts"}},
	// the player and the music sync only read mp3
	Audio: {500 << 20, []string{".mp3"}},
	EPUB:  {500 << 20, []string{".epub"}},
}

// largest piece accepted in one request, under Cloudflare's 100 MB limit
const MaxChunk = 64 << 20

// an upload nobody has touched for this long is forgotten; its files go with the temp cleanup
const idleTimeout = 48 * time.Hour

// OffsetHeader carries where in the file a chunk starts
const OffsetHeader = "Upload-Offset"

var (
	ErrNotFound   = errors.New("that upload isn't here any more, it may have expired or the server restarted. Upload the file again")
	ErrIncomplete = errors.New("the upload hasn't finished yet")
	ErrWrongKind  = errors.New("that upload was started for a different kind of file")
	ErrInUse      = errors.New("that upload is already being added")
)

type upload struct {
	mu       sync.Mutex
	id       string
	owner    string
	kind     Kind
	name     string
	path     string
	size     int64
	received int64
	claimed  bool
	touched  time.Time
}

// Store holds the uploads in progress
type Store struct {
	mu      sync.Mutex
	uploads map[string]*upload
	// where upload folders are made, MEDIA_ROOT/temp by default
	dir func() string
	now func() time.Time
}

var defaultStore = NewStore(func() string {
	return filepath.Join(os.Getenv("MEDIA_ROOT"), "temp")
})

// Default is the store the routes share
func Default() *Store { return defaultStore }

// NewStore makes a store that keeps upload folders in dir()
func NewStore(dir func() string) *Store {
	return &Store{uploads: map[string]*upload{}, dir: dir, now: time.Now}
}

type startRequest struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Kind Kind   `json:"kind"`
}

type progress struct {
	ID       string `json:"id"`
	Received int64  `json:"received"`
	Size     int64  `json:"size"`
	MaxChunk int64  `json:"max_chunk"`
}

/*
Function:	Start
Purpose:	Admin: begin an upload. The name's extension and the size are checked against what
the kind allows, so a wrong file is refused before any of it is sent
Params:
  - w: http response writer
  - r: JSON { name, size, kind }
*/
func (s *Store) Start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	rule, ok := rules[req.Kind]
	if !ok {
		utils.Error(w, http.StatusBadRequest, "kind must be video, audio or epub")
		return
	}
	ext := strings.ToLower(filepath.Ext(req.Name))
	if !slices.Contains(rule.extensions, ext) {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("%s files can't be added here, use %s", orNone(ext), strings.Join(rule.extensions, ", ")))
		return
	}
	if req.Size <= 0 {
		utils.Error(w, http.StatusBadRequest, "the file is empty")
		return
	}
	if req.Size > rule.maxBytes {
		utils.Error(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("the file is larger than the %s allowed", readableSize(rule.maxBytes)))
		return
	}

	s.forgetIdle()
	id, err := newID()
	if utils.InternalError(w, err) {
		return
	}
	dir := filepath.Join(s.dir(), "upload-"+id)
	if err := os.MkdirAll(dir, 0o755); utils.InternalError(w, err) {
		return
	}
	path := filepath.Join(dir, "file"+ext)
	f, err := os.Create(path)
	if err != nil {
		os.RemoveAll(dir)
		utils.InternalError(w, err)
		return
	}
	f.Close()

	u := &upload{
		id: id, owner: auth.GetUser(r).UserID, kind: req.Kind, name: filepath.Base(req.Name),
		path: path, size: req.Size, touched: s.now(),
	}
	s.mu.Lock()
	s.uploads[id] = u
	s.mu.Unlock()

	utils.JSON(w, progress{ID: id, Size: req.Size, MaxChunk: MaxChunk}, http.StatusCreated)
}

/*
Function:	Chunk
Purpose:	Admin: add the next piece of an upload. The Upload-Offset header must equal the
bytes received so far; when it doesn't (a retry after a dropped connection), the answer is 409
with what was received, so the browser can carry on from there
Params:
  - w: http response writer
  - r: the raw bytes, application/octet-stream
*/
func (s *Store) Chunk(w http.ResponseWriter, r *http.Request) {
	u, ok := s.find(chi.URLParam(r, "id"), auth.GetUser(r).UserID)
	if !ok {
		utils.Error(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	offset, err := strconv.ParseInt(r.Header.Get(OffsetHeader), 10, 64)
	if err != nil || offset < 0 {
		utils.Error(w, http.StatusBadRequest, OffsetHeader+" header is required")
		return
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	u.touched = s.now()
	if u.claimed {
		utils.Error(w, http.StatusConflict, ErrInUse.Error())
		return
	}
	if offset != u.received {
		utils.JSON(w, u.progress(), http.StatusConflict)
		return
	}

	f, err := os.OpenFile(u.path, os.O_WRONLY, 0)
	if utils.InternalError(w, err) {
		return
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); utils.InternalError(w, err) {
		return
	}

	// never more than one chunk, or more than the file has left
	body := http.MaxBytesReader(w, r.Body, MaxChunk)
	n, copyErr := io.Copy(f, io.LimitReader(body, u.size-u.received))
	// what reached the disk counts even when the connection dropped part way
	u.received += n
	if copyErr != nil {
		var tooBig *http.MaxBytesError
		if errors.As(copyErr, &tooBig) {
			utils.Error(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("chunks can be at most %d MB", MaxChunk>>20))
			return
		}
		utils.JSON(w, u.progress(), http.StatusConflict)
		return
	}
	utils.JSON(w, u.progress())
}

/*
Function:	Cancel
Purpose:	Admin: give up on an upload and delete what was received
*/
func (s *Store) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, ok := s.find(id, auth.GetUser(r).UserID)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	u.mu.Lock()
	claimed := u.claimed
	u.mu.Unlock()
	if claimed {
		utils.Error(w, http.StatusConflict, ErrInUse.Error())
		return
	}
	s.Done(id)
	w.WriteHeader(http.StatusNoContent)
}

// File is a finished upload claimed by the request adding it to the library
type File struct {
	ID string
	// Path is where it is on disk now, to check and then move
	Path string
	// Name is the name it had on the admin's computer
	Name string
	Size int64
}

// Ext is the file's extension in lower case, such as ".mkv"
func (f File) Ext() string {
	return strings.ToLower(filepath.Ext(f.Path))
}

/*
Function:	Claim
Purpose:	take a finished upload to add it to the library. Until Done or Release no one else
can claim it. Release it when adding fails, so the admin can fix the details and try again
without sending the file again
Params:
  - id: the upload's id
  - userID: who is adding it, the same admin who uploaded it
  - kind: what it is being added as
*/
func (s *Store) Claim(id, userID string, kind Kind) (File, error) {
	u, ok := s.find(id, userID)
	if !ok {
		return File{}, ErrNotFound
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	switch {
	case u.kind != kind:
		return File{}, ErrWrongKind
	case u.claimed:
		return File{}, ErrInUse
	case u.received != u.size:
		return File{}, ErrIncomplete
	}
	u.claimed = true
	u.touched = s.now()
	return File{ID: u.id, Path: u.path, Name: u.name, Size: u.size}, nil
}

/*
Function:	Release
Purpose:	hand a claimed upload back after adding it failed. Its file must be back at its
original path
*/
func (s *Store) Release(id string) {
	s.mu.Lock()
	u := s.uploads[id]
	s.mu.Unlock()
	if u == nil {
		return
	}
	u.mu.Lock()
	u.claimed = false
	u.touched = s.now()
	u.mu.Unlock()
}

/*
Function:	Done
Purpose:	forget an upload and delete its folder, once its file has been moved into the
library (or it was cancelled)
*/
func (s *Store) Done(id string) {
	s.mu.Lock()
	u := s.uploads[id]
	delete(s.uploads, id)
	s.mu.Unlock()
	if u != nil {
		os.RemoveAll(filepath.Dir(u.path))
	}
}

// find an upload belonging to userID
func (s *Store) find(id, userID string) (*upload, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.uploads[id]
	if !ok || u.owner != userID {
		return nil, false
	}
	return u, true
}

// forgetIdle drops uploads abandoned long ago, so the map doesn't grow without end
func (s *Store) forgetIdle() {
	s.mu.Lock()
	var idle []string
	for id, u := range s.uploads {
		u.mu.Lock()
		if !u.claimed && s.now().Sub(u.touched) > idleTimeout {
			idle = append(idle, id)
		}
		u.mu.Unlock()
	}
	s.mu.Unlock()
	for _, id := range idle {
		s.Done(id)
	}
}

func (u *upload) progress() progress {
	return progress{ID: u.id, Received: u.received, Size: u.size, MaxChunk: MaxChunk}
}

/*
Function:	Move
Purpose:	move a file into the library without replacing anything there. A rename when both
are on one disk, otherwise a copy that is only put in place once complete
Params:
  - src: the file now
  - dest: where it goes; its folder is created
*/
func Move(src, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(dest))
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dest); err == nil {
		return nil
	}

	// another disk: copy beside the destination, then rename that into place
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	in.Close()
	os.Remove(src)
	return nil
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// readableSize writes a limit as "500 MB" or "80 GB"
func readableSize(n int64) string {
	if n >= 1<<30 {
		return fmt.Sprintf("%d GB", n>>30)
	}
	return fmt.Sprintf("%d MB", n>>20)
}

func orNone(ext string) string {
	if ext == "" {
		return "Files without an extension"
	}
	return ext
}
