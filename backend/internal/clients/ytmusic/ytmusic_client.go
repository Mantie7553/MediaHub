package ytmusic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// the search filters YouTube Music's own web app sends for its Songs and Videos tabs
const (
	FilterSongs  = "EgWKAQIIAWoKEAkQBRAKEAMQBA=="
	FilterVideos = "EgWKAQIQAWoKEAkQChAFEAMQBA=="
)

// the web app the requests present as; YouTube keeps accepting a version for a long while
const (
	clientName    = "WEB_REMIX"
	clientVersion = "1.20251110.01.00"
)

/*
BaseURL: YouTube Music's internal API, without a trailing slash
*/
type YTMusicConfig struct {
	BaseURL string
}

/*
config: BaseURL for the API
http: pointer to the http client
*/
type YTMusicClient struct {
	config YTMusicConfig
	http   *http.Client
}

/*
Function:	NewYTMusicClient
Purpose:	Build a client for YouTube Music's search, the API its web app uses. It isn't a
published API, so callers should fall back to yt-dlp when it fails
*/
func NewYTMusicClient() *YTMusicClient {
	return &YTMusicClient{
		config: YTMusicConfig{BaseURL: "https://music.youtube.com/youtubei/v1"},
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

/*
Function:	Search
Purpose:	Search YouTube Music, returning songs or videos with their artists, album, explicit
badge and whether they are the label's own upload
Params:
  - query: free-text search term
  - filter: FilterSongs or FilterVideos
  - limit: the most results to return
*/
func (c *YTMusicClient) Search(query, filter string, limit int) ([]Track, error) {
	body, err := json.Marshal(map[string]any{
		"context": map[string]any{
			"client": map[string]string{
				"clientName":    clientName,
				"clientVersion": clientVersion,
				"hl":            "en",
				"gl":            "US",
			},
		},
		"query":  query,
		"params": filter,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.config.BaseURL+"/search?prettyPrint=false", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://music.youtube.com")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube music search: status %d", res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	kind := KindSong
	if filter == FilterVideos {
		kind = KindVideo
	}
	return parseSearch(raw, kind, limit)
}

/*
Function:	parseSearch
Purpose:	pull the tracks out of a search response. The list items are found wherever they sit,
so YouTube moving the shelves around them doesn't break the search
Params:
  - raw: the response body
  - kind: KindSong or KindVideo, what the search asked for
  - limit: the most tracks to return; 0 or less for all
*/
func parseSearch(raw []byte, kind string, limit int) ([]Track, error) {
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	var items []json.RawMessage
	collectItems(doc, &items)

	tracks := []Track{}
	for _, itemRaw := range items {
		var item listItem
		if err := json.Unmarshal(itemRaw, &item); err != nil {
			continue
		}
		t, ok := toTrack(item, itemRaw, kind)
		// very short clips aren't songs
		if !ok || t.DurationSecs <= 30 {
			continue
		}
		tracks = append(tracks, t)
		if limit > 0 && len(tracks) == limit {
			break
		}
	}
	return tracks, nil
}

// collectItems appends every musicResponsiveListItemRenderer under o, in document order
func collectItems(o any, out *[]json.RawMessage) {
	switch v := o.(type) {
	case map[string]any:
		if item, ok := v["musicResponsiveListItemRenderer"]; ok {
			if b, err := json.Marshal(item); err == nil {
				*out = append(*out, b)
			}
			return
		}
		// map order is random; sort keys so results keep the response's order
		for _, k := range sortedKeys(v) {
			collectItems(v[k], out)
		}
	case []any:
		for _, e := range v {
			collectItems(e, out)
		}
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var (
	videoTypeRe = regexp.MustCompile(`"musicVideoType":"(MUSIC_VIDEO_TYPE_[A-Z_]+)"`)
	durationRe  = regexp.MustCompile(`^\d+(:\d{2}){1,2}$`)
	// song thumbnails are sized in the URL, "...=w60-h60-l90-rj"; ask for a bigger one
	thumbSizeRe = regexp.MustCompile(`=w\d+-h\d+`)
)

// the bullet YouTube Music puts between the parts of a subtitle
const bullet = " • "

/*
Function:	toTrack
Purpose:	read one list item. Column 0 is the title; column 1 is "Artists • Album • 3:45" for a
song or "Channel • 1.2M views • 3:45" for a video
*/
func toTrack(item listItem, raw []byte, kind string) (Track, bool) {
	id := item.PlaylistItemData.VideoID
	if id == "" || len(item.FlexColumns) < 2 {
		return Track{}, false
	}
	title := joinRuns(item.FlexColumns[0].Column.Text.Runs)
	if title == "" {
		return Track{}, false
	}

	t := Track{
		ID:    id,
		Title: title,
		Kind:  kind,
		URL:   "https://www.youtube.com/watch?v=" + id,
	}

	parts := splitRuns(item.FlexColumns[1].Column.Text.Runs)
	// unfiltered searches put the kind first ("Song • ..."); drop it if it's there
	if len(parts) > 0 {
		if first := strings.ToLower(joinRuns(parts[0])); first == "song" || first == "video" {
			parts = parts[1:]
		}
	}
	if len(parts) > 0 {
		t.Artists = artistNames(parts[0])
		t.Artist = joinRuns(parts[0])
		t.Uploader = t.Artist
		if len(t.Artists) > 0 {
			t.Uploader = t.Artists[0]
		}
		if last := joinRuns(parts[len(parts)-1]); durationRe.MatchString(last) {
			t.DurationSecs = parseDuration(last)
		}
	}
	if kind == KindSong {
		t.Album = albumName(parts)
	}

	for _, b := range item.Badges {
		if b.Badge.Icon.IconType == "MUSIC_EXPLICIT_BADGE" {
			t.Explicit = true
		}
	}

	if m := videoTypeRe.FindSubmatch(raw); m != nil {
		switch string(m[1]) {
		case "MUSIC_VIDEO_TYPE_ATV", "MUSIC_VIDEO_TYPE_OMV", "MUSIC_VIDEO_TYPE_OFFICIAL_SOURCE_MUSIC":
			t.Official = true
		}
	}

	t.Thumbnail = bestThumbnail(item)
	if t.Thumbnail == "" {
		t.Thumbnail = "https://img.youtube.com/vi/" + id + "/mqdefault.jpg"
	}
	return t, true
}

// albumName is the part linking to an album (browse id MPRE...), else the middle part of a
// three part song subtitle
func albumName(parts [][]run) string {
	for _, p := range parts {
		for _, r := range p {
			if strings.HasPrefix(r.browseID(), "MPRE") {
				return strings.TrimSpace(r.Text)
			}
		}
	}
	if len(parts) == 3 {
		return joinRuns(parts[1])
	}
	return ""
}

// splitRuns groups a column's runs into the parts between bullets
func splitRuns(runs []run) [][]run {
	var parts [][]run
	var cur []run
	for _, r := range runs {
		if strings.TrimSpace(r.Text) == strings.TrimSpace(bullet) {
			if len(cur) > 0 {
				parts = append(parts, cur)
			}
			cur = nil
			continue
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		parts = append(parts, cur)
	}
	return parts
}

// artistNames are the names in an artists part, without the ", " and " & " between them
func artistNames(runs []run) []string {
	names := []string{}
	for _, r := range runs {
		s := strings.TrimSpace(r.Text)
		if s == "" || s == "," || s == "&" {
			continue
		}
		names = append(names, s)
	}
	return names
}

func joinRuns(runs []run) string {
	var b strings.Builder
	for _, r := range runs {
		b.WriteString(r.Text)
	}
	return strings.TrimSpace(b.String())
}

// parseDuration reads "3:45" or "1:02:03" as seconds
func parseDuration(s string) int {
	total := 0
	for _, p := range strings.Split(s, ":") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

func bestThumbnail(item listItem) string {
	best, width := "", -1
	for _, th := range item.Thumbnail.Renderer.Thumbnail.Thumbnails {
		if th.Width > width {
			best, width = th.URL, th.Width
		}
	}
	return thumbSizeRe.ReplaceAllString(best, "=w544-h544")
}
