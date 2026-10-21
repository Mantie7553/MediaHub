package arr

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestQualityProfileID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/qualityprofile" || r.Header.Get("X-Api-Key") != "key" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Write([]byte(`[{"id":1,"name":"Any"},{"id":7,"name":"Anime"}]`))
	}))
	defer srv.Close()
	c := &ArrClient{config: ArrConfig{BaseURL: srv.URL, APIKey: "key"}, http: &http.Client{Timeout: 5 * time.Second}}

	if id, err := c.QualityProfileID("anime"); err != nil || id != 7 {
		t.Errorf("anime = %d, %v; want 7", id, err)
	}
	if _, err := c.QualityProfileID("Ultra-HD"); err == nil {
		t.Error("missing profile should be an error")
	}
}

func TestDeleteSeriesAndMovie(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.Header.Get("X-Api-Key") != "key" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		got = append(got, r.URL.Path+"?"+r.URL.RawQuery)
		switch r.URL.Path {
		case "/api/v3/series/5", "/api/v3/movie/9":
			w.WriteHeader(http.StatusOK)
		case "/api/v3/series/404":
			http.Error(w, "NotFound", http.StatusNotFound)
		default:
			http.Error(w, "database is locked", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	c := &ArrClient{config: ArrConfig{BaseURL: srv.URL, APIKey: "key"}, http: &http.Client{Timeout: 5 * time.Second}}

	if err := c.DeleteSeries(5); err != nil {
		t.Errorf("DeleteSeries: %v", err)
	}
	if err := c.DeleteMovie(9); err != nil {
		t.Errorf("DeleteMovie: %v", err)
	}
	if err := c.DeleteSeries(404); err != nil {
		t.Errorf("already deleted series should succeed, got %v", err)
	}
	if err := c.DeleteSeries(1); err == nil || !strings.Contains(err.Error(), "database is locked") {
		t.Errorf("server error should be returned with its message, got %v", err)
	}

	if len(got) < 2 || !strings.Contains(got[0], "deleteFiles=true") || !strings.Contains(got[1], "deleteFiles=true") {
		t.Errorf("files must be deleted too: %v", got)
	}
}

func TestRemoveOneEpisode(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "key" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		got = append(got, r.Method+" "+r.URL.Path)
		switch r.Method + " " + r.URL.Path {
		case "PUT /api/v3/episode/monitor":
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"episodeIds":[12],"monitored":false}` {
				http.Error(w, "bad body "+string(body), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		case "GET /api/v3/episode/12":
			w.Write([]byte(`{"id":12,"episodeFileId":40,"hasFile":true}`))
		case "GET /api/v3/episode/13":
			w.Write([]byte(`{"id":13,"episodeFileId":0,"hasFile":false}`))
		case "GET /api/v3/episode/99":
			http.Error(w, "NotFound", http.StatusNotFound)
		case "DELETE /api/v3/episodefile/40":
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "unexpected", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	c := &ArrClient{config: ArrConfig{BaseURL: srv.URL, APIKey: "key"}, http: &http.Client{Timeout: 5 * time.Second}}

	if err := c.UnmonitorEpisode(12); err != nil {
		t.Errorf("UnmonitorEpisode: %v", err)
	}
	if id, err := c.EpisodeFileID(12); err != nil || id != 40 {
		t.Errorf("EpisodeFileID(12) = %d, %v; want 40", id, err)
	}
	if id, err := c.EpisodeFileID(13); err != nil || id != 0 {
		t.Errorf("an episode without a file = %d, %v; want 0", id, err)
	}
	if id, err := c.EpisodeFileID(99); err != nil || id != 0 {
		t.Errorf("an episode Sonarr forgot = %d, %v; want 0", id, err)
	}
	if err := c.DeleteEpisodeFile(40); err != nil {
		t.Errorf("DeleteEpisodeFile: %v", err)
	}
	if err := c.UnmonitorEpisode(5); err == nil {
		t.Error("a failed unmonitor must be an error")
	}
}
