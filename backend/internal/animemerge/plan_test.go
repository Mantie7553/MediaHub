package animemerge

import (
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
)

// shaped like Monogatari: two entries in season 1, one in season 2
var maps = arr.AnimeMaps{
	AnilistToShow: map[int]arr.TvdbSeason{
		5081: {TvdbID: 102261, Season: 1}, 11597: {TvdbID: 102261, Season: 1}, 17074: {TvdbID: 102261, Season: 2},
		1: {TvdbID: 300, Season: 1},
	},
	ShowSeasons: map[int][]arr.SeasonEntry{
		102261: {{Season: 1, AnilistID: 5081}, {Season: 1, AnilistID: 11597}, {Season: 2, AnilistID: 17074}},
		300:    {{Season: 1, AnilistID: 1}},
	},
}

var show = []tracking.ShowEntry{
	{AnilistID: 5081, Season: 1, Episodes: 15}, {AnilistID: 11597, Season: 1, Episodes: 11}, {AnilistID: 17074, Season: 2, Episodes: 26},
}

func TestGroupRowsGoesByTvdbNeverByName(t *testing.T) {
	groups := groupRows([]row{
		{ID: "a", Title: "Bakemonogatari", ExternalID: "5081", ExternalSource: "anilist"},
		{ID: "b", Title: "Nisemonogatari", ExternalID: "11597", ExternalSource: "anilist"},
		{ID: "c", Title: "Monogatari", TvdbID: 102261, Sonarr: true},
		{ID: "d", Title: "Monogatari Series: Second Season", ExternalID: "999", ExternalSource: "anilist"}, // unmapped
		{ID: "e", Title: "One Season", ExternalID: "1", ExternalSource: "anilist"},                         // alone
	}, maps)
	if len(groups) != 1 || len(groups[102261]) != 3 {
		t.Fatalf("groups = %v, want only Monogatari's three titles", groups)
	}
}

func TestPlanGroupKeepsTheSonarrTitle(t *testing.T) {
	g := planGroup(102261, []row{
		{ID: "b", Title: "Nisemonogatari", ExternalID: "11597", ExternalSource: "anilist"},
		{ID: "a", Title: "Bakemonogatari", ExternalID: "5081", ExternalSource: "anilist"},
		{ID: "c", Title: "Monogatari", ExternalID: "17074", ExternalSource: "anilist", TvdbID: 102261, Sonarr: true},
	}, maps)
	if g.Keep.ID != "c" || g.Keep.Relative() {
		t.Errorf("keep = %+v, want the Sonarr title, counted as the series", g.Keep)
	}
	if len(g.Drop) != 2 || g.Drop[0].ID != "a" || g.Drop[0].Season != 1 || !g.Drop[0].Relative() {
		t.Errorf("drop = %+v, want Bakemonogatari then Nisemonogatari, each a season", g.Drop)
	}
}

func TestPlanGroupWithoutSonarrKeepsTheEarliestSeason(t *testing.T) {
	g := planGroup(102261, []row{
		{ID: "c", Title: "Monogatari Second Season", ExternalID: "17074", ExternalSource: "anilist"},
		{ID: "a", Title: "Bakemonogatari", ExternalID: "5081", ExternalSource: "anilist"},
	}, maps)
	if g.Keep.ID != "a" || !g.Keep.Relative() {
		t.Errorf("keep = %+v, want Bakemonogatari, still counted as its season", g.Keep)
	}
}

func TestCombine(t *testing.T) {
	n := func(v int) *int { return &v }
	bake := Title{AnilistID: 5081, Season: 1}
	nise := Title{AnilistID: 11597, Season: 1}
	second := Title{AnilistID: 17074, Season: 2}
	series := Title{AnilistID: 5081, Season: -1, Sonarr: true}

	got := combine([]Entry{
		{Title: second, Status: "current", Done: 4, Rating: n(6), RatingIsSet: true},
		{Title: bake, Status: "completed", Done: 15, Rating: n(9), RatingIsSet: true},
		{Title: nise, Status: "completed", Done: 11, Rating: n(7), RatingIsSet: true},
	}, show)
	if got.Done != 30 || got.Status != "current" {
		t.Errorf("progress = %d %s, want 15 + 11 + 4 = 30 current", got.Done, got.Status)
	}
	if got.Rating != nil || len(got.Seasons) != 3 {
		t.Errorf("ratings = %v %+v, want three season ratings and no series rating", got.Rating, got.Seasons)
	}

	// a Sonarr title already watched further keeps its progress and its own rating
	got = combine([]Entry{
		{Title: series, Status: "completed", Done: 52, Rating: n(10), RatingIsSet: true},
		{Title: bake, Status: "current", Done: 3, Rating: n(9), RatingIsSet: true},
	}, show)
	if got.Done != 52 || got.Status != "completed" || got.Rating == nil || *got.Rating != 10 {
		t.Errorf("got %+v, want the series' own 52 completed rated 10", got)
	}

	// a rating that was only an average isn't carried as a pick
	got = combine([]Entry{{Title: series, Status: "current", Done: 2, Rating: n(8)}}, show)
	if got.Rating != nil {
		t.Errorf("an average became a picked rating: %d", *got.Rating)
	}
}
