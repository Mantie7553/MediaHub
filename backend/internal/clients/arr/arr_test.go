package arr

import (
	"net/http"
	"net/http/httptest"
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
