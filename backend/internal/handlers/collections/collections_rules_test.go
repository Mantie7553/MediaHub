package collections

import (
	"errors"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
)

func TestWhatEachKindHolds(t *testing.T) {
	tests := []struct {
		kind, item string
		want       error
	}{
		{kindPlaylist, "music_track", nil},
		{kindPlaylist, "album", nil},
		{kindPlaylist, "anime", errWrongItem},
		{kindPlaylist, kindPlaylist, errWrongItem},
		{kindPlaylist, kindCollection, errWrongItem},
		{kindCollection, "anime", nil},
		{kindCollection, "movie", nil},
		{kindCollection, "manga", nil},
		{kindCollection, "light_novel", nil},
		{kindCollection, "music_track", nil},
		{kindCollection, "album", nil},
		{kindCollection, kindPlaylist, nil},
		{kindCollection, kindCollection, errNestedCollection},
		{kindCollection, "music_album", errWrongItem},
	}
	for _, tt := range tests {
		if got := canHold(tt.kind, tt.item); !errors.Is(got, tt.want) {
			t.Errorf("canHold(%s, %s) = %v, want %v", tt.kind, tt.item, got, tt.want)
		}
	}
}

func TestOnlyAPlaylistTakesAnAlbumAsTracks(t *testing.T) {
	if !expandsToTracks(kindPlaylist, "album") {
		t.Error("an album added to a playlist should add its tracks")
	}
	if expandsToTracks(kindCollection, "album") {
		t.Error("an album added to a collection should stay an album")
	}
	if expandsToTracks(kindPlaylist, "music_track") {
		t.Error("a track is already a track")
	}
}

func TestComputedPlaylistIDs(t *testing.T) {
	for _, id := range []string{smartListened, smartUnlistened} {
		if !isSmart(id) {
			t.Errorf("%q should be a computed playlist", id)
		}
	}
	if isSmart("6f1c2b9e-1d2a-4b3c-8d4e-5f6a7b8c9d0e") {
		t.Error("a stored id isn't computed")
	}
}

func TestArtistPlaylistIDs(t *testing.T) {
	id := artistPrefix + "0123456789abcdef0123456789abcdef"
	if !isArtist(id) || !isSmart(id) {
		t.Errorf("%q should be an artist's playlist, which is computed", id)
	}
	for _, id := range []string{
		smartListened,
		"6f1c2b9e-1d2a-4b3c-8d4e-5f6a7b8c9d0e",
		artistPrefix,
		artistPrefix + "0123456789abcdef",
		artistPrefix + "0123456789ABCDEF0123456789ABCDEF",
		artistPrefix + "0123456789abcdef0123456789abcdeg",
		"0123456789abcdef0123456789abcdef",
	} {
		if isArtist(id) {
			t.Errorf("%q isn't an artist's playlist", id)
		}
	}
	if smartRefusal(id) == smartRefusal(smartListened) {
		t.Error("an artist's playlist fills from the library, not the user's list")
	}
}

func TestPrivateOnesAreTheOwners(t *testing.T) {
	a := access{ownerID: "owner"}
	owner := &auth.UserContext{UserID: "owner", Role: "user"}
	other := &auth.UserContext{UserID: "someone-else", Role: "user"}
	admin := &auth.UserContext{UserID: "admin", Role: "admin"}
	if !a.canView(owner) || !a.canEdit("owner") || !a.canDelete(owner) {
		t.Error("the owner should see, change and delete it")
	}
	if a.canView(other) || a.canEdit("someone-else") || a.canDelete(other) {
		t.Error("nobody else should")
	}
	if a.canView(admin) || a.canDelete(admin) {
		t.Error("admins shouldn't see a private one that isn't waiting for review")
	}
}

func TestAdminsCanOpenOnesInReview(t *testing.T) {
	a := access{ownerID: "owner", inReview: true}
	admin := &auth.UserContext{UserID: "admin", Role: "admin"}
	if !a.canView(admin) {
		t.Error("an admin should be able to look at a submission")
	}
	if a.canEdit("admin") || a.canDelete(admin) {
		t.Error("but not change or delete it")
	}
	if a.canView(&auth.UserContext{UserID: "someone-else", Role: "user"}) {
		t.Error("other users still shouldn't see it")
	}
}

func TestFeaturedOnesAreEveryonesAndNobodysToChange(t *testing.T) {
	a := access{featured: true}
	user := &auth.UserContext{UserID: "u", Role: "user"}
	admin := &auth.UserContext{UserID: "admin", Role: "admin"}
	if !a.canView(user) || !a.canView(admin) {
		t.Error("everyone should see a featured one")
	}
	if a.canEdit("u") || a.canEdit("admin") || a.canEdit("") {
		t.Error("nobody should change a featured one")
	}
	if a.canDelete(user) || !a.canDelete(admin) {
		t.Error("only admins should take one down")
	}
	if (access{featured: true, partOf: true}).canDelete(admin) {
		t.Error("a nested copy goes with its collection, not on its own")
	}
}

func ptr(n int64) *int64 { return &n }

func TestPositionBetween(t *testing.T) {
	tests := []struct {
		name       string
		prev, next *int64
		want       int64
		ok         bool
	}{
		{"into an empty collection", nil, nil, positionStep, true},
		{"on the end", ptr(2048), nil, 2048 + positionStep, true},
		{"between two", ptr(1024), ptr(2048), 1536, true},
		{"to the top", nil, ptr(1024), 512, true},
		{"no gap left between two", ptr(1024), ptr(1025), 0, false},
		{"no gap left at the top", nil, ptr(1), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := positionBetween(tt.prev, tt.next)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Fatalf("positionBetween = %d, %v; want %d, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestNamesAreTrimmedAndLimited(t *testing.T) {
	if name, problem := cleanName("  Road trip  "); name != "Road trip" || problem != "" {
		t.Errorf("cleanName = %q, %q", name, problem)
	}
	if _, problem := cleanName("   "); problem == "" {
		t.Error("a blank name should be refused")
	}
	long := ""
	for i := 0; i < 101; i++ {
		long += "é"
	}
	if _, problem := cleanName(long); problem == "" {
		t.Error("101 characters should be refused, counting characters not bytes")
	}
}

func TestACollectionHoldsAnAnimeWholeOrBySeason(t *testing.T) {
	two := 2
	tests := []struct {
		name                 string
		adding               *int
		hasWhole, hasSeasons bool
		want                 error
	}{
		{"whole into an empty one", nil, false, false, nil},
		{"season into an empty one", &two, false, false, nil},
		{"another season", &two, false, true, nil},
		{"season next to the whole anime", &two, true, false, errWholeHeld},
		{"whole anime next to its seasons", nil, false, true, errSeasonsHeld},
		{"whole anime again", nil, true, false, nil},
	}
	for _, tt := range tests {
		if got := seasonConflict(tt.adding, tt.hasWhole, tt.hasSeasons); !errors.Is(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestOnlyAdminsSetImagesOfTheirOwnOrFeatured(t *testing.T) {
	admin := &auth.UserContext{UserID: "a", Role: "admin"}
	user := &auth.UserContext{UserID: "u", Role: "user"}
	tests := []struct {
		name string
		c    stored
		user *auth.UserContext
		want bool
	}{
		{"admin's own", stored{access: access{ownerID: "a"}}, admin, true},
		{"featured, by an admin", stored{access: access{featured: true}}, admin, true},
		{"someone else's, in review", stored{access: access{ownerID: "u", inReview: true}}, admin, false},
		{"a user's own", stored{access: access{ownerID: "u"}}, user, false},
		{"featured, by a user", stored{access: access{featured: true}}, user, false},
	}
	for _, tt := range tests {
		if got := canSetImage(tt.c, tt.user); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}
