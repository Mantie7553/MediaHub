package search

import (
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
)

type fakeLookup map[int]arr.Show

func (f fakeLookup) Of(id int) (arr.Show, arr.TvdbSeason, bool) {
	for _, show := range f {
		for _, e := range show.Entries {
			if e.AnilistID == id {
				return show, arr.TvdbSeason{TvdbID: show.TvdbID, Season: e.Season}, true
			}
		}
	}
	return arr.Show{}, arr.TvdbSeason{}, false
}

var monogatariShow = arr.Show{TvdbID: 102261, Title: "Monogatari", CoverURL: "poster", AnilistID: 5081, Entries: []arr.ShowEntryInfo{
	{AnilistID: 5081, Season: 1, Episodes: 15}, {AnilistID: 11597, Season: 1, Episodes: 11},
	{AnilistID: 17074, Season: 2, Episodes: 26},
}}

func TestSeriesOfTurnsASeasonIntoItsSeries(t *testing.T) {
	h := &Handler{series: fakeLookup{102261: monogatariShow}}
	progress := 3
	req := saveRequest{ExternalID: "11597", ExternalSource: "anilist", Title: "Nisemonogatari", Type: "anime", Status: "current", Progress: &progress}

	s := h.seriesOf(&req)
	if s.show == nil || s.whole || s.entryID != 11597 {
		t.Fatalf("series = %+v", s)
	}
	if req.Title != "Monogatari" || req.ExternalID != "5081" || req.CoverImageURL != "poster" {
		t.Errorf("request = %+v, want it turned to the series", req)
	}
	if count, status := s.progress(req); count != 18 || status != "current" {
		t.Errorf("progress = %d %s, want Bakemonogatari's 15 + 3", count, status)
	}
}

func TestSeriesOfTakesAGroupedResultAsTheWholeSeries(t *testing.T) {
	h := &Handler{series: fakeLookup{102261: monogatariShow}}
	progress := 40
	req := saveRequest{ExternalID: "5081", ExternalSource: "anilist", Title: "Monogatari", Type: "anime", Status: "current", Progress: &progress, TvdbID: 102261}

	s := h.seriesOf(&req)
	if !s.whole {
		t.Fatal("a save carrying the series' TVDB id is for the whole series")
	}
	if count, _ := s.progress(req); count != 40 {
		t.Errorf("progress = %d, want the 40 given across the series", count)
	}
}

func TestSeriesOfLeavesOtherSavesAlone(t *testing.T) {
	h := &Handler{series: fakeLookup{102261: monogatariShow}}
	for _, req := range []saveRequest{
		{ExternalID: "999", ExternalSource: "anilist", Type: "anime"},
		{ExternalID: "5081", ExternalSource: "anilist", Type: "movie"},
		{ExternalID: "abc", ExternalSource: "mangadex", Type: "manga"},
	} {
		if s := h.seriesOf(&req); s.show != nil {
			t.Errorf("%+v must not be a series", req)
		}
	}
}
