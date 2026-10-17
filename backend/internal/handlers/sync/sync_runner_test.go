package sync

import (
	"testing"
	"time"
)

func TestNextScheduledRun(t *testing.T) {
	loc := time.FixedZone("test", -4*3600)

	t.Setenv("LIBRARY_SYNC_HOUR", "")
	before := time.Date(2026, 10, 17, 2, 30, 0, 0, loc)
	if next, ok := nextScheduledRun(before); !ok || !next.Equal(time.Date(2026, 10, 17, 4, 0, 0, 0, loc)) {
		t.Errorf("default before 4am = %v, %v; want today 04:00", next, ok)
	}
	after := time.Date(2026, 10, 17, 4, 0, 0, 0, loc)
	if next, _ := nextScheduledRun(after); !next.Equal(time.Date(2026, 10, 18, 4, 0, 0, 0, loc)) {
		t.Errorf("at 4am = %v; want tomorrow 04:00", next)
	}

	t.Setenv("LIBRARY_SYNC_HOUR", "23")
	if next, _ := nextScheduledRun(before); next.Hour() != 23 || next.Day() != 17 {
		t.Errorf("hour 23 = %v", next)
	}

	for _, v := range []string{"off", "none", "24", "-1", "4am"} {
		t.Setenv("LIBRARY_SYNC_HOUR", v)
		if _, ok := nextScheduledRun(before); ok {
			t.Errorf("LIBRARY_SYNC_HOUR=%q should disable the schedule", v)
		}
	}
}

func TestStartFullSyncWhileRunning(t *testing.T) {
	h := &Handler{}
	h.runner.status.Running = true
	if h.StartFullSync("manual") {
		t.Fatal("a second sync started while one was running")
	}
	if st := h.SyncStatus(); !st.Running {
		t.Fatal("status lost the running flag")
	}
}
