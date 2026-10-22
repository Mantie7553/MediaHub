package collections

import "errors"

// the two kinds, as stored
const (
	kindPlaylist   = "playlist"
	kindCollection = "collection"
)

// the playlists every user has, filled from their list statuses rather than stored
const (
	smartListened   = "listened"
	smartUnlistened = "unlistened"
)

// how far apart new entries are placed, so one can move between two others without renumbering
const positionStep int64 = 1024

// the most characters a name can have, as the database checks
const maxNameLength = 100

var (
	errWrongItem        = errors.New("that can't go in this")
	errNestedCollection = errors.New("a collection can't go inside another collection")
	errSelf             = errors.New("a collection can't go inside itself")
	errSmartNested      = errors.New("the playlists that fill themselves from your list can't go in a collection")
)

/*
Function:	canHold
Purpose:	Whether a collection of one kind may hold an item of another
Params:
  - kind: the collection's kind
  - item: what the item is, as far as the rules care: a media item's type ("music_track",
    "anime", ...), "album", or "playlist" / "collection" for a nested one

Returns:	nil, or why not
*/
func canHold(kind, item string) error {
	switch kind {
	case kindPlaylist:
		// albums are welcome too, but go in as their tracks (see expandsToTracks)
		if item == "music_track" || item == "album" {
			return nil
		}
	case kindCollection:
		switch item {
		case "anime", "movie", "manga", "light_novel", "music_track", "album", kindPlaylist:
			return nil
		case kindCollection:
			// no collection in a collection, so they can never loop
			return errNestedCollection
		}
	}
	return errWrongItem
}

/*
Function:	expandsToTracks
Purpose:	Whether adding an item to a collection of this kind adds its tracks instead of itself.
A playlist only holds tracks, so an album goes in as its downloaded tracks, a copy at that moment
*/
func expandsToTracks(kind, item string) bool {
	return kind == kindPlaylist && item == "album"
}

// whether an id is one of the computed playlists, which can be played but not changed
func isSmart(id string) bool {
	return id == smartListened || id == smartUnlistened
}

/*
access is who may see and change a collection. It's the one place sharing will extend: today a
collection belongs to its owner alone
*/
type access struct {
	ownerID string
}

// whether the user may open it. Anyone else gets a 404, so ids of other people's don't leak
func (a access) canView(userID string) bool {
	return a.ownerID == userID
}

// whether the user may rename it, delete it or change what's in it
func (a access) canEdit(userID string) bool {
	return a.ownerID == userID
}

/*
Function:	positionBetween
Purpose:	Where an entry goes to sit between two others
Params:
  - prev: the position before it, or nil at the top
  - next: the position after it, or nil at the end

Returns:	the position, and false when there's no room left between the two and the collection
has to be renumbered first
*/
func positionBetween(prev, next *int64) (int64, bool) {
	switch {
	case prev == nil && next == nil:
		return positionStep, true
	case prev == nil:
		// to the top: half way to zero, so the gap shrinks but positions stay positive
		if *next <= 1 {
			return 0, false
		}
		return *next / 2, true
	case next == nil:
		return *prev + positionStep, true
	}
	if *next-*prev < 2 {
		return 0, false
	}
	return *prev + (*next-*prev)/2, true
}
