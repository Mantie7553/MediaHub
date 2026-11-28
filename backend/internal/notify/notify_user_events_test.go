package notify

import (
	"slices"
	"testing"
)

func TestSplitRecipients(t *testing.T) {
	// a requester also wishlisting the title is told once, that it's ready; a deleted
	// requester ("") and repeats are dropped
	ready, others := splitRecipients(
		[]string{"u1", "", "u1", "u2"},
		[]string{"u2", "u3", "u1", "u3", ""},
	)
	if !slices.Equal(ready, []string{"u1", "u2"}) {
		t.Errorf("ready = %v, want [u1 u2]", ready)
	}
	if !slices.Equal(others, []string{"u3"}) {
		t.Errorf("others = %v, want [u3]", others)
	}
	if ready, others := splitRecipients(nil, nil); ready != nil || others != nil {
		t.Errorf("nobody: %v, %v", ready, others)
	}
}

func TestNewPartsTitle(t *testing.T) {
	cases := []struct {
		mediaType string
		added     int
		want      string
	}{
		{"anime", 1, "A new episode of X"},
		{"anime", 3, "3 new episodes of X"},
		{"manga", 0, "A new chapter of X"},
		{"light_novel", 2, "2 new volumes of X"},
	}
	for _, c := range cases {
		if got := newPartsTitle("X", partName(c.mediaType), c.added); got != c.want {
			t.Errorf("%s, %d: %q, want %q", c.mediaType, c.added, got, c.want)
		}
	}
	// titles without parts are only ever "ready"
	for _, mediaType := range []string{"movie", "music_track"} {
		if partName(mediaType) != "" {
			t.Errorf("%s has no parts", mediaType)
		}
	}
}

func TestReadyBody(t *testing.T) {
	cases := map[string]string{
		"anime": "Ready to watch", "movie": "Ready to watch", "manga": "Ready to read",
		"light_novel": "Ready to read", "music_track": "Ready to listen to",
	}
	for mediaType, want := range cases {
		if got := readyBody(mediaType); got != want {
			t.Errorf("readyBody(%q) = %q, want %q", mediaType, got, want)
		}
	}
}

func TestUserKeysStayApartFromAdmins(t *testing.T) {
	// resolving the admins' request notification must leave the requester's alone
	if MyRequestKey("r1") == RequestKey("r1") {
		t.Error("a user's request key must differ from the admins'")
	}
	if TitleKey("x") == DownloadingKey("x") || TitleKey("x") == AlbumKey("x") {
		t.Error("title keys must differ from the others")
	}
}

func TestKindsFor(t *testing.T) {
	user, admin := KindsFor(false), KindsFor(true)
	if !slices.Equal(user, UserKinds) {
		t.Errorf("a user's kinds = %v", user)
	}
	for _, k := range append(slices.Clone(UserKinds), AdminKinds...) {
		if !slices.Contains(admin, k) {
			t.Errorf("an admin can't set %s", k)
		}
	}
	if slices.Contains(user, KindApprovalNeeded) {
		t.Error("a user isn't sent admin kinds")
	}
}
