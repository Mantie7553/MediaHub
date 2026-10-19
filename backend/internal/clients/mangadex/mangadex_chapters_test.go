package mangadex

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func testClient(url string) *MangaDexClient {
	return &MangaDexClient{config: MangaDexConfig{BaseURL: url}, http: &http.Client{Timeout: 5 * time.Second}}
}

// chapterJSON builds one feed entry. pages 0 or an external URL make it unhostable
func chapterJSON(id, number string, pages int, external bool) string {
	ext := "null"
	if external {
		ext = `"https://mangaplus.shueisha.co.jp/viewer/1"`
	}
	num := "null"
	if number != "" {
		num = strconv.Quote(number)
	}
	return fmt.Sprintf(`{"id":%q,"attributes":{"chapter":%s,"title":"T%s","volume":"1","pages":%d,"externalUrl":%s}}`,
		id, num, number, pages, ext)
}

func TestChaptersPagesThroughTheFeedInOneLanguage(t *testing.T) {
	var offsets []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/manga/m1/feed" || q.Get("translatedLanguage[]") != "en" {
			http.Error(w, "unexpected request "+r.URL.String(), http.StatusBadRequest)
			return
		}
		offsets = append(offsets, q.Get("offset"))
		// two pages: the bug in mangal stopped after the first one
		if q.Get("offset") == "0" {
			fmt.Fprintf(w, `{"total":501,"data":[%s,%s,%s,%s]}`,
				chapterJSON("c1", "1", 20, false),
				chapterJSON("c1b", "1", 18, false), // a second group's chapter 1
				chapterJSON("c2", "2", 0, false),   // no pages
				chapterJSON("c3", "3", 0, true),    // licensed, links out
			)
			return
		}
		fmt.Fprintf(w, `{"total":501,"data":[%s,%s,%s]}`,
			chapterJSON("c10", "10", 22, false),
			chapterJSON("c4", "4.5", 9, false),
			chapterJSON("os", "", 12, false),
		)
	}))
	defer srv.Close()

	chapters, err := testClient(srv.URL).Chapters("m1", "en")
	if err != nil {
		t.Fatalf("Chapters: %v", err)
	}
	if len(offsets) != 2 || offsets[1] != "500" {
		t.Errorf("offsets requested = %v, want [0 500]", offsets)
	}

	var got []string
	for _, ch := range chapters {
		got = append(got, ch.ID)
	}
	want := []string{"os", "c1", "c4", "c10"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("chapters = %v, want %v (oneshot first, decimals in order, first group kept, unhostable dropped)", got, want)
	}
}

func TestPageURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/at-home/server/c1" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{"baseUrl":"https://img.example","chapter":{"hash":"h1","data":["1-a.png","2-b.jpg"]}}`))
	}))
	defer srv.Close()

	urls, err := testClient(srv.URL).PageURLs("c1")
	if err != nil {
		t.Fatalf("PageURLs: %v", err)
	}
	want := []string{"https://img.example/data/h1/1-a.png", "https://img.example/data/h1/2-b.jpg"}
	if fmt.Sprint(urls) != fmt.Sprint(want) {
		t.Errorf("urls = %v, want %v", urls, want)
	}
}

func TestErrorsComeBack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	if _, err := c.Chapters("m1", "en"); err == nil {
		t.Error("Chapters should fail on a 429")
	}
	if _, err := c.PageURLs("c1"); err == nil {
		t.Error("PageURLs should fail on a 429")
	}
	if _, err := c.Image(srv.URL + "/x.png"); err == nil {
		t.Error("Image should fail on a 429")
	}
}
