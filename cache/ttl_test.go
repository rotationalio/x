package cache_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/cache"
)

type mayfly struct {
	expired bool
	touched int
}

func (m *mayfly) Touch() {
	m.touched++
}

func (m *mayfly) Expired() bool {
	return m.expired
}

func TestTTL(t *testing.T) {
	t.Parallel()
	ttl := cache.NewTTL[string, *mayfly](8)
	alpha := &mayfly{expired: false}
	bravo := &mayfly{expired: false}

	// The cache should be empty.
	assert.Equal(t, 0, ttl.Len(), "cache should be empty")

	// Put an item into the cache.
	ttl.Put("alpha", alpha)
	assert.Equal(t, 1, ttl.Len(), "cache should have 1 item")

	// Get the item from the cache.
	value, ok := ttl.Get("alpha")
	assert.True(t, ok, "should have found the item")
	assert.False(t, value.Expired(), "item should not be expired")
	assert.Equal(t, 1, value.touched, "item should have been touched once")

	// Put another item into the cache.
	ttl.Put("bravo", bravo)
	assert.Equal(t, 2, ttl.Len(), "cache should have 2 items")

	// Get the item from the cache.
	value, ok = ttl.Get("bravo")
	assert.True(t, ok, "should have found the item")
	assert.False(t, value.Expired(), "item should not be expired")
	assert.Equal(t, 1, value.touched, "item should have been touched once")

	// Get alpha from the cache
	value, ok = ttl.Get("alpha")
	assert.True(t, ok, "should have found the item")
	assert.False(t, value.Expired(), "item should not be expired")
	assert.Equal(t, 1, value.touched, "item should have been touched once on put")

	// Overwrite alpha as expired.
	alpha.expired = true
	assert.Equal(t, 2, ttl.Len(), "cache should have 2 items")

	// Get alpha from the cache
	value, ok = ttl.Get("alpha")
	assert.False(t, ok, "should not have found the item")
	assert.Nil(t, value, "value should be nil")

	// Overwrite alpha key as bravo value and check that it is touched again.
	ttl.Put("alpha", bravo)
	assert.Equal(t, 2, ttl.Len(), "cache should have 2 items")
	value, ok = ttl.Get("alpha")
	assert.True(t, ok, "should have found the item")
	assert.False(t, value.Expired(), "item should not be expired")
	assert.Equal(t, 2, value.touched, "item should have been touched twice")

	// Delete alpha from the cache.
	ttl.Delete("alpha")
	assert.Equal(t, 1, ttl.Len(), "cache should have 1 item")
	value, ok = ttl.Get("alpha")
	assert.False(t, ok, "should not have found the item")
	assert.Nil(t, value, "value should be nil")
}

func TestTTLCacheMiss(t *testing.T) {
	// An item not in the cache should return false.
	t.Parallel()
	ttl := cache.NewTTL[string, *mayfly](8)

	// Populate the cache with some items.
	for i := range 6 {
		ttl.Put(keyOf(i), &mayfly{expired: false})
	}
	assert.Equal(t, 6, ttl.Len(), "cache should have 6 items")

	value, ok := ttl.Get("notinthecacheforsure")
	assert.False(t, ok, "should not have found the item")
	assert.Nil(t, value, "value should be nil")
}

func TestTTLCannotAddExpiredItem(t *testing.T) {
	t.Parallel()
	ttl := cache.NewTTL[string, *mayfly](8)
	alpha := &mayfly{expired: true}

	// Put an expired item into the cache.
	ttl.Put("alpha", alpha)
	assert.Equal(t, 0, ttl.Len(), "cache should be empty")

	// Get the item from the cache.
	value, ok := ttl.Get("alpha")
	assert.False(t, ok, "should not have found the item")
	assert.Nil(t, value, "value should be nil")
	assert.Equal(t, 0, alpha.touched, "item should not have been touched")
}

func TestTTLContains(t *testing.T) {
	t.Parallel()
	ttl := cache.NewTTL[string, *mayfly](8)

	// Populate the cache with some items.
	for i := range 6 {
		ttl.Put(keyOf(i), &mayfly{expired: false})
	}
	assert.Equal(t, 6, ttl.Len(), "cache should have 6 items")

	// Should not contain key zulu
	assert.False(t, ttl.Contains("zulu"), "should not contain key zulu")

	// Should contain key alpha
	assert.True(t, ttl.Contains("alpha"), "should contain key alpha")

	// Delete alpha from the cache.
	ttl.Delete("alpha")
	assert.Equal(t, 5, ttl.Len(), "cache should have 5 items")
	assert.False(t, ttl.Contains("alpha"), "should not contain key alpha")
}

func TestTTLClear(t *testing.T) {
	t.Parallel()
	ttl := cache.NewTTL[string, *mayfly](8)

	// Populate the cache with some items.
	vals := make([]*mayfly, 0, len(keys))
	for _, key := range keys {
		val := &mayfly{expired: false}
		vals = append(vals, val)
		ttl.Put(key, val)
	}
	assert.Equal(t, len(keys), ttl.Len(), "cache should have %d items", len(keys))

	// Expire every third item.
	for i := 0; i < len(vals); i += 3 {
		vals[i].expired = true
	}

	// Clear the cache.
	ttl.Clear()
	assert.Equal(t, (len(vals)/3)*2, ttl.Len(), "cache should be cleared of expired items")
}

func TestTTLRun(t *testing.T) {
	t.Parallel()
	ttl := cache.NewTTL[string, *mayfly](8)

	// Populate the cache with some items.
	vals := make([]*mayfly, 0, len(keys))
	for _, key := range keys {
		val := &mayfly{expired: false}
		vals = append(vals, val)
		ttl.Put(key, val)
	}
	assert.Equal(t, len(keys), ttl.Len(), "cache should have %d items", len(keys))

	// Expire every third item.
	for i := 0; i < len(vals); i += 3 {
		vals[i].expired = true
	}

	// Run the cache clear routine.
	err := ttl.Run(1 * time.Millisecond)
	assert.Ok(t, err, "should not have error running cache clear routine")
	defer ttl.Stop()

	// Cannot run the cache clear routine twice without error
	err = ttl.Run(1 * time.Millisecond)
	assert.ErrorIs(t, err, cache.ErrAlreadyRunning, "should have error running cache clear routine twice")

	time.Sleep(5 * time.Millisecond)
	assert.Equal(t, (len(vals)/3)*2, ttl.Len(), "cache should be cleared of expired items")
}

func TestTTLThreadSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race detector test in short mode")
		return
	}

	// This test should be run with the race detector enabled.
	ttl := cache.NewTTL[string, *mayfly](8)

	// Run the cache clear routine.
	ttl.Run(10 * time.Millisecond)
	defer ttl.Stop()

	// Create values for all keys
	vals := make([]*mayfly, 0, len(keys))
	for range keys {
		vals = append(vals, &mayfly{expired: false})
	}

	// Get a random item from the cache.
	randmayfly := func() (string, *mayfly) {
		key, i := randitem()
		return key, vals[i%len(vals)]
	}

	// Run 8 go routines that will perform random operations on the cache.
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 128 {
				key, val := randmayfly()
				switch rand.Intn(6) {
				case 0:
					ttl.Len()
				case 1:
					ttl.Get(key)
				case 2:
					ttl.Put(key, val)
				case 3:
					ttl.Delete(key)
				case 4:
					ttl.Contains(key)
				case 5:
					ttl.Clear()
				}

				time.Sleep(randsleep())
			}
		}()
	}
	wg.Wait()

	assert.GreaterEqual(t, 1, ttl.Len(), "cache should have at least 1 item")
	assert.LessEqual(t, len(keys), ttl.Len(), "cache should have at most %d items", len(keys))
}
