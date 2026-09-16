package cache

import (
	"fmt"
	"time"

	"go.rtnl.ai/x/cache/internal"
)

var (
	ErrInvalidSize = fmt.Errorf("size must be greater than 0")
)

// Functions of this type get called when a cache entry is evicted.
type EvictionCallback[K comparable, V any] func(key K, value V)

// Implements a very simple LRU cache with a fixed size and expiration time. This LRU
// uses a doubly linked list to track the recently used objects in the cache. Entries
// may also have an expiration time, and will be evicted from the cache if they are
// expired. This implementation is thread-safe and can be used concurrently by multiple
// goroutines.
//
// For higher performance and more features, consider using a more robust solution such
// as Hashicorp's LRU package which provides a TwoQueue cache to track both recently and
// frequently used objects.
type LRU[K comparable, V any] struct {
	size    int
	list    *internal.List[K, V]
	items   map[K]*internal.Entry[K, V]
	onEvict EvictionCallback[K, V]
}

// Constructs an LRU cache of the given size.
func NewLRU[K comparable, V any](size int, onEvict EvictionCallback[K, V]) (*LRU[K, V], error) {
	if size <= 0 {
		return nil, ErrInvalidSize
	}

	c := &LRU[K, V]{
		size:    size,
		list:    internal.NewList[K, V](),
		items:   make(map[K]*internal.Entry[K, V]),
		onEvict: onEvict,
	}

	return c, nil
}

// Len returns the number of items in the cache.
func (c *LRU[K, V]) Len() int {
	return c.list.Len()
}

// Put a value into the cache. Returns true if an eviction occurred.
// If the key had a previous expiration time, it will not be overridden.
func (c *LRU[K, V]) Put(key K, value V) (evicted bool) {
	return c.put(key, value, nil)
}

// Put a value into the cache with a time to live. Returns true if an eviction occurred.
// The expiration time is the absolute time when the item will expire. Any cache
// requests for an expired item will not return a cached value. Setting the expiration
// to a zero valued time will cause the item to not expire based on time.
func (c *LRU[K, V]) PutExpiry(key K, value V, expires time.Time) (evicted bool) {
	return c.put(key, value, &expires)
}

func (c *LRU[K, V]) put(key K, value V, expires *time.Time) (evicted bool) {
	// If the item already exists, move it to the front of the list and update the value.
	if entry, ok := c.items[key]; ok {
		c.list.Forward(entry)
		entry.Value = value
		if expires != nil {
			entry.Expires = *expires
		}
		return false
	}

	// Add a new item to the cache if needed.
	var entry *internal.Entry[K, V]
	if expires != nil {
		entry = c.list.Push(key, value, *expires)
	} else {
		entry = c.list.Push(key, value, time.Time{})
	}

	// Add the entry to the items map and truncate the cache if needed.
	c.items[key] = entry
	evicted = c.list.Len() > c.size
	if evicted {
		c.truncate()
	}
	return evicted
}

// Get a value from the cache. Returns the value, and true if the item was found. If
// the item is not found or the item is expired, the value will be the zero value of V
// and ok will be false.
func (c *LRU[K, V]) Get(key K) (value V, ok bool) {
	var entry *internal.Entry[K, V]
	if entry, ok = c.items[key]; ok {
		if entry.Expired() {
			c.evict(entry)
			return value, false
		}

		// Otherwise, move the entry to the front of the list and return the value.
		// Moving the entry to the front of the list updates the access order, e.g. it
		// was the most recently used item so it should be last to be evicted.
		c.list.Forward(entry)
		return entry.Value, true
	}

	// Return the zero value of V and false to indicate that the item was not found.
	return
}

// Peek returns the key value and expiration (or undefined if not found) without
// updating the access order (meaning it does not make it recently used).
func (c *LRU[K, V]) Peek(key K) (value V, expires time.Time, ok bool) {
	var entry *internal.Entry[K, V]
	if entry, ok = c.items[key]; ok {
		if entry.Expired() {
			return value, expires, false
		}
		return entry.Value, entry.Expires, true
	}
	return
}

// Evict removes the specified key from the cache and calls the eviction callback.
// It does not truncate the LRU cache or remove any expired items.
func (c *LRU[K, V]) Evict(key K) {
	if entry, ok := c.items[key]; ok {
		c.evict(entry)
	}
}

// Contains checks if the key is in the cache without updating the access order or
// evicting any items for being stale or expired. If the item is expired, contains
// will return false.
func (c *LRU[K, V]) Contains(key K) bool {
	if entry, ok := c.items[key]; ok {
		return !entry.Expired()
	}
	return false
}

// Purge is used to completely clear the cache of all items. The eviction callback is
// called for each item currently in the cache (whether or not they are expired).
func (c *LRU[K, V]) Purge() {
	for key, entry := range c.items {
		if c.onEvict != nil {
			c.onEvict(key, entry.Value)
		}
		delete(c.items, key)
	}
	c.list.Clear()
}

// truncate removes the oldest item in the cache, as well as any expired items at the
// tail of the list. NOTE: this will cause at least one item to be evicted.
func (c *LRU[K, V]) truncate() {
	// Remove the oldest item from the cache.
	if oldest := c.list.Tail(); oldest != nil {
		c.evict(oldest)
	}

	// Remove any expired items from the cache.
	entry := c.list.Tail()
	for entry != nil {
		prev := entry.Prev()
		if entry.Expired() {
			c.evict(entry)
			entry = prev
			continue
		}

		// Stop iterating if the entry is not expired.
		break
	}
}

// remove an entry from the cache and call the eviction callback.
func (c *LRU[K, V]) evict(entry *internal.Entry[K, V]) {
	c.list.Remove(entry)
	delete(c.items, entry.Key)
	if c.onEvict != nil {
		c.onEvict(entry.Key, entry.Value)
	}
}
