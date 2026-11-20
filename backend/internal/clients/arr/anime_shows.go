package arr

import (
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

// Show is a TVDB series as one title, the way Sonarr keeps it, with the AniList entries
// (seasons, cours, specials) AniList and MyAnimeList split it into
type Show struct {
	TvdbID int
	// Sonarr's name for the series ("Monogatari"), else its first entry's
	Title    string
	CoverURL string
	// the entry the title is linked to: the first regular one
	AnilistID int
	// the entries in watch order, specials last, with their episode counts
	Entries []ShowEntryInfo
}

// ShowEntryInfo is one entry of a Show
type ShowEntryInfo struct {
	AnilistID int
	Season    int
	Episodes  int
}

// how long a series' entries and name are kept before asking again
const showTTL = 24 * time.Hour

// how many Sonarr lookups run at once; each goes out to Sonarr's metadata server
const lookupWorkers = 6

// Shows builds Shows from Fribb's list, AniList and Sonarr, and keeps them for a day
type Shows struct {
	mu     sync.Mutex
	cached map[int]cachedShow

	maps   func() (AnimeMaps, error)
	media  func(ids []int) (map[int]anilist.Media, error)
	lookup func(term string) (SonarrSeries, error)
	now    func() time.Time
}

type cachedShow struct {
	show Show
	at   time.Time
}

var sharedShows = newShows()

func newShows() *Shows {
	return &Shows{
		cached: map[int]cachedShow{},
		maps:   LoadAnimeMaps,
		media:  anilist.NewAnilistClient("").GetByIDs,
		lookup: func(term string) (SonarrSeries, error) {
			if os.Getenv("SONARR_URL") == "" {
				return SonarrSeries{}, errNoSonarr
			}
			return NewArrClient("SONARR_URL", "SONARR_API_KEY").LookupSeriesInfo(term)
		},
		now: time.Now,
	}
}

type noSonarr struct{}

func (noSonarr) Error() string { return "Sonarr is not set up" }

var errNoSonarr = noSonarr{}

// AnimeShows is the app's one Shows
func AnimeShows() *Shows { return sharedShows }

// Maps is Fribb's list as it stands, empty when it couldn't be fetched
func (s *Shows) Maps() AnimeMaps {
	maps, _ := s.maps()
	return maps
}

/*
Function:	Of
Purpose:	the series an AniList entry is part of
Returns:	false when Fribb doesn't place the entry in a series, or the series couldn't be built
*/
func (s *Shows) Of(anilistID int) (Show, TvdbSeason, bool) {
	maps, _ := s.maps()
	at, ok := maps.AnilistToShow[anilistID]
	if !ok {
		return Show{}, TvdbSeason{}, false
	}
	shows := s.Load([]int{at.TvdbID})
	show, ok := shows[at.TvdbID]
	return show, at, ok
}

/*
Function:	Load
Purpose:	many series at once: their entries' episode counts from AniList in as few requests as it
allows, and their names from Sonarr side by side. A series whose entries AniList couldn't give
is left out
Params:
  - tvdbIDs: the series
*/
func (s *Shows) Load(tvdbIDs []int) map[int]Show {
	out := map[int]Show{}
	var missing []int
	s.mu.Lock()
	for _, id := range tvdbIDs {
		if c, ok := s.cached[id]; ok && s.now().Sub(c.at) < showTTL {
			out[id] = c.show
		} else if _, dup := out[id]; !dup {
			missing = append(missing, id)
		}
	}
	s.mu.Unlock()
	if len(missing) == 0 {
		return out
	}

	maps, err := s.maps()
	if err != nil {
		logger.Warn("Fribb's anime list for series: %s", err.Error())
	}
	var ids []int
	seen := map[int]bool{}
	for _, tvdb := range missing {
		for _, e := range maps.ShowSeasons[tvdb] {
			if !seen[e.AnilistID] {
				seen[e.AnilistID] = true
				ids = append(ids, e.AnilistID)
			}
		}
	}
	media, err := s.media(ids)
	if err != nil {
		logger.Warn("AniList entries for %d series: %s", len(missing), err.Error())
	}

	built := map[int]Show{}
	for _, tvdb := range missing {
		if show, ok := buildShow(tvdb, maps.ShowSeasons[tvdb], media); ok {
			built[tvdb] = show
		}
	}
	s.name(built)

	s.mu.Lock()
	defer s.mu.Unlock()
	for tvdb, show := range built {
		out[tvdb] = show
		// only a series with every entry's episode count is kept; one AniList missed is asked again
		if complete(show, media) {
			s.cached[tvdb] = cachedShow{show, s.now()}
		}
	}
	return out
}

// whether AniList gave every entry of a series
func complete(show Show, media map[int]anilist.Media) bool {
	for _, e := range show.Entries {
		if _, ok := media[e.AnilistID]; !ok {
			return false
		}
	}
	return true
}

/*
Function:	buildShow
Purpose:	a series from its Fribb entries and what AniList has for them. Entries sharing a TVDB
season are ordered by when they started airing (Fribb's order when AniList has no date)
Returns:	false for a series with no entries
*/
func buildShow(tvdb int, seasons []SeasonEntry, media map[int]anilist.Media) (Show, bool) {
	if len(seasons) == 0 {
		return Show{}, false
	}
	// Fribb's order is by season with specials last; inside a season, by air date where known
	type keyed struct {
		SeasonEntry
		date, at int
	}
	entries := make([]keyed, len(seasons))
	for i, e := range seasons {
		date := startDate(media[e.AnilistID])
		if date == 0 {
			date = 99999999
		}
		entries[i] = keyed{e, date, i}
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if (a.Season == 0) != (b.Season == 0) {
			return b.Season == 0
		}
		if a.Season != b.Season {
			return a.Season < b.Season
		}
		if a.date != b.date {
			return a.date < b.date
		}
		return a.at < b.at
	})

	show := Show{TvdbID: tvdb, AnilistID: entries[0].AnilistID}
	for _, e := range entries {
		info := ShowEntryInfo{AnilistID: e.AnilistID, Season: e.Season}
		if m, ok := media[e.AnilistID]; ok && m.Episodes != nil {
			info.Episodes = *m.Episodes
		}
		show.Entries = append(show.Entries, info)
	}
	if first, ok := media[show.AnilistID]; ok {
		show.Title = first.Title.English
		if show.Title == "" {
			show.Title = first.Title.Romaji
		}
		show.CoverURL = first.CoverImage.Large
	}
	return show, true
}

// an AniList start date as yyyymmdd, 0 when it has no year
func startDate(m anilist.Media) int {
	d := m.StartDate
	if d.Year == nil {
		return 0
	}
	n := *d.Year * 10000
	if d.Month != nil {
		n += *d.Month * 100
	}
	if d.Day != nil {
		n += *d.Day
	}
	return n
}

/*
Function:	name
Purpose:	give series of more than one entry Sonarr's name and poster for them ("Monogatari", not
"Bakemonogatari"). A series of one entry already has the right name. When Sonarr can't answer,
the first entry's name stays
Params:
  - shows: the series, changed in place
*/
func (s *Shows) name(shows map[int]Show) {
	var todo []int
	for tvdb, show := range shows {
		if len(show.Entries) > 1 {
			todo = append(todo, tvdb)
		}
	}
	if len(todo) == 0 {
		return
	}

	type named struct {
		tvdb   int
		series SonarrSeries
	}
	jobs := make(chan int)
	results := make(chan named)
	var wg sync.WaitGroup
	for range min(lookupWorkers, len(todo)) {
		wg.Go(func() {
			for tvdb := range jobs {
				series, err := s.lookup("tvdb:" + strconv.Itoa(tvdb))
				if err != nil {
					if err != errNoSonarr {
						logger.Warn("Sonarr's name for TVDB series %d: %s", tvdb, err.Error())
					}
					continue
				}
				results <- named{tvdb, series}
			}
		})
	}
	go func() {
		for _, tvdb := range todo {
			jobs <- tvdb
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	for r := range results {
		show := shows[r.tvdb]
		if r.series.Title != "" && r.series.TvdbID == r.tvdb {
			show.Title = r.series.Title
			if poster := r.series.PosterURL(); poster != "" {
				show.CoverURL = poster
			}
		}
		shows[r.tvdb] = show
	}
}
