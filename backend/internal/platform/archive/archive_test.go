package archive

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Exercises validation only; none of these requests reach the database.
func TestHandlerRejectsBadRequests(t *testing.T) {
	h := Handler(nil, "download_jobs")
	tooMany := `{"ids":["` + strings.Repeat(`00000000-0000-0000-0000-000000000000","`, maxIDs) +
		`00000000-0000-0000-0000-000000000000"]}`

	tests := []struct {
		name string
		body string
	}{
		{"not json", `archive everything`},
		{"no ids", `{"archived":true}`},
		{"empty ids", `{"ids":[],"archived":true}`},
		{"not a uuid", `{"ids":["1; DROP TABLE download_jobs"],"archived":true}`},
		{"too many ids", tooMany},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body)))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestFilter(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"/api/admin/jobs", "j.archived_at IS NULL"},
		{"/api/admin/jobs?archived=false", "j.archived_at IS NULL"},
		{"/api/admin/jobs?archived=true", "j.archived_at IS NOT NULL"},
	}
	for _, tt := range tests {
		if got := Filter(httptest.NewRequest(http.MethodGet, tt.url, nil), "j"); got != tt.want {
			t.Errorf("Filter(%s) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
