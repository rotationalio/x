package nulllocker_test

// Tests for the null-locker. The null locker has no real crypto so most observable
// behavior comes from the conformance suite; this file adds tests for nulllocker-
// specific contracts (variant key sizes, checksum tamper detection, seed validation).

import (
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/locker"
	"go.rtnl.ai/x/purser/locker/lockertest"
)

//=============================================================================
// Tests: conformance
//=============================================================================

// TestLocker_conforms_allVariants runs the shared conformance suite against every
// variant.
func TestLocker_conforms_allVariants(t *testing.T) {
	variants := []struct {
		name string
		v    nulllocker.Variant
	}{
		{"VariantA", nulllocker.VariantA},
		{"VariantB", nulllocker.VariantB},
		{"VariantC", nulllocker.VariantC},
	}
	for _, tc := range variants {
		t.Run(tc.name, func(t *testing.T) {
			err := lockertest.LockerConforms(func() (locker.Locker, error) {
				return nulllocker.New(t, tc.v, []byte("conformance-seed"))
			})
			assert.Ok(t, err)
		})
	}
}

//=============================================================================
// Tests: nulllocker-specific behavior
//=============================================================================

// TestVariant_keyIDSize asserts each variant produces a distinct KeyID length so the
// keyring's per-locker ParseKeyID fallback can disambiguate variants when their wire
// magic is identical.
func TestVariant_keyIDSize(t *testing.T) {
	cases := []struct {
		name    string
		variant nulllocker.Variant
		wantLen int
	}{
		{"VariantA", nulllocker.VariantA, 4},
		{"VariantB", nulllocker.VariantB, 8},
		{"VariantC", nulllocker.VariantC, 16},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lck, err := nulllocker.New(t, tc.variant, []byte("seed"))
			assert.Ok(t, err)
			assert.Equal(t, tc.wantLen, len(lck.KeyID()))
		})
	}
}

// TestNew_nilSeed rejects a nil seed.
func TestNew_nilSeed(t *testing.T) {
	_, err := nulllocker.New(t, nulllocker.VariantA, nil)
	assert.ErrorIs(t, err, perrors.ErrInvalidSeed)
}

// TestNew_nilTB rejects a nil testing.TB (production-use guard).
func TestNew_nilTB(t *testing.T) {
	_, err := nulllocker.New(nil, nulllocker.VariantA, []byte("seed"))
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestSeal_tamperedChecksum verifies Open rejects a wire whose trailing 32-byte
// checksum has been mutated (covers the deferred-tamper detection path that golden
// fixture testing relies on).
func TestSeal_tamperedChecksum(t *testing.T) {
	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	wire, err := lck.Seal("ns", []byte("hello"))
	assert.Ok(t, err)

	// Flip the last byte (inside the trailing SHA-256 region).
	wire[len(wire)-1] ^= 0xff
	_, err = lck.Open("ns", wire)
	assert.ErrorIs(t, err, perrors.ErrDecrypt)
}

// TestSeal_tamperedPlaintext verifies a flip in the plaintext region is also caught
// by the trailing checksum (the wire's framing parses fine; only the SHA-256 fails).
func TestSeal_tamperedPlaintext(t *testing.T) {
	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	wire, err := lck.Seal("ns", []byte("hello"))
	assert.Ok(t, err)

	// Flip a byte that falls inside the plaintext region (well before the trailing
	// 32-byte checksum). Computed SHA-256 will no longer match the stored one.
	wire[len(wire)-33] ^= 0xff
	_, err = lck.Open("ns", wire)
	assert.ErrorIs(t, err, perrors.ErrDecrypt)
}

// TestNilReceiver covers the nil-receiver guards on Seal/Open/KeyID.
func TestNilReceiver(t *testing.T) {
	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed"))
	assert.Ok(t, err)
	_ = lck

	// We cannot call methods on a nil *nullLocker directly because the type is
	// unexported. Instead, exercise the nil-receiver guards via the public Locker
	// interface using a typed nil; nulllocker exposes a zero-cost test bridge for
	// this in nulllocker_test.go (see NewNilLocker below).
	var nl locker.Locker = nulllocker.NewNilLocker()
	_, err = nl.Seal("ns", []byte("x"))
	assert.ErrorIs(t, err, perrors.ErrSealFailed)
	_, err = nl.Open("ns", []byte("x"))
	assert.ErrorIs(t, err, perrors.ErrDecrypt)
	assert.Equal(t, []byte(nil), nl.KeyID())
}
