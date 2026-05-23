// Package keyringtest provides conformance helpers for keyring.Keyring implementations.
//
// KeyringConforms exercises Registrator, Namespacer, and Router contracts using
// nulllocker-shaped wire where routing does not require v1 registry.ParseKeyID parsing.
// Production keyrings use [purser.NewMemring] and [purser.KeySpec] registration (see purser aliases.go).
package keyringtest

// Keyring conformance helpers for tests.

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring"
	"go.rtnl.ai/x/purser/keyring/keyspec"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
)

// NewFunc constructs a keyring for conformance checks.
type NewFunc func() (keyring.Keyring, error)

//=============================================================================
// Public conformance suite
//=============================================================================

// KeyringConforms runs every documented contract from the Keyring interface against
// the implementation produced by newKeyring. Each check builds a fresh keyring.
func KeyringConforms(t *testing.T, newKeyring NewFunc) {
	t.Helper()

	checks := []struct {
		name string
		fn   func(*testing.T, NewFunc) error
	}{
		{"locker_for_requires_default_or_bind", checkLockerForRequiresBinding},
		{"set_default_and_locker_for", checkSetDefaultAndLockerFor},
		{"bind_already_bound", checkBindAlreadyBound},
		{"route_null_locker", checkRouteNullLocker},
		{"register_route_revoke", checkRegisterRouteRevoke},
		{"revoke_missing", checkRevokeMissing},
		{"unbind_returns_locker", checkUnbindReturnsLocker},
		{"unbind_not_bound", checkUnbindNotBound},
		{"namespaces_snapshot", checkNamespacesSnapshot},
		{"locker_for_bind_over_default", checkLockerForBindOverDefault},
		{"parse_key_id", checkParseKeyID},
		{"duplicate_key_id", checkDuplicateKeyID},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			assert.Ok(t, c.fn(t, newKeyring))
		})
	}
}

//=============================================================================
// Checks (one invariant per function so failure messages pinpoint the contract)
//=============================================================================

// checkLockerForRequiresBinding expects ErrNoLocker when no default or bind is set.
func checkLockerForRequiresBinding(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	_, err = kr.LockerFor(routeTestNS)
	return wantErrorIs(err, perrors.ErrNoLocker, "locker_for")
}

// checkSetDefaultAndLockerFor verifies SetDefault serves unbound namespaces via LockerFor.
func checkSetDefaultAndLockerFor(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed-default"))
	if err != nil {
		return err
	}
	if err = kr.SetDefault(lck); err != nil {
		return fmt.Errorf("set_default: %w", err)
	}

	got, err := kr.LockerFor(routeTestNS)
	if err != nil {
		return fmt.Errorf("locker_for: %w", err)
	}
	if len(got.KeyID()) == 0 {
		return fmt.Errorf("locker_for: empty key id")
	}
	if !bytes.Equal(lck.KeyID(), got.KeyID()) {
		return fmt.Errorf("locker_for default: key id mismatch")
	}
	return nil
}

// checkBindAlreadyBound expects ErrAlreadyBound on a duplicate namespace bind.
func checkBindAlreadyBound(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed-bind"))
	if err != nil {
		return err
	}
	if err = kr.SetDefault(lck); err != nil {
		return fmt.Errorf("set_default: %w", err)
	}
	if err = kr.Bind("tenant", lck); err != nil {
		return fmt.Errorf("first bind: %w", err)
	}

	err = kr.Bind("tenant", lck)
	return wantErrorIs(err, perrors.ErrAlreadyBound, "second bind")
}

// checkRouteNullLocker verifies Route resolves null-locker test wire by key id.
func checkRouteNullLocker(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	lck, err := nulllocker.New(t, nulllocker.VariantB, []byte("seed-route"))
	if err != nil {
		return err
	}
	if err = kr.SetDefault(lck); err != nil {
		return fmt.Errorf("set_default: %w", err)
	}

	wire, err := lck.Seal(routeTestNS, []byte("plain"))
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}

	found, err := kr.Route(wire)
	if err != nil {
		return fmt.Errorf("route: %w", err)
	}
	if !bytes.Equal(lck.KeyID(), found.KeyID()) {
		return fmt.Errorf("route: key id mismatch")
	}
	return nil
}

// checkRegisterRouteRevoke verifies Register indexes a locker and Revoke removes it from routing.
func checkRegisterRouteRevoke(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	spec, err := testSeedSpec()
	if err != nil {
		return err
	}

	lck, err := kr.Register(spec)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}

	wire, err := lck.Seal(routeTestNS, []byte("registered"))
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}

	found, err := kr.Route(wire)
	if err != nil {
		return fmt.Errorf("route after register: %w", err)
	}
	if !bytes.Equal(lck.KeyID(), found.KeyID()) {
		return fmt.Errorf("route after register: key id mismatch")
	}

	if err = kr.Revoke(lck.KeyID()); err != nil {
		return fmt.Errorf("revoke: %w", err)
	}

	_, err = kr.Route(wire)
	return wantErrorIs(err, perrors.ErrNoLocker, "route after revoke")
}

// checkRevokeMissing expects ErrNoLocker when revoking an unknown key id.
func checkRevokeMissing(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	err = kr.Revoke([]byte("unknown-key-id-bytes"))
	return wantErrorIs(err, perrors.ErrNoLocker, "revoke missing")
}

// checkUnbindReturnsLocker verifies Unbind returns the former locker and namespace binding is cleared.
func checkUnbindReturnsLocker(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	defaultLck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed-unbind-default"))
	if err != nil {
		return err
	}
	boundLck, err := nulllocker.New(t, nulllocker.VariantB, []byte("seed-unbind-bound"))
	if err != nil {
		return err
	}
	if err = kr.SetDefault(defaultLck); err != nil {
		return fmt.Errorf("set_default: %w", err)
	}

	const boundNS = "bound-tenant"
	if err = kr.Bind(boundNS, boundLck); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	lck, err := kr.Unbind(boundNS)
	if err != nil {
		return fmt.Errorf("unbind: %w", err)
	}
	if !bytes.Equal(boundLck.KeyID(), lck.KeyID()) {
		return fmt.Errorf("unbind: key id mismatch")
	}

	got, err := kr.LockerFor(boundNS)
	if err != nil {
		return fmt.Errorf("locker_for after unbind: %w", err)
	}
	if !bytes.Equal(defaultLck.KeyID(), got.KeyID()) {
		return fmt.Errorf("locker_for after unbind: expected default key id")
	}
	return nil
}

// checkUnbindNotBound expects ErrNotBound when unbinding a namespace that was never bound.
func checkUnbindNotBound(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	_, err = kr.Unbind("never-bound")
	return wantErrorIs(err, perrors.ErrNotBound, "unbind")
}

// checkNamespacesSnapshot verifies Namespaces reflects bindings after Bind.
func checkNamespacesSnapshot(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	lckA, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed-ns-a"))
	if err != nil {
		return err
	}
	lckB, err := nulllocker.New(t, nulllocker.VariantB, []byte("seed-ns-b"))
	if err != nil {
		return err
	}
	if err = kr.SetDefault(lckA); err != nil {
		return fmt.Errorf("set_default: %w", err)
	}

	const nsOne = "tenant-one"
	const nsTwo = "tenant-two"
	if err = kr.Bind(nsOne, lckA); err != nil {
		return fmt.Errorf("bind %q: %w", nsOne, err)
	}
	if err = kr.Bind(nsTwo, lckB); err != nil {
		return fmt.Errorf("bind %q: %w", nsTwo, err)
	}

	snap := kr.Namespaces()
	if !bytes.Equal(lckA.KeyID(), snap[nsOne]) {
		return fmt.Errorf("namespaces %q: key id mismatch", nsOne)
	}
	if !bytes.Equal(lckB.KeyID(), snap[nsTwo]) {
		return fmt.Errorf("namespaces %q: key id mismatch", nsTwo)
	}
	return nil
}

// checkLockerForBindOverDefault verifies a namespace binding overrides the default write locker.
func checkLockerForBindOverDefault(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	defaultLck, err := nulllocker.New(t, nulllocker.VariantA, []byte("seed-precedence-default"))
	if err != nil {
		return err
	}
	boundLck, err := nulllocker.New(t, nulllocker.VariantC, []byte("seed-precedence-bound"))
	if err != nil {
		return err
	}
	if err = kr.SetDefault(defaultLck); err != nil {
		return fmt.Errorf("set_default: %w", err)
	}

	const boundNS = "precedence-tenant"
	if err = kr.Bind(boundNS, boundLck); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	got, err := kr.LockerFor(boundNS)
	if err != nil {
		return fmt.Errorf("locker_for: %w", err)
	}
	if !bytes.Equal(boundLck.KeyID(), got.KeyID()) {
		return fmt.Errorf("locker_for: expected bound locker key id")
	}
	return nil
}

// checkParseKeyID verifies ParseKeyID extracts the sealing key id from nulllocker wire.
func checkParseKeyID(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	lck, err := nulllocker.New(t, nulllocker.VariantB, []byte("seed-parse"))
	if err != nil {
		return err
	}
	if err = kr.SetDefault(lck); err != nil {
		return fmt.Errorf("set_default: %w", err)
	}

	wire, err := lck.Seal(routeTestNS, []byte("plain"))
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}

	kid, err := kr.ParseKeyID(wire)
	if err != nil {
		return fmt.Errorf("parse_key_id: %w", err)
	}
	if !bytes.Equal(lck.KeyID(), kid) {
		return fmt.Errorf("parse_key_id: key id mismatch")
	}
	return nil
}

// checkDuplicateKeyID expects ErrDuplicateKeyID when indexing two lockers with the same key id.
func checkDuplicateKeyID(t *testing.T, newKeyring NewFunc) error {
	t.Helper()

	var err error
	kr, err := newKeyring()
	if err != nil {
		return err
	}

	lckA, err := nulllocker.New(t, nulllocker.VariantA, []byte("dup-seed"))
	if err != nil {
		return err
	}
	lckB, err := nulllocker.New(t, nulllocker.VariantA, []byte("dup-seed"))
	if err != nil {
		return err
	}
	if err = kr.SetDefault(lckA); err != nil {
		return fmt.Errorf("set_default: %w", err)
	}

	err = kr.Bind("dup-ns", lckB)
	return wantErrorIs(err, perrors.ErrDuplicateKeyID, "duplicate bind")
}

//=============================================================================
// Helpers
//=============================================================================

// routeTestNS is the namespace used for routing checks that do not target a specific bind.
const routeTestNS = "keyringtest"

// wantErrorIs returns an error when err is not in the target error chain.
func wantErrorIs(err, target error, step string) error {
	if err == nil {
		return fmt.Errorf("%s: got nil want %w", step, target)
	}
	if !errors.Is(err, target) {
		return fmt.Errorf("%s: got %v want %w", step, err, target)
	}
	return nil
}

// testSeedSpec returns a fixed v1 seed KeySpec for Register conformance checks.
func testSeedSpec() (*keyspec.KeySpec, error) {
	seed := make([]byte, lockerv1.SeedBytes)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	return keyspec.NewSeed(seed, constv1.Edition)
}
