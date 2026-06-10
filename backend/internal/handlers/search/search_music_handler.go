package search

import (
	"net/http"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

/*
Function:	YTSearch
Purpose:	Search YouTube Music via yt-dlp for tracks
Params:
  - w: http response writer
  - r: http request with query param ?q=
*/
func (h *Handler) YTSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	l := r.URL.Query().Get("limit")
	if q == "" {
		utils.Error(w, http.StatusBadRequest, "q is required")
		return
	}

	limit, err := strconv.Atoi(l)
	if err != nil || (limit < 5 || limit > 25) {
		limit = 15
	}

	results, err := h.yt.SearchMusic(q, limit)
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, results)
}
