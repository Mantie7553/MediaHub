package mangadex

type Manga struct {
	ID            string           `json:"id"`
	Attributes    MangaAttributes  `json:"attributes"`
	Relationships []MangaRelations `json:"relationships"`
}

type MangaAttributes struct {
	Title       Title             `json:"title"`
	Status      string            `json:"status"`
	Tags        []Tag             `json:"tags"`
	LastChapter string            `json:"lastChapter"`
	Description map[string]string `json:"description"`
}

type Title struct {
	En   string `json:"en"`
	JaRo string `json:"ja-ro"`
	Ja   string `json:"ja"`
}

type MangaRelations struct {
	Type       string             `json:"type"`
	Attributes RelationAttributes `json:"attributes"`
}

type RelationAttributes struct {
	FileName string `json:"fileName"`
}

type Tag struct {
	Attributes TagAttributes `json:"attributes"`
}

type TagAttributes struct {
	Name  Name   `json:"name"`
	Group string `json:"group"`
}

type Name struct {
	En string `json:"en"`
}
