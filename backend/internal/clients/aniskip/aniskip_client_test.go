package aniskip

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSkipTimes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/skip-times/21/1":
			if r.URL.Query().Get("episodeLength") != "1500.040" || len(r.URL.Query()["types[]"]) != len(skipTypes) {
				http.Error(w, "unexpected query "+r.URL.RawQuery, http.StatusBadRequest)
				return
			}
			// recorded from the API, plus a mixed opening and a type the player doesn't skip
			w.Write([]byte(`{"found":true,"results":[
				{"interval":{"startTime":28.783,"endTime":118.783},"skipType":"op","skipId":"a","episodeLength":1500.04},
				{"interval":{"startTime":1387.996,"endTime":1500},"skipType":"ed","skipId":"b","episodeLength":1500.146},
				{"interval":{"startTime":0,"endTime":20},"skipType":"mixed-op","skipId":"c","episodeLength":1500},
				{"interval":{"startTime":5,"endTime":5},"skipType":"recap","skipId":"d","episodeLength":1500},
				{"interval":{"startTime":10,"endTime":20},"skipType":"preview","skipId":"e","episodeLength":1500}
			],"message":"Successfully found skip times","statusCode":200}`))
		case "/v2/skip-times/999999/1":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"found":false,"results":[],"message":"No skip times found","statusCode":404}`))
		default:
			http.Error(w, "server error", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	c := &AniSkipClient{config: AniSkipConfig{BaseURL: srv.URL}, http: &http.Client{Timeout: 5 * time.Second}}

	got, err := c.SkipTimes(21, 1, 1500.04)
	if err != nil {
		t.Fatalf("SkipTimes: %v", err)
	}
	want := []Segment{{"op", 28.783, 118.783}, {"ed", 1387.996, 1500}, {"op", 0, 20}}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("segment %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	if got, err := c.SkipTimes(999999, 1, 1500); err != nil || len(got) != 0 {
		t.Errorf("unknown episode = %+v, %v; want none and no error", got, err)
	}
	if _, err := c.SkipTimes(1, 1, 1500); err == nil {
		t.Error("a server error should be an error")
	}
}
