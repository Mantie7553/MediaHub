package sync

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	gosync "sync"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

/*
SourceResult: outcome of one library source in a sync run
*/
type SourceResult struct {
	Source   string `json:"source"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Duration string `json:"duration"`
}

/*
Status: the current or last full library sync
*/
type Status struct {
	Running    bool           `json:"running"`
	Trigger    string         `json:"trigger,omitempty"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	NextRun    *time.Time     `json:"next_run,omitempty"`
	Results    []SourceResult `json:"results"`
}

// runner state shared by the admin button and the nightly schedule, so only one sync runs at a time
type runner struct {
	mu     gosync.Mutex
	status Status
}

/*
Function:	sources
Purpose:	every library source in the order a full sync runs them. Sonarr/Radarr first so their
items exist before anything else, the folder scans after
*/
func (h *Handler) sources() []struct {
	name string
	run  func() (string, error)
} {
	plain := func(f func() error) func() (string, error) {
		return func() (string, error) { return "", f() }
	}
	return []struct {
		name string
		run  func() (string, error)
	}{
		{"sonarr", plain(h.syncSonarr)},
		{"radarr", plain(h.syncRadarr)},
		{"manga", plain(h.syncManga)},
		{"light-novels", plain(h.syncLightNovels)},
		{"music", func() (string, error) {
			n, err := h.syncMusic()
			return fmt.Sprintf("%d tracks", n), err
		}},
	}
}

/*
Function:	StartFullSync
Purpose:	Run every library sync in the background. Returns false if one is already running
Params:
  - trigger: what started it ("manual" or "schedule"), shown in the status
*/
func (h *Handler) StartFullSync(trigger string) bool {
	h.runner.mu.Lock()
	if h.runner.status.Running {
		h.runner.mu.Unlock()
		return false
	}
	now := time.Now()
	h.runner.status.Running = true
	h.runner.status.Trigger = trigger
	h.runner.status.StartedAt = &now
	h.runner.status.FinishedAt = nil
	h.runner.status.Results = []SourceResult{}
	h.runner.mu.Unlock()

	go h.runFullSync(trigger)
	return true
}

func (h *Handler) runFullSync(trigger string) {
	logger.Info("Library sync started (%s)", trigger)
	defer func() {
		// a panicking source must not leave the sync marked as running forever
		if r := recover(); r != nil {
			logger.Error("library sync panicked: %v", r)
		}
		now := time.Now()
		h.runner.mu.Lock()
		h.runner.status.Running = false
		h.runner.status.FinishedAt = &now
		h.runner.mu.Unlock()
		logger.Info("Library sync finished")
	}()

	for _, src := range h.sources() {
		start := time.Now()
		detail, err := src.run()
		result := SourceResult{
			Source:   src.name,
			OK:       err == nil,
			Detail:   detail,
			Duration: time.Since(start).Round(time.Second).String(),
		}
		if err != nil {
			result.Error = err.Error()
			logger.Error("library sync of %s failed: %s", src.name, err.Error())
		}
		h.runner.mu.Lock()
		h.runner.status.Results = append(h.runner.status.Results, result)
		h.runner.mu.Unlock()
	}
}

/*
Function:	SyncStatus
Purpose:	current or last full sync, and when the next scheduled one is
*/
func (h *Handler) SyncStatus() Status {
	h.runner.mu.Lock()
	defer h.runner.mu.Unlock()
	st := h.runner.status
	st.Results = append([]SourceResult{}, st.Results...)
	if next, ok := nextScheduledRun(time.Now()); ok {
		st.NextRun = &next
	}
	return st
}

/*
Function:	scheduleHour
Purpose:	hour of day (container local time, TZ) for the nightly sync from LIBRARY_SYNC_HOUR.
Defaults to 4, "off" disables it
*/
func scheduleHour() (int, bool) {
	v := os.Getenv("LIBRARY_SYNC_HOUR")
	if v == "off" || v == "none" {
		return 0, false
	}
	if v == "" {
		return 4, true
	}
	hour, err := strconv.Atoi(v)
	if err != nil || hour < 0 || hour > 23 {
		logger.Warn("LIBRARY_SYNC_HOUR %q is not an hour 0-23, nightly sync disabled", v)
		return 0, false
	}
	return hour, true
}

/*
Function:	nextScheduledRun
Purpose:	the next time the nightly sync is due after now
*/
func nextScheduledRun(now time.Time) (time.Time, bool) {
	hour, ok := scheduleHour()
	if !ok {
		return time.Time{}, false
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next, true
}

/*
Function:	StartSchedule
Purpose:	run a full library sync every night at LIBRARY_SYNC_HOUR
*/
func (h *Handler) StartSchedule() {
	if _, ok := nextScheduledRun(time.Now()); !ok {
		logger.Info("Nightly library sync is off")
		return
	}
	go func() {
		for {
			next, ok := nextScheduledRun(time.Now())
			if !ok {
				return
			}
			time.Sleep(time.Until(next))
			if !h.StartFullSync("schedule") {
				logger.Info("Nightly library sync skipped, one is already running")
			}
		}
	}()
}

// ---- HTTP handlers ----

/*
Function:	StartSync
Purpose:	POST /admin/sync: start a full library sync in the background
*/
func (h *Handler) StartSync(w http.ResponseWriter, r *http.Request) {
	if !h.StartFullSync("manual") {
		utils.Error(w, http.StatusConflict, "a library sync is already running")
		return
	}
	utils.JSON(w, h.SyncStatus(), http.StatusAccepted)
}

/*
Function:	GetSyncStatus
Purpose:	GET /admin/sync: progress and results of the current or last full sync
*/
func (h *Handler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	utils.JSON(w, h.SyncStatus())
}

// single-source endpoints, kept for scripts. They run in the request, not the background runner

func (h *Handler) SyncSonar(w http.ResponseWriter, r *http.Request) {
	if utils.InternalError(w, h.syncSonarr()) {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SyncRadarr(w http.ResponseWriter, r *http.Request) {
	if utils.InternalError(w, h.syncRadarr()) {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SyncManga(w http.ResponseWriter, r *http.Request) {
	if utils.InternalError(w, h.syncManga()) {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SyncLightNovel(w http.ResponseWriter, r *http.Request) {
	if utils.InternalError(w, h.syncLightNovels()) {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SyncMusic(w http.ResponseWriter, r *http.Request) {
	synced, err := h.syncMusic()
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, map[string]int{"synced": synced})
}
