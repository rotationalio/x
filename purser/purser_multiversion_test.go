package purser_test

// Multi-version keyring dispatch tests.
//
// These tests register multiple locker implementations (v0 null variants and v1 envelope)
// in a single keyring and verify that Purser correctly routes ciphertext to the right locker
// on Open, even after switching the active locker.

import (
	"context"
	"crypto/ecdh"
	crand "crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring/memring"
	"go.rtnl.ai/x/purser/locker/v1"
)

// TestMultiVersion_sealWithActiveRetrieve seals with v1 (active) and verifies retrieval
// works through the multi-version keyring.
func TestMultiVersion_sealWithActiveRetrieve(t *testing.T) {
	ctx := context.Background()
	p, _, _, _, _ := newMultiVersionPurser(t)

	id, err := p.Store(ctx, "ns", []byte("v1-data"))
	assert.Ok(t, err)

	got, err := p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v1-data"), got)
}

// TestMultiVersion_v0WireRoutedCorrectly injects a v0 null-locker sealed blob into the hold
// and verifies that the purser routes to the correct locker via key ID.
func TestMultiVersion_v0WireRoutedCorrectly(t *testing.T) {
	ctx := context.Background()
	p, h, _, lckV0A, _ := newMultiVersionPurser(t)

	// Seal directly with v0A and inject into the hold.
	wire, err := lckV0A.Seal("ns", []byte("v0a-secret"))
	assert.Ok(t, err)

	idStr := "0123456789abcdef0123456789abcdef"
	h.BypassSemanticsSetBlobForTest(t, "ns", idStr, wire)

	got, err := p.Retrieve(ctx, "ns", idStr)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v0a-secret"), got)
}

// TestMultiVersion_switchActiveAndRetrieveOld seals rows with v1, switches active to v0A,
// seals more rows, then verifies all rows are still retrievable via the multi-version keyring.
func TestMultiVersion_switchActiveAndRetrieveOld(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lckV1, err := locker.New(priv)
	assert.Ok(t, err)

	lckV0A, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)

	kr, err := memring.New(lckV1, lckV0A)
	assert.Ok(t, err)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	// Seal with v1 (active).
	idV1, err := p.Store(ctx, "ns", []byte("sealed-v1"))
	assert.Ok(t, err)

	// Switch active to v0A and seal.
	assert.Ok(t, kr.SetActive(lckV0A))
	idV0A, err := p.Store(ctx, "ns", []byte("sealed-v0a"))
	assert.Ok(t, err)

	// Retrieve both: keyring should route each to the correct locker.
	gotV1, err := p.Retrieve(ctx, "ns", idV1)
	assert.Ok(t, err)
	assert.Equal(t, []byte("sealed-v1"), gotV1)

	gotV0A, err := p.Retrieve(ctx, "ns", idV0A)
	assert.Ok(t, err)
	assert.Equal(t, []byte("sealed-v0a"), gotV0A)
}

// TestMultiVersion_unparseableWireReturnsNoLocker injects bytes that no registered
// locker's ParseKeyID can decode (random non-magic prefix) and asserts the keyring
// surfaces ErrNoLocker rather than panicking or returning a parser error.
func TestMultiVersion_unparseableWireReturnsNoLocker(t *testing.T) {
	ctx := context.Background()
	p, h, _, _, _ := newMultiVersionPurser(t)

	// 64 bytes of non-magic garbage — too short for v1 ARR1 framing and the
	// wrong magic for any null-locker variant, so every registered locker's
	// ParseKeyID will reject this in turn.
	garbage := make([]byte, 64)
	for i := range garbage {
		garbage[i] = byte(0xAB ^ i)
	}

	idStr := "1122334455667788aabbccddeeff0011"
	h.BypassSemanticsSetBlobForTest(t, "ns", idStr, garbage)

	_, err := p.Retrieve(ctx, "ns", idStr)
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

// TestMultiVersion_unregisteredLockerReturnsError seals with a locker not registered in the
// keyring and verifies retrieval returns ErrNoLocker.
func TestMultiVersion_unregisteredLockerReturnsError(t *testing.T) {
	ctx := context.Background()
	p, h, _, _, _ := newMultiVersionPurser(t)

	// Create a locker not registered in the keyring.
	lckV0C, err := nulllocker.New(t, nulllocker.VariantC, []byte("seedC"))
	assert.Ok(t, err)

	wire, err := lckV0C.Seal("ns", []byte("unknown"))
	assert.Ok(t, err)

	idStr := "aabbccddeeff00112233445566778899"
	h.BypassSemanticsSetBlobForTest(t, "ns", idStr, wire)

	_, err = p.Retrieve(ctx, "ns", idStr)
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

// TestMultiVersion_updateReEncryptsWithActive seals with v0A, switches active to v1,
// updates the row, and verifies the row is now sealed under v1.
func TestMultiVersion_updateReEncryptsWithActive(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	lckV0A, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lckV1, err := locker.New(priv)
	assert.Ok(t, err)

	// Start with v0A as active.
	kr, err := memring.New(lckV0A, lckV1)
	assert.Ok(t, err)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	id, err := p.Store(ctx, "ns", []byte("original"))
	assert.Ok(t, err)

	// Switch to v1 and update.
	assert.Ok(t, kr.SetActive(lckV1))
	assert.Ok(t, p.Update(ctx, "ns", id, []byte("updated")))

	// Verify the row now decrypts correctly with the new active.
	got, err := p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("updated"), got)

	// Verify the wire is now v1 format (starts with "ARR1" magic, not "NULL").
	wire, err := h.Get(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, "ARR1", string(wire[:4]))
}

// TestMultiVersion_moveNamespaceAcrossLockerVersions seals with v0A under one namespace,
// moves to another namespace (re-encrypts with active v1), and verifies the data round-trips.
func TestMultiVersion_moveNamespaceAcrossLockerVersions(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	lckV0A, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lckV1, err := locker.New(priv)
	assert.Ok(t, err)

	// Start with v0A active.
	kr, err := memring.New(lckV0A, lckV1)
	assert.Ok(t, err)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	id, err := p.Store(ctx, "old-ns", []byte("movedata"))
	assert.Ok(t, err)

	// Switch to v1 and move namespace — this re-seals with v1.
	assert.Ok(t, kr.SetActive(lckV1))
	assert.Ok(t, p.MoveNamespace(ctx, "old-ns", "new-ns", id))

	got, err := p.Retrieve(ctx, "new-ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("movedata"), got)
}

//=============================================================================
// Helpers
//=============================================================================

// newMultiVersionPurser builds a purser with three registered lockers: a v1 locker
// (active), a v0 variant-A null locker, and a v0 variant-B null locker.
func newMultiVersionPurser(t *testing.T) (purser.Purser, *hold.MemHold, purser.Locker, purser.Locker, purser.Locker) {
	t.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lckV1, err := locker.New(priv)
	assert.Ok(t, err)

	lckV0A, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)
	lckV0B, err := nulllocker.New(t, nulllocker.VariantB, []byte("seedB"))
	assert.Ok(t, err)

	kr, err := memring.New(lckV1, lckV0A, lckV0B)
	assert.Ok(t, err)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	return p, h, lckV1, lckV0A, lckV0B
}
