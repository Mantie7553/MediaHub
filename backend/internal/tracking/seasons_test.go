package tracking

import "testing"

// shaped like Monogatari: two entries in season 1, one in season 2, a special last
var monogatari = []ShowEntry{
	{AnilistID: 5081, Season: 1, Episodes: 15},  // Bakemonogatari
	{AnilistID: 11597, Season: 1, Episodes: 11}, // Nisemonogatari
	{AnilistID: 17074, Season: 2, Episodes: 26}, // Monogatari Second Season
	{AnilistID: 15689, Season: 0, Episodes: 4},  // Nekomonogatari: Kuro
}

func TestSeasonProgress(t *testing.T) {
	tests := []struct {
		name       string
		anilistID  int
		status     string
		progress   int
		wantCount  int
		wantStatus string
	}{
		{"the first entry counts as is", 5081, "current", 5, 5, "current"},
		{"finishing an early entry keeps the series current", 5081, "completed", 0, 15, "current"},
		{"a later entry counts every entry before it", 11597, "current", 3, 18, "current"},
		{"finishing the last regular entry finishes the series", 17074, "completed", 0, 52, "completed"},
		{"watching a later entry with none seen yet means the ones before are", 17074, "current", 0, 26, "current"},
		{"progress past the entry's episodes stops at its end", 11597, "current", 40, 26, "current"},
		{"dropping counts what was seen", 17074, "dropped", 4, 30, "dropped"},
		{"planning a later entry moves nothing", 17074, "planned", 0, 0, "planned"},
		{"a wishlisted later entry moves nothing", 11597, "wishlist", 0, 0, "wishlist"},
		{"a special moves nothing and can't finish the series", 15689, "completed", 4, 0, "current"},
		{"an entry outside the series keeps its own progress", 999, "current", 7, 7, "current"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, status := SeasonProgress(monogatari, tt.anilistID, tt.status, tt.progress)
			if count != tt.wantCount || status != tt.wantStatus {
				t.Errorf("got %d %s, want %d %s", count, status, tt.wantCount, tt.wantStatus)
			}
		})
	}
}

func TestMergeStatus(t *testing.T) {
	tests := []struct {
		name       string
		oldCount   int
		oldStatus  string
		newCount   int
		newStatus  string
		wantCount  int
		wantStatus string
	}{
		{"further along wins", 15, "current", 52, "completed", 52, "completed"},
		{"an earlier entry saved later never goes back", 52, "completed", 15, "current", 52, "completed"},
		{"a planned later entry keeps what was watched", 15, "current", 0, "planned", 15, "current"},
		{"the same count takes the further status", 0, "wishlist", 0, "planned", 0, "planned"},
		{"the same count keeps a further status", 0, "planned", 0, "wishlist", 0, "planned"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, status := MergeStatus(tt.oldCount, tt.oldStatus, tt.newCount, tt.newStatus)
			if count != tt.wantCount || status != tt.wantStatus {
				t.Errorf("got %d %s, want %d %s", count, status, tt.wantCount, tt.wantStatus)
			}
		})
	}
}
