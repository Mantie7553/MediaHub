package search

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
)

// savedSeries is the anime series a save is for, nil show when it isn't one
type savedSeries struct {
	show *arr.Show
	// the AniList entry the save named, which may be any season of the series
	entryID int
	// the save is for the whole series (a grouped search result), not one season of it
	whole bool
	// the TVDB season the named entry is in
	season int
}

/*
Function:	seriesOf
Purpose:	the series an anime save is for, and the request turned to it: the series' name,
cover and first entry, so every season's save finds or makes the same one title
Params:
  - req: the save, changed in place
*/
func (h *Handler) seriesOf(req *saveRequest) savedSeries {
	if req.Type != "anime" || req.ExternalSource != "anilist" || h.series == nil {
		return savedSeries{}
	}
	entryID, err := strconv.Atoi(req.ExternalID)
	if err != nil {
		return savedSeries{}
	}
	show, at, ok := h.series.Of(entryID)
	if !ok {
		return savedSeries{}
	}
	if show.Title != "" {
		req.Title = show.Title
	}
	if show.CoverURL != "" {
		req.CoverImageURL = show.CoverURL
	}
	req.ExternalID = strconv.Itoa(show.AnilistID)
	whole := req.TvdbID == show.TvdbID || len(show.Entries) == 1
	return savedSeries{show: &show, entryID: entryID, whole: whole, season: at.Season}
}

/*
Function:	progress
Purpose:	the series' episodes watched and status a save means: as given for the whole series,
or worked out from the one season it names
*/
func (s savedSeries) progress(req saveRequest) (int, string) {
	given := 0
	if req.Progress != nil {
		given = *req.Progress
	}
	if s.whole {
		return max(given, 0), req.Status
	}
	entries := make([]tracking.ShowEntry, 0, len(s.show.Entries))
	for _, e := range s.show.Entries {
		entries = append(entries, tracking.ShowEntry{AnilistID: e.AnilistID, Season: e.Season, Episodes: e.Episodes})
	}
	return tracking.SeasonProgress(entries, s.entryID, req.Status, given)
}

/*
Function:	saveRating
Purpose:	keep the rating a save came with: a whole series' is the series' own rating, one entry's
is a rating of the season it is in, which the series' rating is worked out from unless the user
picked one
*/
func (s savedSeries) saveRating(q tracking.Rows, userID, mediaItemID string, rating *int) error {
	if s.whole || rating == nil {
		return nil
	}
	return tracking.SetSeasonRating(q, userID, mediaItemID, s.season, s.entryID, rating)
}

// the rating SaveSeriesEntry gets: a whole series' own, none for one entry (see saveRating)
func (s savedSeries) seriesRating(rating *int) *int {
	if s.whole {
		return rating
	}
	return nil
}

/*
Function:	titleOfSeries
Purpose:	the anime title already standing for a series: the one with its TVDB id, else one saved
from any of its entries (noting the TVDB id on it)
Returns:	sql.ErrNoRows when there is none
*/
func (h *Handler) titleOfSeries(show arr.Show) (string, error) {
	var id string
	err := h.db.QueryRow(`SELECT id FROM media_items WHERE type = 'anime' AND tvdb_id = $1`, show.TvdbID).Scan(&id)
	if !errors.Is(err, sql.ErrNoRows) {
		return id, err
	}
	for _, e := range show.Entries {
		err = h.db.QueryRow(
			`SELECT id FROM media_items WHERE type = 'anime' AND external_source = 'anilist' AND external_id = $1`,
			strconv.Itoa(e.AnilistID),
		).Scan(&id)
		if err == nil {
			h.setTvdbID(id, show.TvdbID)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return id, err
		}
	}
	return "", sql.ErrNoRows
}

// setTvdbID notes the series a title is, unless another title already has it
func (h *Handler) setTvdbID(mediaItemID string, tvdbID int) {
	if _, err := h.db.Exec(
		`UPDATE media_items SET tvdb_id = $1 WHERE id = $2 AND type = 'anime' AND tvdb_id IS NULL
		AND NOT EXISTS (SELECT 1 FROM media_items o WHERE o.type = 'anime' AND o.tvdb_id = $1)`,
		tvdbID, mediaItemID,
	); err != nil {
		logger.Error("failed to save TVDB id %d of %s: %s", tvdbID, mediaItemID, err.Error())
	}
}
