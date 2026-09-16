package cache_test

import (
	"math/rand"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/cache"
)

// 16 keys used to test the LRU cache
var keys = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
}

func keyOf(i int) string {
	return keys[i%len(keys)]
}

func TestLRURequireSize(t *testing.T) {
	t.Parallel()
	invalidSizes := []int{0, -1, -42, -1064}
	for _, size := range invalidSizes {
		_, err := cache.NewLRU[string, int](size, nil)
		assert.ErrorIs(t, err, cache.ErrInvalidSize, "should have returned an invalid size error")
	}
}

func TestLRUMaxSize(t *testing.T) {
	t.Parallel()
	var evictions int
	onEvict := func(key string, value int) {
		evictions++
	}

	lru, err := cache.NewLRU(8, onEvict)
	assert.Ok(t, err, "could not create LRU cache")

	// Put 16 items into the cache.
	for i := range 16 {
		lru.Put(keyOf(i), i)
	}

	// The cache should have 8 items.
	assert.Equal(t, 8, lru.Len(), "cache should have 8 items")

	// Evictions should have occurred for 8 items.
	assert.Equal(t, 8, evictions, "should have evicted 8 items")
}

func TestLRU(t *testing.T) {
	t.Parallel()
	lru, err := cache.NewLRU[string, int](8, nil)
	assert.Ok(t, err, "could not create LRU cache")

	// The LRU should be empty.
	assert.Equal(t, 0, lru.Len(), "cache should be empty")

	// Put an item into the cache.
	lru.Put("alpha", 42)
	assert.Equal(t, 1, lru.Len(), "cache should have 1 item")

	value, ok := lru.Get("alpha")
	assert.True(t, ok, "should have found the item")
	assert.Equal(t, 42, value, "value should be 42")

	// Put another item into the cache.
	lru.Put("bravo", 12)
	assert.Equal(t, 2, lru.Len(), "cache should have 2 items")

	value, ok = lru.Get("bravo")
	assert.True(t, ok, "should have found the item")
	assert.Equal(t, 12, value, "value should be 12")

	// Put the same item again should update the value.
	lru.Put("alpha", 7)
	assert.Equal(t, 2, lru.Len(), "cache should have 2 items")

	value, ok = lru.Get("alpha")
	assert.True(t, ok, "should have found the item")
	assert.Equal(t, 7, value, "value should be 7")

	// An item that is not in the cache should return false.
	value, ok = lru.Get("charlie")
	assert.False(t, ok, "should not have found the item")
	assert.Equal(t, 0, value, "value should be 0")

	// Peek an item that is not in the cache should return false.
	value, expires, ok := lru.Peek("charlie")
	assert.False(t, ok, "should not have found the item")
	assert.Equal(t, 0, value, "value should be 0")
	assert.True(t, expires.IsZero(), "expires should be zero time")

	// Peek an item that is in the cache should return the value and expiration time.
	value, expires, ok = lru.Peek("alpha")
	assert.True(t, ok, "should have found the item")
	assert.Equal(t, 7, value, "value should be 7")
	assert.True(t, expires.IsZero(), "expires should not be zero time")
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	evicted := make([]string, 0, 8)
	onEvict := func(key string, value int) {
		evicted = append(evicted, key)
	}

	lru, err := cache.NewLRU(8, onEvict)
	assert.Ok(t, err, "could not create LRU cache")

	// Put 8 items into the cache with a specific order.
	// The least recently used item should be evicted first.
	lru.PutExpiry("alpha", 1, time.Now().Add(73*time.Second))     // least recently used, not expired
	lru.PutExpiry("bravo", 2, time.Now().Add(-97*time.Second))    // second least recently used, expired
	lru.PutExpiry("charlie", 3, time.Now().Add(-121*time.Second)) // third least recently used, expired
	lru.PutExpiry("delta", 4, time.Now().Add(43*time.Second))     // fourth least recently used, not expired
	lru.PutExpiry("echo", 5, time.Now().Add(-178*time.Second))    // fifth least recently used, expired
	lru.PutExpiry("foxtrot", 6, time.Now().Add(109*time.Second))  // sixth least recently used, not expired
	lru.PutExpiry("golf", 7, time.Now().Add(-212*time.Second))    // seventh least recently used, expired
	lru.PutExpiry("hotel", 8, time.Now().Add(146*time.Second))    // eighth least recently used, not expired

	// NOTE: this works because the cache is only truncated when the size is exceeded
	// and there are no checks of the expiration time when adding items to the cache.
	// If this semantic changes, then this test will need to be updated.
	assert.Equal(t, 8, lru.Len(), "cache should have 8 items")

	// When we add the 9th item, the cache will be truncated and the least recently
	// used item will be evicted (even though it is not expired). as will bravo and
	// charlie since they are expired.
	lru.PutExpiry("india", 9, time.Now().Add(182*time.Second)) // ninth least recently used, not expired
	assert.Equal(t, 6, lru.Len(), "cache should have 5 items")

	// Evicted in the order that they were evicted.
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, evicted)

	// Check that the items that should still be in the cache are still in the cache.
	for _, key := range []string{"alpha", "bravo", "charlie"} {
		assert.False(t, lru.Contains(key), "should contain %q", key)
	}

	// Note: while echo and golf are in the cache, they don't return true from Contains
	// because they are expired. (The eviction and length checks above guarantee it)
	for _, key := range []string{"delta", "foxtrot", "hotel", "india"} {
		assert.True(t, lru.Contains(key), "should contain %q", key)
	}
}

func TestLRUPurge(t *testing.T) {
	t.Parallel()
	var evictions int
	onEvict := func(key string, value int) {
		evictions++
	}

	lru, err := cache.NewLRU(8, onEvict)
	assert.Ok(t, err, "could not create LRU cache")

	// Put 8 items into the cache.
	for i := range 8 {
		lru.Put(keyOf(i), i)
	}

	assert.Equal(t, 8, lru.Len(), "cache should have 8 items")
	assert.Equal(t, 0, evictions, "should not have evicted any items")

	// Purge the cache.
	lru.Purge()
	assert.Equal(t, 0, lru.Len(), "cache should be empty")
	assert.Equal(t, 8, evictions, "should have evicted all items")
}

func TestLRUSingleItem(t *testing.T) {
	t.Parallel()
	lru, err := cache.NewLRU[string, int](1, nil)
	assert.Ok(t, err, "could not create LRU cache")

	// The LRU should be empty.
	assert.Equal(t, 0, lru.Len(), "cache should be empty")

	for i := range 64 {
		lru.Put(keyOf(i), i)
		assert.Equal(t, 1, lru.Len(), "cache should have 1 item")
		value, ok := lru.Get(keyOf(i))
		assert.True(t, ok, "should have found the item")
		assert.Equal(t, i, value, "value should be %d", i)
	}

	assert.Equal(t, 1, lru.Len(), "cache should have 1 item")
	value, ok := lru.Get("papa")
	assert.True(t, ok, "should have found the item")
	assert.Equal(t, 63, value, "value should be 63")
}

func TestGetExpired(t *testing.T) {
	t.Parallel()
	lru, err := cache.NewLRU[string, int](8, nil)
	assert.Ok(t, err, "could not create LRU cache")

	// Put an item into the cache with an expiration time in the past.
	expires := time.Now().Add(-69 * time.Second)
	lru.PutExpiry("alpha", 42, expires)
	assert.Equal(t, 1, lru.Len(), "cache should have 1 item")

	// Peeking an expired item should return true and the expiration time.
	// BUT NOT CHANGE THE SIZE OF THE CACHE.
	value, expires, ok := lru.Peek("alpha")
	assert.False(t, ok, "should not have found the item")
	assert.Equal(t, 0, value, "value should be 0")
	assert.True(t, expires.IsZero(), "expires should be zero time")

	assert.Equal(t, 1, lru.Len(), "cache should have 1 item, peek should not change the size")

	// Contains should return false for an expired item.
	// BUT NOT CHANGE THE SIZE OF THE CACHE.
	assert.False(t, lru.Contains("alpha"), "should not contain the item")
	assert.Equal(t, 1, lru.Len(), "cache should have 1 item, contains should not change the size")

	// Getting an expired item should return false.
	// BUT NOT CHANGE THE SIZE OF THE CACHE.
	value, ok = lru.Get("alpha")
	assert.False(t, ok, "should not have found the item")
	assert.Equal(t, 0, value, "value should be 0")

	// The item should be evicted from the cache.
	assert.Equal(t, 0, lru.Len(), "cache should be empty")
}

func randitem() (string, int) {
	val := rand.Intn(1000)
	key := keyOf(val)
	return key, val
}

func randsleep() time.Duration {
	return time.Duration(rand.Intn(10)+5) * time.Millisecond
}
