package sync

import (
	"reflect"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
)

func TestSeriesAnilistCandidates(t *testing.T) {
	maps := arr.AnimeMaps{
		TvdbToAnilist: map[int]int{1: 10, 2: 20, 3: 30},
		ShowSeasons: map[int][]arr.SeasonEntry{
			// seasons only, and the series entry is its first season
			1: {{Season: 1, AnilistID: 10}, {Season: 2, AnilistID: 11}},
			// Fribb gives no seasons
			// 2: none
			// seasons that don't include the series' own entry
			3: {{Season: 1, AnilistID: 31}},
			4: {{Season: 1, AnilistID: 40}},
		},
	}
	cases := []struct {
		name   string
		tvdbID int
		want   []int
	}{
		{"seasons, series entry not repeated", 1, []int{10, 11}},
		{"series entry only", 2, []int{20}},
		{"seasons then series entry", 3, []int{31, 30}},
		{"seasons only", 4, []int{40}},
		{"unknown series", 99, []int{}},
	}
	for _, c := range cases {
		if got := seriesAnilistCandidates(c.tvdbID, maps); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestMovieAnilistCandidate(t *testing.T) {
	maps := arr.AnimeMaps{TmdbToAnilist: map[int]int{100: 1000}}
	if id, ok := movieAnilistCandidate(100, maps); !ok || id != 1000 {
		t.Errorf("mapped movie: got %d, %v", id, ok)
	}
	if _, ok := movieAnilistCandidate(200, maps); ok {
		t.Error("unmapped movie should have no candidate")
	}
}
