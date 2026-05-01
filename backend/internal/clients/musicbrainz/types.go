package musicbrainz

// Artist mirrors the subset of MusicBrainz's artist payload used by MediaHub.
// ID is the MusicBrainz Identifier (MBID), a stable UUID for the entity.
type Artist struct {
	ID      string
	Name    string
	Type    string
	Country string
	Score   int
}

// Release represents an album-level entity. Tracks is populated only by lookup,
// not by search; search results return TrackCount as a hint.
type Release struct {
	ID         string
	Title      string
	Artist     string
	Date       string
	TrackCount int
	Tracks     []Track
}

type Track struct {
	ID           string
	Title        string
	TrackNumber  int
	LengthMillis int
}
