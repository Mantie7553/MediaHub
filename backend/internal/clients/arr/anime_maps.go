package arr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type fribbEntry struct {
	Type      string `json:"type"`
	TvdbID    int    `json:"tvdb_id"`
	AnilistID int    `json:"anilist_id"`
	MalID     int    `json:"mal_id"`
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

func FetchAnimeMaps() (tvdbToAnilist map[int]int, tmdbToAnilist map[int]int, seasonToMal map[TvdbSeason]int, err error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get("https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json")
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, nil, fmt.Errorf("fribb anime-lists returned %d", resp.StatusCode)
	}

	var entries []fribbEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, nil, nil, err
	}

	tvdbToAnilist = make(map[int]int, len(entries))
	tmdbToAnilist = make(map[int]int)

	for _, e := range entries {
		if e.TvdbID > 0 && e.AnilistID > 0 {
			tvdbToAnilist[e.TvdbID] = e.AnilistID
		}
		if e.Type == "MOVIE" && e.AnilistID > 0 && len(e.TmdbID.Movie) > 0 {
			tmdbToAnilist[e.TmdbID.Movie[0]] = e.AnilistID
		}
	}

	return tvdbToAnilist, tmdbToAnilist, seasonMalMap(entries), nil
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
