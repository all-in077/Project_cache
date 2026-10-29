package cache

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func newTestCache(t *testing.T, capacity int, ttl time.Duration) *LRUcache {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return NewLRUcache(capacity, ttl, ctx)
}

func TestSetAndGet(t *testing.T) {
	c := newTestCache(t, 3, time.Minute)

	c.Set("a", []byte("1"))

	got, ok := c.Get("a")
	if !ok || !bytes.Equal(got, []byte("1")) {
		t.Fatalf("expected 1, got %q (ok=%v)", got, ok)
	}
}

func TestGetMiss(t *testing.T) {
	c := newTestCache(t, 3, time.Minute)

	if _, ok := c.Get("nope"); ok {
		t.Fatal("expected miss")
	}
	if s := c.Stats(); s["misses"] != 1 {
		t.Fatalf("expected 1 miss, got %d", s["misses"])
	}
}

func TestOverwriteKey(t *testing.T) {
	c := newTestCache(t, 3, time.Minute)

	c.Set("a", []byte("1"))
	c.Set("a", []byte("2"))

	got, _ := c.Get("a")
	if !bytes.Equal(got, []byte("2")) {
		t.Fatalf("expected 2, got %q", got)
	}
	if s := c.Stats(); s["currentSize"] != 1 {
		t.Fatalf("expected size 1, got %d", s["currentSize"])
	}
}

func TestEvictsLeastRecentlyUsed(t *testing.T) {
	c := newTestCache(t, 2, time.Minute)

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Set("c", []byte("3")) // вытесняет "a"

	if _, ok := c.Get("a"); ok {
		t.Fatal("a should have been evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("b should still be present")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("c should still be present")
	}
	if s := c.Stats(); s["evictions"] != 1 {
		t.Fatalf("expected 1 eviction, got %d", s["evictions"])
	}
}

func TestGetRefreshesOrder(t *testing.T) {
	c := newTestCache(t, 2, time.Minute)

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Get("a")              // теперь "b" самый старый
	c.Set("c", []byte("3")) // вытесняет "b"

	if _, ok := c.Get("b"); ok {
		t.Fatal("b should have been evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("a should still be present")
	}
}

func TestDelete(t *testing.T) {
	c := newTestCache(t, 3, time.Minute)

	c.Set("a", []byte("1"))

	if !c.Delete("a") {
		t.Fatal("expected delete to succeed")
	}
	if c.Delete("a") {
		t.Fatal("second delete should return false")
	}
	if _, ok := c.Get("a"); ok {
		t.Fatal("a should be gone")
	}
}

func TestDeleteMiddleHeadTail(t *testing.T) {
	c := newTestCache(t, 5, time.Minute)

	for _, k := range []string{"a", "b", "c"} { // список: c -> b -> a
		c.Set(k, []byte(k))
	}
	c.Delete("b") // середина
	c.Delete("c") // голова
	c.Delete("a") // последний оставшийся (и голова, и хвост)

	if s := c.Stats(); s["currentSize"] != 0 {
		t.Fatalf("expected empty cache, got %d", s["currentSize"])
	}

	c.Set("x", []byte("x")) // список должен снова работать
	if _, ok := c.Get("x"); !ok {
		t.Fatal("x should be present")
	}
}

func TestTTLExpiryOnGet(t *testing.T) {
	c := newTestCache(t, 3, 50*time.Millisecond)

	c.Set("a", []byte("1"))
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Get("a"); ok {
		t.Fatal("a should have expired")
	}
}

func TestCleanupExpired(t *testing.T) {
	c := newTestCache(t, 3, time.Hour) // воркер не помешает
	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))

	// искусственно «состариваем» записи
	c.mu.Lock()
	for _, n := range c.items {
		n.expireAt = time.Now().Add(-time.Second)
	}
	c.mu.Unlock()

	c.cleanupExpired()

	if s := c.Stats(); s["currentSize"] != 0 {
		t.Fatalf("expected empty cache, got %d", s["currentSize"])
	}
}

func TestBackgroundWorkerRemovesExpired(t *testing.T) {
	c := newTestCache(t, 3, 40*time.Millisecond)
	c.Set("a", []byte("1"))

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if c.Stats()["currentSize"] == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("background worker did not remove expired entry")
}

func TestGetReturnsCopy(t *testing.T) {
	c := newTestCache(t, 3, time.Minute)
	c.Set("a", []byte("hello"))

	got, _ := c.Get("a")
	got[0] = 'X'

	again, _ := c.Get("a")
	if !bytes.Equal(again, []byte("hello")) {
		t.Fatalf("cache data was mutated: %q", again)
	}
}

func TestStats(t *testing.T) {
	c := newTestCache(t, 3, time.Minute)

	c.Set("a", []byte("1"))
	c.Get("a")
	c.Get("a")
	c.Get("missing")

	s := c.Stats()
	if s["hits"] != 2 || s["misses"] != 1 || s["currentSize"] != 1 {
		t.Fatalf("unexpected stats: %v", s)
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := newTestCache(t, 10, time.Minute)

	var wg sync.WaitGroup
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				key := fmt.Sprintf("k%d", (g+i)%20)
				switch i % 3 {
				case 0:
					c.Set(key, []byte("v"))
				case 1:
					c.Get(key)
				default:
					c.Delete(key)
				}
			}
		}(g)
	}
	wg.Wait()

	if s := c.Stats(); s["currentSize"] > 10 {
		t.Fatalf("capacity exceeded: %d", s["currentSize"])
	}
}
