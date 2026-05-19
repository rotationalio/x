package keyringtest

// Negative tests for keyring conformance checks.

import (
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
)

//=============================================================================
// Tests: negative conformance checks
//=============================================================================

// TestKeyringConformanceChecks_negative ensures each check fails for a deliberately broken keyring.
func TestKeyringConformanceChecks_negative(t *testing.T) {
	t.Run("new_rejects_nil_active", func(t *testing.T) {
		err := checkNewRejectsNilActive(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{}, nil
		})
		assert.Error(t, err)
	})

	t.Run("active_returns_write_locker", func(t *testing.T) {
		err := checkActiveReturnsWriteLocker(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{active: testLocker{keyID: []byte("wrong")}}, nil
		})
		assert.Error(t, err)
	})

	t.Run("lookup_registered_and_unknown", func(t *testing.T) {
		err := checkLookupRegisteredAndUnknown(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{
				active: active,
				lookupFn: func([]byte) (purser.Locker, bool) {
					return nil, false
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("register_rejects_nil", func(t *testing.T) {
		err := checkRegisterRejectsNil(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{
				active: active,
				registerFn: func(purser.Locker) error {
					return nil
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("register_rejects_duplicate_key_id", func(t *testing.T) {
		err := checkRegisterRejectsDuplicateKeyID(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{
				active: active,
				registerFn: func(purser.Locker) error {
					return nil
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("set_active_rejects_nil", func(t *testing.T) {
		err := checkSetActiveRejectsNil(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{
				active: active,
				setActiveFn: func(purser.Locker) error {
					return nil
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("set_active_registers_and_switches", func(t *testing.T) {
		err := checkSetActiveRegistersAndSwitches(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{
				active: active,
				setActiveFn: func(l purser.Locker) error {
					_ = l
					return nil
				},
				lookupFn: func([]byte) (purser.Locker, bool) {
					return nil, false
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("route_key_id_cross_locker_parse_fallback", func(t *testing.T) {
		err := checkRouteKeyIDCrossLockerParseFallback(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{
				active: active,
				routeFn: func([]byte) (purser.Locker, error) {
					return nil, perrors.ErrNoLocker
				},
			}, nil
		})
		assert.Error(t, err)
	})

	t.Run("route_key_id_no_match_returns_err_no_locker", func(t *testing.T) {
		err := checkRouteKeyIDNoMatchReturnsErrNoLocker(func(active purser.Locker, others ...purser.Locker) (purser.Keyring, error) {
			return &stubKeyring{
				active: active,
				routeFn: func([]byte) (purser.Locker, error) {
					return testLocker{keyID: []byte("unexpected")}, nil
				},
			}, nil
		})
		assert.Error(t, err)
	})
}

//=============================================================================
// Stubs
//=============================================================================

// stubKeyring is a configurable test double for purser.Keyring behavior.
type stubKeyring struct {
	active      purser.Locker
	lookupFn    func([]byte) (purser.Locker, bool)
	registerFn  func(purser.Locker) error
	setActiveFn func(purser.Locker) error
	routeFn     func([]byte) (purser.Locker, error)
}

// Active returns the configured active locker.
func (s *stubKeyring) Active() purser.Locker { return s.active }

// Lookup delegates to lookupFn when provided.
func (s *stubKeyring) Lookup(keyID []byte) (purser.Locker, bool) {
	if s.lookupFn == nil {
		return nil, false
	}
	return s.lookupFn(keyID)
}

// Register delegates to registerFn when provided.
func (s *stubKeyring) Register(l purser.Locker) error {
	if s.registerFn == nil {
		return perrors.ErrInvalidNewArgs
	}
	return s.registerFn(l)
}

// SetActive delegates to setActiveFn when provided and updates active on success.
func (s *stubKeyring) SetActive(l purser.Locker) error {
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
func (s *stubKeyring) RouteKeyID(ciphertext []byte) (purser.Locker, error) {
	if s.routeFn == nil {
		return nil, perrors.ErrNoLocker
	}
	return s.routeFn(ciphertext)
}

// testLocker is a minimal purser.Locker used by keyring stubs.
type testLocker struct {
	keyID []byte
}

// KeyID returns the configured key identifier.
func (l testLocker) KeyID() []byte { return append([]byte(nil), l.keyID...) }

// Seal is not used by these tests.
func (l testLocker) Seal(string, []byte) ([]byte, error) { return nil, fmt.Errorf("not implemented") }

// Open is not used by these tests.
func (l testLocker) Open(string, []byte) ([]byte, error) { return nil, fmt.Errorf("not implemented") }

// ParseKeyID is not used by these tests.
func (l testLocker) ParseKeyID([]byte) ([]byte, error) { return nil, fmt.Errorf("not implemented") }
