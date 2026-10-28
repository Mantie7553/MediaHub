package mangadex

import "strconv"

type Manga struct {
	ID            string           `json:"id"`
	Attributes    MangaAttributes  `json:"attributes"`
	Relationships []MangaRelations `json:"relationships"`
}

// Genres: the names of the manga's genre tags (MangaDex also tags themes and formats)
func (m Manga) Genres() []string {
	var genres []string
	for _, tag := range m.Attributes.Tags {
		if tag.Attributes.Group == "genre" {
			genres = append(genres, tag.Attributes.Name.En)
		}
	}
	return genres
}

// TotalChapters: the last chapter's number, 0 when MangaDex doesn't know it
func (m Manga) TotalChapters() int {
	n, _ := strconv.Atoi(m.Attributes.LastChapter)
	return n
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

type DiscoveryResult struct {
	Trending []Manga
	Popular  []Manga
	Latest   []Manga
}
