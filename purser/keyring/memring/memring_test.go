package memring_test

// Tests for the in-memory Keyring implementation.

import (
	"sync"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring/keyringtest"
	"go.rtnl.ai/x/purser/keyring/memring"
)

//=============================================================================
// Tests: conformance
//=============================================================================

// TestMemring_conformance runs the shared keyring conformance suite.
func TestMemring_conformance(t *testing.T) {
	keyringtest.KeyringConforms(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
		return memring.New(active, others...)
	})
}

//=============================================================================
// Tests: memring-specific behavior
//=============================================================================

// TestNew_nilOther rejects a nil locker in the others list.
func TestNew_nilOther(t *testing.T) {
	lck := newTestLocker(t, nulllocker.VariantA, "seed-a")
	_, err := memring.New(lck, nil)
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestRouteKeyID_multiVersion registers two null locker variants with different KeyID
// lengths and verifies RouteKeyID dispatches each wire blob to the locker that produced
// it. Cross-magic-prefix routing (NULL vs PURS) is exercised separately by purser's
// multi-version tests; this test focuses on the keyring's per-locker ParseKeyID
// fallback when the keyring contains multiple registered lockers.
func TestRouteKeyID_multiVersion(t *testing.T) {
	lckA := newTestLocker(t, nulllocker.VariantA, "seed-a")
	lckB := newTestLocker(t, nulllocker.VariantB, "seed-b")
	mr, err := memring.New(lckA, lckB)
	assert.Ok(t, err)

	wireA, err := lckA.Seal("ns", []byte("aaaa"))
	assert.Ok(t, err)
	wireB, err := lckB.Seal("ns", []byte("bbbb"))
	assert.Ok(t, err)

	// Each wire blob routes back to the locker that produced it.
	foundA, err := mr.RouteKeyID(wireA)
	assert.Ok(t, err)
	assert.Equal(t, lckA.KeyID(), foundA.KeyID())

	foundB, err := mr.RouteKeyID(wireB)
	assert.Ok(t, err)
	assert.Equal(t, lckB.KeyID(), foundB.KeyID())
}

//=============================================================================
// Tests: concurrency
//=============================================================================

// TestConcurrentAccess exercises Register, Lookup, Active, SetActive, and RouteKeyID
// under concurrent goroutine pressure. Primarily a race-detector probe; success
// criterion is "no race, no deadlock, no panic." We include Register with fresh
// lockers per iteration so the structure-mutating path (not just active-pointer swap)
// is contended.
func TestConcurrentAccess(t *testing.T) {
	lckActive := newTestLocker(t, nulllocker.VariantA, "active")
	mr, err := memring.New(lckActive)
	assert.Ok(t, err)
	wire, err := lckActive.Seal("ns", []byte("hello"))
	assert.Ok(t, err)

	// Pre-build a small pool of additional lockers each goroutine can attempt to
	// Register; only the first goroutine to register a given key id will succeed,
	// later attempts return ErrDuplicateKeyID — that's expected and ignored.
	const goroutines = 16
	const iters = 32
	others := make([]contract.Locker, goroutines)
	for i := range others {
		others[i] = newTestLocker(t, nulllocker.VariantC, "concurrent-seed-"+string(rune('a'+i)))
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			for range iters {
				_ = mr.Active()
				_, _ = mr.Lookup(lckActive.KeyID())
				_, _ = mr.RouteKeyID(wire)
				_ = mr.SetActive(lckActive)
				_ = mr.Register(others[i])
			}
		}(i)
	}
	wg.Wait()
}

//=============================================================================
// Helpers
//=============================================================================

// newTestLocker returns a null locker; tests use it whenever they need "some locker"
// rather than a specific cryptographic implementation.
func newTestLocker(tb testing.TB, variant nulllocker.Variant, seed string) contract.Locker {
	tb.Helper()
	lck, err := nulllocker.New(tb, variant, []byte(seed))
	assert.Ok(tb, err)
	return lck
}
