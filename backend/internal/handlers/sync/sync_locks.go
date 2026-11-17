package sync

import (
	gosync "sync"
	"time"
)

/*
keyedLocks: a mutex per key, created when first locked and dropped when nobody holds or waits
for it. Keeps a webhook and the full sync from syncing the same series at once, which could
create its media item twice
*/
type keyedLocks struct {
	mu   gosync.Mutex
	held map[string]*keyedLock
}

type keyedLock struct {
	mu    gosync.Mutex
	users int
}

/*
Function:	lock
Purpose:	lock key, returning the function that unlocks it
Params:
  - key: e.g. "sonarr:12"
*/
func (k *keyedLocks) lock(key string) func() {
	k.mu.Lock()
	if k.held == nil {
		k.held = map[string]*keyedLock{}
	}
	l := k.held[key]
	if l == nil {
		l = &keyedLock{}
		k.held[key] = l
	}
	l.users++
	k.mu.Unlock()

	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		k.mu.Lock()
		l.users--
		if l.users == 0 {
			delete(k.held, key)
		}
		k.mu.Unlock()
	}
}

// how long AniList data of a title is used before a sync asks for it again
const enrichAfter = 7 * 24 * time.Hour

/*
Function:	needsEnrichment
Purpose:	whether a title's AniList data is missing or old enough to fetch again
Params:
  - syncedAt: when it was last fetched, nil for never
  - now: the current time
*/
func needsEnrichment(syncedAt *time.Time, now time.Time) bool {
	return syncedAt == nil || now.Sub(*syncedAt) >= enrichAfter
}
