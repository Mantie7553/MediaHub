package search

import (
	"net/http"

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
	if q == "" {
		utils.Error(w, http.StatusBadRequest, "q is required")
		return
	}

	results, err := h.yt.SearchMusic(q, 10)
	if utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, results)
}
