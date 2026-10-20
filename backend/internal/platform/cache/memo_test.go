package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoReturnsFreshResultsWithoutLoading(t *testing.T) {
	m := NewMemo[string](time.Minute, time.Hour, 10)
	var calls int32
	load := func() (string, error) { atomic.AddInt32(&calls, 1); return "v", nil }

	for i := 0; i < 3; i++ {
		if v, err := m.Get("k", load); err != nil || v != "v" {
			t.Fatalf("Get = %q, %v", v, err)
		}
	}
	if calls != 1 {
		t.Errorf("load ran %d times, want 1", calls)
	}
}

func TestMemoServesStaleWhileRefreshing(t *testing.T) {
	now := time.Now()
	m := NewMemo[string](time.Minute, time.Hour, 10)
	var mu sync.Mutex
	m.now = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	m.Get("k", func() (string, error) { return "old", nil })

	mu.Lock()
	now = now.Add(2 * time.Minute)
	mu.Unlock()
	v, _ := m.Get("k", func() (string, error) { return "new", nil })
	if v != "old" {
		t.Errorf("stale Get = %q, want the old result straight away", v)
	}
	waitFor(t, func() bool {
		v, _ := m.Get("k", func() (string, error) { return "new", nil })
		return v == "new"
	})
}

func TestMemoKeepsStaleResultWhenRefreshFails(t *testing.T) {
	now := time.Now()
	m := NewMemo[string](time.Minute, time.Hour, 10)
	m.now = func() time.Time { return now }
	m.Get("k", func() (string, error) { return "old", nil })

	now = now.Add(2 * time.Minute)
	down := func() (string, error) { return "", errors.New("down") }
	m.Get("k", down)
	waitFor(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return len(m.inflight) == 0 })
	if v, err := m.Get("k", down); v != "old" || err != nil {
		t.Errorf("Get after a failed refresh = %q, %v; want the old result", v, err)
	}
}

func TestMemoLoadsAgainOnceTooOld(t *testing.T) {
	now := time.Now()
	m := NewMemo[string](time.Minute, time.Hour, 10)
	m.now = func() time.Time { return now }
	m.Get("k", func() (string, error) { return "old", nil })

	now = now.Add(2 * time.Hour)
	if v, _ := m.Get("k", func() (string, error) { return "new", nil }); v != "new" {
		t.Errorf("Get = %q, want a new lookup past the stale limit", v)
	}
}

func TestMemoDoesNotKeepErrors(t *testing.T) {
	m := NewMemo[string](time.Minute, time.Hour, 10)
	if _, err := m.Get("k", func() (string, error) { return "", errors.New("down") }); err == nil {
		t.Fatal("want the load error")
	}
	if v, err := m.Get("k", func() (string, error) { return "v", nil }); v != "v" || err != nil {
		t.Errorf("Get after an error = %q, %v; want a new lookup", v, err)
	}
}

func TestMemoSharesConcurrentLoads(t *testing.T) {
	m := NewMemo[string](time.Minute, time.Hour, 10)
	var calls int32
	release := make(chan struct{})
	load := func() (string, error) { atomic.AddInt32(&calls, 1); <-release; return "v", nil }

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); m.Get("k", load) }()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if calls != 1 {
		t.Errorf("load ran %d times, want 1", calls)
	}
}

func TestMemoWarmLoadsAhead(t *testing.T) {
	m := NewMemo[string](time.Minute, time.Hour, 10)
	m.Warm("k", func() (string, error) { return "v", nil })
	waitFor(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); _, ok := m.entries["k"]; return ok })
	if v, _ := m.Get("k", func() (string, error) { return "late", nil }); v != "v" {
		t.Errorf("Get = %q, want the warmed result", v)
	}
}

func TestMemoDropsOldestWhenFull(t *testing.T) {
	now := time.Now()
	m := NewMemo[int](time.Hour, time.Hour, 2)
	m.now = func() time.Time { return now }
	for i, k := range []string{"a", "b", "c"} {
		m.Get(k, func() (int, error) { return i, nil })
		now = now.Add(time.Second)
	}
	if _, ok := m.entries["a"]; ok || len(m.entries) != 2 {
		t.Errorf("entries = %v, want the oldest dropped", m.entries)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
