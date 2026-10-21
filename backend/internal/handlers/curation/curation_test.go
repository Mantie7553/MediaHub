package curation

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/locks"
)

func TestOptionalTellsNullFromMissing(t *testing.T) {
	var req editMediaRequest
	if err := json.Unmarshal([]byte(`{"description":null,"title":"New"}`), &req); err != nil {
		t.Fatal(err)
	}
	if !req.Description.Set || req.Description.Value != nil {
		t.Errorf("null description = %+v, want set and nil", req.Description)
	}
	if !req.Title.Set || *req.Title.Value != "New" {
		t.Errorf("title = %+v", req.Title)
	}
	if req.Genres.Set {
		t.Error("a missing field must not count as set")
	}
}

func decodeEdit(t *testing.T, body string) editMediaRequest {
	t.Helper()
	var req editMediaRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return req
}

func TestMediaChanges(t *testing.T) {
	item, meta, edited, err := mediaChanges(decodeEdit(t,
		`{"title":"  Frieren  ","description":"","release_date":"2023-09-29","genres":["Fantasy"," fantasy ","Drama",""],"studio":"Madhouse","status":"airing"}`),
		"anime")
	if err != nil {
		t.Fatal(err)
	}
	if item[0].column != "title" || item[0].value != "Frieren" {
		t.Errorf("title = %+v, want trimmed", item[0])
	}
	if item[1].column != "description" || item[1].value != (*string)(nil) {
		t.Errorf("an empty description is stored as NULL, got %+v", item[1])
	}
	wantEdited := []string{locks.Title, locks.Description, locks.ReleaseDate, locks.Genres, locks.Status, locks.Studio}
	if !reflect.DeepEqual(edited, wantEdited) {
		t.Errorf("edited = %v, want %v", edited, wantEdited)
	}
	if len(meta) != 3 {
		t.Errorf("meta = %+v, want genres, status and studio", meta)
	}
}

func TestMediaChangesRefusesBadValues(t *testing.T) {
	tests := []struct {
		name, body, mediaType, want string
	}{
		{"empty title", `{"title":"  "}`, "anime", "title"},
		{"bad date", `{"release_date":"29/09/2023"}`, "movie", "YYYY-MM-DD"},
		{"field of another type", `{"director":"Someone"}`, "anime", "has no director"},
		{"anime status on manga", `{"status":"airing"}`, "manga", "ongoing"},
		{"negative count", `{"total_chapters":-1}`, "manga", "between 0"},
		{"track without artist", `{"artist":""}`, "music_track", "artist"},
		{"nothing", `{}`, "movie", "nothing to change"},
		{"too many genres", `{"genres":[` + strings.Repeat(`"g",`, 0) + genresJSON(maxGenres+1) + `]}`, "anime", "genres"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := mediaChanges(decodeEdit(t, tt.body), tt.mediaType)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want one mentioning %q", err, tt.want)
			}
		})
	}
}

func genresJSON(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = `"genre` + string(rune('a'+i%26)) + string(rune('a'+i/26)) + `"`
	}
	return strings.Join(parts, ",")
}

func TestMediaChangesClearsOptionalNumbers(t *testing.T) {
	_, meta, _, err := mediaChanges(decodeEdit(t, `{"runtime_mins":null}`), "movie")
	if err != nil {
		t.Fatal(err)
	}
	if len(meta) != 1 || meta[0].column != "runtime_mins" || meta[0].value != (*int)(nil) {
		t.Errorf("meta = %+v, want runtime_mins cleared", meta)
	}
}

func TestNextLocks(t *testing.T) {
	got := nextLocks([]string{"title", "cover"}, []string{"genres", "title"}, []string{"cover"})
	want := []string{"genres", "title"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nextLocks = %v, want %v", got, want)
	}
	if got := nextLocks(nil, nil, nil); got == nil || len(got) != 0 {
		t.Errorf("no locks should be an empty list, not NULL: %#v", got)
	}
}

func TestUpdateQuery(t *testing.T) {
	q, args := updateQuery("media_items", "id-1", []assignment{{"title", "A"}, {"description", nil}})
	if q != "UPDATE media_items SET title = $2, description = $3" {
		t.Errorf("query = %q", q)
	}
	if len(args) != 3 || args[0] != "id-1" || args[1] != "A" {
		t.Errorf("args = %v", args)
	}
}

func TestSeriesAndSeasonFolders(t *testing.T) {
	tv := filepath.Join("media", "TV Shows")
	series := filepath.Join(tv, "Frieren (2023)")
	existing := []string{
		filepath.Join(series, "Season 01", "Frieren - S01E01.mkv"),
		filepath.Join(series, "Specials", "Frieren - S00E01.mkv"),
	}

	if got := seriesFolder(tv, "Frieren", existing); got != series {
		t.Errorf("seriesFolder = %q, want the episodes' folder %q", got, series)
	}
	if got := seriesFolder(tv, "Frieren: Beyond", nil); got != filepath.Join(tv, "Frieren Beyond") {
		t.Errorf("a new series folder = %q", got)
	}
	// a file outside the TV folder isn't trusted as the series folder
	outside := []string{filepath.Join("elsewhere", "Season 01", "x.mkv")}
	if got := seriesFolder(tv, "Frieren", outside); got != filepath.Join(tv, "Frieren") {
		t.Errorf("seriesFolder with an outside file = %q", got)
	}

	if got := seasonFolder(series, 1, existing); got != filepath.Join(series, "Season 01") {
		t.Errorf("season 1 = %q", got)
	}
	if got := seasonFolder(series, 0, existing); got != filepath.Join(series, "Specials") {
		t.Errorf("season 0 = %q", got)
	}
	if got := seasonFolder(series, 2, existing); got != filepath.Join(series, "Season 02") {
		t.Errorf("a new season = %q", got)
	}
	// Sonarr set to unpadded season folders
	unpadded := []string{filepath.Join(series, "Season 3", "x.mkv")}
	if got := seasonFolder(series, 3, unpadded); got != filepath.Join(series, "Season 3") {
		t.Errorf("season 3 = %q, want the existing unpadded folder", got)
	}
}

func TestEpisodeFileName(t *testing.T) {
	if got := episodeFileName("Frieren", 1, 5, "Phantoms of the Dead", ".mkv"); got != "Frieren - S01E05 - Phantoms of the Dead.mkv" {
		t.Errorf("got %q", got)
	}
	if got := episodeFileName("Show", 1, 105, "What?", ".mp4"); got != "Show - S01E105 - What.mp4" {
		t.Errorf("got %q", got)
	}
	if got := episodeFileName("Show", 2, 3, "", ".mkv"); got != "Show - S02E03.mkv" {
		t.Errorf("got %q", got)
	}
}

func TestParseChapterNumber(t *testing.T) {
	s := func(v string) *string { return &v }
	if c, ok := parseChapterNumber(s(" 098 ")); !ok || c.Number != "98" || c.Value != 98 {
		t.Errorf("098 = %+v, %v", c, ok)
	}
	if c, ok := parseChapterNumber(s("52.5")); !ok || c.Number != "52.5" {
		t.Errorf("52.5 = %+v, %v", c, ok)
	}
	for _, bad := range []*string{nil, s(""), s("-1"), s("twelve")} {
		if _, ok := parseChapterNumber(bad); ok {
			t.Errorf("%v should be refused", bad)
		}
	}
}
