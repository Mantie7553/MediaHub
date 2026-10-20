package search

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Exercises validation only; none of these requests reach the database.
func TestSaveRefusesUnsafeMusicRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"option instead of a link", `{"title":"t","type":"music_track","action":"download","artist":"A","source_url":"--exec=touch /tmp/pwned"}`},
		{"another site", `{"title":"t","type":"music_track","action":"download","artist":"A","source_url":"http://sonarr:8989/api"}`},
		{"no link for a download", `{"title":"t","type":"music_track","action":"download","artist":"A"}`},
		{"bad link on a list-only save", `{"title":"t","type":"music_track","action":"list","artist":"A","source_url":"--exec=x"}`},
		{"artist that cleans to nothing", `{"title":"t","type":"music_track","action":"download","artist":" .. ","source_url":"https://youtu.be/ZRtdQ81jPUQ"}`},
		{"no artist", `{"title":"t","type":"music_track","action":"download","source_url":"https://youtu.be/ZRtdQ81jPUQ"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewHandler(nil).Save(rec, httptest.NewRequest(http.MethodPost, "/api/search/save", strings.NewReader(tt.body)))
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}
