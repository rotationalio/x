/*
Package lockertest provides conformance helpers for locker.Locker implementations.

LockerConforms encodes the version-neutral invariants every Locker must satisfy
(KeyID non-empty + defensive copy, Seal/Open round-trip including empty plaintext
and empty namespace, Open rejects wrong namespace and corrupt wire, ParseKeyID
matches KeyID for self-sealed wire and rejects empty input). New locker
implementations should call LockerConforms in their package tests to self-certify.

Implementation-specific properties (for example "Seal produces unique ciphertexts
across repeated calls" — true for the v1 randomized envelope but not for the
deterministic null locker) are NOT enforced here; those belong in each locker's own
test file.
*/
package lockertest

import (
	"bytes"
	"errors"
	"fmt"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker"
)

// NewFunc constructs a fresh locker for each call. Each conformance check builds
// a new locker so checks remain independent of one another.
type NewFunc func() (locker.Locker, error)

//=============================================================================
// Public conformance suite
//=============================================================================

// LockerConforms runs every conformance check against newLocker and returns the first
// failing check's error (or nil on full conformance). Callers typically wrap with:
//
//	if err := lockertest.LockerConforms(myFactory); err != nil {
//	    t.Fatal(err) // or use an assert helperlibrary
//	}
func LockerConforms(newLocker NewFunc) error {
	checks := []struct {
		name string
		fn   func(NewFunc) error
	}{
		{"KeyIDNonEmpty", checkKeyIDNonEmpty},
		{"KeyIDDefensiveCopy", checkKeyIDDefensiveCopy},
		{"SealOpenRoundTrip", checkSealOpenRoundTrip},
		{"SealOpenEmptyPlaintext", checkSealOpenEmptyPlaintext},
		{"SealOpenEmptyNamespace", checkSealOpenEmptyNamespace},
		{"OpenWrongNamespace", checkOpenWrongNamespace},
		{"OpenEmptyWire", checkOpenEmptyWire},
		{"OpenZeroedWire", checkOpenZeroedWire},
		{"ParseKeyIDMatchesKeyID", checkParseKeyIDMatchesKeyID},
		{"ParseKeyIDRejectsEmpty", checkParseKeyIDRejectsEmpty},
	}
	for _, c := range checks {
		if err := c.fn(newLocker); err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
	}
	return nil
}

//=============================================================================
// Checks (one invariant per function so failure messages pinpoint the contract)
//=============================================================================

// checkKeyIDNonEmpty verifies KeyID returns at least one byte. The keyring uses the
// returned bytes as a map key; an empty key would collide with every other empty key.
func checkKeyIDNonEmpty(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	if len(lck.KeyID()) == 0 {
		return errors.New("KeyID returned empty slice")
	}
	return nil
}

// checkKeyIDDefensiveCopy verifies mutating the returned KeyID does not affect a
// subsequent call. Without a defensive copy, a caller could mutate the keyring's
// internal state simply by holding the slice.
func checkKeyIDDefensiveCopy(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	a := lck.KeyID()
	if len(a) == 0 {
		return errors.New("KeyID returned empty slice")
	}
	a[0] ^= 0xff
	b := lck.KeyID()
	if a[0] == b[0] {
		return errors.New("KeyID mutation propagated to subsequent call (not a defensive copy)")
	}
	return nil
}

// checkSealOpenRoundTrip verifies non-empty plaintext survives Seal/Open under the
// same namespace.
func checkSealOpenRoundTrip(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	plain := []byte("conformance-roundtrip")
	wire, err := lck.Seal("ns", plain)
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}
	got, err := lck.Open("ns", wire)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	if !bytes.Equal(plain, got) {
		return fmt.Errorf("open: got %q want %q", got, plain)
	}
	return nil
}

// checkSealOpenEmptyPlaintext verifies an empty plaintext round-trips losslessly.
// Empty rows are valid (e.g., presence-only secrets) and must not crash the locker.
func checkSealOpenEmptyPlaintext(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	wire, err := lck.Seal("ns", []byte{})
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}
	got, err := lck.Open("ns", wire)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	if len(got) != 0 {
		return fmt.Errorf("open: expected empty plaintext, got %q", got)
	}
	return nil
}

// checkSealOpenEmptyNamespace verifies the empty string is a valid namespace and
// survives a Seal/Open round-trip. The namespace is a label, not a secret, and may
// legitimately be empty (e.g., single-tenant deployments).
func checkSealOpenEmptyNamespace(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	plain := []byte("ok")
	wire, err := lck.Seal("", plain)
	if err != nil {
		return fmt.Errorf("seal(empty namespace): %w", err)
	}
	got, err := lck.Open("", wire)
	if err != nil {
		return fmt.Errorf("open(empty namespace): %w", err)
	}
	if !bytes.Equal(plain, got) {
		return fmt.Errorf("open: got %q want %q", got, plain)
	}
	return nil
}

// checkOpenWrongNamespace verifies Open under a different namespace than Seal returns
// ErrNamespaceMismatch. This is the namespace-binding contract every locker enforces.
func checkOpenWrongNamespace(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	wire, err := lck.Seal("ns-a", []byte("data"))
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}
	_, err = lck.Open("ns-b", wire)
	if !errors.Is(err, perrors.ErrNamespaceMismatch) {
		return fmt.Errorf("open(wrong ns): got %v want %v", err, perrors.ErrNamespaceMismatch)
	}
	return nil
}

// checkOpenEmptyWire verifies Open returns a non-nil error for nil and zero-length
// input. The specific error class is locker-specific; we only assert "some error."
func checkOpenEmptyWire(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	if _, err = lck.Open("ns", nil); err == nil {
		return errors.New("open(nil): want some error, got nil")
	}
	if _, err = lck.Open("ns", []byte{}); err == nil {
		return errors.New("open(empty): want some error, got nil")
	}
	return nil
}

// checkOpenZeroedWire verifies Open rejects an all-zeros wire (no valid magic prefix
// in any locker version we ship).
func checkOpenZeroedWire(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	if _, err = lck.Open("ns", make([]byte, 64)); err == nil {
		return errors.New("open(zeroed): want some error, got nil")
	}
	return nil
}

// checkParseKeyIDMatchesKeyID verifies ParseKeyID extracts the same bytes that KeyID
// reports for a wire blob the locker just produced. The keyring relies on this to
// route ciphertext to the locker that sealed it.
func checkParseKeyIDMatchesKeyID(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	wire, err := lck.Seal("ns", []byte("x"))
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}
	parsed, err := lck.ParseKeyID(wire)
	if err != nil {
		return fmt.Errorf("ParseKeyID: %w", err)
	}
	if !bytes.Equal(parsed, lck.KeyID()) {
		return fmt.Errorf("ParseKeyID: got %x want %x", parsed, lck.KeyID())
	}
	return nil
}

// checkParseKeyIDRejectsEmpty verifies ParseKeyID returns an error for empty input.
// Without this, a keyring iterating ParseKeyID over registered lockers could be
// tricked into matching against garbage.
func checkParseKeyIDRejectsEmpty(newLocker NewFunc) error {
	lck, err := newLocker()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	if _, err := lck.ParseKeyID(nil); err == nil {
		return errors.New("ParseKeyID(nil): want some error, got nil")
	}
	if _, err := lck.ParseKeyID([]byte{}); err == nil {
		return errors.New("ParseKeyID(empty): want some error, got nil")
	}
	return nil
}
