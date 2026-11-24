package arr

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func seasonOrder(entries []SeasonEntry) string {
	var out []string
	for _, e := range entries {
		out = append(out, fmt.Sprintf("%d:%d", e.Season, e.AnilistID))
	}
	return fmt.Sprint(out)
}

func TestApplyLinksAddsAnUnmappedEntry(t *testing.T) {
	maps := buildAnimeMaps([]fribbEntry{
		fe("TV", 79089, 2, 9969, 200), // Gintama', Fribb's season 2
		fe("TV", 79089, 0, 9970, 300), // a special
	})
	linked := applyLinks(maps, []SeriesLink{{AnilistID: 918, TvdbID: 79089, Season: 1}})

	if at := linked.AnilistToShow[918]; at != (TvdbSeason{79089, 1}) {
		t.Errorf("AnilistToShow[918] = %v, want series 79089 season 1", at)
	}
	if got, want := seasonOrder(linked.ShowSeasons[79089]), "[1:918 2:9969 0:9970]"; got != want {
		t.Errorf("seasons = %s, want %s (specials last)", got, want)
	}
	// the maps given are left alone
	if _, ok := maps.AnilistToShow[918]; ok {
		t.Error("applyLinks changed the AnilistToShow it was given")
	}
	if got := seasonOrder(maps.ShowSeasons[79089]); got != "[2:9969 0:9970]" {
		t.Errorf("applyLinks changed the ShowSeasons it was given: %s", got)
	}
}

func TestApplyLinksMovesAnEntryBetweenSeries(t *testing.T) {
	maps := buildAnimeMaps([]fribbEntry{
		fe("TV", 100, 1, 1, 1),
		fe("TV", 100, 2, 2, 2),
		fe("TV", 200, 1, 3, 3),
	})
	linked := applyLinks(maps, []SeriesLink{{AnilistID: 2, TvdbID: 200, Season: 2}})

	if got := seasonOrder(linked.ShowSeasons[100]); got != "[1:1]" {
		t.Errorf("old series = %s, want the entry taken out", got)
	}
	if got := seasonOrder(linked.ShowSeasons[200]); got != "[1:3 2:2]" {
		t.Errorf("new series = %s, want the entry added as season 2", got)
	}
	if got := seasonOrder(maps.ShowSeasons[100]); got != "[1:1 2:2]" {
		t.Errorf("applyLinks changed the series it took the entry from: %s", got)
	}
}

func TestApplyLinksStartsASeriesFribbDoesntKnow(t *testing.T) {
	linked := applyLinks(buildAnimeMaps(nil), []SeriesLink{{AnilistID: 918, TvdbID: 79089, Season: 1}})
	if linked.TvdbToAnilist[79089] != 918 {
		t.Errorf("TvdbToAnilist[79089] = %d, want the linked entry", linked.TvdbToAnilist[79089])
	}
	if got := seasonOrder(linked.ShowSeasons[79089]); got != "[1:918]" {
		t.Errorf("seasons = %s", got)
	}
}

func TestShowsOfALinkedEntry(t *testing.T) {
	var lookups atomic.Int32
	s := testShows(t, &lookups)
	base := s.maps
	// Bakemonogatari's single-entry neighbour placed as Monogatari's season 3 by hand
	s.maps = func() (AnimeMaps, error) {
		m, err := base()
		return applyLinks(m, []SeriesLink{{AnilistID: 1, TvdbID: 102261, Season: 3}}), err
	}

	show, at, ok := s.Of(1)
	if !ok || at != (TvdbSeason{102261, 3}) {
		t.Fatalf("Of(1) = %v, %v, want Monogatari season 3", at, ok)
	}
	var order []string
	for _, e := range show.Entries {
		order = append(order, fmt.Sprintf("%d:%d", e.AnilistID, e.Episodes))
	}
	if want := "[5081:15 11597:11 17074:26 1:12 15689:4]"; fmt.Sprint(order) != want {
		t.Errorf("entries = %v, want %s", order, want)
	}
}

func TestShowsForget(t *testing.T) {
	var lookups atomic.Int32
	s := testShows(t, &lookups)
	s.Of(5081)
	s.Forget(102261)
	s.Of(5081)
	if lookups.Load() != 2 {
		t.Errorf("lookups = %d, a forgotten series should be built again", lookups.Load())
	}
}

func TestAnimeMapsCacheLaysLinksOver(t *testing.T) {
	links, version := []SeriesLink(nil), 0
	c := &animeMapsCache{
		now: time.Now,
		fetch: func() (AnimeMaps, error) {
			return buildAnimeMaps([]fribbEntry{fe("TV", 100, 1, 1, 1)}), nil
		},
		links: func() ([]SeriesLink, int) { return links, version },
	}
	if m, _ := c.load(); len(m.ShowSeasons[100]) != 1 {
		t.Fatalf("seasons = %v before any link", m.ShowSeasons[100])
	}
	links, version = []SeriesLink{{AnilistID: 2, TvdbID: 100, Season: 2}}, 1
	if m, _ := c.load(); seasonOrder(m.ShowSeasons[100]) != "[1:1 2:2]" {
		t.Errorf("seasons = %s, want the new link laid over", seasonOrder(m.ShowSeasons[100]))
	}
	links, version = nil, 2
	if m, _ := c.load(); len(m.ShowSeasons[100]) != 1 {
		t.Errorf("seasons = %v, want the removed link gone", m.ShowSeasons[100])
	}
}
