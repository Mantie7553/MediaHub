package locks

import "testing"

func TestKeep(t *testing.T) {
	got := Keep(Title, "title", "$1")
	want := "title = CASE WHEN 'title' = ANY(media_items.locked_fields) THEN media_items.title ELSE $1 END"
	if got != want {
		t.Errorf("Keep = %q\nwant %q", got, want)
	}
}

func TestKeepMeta(t *testing.T) {
	got := KeepMeta(Genres, "anime_metadata", "genres", "EXCLUDED.genres")
	want := "genres = CASE WHEN (SELECT 'genres' = ANY(locked_fields) FROM media_items WHERE id = anime_metadata.media_item_id) THEN anime_metadata.genres ELSE EXCLUDED.genres END"
	if got != want {
		t.Errorf("KeepMeta = %q\nwant %q", got, want)
	}
}

func TestUnknownFieldPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("an unknown field must not reach SQL")
		}
	}()
	Keep("title'; DROP TABLE users; --", "title", "$1")
}
