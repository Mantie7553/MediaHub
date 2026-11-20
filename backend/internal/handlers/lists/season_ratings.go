package lists

import (
	"net/http"
	"strconv"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/Mantie7553/MediaHub/backend/internal/tracking"
	"github.com/go-chi/chi/v5"
)

type seasonRatingRequest struct {
	// 1 to 10, null clears the season's own rating (its entries' average shows again, if any)
	Rating *int `json:"rating"`
}

/*
Function:	GetSeasonRatings
Purpose:	the user's ratings of an anime series' seasons, each its own or its entries' average
Params:
  - w: http response writer to respond to the front end
  - r: http request with the series' media item id in the URL
*/
func (h *Handler) GetSeasonRatings(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	scores, err := tracking.LoadSeasonScores(h.db, user.UserID, id)
	if utils.InternalError(w, err) {
		return
	}
	if scores == nil {
		scores = []tracking.SeasonScore{}
	}
	utils.JSON(w, scores)
}

/*
Function:	RateSeason
Purpose:	rate one season of an anime series, or clear its rating. The series' rating follows the
seasons' average unless the user picked one for it
Params:
  - w: http response writer to respond to the front end
  - r: http request with the series' id and the season number in the URL
*/
func (h *Handler) RateSeason(w http.ResponseWriter, r *http.Request) {
	var req seasonRatingRequest
	user := auth.GetUser(r)
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	season, err := strconv.Atoi(chi.URLParam(r, "seasonNumber"))
	if err != nil || season < 0 {
		utils.Error(w, http.StatusBadRequest, "season must be a number")
		return
	}
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if !validRating(req.Rating) {
		utils.Error(w, http.StatusBadRequest, "rating must be 1 to 10")
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()
	if utils.InternalError(w, tracking.SetSeasonRating(tx, user.UserID, id, season, 0, req.Rating)) {
		return
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
