/*
Package insights answers the admin Insights page: how much the library takes up on disk, what
people watch and read, who has been around, and how downloads are going
*/
package insights

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	Get
Purpose:	everything the Insights page shows. The folder sizes come from a scan kept for an
hour; ?refresh=1 scans again
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	resp := insightsResponse{GeneratedAt: now}

	var err error
	if resp.TopWatched, err = topWatched(h.db); utils.InternalError(w, err) {
		return
	}
	if resp.TopRead, err = topRead(h.db); utils.InternalError(w, err) {
		return
	}
	if resp.Users, err = users(h.db, now); utils.InternalError(w, err) {
		return
	}
	if resp.Jobs, err = jobs(h.db); utils.InternalError(w, err) {
		return
	}

	used, max, _ := cache.Disk().Stats()
	resp.Cache = cacheUsage{Enabled: max > 0, Used: used, Max: max}
	resp.Storage = mediaScanner().Get(r.URL.Query().Get("refresh") == "1")

	utils.JSON(w, resp)
}
