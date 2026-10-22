package lists

import (
	"database/sql"
	"testing"
)

func str(s string) sql.NullString    { return sql.NullString{String: s, Valid: true} }
func num(n int64) sql.NullInt64      { return sql.NullInt64{Int64: n, Valid: true} }
func secs(f float64) sql.NullFloat64 { return sql.NullFloat64{Float64: f, Valid: true} }
func flag(b bool) sql.NullBool       { return sql.NullBool{Bool: b, Valid: true} }

func TestContinueLabel(t *testing.T) {
	tests := []struct {
		name string
		p    continueProgress
		want string
	}{
		{"one season names only the episode", continueProgress{Kind: "anime", Season: num(1), Seasons: num(1), Number: str("5")}, "E5"},
		{"several seasons name the season", continueProgress{Kind: "anime", Season: num(2), Seasons: num(3), Number: str("1")}, "S2 · E1"},
		{"a movie says the minutes left, rounded up", continueProgress{Kind: "movie", PositionSecs: secs(3000), DurationSecs: secs(5530)}, "43 min left"},
		{"a chapter being read gives the page, counted from 1", continueProgress{Kind: "manga", Number: str("12"), Page: num(5), Completed: flag(false)}, "Ch 12 · p. 6"},
		{"a finished chapter points at the next", continueProgress{Kind: "manga", Number: str("12"), Completed: flag(true), NextNumber: str("12.5")}, "Next: Ch 12.5"},
		{"chapter numbers lose trailing zeros", continueProgress{Kind: "manga", Number: str("7.0"), Completed: flag(false)}, "Ch 7"},
		{"a volume being read gives how far in", continueProgress{Kind: "light_novel", Number: str("2"), Scroll: secs(0.354), Completed: flag(false)}, "Vol 2 · 35%"},
		{"an opened volume not scrolled yet", continueProgress{Kind: "light_novel", Number: str("2"), Scroll: secs(0), Completed: flag(false)}, "Vol 2"},
		{"a finished volume points at the next", continueProgress{Kind: "light_novel", Number: str("2"), Completed: flag(true), NextNumber: str("3")}, "Next: Vol 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.label()
			if got == nil || *got != tt.want {
				t.Fatalf("label() = %v, want %q", deref(got), tt.want)
			}
		})
	}
}

func TestContinueLabelWithoutEnoughToSay(t *testing.T) {
	tests := map[string]continueProgress{
		"a movie of unknown length":                   {Kind: "movie", PositionSecs: secs(600)},
		"an anime without a next episode":             {Kind: "anime", Seasons: num(1)},
		"a finished chapter with only earlier unread": {Kind: "manga", Number: str("12"), Completed: flag(true)},
		"an unknown kind":                             {Kind: "music_album"},
	}
	for name, p := range tests {
		t.Run(name, func(t *testing.T) {
			if got := p.label(); got != nil {
				t.Fatalf("label() = %q, want none", *got)
			}
		})
	}
}

func TestContinuePct(t *testing.T) {
	tests := []struct {
		name string
		p    continueProgress
		want *int
	}{
		{"a movie by position", continueProgress{Kind: "movie", PositionSecs: secs(1800), DurationSecs: secs(5400)}, intPtr(33)},
		{"a series by finished parts", continueProgress{Kind: "anime", Done: num(3), Total: num(12)}, intPtr(25)},
		{"nothing to measure against", continueProgress{Kind: "movie", PositionSecs: secs(1800)}, nil},
		{"no parts with files", continueProgress{Kind: "manga", Done: num(0), Total: num(0)}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.pct()
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Fatalf("pct() = %v, want %v", derefInt(got), derefInt(tt.want))
			}
		})
	}
}

func intPtr(n int) *int { return &n }

func deref(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func derefInt(n *int) any {
	if n == nil {
		return nil
	}
	return *n
}
