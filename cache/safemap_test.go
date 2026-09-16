package cache_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/cache/cache"
)

// TestDoubleCheckedCache_FindKey checks that FindKey returns the correct key
// when the predicate matches, returns false when nothing matches, and handles
// an empty cache. It does not stress concurrency.
func TestDoubleCheckedCache_FindKey(t *testing.T) {
	c := cache.SafeMap[string, int]{}
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	t.Run("finds key when predicate matches", func(t *testing.T) {
		k, found := c.FindKey(func(v int) bool { return v == 2 })
		assert.True(t, found)
		assert.Equal(t, "b", k)
	})

	t.Run("returns first matching key", func(t *testing.T) {
		k, found := c.FindKey(func(v int) bool { return v > 1 })
		assert.True(t, found)
		assert.Assert(t, k == "b" || k == "c", "expected key b or c, got %q", k)
	})

	t.Run("returns false when no match", func(t *testing.T) {
		k, found := c.FindKey(func(v int) bool { return v == 99 })
		assert.False(t, found)
		assert.Equal(t, "", k)
	})

	t.Run("empty cache returns false", func(t *testing.T) {
		empty := cache.SafeMap[string, int]{}
		k, found := empty.FindKey(func(v int) bool { return true })
		assert.False(t, found)
		assert.Equal(t, "", k)
	})
}

// TestDoubleCheckedCache_ConcurrentGetPutDeleteFindKey stresses the cache under
// mixed concurrent read/write load. Many goroutines repeatedly perform Put, Get,
// FindKey, and Delete on a shared set of keys. Run with -race to verify.
func TestDoubleCheckedCache_ConcurrentGetPutDeleteFindKey(t *testing.T) {
	c := cache.SafeMap[int, int]{}

	const numKeys = 32
	const numGoroutines = 32
	const opsPerGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := (worker + i) % numKeys
				switch i % 4 {
				case 0:
					c.Put(key, key*10)
				case 1:
					_ = c.Get(key)
				case 2:
					// FindKey holds RLock; if we find a value, key must match.
					k, found := c.FindKey(func(v int) bool { return v == key*10 })
					if found {
						assert.Equal(t, key, k)
					}
				case 3:
					c.Delete(key)
				}
			}
		}(g)
	}
	wg.Wait()
}

// TestDoubleCheckedCache_GetOrCreateOnlyCreatesOnce verifies that when many
// goroutines call GetOrCreate for the same key, the create function is invoked
// only once (per-key mutex; other keys do not participate).
func TestDoubleCheckedCache_GetOrCreateOnlyCreatesOnce(t *testing.T) {
	c := cache.SafeMap[string, int]{}

	const key = "single"
	var createCalls atomic.Int32

	create := func(k string) (int, error) {
		createCalls.Add(1)
		return 42, nil
	}

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			v, err := c.GetOrCreate(key, create)
			assert.Ok(t, err)
			assert.Equal(t, 42, v)
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), createCalls.Load(), "create should be called exactly once")
	assert.Equal(t, 42, c.Get(key))
}

// TestDoubleCheckedCache_ConcurrentGetOrCreateDifferentKeys stresses GetOrCreate
// when many different keys are requested concurrently. Each key is requested
// by multiple goroutines; all should receive the same value and the final cache
// state must be consistent (no lost or duplicate entries due to lock issues).
func TestDoubleCheckedCache_ConcurrentGetOrCreateDifferentKeys(t *testing.T) {
	c := cache.SafeMap[int, int]{}

	const numKeys = 50
	const numGoroutines = 4

	var wg sync.WaitGroup
	wg.Add(numGoroutines * numKeys)
	for g := 0; g < numGoroutines; g++ {
		for k := 0; k < numKeys; k++ {
			key := k
			go func() {
				defer wg.Done()
				v, err := c.GetOrCreate(key, func(id int) (int, error) { return id * 2, nil })
				assert.Ok(t, err)
				assert.Equal(t, key*2, v)
			}()
		}
	}
	wg.Wait()

	// Final state: every key present with correct value.
	for k := 0; k < numKeys; k++ {
		assert.Equal(t, k*2, c.Get(k))
	}
}

// TestDoubleCheckedCache_FindKeyConcurrentWithWrites runs readers (FindKey, Get)
// concurrently with writers (Put).
func TestDoubleCheckedCache_FindKeyConcurrentWithWrites(t *testing.T) {
	c := cache.SafeMap[int, int]{}

	for i := 0; i < 20; i++ {
		c.Put(i, i)
	}

	const numGoroutines = 24
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				key := (worker + i) % 30
				// Spread work: 1/3 writers, 1/3 FindKey, 1/3 Get.
				switch worker % 3 {
				case 0:
					c.Put(key, key)
				case 1:
					c.FindKey(func(v int) bool { return v == key })
				default:
					_ = c.Get(key)
				}
			}
		}(g)
	}
	wg.Wait()
}

// TestDoubleCheckedCache_CloseAllUnderConcurrentAccess runs Get and FindKey in
// several goroutines while another goroutine calls CloseAll. Verifies no deadlock
// and no panic from concurrent access.
func TestDoubleCheckedCache_CloseAllUnderConcurrentAccess(t *testing.T) {
	c := cache.SafeMap[int, int]{}
	for i := 0; i < 20; i++ {
		c.Put(i, i)
	}

	var wg sync.WaitGroup
	wg.Add(10)
	for g := 0; g < 10; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = c.Get(i % 20)
				c.FindKey(func(v int) bool { return v == i%20 })
			}
		}()
	}
	closed := make(chan struct{})
	go func() {
		c.CloseAll(func(k, v int) {})
		close(closed) // signal that CloseAll finished
	}()
	wg.Wait()
	<-closed // wait for CloseAll so the test does not exit with goroutines still running
}

// TestSafeMap_GetOrCreateDifferentKeysParallelCreate asserts that a blocked create
// for key A does not prevent GetOrCreate for key B from completing.
func TestSafeMap_GetOrCreateDifferentKeysParallelCreate(t *testing.T) {
	c := cache.SafeMap[int, int]{}

	blockA := make(chan struct{})
	doneA := make(chan struct{})
	var callsA atomic.Int32

	go func() {
		defer close(doneA)
		_, _ = c.GetOrCreate(1, func(int) (int, error) {
			callsA.Add(1)
			<-blockA
			return 1, nil
		})
	}()

	deadline := time.Now().Add(time.Second)
	for callsA.Load() != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, int32(1), callsA.Load(), "create for key 1 should start")

	doneB := make(chan struct{})
	go func() {
		defer close(doneB)
		v, err := c.GetOrCreate(2, func(int) (int, error) { return 2, nil })
		assert.Ok(t, err)
		assert.Equal(t, 2, v)
	}()

	select {
	case <-doneB:
	case <-time.After(2 * time.Second):
		close(blockA)
		t.Fatal("GetOrCreate for key 2 blocked while key 1’s create was waiting")
	}

	close(blockA)
	<-doneA
}
