package musicbrainz

// Artist mirrors the subset of MusicBrainz's artist payload used by MediaHub.
// ID is the MusicBrainz Identifier (MBID), a stable UUID for the entity.
type Artist struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Country string `json:"country"`
	Score   int    `json:"score"`
}

// Release represents an album-level entity. Tracks is populated only by lookup,
// not by search; search results return TrackCount as a hint.
type Release struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Date       string  `json:"date"`
	TrackCount int     `json:"track_count"`
	Tracks     []Track `json:"tracks"`
}

type Track struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	TrackNumber  int    `json:"track_number"`
	LengthMillis int    `json:"length_millis"`
}
