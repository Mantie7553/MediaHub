package arr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

type fribbEntry struct {
	Type      string `json:"type"`
	TvdbID    int    `json:"tvdb_id"`
	AnilistID int    `json:"anilist_id"`
	MalID     int    `json:"mal_id"`
	AnidbID   int    `json:"anidb_id"`
	TmdbID    struct {
		Movie []int `json:"movie"`
	} `json:"themoviedb_id"`
	Season struct {
		Tvdb *int `json:"tvdb"`
	} `json:"season"`
}

// TvdbSeason is a TVDB series and one of its seasons
type TvdbSeason struct {
	TvdbID int
	Season int
}

// SeasonEntry is one AniList/MAL entry inside a TVDB series: a season, a cour of one, or a
// special. Shows like Monogatari give each entry its own name, yet TVDB keeps them as one series
type SeasonEntry struct {
	Season    int
	AnilistID int
	MalID     int
	anidbID   int
}

func FetchAnimeMaps() (AnimeMaps, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get("https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json")
	if err != nil {
		return AnimeMaps{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AnimeMaps{}, fmt.Errorf("fribb anime-lists returned %d", resp.StatusCode)
	}

	var entries []fribbEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return AnimeMaps{}, err
	}
	return buildAnimeMaps(entries), nil
}

/*
Function:	buildAnimeMaps
Purpose:	key Fribb's list every way the app looks it up
Params:
  - entries: Fribb's anime list
*/
func buildAnimeMaps(entries []fribbEntry) AnimeMaps {
	tmdbToAnilist := make(map[int]int)
	for _, e := range entries {
		if e.Type == "MOVIE" && e.AnilistID > 0 && len(e.TmdbID.Movie) > 0 {
			tmdbToAnilist[e.TmdbID.Movie[0]] = e.AnilistID
		}
	}

	shows := showSeasons(entries)
	tvdbToAnilist := make(map[int]int, len(shows))
	anilistToShow := make(map[int]TvdbSeason)
	for tvdb, seasons := range shows {
		if first := canonicalEntry(seasons); first.AnilistID > 0 {
			tvdbToAnilist[tvdb] = first.AnilistID
		}
		for _, s := range seasons {
			if s.AnilistID > 0 {
				anilistToShow[s.AnilistID] = TvdbSeason{tvdb, s.Season}
			}
		}
	}

	// series Fribb gives no seasons for still map to one of their entries, as they always have
	for _, e := range entries {
		if _, ok := tvdbToAnilist[e.TvdbID]; !ok && e.TvdbID > 0 && e.AnilistID > 0 && e.Type != "MOVIE" {
			tvdbToAnilist[e.TvdbID] = e.AnilistID
		}
	}

	return AnimeMaps{
		TvdbToAnilist: tvdbToAnilist,
		TmdbToAnilist: tmdbToAnilist,
		SeasonToMal:   seasonMalMap(entries),
		AnilistToShow: anilistToShow,
		ShowSeasons:   shows,
	}
}

/*
Function:	showSeasons
Purpose:	each TVDB series' entries in watch order: by TVDB season with the specials (season 0)
last, and inside a season by AniDB id, which grows with each new cour. Movies are left out,
they're their own titles
Params:
  - entries: Fribb's anime list
*/
func showSeasons(entries []fribbEntry) map[int][]SeasonEntry {
	out := map[int][]SeasonEntry{}
	seen := map[int]bool{}
	for _, e := range entries {
		if e.TvdbID <= 0 || e.Season.Tvdb == nil || e.Type == "MOVIE" || e.AnilistID <= 0 {
			continue
		}
		if seen[e.AnilistID] {
			continue
		}
		seen[e.AnilistID] = true
		out[e.TvdbID] = append(out[e.TvdbID], SeasonEntry{
			Season: *e.Season.Tvdb, AnilistID: e.AnilistID, MalID: e.MalID, anidbID: e.AnidbID,
		})
	}
	for _, seasons := range out {
		sortSeasons(seasons)
	}
	return out
}

// a series' entries in watch order: by TVDB season, specials last, then by AniDB id
func sortSeasons(seasons []SeasonEntry) {
	sort.SliceStable(seasons, func(i, j int) bool {
		a, b := seasons[i], seasons[j]
		if (a.Season == 0) != (b.Season == 0) {
			return b.Season == 0
		}
		if a.Season != b.Season {
			return a.Season < b.Season
		}
		return a.anidbID < b.anidbID
	})
}

// the entry that stands for the whole series: the first of its lowest numbered season, or a
// special when that's all the series has
func canonicalEntry(seasons []SeasonEntry) SeasonEntry {
	if len(seasons) == 0 {
		return SeasonEntry{}
	}
	return seasons[0]
}

/*
Function:	seasonMalMap
Purpose:	map each TVDB season to its MyAnimeList entry. A season more than one entry claims
(a split cour, the specials) is left out, its episode numbers don't match any one entry
Params:
  - entries: Fribb's anime list
*/
func seasonMalMap(entries []fribbEntry) map[TvdbSeason]int {
	out := map[TvdbSeason]int{}
	ambiguous := map[TvdbSeason]bool{}
	for _, e := range entries {
		if e.TvdbID <= 0 || e.MalID <= 0 || e.Season.Tvdb == nil {
			continue
		}
		key := TvdbSeason{e.TvdbID, *e.Season.Tvdb}
		if existing, ok := out[key]; ok && existing != e.MalID {
			ambiguous[key] = true
		}
		out[key] = e.MalID
	}
	for key := range ambiguous {
		delete(out, key)
	}
	return out
}
