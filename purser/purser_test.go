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
	"strconv"
	"sync"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/hold/holdtest"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring"
	"go.rtnl.ai/x/purser/keyring/memring"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/pursertest"
	"go.rtnl.ai/x/purser/registry"
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

// TestRegisterLockerVersion_concurrent exercises parallel registration and reads.
func TestRegisterLockerVersion_concurrent(t *testing.T) {
	const goroutines = 32
	const perGoroutine = 8

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for i := range perGoroutine {
				edition := fmt.Sprintf("purser-concurrency-%d-%d", id, i)
				registry.Register(edition, registry.Hooks{})
			}
		}(g)

		go func() {
			defer wg.Done()
			for range perGoroutine {
				_ = registry.Editions()
				seed := make([]byte, lockerv1.SeedBytes)
				_, _ = registry.FromSeed(constv1.Edition, seed)
			}
		}()
	}

	wg.Wait()

	got := registry.Editions()
	assert.True(t, len(got) >= 1+goroutines*perGoroutine)

	seen := make(map[string]struct{}, len(got))
	for _, edition := range got {
		if _, dup := seen[edition]; dup {
			t.Fatalf("duplicate edition in registry.Editions: %q", edition)
		}
		seen[edition] = struct{}{}
	}
	_, ok := seen[constv1.Edition]
	assert.True(t, ok)

	for g := range goroutines {
		for i := range perGoroutine {
			edition := fmt.Sprintf("purser-concurrency-%d-%d", g, i)
			if _, ok := seen[edition]; !ok {
				t.Fatalf("missing registered edition %q", edition)
			}
		}
	}
}

// TestRegisterLockerVersion_overwriteSameEdition verifies re-registering one edition is safe.
func TestRegisterLockerVersion_overwriteSameEdition(t *testing.T) {
	edition := "purser-concurrency-overwrite"
	registry.Register(edition, registry.Hooks{})
	registry.Register(edition, registry.Hooks{
		FromSeed: func([]byte) (contract.Locker, error) {
			return nil, strconv.ErrSyntax
		},
	})

	seed := make([]byte, lockerv1.SeedBytes)
	_, err := registry.FromSeed(edition, seed)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
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
	assert.Equal(t, int(constv1.Version), lck.Version())
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
	salt, err := keyring.RandSalt()
	assert.Ok(t, err)
	lck, err := registry.FromPassword(constv1.Edition, []byte("pw"), salt, keyring.MemoryConstrainedParams())
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
// Store / Retrieve
//=============================================================================

// TestPurser_storeRetrieveRoundTrip exercises Store then Retrieve with a null locker.
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

	id, err := p.Store(ctx, "ns", []byte("v1"))
	assert.Ok(t, err)

	assert.Ok(t, p.Update(ctx, "ns", id, []byte("v2")))
	got, err := p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v2"), got)

	err = p.Update(ctx, "ns", "00112233445566778899aabbccddeeff", []byte("v3"))
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestPurser_compareAndSwap covers wrong-current, success, and missing-row branches.
func TestPurser_compareAndSwap(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	id, err := p.Store(ctx, "ns", []byte("v1"))
	assert.Ok(t, err)

	err = p.CompareAndSwap(ctx, "ns", id, []byte("wrong"), []byte("v2"))
	assert.ErrorIs(t, err, perrors.ErrWrongCurrent)

	got, err := p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v1"), got)

	assert.Ok(t, p.CompareAndSwap(ctx, "ns", id, []byte("v1"), []byte("v2")))
	got, err = p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v2"), got)

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

	assert.Ok(t, p.MoveNamespace(ctx, "ns-a", "ns-b", id))
	got, err := p.Retrieve(ctx, "ns-b", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("v"), got)

	_, err = p.Retrieve(ctx, "ns-a", id)
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)

	assert.Ok(t, p.MoveNamespace(ctx, "ns-b", "ns-b", id))
	got, err = p.Retrieve(ctx, "ns-b", id)
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
	kr, err := memring.New(lck)
	assert.Ok(t, err)
	p, err := purser.New(failing, kr)
	assert.Ok(t, err)

	id, err := p.Store(ctx, "old-ns", []byte("payload"))
	assert.Ok(t, err)

	err = p.MoveNamespace(ctx, "old-ns", "new-ns", id)
	assert.ErrorIs(t, err, perrors.ErrMoveNamespaceIncomplete)
	assert.ErrorIs(t, err, perrors.ErrHold)

	got, err := p.Retrieve(ctx, "new-ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("payload"), got)

	gotOld, err := p.Retrieve(ctx, "old-ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("payload"), gotOld)
}

// TestPurser_delete covers idempotent delete and post-condition.
func TestPurser_delete(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

	id, err := p.Store(ctx, "ns", []byte("v"))
	assert.Ok(t, err)

	assert.Ok(t, p.Delete(ctx, "ns", id))

	_, err = p.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, perrors.ErrHold)
	assert.ErrorIs(t, err, perrors.ErrNotFound)

	assert.Ok(t, p.Delete(ctx, "ns", id))
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

	id, err := pA.Store(ctx, "ns", []byte("secret"))
	assert.Ok(t, err)

	_, err = pB.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
}

// TestPurser_keyringRouteFailurePropagates confirms RouteKeyID errors propagate verbatim.
func TestPurser_keyringRouteFailurePropagates(t *testing.T) {
	ctx := context.Background()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)

	real, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	kr := &stubKeyring{
		active: real,
		route: func([]byte) (contract.Locker, error) {
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

	id, err := p.Store(ctx, "ns", []byte("v"))
	assert.Ok(t, err)

	withFailingEntropy(t)

	err = p.Update(ctx, "ns", id, []byte("v2"))
	assert.ErrorIs(t, err, perrors.ErrSealFailed)
}

// TestPurser_moveNamespaceEntropyFailure ensures MoveNamespace propagates entropy failures.
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
// Concurrent access
//=============================================================================

// TestPurser_concurrent exercises Store/Retrieve/Update in parallel (race-detector probe).
func TestPurser_concurrent(t *testing.T) {
	ctx := context.Background()
	p, _ := newPurser(t)

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
// Nil receiver
//=============================================================================

// TestPurser_nilReceiverContract asserts every method on a typed-nil purserImpl returns ErrNilPurser.
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
// Multi-version keyring
//=============================================================================

// TestMultiVersion_sealWithActiveRetrieve seals with v1 (active) and verifies retrieval.
func TestMultiVersion_sealWithActiveRetrieve(t *testing.T) {
	ctx := context.Background()
	p, _, _, _, _ := newMultiVersionPurser(t)

	id, err := p.Store(ctx, "ns", []byte("v1-data"))
	assert.Ok(t, err)

	got, err := p.Retrieve(ctx, "ns", id)
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

	kr, err := memring.New(lckV1, lckV0A)
	assert.Ok(t, err)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	idV1, err := p.Store(ctx, "ns", []byte("sealed-v1"))
	assert.Ok(t, err)

	assert.Ok(t, kr.SetActive(lckV0A))
	idV0A, err := p.Store(ctx, "ns", []byte("sealed-v0a"))
	assert.Ok(t, err)

	gotV1, err := p.Retrieve(ctx, "ns", idV1)
	assert.Ok(t, err)
	assert.Equal(t, []byte("sealed-v1"), gotV1)

	gotV0A, err := p.Retrieve(ctx, "ns", idV0A)
	assert.Ok(t, err)
	assert.Equal(t, []byte("sealed-v0a"), gotV0A)
}

// TestMultiVersion_unparseableWireReturnsNoLocker injects garbage wire and expects ErrNoLocker.
func TestMultiVersion_unparseableWireReturnsNoLocker(t *testing.T) {
	ctx := context.Background()
	p, h, _, _, _ := newMultiVersionPurser(t)

	garbage := make([]byte, 64)
	for i := range garbage {
		garbage[i] = byte(0xAB ^ i)
	}

	idStr := "1122334455667788aabbccddeeff0011"
	h.BypassSemanticsSetBlobForTest(t, "ns", idStr, garbage)

	_, err := p.Retrieve(ctx, "ns", idStr)
	assert.ErrorIs(t, err, perrors.ErrNoLocker)
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

	kr, err := memring.New(lckV0A, lckV1)
	assert.Ok(t, err)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	id, err := p.Store(ctx, "ns", []byte("original"))
	assert.Ok(t, err)

	assert.Ok(t, kr.SetActive(lckV1))
	assert.Ok(t, p.Update(ctx, "ns", id, []byte("updated")))

	got, err := p.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("updated"), got)

	wire, err := h.Get(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, "ARR1", string(wire[:4]))
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

	kr, err := memring.New(lckV0A, lckV1)
	assert.Ok(t, err)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	id, err := p.Store(ctx, "old-ns", []byte("movedata"))
	assert.Ok(t, err)

	assert.Ok(t, kr.SetActive(lckV1))
	assert.Ok(t, p.MoveNamespace(ctx, "old-ns", "new-ns", id))

	got, err := p.Retrieve(ctx, "new-ns", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("movedata"), got)
}

//=============================================================================
// Helpers
//=============================================================================

// newPurser builds a null-locker-backed purser through pursertest.
func newPurser(tb testing.TB) (contract.Purser, *hold.MemHold) {
	tb.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(tb, err, "new memhold")
	return pursertest.NewTestPurser(tb, h), h
}

// newCryptoPurser builds a v1-locker-backed purser for entropy-failure tests.
func newCryptoPurser(tb testing.TB) (contract.Purser, *hold.MemHold) {
	tb.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(tb, err, "new memhold")
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(tb, err, "new key")
	lck, err := lockerv1.New(priv)
	assert.Ok(tb, err, "new locker")
	kr, err := memring.New(lck)
	assert.Ok(tb, err, "new keyring")
	p, err := purser.New(h, kr)
	assert.Ok(tb, err, "new purser")
	return p, h
}

// newMultiVersionPurser builds a purser with v1 (active), v0-A, and v0-B registered.
func newMultiVersionPurser(t *testing.T) (contract.Purser, *hold.MemHold, contract.Locker, contract.Locker, contract.Locker) {
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

	kr, err := memring.New(lckV1, lckV0A, lckV0B)
	assert.Ok(t, err)
	p, err := purser.New(h, kr)
	assert.Ok(t, err)

	return p, h, lckV1, lckV0A, lckV0B
}

// withFailingEntropy swaps crypto/rand.Reader for one that always returns io.EOF.
func withFailingEntropy(t *testing.T) {
	t.Helper()
	orig := crand.Reader
	t.Cleanup(func() { crand.Reader = orig })
	crand.Reader = eofReader{}
}

// eofReader is a crypto/rand.Reader substitute that always returns io.EOF.
type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

// stubKeyring is a minimal Keyring with overridable RouteKeyID.
type stubKeyring struct {
	active contract.Locker
	route  func([]byte) (contract.Locker, error)
}

func (s *stubKeyring) Active() contract.Locker { return s.active }

func (s *stubKeyring) Lookup([]byte) (contract.Locker, bool) { return nil, false }

func (s *stubKeyring) Register(contract.Locker) error { return perrors.ErrInvalidNewArgs }

func (s *stubKeyring) SetActive(contract.Locker) error { return perrors.ErrInvalidNewArgs }

func (s *stubKeyring) RouteKeyID(c []byte) (contract.Locker, error) { return s.route(c) }

// deleteFailingHold forces Delete on a specific namespace to fail.
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
