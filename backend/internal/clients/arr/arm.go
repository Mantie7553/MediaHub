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
	TmdbID    struct {
		Movie []int `json:"movie"`
	} `json:"themoviedb_id"`
}

func FetchAnimeMaps() (tvdbToAnilist map[int]int, tmdbToAnilist map[int]int, err error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get("https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json")
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fribb anime-lists returned %d", resp.StatusCode)
	}

	var entries []fribbEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, nil, err
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

	return tvdbToAnilist, tmdbToAnilist, nil
}
