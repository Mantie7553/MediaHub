package manga

import (
	"fmt"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/suwayomi"
)

func TestSameTitle(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Frieren: Beyond Journey's End", "frieren beyond journeys end", true},
		{"Chainsaw Man", "Chainsaw Man 2", false},
		{"", "", false},
		{"!!!", "???", false},
		{"Kaguya-sama", "KAGUYA SAMA", true},
	}
	for _, c := range cases {
		if got := sameTitle(c.a, c.b); got != c.want {
			t.Errorf("sameTitle(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestRankCandidatesPutsTheNewestExactMatchFirst(t *testing.T) {
	stats := func(latest float64) *ChapterStats { return &ChapterStats{Latest: latest, Count: int(latest)} }
	results := []SuwayomiCandidate{
		{Manga: suwayomi.Manga{ID: 1, Title: "Chainsaw Man", SourceName: "A"}, Exact: true, Chapters: stats(97)},
		{Manga: suwayomi.Manga{ID: 2, Title: "Chainsaw Man", SourceName: "B"}, Exact: true, Chapters: stats(213)},
		{Manga: suwayomi.Manga{ID: 3, Title: "Chainsaw Man", SourceName: "A"}, Exact: true},
		{Manga: suwayomi.Manga{ID: 4, Title: "Chainsaw Man (Colored)", SourceName: "A"}, Chapters: stats(300)},
	}
	rankCandidates(results)

	var order []int
	for _, r := range results {
		order = append(order, r.ID)
	}
	if fmt.Sprint(order) != "[2 1 3 4]" {
		t.Errorf("order = %v, want [2 1 3 4]", order)
	}
	if !results[0].MostRecent || results[1].MostRecent || results[3].MostRecent {
		t.Error("only the newest exact match should be marked most recent")
	}
}

func TestRankCandidatesWithoutStatsMarksNothing(t *testing.T) {
	results := []SuwayomiCandidate{{Manga: suwayomi.Manga{ID: 1}, Exact: true}}
	rankCandidates(results)
	if results[0].MostRecent {
		t.Error("a result with no chapters looked up can't be the most recent")
	}
}
