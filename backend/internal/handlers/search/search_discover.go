package search

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/ytdlp"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// the media types Discover lists, as the API names them
var discoverTypes = []string{"anime", "movie", "manga"}

// Discover's Trending, Popular and Top Rated (Latest Updates for manga) lists
type discoveryResponse struct {
	Trending []SearchResult `json:"trending"`
	Popular  []SearchResult `json:"popular"`
	TopRated []SearchResult `json:"top_rated"`
}

// the lists on Discover change over hours, so they are kept for an hour and, past that, still
// shown for up to a day while they're fetched again in the background
func newDiscoveryCache() *cache.Memo[discoveryResponse] {
	return cache.NewMemo[discoveryResponse](time.Hour, 24*time.Hour, len(discoverTypes))
}

// text searches, kept for a while so going back to a search or repeating it is instant
func newSearchCache() *cache.Memo[[]SearchResult] {
	return cache.NewMemo[[]SearchResult](15*time.Minute, 15*time.Minute, 300)
}

// YouTube results for a query barely change within the hour
func newMusicSearchCache() *cache.Memo[[]ytdlp.SearchResult] {
	return cache.NewMemo[[]ytdlp.SearchResult](time.Hour, time.Hour, 200)
}

/*
Function:	searchKey
Purpose:	a cache key for a search, ignoring the query's case and extra spaces
*/
func searchKey(scope, query string) string {
	return scope + ":" + strings.ToLower(strings.Join(strings.Fields(query), " "))
}

/*
Function:	Search
Purpose:	Search Anilist or MangaDex for media of a given type, or with no query, the lists
Discover shows for that type
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the front end, with ?type= and an optional ?q=
*/
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	mediaType := r.URL.Query().Get("type")
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	known := false
	for _, t := range discoverTypes {
		known = known || t == mediaType
	}
	if !known {
		utils.Error(w, http.StatusBadRequest, "type must be anime, movie or manga")
		return
	}

	if q == "" {
		discovery, err := h.discovery.Get(mediaType, func() (discoveryResponse, error) {
			return loadDiscovery(mediaType)
		})
		if err != nil {
			logger.Error("discovery for %s: %s", mediaType, err.Error())
			utils.Error(w, http.StatusInternalServerError, "search failed")
			return
		}
		utils.JSON(w, discovery)
		return
	}

	results, err := h.searches.Get(searchKey(mediaType, q), func() ([]SearchResult, error) {
		return loadSearch(mediaType, q)
	})
	if err != nil {
		logger.Error("search for %s %q: %s", mediaType, q, err.Error())
		utils.Error(w, http.StatusInternalServerError, "search failed")
		return
	}
	utils.JSON(w, results)
}

/*
Function:	WarmDiscovery
Purpose:	fetch Discover's lists in the background at startup, so the first visit is instant
*/
func (h *Handler) WarmDiscovery() {
	for _, t := range discoverTypes {
		h.discovery.Warm(t, func() (discoveryResponse, error) {
			d, err := loadDiscovery(t)
			if err != nil {
				// the first visit tries again
				logger.Warn("loading Discover's %s lists ahead: %s", t, err.Error())
			}
			return d, err
		})
	}
}

// Discover's lists for a media type from AniList or MangaDex
func loadDiscovery(mediaType string) (discoveryResponse, error) {
	if mediaType == "manga" {
		d, err := mangadex.NewMangaDexClient("").Discovery(100)
		if err != nil {
			return discoveryResponse{}, err
		}
		return discoveryResponse{
			Trending: fromMangaDex(d.Trending),
			Popular:  fromMangaDex(d.Popular),
			TopRated: fromMangaDex(d.Latest),
		}, nil
	}

	d, err := anilist.NewAnilistClient("").Discovery("ANIME", anilistFormat(mediaType), 100)
	if err != nil {
		return discoveryResponse{}, err
	}
	return discoveryResponse{
		Trending: fromAnilist(d.Trending, mediaType),
		Popular:  fromAnilist(d.Popular, mediaType),
		TopRated: fromAnilist(d.TopRated, mediaType),
	}, nil
}

// search results for a query from AniList or MangaDex
func loadSearch(mediaType, q string) ([]SearchResult, error) {
	if mediaType == "manga" {
		results, err := mangadex.NewMangaDexClient("").Search(q)
		if err != nil {
			return nil, err
		}
		return fromMangaDex(results), nil
	}

	results, err := anilist.NewAnilistClient("").Search("ANIME", q, 100, anilistFormat(mediaType))
	if err != nil {
		return nil, err
	}
	return fromAnilist(results, mediaType), nil
}

// AniList's format for a media type: anime are TV series, movies are MOVIE
func anilistFormat(mediaType string) string {
	if mediaType == "movie" {
		return "MOVIE"
	}
	return "TV"
}

// AniList media as results, titled in English when AniList has an English title
func fromAnilist(media []anilist.Media, mediaType string) []SearchResult {
	out := make([]SearchResult, 0, len(media))
	for _, m := range media {
		title := m.Title.English
		if title == "" {
			title = m.Title.Romaji
		}
		out = append(out, SearchResult{
			ExternalID:     strconv.Itoa(m.ID),
			ExternalSource: "anilist",
			Title:          title,
			CoverImageURL:  m.CoverImage.Large,
			Type:           mediaType,
		})
	}
	return out
}

// MangaDex manga as results. Covers use MangaDex's 512px thumbnail rather than the original
// upload, which is often several MB and far larger than any card or page shows it
func fromMangaDex(manga []mangadex.Manga) []SearchResult {
	out := make([]SearchResult, 0, len(manga))
	for _, m := range manga {
		var fileName string
		for _, rel := range m.Relationships {
			if rel.Type == "cover_art" {
				fileName = rel.Attributes.FileName
				break
			}
		}
		var coverURL string
		if fileName != "" {
			coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.512.jpg", m.ID, fileName)
		}
		title := m.Attributes.Title.En
		if title == "" {
			title = m.Attributes.Title.JaRo
		}
		if title == "" {
			title = m.Attributes.Title.Ja
		}
		out = append(out, SearchResult{
			ExternalID:     m.ID,
			ExternalSource: "mangadex",
			Title:          title,
			CoverImageURL:  coverURL,
			Type:           "manga",
		})
	}
	return out
}
