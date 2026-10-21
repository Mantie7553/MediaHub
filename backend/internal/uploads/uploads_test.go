package uploads

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/go-chi/chi/v5"
)

func router(s *Store, userID string) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), auth.UserContextKey, &auth.UserContext{UserID: userID, Role: "admin"})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Post("/uploads", s.Start)
	r.Put("/uploads/{id}", s.Chunk)
	r.Delete("/uploads/{id}", s.Cancel)
	return r
}

func start(t *testing.T, h http.Handler, body string) (int, progress) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(body)))
	var p progress
	json.Unmarshal(rec.Body.Bytes(), &p)
	return rec.Code, p
}

func chunk(h http.Handler, id string, offset int64, data []byte) (int, progress) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/uploads/"+id, bytes.NewReader(data))
	req.Header.Set(OffsetHeader, strconv.FormatInt(offset, 10))
	h.ServeHTTP(rec, req)
	var p progress
	json.Unmarshal(rec.Body.Bytes(), &p)
	return rec.Code, p
}

func TestStartChecksTypeAndSize(t *testing.T) {
	s := NewStore(t.TempDir)
	h := router(s, "admin-1")
	tests := []struct {
		body   string
		status int
	}{
		{`{"name":"ep1.mkv","size":10,"kind":"video"}`, http.StatusCreated},
		{`{"name":"EP1.MKV","size":10,"kind":"video"}`, http.StatusCreated},
		{`{"name":"song.flac","size":10,"kind":"audio"}`, http.StatusBadRequest},
		{`{"name":"book.epub","size":10,"kind":"audio"}`, http.StatusBadRequest},
		{`{"name":"noext","size":10,"kind":"epub"}`, http.StatusBadRequest},
		{`{"name":"book.epub","size":0,"kind":"epub"}`, http.StatusBadRequest},
		{`{"name":"book.epub","size":999999999999,"kind":"epub"}`, http.StatusRequestEntityTooLarge},
		{`{"name":"a.exe","size":10,"kind":"program"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		if code, _ := start(t, h, tt.body); code != tt.status {
			t.Errorf("%s: status %d, want %d", tt.body, code, tt.status)
		}
	}
}

func TestChunksResumeAndClaim(t *testing.T) {
	s := NewStore(t.TempDir)
	h := router(s, "admin-1")
	data := []byte("0123456789")

	code, p := start(t, h, `{"name":"Show - S01E01.mkv","size":10,"kind":"video"}`)
	if code != http.StatusCreated || p.ID == "" {
		t.Fatalf("start: %d %+v", code, p)
	}

	if code, p := chunk(h, p.ID, 0, data[:4]); code != http.StatusOK || p.Received != 4 {
		t.Fatalf("first chunk: %d %+v", code, p)
	}
	// a retry of a chunk that already arrived says where to carry on from
	if code, p := chunk(h, p.ID, 0, data[:4]); code != http.StatusConflict || p.Received != 4 {
		t.Fatalf("repeated chunk: %d %+v", code, p)
	}
	if _, err := s.Claim(p.ID, "admin-1", Video); err != ErrIncomplete {
		t.Fatalf("claim before the end = %v, want ErrIncomplete", err)
	}
	// anything past the declared size is ignored
	if code, p := chunk(h, p.ID, 4, append(data[4:], 'x', 'y')); code != http.StatusOK || p.Received != 10 {
		t.Fatalf("last chunk: %d %+v", code, p)
	}

	if _, err := s.Claim(p.ID, "someone-else", Video); err != ErrNotFound {
		t.Errorf("another user's claim = %v, want ErrNotFound", err)
	}
	if _, err := s.Claim(p.ID, "admin-1", Audio); err != ErrWrongKind {
		t.Errorf("claim as audio = %v, want ErrWrongKind", err)
	}
	f, err := s.Claim(p.ID, "admin-1", Video)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(f.Path); string(got) != string(data) {
		t.Errorf("file = %q, want %q", got, data)
	}
	if f.Name != "Show - S01E01.mkv" || f.Ext() != ".mkv" {
		t.Errorf("file = %+v", f)
	}
	if _, err := s.Claim(p.ID, "admin-1", Video); err != ErrInUse {
		t.Errorf("second claim = %v, want ErrInUse", err)
	}

	s.Release(p.ID)
	if _, err := s.Claim(p.ID, "admin-1", Video); err != nil {
		t.Errorf("claim after release: %v", err)
	}
	s.Done(p.ID)
	if _, err := os.Stat(filepath.Dir(f.Path)); !os.IsNotExist(err) {
		t.Error("Done must remove the upload's folder")
	}
	if code, _ := chunk(h, p.ID, 10, data); code != http.StatusNotFound {
		t.Errorf("chunk after Done: %d, want 404", code)
	}
}

func TestChunkNeedsOffset(t *testing.T) {
	s := NewStore(t.TempDir)
	h := router(s, "admin-1")
	_, p := start(t, h, `{"name":"a.mp3","size":3,"kind":"audio"}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/uploads/"+p.ID, strings.NewReader("abc")))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("no offset: %d, want 400", rec.Code)
	}
}

func TestCancelRemovesTheFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(func() string { return dir })
	h := router(s, "admin-1")
	_, p := start(t, h, `{"name":"a.mp3","size":3,"kind":"audio"}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/uploads/"+p.ID, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("cancel: %d", rec.Code)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("left behind %v", entries)
	}
}

func TestIdleUploadsAreForgotten(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(func() string { return dir })
	now := time.Now()
	s.now = func() time.Time { return now }
	h := router(s, "admin-1")
	_, old := start(t, h, `{"name":"a.mp3","size":3,"kind":"audio"}`)

	now = now.Add(idleTimeout + time.Minute)
	start(t, h, `{"name":"b.mp3","size":3,"kind":"audio"}`)
	if _, ok := s.find(old.ID, "admin-1"); ok {
		t.Error("an upload idle past the timeout should be forgotten")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("want only the new upload's folder, got %d", len(entries))
	}
}

func TestMoveNeverReplaces(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.mkv")
	dest := filepath.Join(dir, "Show", "Season 01", "ep.mkv")
	os.WriteFile(src, []byte("new"), 0o644)

	if err := Move(src, dest); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "new" {
		t.Errorf("dest = %q", got)
	}

	os.WriteFile(src, []byte("other"), 0o644)
	if err := Move(src, dest); err == nil {
		t.Error("moving onto an existing file must fail")
	}
	if got, _ := os.ReadFile(dest); string(got) != "new" {
		t.Error("the existing file must be untouched")
	}
}
