package purser_test

// Tests for registry dispatch, purser orchestration, and multi-version keyring routing.

import (
	"context"
	"crypto/ecdh"
	crand "crypto/rand"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/hold/holdtest"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring/kdf"
	"go.rtnl.ai/x/purser/keyring/memring"
	"go.rtnl.ai/x/purser/keyring/registry"
	"go.rtnl.ai/x/purser/locker"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/pursertest"
)

//=============================================================================
// Registry
//=============================================================================

// TestRegisteredLockerEditions documents which locker/vN packages are linked into tests.
// When adding locker/v2, add that package's constv2.Edition to required.
func TestRegisteredLockerEditions(t *testing.T) {
	required := []string{constv1.Edition}
	got := registry.Editions()
	for _, edition := range required {
		assert.True(t, slices.Contains(got, edition))
	}
}

// TestFromSeed_v1_roundtrip verifies version-dispatched seed construction.
func TestFromSeed_v1_roundtrip(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	for i := range seed {
		seed[i] = byte(i)
	}
	lck, err := registry.FromSeed(constv1.Edition, seed)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
	assert.True(t, len(lck.KeyID()) > 0)
}

// TestFromSeed_editionMethods verifies dispatch returns a locker with v1 edition metadata.
func TestFromSeed_editionMethods(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	_, err := crand.Read(seed)
	assert.Ok(t, err)
	lck, err := registry.FromSeed(constv1.Edition, seed)
	assert.Ok(t, err)
	assert.Equal(t, constv1.Version, lck.Version())
	assert.Equal(t, constv1.Edition, lck.Edition())
	assert.Equal(t, constv1.Recipe, lck.Recipe())
	assert.Equal(t, constv1.Context, lck.Context())
}

// TestFromSeed_unsupportedVersion rejects unknown edition strings.
func TestFromSeed_unsupportedVersion(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	_, err := registry.FromSeed("v0", seed)
	assert.ErrorIs(t, err, perrors.ErrUnsupportedLockerVersion)
}

// TestFromPassword_v1_roundtrip verifies version-dispatched password construction.
func TestFromPassword_v1_roundtrip(t *testing.T) {
	salt, err := kdf.RandSalt()
	assert.Ok(t, err)
	lck, err := registry.FromPassword(constv1.Edition, []byte("pw"), salt, kdf.MemoryConstrainedParams)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

// TestFromPKCS8_roundtrip loads v1 from PKCS#8 material.
func TestFromPKCS8_roundtrip(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	assert.Ok(t, err)
	lck, err := registry.FromPKCS8(der)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

// TestFromPKCS8_invalidDER rejects malformed input.
func TestFromPKCS8_invalidDER(t *testing.T) {
	_, err := registry.FromPKCS8([]byte{0x30, 0x01, 0x02})
	assert.ErrorIs(t, err, perrors.ErrInvalidWrappingKey)
}

// TestFromKey_ok accepts an X25519 private key.
func TestFromKey_ok(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lck, err := registry.FromKey(priv)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

// TestFromKey_rejectsWrongType rejects non-key input.
func TestFromKey_rejectsWrongType(t *testing.T) {
	_, err := registry.FromKey("not-a-key")
	assert.ErrorIs(t, err, perrors.ErrInvalidWrappingKey)
}

// TestParseKeyID_v1Wire matches the locker key id on self-sealed v1 ciphertext.
func TestParseKeyID_v1Wire(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	_, err := crand.Read(seed)
	assert.Ok(t, err)
	lck, err := registry.FromSeed(constv1.Edition, seed)
	assert.Ok(t, err)
	wire, err := lck.Seal("ns", []byte("data"))
	assert.Ok(t, err)
	kid, err := registry.ParseKeyID(wire)
	assert.Ok(t, err)
	assert.Equal(t, lck.KeyID(), kid)
}

// TestParseKeyID_unrecognized rejects garbage and nulllocker holdtest wire.
func TestParseKeyID_unrecognized(t *testing.T) {
	_, err := registry.ParseKeyID([]byte("too-short"))
	assert.ErrorIs(t, err, perrors.ErrUnrecognizedCiphertext)

	wire := holdtest.Ciphertext(t, "ns", []byte("plain"))
	_, err = registry.ParseKeyID(wire)
	assert.ErrorIs(t, err, perrors.ErrUnrecognizedCiphertext)
}

//=============================================================================
// New
//=============================================================================

// TestNew_nilHold verifies New rejects a nil hold.
func TestNew_nilHold(t *testing.T) {
	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	kr := memring.New()
	assert.Ok(t, kr.SetDefault(lck))

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
// Store / Retrieve
//=============================================================================

// TestPurser_storeRetrieveRoundTrip exercises Store then Retrieve with a null locker.
func TestPurser_storeRetrieveRoundTrip(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	res, err := p.Store(ctx, "ns", []byte("hello"))
	assert.Ok(t, err)
	assert.True(t, res.ID != "", "Store: expected non-empty identifier")

	got, err := p.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("hello"), got)
}

// TestPurser_storeErrNoLockerWithoutDefault ensures Store fails when the keyring has no default or bind.
func TestPurser_storeErrNoLockerWithoutDefault(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)
	p, err := purser.New(h, memring.New())
	assert.Ok(t, err)

	_, err = p.Store(ctx, "ns", []byte("plain"))
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

// TestPurser_updateErrNoLockerWithoutDefault ensures Update fails when the keyring has no default or bind.
func TestPurser_updateErrNoLockerWithoutDefault(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)
	kr := memring.New()
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	assert.Ok(t, kr.SetDefault(lck))
	res, err := p.Store(ctx, "ns", []byte("v"))
	assert.Ok(t, err)

	assert.Ok(t, kr.Revoke(lck.KeyID()))
	_, err = p.Update(ctx, "ns", res.ID, []byte("v2"))
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

// TestPurser_storeResultFields asserts Result carries namespace, key id, and edition metadata.
func TestPurser_storeResultFields(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)
	p, lck := pursertest.NewTestPurserWithLocker(t, h)

	const ns = "tenant-a"
	res, err := p.Store(ctx, ns, []byte("payload"))
	assert.Ok(t, err)
	assert.Equal(t, ns, res.Namespace)
	assert.Equal(t, lck.KeyID(), res.KeyID)
	assert.Equal(t, "null", res.Edition)
}

// TestPurser_storeResultFields_v1 asserts Result edition metadata for a v1 locker.
func TestPurser_storeResultFields_v1(t *testing.T) {
	ctx := context.Background()
	p, _ := newCryptoPurser(t)

	res, err := p.Store(ctx, "ns", []byte("secret"))
	assert.Ok(t, err)
	assert.Equal(t, "ns", res.Namespace)
	assert.True(t, len(res.KeyID) > 0)
	assert.Equal(t, constv1.Edition, res.Edition)
}

// TestPurser_retrieveMissing asserts a hex-formatted but unbound id surfaces ErrNotFound.
func TestPurser_retrieveMissing(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	_, err := p.Retrieve(ctx, "ns", "00112233445566778899aabbccddeeff")
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_retrieveMalformedIdentifier asserts invalid hex surfaces ErrInvalidIdentifier.
func TestPurser_retrieveMalformedIdentifier(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	_, err := p.Retrieve(ctx, "ns", "not-a-hex-id")
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrInvalidIdentifier)
}

//=============================================================================
// Update / CompareAndSwap / MoveNamespace / Delete
//=============================================================================

// TestPurser_update covers happy-path replacement and the missing-row failure mode.
func TestPurser_update(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	res, err := p.Store(ctx, "ns", []byte("v1"))
	assert.Ok(t, err)

	_, err = p.Update(ctx, "ns", res.ID, []byte("v2"))
	assert.Ok(t, err)
	got, err := p.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v2"), got)

	_, err = p.Update(ctx, "ns", "00112233445566778899aabbccddeeff", []byte("v3"))
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_compareAndSwap covers wrong-current, success, and missing-row branches.
func TestPurser_compareAndSwap(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	res, err := p.Store(ctx, "ns", []byte("v1"))
	assert.Ok(t, err)

	casRes, err := p.CompareAndSwap(ctx, "ns", res.ID, []byte("wrong"), []byte("v2"))
	assert.Equal(t, purser.Result{}, casRes)
	assert.ErrorIs(t, err, perrors.ErrWrongCurrent)

	got, err := p.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v1"), got)

	casRes, err = p.CompareAndSwap(ctx, "ns", res.ID, []byte("v1"), []byte("v2"))
	assert.Equal(t, "ns", casRes.Namespace)
	assert.Equal(t, res.KeyID, casRes.KeyID)
	assert.Equal(t, res.Edition, casRes.Edition)
	assert.Equal(t, res.ID, casRes.ID)
	assert.Ok(t, err)
	got, err = p.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v2"), got)

	casRes, err = p.CompareAndSwap(ctx, "ns", "aabbccddeeff00112233445566778899", []byte("a"), []byte("b"))
	assert.Equal(t, purser.Result{}, casRes)
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_moveNamespace covers happy path, same-namespace no-op, and missing-row.
func TestPurser_moveNamespace(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	res, err := p.Store(ctx, "ns-a", []byte("v"))
	assert.Ok(t, err)

	err = p.MoveNamespace(ctx, "ns-a", "ns-b", res.ID)
	assert.Ok(t, err)
	got, err := p.Retrieve(ctx, "ns-b", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v"), got)

	_, err = p.Retrieve(ctx, "ns-a", res.ID)
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)

	err = p.MoveNamespace(ctx, "ns-b", "ns-b", res.ID)
	assert.Ok(t, err)
	got, err = p.Retrieve(ctx, "ns-b", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v"), got)

	err = p.MoveNamespace(ctx, "ns-x", "ns-y", "00112233445566778899aabbccddeeff")
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_moveNamespacePartialFailure asserts ErrMoveNamespaceIncomplete when Delete fails.
func TestPurser_moveNamespacePartialFailure(t *testing.T) {
	ctx := context.Background()

	mem, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)
	failing := &deleteFailingHold{Hold: mem, failOn: "old-ns"}

	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	kr := memring.New()
	assert.Ok(t, kr.SetDefault(lck))
	p, err := purser.New(failing, kr)
	assert.Ok(t, err)

	res, err := p.Store(ctx, "old-ns", []byte("payload"))
	assert.Ok(t, err)

	err = p.MoveNamespace(ctx, "old-ns", "new-ns", res.ID)
	assert.ErrorIs(t, err, perrors.ErrMoveNamespaceIncomplete)
	assert.ErrorIs(t, err, perrors.ErrHold)

	got, err := p.Retrieve(ctx, "new-ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("payload"), got)

	gotOld, err := p.Retrieve(ctx, "old-ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("payload"), gotOld)
}

// TestPurser_delete covers idempotent delete and post-condition.
func TestPurser_delete(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	res, err := p.Store(ctx, "ns", []byte("v"))
	assert.Ok(t, err)

	assert.Ok(t, p.Delete(ctx, "ns", res.ID))

	_, err = p.Retrieve(ctx, "ns", res.ID)
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)

	assert.Ok(t, p.Delete(ctx, "ns", res.ID))
	assert.Ok(t, p.Delete(ctx, "ns", "00112233445566778899aabbccddeeff"))
}

//=============================================================================
// Cross-locker routing
//=============================================================================

// TestPurser_retrieveMissingLocker ensures rows sealed under one purser cannot open on another.
func TestPurser_retrieveMissingLocker(t *testing.T) {
	ctx := context.Background()

	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	lckA, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)
	krA := newTestKeyring(t, lckA)
	pA, err := purser.New(h, krA)
	assert.Ok(t, err)

	lckB, err := nulllocker.New(t, nulllocker.VariantB, []byte("seedB"))
	assert.Ok(t, err)
	krB := newTestKeyring(t, lckB)
	pB, err := purser.New(h, krB)
	assert.Ok(t, err)

	res, err := pA.Store(ctx, "ns", []byte("secret"))
	assert.Ok(t, err)

	_, err = pB.Retrieve(ctx, "ns", res.ID)
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

//=============================================================================
// Entropy-failure paths (v1 locker)
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

	res, err := p.Store(ctx, "ns", []byte("v"))
	assert.Ok(t, err)

	withFailingEntropy(t)

	_, err = p.Update(ctx, "ns", res.ID, []byte("v2"))
	assert.ErrorIs(t, err, perrors.ErrSealFailed)
}

// TestPurser_moveNamespaceEntropyFailure ensures MoveNamespace propagates entropy failures.
func TestPurser_moveNamespaceEntropyFailure(t *testing.T) {
	ctx := context.Background()
	p, _ := newCryptoPurser(t)

	res, err := p.Store(ctx, "ns-a", []byte("v"))
	assert.Ok(t, err)

	withFailingEntropy(t)

	err = p.MoveNamespace(ctx, "ns-a", "ns-b", res.ID)
	assert.ErrorIs(t, err, perrors.ErrSealFailed)
}

//=============================================================================
// Concurrent access
//=============================================================================

// TestPurser_concurrent exercises Store/Retrieve/Update in parallel (race-detector probe).
func TestPurser_concurrent(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	res, err := p.Store(ctx, "ns", []byte("seed"))
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
				_, _ = p.Retrieve(ctx, "ns", res.ID)
				_, _ = p.Update(ctx, "ns", res.ID, []byte{byte(i), byte(j)})
			}
		}(i)
	}
	wg.Wait()
}

//=============================================================================
// Multi-version keyring
//=============================================================================

// TestMultiVersion_sealWithActiveRetrieve seals with v1 (active) and verifies retrieval.
func TestMultiVersion_sealWithActiveRetrieve(t *testing.T) {
	ctx := context.Background()
	p, _, _, _, _ := newMultiVersionPurser(t)

	res, err := p.Store(ctx, "ns", []byte("v1-data"))
	assert.Ok(t, err)

	got, err := p.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v1-data"), got)
}

// TestMultiVersion_v0WireRoutedCorrectly injects v0 wire and verifies routing by key id.
func TestMultiVersion_v0WireRoutedCorrectly(t *testing.T) {
	ctx := context.Background()
	p, h, _, lckV0A, _ := newMultiVersionPurser(t)

	wire, err := lckV0A.Seal("ns", []byte("v0a-secret"))
	assert.Ok(t, err)

	idStr := "0123456789abcdef0123456789abcdef"
	h.BypassSemanticsSetBlobForTest(t, "ns", idStr, wire)

	got, err := p.Retrieve(ctx, "ns", idStr)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v0a-secret"), got)
}

// TestMultiVersion_switchActiveAndRetrieveOld verifies rows remain readable after SetActive.
func TestMultiVersion_switchActiveAndRetrieveOld(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lckV1, err := lockerv1.New(priv)
	assert.Ok(t, err)

	lckV0A, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)

	kr := newTestKeyring(t, lckV1, lckV0A)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	resV1, err := p.Store(ctx, "ns", []byte("sealed-v1"))
	assert.Ok(t, err)

	assert.Ok(t, kr.Bind("v0-write", lckV0A))
	resV0A, err := p.Store(ctx, "ns", []byte("sealed-v0a"))
	assert.Ok(t, err)

	gotV1, err := p.Retrieve(ctx, "ns", resV1.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("sealed-v1"), gotV1)

	gotV0A, err := p.Retrieve(ctx, "ns", resV0A.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("sealed-v0a"), gotV0A)
}

// TestMultiVersion_unparseableWireReturnsUnrecognized injects garbage wire.
func TestMultiVersion_unparseableWireReturnsUnrecognized(t *testing.T) {
	ctx := context.Background()
	p, h, _, _, _ := newMultiVersionPurser(t)

	garbage := make([]byte, 64)
	for i := range garbage {
		garbage[i] = byte(0xAB ^ i)
	}

	idStr := "1122334455667788aabbccddeeff0011"
	h.BypassSemanticsSetBlobForTest(t, "ns", idStr, garbage)

	_, err := p.Retrieve(ctx, "ns", idStr)
	assert.ErrorIs(t, err, perrors.ErrUnrecognizedCiphertext)
}

// TestMultiVersion_unregisteredLockerReturnsError seals with an unregistered locker.
func TestMultiVersion_unregisteredLockerReturnsError(t *testing.T) {
	ctx := context.Background()
	p, h, _, _, _ := newMultiVersionPurser(t)

	lckV0C, err := nulllocker.New(t, nulllocker.VariantC, []byte("seedC"))
	assert.Ok(t, err)

	wire, err := lckV0C.Seal("ns", []byte("unknown"))
	assert.Ok(t, err)

	idStr := "aabbccddeeff00112233445566778899"
	h.BypassSemanticsSetBlobForTest(t, "ns", idStr, wire)

	_, err = p.Retrieve(ctx, "ns", idStr)
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

// TestMultiVersion_updateReEncryptsWithActive switches active to v1 and re-seals on Update.
func TestMultiVersion_updateReEncryptsWithActive(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	lckV0A, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lckV1, err := lockerv1.New(priv)
	assert.Ok(t, err)

	kr := newTestKeyring(t, lckV1, lckV0A)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	res, err := p.Store(ctx, "ns", []byte("original"))
	assert.Ok(t, err)

	assert.Ok(t, kr.SetDefault(lckV1))
	upd, err := p.Update(ctx, "ns", res.ID, []byte("updated"))
	assert.Ok(t, err)
	assert.Equal(t, "ns", upd.Namespace)
	assert.Equal(t, lckV1.KeyID(), upd.KeyID)
	assert.Equal(t, constv1.Edition, upd.Edition)

	got, err := p.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("updated"), got)

	wire, err := h.Get(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, "PURS", string(wire[:4]))
}

// TestMultiVersion_moveNamespaceAcrossLockerVersions re-seals with active v1 on namespace move.
func TestMultiVersion_moveNamespaceAcrossLockerVersions(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	lckV0A, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lckV1, err := lockerv1.New(priv)
	assert.Ok(t, err)

	kr := newTestKeyring(t, lckV1, lckV0A)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	res, err := p.Store(ctx, "old-ns", []byte("movedata"))
	assert.Ok(t, err)

	assert.Ok(t, kr.SetDefault(lckV1))
	err = p.MoveNamespace(ctx, "old-ns", "new-ns", res.ID)
	assert.Ok(t, err)

	got, err := p.Retrieve(ctx, "new-ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, []byte("movedata"), got)
}

//=============================================================================
// Helpers
//=============================================================================

// newTestKeyring builds a memring with defaultLck and optional indexed bind namespaces.
func newTestKeyring(tb testing.TB, defaultLck locker.Locker, index ...locker.Locker) *memring.Memring {
	tb.Helper()
	kr := memring.New()
	assert.Ok(tb, kr.SetDefault(defaultLck))
	for i, lck := range index {
		ns := fmt.Sprintf("index-%d", i)
		assert.Ok(tb, kr.Bind(ns, lck))
	}
	return kr
}

// newPurser returns a pursertest purser and its memhold.
func newPurser(tb testing.TB) (*purser.Purser, *hold.MemHold) {
	tb.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(tb, err, "new memhold")
	return pursertest.NewTestPurser(tb, h), h
}

// newCryptoPurser returns a purser wired with a real v1 locker.
func newCryptoPurser(tb testing.TB) (*purser.Purser, *hold.MemHold) {
	tb.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(tb, err, "new memhold")
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(tb, err, "new key")
	lck, err := lockerv1.New(priv)
	assert.Ok(tb, err, "new locker")
	kr := newTestKeyring(tb, lck)
	p, err := purser.New(h, kr)
	assert.Ok(tb, err, "new purser")
	return p, h
}

// newMultiVersionPurser returns a purser with v1 default plus two null locker variants indexed.
func newMultiVersionPurser(t *testing.T) (*purser.Purser, *hold.MemHold, locker.Locker, locker.Locker, locker.Locker) {
	t.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lckV1, err := lockerv1.New(priv)
	assert.Ok(t, err)

	lckV0A, err := nulllocker.New(t, nulllocker.VariantA, []byte("seedA"))
	assert.Ok(t, err)
	lckV0B, err := nulllocker.New(t, nulllocker.VariantB, []byte("seedB"))
	assert.Ok(t, err)

	kr := newTestKeyring(t, lckV1, lckV0A, lckV0B)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	return p, h, lckV1, lckV0A, lckV0B
}

// withFailingEntropy replaces crand.Reader with eofReader for the remainder of the test.
func withFailingEntropy(t *testing.T) {
	t.Helper()
	orig := crand.Reader
	t.Cleanup(func() { crand.Reader = orig })
	crand.Reader = eofReader{}
}

// eofReader is a test double that always returns io.EOF from Read.
type eofReader struct{}

// Read implements io.Reader for eofReader.
func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

// deleteFailingHold simulates hold delete failures for a configured namespace.
type deleteFailingHold struct {
	hold.Hold
	failOn string
}

// Delete fails when namespace matches failOn; otherwise delegates to the wrapped hold.
func (h *deleteFailingHold) Delete(ctx context.Context, namespace, identifier string) error {
	if namespace == h.failOn {
		return errors.New("simulated delete failure")
	}
	return h.Hold.Delete(ctx, namespace, identifier)
}
