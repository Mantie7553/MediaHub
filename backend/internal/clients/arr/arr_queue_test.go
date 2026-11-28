package arr

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetQueueFollowsPages(t *testing.T) {
	pages := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "key" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/api/v3/queue" || r.URL.Query().Get("pageSize") != "200" {
			http.NotFound(w, r)
			return
		}
		pages++
		switch r.URL.Query().Get("page") {
		case "1":
			w.Write([]byte(`{"totalRecords":3,"records":[
				{"id":1,"seriesId":5,"title":"a","trackedDownloadStatus":"warning","added":"2026-11-27T10:00:00Z",
				 "statusMessages":[{"title":"a","messages":["The download is stalled with no connections"]}]},
				{"id":2,"seriesId":5,"title":"b","trackedDownloadStatus":"ok"}]}`))
		case "2":
			w.Write([]byte(`{"totalRecords":3,"records":[{"id":3,"movieId":9,"title":"c","trackedDownloadState":"importBlocked"}]}`))
		default:
			w.Write([]byte(`{"totalRecords":3,"records":[]}`))
		}
	}))
	defer srv.Close()
	c := &ArrClient{config: ArrConfig{BaseURL: srv.URL, APIKey: "key"}, http: &http.Client{Timeout: 5 * time.Second}}

	items, err := c.GetQueue()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || pages != 2 {
		t.Fatalf("got %d items over %d pages, want 3 over 2", len(items), pages)
	}
	first := items[0]
	if first.SeriesID != 5 || first.TrackedDownloadStatus != "warning" || first.Added.IsZero() ||
		first.StatusMessages[0].Messages[0] != "The download is stalled with no connections" {
		t.Errorf("first item = %+v", first)
	}
	if items[2].MovieID != 9 || items[2].TrackedDownloadState != "importBlocked" || !items[2].Added.IsZero() {
		t.Errorf("third item = %+v", items[2])
	}
}

func TestConfigured(t *testing.T) {
	if (&ArrClient{}).Configured() {
		t.Error("a client with no URL is configured")
	}
	if !(&ArrClient{config: ArrConfig{BaseURL: "http://sonarr:8989"}}).Configured() {
		t.Error("a client with a URL isn't configured")
	}
}
