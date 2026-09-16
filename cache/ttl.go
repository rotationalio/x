package cache

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrAlreadyRunning = errors.New("cache clear routine is already running")
)

// Perishable is an interface that represents an item that can be expired and is used
// in the values of a TTL cache. The cache itself does not handle expiration, it is up
// to the Perishable implementation to do so.
type Perishable interface {
	// Touch is used to update the last accessed time of the item. This can be used to
	// extend the time to live of the item if necessary.
	Touch()

	// If Expired returns true, the item will be evicted from the cache.
	Expired() bool
}

// TTL is an unbounded cache that stores items that can expire. The cache itself does
// not handle expiration, each individual perishable item may have its own time to live.
type TTL[K comparable, V Perishable] struct {
	sync.RWMutex
	items map[K]V
	stop  chan<- struct{}
}

// NewTTL creates a new TTL cache with the given size. The size simply initializes the
// internal map to the given size to avoid reallocating as items are added. However, the
// cache is unbounded and will not evict items based on the size.
func NewTTL[K comparable, V Perishable](size int) *TTL[K, V] {
	return &TTL[K, V]{
		items: make(map[K]V, size),
	}
}

// Len returns the number of items in the cache.
func (c *TTL[K, V]) Len() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.items)
}

// Put inserts a value into the cache for the given key. The value is then touched to
// ensure its TTL is set from the time it was added to the cache.
func (c *TTL[K, V]) Put(key K, value V) {
	// Do not add the item to the cache if it is expired.
	if value.Expired() {
		return
	}

	c.Lock()
	defer c.Unlock()
	c.items[key] = value
	value.Touch()
}

// Get retrieves a value from the cache for the given key. If the key does not exist, or
// the value is expired, the zero value of V and false will be returned.
func (c *TTL[K, V]) Get(key K) (value V, ok bool) {
	c.RLock()
	value, ok = c.items[key]

	// If the item is expired, used double checked locking to remove the item from the cache.
	if ok && value.Expired() {
		c.RUnlock()
		c.Lock()
		defer c.Unlock()

		if value, ok = c.items[key]; ok && value.Expired() {
			delete(c.items, key)
		}

		// Always return the zero value of V because the item was not found in the
		// cache at the time of the read lock, and we stay consistent even if the item
		// was modified between the read unlock and the write lock.
		var zero V
		return zero, false
	}

	c.RUnlock()
	return value, ok
}

// Delete removes the item from the cache for the given key.
func (c *TTL[K, V]) Delete(key K) {
	c.Lock()
	defer c.Unlock()
	delete(c.items, key)
}

// Contains checks if the key is in the cache. If the item is expired, contains will return false.
func (c *TTL[K, V]) Contains(key K) bool {
	c.RLock()
	defer c.RUnlock()
	if value, ok := c.items[key]; ok {
		return !value.Expired()
	}
	return false
}

// Clear is used to remove the cache of any expired items. During the clear operation,
// the cache is locked and no other operations can be performed on the cache.
func (c *TTL[K, V]) Clear() {
	c.Lock()
	defer c.Unlock()
	for key, value := range c.items {
		if value.Expired() {
			delete(c.items, key)
		}
	}
}

// Run starts a go routine that will periodically clear the cache of any expired items.
// If the cache is already running, ErrAlreadyRunning will be returned.
func (c *TTL[K, V]) Run(interval time.Duration) error {
	c.Lock()
	defer c.Unlock()

	if c.stop != nil {
		return ErrAlreadyRunning
	}

	stop := make(chan struct{}, 1)
	c.stop = stop

	go func(stop <-chan struct{}) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.Clear()
			case <-stop:
				return
			}
		}
	}(stop)
	return nil
}

// Stop stops the cache clear routine.
func (c *TTL[K, V]) Stop() {
	c.Lock()
	defer c.Unlock()
	if c.stop != nil {
		close(c.stop)
		c.stop = nil
	}
}
