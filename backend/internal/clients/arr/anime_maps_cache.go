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
}

// how long a downloaded copy of Fribb's list is used before fetching it again
const animeMapsTTL = 24 * time.Hour

type animeMapsCache struct {
	mu      sync.Mutex
	maps    *AnimeMaps
	fetched time.Time
	fetch   func() (AnimeMaps, error)
	now     func() time.Time
}

var sharedAnimeMaps = &animeMapsCache{
	fetch: func() (AnimeMaps, error) {
		tvdb, tmdb, seasons, err := FetchAnimeMaps()
		return AnimeMaps{tvdb, tmdb, seasons}, err
	},
	now: time.Now,
}

/*
Function:	LoadAnimeMaps
Purpose:	Fribb's anime list, downloaded at most once a day and shared by the full sync and the
webhooks. When a download fails the last copy is returned with the error; with no copy yet the
maps are empty
*/
func LoadAnimeMaps() (AnimeMaps, error) {
	return sharedAnimeMaps.load()
}

func (c *animeMapsCache) load() (AnimeMaps, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.maps != nil && c.now().Sub(c.fetched) < animeMapsTTL {
		return *c.maps, nil
	}
	fresh, err := c.fetch()
	if err != nil {
		if c.maps != nil {
			return *c.maps, err
		}
		return AnimeMaps{map[int]int{}, map[int]int{}, map[TvdbSeason]int{}}, err
	}
	c.maps = &fresh
	c.fetched = c.now()
	return fresh, nil
}
