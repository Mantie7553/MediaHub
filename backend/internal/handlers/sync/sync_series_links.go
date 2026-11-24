package sync

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Mantie7553/MediaHub/backend/internal/animemerge"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// a series an anime title can be attached to
type animeSeries struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	TvdbID int    `json:"tvdb_id"`
}

/*
Function:	GetAnimeSeries
Purpose:	the anime titles that stand for a TVDB series, the ones a title can be attached to
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the front end
*/
func (h *Handler) GetAnimeSeries(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`SELECT id, title, tvdb_id FROM media_items
		WHERE type = 'anime' AND tvdb_id IS NOT NULL ORDER BY lower(title)`)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()
	out := []animeSeries{}
	for rows.Next() {
		var s animeSeries
		if utils.InternalError(w, rows.Scan(&s.ID, &s.Title, &s.TvdbID)) {
			return
		}
		out = append(out, s)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}
	utils.JSON(w, out)
}

type attachRequest struct {
	TvdbID int `json:"tvdb_id"`
	Season int `json:"season"`
}

// the attached title, as attaching needs it
type attachSource struct {
	title, kind, externalID, externalSource string
	tvdb                                    bool
	sonarr                                  bool
}

/*
Function:	AttachToSeries
Purpose:	place an anime title's AniList entry in a TVDB series by hand, for entries Fribb's list
leaves out of their series, then merge the title into the series' one. The link stays, so saving
or syncing the entry later lands on the series too
Params:
  - w: http response writer to respond to the front end
  - r: http request with the title's id in the URL and the series and season in the body
*/
func (h *Handler) AttachToSeries(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req attachRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if req.TvdbID <= 0 || req.Season < 0 {
		utils.Error(w, http.StatusBadRequest, "pick a series and a season (0 for specials)")
		return
	}
	if h.syncRunning() {
		utils.Error(w, http.StatusConflict, "a library sync is running, attach once it's done")
		return
	}

	var src attachSource
	err := h.db.QueryRow(`SELECT mi.title, mi.type, COALESCE(mi.external_id, ''),
		COALESCE(mi.external_source::text, ''), mi.tvdb_id IS NOT NULL,
		EXISTS (SELECT 1 FROM sonarr_items si WHERE si.media_item_id = mi.id)
		FROM media_items mi WHERE mi.id = $1`, id).
		Scan(&src.title, &src.kind, &src.externalID, &src.externalSource, &src.tvdb, &src.sonarr)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "title not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	anilistID, convErr := strconv.Atoi(src.externalID)
	if src.kind != "anime" || src.externalSource != "anilist" || convErr != nil || anilistID <= 0 {
		utils.Error(w, http.StatusBadRequest, "only an anime title saved from AniList can be attached to a series")
		return
	}
	if src.sonarr || src.tvdb {
		utils.Error(w, http.StatusConflict, "this title is already part of a series")
		return
	}
	var seriesID string
	err = h.db.QueryRow(`SELECT id FROM media_items WHERE type = 'anime' AND tvdb_id = $1`, req.TvdbID).Scan(&seriesID)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "that series isn't in the library")
		return
	}
	if utils.InternalError(w, err) {
		return
	}

	// the series Fribb had the entry in, if any, is built again without it
	before, _ := arr.LoadAnimeMaps()
	old := before.AnilistToShow[anilistID].TvdbID

	_, err = h.db.Exec(`INSERT INTO anime_series_links (anilist_id, tvdb_id, season_number, title)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (anilist_id) DO UPDATE SET tvdb_id = EXCLUDED.tvdb_id,
		season_number = EXCLUDED.season_number, title = EXCLUDED.title, created_at = NOW()`,
		anilistID, req.TvdbID, req.Season, src.title)
	if utils.InternalError(w, err) {
		return
	}
	if !h.reloadLinks(w, old, req.TvdbID) {
		return
	}

	results, err := animemerge.Merge(h.db, arr.AnimeShows(), []int{req.TvdbID})
	if utils.InternalError(w, err) {
		return
	}
	for _, res := range results {
		if res.TvdbID != req.TvdbID {
			continue
		}
		if res.Error != "" {
			// the link is kept: the merge can be run again from Settings
			utils.Error(w, http.StatusBadGateway, "linked, but the merge failed: "+res.Error)
			return
		}
		logger.Info("attached %s (AniList %d) to TVDB series %d as season %d", src.title, anilistID, req.TvdbID, req.Season)
		utils.JSON(w, res)
		return
	}
	utils.Error(w, http.StatusInternalServerError, "linked, but the title wasn't found in the series to merge")
}

// reloadLinks lays the links as they now are over Fribb's list and drops the series they changed
func (h *Handler) reloadLinks(w http.ResponseWriter, tvdbIDs ...int) bool {
	if _, err := animemerge.LoadLinks(h.db); utils.InternalError(w, err) {
		return false
	}
	arr.AnimeShows().Forget(tvdbIDs...)
	return true
}

// a series link as the Settings page lists it
type seriesLinkView struct {
	arr.SeriesLink
	// the title it was attached from
	Title string `json:"title"`
	// the series' title in the library, empty when it's gone
	Series string `json:"series"`
}

/*
Function:	GetSeriesLinks
Purpose:	the anime entries an admin attached to a series by hand
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the front end
*/
func (h *Handler) GetSeriesLinks(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`SELECT l.anilist_id, l.tvdb_id, l.season_number, l.title, COALESCE(mi.title, '')
		FROM anime_series_links l
		LEFT JOIN media_items mi ON mi.type = 'anime' AND mi.tvdb_id = l.tvdb_id
		ORDER BY lower(COALESCE(mi.title, l.title)), l.season_number`)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()
	out := []seriesLinkView{}
	for rows.Next() {
		var v seriesLinkView
		if utils.InternalError(w, rows.Scan(&v.AnilistID, &v.TvdbID, &v.Season, &v.Title, &v.Series)) {
			return
		}
		out = append(out, v)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}
	utils.JSON(w, out)
}

/*
Function:	DeleteSeriesLink
Purpose:	forget a series link. Only later saves and syncs change: a title already merged into the
series stays merged
Params:
  - w: http response writer to respond to the front end
  - r: http request with the AniList id in the URL
*/
func (h *Handler) DeleteSeriesLink(w http.ResponseWriter, r *http.Request) {
	anilistID, err := strconv.Atoi(chi.URLParam(r, "anilist_id"))
	if err != nil || anilistID <= 0 {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	var tvdb int
	err = h.db.QueryRow(`DELETE FROM anime_series_links WHERE anilist_id = $1 RETURNING tvdb_id`, anilistID).Scan(&tvdb)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	// the series Fribb puts the entry in, if any, gets it back
	if !h.reloadLinks(w, tvdb) {
		return
	}
	after, _ := arr.LoadAnimeMaps()
	if at, ok := after.AnilistToShow[anilistID]; ok {
		arr.AnimeShows().Forget(at.TvdbID)
	}
	w.WriteHeader(http.StatusNoContent)
}
