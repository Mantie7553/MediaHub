package musicbrainz

// Library represents a single Plex library section (one row in /library/sections).
type Library struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// Item represents a metadata entry returned by Plex (movie, episode, track, etc.).
// ParentRatingKey points to the season or album; GrandparentRatingKey to the show or artist.
type Item struct {
	RatingKey            string `json:"rating_key"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	Summary              string `json:"summary"`
	Year                 int    `json:"year"`
	Thumb                string `json:"thumb"`
	Art                  string `json:"art"`
	DurationMillis       int    `json:"duration_millis"`
	AddedAt              int64  `json:"added_at"`
	ParentRatingKey      string `json:"parent_rating_key"`
	GrandparentRatingKey string `json:"grandparent_rating_key"`
}
