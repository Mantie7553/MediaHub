package arr

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
)

func testShows(t *testing.T, lookups *atomic.Int32) *Shows {
	t.Helper()
	ep := func(n int) *int { return &n }
	year := func(y, m int) anilist.FuzzyDate { return anilist.FuzzyDate{Year: &y, Month: &m} }
	maps := buildAnimeMaps([]fribbEntry{
		fe("TV", 102261, 1, 11597, 8658), // Nisemonogatari, listed first
		fe("TV", 102261, 1, 5081, 6327),  // Bakemonogatari
		fe("TV", 102261, 0, 15689, 8000), // a special
		fe("TV", 102261, 2, 17074, 9000), // Monogatari Second Season
		fe("TV", 300, 1, 1, 1),           // a series of one entry
	})
	media := map[int]anilist.Media{
		5081:  {ID: 5081, Episodes: ep(15), StartDate: year(2009, 7), Title: anilist.Title{Romaji: "Bakemonogatari"}},
		11597: {ID: 11597, Episodes: ep(11), StartDate: year(2012, 1)},
		17074: {ID: 17074, Episodes: ep(26), StartDate: year(2013, 7)},
		15689: {ID: 15689, Episodes: ep(4), StartDate: year(2012, 12)},
		1:     {ID: 1, Episodes: ep(12), Title: anilist.Title{English: "One Season"}},
	}
	return &Shows{
		cached: map[int]cachedShow{},
		maps:   func() (AnimeMaps, error) { return maps, nil },
		media: func(ids []int) (map[int]anilist.Media, error) {
			out := map[int]anilist.Media{}
			for _, id := range ids {
				if m, ok := media[id]; ok {
					out[id] = m
				}
			}
			return out, nil
		},
		lookup: func(term string) (SonarrSeries, error) {
			lookups.Add(1)
			if term != "tvdb:102261" {
				return SonarrSeries{}, errors.New("unexpected lookup " + term)
			}
			return SonarrSeries{TvdbID: 102261, Title: "Monogatari"}, nil
		},
		now: time.Now,
	}
}

func fe(typ string, tvdb, s, anilistID, anidb int) fribbEntry {
	e := fribbEntry{Type: typ, TvdbID: tvdb, AnilistID: anilistID, AnidbID: anidb}
	e.Season.Tvdb = &s
	return e
}

func TestShowsOfMonogatari(t *testing.T) {
	var lookups atomic.Int32
	s := testShows(t, &lookups)

	show, at, ok := s.Of(11597)
	if !ok || at != (TvdbSeason{102261, 1}) {
		t.Fatalf("Of(Nisemonogatari) = %v, %v", at, ok)
	}
	if show.Title != "Monogatari" || show.AnilistID != 5081 {
		t.Errorf("show = %q linked to %d, want Monogatari linked to Bakemonogatari", show.Title, show.AnilistID)
	}
	var order []string
	for _, e := range show.Entries {
		order = append(order, fmt.Sprintf("%d:%d", e.AnilistID, e.Episodes))
	}
	if want := "[5081:15 11597:11 17074:26 15689:4]"; fmt.Sprint(order) != want {
		t.Errorf("entries = %v, want %s", order, want)
	}

	// kept, not asked again
	s.Of(5081)
	if n := lookups.Load(); n != 1 {
		t.Errorf("Sonarr asked %d times, want 1", n)
	}
}

func TestShowsOfOneEntryKeepsItsName(t *testing.T) {
	var lookups atomic.Int32
	s := testShows(t, &lookups)
	show, _, ok := s.Of(1)
	if !ok || show.Title != "One Season" {
		t.Fatalf("show = %+v, %v", show, ok)
	}
	if lookups.Load() != 0 {
		t.Error("a series of one entry must not ask Sonarr")
	}
	if _, _, ok := s.Of(424242); ok {
		t.Error("an entry Fribb doesn't place must not be a series")
	}
}
