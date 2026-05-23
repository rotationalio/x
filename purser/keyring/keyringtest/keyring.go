// Package keyringtest provides conformance helpers for keyring implementations.
package keyringtest

// Keyring conformance helpers for tests.

import (
	"errors"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/internal/nulllocker"
)

// routeTestNS is the namespace used when sealing wire for RouteKeyID conformance checks.
const routeTestNS = "keyringtest"

// NewFunc constructs a keyring from an active locker and optional others.
type NewFunc func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error)

// checkFunc runs one keyring conformance invariant.
type checkFunc func(*testing.T, NewFunc) error

//=============================================================================
// Public conformance suite
//=============================================================================

// KeyringConforms runs every documented contract against the keyring implementation
// produced by newKeyring. Each check builds a fresh keyring via newKeyring so checks
// remain independent.
func KeyringConforms(t *testing.T, newKeyring NewFunc) {
	t.Helper()

	checks := []struct {
		name string
		fn   checkFunc
	}{
		{"new_rejects_nil_active", checkNewRejectsNilActive},
		{"active_returns_write_locker", checkActiveReturnsWriteLocker},
		{"lookup_registered_and_unknown", checkLookupRegisteredAndUnknown},
		{"register_rejects_nil", checkRegisterRejectsNil},
		{"register_rejects_duplicate_key_id", checkRegisterRejectsDuplicateKeyID},
		{"set_active_rejects_nil", checkSetActiveRejectsNil},
		{"set_active_registers_and_switches", checkSetActiveRegistersAndSwitches},
		{"route_key_id_cross_locker_parse_fallback", checkRouteKeyIDCrossLockerParseFallback},
		{"route_key_id_no_match_returns_err_no_locker", checkRouteKeyIDNoMatchReturnsErrNoLocker},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			assert.Ok(t, c.fn(t, newKeyring))
		})
	}
}

//=============================================================================
// Null locker fixtures
//=============================================================================

// nullLocker builds a nulllocker with a deterministic key ID from seed bytes.
func nullLocker(t *testing.T, variant nulllocker.Variant, seed []byte) (contract.Locker, error) {
	t.Helper()
	return nulllocker.New(t, variant, seed)
}

//=============================================================================
// Checks (one invariant per function so failure messages pinpoint the contract)
//=============================================================================

// checkNewRejectsNilActive verifies New rejects a nil active locker.
func checkNewRejectsNilActive(t *testing.T, newKeyring NewFunc) error {
	_, err := newKeyring(nil)
	if !errors.Is(err, perrors.ErrInvalidNewArgs) {
		return fmt.Errorf("new(nil): got %v want %v", err, perrors.ErrInvalidNewArgs)
	}
	return nil
}

// checkActiveReturnsWriteLocker verifies Active returns the locker passed to New.
func checkActiveReturnsWriteLocker(t *testing.T, newKeyring NewFunc) error {
	active, err := nullLocker(t, nulllocker.VariantA, []byte("active"))
	if err != nil {
		return fmt.Errorf("null locker(active): %w", err)
	}

	kr, err := newKeyring(active)
	if err != nil {
		return fmt.Errorf("new(active): %w", err)
	}
	if kr.Active() == nil {
		return fmt.Errorf("active: got nil locker")
	}
	if string(kr.Active().KeyID()) != string(active.KeyID()) {
		return fmt.Errorf("active key id mismatch: got %q want %q", kr.Active().KeyID(), active.KeyID())
	}
	return nil
}

// checkLookupRegisteredAndUnknown verifies Lookup returns registered lockers and misses unknown IDs.
func checkLookupRegisteredAndUnknown(t *testing.T, newKeyring NewFunc) error {
	active, err := nullLocker(t, nulllocker.VariantA, []byte("active"))
	if err != nil {
		return fmt.Errorf("null locker(active): %w", err)
	}
	other, err := nullLocker(t, nulllocker.VariantB, []byte("other"))
	if err != nil {
		return fmt.Errorf("null locker(other): %w", err)
	}

	kr, err := newKeyring(active, other)
	if err != nil {
		return fmt.Errorf("new(active, other): %w", err)
	}

	found, ok := kr.Lookup(other.KeyID())
	if !ok {
		return fmt.Errorf("lookup(other): expected hit")
	}
	if found == nil {
		return fmt.Errorf("lookup(other): got nil locker")
	}
	if string(found.KeyID()) != string(other.KeyID()) {
		return fmt.Errorf("lookup(other) key id mismatch: got %q want %q", found.KeyID(), other.KeyID())
	}

	_, ok = kr.Lookup([]byte("missing"))
	if ok {
		return fmt.Errorf("lookup(missing): expected miss")
	}
	return nil
}

// checkRegisterRejectsNil verifies Register rejects a nil locker.
func checkRegisterRejectsNil(t *testing.T, newKeyring NewFunc) error {
	active, err := nullLocker(t, nulllocker.VariantA, []byte("active"))
	if err != nil {
		return fmt.Errorf("null locker(active): %w", err)
	}

	kr, err := newKeyring(active)
	if err != nil {
		return fmt.Errorf("new(active): %w", err)
	}
	err = kr.Register(nil)
	if !errors.Is(err, perrors.ErrInvalidNewArgs) {
		return fmt.Errorf("register(nil): got %v want %v", err, perrors.ErrInvalidNewArgs)
	}
	return nil
}

// checkRegisterRejectsDuplicateKeyID verifies Register rejects duplicate key identifiers.
func checkRegisterRejectsDuplicateKeyID(t *testing.T, newKeyring NewFunc) error {
	active, err := nullLocker(t, nulllocker.VariantA, []byte("dup"))
	if err != nil {
		return fmt.Errorf("null locker(dup): %w", err)
	}

	kr, err := newKeyring(active)
	if err != nil {
		return fmt.Errorf("new(active): %w", err)
	}
	dup, err := nullLocker(t, nulllocker.VariantA, []byte("dup"))
	if err != nil {
		return fmt.Errorf("null locker(dup): %w", err)
	}
	err = kr.Register(dup)
	if !errors.Is(err, perrors.ErrDuplicateKeyID) {
		return fmt.Errorf("register(duplicate): got %v want %v", err, perrors.ErrDuplicateKeyID)
	}
	return nil
}

// checkSetActiveRejectsNil verifies SetActive rejects nil input.
func checkSetActiveRejectsNil(t *testing.T, newKeyring NewFunc) error {
	active, err := nullLocker(t, nulllocker.VariantA, []byte("active"))
	if err != nil {
		return fmt.Errorf("null locker(active): %w", err)
	}

	kr, err := newKeyring(active)
	if err != nil {
		return fmt.Errorf("new(active): %w", err)
	}
	err = kr.SetActive(nil)
	if !errors.Is(err, perrors.ErrInvalidNewArgs) {
		return fmt.Errorf("set_active(nil): got %v want %v", err, perrors.ErrInvalidNewArgs)
	}
	return nil
}

// checkSetActiveRegistersAndSwitches verifies SetActive updates Active and ensures lookup registration.
func checkSetActiveRegistersAndSwitches(t *testing.T, newKeyring NewFunc) error {
	active, err := nullLocker(t, nulllocker.VariantA, []byte("active"))
	if err != nil {
		return fmt.Errorf("null locker(active): %w", err)
	}
	next, err := nullLocker(t, nulllocker.VariantB, []byte("next"))
	if err != nil {
		return fmt.Errorf("null locker(next): %w", err)
	}

	kr, err := newKeyring(active)
	if err != nil {
		return fmt.Errorf("new(active): %w", err)
	}

	if err = kr.SetActive(next); err != nil {
		return fmt.Errorf("set_active(next): %w", err)
	}
	if kr.Active() == nil {
		return fmt.Errorf("active after set_active: got nil")
	}
	if string(kr.Active().KeyID()) != string(next.KeyID()) {
		return fmt.Errorf("active after set_active mismatch: got %q want %q", kr.Active().KeyID(), next.KeyID())
	}

	found, ok := kr.Lookup(next.KeyID())
	if !ok {
		return fmt.Errorf("lookup(next): expected hit after set_active")
	}
	if found == nil {
		return fmt.Errorf("lookup(next): got nil locker")
	}
	if string(found.KeyID()) != string(next.KeyID()) {
		return fmt.Errorf("lookup(next) key id mismatch: got %q want %q", found.KeyID(), next.KeyID())
	}
	return nil
}

// checkRouteKeyIDCrossLockerParseFallback verifies routing does not depend on active locker parsing all formats.
func checkRouteKeyIDCrossLockerParseFallback(t *testing.T, newKeyring NewFunc) error {
	active, err := nullLocker(t, nulllocker.VariantA, []byte("active"))
	if err != nil {
		return fmt.Errorf("null locker(active): %w", err)
	}
	other, err := nullLocker(t, nulllocker.VariantB, []byte("other"))
	if err != nil {
		return fmt.Errorf("null locker(other): %w", err)
	}

	kr, err := newKeyring(active, other)
	if err != nil {
		return fmt.Errorf("new(active, other): %w", err)
	}

	wire, err := other.Seal(routeTestNS, []byte("route"))
	if err != nil {
		return fmt.Errorf("seal(other): %w", err)
	}
	found, err := kr.RouteKeyID(wire)
	if err != nil {
		return fmt.Errorf("route_key_id(other wire): %w", err)
	}
	if found == nil {
		return fmt.Errorf("route_key_id(other wire): got nil locker")
	}
	if string(found.KeyID()) != string(other.KeyID()) {
		return fmt.Errorf("route_key_id(other wire) key id mismatch: got %q want %q", found.KeyID(), other.KeyID())
	}
	return nil
}

// checkRouteKeyIDNoMatchReturnsErrNoLocker verifies unknown ciphertext returns ErrNoLocker.
func checkRouteKeyIDNoMatchReturnsErrNoLocker(t *testing.T, newKeyring NewFunc) error {
	active, err := nullLocker(t, nulllocker.VariantA, []byte("active"))
	if err != nil {
		return fmt.Errorf("null locker(active): %w", err)
	}

	kr, err := newKeyring(active)
	if err != nil {
		return fmt.Errorf("new(active): %w", err)
	}

	_, err = kr.RouteKeyID([]byte("not-null-locker-wire"))
	if !errors.Is(err, perrors.ErrNoLocker) {
		return fmt.Errorf("route_key_id(garbage): got %v want %v", err, perrors.ErrNoLocker)
	}
	return nil
}
