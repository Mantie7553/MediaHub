package search

import (
	"database/sql"
	"testing"
)

type recordingExecer struct {
	query string
	args  []any
}

func (r *recordingExecer) Exec(query string, args ...any) (sql.Result, error) {
	r.query, r.args = query, args
	return nil, nil
}

func TestSaveListProgressWritesWhatTheAppReads(t *testing.T) {
	tests := []struct {
		mediaType string
		want      string
	}{
		{"anime", markEpisodesWatched},
		{"manga", markChaptersRead},
	}
	for _, tt := range tests {
		t.Run(tt.mediaType, func(t *testing.T) {
			db := &recordingExecer{}
			if err := saveListProgress(db, "user", "title", tt.mediaType, 4); err != nil {
				t.Fatal(err)
			}
			if db.query != tt.want {
				t.Fatalf("ran %q, want the %s query", db.query, tt.mediaType)
			}
			if len(db.args) != 3 || db.args[0] != "user" || db.args[1] != "title" || db.args[2] != 4 {
				t.Fatalf("args = %v, want [user title 4]", db.args)
			}
		})
	}
}

func TestSaveListProgressSkipsTypesWithoutACount(t *testing.T) {
	db := &recordingExecer{}
	if err := saveListProgress(db, "user", "title", "movie", 1); err != nil {
		t.Fatal(err)
	}
	if db.query != "" {
		t.Fatalf("ran %q for a movie, want nothing", db.query)
	}
}
