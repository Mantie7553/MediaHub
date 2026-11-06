package suwayomi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MaxPageBytes caps one page image; real pages are a few MB, anything bigger is refused
const MaxPageBytes = 20 << 20

// largest GraphQL answer read, a long chapter list is well under this
const maxResponseBytes = 16 << 20

// ErrNotConfigured: the backend has no Suwayomi address, so it isn't used
var ErrNotConfigured = errors.New("suwayomi is not set up (SUWAYOMI_URL is empty)")

/*
Client: talks to the Suwayomi server on the private compose network. Everything it returns
comes from third-party extensions, so callers treat it as untrusted input
*/
type Client struct {
	baseURL  string
	user     string
	password string
	http     *http.Client
	images   *http.Client
}

/*
Function:	NewClient
Purpose:	A client for the Suwayomi server named by SUWAYOMI_URL, logging in with
SUWAYOMI_USER and SUWAYOMI_PASSWORD
*/
func NewClient() *Client {
	return &Client{
		baseURL:  strings.TrimRight(os.Getenv("SUWAYOMI_URL"), "/"),
		user:     os.Getenv("SUWAYOMI_USER"),
		password: os.Getenv("SUWAYOMI_PASSWORD"),
		// listing chapters makes Suwayomi ask the source site, which can be slow
		http: &http.Client{Timeout: 60 * time.Second},
		// a page is fetched from the source through Suwayomi
		images: &http.Client{Timeout: 90 * time.Second},
	}
}

// Configured: whether a Suwayomi server is set up
func (c *Client) Configured() bool { return c.baseURL != "" }

/*
Source: one installed extension source, like "MangaFire (EN)"
*/
type Source struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Lang string `json:"lang"`
}

/*
Manga: a search result from one source. ID is Suwayomi's id, stable once it has seen the title
*/
type Manga struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
}

/*
Chapter: one chapter a source has. Number is as Suwayomi parsed it from the source
*/
type Chapter struct {
	ID     int
	Number float64
	Title  string
}

/*
Function:	Sources
Purpose:	The installed sources that can be searched for a language ("en"). Sources for every
language ("all") are kept too, and Suwayomi's built-in local source is left out
*/
func (c *Client) Sources(lang string) ([]Source, error) {
	var out struct {
		Sources struct {
			Nodes []struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
				Lang        string `json:"lang"`
			} `json:"nodes"`
		} `json:"sources"`
	}
	if err := c.graphql(`query { sources { nodes { id displayName lang } } }`, nil, &out); err != nil {
		return nil, err
	}
	var sources []Source
	for _, s := range out.Sources.Nodes {
		if s.ID == "0" || (s.Lang != lang && s.Lang != "all") {
			continue
		}
		sources = append(sources, Source{ID: s.ID, Name: s.DisplayName, Lang: s.Lang})
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Name < sources[j].Name })
	return sources, nil
}

/*
Function:	Search
Purpose:	The first page of a source's search results for a title
*/
func (c *Client) Search(source Source, query string) ([]Manga, error) {
	var out struct {
		FetchSourceManga struct {
			Mangas []struct {
				ID    int    `json:"id"`
				Title string `json:"title"`
			} `json:"mangas"`
		} `json:"fetchSourceManga"`
	}
	err := c.graphql(
		`mutation($source: LongString!, $query: String) {
			fetchSourceManga(input: {source: $source, type: SEARCH, page: 1, query: $query}) {
				mangas { id title }
			}
		}`,
		map[string]any{"source": source.ID, "query": query}, &out)
	if err != nil {
		return nil, err
	}
	results := make([]Manga, 0, len(out.FetchSourceManga.Mangas))
	for _, m := range out.FetchSourceManga.Mangas {
		results = append(results, Manga{ID: m.ID, Title: m.Title, SourceID: source.ID, SourceName: source.Name})
	}
	return results, nil
}

/*
Function:	MangaByID
Purpose:	One title Suwayomi has seen, with the source it comes from
*/
func (c *Client) MangaByID(id int) (Manga, error) {
	var out struct {
		Manga struct {
			ID       int    `json:"id"`
			Title    string `json:"title"`
			SourceID string `json:"sourceId"`
		} `json:"manga"`
	}
	if err := c.graphql(`query($id: Int!) { manga(id: $id) { id title sourceId } }`, map[string]any{"id": id}, &out); err != nil {
		return Manga{}, err
	}
	if out.Manga.ID != id {
		return Manga{}, fmt.Errorf("suwayomi has no manga %d", id)
	}
	return Manga{ID: out.Manga.ID, Title: clip(out.Manga.Title, 300), SourceID: out.Manga.SourceID}, nil
}

/*
Function:	Chapters
Purpose:	A manga's chapters, fresh from its source, one per chapter number in reading order.
Chapters the source gives no number for are left out, they can't be matched to the library
*/
func (c *Client) Chapters(mangaID int) ([]Chapter, error) {
	var out struct {
		FetchChapters struct {
			Chapters []struct {
				ID            int     `json:"id"`
				Name          string  `json:"name"`
				ChapterNumber float64 `json:"chapterNumber"`
			} `json:"chapters"`
		} `json:"fetchChapters"`
	}
	err := c.graphql(
		`mutation($id: Int!) { fetchChapters(input: {mangaId: $id}) { chapters { id name chapterNumber } } }`,
		map[string]any{"id": mangaID}, &out)
	if err != nil {
		return nil, err
	}

	byNumber := map[float64]Chapter{}
	for _, ch := range out.FetchChapters.Chapters {
		if ch.ChapterNumber < 0 {
			continue
		}
		if _, seen := byNumber[ch.ChapterNumber]; !seen {
			byNumber[ch.ChapterNumber] = Chapter{ID: ch.ID, Number: ch.ChapterNumber, Title: clip(chapterTitle(ch.Name), maxTitleRunes)}
		}
	}
	chapters := make([]Chapter, 0, len(byNumber))
	for _, ch := range byNumber {
		chapters = append(chapters, ch)
	}
	sort.Slice(chapters, func(i, j int) bool { return chapters[i].Number < chapters[j].Number })
	return chapters, nil
}

// longest chapter title kept, it ends up in a file name
const maxTitleRunes = 100

// clip shortens s to at most n characters
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n]))
}

/*
Function:	chapterTitle
Purpose:	The chapter's own title from a source's name for it. Sources mostly write
"Chapter 12" or "Chapter 12: The Title", and only "The Title" is worth keeping
*/
func chapterTitle(name string) string {
	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)
	for _, prefix := range []string{"chapter ", "ch. ", "ch "} {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		rest := name[len(prefix):]
		// skip the number
		i := 0
		for i < len(rest) && (rest[i] == '.' || (rest[i] >= '0' && rest[i] <= '9')) {
			i++
		}
		if i == 0 {
			return name
		}
		return strings.TrimSpace(strings.TrimLeft(rest[i:], " :-–—"))
	}
	return name
}

/*
Function:	PageURLs
Purpose:	The image URLs of a chapter's pages, in order. They point at Suwayomi, which fetches
each page from the source
*/
func (c *Client) PageURLs(chapterID string) ([]string, error) {
	id, err := strconv.Atoi(chapterID)
	if err != nil {
		return nil, fmt.Errorf("bad suwayomi chapter id %q", chapterID)
	}
	var out struct {
		FetchChapterPages struct {
			Pages []string `json:"pages"`
		} `json:"fetchChapterPages"`
	}
	err = c.graphql(
		`mutation($id: Int!) { fetchChapterPages(input: {chapterId: $id}) { pages } }`,
		map[string]any{"id": id}, &out)
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(out.FetchChapterPages.Pages))
	for _, p := range out.FetchChapterPages.Pages {
		u, err := c.pageURL(p)
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}

/*
Function:	pageURL
Purpose:	A page path from Suwayomi as a full URL on the Suwayomi server. Anything pointing
elsewhere is refused, so the login is only ever sent to Suwayomi
*/
func (c *Client) pageURL(p string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(p)
	if err != nil {
		return "", fmt.Errorf("bad page url from suwayomi")
	}
	u := base.ResolveReference(ref)
	if u.Scheme != base.Scheme || u.Host != base.Host {
		return "", fmt.Errorf("suwayomi gave a page url on another host")
	}
	return u.String(), nil
}

/*
Function:	Image
Purpose:	Download one page image through Suwayomi
Params:
  - imageURL: from PageURLs
*/
func (c *Client) Image(imageURL string) ([]byte, error) {
	if _, err := c.pageURL(imageURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.images.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("suwayomi returned %d for a page", resp.StatusCode)
	}

	img, err := io.ReadAll(io.LimitReader(resp.Body, MaxPageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(img) > MaxPageBytes {
		return nil, fmt.Errorf("page image is over %d MB", MaxPageBytes>>20)
	}
	return img, nil
}

/*
Function:	graphql
Purpose:	Run a query or mutation against Suwayomi's GraphQL API and decode its data into out
*/
func (c *Client) graphql(query string, variables map[string]any, out any) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.baseURL+"/api/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("suwayomi refused the login, check SUWAYOMI_PASSWORD")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("suwayomi returned %d", resp.StatusCode)
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&envelope); err != nil {
		return fmt.Errorf("unreadable answer from suwayomi: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("suwayomi: %s", envelope.Errors[0].Message)
	}
	return json.Unmarshal(envelope.Data, out)
}
