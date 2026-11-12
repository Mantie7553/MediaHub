package tracking

import "testing"

func TestNextStatus(t *testing.T) {
	tests := []struct {
		name        string
		done, total int
		current     *string
		want        string
		change      bool
	}{
		{"a title with no parts is left alone", 0, 0, new("current"), "", false},
		{"everything done completes it", 5, 5, new("current"), "completed", true},
		{"everything done adds it to the list as completed", 5, 5, nil, "completed", true},
		{"already completed doesn't change", 5, 5, new("completed"), "completed", false},
		{"some done is current", 2, 5, new("planned"), "current", true},
		{"some done from completed goes back to current", 4, 5, new("completed"), "current", true},
		{"some done picks a dropped title back up", 1, 5, new("dropped"), "current", true},
		{"some done adds it to the list as current", 1, 5, nil, "current", true},
		{"nothing done takes current back to planned", 0, 5, new("current"), "planned", true},
		{"nothing done takes completed back to planned", 0, 5, new("completed"), "planned", true},
		{"nothing done leaves dropped alone", 0, 5, new("dropped"), "", false},
		{"nothing done leaves planned alone", 0, 5, new("planned"), "", false},
		{"nothing done doesn't add it to the list", 0, 5, nil, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, change := nextStatus(tt.done, tt.total, tt.current)
			if change != tt.change || (change && got != tt.want) {
				t.Fatalf("nextStatus(%d, %d) = (%q, %v), want (%q, %v)", tt.done, tt.total, got, change, tt.want, tt.change)
			}
		})
	}
}

func TestProgressCountQueries(t *testing.T) {
	tests := []struct {
		mediaType string
		want      []string
	}{
		{"manga", []string{setChaptersRead, setChaptersUnread}},
		{"light_novel", []string{setVolumesRead, setVolumesUnread}},
		{"anime", []string{setEpisodesWatched, setEpisodesUnwatched}},
		{"movie", nil},
		{"music_track", nil},
	}
	for _, tt := range tests {
		t.Run(tt.mediaType, func(t *testing.T) {
			got := progressCountQueries(tt.mediaType)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d queries, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("query %d differs", i)
				}
			}
		})
	}
}

func TestPartsTotal(t *testing.T) {
	tests := []struct {
		name  string
		parts Parts
		want  int
	}{
		{"the source has more than the library", Parts{Files: 20, Source: 100}, 100},
		{"the library has more than the source says", Parts{Files: 12, Source: 10}, 12},
		{"an unknown source total goes by the files", Parts{Files: 7}, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.parts.Total(); got != tt.want {
				t.Errorf("Total() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPartsCountHolds(t *testing.T) {
	tests := []struct {
		name  string
		parts Parts
		count int
		want  bool
	}{
		{"every file read, the count goes past them", Parts{Done: 20, Files: 20, Source: 100}, 50, true},
		{"one file inside the count unread", Parts{Done: 19, Files: 20, Source: 100}, 50, false},
		{"the count is inside the files and they're done", Parts{Done: 10, Files: 20}, 10, true},
		{"a file inside a small count unread", Parts{Done: 9, Files: 20}, 10, false},
		{"reading past the count keeps it", Parts{Done: 15, Files: 20}, 10, true},
		{"no files yet", Parts{Source: 100}, 30, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.parts.CountHolds(tt.count); got != tt.want {
				t.Errorf("CountHolds(%d) = %v, want %v", tt.count, got, tt.want)
			}
		})
	}
}
