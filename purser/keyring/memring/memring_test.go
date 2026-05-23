package memring_test

// Tests for the in-memory Keyring implementation.

import (
	"sync"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring"
	"go.rtnl.ai/x/purser/keyring/keyringtest"
	"go.rtnl.ai/x/purser/keyring/memring"
	"go.rtnl.ai/x/purser/locker"
)

//=============================================================================
// Tests: conformance
//=============================================================================

// TestMemring_conformance runs the shared keyring conformance suite.
func TestMemring_conformance(t *testing.T) {
	keyringtest.KeyringConforms(t, func() (keyring.Keyring, error) {
		return memring.New(), nil
	})
}

//=============================================================================
// Tests: memring-specific behavior
//=============================================================================

// TestBind_nilLocker rejects a nil locker.
func TestBind_nilLocker(t *testing.T) {
	mr := memring.New()
	err := mr.Bind("tenant", nil)
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestBind_emptyNamespace rejects an empty namespace string.
func TestBind_emptyNamespace(t *testing.T) {
	lck := newTestLocker(t, nulllocker.VariantA, "empty-ns")
	mr := memring.New()
	err := mr.Bind("", lck)
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestUnbind_emptyNamespace rejects an empty namespace string.
func TestUnbind_emptyNamespace(t *testing.T) {
	mr := memring.New()
	_, err := mr.Unbind("")
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestLockerFor_emptyNamespaceUsesDefault verifies an empty namespace uses SetDefault for sealing.
func TestLockerFor_emptyNamespaceUsesDefault(t *testing.T) {
	lck := newTestLocker(t, nulllocker.VariantA, "default-seed")
	mr := memring.New()
	assert.Ok(t, mr.SetDefault(lck))

	got, err := mr.LockerFor("")
	assert.Ok(t, err)
	assert.Equal(t, lck.KeyID(), got.KeyID())
}

// TestRoute_multiVersion registers two null locker variants and verifies Route
// dispatches each wire blob to the locker that produced it.
func TestRoute_multiVersion(t *testing.T) {
	lckA := newTestLocker(t, nulllocker.VariantA, "seed-a")
	lckB := newTestLocker(t, nulllocker.VariantB, "seed-b")
	mr := memring.New()
	assert.Ok(t, mr.SetDefault(lckA))
	assert.Ok(t, mr.Bind("b-ns", lckB))

	wireA, err := lckA.Seal("ns", []byte("aaaa"))
	assert.Ok(t, err)
	wireB, err := lckB.Seal("ns", []byte("bbbb"))
	assert.Ok(t, err)

	foundA, err := mr.Route(wireA)
	assert.Ok(t, err)
	assert.Equal(t, lckA.KeyID(), foundA.KeyID())

	foundB, err := mr.Route(wireB)
	assert.Ok(t, err)
	assert.Equal(t, lckB.KeyID(), foundB.KeyID())
}

//=============================================================================
// Tests: concurrency
//=============================================================================

// TestConcurrentAccess exercises Bind, LockerFor, SetDefault, and Route under
// concurrent goroutine pressure (race-detector probe).
func TestConcurrentAccess(t *testing.T) {
	lckDefault := newTestLocker(t, nulllocker.VariantA, "active")
	mr := memring.New()
	assert.Ok(t, mr.SetDefault(lckDefault))
	wire, err := lckDefault.Seal("ns", []byte("hello"))
	assert.Ok(t, err)

	const goroutines = 16
	const iters = 32
	others := make([]locker.Locker, goroutines)
	for i := range others {
		others[i] = newTestLocker(t, nulllocker.VariantC, "concurrent-seed-"+string(rune('a'+i)))
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			ns := "concurrent-" + string(rune('a'+i))
			for range iters {
				_, _ = mr.LockerFor("default-ns")
				_, _ = mr.Route(wire)
				_ = mr.SetDefault(lckDefault)
				_ = mr.Bind(ns, others[i])
			}
		}(i)
	}
	wg.Wait()
}

//=============================================================================
// Helpers
//=============================================================================

// newTestLocker returns a null locker for tests.
func newTestLocker(tb testing.TB, variant nulllocker.Variant, seed string) locker.Locker {
	tb.Helper()
	lck, err := nulllocker.New(tb, variant, []byte(seed))
	assert.Ok(tb, err)
	return lck
}
