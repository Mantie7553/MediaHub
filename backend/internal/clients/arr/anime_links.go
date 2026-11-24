package arr

import (
	"sync"
)

// SeriesLink places an AniList entry in a TVDB series by hand, for entries Fribb's list leaves
// out of their series (Gintama's 2006 entry) or puts in the wrong one
type SeriesLink struct {
	AnilistID int `json:"anilist_id"`
	TvdbID    int `json:"tvdb_id"`
	Season    int `json:"season"`
}

var seriesLinks = struct {
	mu      sync.Mutex
	links   []SeriesLink
	version int
}{}

/*
Function:	SetSeriesLinks
Purpose:	the links an admin made, laid over Fribb's list from now on. Replaces the ones set before
Params:
  - links: every link
*/
func SetSeriesLinks(links []SeriesLink) {
	seriesLinks.mu.Lock()
	defer seriesLinks.mu.Unlock()
	seriesLinks.links = append([]SeriesLink(nil), links...)
	seriesLinks.version++
}

func currentLinks() ([]SeriesLink, int) {
	seriesLinks.mu.Lock()
	defer seriesLinks.mu.Unlock()
	return seriesLinks.links, seriesLinks.version
}

/*
Function:	applyLinks
Purpose:	Fribb's maps with the links laid over them: each linked entry is taken out of any series
Fribb put it in and added to its linked series' season. The maps given are left as they were
Params:
  - maps: Fribb's maps
  - links: the links
*/
func applyLinks(maps AnimeMaps, links []SeriesLink) AnimeMaps {
	if len(links) == 0 {
		return maps
	}
	out := maps
	out.AnilistToShow = make(map[int]TvdbSeason, len(maps.AnilistToShow)+len(links))
	for k, v := range maps.AnilistToShow {
		out.AnilistToShow[k] = v
	}
	out.TvdbToAnilist = make(map[int]int, len(maps.TvdbToAnilist)+len(links))
	for k, v := range maps.TvdbToAnilist {
		out.TvdbToAnilist[k] = v
	}
	// series slices are shared with the maps given until one is changed, then copied
	out.ShowSeasons = make(map[int][]SeasonEntry, len(maps.ShowSeasons)+len(links))
	for k, v := range maps.ShowSeasons {
		out.ShowSeasons[k] = v
	}
	copied := map[int]bool{}
	own := func(tvdb int) []SeasonEntry {
		if !copied[tvdb] {
			copied[tvdb] = true
			out.ShowSeasons[tvdb] = append([]SeasonEntry(nil), out.ShowSeasons[tvdb]...)
		}
		return out.ShowSeasons[tvdb]
	}

	for _, l := range links {
		if l.AnilistID <= 0 || l.TvdbID <= 0 || l.Season < 0 {
			continue
		}
		var malID int
		if at, ok := out.AnilistToShow[l.AnilistID]; ok {
			seasons := own(at.TvdbID)
			kept := seasons[:0]
			for _, e := range seasons {
				if e.AnilistID == l.AnilistID {
					malID = e.MalID
					continue
				}
				kept = append(kept, e)
			}
			out.ShowSeasons[at.TvdbID] = kept
		}
		out.AnilistToShow[l.AnilistID] = TvdbSeason{l.TvdbID, l.Season}
		out.ShowSeasons[l.TvdbID] = append(own(l.TvdbID), SeasonEntry{
			Season: l.Season, AnilistID: l.AnilistID, MalID: malID,
		})
		if _, ok := out.TvdbToAnilist[l.TvdbID]; !ok {
			out.TvdbToAnilist[l.TvdbID] = l.AnilistID
		}
	}
	for tvdb := range copied {
		sortSeasons(out.ShowSeasons[tvdb])
	}
	return out
}
