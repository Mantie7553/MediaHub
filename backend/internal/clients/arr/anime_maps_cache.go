package arr

import (
	"sync"
	"time"
)

// AnimeMaps: Fribb's anime list, keyed the ways the syncs look it up
type AnimeMaps struct {
	TvdbToAnilist map[int]int
	TmdbToAnilist map[int]int
	SeasonToMal   map[TvdbSeason]int
	// each AniList entry's TVDB series and season
	AnilistToShow map[int]TvdbSeason
	// each TVDB series' AniList entries in watch order, specials last
	ShowSeasons map[int][]SeasonEntry
}

/*
Function:	Show
Purpose:	the TVDB series an AniList entry is part of, and that series' entries in watch order
Returns:	false when Fribb doesn't place the entry in a series
*/
func (m AnimeMaps) Show(anilistID int) (TvdbSeason, []SeasonEntry, bool) {
	at, ok := m.AnilistToShow[anilistID]
	if !ok {
		return TvdbSeason{}, nil, false
	}
	return at, m.ShowSeasons[at.TvdbID], true
}

// how long a downloaded copy of Fribb's list is used before fetching it again
const animeMapsTTL = 24 * time.Hour

type animeMapsCache struct {
	mu      sync.Mutex
	maps    *AnimeMaps
	fetched time.Time
	fetch   func() (AnimeMaps, error)
	now     func() time.Time

	// the maps with the admin's series links laid over them, and what they were built from
	linked        *AnimeMaps
	linkedFrom    *AnimeMaps
	linkedVersion int
	links         func() ([]SeriesLink, int)
}

var sharedAnimeMaps = &animeMapsCache{
	fetch: FetchAnimeMaps,
	now:   time.Now,
	links: currentLinks,
}

/*
Function:	LoadAnimeMaps
Purpose:	Fribb's anime list, downloaded at most once a day and shared by the full sync and the
webhooks, with the series links an admin made laid over it. When a download fails the last copy
is returned with the error; with no copy yet the maps hold only the links
*/
func LoadAnimeMaps() (AnimeMaps, error) {
	return sharedAnimeMaps.load()
}

func (c *animeMapsCache) load() (AnimeMaps, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	base, err := c.fribb()
	var links []SeriesLink
	var version int
	if c.links != nil {
		links, version = c.links()
	}
	if base == nil {
		empty := buildAnimeMaps(nil)
		return applyLinks(empty, links), err
	}
	if c.linked == nil || c.linkedFrom != base || c.linkedVersion != version {
		linked := applyLinks(*base, links)
		c.linked, c.linkedFrom, c.linkedVersion = &linked, base, version
	}
	return *c.linked, err
}

// Fribb's list as last downloaded, fetched again once a day; nil when it never came
func (c *animeMapsCache) fribb() (*AnimeMaps, error) {
	if c.maps != nil && c.now().Sub(c.fetched) < animeMapsTTL {
		return c.maps, nil
	}
	fresh, err := c.fetch()
	if err != nil {
		return c.maps, err
	}
	c.maps = &fresh
	c.fetched = c.now()
	return c.maps, nil
}
