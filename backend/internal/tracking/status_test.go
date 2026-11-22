package tracking

import "testing"

func TestValidStatus(t *testing.T) {
	for _, status := range []string{"current", "planned", "completed", "dropped"} {
		if !ValidStatus(status) {
			t.Errorf("%q should be a list status", status)
		}
	}
	// the wishlist is kept apart from the list, so it can't replace a status
	for _, status := range []string{"wishlist", "", "watching"} {
		if ValidStatus(status) {
			t.Errorf("%q shouldn't be a list status", status)
		}
	}
}
