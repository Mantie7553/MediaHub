package sync

import (
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/animemerge"
	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

type mergeRequest struct {
	// the series to merge, as GetAnimeMerges lists them
	TvdbIDs []int `json:"tvdb_ids"`
}

// whether a full library sync is running, which a merge must not race
func (h *Handler) syncRunning() bool {
	h.runner.mu.Lock()
	defer h.runner.mu.Unlock()
	return h.runner.status.Running
}

/*
Function:	GetAnimeMerges
Purpose:	the anime series with more than one title (seasons saved as titles of their own before
titles followed Sonarr), and the title each would keep. A dry run: changes nothing
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the front end
*/
func (h *Handler) GetAnimeMerges(w http.ResponseWriter, r *http.Request) {
	groups, err := animemerge.Find(h.db, arr.AnimeShows())
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, groups)
}

/*
Function:	MergeAnime
Purpose:	merge the series asked for into one title each, carrying everyone's progress, status and
ratings over. Refused while a library sync is running
Params:
  - w: http response writer to respond to the front end
  - r: http request with the series' TVDB ids
*/
func (h *Handler) MergeAnime(w http.ResponseWriter, r *http.Request) {
	var req mergeRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if len(req.TvdbIDs) == 0 {
		utils.Error(w, http.StatusBadRequest, "pick at least one series to merge")
		return
	}
	if h.syncRunning() {
		utils.Error(w, http.StatusConflict, "a library sync is running, merge once it's done")
		return
	}
	results, err := animemerge.Merge(h.db, arr.AnimeShows(), req.TvdbIDs)
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, results)
}
