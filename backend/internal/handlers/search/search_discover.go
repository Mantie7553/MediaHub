package search

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
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

// YouTube Music results for a query barely change within the hour
func newMusicSearchCache() *cache.Memo[musicSearchResponse] {
	return cache.NewMemo[musicSearchResponse](time.Hour, time.Hour, 200)
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

	if mediaType == "anime" {
		return loadAnimeDiscovery(anilist.NewAnilistClient("").DiscoveryPage, arr.AnimeShows())
	}
	d, err := anilist.NewAnilistClient("").Discovery("ANIME", anilistFormat(mediaType), 50)
	if err != nil {
		return discoveryResponse{}, err
	}
	return discoveryResponse{
		Trending: fromAnilist(d.Trending, mediaType),
		Popular:  fromAnilist(d.Popular, mediaType),
		TopRated: fromAnilist(d.TopRated, mediaType),
	}, nil
}

// how many series each anime Discover list aims for, and how many of AniList's pages of 50 it
// reads to get there
const (
	discoverTarget   = 50
	discoverMaxPages = 4
)

// a page of AniList's Discover lists: anilist.AnilistClient.DiscoveryPage, or a stand-in in tests
type discoveryPager func(mediaType, format string, perPage, page int) (*anilist.DiscoveryResult, error)

/*
Function:	loadAnimeDiscovery
Purpose:	the anime Discover lists, one result per series. AniList ranks each season on its own,
so a top rated list is often several seasons of a few shows; pages are read until every list
has discoverTarget series, or discoverMaxPages pages. A later page failing keeps what was read
Params:
  - page: reads one page of AniList's lists
  - shows: the series
*/
func loadAnimeDiscovery(page discoveryPager, shows seriesSource) (discoveryResponse, error) {
	var trending, popular, topRated []anilist.Media
	var out discoveryResponse
	for n := 1; n <= discoverMaxPages; n++ {
		d, err := page("ANIME", "TV", 50, n)
		if err != nil {
			if n == 1 {
				return discoveryResponse{}, err
			}
			logger.Warn("page %d of Discover's anime lists: %s", n, err.Error())
			break
		}
		trending = append(trending, d.Trending...)
		popular = append(popular, d.Popular...)
		topRated = append(topRated, d.TopRated...)

		lists := groupSeries(shows, fromAnilist(trending, "anime"), fromAnilist(popular, "anime"), fromAnilist(topRated, "anime"))
		out = discoveryResponse{Trending: lists[0], Popular: lists[1], TopRated: lists[2]}

		ended := len(d.Trending) < 50 && len(d.Popular) < 50 && len(d.TopRated) < 50
		if ended || (len(out.Trending) >= discoverTarget && len(out.Popular) >= discoverTarget && len(out.TopRated) >= discoverTarget) {
			break
		}
	}
	out.Trending = out.Trending[:min(len(out.Trending), discoverTarget)]
	out.Popular = out.Popular[:min(len(out.Popular), discoverTarget)]
	out.TopRated = out.TopRated[:min(len(out.TopRated), discoverTarget)]
	return out, nil
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
	out := fromAnilist(results, mediaType)
	if mediaType == "anime" {
		out = groupSeries(arr.AnimeShows(), out)[0]
	}
	return out, nil
}

// where a list's results' series come from: arr.Shows, or a stand-in in tests
type seriesSource interface {
	Maps() arr.AnimeMaps
	Load(tvdbIDs []int) map[int]arr.Show
}

/*
Function:	groupSeries
Purpose:	turn AniList's anime results, one per season or cour, into one per TVDB series the way
Sonarr keeps them: "Bakemonogatari" and "Nisemonogatari" become one "Monogatari", at the place
of whichever ranked highest. Grouping goes by Fribb's TVDB ids, never by name. Results Fribb
doesn't place stay as they are
Params:
  - shows: the series
  - lists: the result lists, grouped together so their series are looked up once
*/
func groupSeries(shows seriesSource, lists ...[]SearchResult) [][]SearchResult {
	maps := shows.Maps()
	var multi []int
	seen := map[int]bool{}
	for _, list := range lists {
		for _, r := range list {
			id, _ := strconv.Atoi(r.ExternalID)
			at, ok := maps.AnilistToShow[id]
			if ok && len(maps.ShowSeasons[at.TvdbID]) > 1 && !seen[at.TvdbID] {
				seen[at.TvdbID] = true
				multi = append(multi, at.TvdbID)
			}
		}
	}
	series := shows.Load(multi)

	out := make([][]SearchResult, len(lists))
	for i, list := range lists {
		placed := map[int]bool{}
		grouped := make([]SearchResult, 0, len(list))
		for _, r := range list {
			id, _ := strconv.Atoi(r.ExternalID)
			at, ok := maps.AnilistToShow[id]
			if !ok {
				grouped = append(grouped, r)
				continue
			}
			show, many := series[at.TvdbID]
			if !many {
				// a series of one entry is the whole series; one of several that couldn't be
				// looked up stays a season, saved as one
				if len(maps.ShowSeasons[at.TvdbID]) == 1 {
					r.TvdbID = at.TvdbID
				}
				grouped = append(grouped, r)
				continue
			}
			if placed[at.TvdbID] {
				continue
			}
			placed[at.TvdbID] = true
			grouped = append(grouped, seriesResult(r, show))
		}
		out[i] = grouped
	}
	return out
}

// a series as one result, taking its name, cover and episodes from the series
func seriesResult(r SearchResult, show arr.Show) SearchResult {
	r.ExternalID = strconv.Itoa(show.AnilistID)
	r.TvdbID = show.TvdbID
	if show.Title != "" {
		r.Title = show.Title
	}
	if show.CoverURL != "" {
		r.CoverImageURL = show.CoverURL
	}
	seasons := map[int]bool{}
	total := 0
	for _, e := range show.Entries {
		if e.Season != 0 {
			seasons[e.Season] = true
		}
		if total >= 0 && e.Episodes > 0 {
			total += e.Episodes
		} else {
			total = -1 // one entry's count unknown makes the series' unknown
		}
	}
	r.Seasons = len(seasons)
	r.TotalParts = nil
	if total > 0 {
		r.TotalParts = &total
	}
	return r
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
			TotalParts:     m.Episodes,
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
			TotalParts:     knownCount(m.TotalChapters()),
		})
	}
	return out
}

// a count the source gave, nil for 0 (MangaDex has no last chapter for most running series)
func knownCount(n int) *int {
	if n <= 0 {
		return nil
	}
	return &n
}
