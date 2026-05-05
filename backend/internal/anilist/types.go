package anilist

// Media is the public, domain-shaped representation of an Anilist entry.
// JSON tags use snake_case to stay consistent with the rest of the API surface.
// Format-specific fields (Episodes, Chapters, Volumes) are nil when not applicable.
type Media struct {
	ID          int        `json:"id"`
	Type        string     `json:"type"`
	Format      string     `json:"format"`
	Status      string     `json:"status"`
	Title       Title      `json:"title"`
	Description *string    `json:"description"`
	Episodes    *int       `json:"episodes"`
	Chapters    *int       `json:"chapters"`
	Volumes     *int       `json:"volumes"`
	CoverImage  CoverImage `json:"cover_image"`
	Genres      []string   `json:"genres"`
	Studios     []string   `json:"studios"`
	StartDate   FuzzyDate  `json:"start_date"`
}

// Title carries Anilist's three localised title variants. Any may be empty.
// Field names already match Anilist's JSON keys, so the same struct serves both
// inbound decoding and outbound encoding.
type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}

type CoverImage struct {
	Large  string `json:"large"`
	Medium string `json:"medium"`
}

// FuzzyDate maps Anilist's partial-date type; any field may be nil when unknown.
type FuzzyDate struct {
	Year  *int `json:"year"`
	Month *int `json:"month"`
	Day   *int `json:"day"`
}

// wireMedia mirrors Anilist's GraphQL response field-for-field, including camelCase
// keys and the nested studios.nodes shape. Unexported because callers should only
// ever see the cleaned-up Media type, never the raw form.
type wireMedia struct {
	ID          int     `json:"id"`
	Type        string  `json:"type"`
	Format      string  `json:"format"`
	Status      string  `json:"status"`
	Title       Title   `json:"title"`
	Description *string `json:"description"`
	Episodes    *int    `json:"episodes"`
	Chapters    *int    `json:"chapters"`
	Volumes     *int    `json:"volumes"`
	CoverImage  struct {
		Large  string `json:"large"`
		Medium string `json:"medium"`
	} `json:"coverImage"`
	Genres  []string `json:"genres"`
	Studios struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"studios"`
	StartDate FuzzyDate `json:"startDate"`
}

/*
	Function:	toDomain
	Purpose:	Flatten a raw Anilist response entry into the public Media shape,
				collapsing studios.nodes[].name into a flat []string and renaming
				camelCase fields.
*/
func (w wireMedia) toDomain() Media {
	studios := make([]string, 0, len(w.Studios.Nodes))
	for _, s := range w.Studios.Nodes {
		studios = append(studios, s.Name)
	}
	return Media{
		ID:          w.ID,
		Type:        w.Type,
		Format:      w.Format,
		Status:      w.Status,
		Title:       w.Title,
		Description: w.Description,
		Episodes:    w.Episodes,
		Chapters:    w.Chapters,
		Volumes:     w.Volumes,
		CoverImage:  CoverImage{Large: w.CoverImage.Large, Medium: w.CoverImage.Medium},
		Genres:      w.Genres,
		Studios:     studios,
		StartDate:   w.StartDate,
	}
}
