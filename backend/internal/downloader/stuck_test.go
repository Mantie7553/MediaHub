package downloader

import (
	"slices"
	"testing"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
)

func queueItem(id, seriesID int, status, state string, added time.Time, msgs ...string) arr.QueueItem {
	item := arr.QueueItem{ID: id, SeriesID: seriesID, Title: "release", TrackedDownloadStatus: status,
		TrackedDownloadState: state, Added: added}
	if len(msgs) > 0 {
		item.StatusMessages = append(item.StatusMessages, struct {
			Title    string   `json:"title"`
			Messages []string `json:"messages"`
		}{Title: "release", Messages: msgs})
	}
	return item
}

func TestProblemReason(t *testing.T) {
	cases := []struct {
		name   string
		item   arr.QueueItem
		reason string
		ok     bool
	}{
		{"clean download", queueItem(1, 5, "ok", "downloading", time.Time{}), "", false},
		{"import pending is fine", queueItem(1, 5, "ok", "importPending", time.Time{}), "", false},
		{"stalled warning", queueItem(1, 5, "warning", "downloading", time.Time{}, "", "The download is stalled with no connections"),
			"The download is stalled with no connections", true},
		{"error message", arr.QueueItem{TrackedDownloadStatus: "error", ErrorMessage: "qBittorrent is unavailable"},
			"qBittorrent is unavailable", true},
		{"import blocked", queueItem(1, 5, "ok", "importBlocked", time.Time{}), "it downloaded but can't be imported", true},
		{"failed pending", queueItem(1, 5, "ok", "failedPending", time.Time{}), "the download failed", true},
		{"bare warning", queueItem(1, 5, "warning", "downloading", time.Time{}), "the download has a warning", true},
	}
	for _, c := range cases {
		reason, ok := problemReason(c.item)
		if reason != c.reason || ok != c.ok {
			t.Errorf("%s: got %q, %v, want %q, %v", c.name, reason, ok, c.reason, c.ok)
		}
	}
}

func TestStuckGroupsWaitsOutTheGrace(t *testing.T) {
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	items := []arr.QueueItem{
		// three episodes of one series, two in trouble for long enough
		queueItem(1, 5, "warning", "downloading", now.Add(-3*time.Hour), "stalled"),
		queueItem(2, 5, "warning", "downloading", now.Add(-2*time.Hour), "stalled"),
		queueItem(3, 5, "ok", "downloading", now.Add(-3*time.Hour)),
		// in trouble, but only just grabbed
		queueItem(4, 6, "warning", "downloading", now.Add(-10*time.Minute), "stalled"),
		// no grab time: counts from the first pass that saw it
		queueItem(5, 7, "error", "downloading", time.Time{}, "no files"),
	}
	firstSeen := map[string]time.Time{"radarr:5": now.Add(-5 * time.Hour)}

	groups := stuckGroups(items, "sonarr", firstSeen, now)
	if len(groups) != 1 || groups[5].count != 2 || groups[5].reason != "stalled" || groups[5].release != "release" {
		t.Fatalf("first pass groups = %+v", groups)
	}
	if !firstSeen["sonarr:5"].Equal(now) {
		t.Errorf("item 5 first seen at %v, want now", firstSeen["sonarr:5"])
	}
	if _, ok := firstSeen["radarr:5"]; !ok {
		t.Error("a Sonarr pass dropped a Radarr item's first sighting")
	}

	// an hour on, item 5 has been in trouble long enough; item 1 is fine now
	later := now.Add(stuckGrace)
	items[0].TrackedDownloadStatus = "ok"
	groups = stuckGroups(items, "sonarr", firstSeen, later)
	if groups[5].count != 1 || groups[7].count != 1 || groups[6].count != 1 {
		t.Errorf("second pass groups = %+v", groups)
	}
	if _, ok := firstSeen["sonarr:1"]; ok {
		t.Error("a queue item that's fine kept its first sighting")
	}
}

func TestWithoutRelease(t *testing.T) {
	cands := []noReleaseCandidate{
		{requestID: "a", handler: "sonarr", arrID: 5}, // nothing queued: told
		{requestID: "b", handler: "sonarr", arrID: 6}, // something queued
		{requestID: "c", handler: "sonarr", arrID: 0}, // never linked
		{requestID: "d", handler: "radarr", arrID: 9}, // Radarr couldn't be read
	}
	fetched := map[string]bool{"sonarr": true}
	queued := map[string]map[int]bool{"sonarr": {6: true}}
	got := withoutRelease(cands, fetched, queued)
	if len(got) != 1 || got[0].requestID != "a" {
		t.Errorf("withoutRelease = %+v", got)
	}
}

func TestSettle(t *testing.T) {
	flagged := map[string]string{
		"stuck:sonarr:5":  "sonarr", // still stuck
		"stuck:sonarr:6":  "sonarr", // fixed
		"stuck:radarr:9":  "radarr", // Radarr couldn't be read
		"norelease:req-1": "sonarr", // grabbed now
	}
	current := map[string]string{"stuck:sonarr:5": "sonarr"}
	resolved := settle(flagged, current, map[string]bool{"sonarr": true})
	slices.Sort(resolved)
	if !slices.Equal(resolved, []string{"norelease:req-1", "stuck:sonarr:6"}) {
		t.Errorf("resolved = %v", resolved)
	}
	if current["stuck:radarr:9"] != "radarr" || len(current) != 2 {
		t.Errorf("current = %v, want the Radarr key carried over", current)
	}
}

func TestCheckInterval(t *testing.T) {
	cases := []struct {
		env   string
		every time.Duration
		ok    bool
	}{
		{"", defaultCheckEvery, true},
		{"5", 5 * time.Minute, true},
		{"off", 0, false},
		{"0", 0, false},
		{"soon", 0, false},
	}
	for _, c := range cases {
		t.Setenv("DOWNLOAD_CHECK_MINUTES", c.env)
		every, ok := checkInterval()
		if every != c.every || ok != c.ok {
			t.Errorf("%q: got %v, %v, want %v, %v", c.env, every, ok, c.every, c.ok)
		}
	}
}

func TestIsNoReleaseKey(t *testing.T) {
	if !isNoReleaseKey("norelease:abc") || isNoReleaseKey("stuck:sonarr:5") {
		t.Error("isNoReleaseKey told the keys apart wrong")
	}
}
