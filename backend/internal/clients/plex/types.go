package plex

// Library represents a single Plex library section (one row in /library/sections).
type Library struct {
	ID    string
	Title string
	Type  string
}

// Item represents a metadata entry returned by Plex (movie, episode, track, etc.).
// ParentRatingKey points to the season or album; GrandparentRatingKey to the show or artist.
type Item struct {
	RatingKey            string
	Title                string
	Type                 string
	Summary              string
	Year                 int
	Thumb                string
	Art                  string
	DurationMillis       int
	AddedAt              int64
	ParentRatingKey      string
	GrandparentRatingKey string
}
