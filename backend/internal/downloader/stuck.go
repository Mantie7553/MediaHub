package downloader

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
	"github.com/Mantie7553/MediaHub/backend/internal/notify"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

/*
Sonarr and Radarr have no "download failed" webhook: a torrent with no seeders, an import that
needs a hand or a title no indexer has would never reach anyone. Every so often their queues
are read and the admins are told about downloads that are stuck, and about requests a day old
that still have nothing grabbed. Once a problem is gone its notification is marked read
*/

const (
	// a download has to be in trouble this long before admins hear, a torrent finding its
	// first peers is briefly "stalled"
	stuckGrace = time.Hour
	// how long a request waits for Sonarr / Radarr to grab something before admins hear
	noReleaseAfter = 24 * time.Hour
	// how often the queues are read, unless DOWNLOAD_CHECK_MINUTES says otherwise
	defaultCheckEvery = 30 * time.Minute
)

// arrSource is Sonarr or Radarr, and where MediaHub keeps which title its items are
type arrSource struct {
	name   string // "sonarr" or "radarr"
	label  string // "Sonarr" or "Radarr"
	table  string
	column string
	client *arr.ArrClient
}

// stuckGroup is the stuck downloads of one Sonarr series / Radarr movie
type stuckGroup struct {
	count  int
	reason string
	// a release's name, the title when MediaHub doesn't know the series / movie
	release string
}

// noReleaseCandidate is a request old enough that Sonarr / Radarr should have grabbed something
type noReleaseCandidate struct {
	requestID string
	title     string
	handler   string // "sonarr" or "radarr"
	arrID     int    // 0 when the title was never linked to a series / movie
}

// stuckChecker remembers between passes what it told the admins about
type stuckChecker struct {
	db      *sql.DB
	sources []arrSource
	// when a problem queue item ("sonarr:12") was first seen, for items with no grab time
	firstSeen map[string]time.Time
	// notification keys sent and not yet resolved, to the arr they're about
	flagged map[string]string
}

/*
Function:	checkInterval
Purpose:	how often to read the queues, from DOWNLOAD_CHECK_MINUTES ("off" turns it off)
*/
func checkInterval() (time.Duration, bool) {
	v := os.Getenv("DOWNLOAD_CHECK_MINUTES")
	if v == "off" || v == "none" {
		return 0, false
	}
	if v == "" {
		return defaultCheckEvery, true
	}
	minutes, err := strconv.Atoi(v)
	if err != nil || minutes < 1 {
		logger.Warn("DOWNLOAD_CHECK_MINUTES %q is not a number of minutes, stuck download checks disabled", v)
		return 0, false
	}
	return time.Duration(minutes) * time.Minute, true
}

/*
Function:	StartStuckChecks
Purpose:	read Sonarr's and Radarr's queues every DOWNLOAD_CHECK_MINUTES (30 by default) and
tell the admins about stuck downloads, until ctx ends. Does nothing when neither is set up
*/
func StartStuckChecks(ctx context.Context, db *sql.DB) {
	every, ok := checkInterval()
	if !ok {
		return
	}
	c := &stuckChecker{
		db:        db,
		firstSeen: map[string]time.Time{},
		flagged:   map[string]string{},
	}
	for _, s := range []arrSource{
		{"sonarr", "Sonarr", "sonarr_items", "sonarr_series_id", arr.NewArrClient("SONARR_URL", "SONARR_API_KEY")},
		{"radarr", "Radarr", "radarr_items", "radarr_movie_id", arr.NewArrClient("RADARR_URL", "RADARR_API_KEY")},
	} {
		if s.client.Configured() {
			c.sources = append(c.sources, s)
		}
	}
	if len(c.sources) == 0 {
		return
	}
	c.loadFlagged()
	for {
		// waits first: at startup Sonarr / Radarr may still be starting too
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
		c.check(time.Now())
	}
}

/*
Function:	check
Purpose:	one pass: read the queues, tell the admins what's newly stuck, resolve what's fine now
*/
func (c *stuckChecker) check(now time.Time) {
	current := map[string]string{}
	fetched := map[string]bool{}
	queued := map[string]map[int]bool{}

	for _, s := range c.sources {
		items, err := s.client.GetQueue()
		if err != nil {
			logger.Warn("reading %s's queue: %v", s.label, err)
			continue
		}
		fetched[s.name] = true
		queued[s.name] = queuedIDs(items, s.name)
		for arrID, g := range stuckGroups(items, s.name, c.firstSeen, now) {
			key := notify.StuckKey(s.name, arrID)
			current[key] = s.name
			c.notifyStuck(s, arrID, g, key)
		}
	}

	candidates, err := c.noReleaseCandidates(now)
	if err != nil {
		logger.Warn("finding requests with nothing grabbed: %v", err)
		// what was flagged for this stays as it was, rather than resolved on a failed query
		for key, src := range c.flagged {
			if isNoReleaseKey(key) {
				current[key] = src
			}
		}
	}
	for _, cand := range withoutRelease(candidates, fetched, queued) {
		if cand.handler == "radarr" && !c.movieAvailable(cand.arrID) {
			continue
		}
		key := notify.NoReleaseKey(cand.requestID)
		current[key] = cand.handler
		label := "Sonarr"
		if cand.handler == "radarr" {
			label = "Radarr"
		}
		notify.Admins(c.db, notify.Notification{
			Kind:  notify.KindDownloadStuck,
			Title: "No release found for " + cand.title,
			Body:  label + " hasn't grabbed anything for it in over a day. Search for it there by hand, or try another release",
			Link:  notify.RequestLink(cand.requestID),
			Key:   key,
		})
	}

	for _, key := range settle(c.flagged, current, fetched) {
		notify.Resolve(c.db, key)
	}
	c.flagged = current
}

/*
Function:	loadFlagged
Purpose:	the stuck notifications still unread from before a restart, so they are resolved
once fixed too. A no-release one is about the arr its request's job went to
*/
func (c *stuckChecker) loadFlagged() {
	rows, err := c.db.Query(
		`SELECT DISTINCT n.dedupe_key, COALESCE(CASE
			WHEN n.dedupe_key LIKE 'stuck:%' THEN split_part(n.dedupe_key, ':', 2)
			ELSE (SELECT j.handler::text FROM download_jobs j
				WHERE 'norelease:' || j.request_id::text = n.dedupe_key LIMIT 1)
		END, '')
		FROM notifications n WHERE n.kind = $1 AND n.read_at IS NULL`,
		notify.KindDownloadStuck,
	)
	if err != nil {
		logger.Warn("reading unread stuck download notifications: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var key, src string
		if err := rows.Scan(&key, &src); err != nil {
			logger.Warn("reading unread stuck download notifications: %v", err)
			return
		}
		c.flagged[key] = src
	}
}

/*
Function:	notifyStuck
Purpose:	tell the admins a series' / movie's downloads are stuck, by its MediaHub title when
it has one
*/
func (c *stuckChecker) notifyStuck(s arrSource, arrID int, g stuckGroup, key string) {
	title, link := g.release, "/downloads"
	var mediaItemID, mediaType, name string
	err := c.db.QueryRow(
		`SELECT mi.id, mi.media_type, mi.title FROM `+s.table+` x
		JOIN media_items mi ON mi.id = x.media_item_id
		WHERE x.`+s.column+` = $1 LIMIT 1`, arrID,
	).Scan(&mediaItemID, &mediaType, &name)
	if err == nil {
		title, link = name, notify.DetailsLink(mediaType, mediaItemID, "")
	} else if err != sql.ErrNoRows {
		logger.Warn("finding the title of %s item %d: %v", s.label, arrID, err)
	}
	body := s.label + ": " + g.reason
	if g.count > 1 {
		body = fmt.Sprintf("%s, %d downloads: %s", s.label, g.count, g.reason)
	}
	notify.Admins(c.db, notify.Notification{
		Kind:  notify.KindDownloadStuck,
		Title: title + " is stuck downloading",
		Body:  body,
		Link:  link,
		Key:   key,
	})
}

/*
Function:	noReleaseCandidates
Purpose:	open Sonarr / Radarr requests whose job is over a day old and whose title has no file
yet. A series also needs an aired episode, one that hasn't aired can't have a release
*/
func (c *stuckChecker) noReleaseCandidates(now time.Time) ([]noReleaseCandidate, error) {
	rows, err := c.db.Query(
		`SELECT DISTINCT ON (r.id) r.id, mi.title, j.handler::text,
			COALESCE(CASE WHEN j.handler = 'sonarr' THEN si.sonarr_series_id ELSE ri.radarr_movie_id END, 0)
		FROM download_requests r
		JOIN download_jobs j ON j.request_id = r.id
		JOIN media_items mi ON mi.id = r.media_item_id
		LEFT JOIN sonarr_items si ON si.media_item_id = mi.id
		LEFT JOIN radarr_items ri ON ri.media_item_id = mi.id
		WHERE r.status IN ('approved', 'queued', 'downloading')
			AND j.handler IN ('sonarr', 'radarr') AND j.status <> 'failed'
			AND j.created_at < $1
			AND NOT EXISTS (SELECT 1 FROM episodes e WHERE e.media_item_id = mi.id AND e.file_path IS NOT NULL)
			AND NOT EXISTS (SELECT 1 FROM movie_metadata m WHERE m.media_item_id = mi.id AND m.file_path IS NOT NULL)
			AND (j.handler <> 'sonarr' OR EXISTS (SELECT 1 FROM episodes e WHERE e.media_item_id = mi.id))
		ORDER BY r.id, j.created_at DESC`,
		now.Add(-noReleaseAfter),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []noReleaseCandidate
	for rows.Next() {
		var cand noReleaseCandidate
		if err := rows.Scan(&cand.requestID, &cand.title, &cand.handler, &cand.arrID); err != nil {
			return nil, err
		}
		out = append(out, cand)
	}
	return out, rows.Err()
}

/*
Function:	movieAvailable
Purpose:	whether Radarr thinks a movie is out, one still in cinemas has no release to find.
Unknown counts as not, so a Radarr hiccup never tells anyone
*/
func (c *stuckChecker) movieAvailable(movieID int) bool {
	for _, s := range c.sources {
		if s.name == "radarr" {
			m, err := s.client.GetMovie(movieID)
			return err == nil && m.IsAvailable
		}
	}
	return false
}

/*
Function:	problemReason
Purpose:	why a queue item is in trouble, and whether it is: Sonarr / Radarr flagged it a
warning or error, or it can't be imported, or its download failed
*/
func problemReason(item arr.QueueItem) (string, bool) {
	trouble := item.TrackedDownloadStatus == "warning" || item.TrackedDownloadStatus == "error" ||
		item.TrackedDownloadState == "importBlocked" || item.TrackedDownloadState == "failedPending"
	if !trouble {
		return "", false
	}
	for _, m := range item.StatusMessages {
		for _, msg := range m.Messages {
			if msg != "" {
				return msg, true
			}
		}
	}
	if item.ErrorMessage != "" {
		return item.ErrorMessage, true
	}
	switch item.TrackedDownloadState {
	case "importBlocked":
		return "it downloaded but can't be imported", true
	case "failedPending":
		return "the download failed", true
	}
	return "the download has a " + item.TrackedDownloadStatus, true
}

/*
Function:	stuckGroups
Purpose:	the queue's items that have been in trouble for over stuckGrace, by Sonarr series /
Radarr movie. Items with no grab time count from when a pass first saw them in trouble; that
record is kept in firstSeen and dropped once an item is fine or gone
Params:
  - items: the queue
  - arrName: "sonarr" or "radarr"
  - firstSeen: when each problem item was first seen, updated in place
  - now: the time now
*/
func stuckGroups(items []arr.QueueItem, arrName string, firstSeen map[string]time.Time, now time.Time) map[int]stuckGroup {
	prefix := arrName + ":"
	inTrouble := map[string]bool{}
	groups := map[int]stuckGroup{}
	for _, item := range items {
		reason, ok := problemReason(item)
		if !ok {
			continue
		}
		seenKey := prefix + strconv.Itoa(item.ID)
		inTrouble[seenKey] = true
		if _, ok := firstSeen[seenKey]; !ok {
			firstSeen[seenKey] = now
		}
		since := item.Added
		if since.IsZero() {
			since = firstSeen[seenKey]
		}
		if now.Sub(since) < stuckGrace {
			continue
		}
		arrID := arrItemID(item, arrName)
		if arrID == 0 {
			continue
		}
		g := groups[arrID]
		if g.count == 0 {
			g.reason, g.release = reason, item.Title
		}
		g.count++
		groups[arrID] = g
	}
	for k := range firstSeen {
		if strings.HasPrefix(k, prefix) && !inTrouble[k] {
			delete(firstSeen, k)
		}
	}
	return groups
}

// the Sonarr series / Radarr movie a queue item is for, 0 when it has none
func arrItemID(item arr.QueueItem, arrName string) int {
	if arrName == "radarr" {
		return item.MovieID
	}
	return item.SeriesID
}

// the series / movies with anything in the queue, stuck or not
func queuedIDs(items []arr.QueueItem, arrName string) map[int]bool {
	ids := map[int]bool{}
	for _, item := range items {
		if id := arrItemID(item, arrName); id != 0 {
			ids[id] = true
		}
	}
	return ids
}

/*
Function:	withoutRelease
Purpose:	the candidates Sonarr / Radarr really has nothing for: linked to a series / movie
with nothing in its queue. One whose arr couldn't be read this pass is left out
*/
func withoutRelease(cands []noReleaseCandidate, fetched map[string]bool, queued map[string]map[int]bool) []noReleaseCandidate {
	var out []noReleaseCandidate
	for _, cand := range cands {
		if cand.arrID == 0 || !fetched[cand.handler] || queued[cand.handler][cand.arrID] {
			continue
		}
		out = append(out, cand)
	}
	return out
}

func isNoReleaseKey(key string) bool { return strings.HasPrefix(key, notify.NoReleaseKey("")) }

/*
Function:	settle
Purpose:	the keys flagged last pass that aren't any more, to resolve. Those of an arr that
couldn't be read this pass are carried into current instead: it being down isn't the problem
being fixed
Params:
  - flagged: the keys flagged last pass, to their arr
  - current: the keys flagged this pass, added to in place
  - fetched: the arrs read this pass
*/
func settle(flagged, current map[string]string, fetched map[string]bool) []string {
	var out []string
	for key, src := range flagged {
		if _, still := current[key]; still {
			continue
		}
		if !fetched[src] {
			current[key] = src
			continue
		}
		out = append(out, key)
	}
	return out
}
