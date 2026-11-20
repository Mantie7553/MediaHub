package tracking

import (
	"fmt"
	"testing"
)

func TestSeasonScores(t *testing.T) {
	got := SeasonScores([]SeasonRating{
		{Season: 1, AnilistID: 5081, Rating: 9},  // Bakemonogatari
		{Season: 1, AnilistID: 11597, Rating: 6}, // Nisemonogatari
		{Season: 2, AnilistID: 17074, Rating: 7},
		{Season: 2, AnilistID: 0, Rating: 10}, // rated on the season itself
		{Season: 0, AnilistID: 15689, Rating: 3},
	})
	want := []SeasonScore{{0, 3, false}, {1, 8, false}, {2, 10, true}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %v, want %v (season 1 is 7.5 rounded up)", got, want)
	}
	if r := SeriesRating(got); r == nil || *r != 9 {
		t.Errorf("series = %v, want 9: seasons 1 and 2 averaged, the special left out", r)
	}
}

func TestSeriesRatingNeedsARegularSeason(t *testing.T) {
	if r := SeriesRating([]SeasonScore{{Season: 0, Rating: 8}}); r != nil {
		t.Errorf("only specials rated = %d, want none", *r)
	}
	if r := SeriesRating(nil); r != nil {
		t.Errorf("nothing rated = %d, want none", *r)
	}
}
