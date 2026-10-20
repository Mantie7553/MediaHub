/*
Package cache keeps slow answers close: Memo holds results from external APIs in memory, and
Files keeps local copies of media files read from the NAS
*/
package cache

import (
	"sync"
	"time"
)

/*
Memo remembers the results of a slow lookup by key. A result younger than fresh is returned
as is. One younger than stale is returned straight away too, while it is looked up again in
the background, so nobody waits on a refresh; if that refresh fails the old result keeps being
used until it is older than stale. Concurrent lookups of the same key share one call
*/
type Memo[V any] struct {
	fresh, stale time.Duration
	max          int
	now          func() time.Time

	mu       sync.Mutex
	entries  map[string]memoEntry[V]
	inflight map[string]*memoCall[V]
}

type memoEntry[V any] struct {
	val V
	at  time.Time
}

type memoCall[V any] struct {
	done chan struct{}
	val  V
	err  error
}

/*
Function:	NewMemo
Purpose:	a Memo whose results are fresh for fresh and usable for stale (at least fresh),
holding at most max keys, the oldest dropped first
*/
func NewMemo[V any](fresh, stale time.Duration, max int) *Memo[V] {
	if stale < fresh {
		stale = fresh
	}
	return &Memo[V]{
		fresh:    fresh,
		stale:    stale,
		max:      max,
		entries:  map[string]memoEntry[V]{},
		inflight: map[string]*memoCall[V]{},
	}
}

/*
Function:	Get
Purpose:	the result for key, from memory when it is recent enough, otherwise from load
Params:
  - key: what is being looked up
  - load: does the slow lookup; an error is returned to the caller and nothing is kept
*/
func (m *Memo[V]) Get(key string, load func() (V, error)) (V, error) {
	m.mu.Lock()
	if e, ok := m.entries[key]; ok {
		age := m.clock().Sub(e.at)
		if age <= m.fresh {
			m.mu.Unlock()
			return e.val, nil
		}
		if age <= m.stale {
			m.startLocked(key, load)
			m.mu.Unlock()
			return e.val, nil
		}
	}
	c := m.startLocked(key, load)
	m.mu.Unlock()

	<-c.done
	return c.val, c.err
}

/*
Function:	Warm
Purpose:	look key up in the background so the first request finds it ready
*/
func (m *Memo[V]) Warm(key string, load func() (V, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.entries[key]; !ok {
		m.startLocked(key, load)
	}
}

// starts load for key unless it is already running; m.mu must be held
func (m *Memo[V]) startLocked(key string, load func() (V, error)) *memoCall[V] {
	if c, ok := m.inflight[key]; ok {
		return c
	}
	c := &memoCall[V]{done: make(chan struct{})}
	m.inflight[key] = c
	go func() {
		val, err := load()
		m.mu.Lock()
		delete(m.inflight, key)
		if err == nil {
			m.putLocked(key, val)
		}
		m.mu.Unlock()
		c.val, c.err = val, err
		close(c.done)
	}()
	return c
}

// keeps val for key, dropping the oldest result when full; m.mu must be held
func (m *Memo[V]) putLocked(key string, val V) {
	if _, ok := m.entries[key]; !ok && m.max > 0 && len(m.entries) >= m.max {
		var oldest string
		var oldestAt time.Time
		for k, e := range m.entries {
			if oldest == "" || e.at.Before(oldestAt) {
				oldest, oldestAt = k, e.at
			}
		}
		delete(m.entries, oldest)
	}
	m.entries[key] = memoEntry[V]{val: val, at: m.clock()}
}

func (m *Memo[V]) clock() time.Time {
	if m.now != nil {
		return m.now()
	}
	return time.Now()
}
