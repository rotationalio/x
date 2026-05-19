// Package memring provides an in-memory [purser.Keyring] implementation.
package memring

import (
	"sync"

	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
)

// Memring is a thread-safe in-memory keyring that supports runtime Register and SetActive operations.
type Memring struct {
	mu     sync.RWMutex
	active purser.Locker
	byID   map[string]purser.Locker
}

// Memring implements [purser.Keyring].
var _ purser.Keyring = (*Memring)(nil)

// New builds an in-memory key registry with an active locker and optional others.
func New(active purser.Locker, others ...purser.Locker) (*Memring, error) {
	if active == nil {
		return nil, perrors.ErrInvalidNewArgs
	}

	m := &Memring{
		active: active,
		byID:   make(map[string]purser.Locker),
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
func (m *Memring) Active() purser.Locker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// Lookup returns a registered locker for keyID.
func (m *Memring) Lookup(keyID []byte) (purser.Locker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	lck, ok := m.byID[string(keyID)]
	return lck, ok
}

// Register adds a locker for decrypt routing.
func (m *Memring) Register(lck purser.Locker) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return registerIntoMap(m.byID, lck)
}

// SetActive sets the write locker. The locker is registered if it wasn't already present.
// Re-activating an already-registered locker (same key ID) is allowed.
func (m *Memring) SetActive(lck purser.Locker) error {
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

// RouteKeyID tries each registered locker's ParseKeyID on ciphertext. The first locker whose
// ParseKeyID succeeds and whose extracted key ID maps to a registered locker wins. This lets
// a keyring with multiple locker versions (different wire formats) route ciphertext to the
// correct locker without requiring the active locker to understand every format.
func (m *Memring) RouteKeyID(ciphertext []byte) (purser.Locker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, lck := range m.byID {
		keyID, err := lck.ParseKeyID(ciphertext)
		if err != nil {
			continue
		}
		if found, ok := m.byID[string(keyID)]; ok {
			return found, nil
		}
	}

	return nil, perrors.ErrNoLocker
}

// registerIntoMap validates and registers a locker by key id. Duplicate key IDs are rejected
// so callers cannot silently shadow an existing locker.
func registerIntoMap(byID map[string]purser.Locker, lck purser.Locker) error {
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
