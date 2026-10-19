package mangadex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// MangaDex asks API clients to identify themselves
const userAgent = "MediaHub (self-hosted media server)"

// the feed endpoint's largest page
const feedPageSize = 500

/*
Chapter: one chapter MangaDex hosts the pages for
  - Number: as MangaDex writes it ("12", "52.5"), empty for a oneshot
  - Title: the chapter's own title, often empty
*/
type Chapter struct {
	ID     string
	Number string
	Title  string
	Volume string
	Pages  int
}

/*
Function:	NumberValue
Purpose:	The chapter number for sorting and storing, a oneshot counts as 0
*/
func (ch Chapter) NumberValue() float64 {
	n, err := strconv.ParseFloat(ch.Number, 64)
	if err != nil {
		return 0
	}
	return n
}

type feedChapter struct {
	ID         string `json:"id"`
	Attributes struct {
		Chapter     *string `json:"chapter"`
		Title       *string `json:"title"`
		Volume      *string `json:"volume"`
		Pages       int     `json:"pages"`
		ExternalURL *string `json:"externalUrl"`
	} `json:"attributes"`
}

/*
Function:	Chapters
Purpose:	List the chapters of a manga whose pages MangaDex hosts in one language, one per
chapter number, in reading order. Chapters that only link to another site (licensed
releases) have no pages here and are left out

Params:
  - mangaID: MangaDex's id for the manga
  - language: a MangaDex language code, like "en"
*/
func (c *MangaDexClient) Chapters(mangaID string, language string) ([]Chapter, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(feedPageSize))
	params.Add("translatedLanguage[]", language)
	// the title was picked by the user, so don't let a rating filter hide its chapters
	for _, rating := range []string{"safe", "suggestive", "erotica", "pornographic"} {
		params.Add("contentRating[]", rating)
	}
	params.Set("includeExternalUrl", "0")
	params.Set("includeEmptyPages", "0")
	params.Set("includeFuturePublishAt", "0")
	params.Set("order[chapter]", "asc")

	// several groups may have translated the same chapter, the first one listed is kept
	byNumber := map[string]Chapter{}
	for offset := 0; ; offset += feedPageSize {
		params.Set("offset", strconv.Itoa(offset))
		var page struct {
			Data  []feedChapter `json:"data"`
			Total int           `json:"total"`
		}
		if err := c.getJSON("/manga/"+mangaID+"/feed?"+params.Encode(), &page); err != nil {
			return nil, err
		}

		for _, fc := range page.Data {
			a := fc.Attributes
			if a.ExternalURL != nil || a.Pages == 0 {
				continue
			}
			ch := Chapter{ID: fc.ID, Pages: a.Pages}
			if a.Chapter != nil {
				ch.Number = *a.Chapter
			}
			if a.Title != nil {
				ch.Title = *a.Title
			}
			if a.Volume != nil {
				ch.Volume = *a.Volume
			}
			if _, seen := byNumber[ch.Number]; !seen {
				byNumber[ch.Number] = ch
			}
		}

		if len(page.Data) == 0 || offset+feedPageSize >= page.Total {
			break
		}
	}

	chapters := make([]Chapter, 0, len(byNumber))
	for _, ch := range byNumber {
		chapters = append(chapters, ch)
	}
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].NumberValue() < chapters[j].NumberValue()
	})
	return chapters, nil
}

/*
Function:	PageURLs
Purpose:	The image URLs of a chapter's pages, in order, from the MangaDex@Home server
assigned to it. The URLs stop working after about 15 minutes

Params:
  - chapterID: MangaDex's id for the chapter
*/
func (c *MangaDexClient) PageURLs(chapterID string) ([]string, error) {
	var server struct {
		BaseURL string `json:"baseUrl"`
		Chapter struct {
			Hash string   `json:"hash"`
			Data []string `json:"data"`
		} `json:"chapter"`
	}
	if err := c.getJSON("/at-home/server/"+chapterID, &server); err != nil {
		return nil, err
	}
	if server.BaseURL == "" || server.Chapter.Hash == "" {
		return nil, fmt.Errorf("mangadex gave no image server for chapter %s", chapterID)
	}

	urls := make([]string, len(server.Chapter.Data))
	for i, file := range server.Chapter.Data {
		urls[i] = fmt.Sprintf("%s/data/%s/%s", server.BaseURL, server.Chapter.Hash, file)
	}
	return urls, nil
}

/*
Function:	Image
Purpose:	Download one page image
Params:
  - imageURL: from PageURLs
*/
func (c *MangaDexClient) Image(imageURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	// images are larger and come from volunteer servers, so they get longer than API calls
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("image server returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

/*
Function:	getJSON
Purpose:	GET an API path and decode the JSON answer into out
*/
func (c *MangaDexClient) getJSON(path string, out any) error {
	req, err := http.NewRequest("GET", c.config.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("mangadex returned %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
