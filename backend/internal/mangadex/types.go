package mangadex

type Manga struct {
	ID            string           `json:"id"`
	Attributes    MangaAttributes  `json:"attributes"`
	Relationships []MangaRelations `json:"relationships"`
}

type MangaAttributes struct {
	Title  Title  `json:"title"`
	Status string `json:"status"`
}

type Title struct {
	En string `json:"en"`
}

type MangaRelations struct {
	Type       string             `json:"type"`
	Attributes RelationAttributes `json:"attributes"`
}

type RelationAttributes struct {
	FileName string `json:"fileName"`
}
