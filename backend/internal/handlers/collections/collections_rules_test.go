package collections

import (
	"errors"
	"testing"
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

func TestOnlyTheOwnerHasAccess(t *testing.T) {
	a := access{ownerID: "owner"}
	if !a.canView("owner") || !a.canEdit("owner") {
		t.Error("the owner should see and change it")
	}
	if a.canView("someone-else") || a.canEdit("someone-else") {
		t.Error("nobody else should, until sharing exists")
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
