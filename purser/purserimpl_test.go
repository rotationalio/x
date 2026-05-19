package purser_test

// Tests for purser orchestration and locker wiring.
//
// Most tests use the null locker (via pursertest) because they exercise the purser →
// keyring → locker → hold wiring, not the locker's crypto. Tests that specifically
// depend on the locker consuming crypto/rand.Reader (entropy-failure paths) build a
// v1-backed purser inline.

import (
	"context"
	"crypto/ecdh"
	crand "crypto/rand"
	"errors"
	"io"
	"sync"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring/memring"
	v1 "go.rtnl.ai/x/purser/locker/v1"
	"go.rtnl.ai/x/purser/pursertest"
)

//=============================================================================
// Tests: New
//=============================================================================

// TestNew_nilHold verifies New rejects a nil hold.
func TestNew_nilHold(t *testing.T) {
	// A valid keyring built around a null locker so only the hold argument is invalid.
	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	kr, err := memring.New(lck)
	assert.Ok(t, err)

	_, err = purser.New(nil, kr)
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestNew_nilKeyring verifies New rejects a nil Keyring.
func TestNew_nilKeyring(t *testing.T) {
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)
	_, err = purser.New(h, nil)
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestNew_ok verifies a minimal valid New succeeds.
func TestNew_ok(t *testing.T) {
	p, _ := newPurser(t)
	assert.NotNil(t, p)
}

//=============================================================================
// Tests: Store / Retrieve
//=============================================================================

// TestPurser_storeRetrieveRoundTrip exercises the basic happy path: Store then Retrieve.
// The active locker is a null locker so this only checks orchestration wiring; the
// envelope crypto round-trip is exercised directly in locker/v1's locker_test.go.
func TestPurser_storeRetrieveRoundTrip(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	id, err := p.Store(ctx, "ns", []byte("hello"))
	assert.Ok(t, err)
	assert.True(t, id != "", "Store: expected non-empty identifier")

	got, err := p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("hello"), got)
}

// TestPurser_retrieveMissing asserts a hex-formatted but unbound id surfaces ErrNotFound.
func TestPurser_retrieveMissing(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	_, err := p.Retrieve(ctx, "ns", "00112233445566778899aabbccddeeff")
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_retrieveMalformedIdentifier asserts an invalid hex string surfaces
// ErrInvalidIdentifier (joined with ErrHold) rather than panicking.
func TestPurser_retrieveMalformedIdentifier(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	_, err := p.Retrieve(ctx, "ns", "not-a-hex-id")
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrInvalidIdentifier)
}

//=============================================================================
// Tests: Update / CompareAndSwap / MoveNamespace / Delete
//=============================================================================

// TestPurser_update covers happy-path replacement and the missing-row failure mode.
func TestPurser_update(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	id, err := p.Store(ctx, "ns", []byte("v1"))
	assert.Ok(t, err)

	// Happy path: re-seal under the same id.
	assert.Ok(t, p.Update(ctx, "ns", id, []byte("v2")))
	got, err := p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v2"), got)

	// Update on a never-stored identifier surfaces ErrNotFound (joined with ErrHold).
	err = p.Update(ctx, "ns", "00112233445566778899aabbccddeeff", []byte("v3"))
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_compareAndSwap covers the wrong-current branch (with post-fail invariant
// check that the row is unchanged), the success branch, and the missing-row branch.
func TestPurser_compareAndSwap(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	id, err := p.Store(ctx, "ns", []byte("v1"))
	assert.Ok(t, err)

	// Wrong current — refuses to swap with ErrWrongCurrent.
	err = p.CompareAndSwap(ctx, "ns", id, []byte("wrong"), []byte("v2"))
	assert.ErrorIs(t, err, perrors.ErrWrongCurrent)

	// Post-fail invariant: the row's plaintext must still be "v1".
	got, err := p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v1"), got)

	// Correct current — swap succeeds and the new value is observable.
	assert.Ok(t, p.CompareAndSwap(ctx, "ns", id, []byte("v1"), []byte("v2")))
	got, err = p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v2"), got)

	// CAS on a missing identifier — the hold-layer Get fails first, so we surface
	// ErrHold/ErrNotFound rather than ErrWrongCurrent.
	err = p.CompareAndSwap(ctx, "ns", "aabbccddeeff00112233445566778899", []byte("a"), []byte("b"))
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_moveNamespace covers happy path, same-namespace no-op, and missing-row.
func TestPurser_moveNamespace(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	id, err := p.Store(ctx, "ns-a", []byte("v"))
	assert.Ok(t, err)

	// Successful move — re-seals under new namespace and deletes the old row.
	assert.Ok(t, p.MoveNamespace(ctx, "ns-a", "ns-b", id))
	got, err := p.Retrieve(ctx, "ns-b", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v"), got)

	_, err = p.Retrieve(ctx, "ns-a", id)
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)

	// Same-namespace move is a no-op (preserves the row in place and returns nil).
	assert.Ok(t, p.MoveNamespace(ctx, "ns-b", "ns-b", id))
	got, err = p.Retrieve(ctx, "ns-b", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v"), got)

	// Move from a missing source surfaces the hold-layer not-found error.
	err = p.MoveNamespace(ctx, "ns-x", "ns-y", "00112233445566778899aabbccddeeff")
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_moveNamespacePartialFailure asserts the post-CreateWithIdentifier
// Delete-failure branch surfaces ErrMoveNamespaceIncomplete (joined with ErrHold)
// and leaves the new row in place — i.e. the move is observable in both namespaces
// so an operator can recover.
func TestPurser_moveNamespacePartialFailure(t *testing.T) {
	ctx := context.Background()

	mem, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)
	failing := &deleteFailingHold{Hold: mem, failOn: "old-ns"}

	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	kr, err := memring.New(lck)
	assert.Ok(t, err)
	p, err := purser.New(failing, kr)
	assert.Ok(t, err)

	id, err := p.Store(ctx, "old-ns", []byte("payload"))
	assert.Ok(t, err)

	err = p.MoveNamespace(ctx, "old-ns", "new-ns", id)
	assert.ErrorIs(t, err, perrors.ErrMoveNamespaceIncomplete)
	assert.ErrorIs(t, err, perrors.ErrHold)

	// The new row landed before Delete failed — confirm it is retrievable so
	// callers can resume cleanup deterministically.
	got, err := p.Retrieve(ctx, "new-ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("payload"), got)

	// The old row is still present (Delete on the source namespace fails).
	gotOld, err := p.Retrieve(ctx, "old-ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("payload"), gotOld)
}

// TestPurser_delete covers idempotent delete, post-condition (row absent), and
// deleting a never-created row.
func TestPurser_delete(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	id, err := p.Store(ctx, "ns", []byte("v"))
	assert.Ok(t, err)

	assert.Ok(t, p.Delete(ctx, "ns", id))

	// Post-condition: the row is no longer retrievable.
	_, err = p.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)

	// Re-deleting is a no-op (idempotent).
	assert.Ok(t, p.Delete(ctx, "ns", id))

	// Deleting a never-created row is also a no-op (per MemHold contract).
	assert.Ok(t, p.Delete(ctx, "ns", "00112233445566778899aabbccddeeff"))
}

//=============================================================================
// Tests: cross-locker routing
//=============================================================================

// TestPurser_retrieveMissingLocker ensures rows sealed under one purser cannot be
// opened by a second purser whose keyring lacks the row's locker. Two null variants
// (different KeyIDs) suffice — no envelope crypto is needed for routing semantics.
func TestPurser_retrieveMissingLocker(t *testing.T) {
	ctx := context.Background()

	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	lckA, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)
	krA, err := memring.New(lckA)
	assert.Ok(t, err)
	pA, err := purser.New(h, krA)
	assert.Ok(t, err)

	lckB, err := nulllocker.New(t, nulllocker.VariantB, []byte("seedB"))
	assert.Ok(t, err)
	krB, err := memring.New(lckB)
	assert.Ok(t, err)
	pB, err := purser.New(h, krB)
	assert.Ok(t, err)

	// Seal a row through pA (lckA's key id), then try to retrieve it via pB whose
	// keyring does not know lckA.
	id, err := pA.Store(ctx, "ns", []byte("secret"))
	assert.Ok(t, err)

	_, err = pB.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

// TestPurser_keyringRouteFailurePropagates uses a stub keyring whose RouteKeyID always
// returns ErrNoLocker to confirm purser propagates that error verbatim instead of
// wrapping or swallowing it.
func TestPurser_keyringRouteFailurePropagates(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	// Real null locker so Active().Seal succeeds; the stub overrides only RouteKeyID.
	real, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	kr := &stubKeyring{
		active: real,
		route: func([]byte) (purser.Locker, error) {
			return nil, perrors.ErrNoLocker
		},
	}
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	id, err := p.Store(ctx, "ns", []byte("v"))
	assert.Ok(t, err)

	_, err = p.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

//=============================================================================
// Tests: entropy-failure paths
//
// These tests need a locker that consumes crypto/rand.Reader during Seal — the null
// locker is deterministic and would silently no-op the swap. Using a v1 locker proves
// purser correctly maps reader failures to ErrSealFailed regardless of which Seal-side
// operation triggered them (Store, Update, MoveNamespace).
//=============================================================================

// TestPurser_storeEntropyFailure ensures Store maps crand.Reader failures to ErrSealFailed.
func TestPurser_storeEntropyFailure(t *testing.T) {
	ctx := context.Background()
	p, _ := newCryptoPurser(t)

	withFailingEntropy(t)

	_, err := p.Store(ctx, "ns", []byte("x"))
	assert.ErrorIs(t, err, perrors.ErrSealFailed)
}

// TestPurser_updateEntropyFailure ensures Update propagates entropy failures.
func TestPurser_updateEntropyFailure(t *testing.T) {
	ctx := context.Background()
	p, _ := newCryptoPurser(t)

	id, err := p.Store(ctx, "ns", []byte("v"))
	assert.Ok(t, err)

	withFailingEntropy(t)

	err = p.Update(ctx, "ns", id, []byte("v2"))
	assert.ErrorIs(t, err, perrors.ErrSealFailed)
}

// TestPurser_moveNamespaceEntropyFailure ensures MoveNamespace propagates entropy
// failures during the re-seal step.
func TestPurser_moveNamespaceEntropyFailure(t *testing.T) {
	ctx := context.Background()
	p, _ := newCryptoPurser(t)

	id, err := p.Store(ctx, "ns-a", []byte("v"))
	assert.Ok(t, err)

	withFailingEntropy(t)

	err = p.MoveNamespace(ctx, "ns-a", "ns-b", id)
	assert.ErrorIs(t, err, perrors.ErrSealFailed)
}

//=============================================================================
// Tests: concurrent access
//=============================================================================

// TestPurser_concurrent exercises Store/Retrieve/Update in parallel goroutines against
// a single shared purser. Primarily a race-detector probe (run with `go test -race`);
// success criterion is "no race, no deadlock, no panic."
func TestPurser_concurrent(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	// Pre-seed one row so Retrieve and Update have a known target.
	id, err := p.Store(ctx, "ns", []byte("seed"))
	assert.Ok(t, err)

	const goroutines = 8
	const iters = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			for j := range iters {
				_, _ = p.Store(ctx, "ns", []byte{byte(i), byte(j)})
				_, _ = p.Retrieve(ctx, "ns", id)
				_ = p.Update(ctx, "ns", id, []byte{byte(i), byte(j)})
			}
		}(i)
	}
	wg.Wait()
}

//=============================================================================
// Tests: nil receiver contract
//=============================================================================

// TestPurser_nilReceiverContract asserts every Purser method on a typed-nil purserImpl
// returns ErrNilPurser without leaking values. The typed nil is constructed via
// NewNilPurser (export_test.go) since the underlying type is unexported.
func TestPurser_nilReceiverContract(t *testing.T) {
	ctx := context.Background()
	p := purser.NewNilPurser()
	const id = "0123456789abcdef0123456789abcdef"

	gotID, err := p.Store(ctx, "ns", []byte("x"))
	assert.ErrorIs(t, err, perrors.ErrNilPurser)
	assert.Equal(t, "", gotID)

	gotPlain, err := p.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, perrors.ErrNilPurser)
	assert.Equal(t, []byte(nil), gotPlain)

	assert.ErrorIs(t, p.Update(ctx, "ns", id, []byte("z")), perrors.ErrNilPurser)
	assert.ErrorIs(t, p.CompareAndSwap(ctx, "ns", id, []byte("a"), []byte("b")), perrors.ErrNilPurser)
	assert.ErrorIs(t, p.MoveNamespace(ctx, "from", "to", id), perrors.ErrNilPurser)
	assert.ErrorIs(t, p.Delete(ctx, "ns", id), perrors.ErrNilPurser)
}

//=============================================================================
// Helpers
//=============================================================================

// newPurser builds a null-locker-backed purser through real purser.New orchestration.
// Use for any test that does not specifically depend on real envelope crypto.
func newPurser(tb testing.TB) (purser.Purser, *hold.MemHold) {
	tb.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(tb, err, "new memhold")
	return pursertest.NewTestPurser(tb, h), h
}

// newCryptoPurser builds a v1-locker-backed purser for tests whose contract depends
// on the locker consuming crypto/rand.Reader (entropy-failure simulation).
func newCryptoPurser(tb testing.TB) (purser.Purser, *hold.MemHold) {
	tb.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(tb, err, "new memhold")
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(tb, err, "new key")
	lck, err := v1.New(priv)
	assert.Ok(tb, err, "new locker")
	kr, err := memring.New(lck)
	assert.Ok(tb, err, "new keyring")
	p, err := purser.New(h, kr)
	assert.Ok(tb, err, "new purser")
	return p, h
}

// withFailingEntropy swaps crypto/rand.Reader for one that always returns io.EOF and
// restores the original on test cleanup.
func withFailingEntropy(t *testing.T) {
	t.Helper()
	orig := crand.Reader
	t.Cleanup(func() { crand.Reader = orig })
	crand.Reader = eofReader{}
}

// eofReader is a crypto/rand.Reader substitute that always returns io.EOF.
type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

// stubKeyring is a minimal Keyring whose RouteKeyID is overridable; everything else is
// a stub. Useful for asserting purser's behavior when the keyring layer fails.
type stubKeyring struct {
	active purser.Locker
	route  func([]byte) (purser.Locker, error)
}

func (s *stubKeyring) Active() purser.Locker { return s.active }

func (s *stubKeyring) Lookup([]byte) (purser.Locker, bool) { return nil, false }

func (s *stubKeyring) Register(purser.Locker) error { return perrors.ErrInvalidNewArgs }

func (s *stubKeyring) SetActive(purser.Locker) error { return perrors.ErrInvalidNewArgs }

func (s *stubKeyring) RouteKeyID(c []byte) (purser.Locker, error) { return s.route(c) }

// deleteFailingHold wraps a real hold.Hold and forces Delete on a specific namespace
// to return an error. Used by the MoveNamespace partial-failure test so the inner
// CreateWithIdentifier still succeeds and the orchestration reaches the Delete branch.
type deleteFailingHold struct {
	hold.Hold
	failOn string
}

func (h *deleteFailingHold) Delete(ctx context.Context, namespace, identifier string) error {
	if namespace == h.failOn {
		return errors.New("simulated delete failure")
	}
	return h.Hold.Delete(ctx, namespace, identifier)
}
