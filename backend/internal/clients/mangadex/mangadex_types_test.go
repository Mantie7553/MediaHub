package mangadex

import (
	"reflect"
	"testing"
)

func TestMangaGenresKeepsOnlyGenreTags(t *testing.T) {
	tag := func(group, name string) Tag {
		return Tag{Attributes: TagAttributes{Group: group, Name: Name{En: name}}}
	}
	m := Manga{Attributes: MangaAttributes{Tags: []Tag{tag("genre", "Action"), tag("theme", "School"), tag("genre", "Drama")}}}
	if got := m.Genres(); !reflect.DeepEqual(got, []string{"Action", "Drama"}) {
		t.Errorf("Genres() = %v", got)
	}
}

func TestMangaTotalChapters(t *testing.T) {
	for last, want := range map[string]int{"42": 42, "": 0, "12.5": 0} {
		m := Manga{Attributes: MangaAttributes{LastChapter: last}}
		if got := m.TotalChapters(); got != want {
			t.Errorf("TotalChapters() with lastChapter %q = %d, want %d", last, got, want)
		}
	}
}
