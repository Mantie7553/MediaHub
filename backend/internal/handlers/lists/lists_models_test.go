package lists

import (
	"encoding/json"
	"testing"
)

func TestValidRating(t *testing.T) {
	for _, tt := range []struct {
		rating *int
		want   bool
	}{
		{nil, true},
		{new(1), true},
		{new(10), true},
		{new(0), false},
		{new(11), false},
		{new(-3), false},
	} {
		if got := validRating(tt.rating); got != tt.want {
			t.Errorf("validRating(%v) = %v, want %v", tt.rating, got, tt.want)
		}
	}
}

// a status change alone must not touch the rating; a null rating clears it
func TestUpdateRequestKeepsLeftOutFields(t *testing.T) {
	var statusOnly updateRequest
	if err := json.Unmarshal([]byte(`{"status":"completed"}`), &statusOnly); err != nil {
		t.Fatal(err)
	}
	if statusOnly.Rating.Set {
		t.Error("a status change must leave the rating alone")
	}
	if !statusOnly.Status.Set || *statusOnly.Status.Value != "completed" {
		t.Errorf("status = %+v", statusOnly.Status)
	}

	var cleared updateRequest
	if err := json.Unmarshal([]byte(`{"rating":null}`), &cleared); err != nil {
		t.Fatal(err)
	}
	if !cleared.Rating.Set || cleared.Rating.Value != nil {
		t.Errorf("null rating = %+v, want set and nil", cleared.Rating)
	}
	if cleared.Status.Set {
		t.Error("a rating change must leave the status alone")
	}
}
