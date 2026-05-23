package keyringtest

// Negative conformance tests for the Keyring contract (see [keyring.Keyring]).
//
// The helpers in keyring.go (check…, KeyringConforms) encode contracts that real keyring
// implementations such as memring must satisfy. Each subtest here wires a deliberately
// broken fake into one of those checks and asserts the check returns a non-nil error—
// proving the check would catch a non-conforming implementation.

import (
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/internal/nulllocker"
	"go.rtnl.ai/x/purser/keyring"
	"go.rtnl.ai/x/purser/keyring/keyspec"
	"go.rtnl.ai/x/purser/locker"
)

//=============================================================================
// Tests: negative Keyring conformance
//=============================================================================

// TestKeyringConformance_negative runs table-style subtests; each pairs a broken keyring
// with a conformance check that should detect the defect.
func TestKeyringConformance_negative(t *testing.T) {
	t.Run("locker_for_requires_default_or_bind", func(t *testing.T) {
		var err error
		lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("permissive"))
		assert.Ok(t, err)

		err = checkLockerForRequiresBinding(t, newFunc(&krLockerForAlways{lck: lck}))
		assert.Error(t, err, "expected conformance check to fail")
	})

	t.Run("set_default_and_locker_for", func(t *testing.T) {
		err := checkSetDefaultAndLockerFor(t, newFunc(krSetDefaultNoop{}))
		assert.Error(t, err, "expected conformance check to fail")
	})

	t.Run("bind_already_bound", func(t *testing.T) {
		err := checkBindAlreadyBound(t, newFunc(&stubBindKeyring{allowRebind: true}))
		assert.Error(t, err, "expected conformance check to fail for permissive bind")
	})

	t.Run("route_null_locker", func(t *testing.T) {
		var err error
		var sealed, wrong locker.Locker

		sealed, err = nulllocker.New(t, nulllocker.VariantB, []byte("route-sealed"))
		assert.Ok(t, err)
		wrong, err = nulllocker.New(t, nulllocker.VariantC, []byte("route-wrong"))
		assert.Ok(t, err)

		err = checkRouteNullLocker(t, newFunc(&krRouteWrong{sealed: sealed, wrong: wrong}))
		assert.Error(t, err, "expected conformance check to fail for wrong route")
	})

	t.Run("register_route_revoke", func(t *testing.T) {
		err := checkRegisterRouteRevoke(t, newFunc(&krRevokeNoop{}))
		assert.Error(t, err, "expected conformance check to fail for noop revoke")
	})

	t.Run("revoke_missing", func(t *testing.T) {
		err := checkRevokeMissing(t, newFunc(krRevokeSilent{}))
		assert.Error(t, err, "expected conformance check to fail for silent revoke")
	})

	t.Run("unbind_returns_locker", func(t *testing.T) {
		var err error
		wrong, err := nulllocker.New(t, nulllocker.VariantC, []byte("wrong-unbind"))
		assert.Ok(t, err)

		err = checkUnbindReturnsLocker(t, newFunc(&stubBindKeyring{unbindReturn: wrong}))
		assert.Error(t, err, "expected conformance check to fail for wrong unbind locker")
	})

	t.Run("unbind_not_bound", func(t *testing.T) {
		var err error
		lck, err := nulllocker.New(t, nulllocker.VariantA, []byte("permissive-unbind"))
		assert.Ok(t, err)

		err = checkUnbindNotBound(t, newFunc(krUnbindPermissive{lck: lck}))
		assert.Error(t, err, "expected conformance check to fail for permissive unbind")
	})

	t.Run("namespaces_snapshot", func(t *testing.T) {
		err := checkNamespacesSnapshot(t, newFunc(&stubBindKeyring{namespacesAlwaysEmpty: true}))
		assert.Error(t, err, "expected conformance check to fail for empty namespaces map")
	})

	t.Run("locker_for_bind_over_default", func(t *testing.T) {
		err := checkLockerForBindOverDefault(t, newFunc(&stubBindKeyring{ignoreBindInLockerFor: true}))
		assert.Error(t, err, "expected conformance check to fail when bind is ignored")
	})

	t.Run("parse_key_id", func(t *testing.T) {
		err := checkParseKeyID(t, newFunc(&krParseKeyIDWrong{}))
		assert.Error(t, err, "expected conformance check to fail for wrong parse_key_id")
	})

	t.Run("duplicate_key_id", func(t *testing.T) {
		err := checkDuplicateKeyID(t, newFunc(&stubBindKeyring{bindPermissive: true}))
		assert.Error(t, err, "expected conformance check to fail for permissive duplicate key id")
	})
}

// newFunc wraps a fixed keyring in a NewFunc for conformance checks.
func newFunc(kr keyring.Keyring) NewFunc {
	return func() (keyring.Keyring, error) { return kr, nil }
}

//=============================================================================
// Broken keyring fakes
//=============================================================================

// denyKeyring is the default stub: every method returns ErrInvalidNewArgs.
type denyKeyring struct{}

// Register rejects all specs on denyKeyring.
func (denyKeyring) Register(*keyspec.KeySpec) (locker.Locker, error) {
	return nil, perrors.ErrInvalidNewArgs
}

// Revoke rejects all key ids on denyKeyring.
func (denyKeyring) Revoke([]byte) error {
	return perrors.ErrInvalidNewArgs
}

// SetDefault rejects on denyKeyring.
func (denyKeyring) SetDefault(locker.Locker) error {
	return perrors.ErrInvalidNewArgs
}

// Bind rejects on denyKeyring.
func (denyKeyring) Bind(string, locker.Locker) error {
	return perrors.ErrInvalidNewArgs
}

// Unbind rejects on denyKeyring.
func (denyKeyring) Unbind(string) (locker.Locker, error) {
	return nil, perrors.ErrInvalidNewArgs
}

// Namespaces returns nil on denyKeyring.
func (denyKeyring) Namespaces() map[string][]byte {
	return nil
}

// LockerFor rejects on denyKeyring.
func (denyKeyring) LockerFor(string) (locker.Locker, error) {
	return nil, perrors.ErrInvalidNewArgs
}

// ParseKeyID rejects on denyKeyring.
func (denyKeyring) ParseKeyID([]byte) ([]byte, error) {
	return nil, perrors.ErrInvalidNewArgs
}

// Route rejects on denyKeyring.
func (denyKeyring) Route([]byte) (locker.Locker, error) {
	return nil, perrors.ErrInvalidNewArgs
}

// krLockerForAlways returns a locker from LockerFor even when nothing is configured.
type krLockerForAlways struct {
	denyKeyring
	lck locker.Locker
}

// LockerFor always succeeds on krLockerForAlways.
func (k *krLockerForAlways) LockerFor(string) (locker.Locker, error) {
	return k.lck, nil
}

// krSetDefaultNoop accepts SetDefault but never makes LockerFor succeed.
type krSetDefaultNoop struct {
	denyKeyring
}

// SetDefault is a no-op on krSetDefaultNoop.
func (krSetDefaultNoop) SetDefault(locker.Locker) error {
	return nil
}

// LockerFor always returns ErrNoLocker on krSetDefaultNoop.
func (krSetDefaultNoop) LockerFor(string) (locker.Locker, error) {
	return nil, perrors.ErrNoLocker
}

// stubBindKeyring implements SetDefault/Bind/LockerFor/Namespaces with configurable defects.
type stubBindKeyring struct {
	denyKeyring
	defaultLck locker.Locker
	bound      map[string]locker.Locker

	allowRebind           bool
	ignoreBindInLockerFor bool
	namespacesAlwaysEmpty bool
	bindPermissive        bool
	unbindReturn          locker.Locker
}

// SetDefault records the default locker on stubBindKeyring.
func (k *stubBindKeyring) SetDefault(l locker.Locker) error {
	k.defaultLck = l
	return nil
}

// Bind stores or overwrites namespace bindings per stubBindKeyring flags.
func (k *stubBindKeyring) Bind(namespace string, l locker.Locker) error {
	if k.bindPermissive {
		return nil
	}
	if k.namespacesAlwaysEmpty {
		return nil
	}
	if k.bound == nil {
		k.bound = make(map[string]locker.Locker)
	}
	if !k.allowRebind {
		if _, exists := k.bound[namespace]; exists {
			return perrors.ErrAlreadyBound
		}
	}
	k.bound[namespace] = l
	return nil
}

// Unbind returns a configured locker or rejects on stubBindKeyring.
func (k *stubBindKeyring) Unbind(string) (locker.Locker, error) {
	if k.unbindReturn != nil {
		return k.unbindReturn, nil
	}
	return nil, perrors.ErrInvalidNewArgs
}

// Namespaces returns an empty or bound snapshot per stubBindKeyring flags.
func (k *stubBindKeyring) Namespaces() map[string][]byte {
	if k.namespacesAlwaysEmpty {
		return map[string][]byte{}
	}
	if k.bound == nil {
		return nil
	}
	out := make(map[string][]byte, len(k.bound))
	for ns, lck := range k.bound {
		out[ns] = lck.KeyID()
	}
	return out
}

// LockerFor resolves bound or default locker per stubBindKeyring flags.
func (k *stubBindKeyring) LockerFor(ns string) (locker.Locker, error) {
	if k.defaultLck == nil {
		return nil, perrors.ErrNoLocker
	}
	if !k.ignoreBindInLockerFor {
		if lck, ok := k.bound[ns]; ok {
			return lck, nil
		}
	}
	return k.defaultLck, nil
}

// krRouteWrong routes every wire blob to a locker that did not seal it.
type krRouteWrong struct {
	denyKeyring
	sealed locker.Locker
	wrong  locker.Locker
}

// SetDefault records the sealing locker on krRouteWrong.
func (k *krRouteWrong) SetDefault(l locker.Locker) error {
	k.sealed = l
	return nil
}

// LockerFor returns the sealing locker on krRouteWrong.
func (k *krRouteWrong) LockerFor(string) (locker.Locker, error) {
	return k.sealed, nil
}

// Route returns the wrong locker on krRouteWrong.
func (k *krRouteWrong) Route([]byte) (locker.Locker, error) {
	return k.wrong, nil
}

// krRevokeNoop registers lockers but Revoke is a no-op.
type krRevokeNoop struct {
	byID map[string]locker.Locker
}

// Register indexes a locker from spec on krRevokeNoop.
func (k *krRevokeNoop) Register(spec *keyspec.KeySpec) (locker.Locker, error) {
	lck, err := spec.Locker()
	if err != nil {
		return nil, err
	}
	if k.byID == nil {
		k.byID = make(map[string]locker.Locker)
	}
	k.byID[string(lck.KeyID())] = lck
	return lck, nil
}

// Revoke is a no-op on krRevokeNoop.
func (k *krRevokeNoop) Revoke([]byte) error {
	return nil
}

// SetDefault rejects on krRevokeNoop.
func (k *krRevokeNoop) SetDefault(locker.Locker) error {
	return perrors.ErrInvalidNewArgs
}

// Bind rejects on krRevokeNoop.
func (k *krRevokeNoop) Bind(string, locker.Locker) error {
	return perrors.ErrInvalidNewArgs
}

// Unbind rejects on krRevokeNoop.
func (k *krRevokeNoop) Unbind(string) (locker.Locker, error) {
	return nil, perrors.ErrInvalidNewArgs
}

// Namespaces returns nil on krRevokeNoop.
func (k *krRevokeNoop) Namespaces() map[string][]byte {
	return nil
}

// LockerFor rejects on krRevokeNoop.
func (k *krRevokeNoop) LockerFor(string) (locker.Locker, error) {
	return nil, perrors.ErrInvalidNewArgs
}

// ParseKeyID scans indexed lockers on krRevokeNoop.
func (k *krRevokeNoop) ParseKeyID(wire []byte) ([]byte, error) {
	for _, lck := range k.byID {
		kid, err := lck.ParseKeyID(wire)
		if err == nil {
			return kid, nil
		}
	}
	return nil, perrors.ErrUnrecognizedCiphertext
}

// Route resolves wire via ParseKeyID on krRevokeNoop.
func (k *krRevokeNoop) Route(wire []byte) (locker.Locker, error) {
	kid, err := k.ParseKeyID(wire)
	if err != nil {
		return nil, err
	}
	lck, ok := k.byID[string(kid)]
	if !ok {
		return nil, perrors.ErrNoLocker
	}
	return lck, nil
}

// krRevokeSilent returns nil when revoking an unknown key id.
type krRevokeSilent struct {
	denyKeyring
}

// Revoke always succeeds on krRevokeSilent.
func (krRevokeSilent) Revoke([]byte) error {
	return nil
}

// krUnbindPermissive succeeds on Unbind for namespaces that were never bound.
type krUnbindPermissive struct {
	denyKeyring
	lck locker.Locker
}

// Unbind always returns a locker on krUnbindPermissive.
func (k krUnbindPermissive) Unbind(string) (locker.Locker, error) {
	return k.lck, nil
}

// krParseKeyIDWrong returns a constant bogus key id from ParseKeyID.
type krParseKeyIDWrong struct {
	denyKeyring
	sealed locker.Locker
}

// SetDefault records the sealing locker on krParseKeyIDWrong.
func (k *krParseKeyIDWrong) SetDefault(l locker.Locker) error {
	k.sealed = l
	return nil
}

// LockerFor returns the sealing locker on krParseKeyIDWrong.
func (k *krParseKeyIDWrong) LockerFor(string) (locker.Locker, error) {
	return k.sealed, nil
}

// ParseKeyID returns bogus bytes on krParseKeyIDWrong.
func (k *krParseKeyIDWrong) ParseKeyID([]byte) ([]byte, error) {
	return []byte("bogus"), nil
}
