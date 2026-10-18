package arr

import (
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
