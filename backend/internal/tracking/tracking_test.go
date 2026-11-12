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
