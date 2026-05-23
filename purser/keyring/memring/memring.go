// Package memring provides an in-memory [contract.Keyring] implementation.
// RouteKeyID delegates wire classification to [registry.ParseKeyID] before keyring lookup.
package memring

import (
	"errors"
	"sync"

	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/registry"
)

// Memring is a thread-safe in-memory keyring that supports runtime Register and SetActive operations.
type Memring struct {
	mu     sync.RWMutex
	active contract.Locker
	byID   map[string]contract.Locker
}

// Memring implements [contract.Keyring].
var _ contract.Keyring = (*Memring)(nil)

// New builds an in-memory key registry with an active locker and optional others.
func New(active contract.Locker, others ...contract.Locker) (*Memring, error) {
	if active == nil {
		return nil, perrors.ErrInvalidNewArgs
	}

	m := &Memring{
		active: active,
		byID:   make(map[string]contract.Locker),
	}

	if err := registerIntoMap(m.byID, active); err != nil {
		return nil, err
	}
	for _, lck := range others {
		if err := registerIntoMap(m.byID, lck); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// Active returns the current write locker.
func (m *Memring) Active() contract.Locker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// Lookup returns a registered locker for keyID.
func (m *Memring) Lookup(keyID []byte) (contract.Locker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	lck, ok := m.byID[string(keyID)]
	return lck, ok
}

// Register adds a locker for decrypt routing.
func (m *Memring) Register(lck contract.Locker) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return registerIntoMap(m.byID, lck)
}

// SetActive sets the write locker. The locker is registered if it wasn't already present.
// Re-activating an already-registered locker (same key ID) is allowed.
func (m *Memring) SetActive(lck contract.Locker) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if lck == nil {
		return perrors.ErrInvalidNewArgs
	}
	keyID := lck.KeyID()
	if len(keyID) == 0 {
		return perrors.ErrInvalidNewArgs
	}

	// Allow re-activation of an already-registered locker.
	k := string(keyID)
	if _, exists := m.byID[k]; !exists {
		m.byID[k] = lck
	}

	m.active = lck
	return nil
}

// RouteKeyID parses the key identifier and returns the matching locker.
// If the key identifier is not recognized, it returns [ErrNoLocker]. If the
// ciphertext is not a valid purser wire, it returns [ErrUnrecognizedCiphertext].
func (m *Memring) RouteKeyID(ciphertext []byte) (contract.Locker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try registry.ParseKeyID first to avoid unnecessary decryption.
	keyID, err := registry.ParseKeyID(ciphertext)
	if err == nil {
		if lck, ok := m.byID[string(keyID)]; ok {
			return lck, nil
		}
		return nil, perrors.ErrNoLocker
	}
	if !errors.Is(err, perrors.ErrUnrecognizedCiphertext) {
		return nil, err
	}

	// Fall back for non-PURS wire (e.g. test nulllocker).
	for _, lck := range m.byID {
		kid, parseErr := lck.ParseKeyID(ciphertext)
		if parseErr != nil {
			continue
		}
		if found, ok := m.byID[string(kid)]; ok {
			return found, nil
		}
	}
	return nil, perrors.ErrNoLocker
}

// registerIntoMap validates and registers a locker by key id. Duplicate key IDs are rejected
// so callers cannot silently shadow an existing locker.
func registerIntoMap(byID map[string]contract.Locker, lck contract.Locker) error {
	if lck == nil {
		return perrors.ErrInvalidNewArgs
	}
	keyID := lck.KeyID()
	if len(keyID) == 0 {
		return perrors.ErrInvalidNewArgs
	}
	k := string(keyID)
	if _, exists := byID[k]; exists {
		return perrors.ErrDuplicateKeyID
	}
	byID[k] = lck
	return nil
}
