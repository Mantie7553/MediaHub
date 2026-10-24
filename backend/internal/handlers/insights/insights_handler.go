/*
Package insights answers the admin Insights page: how much the library takes up on disk, what
people watch, read and listen to, who has been around, and how downloads are going
*/
package insights

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

const (
	// how long after startup the first daily snapshot is taken, so a restart doesn't walk the
	// NAS while everything else is starting
	firstSnapshotDelay = 10 * time.Minute
	snapshotEvery      = 24 * time.Hour
)

type Handler struct {
	db      *sql.DB
	scanner *storageScanner
}

func NewHandler(db *sql.DB) *Handler {
	h := &Handler{db: db}
	h.scanner = newStorageScanner(os.Getenv("MEDIA_ROOT"), func(u storageUsage) {
		if err := recordSnapshot(db, u); err != nil {
			logger.Warn("could not save today's storage snapshot: %s", err.Error())
		}
	})
	return h
}

/*
Function:	StartDailySnapshots
Purpose:	scan the library once a day, so the storage chart has a point for every day even when
nobody opens the Insights page
*/
func (h *Handler) StartDailySnapshots() {
	if h.scanner.root == "" {
		return
	}
	go func() {
		time.Sleep(firstSnapshotDelay)
		for {
			h.scanner.Get(false)
			time.Sleep(snapshotEvery)
		}
	}()
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
	if resp.TopPlayed, err = topPlayed(h.db); utils.InternalError(w, err) {
		return
	}
	if resp.Users, err = users(h.db, now); utils.InternalError(w, err) {
		return
	}
	if resp.Jobs, err = jobs(h.db); utils.InternalError(w, err) {
		return
	}
	if resp.Activity, err = activity(h.db); utils.InternalError(w, err) {
		return
	}

	used, max, _ := cache.Disk().Stats()
	resp.Cache = cacheUsage{Enabled: max > 0, Used: used, Max: max}
	// scanned before reading the history, so a new scan's snapshot is in it
	resp.Storage = h.scanner.Get(r.URL.Query().Get("refresh") == "1")
	if resp.StorageHistory, err = storageHistory(h.db); utils.InternalError(w, err) {
		return
	}

	utils.JSON(w, resp)
}
