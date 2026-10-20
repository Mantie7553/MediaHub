package search

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/anilist"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/mangadex"
)

func TestFromMangaDexUsesThumbnails(t *testing.T) {
	m := mangadex.Manga{ID: "abc"}
	m.Attributes.Title.JaRo = "Sousou no Frieren"
	m.Relationships = []mangadex.MangaRelations{{Type: "author"}, {Type: "cover_art"}}
	m.Relationships[1].Attributes.FileName = "cover.png"

	got := fromMangaDex([]mangadex.Manga{m, {ID: "none"}})
	if want := "https://uploads.mangadex.org/covers/abc/cover.png.512.jpg"; got[0].CoverImageURL != want {
		t.Errorf("cover = %q, want %q", got[0].CoverImageURL, want)
	}
	if got[0].Title != "Sousou no Frieren" {
		t.Errorf("title = %q, want the romaji title when there is no English one", got[0].Title)
	}
	if got[1].CoverImageURL != "" {
		t.Errorf("cover = %q, want none for a manga without cover art", got[1].CoverImageURL)
	}
}

func TestFromAnilistPrefersEnglishTitles(t *testing.T) {
	a, b := anilist.Media{ID: 1}, anilist.Media{ID: 2}
	a.Title.English, a.Title.Romaji = "Frieren", "Sousou no Frieren"
	b.Title.Romaji = "Dandadan"

	got := fromAnilist([]anilist.Media{a, b}, "movie")
	if got[0].Title != "Frieren" || got[1].Title != "Dandadan" {
		t.Errorf("titles = %q, %q", got[0].Title, got[1].Title)
	}
	if got[0].ExternalID != "1" || got[0].Type != "movie" || got[0].ExternalSource != "anilist" {
		t.Errorf("result = %+v", got[0])
	}
}

func TestSearchKeyIgnoresCaseAndSpacing(t *testing.T) {
	if searchKey("anime", "  Frieren   Beyond ") != searchKey("anime", "frieren beyond") {
		t.Error("case and spacing should not make a different search")
	}
	if searchKey("anime", "frieren") == searchKey("manga", "frieren") {
		t.Error("the media type is part of the search")
	}
}

func TestSearchRejectsUnknownTypes(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(nil).Search(rec, httptest.NewRequest(http.MethodGet, "/api/search?type=book", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
