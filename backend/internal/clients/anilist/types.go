package anilist

// Media mirrors the subset of Anilist's Media type used by MediaHub.
// Both anime and manga responses share this shape; format-specific fields
// (Episodes, Chapters, Volumes) are nil when not applicable.
type Media struct {
	ID          int
	Type        string
	Format      string
	Status      string
	Title       Title
	Description *string
	Episodes    *int
	Chapters    *int
	Volumes     *int
	CoverImage  CoverImage
	Genres      []string
	Studios     []string
	StartDate   FuzzyDate
}

// Title carries Anilist's three localised title variants. Any may be empty.
type Title struct {
	Romaji  string
	English string
	Native  string
}

type CoverImage struct {
	Large  string
	Medium string
}

// FuzzyDate maps Anilist's partial-date type; any field may be nil when unknown.
type FuzzyDate struct {
	Year  *int
	Month *int
	Day   *int
}
