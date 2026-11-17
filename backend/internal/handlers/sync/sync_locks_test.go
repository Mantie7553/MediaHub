package sync

import (
	gosync "sync"
	"testing"
	"time"
)

func TestKeyedLocksSerializeOneKey(t *testing.T) {
	var k keyedLocks
	var wg gosync.WaitGroup
	inside, most := 0, 0
	var mu gosync.Mutex
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := k.lock("sonarr:1")
			mu.Lock()
			inside++
			if inside > most {
				most = inside
			}
			mu.Unlock()
			time.Sleep(time.Millisecond)
			mu.Lock()
			inside--
			mu.Unlock()
			unlock()
		}()
	}
	wg.Wait()
	if most != 1 {
		t.Errorf("%d held the same key at once, want 1", most)
	}
	if len(k.held) != 0 {
		t.Errorf("%d locks left after all were released", len(k.held))
	}
}

func TestKeyedLocksOtherKeysDontWait(t *testing.T) {
	var k keyedLocks
	unlock := k.lock("sonarr:1")
	defer unlock()
	done := make(chan struct{})
	go func() {
		k.lock("radarr:1")()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("a different key waited on a held one")
	}
}

func TestNeedsEnrichment(t *testing.T) {
	now := time.Date(2026, 11, 16, 4, 0, 0, 0, time.UTC)
	recent := now.Add(-6 * 24 * time.Hour)
	old := now.Add(-7 * 24 * time.Hour)
	if !needsEnrichment(nil, now) {
		t.Error("never enriched should need it")
	}
	if needsEnrichment(&recent, now) {
		t.Error("enriched 6 days ago shouldn't need it")
	}
	if !needsEnrichment(&old, now) {
		t.Error("enriched a week ago should need it")
	}
}
