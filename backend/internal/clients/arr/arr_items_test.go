package arr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetItemsAndEpisodeFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "key" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path + "?" + r.URL.RawQuery {
		case "/api/v3/series/5?":
			w.Write([]byte(`{"id":5,"title":"Frieren","tvdbId":424536}`))
		case "/api/v3/movie/9?":
			w.Write([]byte(`{"id":9,"title":"Akira","hasFile":true,"movieFile":{"path":"/m/akira.mkv"}}`))
		case "/api/v3/episodefile?seriesId=5":
			w.Write([]byte(`[{"id":1,"path":"/a/e1.mkv"},{"id":2,"path":"/a/e2.mkv"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := &ArrClient{config: ArrConfig{BaseURL: srv.URL, APIKey: "key"}, http: &http.Client{Timeout: 5 * time.Second}}

	if s, err := c.GetSeries(5); err != nil || s.Title != "Frieren" || s.TvdbID != 424536 {
		t.Errorf("GetSeries = %+v, %v", s, err)
	}
	if m, err := c.GetMovie(9); err != nil || !m.HasFile || m.MovieFile.Path != "/m/akira.mkv" {
		t.Errorf("GetMovie = %+v, %v", m, err)
	}
	files, err := c.GetEpisodeFiles(5)
	if err != nil || len(files) != 2 || files[2] != "/a/e2.mkv" {
		t.Errorf("GetEpisodeFiles = %v, %v", files, err)
	}
	if _, err := c.GetSeries(6); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing series err = %v, want ErrNotFound", err)
	}
}

func TestAnimeMapsCache(t *testing.T) {
	now := time.Date(2026, 11, 16, 4, 0, 0, 0, time.UTC)
	calls := 0
	fail := false
	c := &animeMapsCache{
		now: func() time.Time { return now },
		fetch: func() (AnimeMaps, error) {
			calls++
			if fail {
				return AnimeMaps{}, errors.New("down")
			}
			return AnimeMaps{TvdbToAnilist: map[int]int{1: calls}}, nil
		},
	}

	// nothing cached and the download fails: empty maps, not nil ones
	fail = true
	m, err := c.load()
	if err == nil || m.TvdbToAnilist == nil || m.SeasonToMal == nil {
		t.Fatalf("first failed load = %+v, %v", m, err)
	}

	fail = false
	if m, _ := c.load(); m.TvdbToAnilist[1] != 2 {
		t.Fatalf("load = %v, want the fresh copy", m.TvdbToAnilist)
	}
	now = now.Add(time.Hour)
	if c.load(); calls != 2 {
		t.Errorf("calls = %d, a copy under a day old should be reused", calls)
	}

	// stale: fetched again; when that fails the old copy comes back with the error
	now = now.Add(24 * time.Hour)
	fail = true
	m, err = c.load()
	if err == nil || m.TvdbToAnilist[1] != 2 {
		t.Errorf("stale failed load = %v, %v; want the old copy and the error", m.TvdbToAnilist, err)
	}
}
