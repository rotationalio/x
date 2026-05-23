// Package memring provides an in-memory [keyring.Keyring] for tests and simple programs.
// Single-import clients use [purser.Memring] and [purser.NewMemring] (see purser aliases.go).
package memring

import (
	"errors"
	"sync"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/keyring"
	"go.rtnl.ai/x/purser/keyring/keyspec"
	"go.rtnl.ai/x/purser/keyring/registry"
	"go.rtnl.ai/x/purser/locker"
)

// Memring is a thread-safe in-memory keyring.
type Memring struct {
	mu sync.RWMutex

	byID       map[string]locker.Locker
	namespaces map[string]string // namespace -> keyID string
	defaultID  string
}

var (
	_ keyring.Registrator = (*Memring)(nil)
	_ keyring.Namespacer  = (*Memring)(nil)
	_ keyring.Router      = (*Memring)(nil)
	_ keyring.Keyring     = (*Memring)(nil)
)

// New returns an empty Memring.
func New() *Memring {
	return &Memring{
		byID:       make(map[string]locker.Locker),
		namespaces: make(map[string]string),
	}
}

//=============================================================================
// Registrator
//=============================================================================

// Register builds a locker from spec and indexes it for decrypt routing.
func (m *Memring) Register(spec *keyspec.KeySpec) (locker.Locker, error) {
	lck, err := spec.Locker()
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err = indexLocker(m.byID, lck); err != nil {
		return nil, err
	}
	return lck, nil
}

// Revoke removes a locker and any namespace bindings for its key id.
func (m *Memring) Revoke(keyID []byte) error {
	if len(keyID) == 0 {
		return perrors.ErrInvalidNewArgs
	}
	k := string(keyID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.byID[k]; !ok {
		return perrors.ErrNoLocker
	}
	delete(m.byID, k)

	for ns, id := range m.namespaces {
		if id == k {
			delete(m.namespaces, ns)
		}
	}
	if m.defaultID == k {
		m.defaultID = ""
	}
	return nil
}

//=============================================================================
// Namespacer
//=============================================================================

// Bind associates namespace with an indexed locker. Use [Memring.SetDefault] for the fallback write locker.
func (m *Memring) Bind(namespace string, l locker.Locker) error {
	if namespace == "" || l == nil {
		return perrors.ErrInvalidNewArgs
	}
	kid := string(l.KeyID())
	if len(kid) == 0 {
		return perrors.ErrInvalidNewArgs
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.namespaces[namespace]; exists {
		return perrors.ErrAlreadyBound
	}
	if err := indexLocker(m.byID, l); err != nil {
		return err
	}
	m.namespaces[namespace] = kid
	return nil
}

// Unbind removes a namespace binding and returns the former key id.
func (m *Memring) Unbind(namespace string) (locker.Locker, error) {
	if namespace == "" {
		return nil, perrors.ErrInvalidNewArgs
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	kid, ok := m.namespaces[namespace]
	if !ok {
		return nil, perrors.ErrNotBound
	}
	delete(m.namespaces, namespace)
	return m.byID[kid], nil
}

// SetDefault sets the fallback write locker.
func (m *Memring) SetDefault(l locker.Locker) error {
	if l == nil {
		return perrors.ErrInvalidNewArgs
	}
	kid := l.KeyID()
	if len(kid) == 0 {
		return perrors.ErrInvalidNewArgs
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := indexLocker(m.byID, l); err != nil {
		return err
	}
	m.defaultID = string(kid)
	return nil
}

// Namespaces returns a snapshot of namespace to key id bindings.
func (m *Memring) Namespaces() map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string][]byte, len(m.namespaces))
	for ns, kid := range m.namespaces {
		out[ns] = []byte(kid)
	}
	return out
}

// LockerFor returns the locker bound to namespace, or the default when namespace is empty or unbound.
func (m *Memring) LockerFor(namespace string) (locker.Locker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if kid, ok := m.namespaces[namespace]; ok {
		if lck, found := m.byID[kid]; found {
			return lck, nil
		}
	}
	if m.defaultID != "" {
		if lck, ok := m.byID[m.defaultID]; ok {
			return lck, nil
		}
	}
	return nil, perrors.ErrNoLocker
}

//=============================================================================
// Router
//=============================================================================

// ParseKeyID delegates to registry with a nulllocker fallback for test wire.
func (m *Memring) ParseKeyID(ciphertext []byte) ([]byte, error) {
	keyID, err := registry.ParseKeyID(ciphertext)
	if err == nil {
		return keyID, nil
	}
	if !errors.Is(err, perrors.ErrUnrecognizedCiphertext) {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, lck := range m.byID {
		kid, parseErr := lck.ParseKeyID(ciphertext)
		if parseErr != nil {
			continue
		}
		return kid, nil
	}
	return nil, perrors.ErrUnrecognizedCiphertext
}

// Route resolves a locker for decrypting ciphertext.
func (m *Memring) Route(ciphertext []byte) (locker.Locker, error) {
	keyID, err := m.ParseKeyID(ciphertext)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if lck, ok := m.byID[string(keyID)]; ok {
		return lck, nil
	}
	return nil, perrors.ErrNoLocker
}

// indexLocker adds lck to byID or returns ErrDuplicateKeyID for a different instance.
func indexLocker(byID map[string]locker.Locker, lck locker.Locker) error {
	if lck == nil {
		return perrors.ErrInvalidNewArgs
	}
	kid := lck.KeyID()
	if len(kid) == 0 {
		return perrors.ErrInvalidNewArgs
	}
	k := string(kid)
	if existing, exists := byID[k]; exists {
		if existing == lck {
			return nil
		}
		return perrors.ErrDuplicateKeyID
	}
	byID[k] = lck
	return nil
}
