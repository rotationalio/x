package cache

import (
	"sync"
)

// cell holds one map entry with its own mutex so [SafeMap.GetOrCreate] can run the
// create callback without holding a package-wide lock; only the same key serializes.
type cell[V any] struct {
	mu sync.Mutex
	v  V
	ok bool
}

// SafeMap stores values in a concurrency-safe fashion. Key type K must be
// comparable.
//
// Values live in per-key [cell] wrappers: [GetOrCreate] locks only the cell for
// that key while create runs, so work for different keys proceeds in parallel.
type SafeMap[K comparable, V any] struct {
	m sync.Map // K -> *cell[V]
}

// Put inserts a value into the cache for the given key, overwriting any
// existing value.
func (c *SafeMap[K, V]) Put(id K, value V) {
	x, _ := c.m.LoadOrStore(id, &cell[V]{})
	ac := x.(*cell[V])
	ac.mu.Lock()
	ac.v = value
	ac.ok = true
	ac.mu.Unlock()
}

// LoadOrStore returns the stored value, inserting value if the key is absent.
// If the key exists only as an empty placeholder from an in-flight [GetOrCreate],
// the provided value is installed and (value, false) is returned.
// The loaded result reports whether the key already held a committed value.
func (c *SafeMap[K, V]) LoadOrStore(id K, value V) (actual V, loaded bool) {
	newC := &cell[V]{v: value, ok: true}
	x, loaded := c.m.LoadOrStore(id, newC)
	ac := x.(*cell[V])
	if !loaded {
		return value, false
	}
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if !ac.ok {
		ac.v = value
		ac.ok = true
		return value, false
	}
	return ac.v, true
}

// GetOrCreate returns the value for the given key, creating it with create if
// necessary. The create function runs without blocking other keys; concurrent
// callers for the same key share one serialized create.
func (c *SafeMap[K, V]) GetOrCreate(id K, create func(K) (V, error)) (V, error) {
	x, _ := c.m.LoadOrStore(id, &cell[V]{})
	ac := x.(*cell[V])
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if ac.ok {
		return ac.v, nil
	}
	v, err := create(id)
	if err != nil {
		var zero V
		return zero, err
	}
	ac.v = v
	ac.ok = true
	return v, nil
}

// Load returns (v, true) when the key exists and a value was committed; otherwise (zero, false).
// A key reserved by an in-flight [GetOrCreate] (uncommitted placeholder) yields (zero, false).
func (c *SafeMap[K, V]) Load(id K) (v V, ok bool) {
	x, exists := c.m.Load(id)
	if !exists {
		var zero V
		return zero, false
	}
	ac := x.(*cell[V])
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if !ac.ok {
		var zero V
		return zero, false
	}
	return ac.v, true
}

// Get returns the value for the given key, or the zero value of V if not
// present or not yet committed (e.g. [GetOrCreate] still running).
func (c *SafeMap[K, V]) Get(id K) V {
	x, ok := c.m.Load(id)
	if !ok {
		var zero V
		return zero
	}
	ac := x.(*cell[V])
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if !ac.ok {
		var zero V
		return zero
	}
	return ac.v
}

// FindKey returns the first key whose value satisfies predicate, or the zero
// value of K and false if none match. Uncommitted placeholders are skipped.
func (c *SafeMap[K, V]) FindKey(predicate func(V) bool) (k K, found bool) {
	c.m.Range(func(key, value any) bool {
		ac := value.(*cell[V])
		ac.mu.Lock()
		v := ac.v
		ok := ac.ok
		ac.mu.Unlock()
		if ok && predicate(v) {
			k = key.(K)
			found = true
			return false
		}
		return true
	})
	return k, found
}

// Delete removes the value for the given key from the cache.
func (c *SafeMap[K, V]) Delete(id K) {
	c.m.Delete(id)
}

// CloseAll calls closeFn for each committed (key, value) and then deletes each pair from
// the cache.
func (c *SafeMap[K, V]) CloseAll(closeFn func(K, V)) {
	c.m.Range(func(key, value any) bool {
		k := key.(K)
		ac := value.(*cell[V])
		ac.mu.Lock()
		v := ac.v
		ok := ac.ok
		ac.mu.Unlock()
		if ok {
			closeFn(k, v)
		}
		c.m.Delete(key)
		return true
	})
}
