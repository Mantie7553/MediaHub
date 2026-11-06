package suwayomi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeServer answers GraphQL with the reply for the first operation name found in the query
func fakeServer(t *testing.T, replies map[string]string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "admin" || p != "pw" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/api/graphql" {
			w.Write([]byte("\x89PNG\r\n\x1a\nimage"))
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req struct{ Query string }
		json.Unmarshal(body, &req)
		for op, reply := range replies {
			if strings.Contains(req.Query, op) {
				w.Write([]byte(reply))
				return
			}
		}
		t.Errorf("unexpected query %s", req.Query)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("SUWAYOMI_URL", srv.URL)
	t.Setenv("SUWAYOMI_USER", "admin")
	t.Setenv("SUWAYOMI_PASSWORD", "pw")
	return NewClient()
}

func TestSourcesKeepsTheLanguageAndSkipsLocal(t *testing.T) {
	c := fakeServer(t, map[string]string{"sources": `{"data":{"sources":{"nodes":[
		{"id":"0","displayName":"Local source","lang":"localsourcelang"},
		{"id":"2","displayName":"MangaFire (EN)","lang":"en"},
		{"id":"3","displayName":"MangaFire (JA)","lang":"ja"},
		{"id":"4","displayName":"Comick","lang":"all"}]}}}`})
	got, err := c.Sources("en")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Comick" || got[1].ID != "2" {
		t.Errorf("sources = %+v", got)
	}
}

func TestChaptersOnePerNumberInOrder(t *testing.T) {
	c := fakeServer(t, map[string]string{"fetchChapters": `{"data":{"fetchChapters":{"chapters":[
		{"id":12,"name":"Chapter 2: The Return","chapterNumber":2},
		{"id":11,"name":"Chapter 1","chapterNumber":1},
		{"id":13,"name":"Chapter 2 (other group)","chapterNumber":2},
		{"id":14,"name":"Announcement","chapterNumber":-1},
		{"id":15,"name":"Ch. 1.5 - Extra","chapterNumber":1.5}]}}}`})
	got, err := c.Chapters(7)
	if err != nil {
		t.Fatal(err)
	}
	want := []Chapter{{11, 1, ""}, {15, 1.5, "Extra"}, {12, 2, "The Return"}}
	if len(got) != len(want) {
		t.Fatalf("chapters = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("chapter %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPageURLsStayOnSuwayomi(t *testing.T) {
	c := fakeServer(t, map[string]string{"fetchChapterPages": `{"data":{"fetchChapterPages":{"pages":[
		"/api/v1/manga/7/chapter/1/page/0","/api/v1/manga/7/chapter/1/page/1"]}}}`})
	urls, err := c.PageURLs("31")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 2 || !strings.HasPrefix(urls[0], c.baseURL+"/api/v1/") {
		t.Fatalf("urls = %v", urls)
	}
	img, err := c.Image(urls[0])
	if err != nil || !strings.HasPrefix(string(img), "\x89PNG") {
		t.Errorf("image = %q, %v", img, err)
	}

	// the login must never go to another host
	for _, bad := range []string{"https://evil.example/x.png", "//evil.example/x.png"} {
		if _, err := c.pageURL(bad); err == nil {
			t.Errorf("pageURL(%q) should be refused", bad)
		}
		if _, err := c.Image(bad); err == nil {
			t.Errorf("Image(%q) should be refused", bad)
		}
	}
}

func TestGraphQLErrorsAndLogin(t *testing.T) {
	c := fakeServer(t, map[string]string{"fetchChapters": `{"data":null,"errors":[{"message":"source is down"}]}`})
	if _, err := c.Chapters(1); err == nil || !strings.Contains(err.Error(), "source is down") {
		t.Errorf("err = %v, want the GraphQL error", err)
	}
	c.password = "wrong"
	if _, err := c.Chapters(1); err == nil || !strings.Contains(err.Error(), "login") {
		t.Errorf("err = %v, want a login error", err)
	}
}

func TestNotConfigured(t *testing.T) {
	t.Setenv("SUWAYOMI_URL", "")
	if _, err := NewClient().Sources("en"); err != ErrNotConfigured {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}

func TestChapterTitle(t *testing.T) {
	cases := map[string]string{
		"Chapter 12":            "",
		"Chapter 12: The Title": "The Title",
		"Ch. 3.5 - Side Story":  "Side Story",
		"Vol.1 Chapter 2":       "Vol.1 Chapter 2",
		"Chapter Final":         "Chapter Final",
		"  chapter 7 — Dawn  ":  "Dawn",
	}
	for in, want := range cases {
		if got := chapterTitle(in); got != want {
			t.Errorf("chapterTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMangaByID(t *testing.T) {
	c := fakeServer(t, map[string]string{"manga(": `{"data":{"manga":{"id":7,"title":"Frieren","sourceId":"2"}}}`})
	m, err := c.MangaByID(7)
	if err != nil || m.Title != "Frieren" || m.SourceID != "2" {
		t.Errorf("manga = %+v, %v", m, err)
	}
	if _, err := c.MangaByID(8); err == nil {
		t.Error("a different id in the answer should be refused")
	}
}
