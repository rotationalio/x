// Package keyringtest provides conformance helpers for keyring implementations.
package keyringtest

// Keyring conformance helpers for tests.

import (
	"errors"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
)

// NewFunc constructs a keyring from an active locker and optional others.
type NewFunc func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error)

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
		fn   func(NewFunc) error
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
			assert.Ok(t, c.fn(newKeyring))
		})
	}
}

//=============================================================================
// Checks (one invariant per function so failure messages pinpoint the contract)
//=============================================================================

// checkNewRejectsNilActive verifies New rejects a nil active locker.
func checkNewRejectsNilActive(newKeyring NewFunc) error {
	_, err := newKeyring(nil)
	if !errors.Is(err, perrors.ErrInvalidNewArgs) {
		return fmt.Errorf("new(nil): got %v want %v", err, perrors.ErrInvalidNewArgs)
	}
	return nil
}

// checkActiveReturnsWriteLocker verifies Active returns the locker passed to New.
func checkActiveReturnsWriteLocker(newKeyring NewFunc) error {
	active := parseLocker{keyID: []byte("active")}

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
func checkLookupRegisteredAndUnknown(newKeyring NewFunc) error {
	active := parseLocker{keyID: []byte("active")}
	other := parseLocker{keyID: []byte("other")}

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
func checkRegisterRejectsNil(newKeyring NewFunc) error {
	active := parseLocker{keyID: []byte("active")}

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
func checkRegisterRejectsDuplicateKeyID(newKeyring NewFunc) error {
	active := parseLocker{keyID: []byte("dup")}

	kr, err := newKeyring(active)
	if err != nil {
		return fmt.Errorf("new(active): %w", err)
	}
	err = kr.Register(parseLocker{keyID: []byte("dup")})
	if !errors.Is(err, perrors.ErrDuplicateKeyID) {
		return fmt.Errorf("register(duplicate): got %v want %v", err, perrors.ErrDuplicateKeyID)
	}
	return nil
}

// checkSetActiveRejectsNil verifies SetActive rejects nil input.
func checkSetActiveRejectsNil(newKeyring NewFunc) error {
	active := parseLocker{keyID: []byte("active")}

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
func checkSetActiveRegistersAndSwitches(newKeyring NewFunc) error {
	active := parseLocker{keyID: []byte("active")}
	next := parseLocker{keyID: []byte("next")}

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
func checkRouteKeyIDCrossLockerParseFallback(newKeyring NewFunc) error {
	active := parseLocker{keyID: []byte("active"), tag: "active"}
	other := parseLocker{keyID: []byte("other"), tag: "other"}

	kr, err := newKeyring(active, other)
	if err != nil {
		return fmt.Errorf("new(active, other): %w", err)
	}

	found, err := kr.RouteKeyID([]byte("wire:other"))
	if err != nil {
		return fmt.Errorf("route_key_id(wire:other): %w", err)
	}
	if found == nil {
		return fmt.Errorf("route_key_id(wire:other): got nil locker")
	}
	if string(found.KeyID()) != string(other.KeyID()) {
		return fmt.Errorf("route_key_id(wire:other) key id mismatch: got %q want %q", found.KeyID(), other.KeyID())
	}
	return nil
}

// checkRouteKeyIDNoMatchReturnsErrNoLocker verifies unknown ciphertext returns ErrNoLocker.
func checkRouteKeyIDNoMatchReturnsErrNoLocker(newKeyring NewFunc) error {
	active := parseLocker{keyID: []byte("active"), tag: "active"}

	kr, err := newKeyring(active)
	if err != nil {
		return fmt.Errorf("new(active): %w", err)
	}

	_, err = kr.RouteKeyID([]byte("wire:unknown"))
	if !errors.Is(err, perrors.ErrNoLocker) {
		return fmt.Errorf("route_key_id(wire:unknown): got %v want %v", err, perrors.ErrNoLocker)
	}
	return nil
}

//=============================================================================
// Test stub locker
//
// parseLocker is a tag-driven stub: it lets each check assign a synthetic KeyID
// and a string tag so RouteKeyID conformance can be exercised without depending on
// any real wire format. The null locker can't replace it because parseLocker's
// ParseKeyID matches a custom "wire:<tag>" prefix the tests inject; null locker's
// wire format is fixed.
//=============================================================================

// parseLocker is a minimal locker used by conformance tests.
type parseLocker struct {
	keyID []byte
	tag   string
}

// KeyID returns the configured key identifier.
func (l parseLocker) KeyID() []byte { return append([]byte(nil), l.keyID...) }

// Seal is not used by these conformance helpers.
func (l parseLocker) Seal(string, []byte) ([]byte, error) { return nil, fmt.Errorf("not implemented") }

// Open is not used by these conformance helpers.
func (l parseLocker) Open(string, []byte) ([]byte, error) { return nil, fmt.Errorf("not implemented") }

// ParseKeyID recognizes ciphertext with prefix "wire:<tag>" and returns the configured key ID.
func (l parseLocker) ParseKeyID(ciphertext []byte) ([]byte, error) {
	want := "wire:" + l.tag
	if string(ciphertext) != want {
		return nil, fmt.Errorf("parse failed")
	}
	return l.KeyID(), nil
}
