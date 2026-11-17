package webhooks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestActionFor(t *testing.T) {
	tests := []struct {
		event, reason string
		want          action
	}{
		{"Download", "", syncItem},
		{"Rename", "", syncItem},
		{"EpisodeFileDelete", "manual", clearFiles},
		{"EpisodeFileDelete", "missingFromDisk", clearFiles},
		{"EpisodeFileDelete", "upgrade", ignore},
		{"MovieFileDelete", "manual", clearFiles},
		{"MovieFileDelete", "upgrade", ignore},
		{"SeriesDelete", "", forgetItem},
		{"MovieDelete", "", forgetItem},
		// added with no files yet: the Download that follows syncs it. Syncing on add could race
		// a MediaHub request linking the series and create the title twice
		{"SeriesAdd", "", ignore},
		{"MovieAdded", "", ignore},
		{"Grab", "", ignore},
		{"Test", "", ignore},
	}
	for _, tt := range tests {
		if got := actionFor(tt.event, tt.reason); got != tt.want {
			t.Errorf("actionFor(%q, %q) = %d, want %d", tt.event, tt.reason, got, tt.want)
		}
	}
}

type fakeSyncer struct{ series, movies []int }

func (f *fakeSyncer) SyncSeriesByID(id int) error { f.series = append(f.series, id); return nil }
func (f *fakeSyncer) SyncMovieByID(id int) error  { f.movies = append(f.movies, id); return nil }

func TestDownloadSyncsTheItem(t *testing.T) {
	f := &fakeSyncer{}
	h := &Handler{syncer: f, run: func(fn func()) { fn() }}

	post := func(handler http.HandlerFunc, body string) int {
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
		return rec.Code
	}

	if code := post(h.SonarrWebhook, `{"eventType":"Download","series":{"id":12},"episodes":[{"id":3}]}`); code != http.StatusOK {
		t.Errorf("sonarr status = %d", code)
	}
	if code := post(h.RadarrWebhook, `{"eventType":"Rename","movie":{"id":7}}`); code != http.StatusOK {
		t.Errorf("radarr status = %d", code)
	}
	if code := post(h.SonarrWebhook, `{"eventType":"Test","series":{"id":1}}`); code != http.StatusOK {
		t.Errorf("test event status = %d", code)
	}
	if len(f.series) != 1 || f.series[0] != 12 || len(f.movies) != 1 || f.movies[0] != 7 {
		t.Errorf("synced series %v, movies %v; want [12], [7]", f.series, f.movies)
	}
}
