package keyringtest

// Negative tests for keyring conformance checks.

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/internal/nulllocker"
)

//=============================================================================
// Tests: negative conformance checks
//=============================================================================

// TestKeyringConformanceChecks_negative ensures each check fails for a deliberately broken keyring.
func TestKeyringConformanceChecks_negative(t *testing.T) {
	t.Run("new_rejects_nil_active", func(t *testing.T) {
		err := checkNewRejectsNilActive(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			return &stubKeyring{}, nil
		})
		assert.Error(t, err)
	})

	t.Run("active_returns_write_locker", func(t *testing.T) {
		err := checkActiveReturnsWriteLocker(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			wrong, err := nullLocker(t, nulllocker.VariantA, []byte("wrong"))
			if err != nil {
				return nil, err
			}
			return &stubKeyring{active: wrong}, nil
		})
		assert.Error(t, err)
	})

	t.Run("lookup_registered_and_unknown", func(t *testing.T) {
		err := checkLookupRegisteredAndUnknown(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			return &stubKeyring{
				active: active,
				lookupFn: func([]byte) (contract.Locker, bool) {
					return nil, false
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("register_rejects_nil", func(t *testing.T) {
		err := checkRegisterRejectsNil(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			return &stubKeyring{
				active: active,
				registerFn: func(contract.Locker) error {
					return nil
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("register_rejects_duplicate_key_id", func(t *testing.T) {
		err := checkRegisterRejectsDuplicateKeyID(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			return &stubKeyring{
				active: active,
				registerFn: func(contract.Locker) error {
					return nil
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("set_active_rejects_nil", func(t *testing.T) {
		err := checkSetActiveRejectsNil(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			return &stubKeyring{
				active: active,
				setActiveFn: func(contract.Locker) error {
					return nil
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("set_active_registers_and_switches", func(t *testing.T) {
		err := checkSetActiveRegistersAndSwitches(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			return &stubKeyring{
				active: active,
				setActiveFn: func(l contract.Locker) error {
					_ = l
					return nil
				},
				lookupFn: func([]byte) (contract.Locker, bool) {
					return nil, false
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("route_key_id_cross_locker_parse_fallback", func(t *testing.T) {
		err := checkRouteKeyIDCrossLockerParseFallback(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			return &stubKeyring{
				active: active,
				routeFn: func([]byte) (contract.Locker, error) {
					return nil, perrors.ErrNoLocker
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("route_key_id_no_match_returns_err_no_locker", func(t *testing.T) {
		err := checkRouteKeyIDNoMatchReturnsErrNoLocker(t, func(active contract.Locker, others ...contract.Locker) (contract.Keyring, error) {
			return &stubKeyring{
				active: active,
				routeFn: func([]byte) (contract.Locker, error) {
					return active, nil
				},
			}, nil
		})
		assert.Error(t, err)
	})
}

//=============================================================================
// Stubs
//=============================================================================

// stubKeyring is a configurable test double for contract.Keyring behavior.
type stubKeyring struct {
	active      contract.Locker
	lookupFn    func([]byte) (contract.Locker, bool)
	registerFn  func(contract.Locker) error
	setActiveFn func(contract.Locker) error
	routeFn     func([]byte) (contract.Locker, error)
}

// Active returns the configured active locker.
func (s *stubKeyring) Active() contract.Locker { return s.active }

// Lookup delegates to lookupFn when provided.
func (s *stubKeyring) Lookup(keyID []byte) (contract.Locker, bool) {
	if s.lookupFn == nil {
		return nil, false
	}
	return s.lookupFn(keyID)
}

// Register delegates to registerFn when provided.
func (s *stubKeyring) Register(l contract.Locker) error {
	if s.registerFn == nil {
		return perrors.ErrInvalidNewArgs
	}
	return s.registerFn(l)
}

// SetActive delegates to setActiveFn when provided and updates active on success.
func (s *stubKeyring) SetActive(l contract.Locker) error {
	if s.setActiveFn == nil {
		return perrors.ErrInvalidNewArgs
	}
	if err := s.setActiveFn(l); err != nil {
		return err
	}
	s.active = l
	return nil
}

// RouteKeyID delegates to routeFn when provided.
func (s *stubKeyring) RouteKeyID(ciphertext []byte) (contract.Locker, error) {
	if s.routeFn == nil {
		return nil, perrors.ErrNoLocker
	}
	return s.routeFn(ciphertext)
}
