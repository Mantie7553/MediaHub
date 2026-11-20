package search

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
)

func TestFromMangaDexUsesThumbnails(t *testing.T) {
	m := mangadex.Manga{ID: "abc"}
	m.Attributes.Title.JaRo = "Sousou no Frieren"
	m.Relationships = []mangadex.MangaRelations{{Type: "author"}, {Type: "cover_art"}}
	m.Relationships[1].Attributes.FileName = "cover.png"

	got := fromMangaDex([]mangadex.Manga{m, {ID: "none"}})
	if want := "https://uploads.mangadex.org/covers/abc/cover.png.512.jpg"; got[0].CoverImageURL != want {
		t.Errorf("cover = %q, want %q", got[0].CoverImageURL, want)
	}
	if got[0].Title != "Sousou no Frieren" {
		t.Errorf("title = %q, want the romaji title when there is no English one", got[0].Title)
	}
	if got[1].CoverImageURL != "" {
		t.Errorf("cover = %q, want none for a manga without cover art", got[1].CoverImageURL)
	}
}

func TestFromAnilistPrefersEnglishTitles(t *testing.T) {
	a, b := anilist.Media{ID: 1}, anilist.Media{ID: 2}
	a.Title.English, a.Title.Romaji = "Frieren", "Sousou no Frieren"
	b.Title.Romaji = "Dandadan"

	got := fromAnilist([]anilist.Media{a, b}, "movie")
	if got[0].Title != "Frieren" || got[1].Title != "Dandadan" {
		t.Errorf("titles = %q, %q", got[0].Title, got[1].Title)
	}
	if got[0].ExternalID != "1" || got[0].Type != "movie" || got[0].ExternalSource != "anilist" {
		t.Errorf("result = %+v", got[0])
	}
}

func TestSearchKeyIgnoresCaseAndSpacing(t *testing.T) {
	if searchKey("anime", "  Frieren   Beyond ") != searchKey("anime", "frieren beyond") {
		t.Error("case and spacing should not make a different search")
	}
	if searchKey("anime", "frieren") == searchKey("manga", "frieren") {
		t.Error("the media type is part of the search")
	}
}

func TestSearchRejectsUnknownTypes(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(nil).Search(rec, httptest.NewRequest(http.MethodGet, "/api/search?type=book", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

type fakeSeries struct {
	maps  arr.AnimeMaps
	shows map[int]arr.Show
	asked []int
}

func (f *fakeSeries) Maps() arr.AnimeMaps { return f.maps }
func (f *fakeSeries) Load(ids []int) map[int]arr.Show {
	f.asked = append(f.asked, ids...)
	out := map[int]arr.Show{}
	for _, id := range ids {
		if s, ok := f.shows[id]; ok {
			out[id] = s
		}
	}
	return out
}

func TestGroupSeriesMonogatari(t *testing.T) {
	f := &fakeSeries{
		maps: arr.AnimeMaps{
			AnilistToShow: map[int]arr.TvdbSeason{
				5081: {TvdbID: 102261, Season: 1}, 11597: {TvdbID: 102261, Season: 1},
				17074: {TvdbID: 102261, Season: 2}, 1: {TvdbID: 300, Season: 1},
				50: {TvdbID: 400, Season: 1}, 51: {TvdbID: 400, Season: 2},
			},
			ShowSeasons: map[int][]arr.SeasonEntry{
				102261: {{Season: 1, AnilistID: 5081}, {Season: 1, AnilistID: 11597}, {Season: 2, AnilistID: 17074}},
				300:    {{Season: 1, AnilistID: 1}},
				400:    {{Season: 1, AnilistID: 50}, {Season: 2, AnilistID: 51}},
			},
		},
		shows: map[int]arr.Show{
			102261: {TvdbID: 102261, Title: "Monogatari", CoverURL: "poster", AnilistID: 5081, Entries: []arr.ShowEntryInfo{
				{AnilistID: 5081, Season: 1, Episodes: 15}, {AnilistID: 11597, Season: 1, Episodes: 11},
				{AnilistID: 17074, Season: 2, Episodes: 26},
			}},
			// 400 couldn't be looked up
		},
	}
	r := func(id, title string) SearchResult {
		return SearchResult{ExternalID: id, ExternalSource: "anilist", Title: title, Type: "anime"}
	}
	lists := groupSeries(f,
		[]SearchResult{r("99", "Unmapped"), r("17074", "Monogatari Second Season"), r("1", "One Season"), r("11597", "Nisemonogatari")},
		[]SearchResult{r("50", "Show Season 1"), r("51", "Show Season 2")},
	)

	got := lists[0]
	if len(got) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(got), got)
	}
	if got[0].Title != "Unmapped" || got[0].TvdbID != 0 {
		t.Errorf("an unmapped result must stay as it is: %+v", got[0])
	}
	m := got[1]
	if m.Title != "Monogatari" || m.ExternalID != "5081" || m.TvdbID != 102261 || m.Seasons != 2 ||
		m.CoverImageURL != "poster" || m.TotalParts == nil || *m.TotalParts != 52 {
		t.Errorf("series = %+v, want Monogatari at the second season's place", m)
	}
	if got[2].Title != "One Season" || got[2].TvdbID != 300 || got[2].Seasons != 0 {
		t.Errorf("a series of one entry = %+v", got[2])
	}
	if len(lists[1]) != 2 || lists[1][0].TvdbID != 0 {
		t.Errorf("a series that couldn't be looked up must stay as seasons: %+v", lists[1])
	}
	if len(f.asked) != 2 {
		t.Errorf("looked up %v, want only the two series of several entries", f.asked)
	}
}

func TestLoadAnimeDiscoveryReadsMorePagesWhenSeasonsCollapse(t *testing.T) {
	// series 1..: every series is two entries (ids 2k and 2k+1) ranked side by side, so a page
	// of 50 entries holds 25 series
	f := &fakeSeries{maps: arr.AnimeMaps{AnilistToShow: map[int]arr.TvdbSeason{}, ShowSeasons: map[int][]arr.SeasonEntry{}}, shows: map[int]arr.Show{}}
	for k := 1; k <= 200; k++ {
		a, b := 2*k, 2*k+1
		f.maps.AnilistToShow[a] = arr.TvdbSeason{TvdbID: k, Season: 1}
		f.maps.AnilistToShow[b] = arr.TvdbSeason{TvdbID: k, Season: 2}
		f.maps.ShowSeasons[k] = []arr.SeasonEntry{{Season: 1, AnilistID: a}, {Season: 2, AnilistID: b}}
		f.shows[k] = arr.Show{TvdbID: k, AnilistID: a, Entries: []arr.ShowEntryInfo{{AnilistID: a, Season: 1}, {AnilistID: b, Season: 2}}}
	}
	pages := 0
	pager := func(_, _ string, perPage, page int) (*anilist.DiscoveryResult, error) {
		pages++
		var list []anilist.Media
		for i := range perPage {
			list = append(list, anilist.Media{ID: 2 + (page-1)*perPage + i})
		}
		return &anilist.DiscoveryResult{Trending: list, Popular: list, TopRated: list}, nil
	}

	got, err := loadAnimeDiscovery(pager, f)
	if err != nil {
		t.Fatal(err)
	}
	if pages != 2 {
		t.Errorf("read %d pages, want 2 (25 series a page)", pages)
	}
	if len(got.TopRated) != discoverTarget {
		t.Errorf("top rated has %d series, want %d", len(got.TopRated), discoverTarget)
	}
	seen := map[string]bool{}
	for _, r := range got.TopRated {
		if seen[r.ExternalID] {
			t.Fatalf("series %s listed twice", r.ExternalID)
		}
		seen[r.ExternalID] = true
	}
}

func TestLoadAnimeDiscoveryKeepsEarlierPagesWhenALaterOneFails(t *testing.T) {
	f := &fakeSeries{maps: arr.AnimeMaps{}, shows: map[int]arr.Show{}}
	pager := func(_, _ string, _, page int) (*anilist.DiscoveryResult, error) {
		if page > 1 {
			return nil, errors.New("rate limited")
		}
		return &anilist.DiscoveryResult{TopRated: []anilist.Media{{ID: 1}}, Trending: make([]anilist.Media, 50)}, nil
	}
	got, err := loadAnimeDiscovery(pager, f)
	if err != nil || len(got.TopRated) != 1 {
		t.Errorf("got %v, %v; want the first page", got.TopRated, err)
	}
}
