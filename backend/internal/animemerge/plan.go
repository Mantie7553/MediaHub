/*
Package animemerge folds anime titles that are seasons of one series into that series' title.
Before titles followed Sonarr (one per TVDB series), saving a season from AniList or a
MyAnimeList import made a title per season: "Bakemonogatari", "Nisemonogatari", ... A merge
keeps one, carries everyone's progress, status and ratings over to it as the series', moves
everything else that points at the others, and deletes them
*/
package animemerge

import (
	"sort"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
)

// Title is an anime title in the library, as the merge sees it
type Title struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// the AniList entry it was saved from, 0 for none
	AnilistID int `json:"anilist_id"`
	// the TVDB season that entry is in, -1 when the title is the whole series or unknown
	Season int `json:"season"`
	// linked to a Sonarr series
	Sonarr bool `json:"sonarr"`
	// has its TVDB id: saved as the series, since titles follow Sonarr
	HasTvdb bool `json:"-"`
	// where its AniList entry is in the series' watch order, len(entries) when it isn't one
	order int
}

// Relative is whether the title's progress counts from the start of its own season, not the
// series': one saved from a season's AniList entry before titles followed Sonarr
func (t Title) Relative() bool {
	return !t.Sonarr && !t.HasTvdb && t.Season >= 0
}

// Group is a series with more than one title in the library
type Group struct {
	TvdbID int    `json:"tvdb_id"`
	Series string `json:"series"`
	// the title the others fold into
	Keep Title   `json:"keep"`
	Drop []Title `json:"drop"`
}

// a library row read for grouping
type row struct {
	ID, Title      string
	ExternalID     string
	ExternalSource string
	TvdbID         int
	Sonarr         bool
}

/*
Function:	groupRows
Purpose:	the series with more than one title: titles with the same TVDB id, or saved from AniList
entries Fribb places in the same series. Grouping never goes by name
Params:
  - rows: every anime title
  - maps: Fribb's list
*/
func groupRows(rows []row, maps arr.AnimeMaps) map[int][]row {
	groups := map[int][]row{}
	for _, r := range rows {
		tvdb := r.TvdbID
		if tvdb == 0 && r.ExternalSource == "anilist" {
			if id, err := strconv.Atoi(r.ExternalID); err == nil {
				tvdb = maps.AnilistToShow[id].TvdbID
			}
		}
		if tvdb > 0 {
			groups[tvdb] = append(groups[tvdb], r)
		}
	}
	for tvdb, g := range groups {
		if len(g) < 2 {
			delete(groups, tvdb)
		}
	}
	return groups
}

/*
Function:	planGroup
Purpose:	which of a series' titles to keep: the one linked to Sonarr, else one already saved as
the series, else the earliest season's
Params:
  - tvdb: the series
  - rows: its titles
  - maps: Fribb's list
*/
func planGroup(tvdb int, rows []row, maps arr.AnimeMaps) Group {
	entries := maps.ShowSeasons[tvdb]
	titles := make([]Title, 0, len(rows))
	for _, r := range rows {
		t := Title{ID: r.ID, Title: r.Title, Sonarr: r.Sonarr, HasTvdb: r.TvdbID > 0, Season: -1, order: len(entries)}
		if r.ExternalSource == "anilist" {
			t.AnilistID, _ = strconv.Atoi(r.ExternalID)
		}
		for i, e := range entries {
			if e.AnilistID == t.AnilistID && t.AnilistID > 0 {
				t.Season, t.order = e.Season, i
			}
		}
		titles = append(titles, t)
	}
	sort.SliceStable(titles, func(i, j int) bool {
		a, b := titles[i], titles[j]
		if a.Sonarr != b.Sonarr {
			return a.Sonarr
		}
		if a.HasTvdb != b.HasTvdb {
			return a.HasTvdb
		}
		return a.order < b.order
	})
	return Group{TvdbID: tvdb, Series: titles[0].Title, Keep: titles[0], Drop: titles[1:]}
}

// Entry is one user's list entry for one of a series' titles
type Entry struct {
	Title  Title
	Status string
	// episodes done: the count given or the episodes watched, whichever is further
	Done int
	// their rating, and whether they picked it (not an average of season ratings)
	Rating      *int
	RatingIsSet bool
}

// SeasonScore is a rating that becomes a rating of one season of the series
type SeasonScore struct {
	Season    int
	AnilistID int
	Rating    int
}

// Outcome is what one user's list has for the series after a merge
type Outcome struct {
	Status string
	Done   int
	// the series' own rating, nil to work it out from the season ratings
	Rating *int
	// ratings of single seasons, from titles that stood for one
	Seasons []SeasonScore
}

/*
Function:	combine
Purpose:	one user's entries for a series' titles as one entry for the series. A title saved from
a season counts that season's progress from where the series' earlier seasons end, and its
rating becomes that season's; a title saved as the series keeps its own. The furthest progress
wins (tracking.MergeStatus), so the order the titles are seen in never matters
Params:
  - entries: the user's entries, at least one
  - show: the series' entries in watch order, with their episode counts
*/
func combine(entries []Entry, show []tracking.ShowEntry) Outcome {
	var out Outcome
	first := true
	for _, e := range entries {
		done, status := e.Done, e.Status
		if e.Title.Relative() {
			done, status = tracking.SeasonProgress(show, e.Title.AnilistID, e.Status, e.Done)
			if e.Rating != nil && e.RatingIsSet {
				out.Seasons = append(out.Seasons, SeasonScore{e.Title.Season, e.Title.AnilistID, *e.Rating})
			}
		} else if e.Rating != nil && e.RatingIsSet && out.Rating == nil {
			out.Rating = e.Rating
		}
		if first {
			out.Done, out.Status, first = done, status, false
		} else {
			out.Done, out.Status = tracking.MergeStatus(out.Done, out.Status, done, status)
		}
	}
	return out
}
