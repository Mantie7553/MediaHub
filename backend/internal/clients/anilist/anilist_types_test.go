package anilist

import "testing"

func TestLibraryStatus(t *testing.T) {
	for status, want := range map[string]string{
		"RELEASING": "airing", "FINISHED": "finished", "NOT_YET_RELEASED": "upcoming", "CANCELLED": "",
	} {
		if got := (Media{Status: status}).LibraryStatus(); got != want {
			t.Errorf("LibraryStatus() for %s = %q, want %q", status, got, want)
		}
	}
}
