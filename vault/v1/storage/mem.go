package storage

// In-memory [Storage] using [sync.Map] for concurrent tests and examples.

import (
	"context"
	"sync"

	verrors "go.rtnl.ai/x/vault/v1/errors"
)

// MemStorage is an in-memory [Storage]. Ciphertext values are opaque bytes;
// internally they are stored as strings so [sync.Map.CompareAndSwap] compares
// blob content correctly (distinct []byte snapshots are not comparable via
// interface equality).
type MemStorage struct {
	m sync.Map // mapKey -> string (opaque blob)
}

// MemStorage implements [Storage].
var _ Storage = (*MemStorage)(nil)

// NewMemStorage returns an empty [MemStorage] ready for use.
func NewMemStorage() *MemStorage {
	return &MemStorage{}
}

// Create inserts a new row; duplicate (namespace, id) returns [verrors.ErrDuplicateKey].
func (s *MemStorage) Create(ctx context.Context, namespace, id string, ciphertext []byte) error {
	_ = ctx
	k := mapKey{ns: namespace, id: id}
	val := string(ciphertext)
	_, loaded := s.m.LoadOrStore(k, val)
	if loaded {
		return verrors.ErrDuplicateKey
	}
	return nil
}

// Get returns a fresh copy of the blob or [verrors.ErrNotFound].
func (s *MemStorage) Get(ctx context.Context, namespace, id string) ([]byte, error) {
	_ = ctx
	k := mapKey{ns: namespace, id: id}
	v, ok := s.m.Load(k)
	if !ok {
		return nil, verrors.ErrNotFound
	}
	return []byte(v.(string)), nil
}

// Replace overwrites ciphertext for an existing row; missing row returns [verrors.ErrNotFound].
func (s *MemStorage) Replace(ctx context.Context, namespace, id string, ciphertext []byte) error {
	_ = ctx
	k := mapKey{ns: namespace, id: id}
	if _, ok := s.m.Load(k); !ok {
		return verrors.ErrNotFound
	}
	s.m.Store(k, string(ciphertext))
	return nil
}

// Delete removes a row if present; missing key is a no-op (implementation always
// returns nil).
func (s *MemStorage) Delete(ctx context.Context, namespace, id string) error {
	_ = ctx
	s.m.Delete(mapKey{ns: namespace, id: id})
	return nil
}

// CompareAndSwap sets newCiphertext only when the stored blob equals oldCiphertext.
// Wrong old value returns [verrors.ErrCASFailed]; missing row returns [verrors.ErrNotFound].
func (s *MemStorage) CompareAndSwap(ctx context.Context, namespace, id string, oldCiphertext, newCiphertext []byte) error {
	_ = ctx
	k := mapKey{ns: namespace, id: id}
	wantOld := string(oldCiphertext)
	next := string(newCiphertext)

	// sync.Map compares the stored string to wantOld; distinct []byte snapshots
	// are not comparable as interface values.
	if s.m.CompareAndSwap(k, wantOld, next) {
		return nil
	}

	// If the CAS failed determine for which reason.
	if _, ok := s.m.Load(k); !ok {
		return verrors.ErrNotFound
	}
	return verrors.ErrCASFailed
}

// BypassSemanticsSetBlobForTest overwrites the stored blob for (namespace, id)
// without checking row existence or duplicate semantics; for tests that need a
// corrupt or synthetic ciphertext. The name is long and obtuse to discourage
// use outside of tests, in case you use [MemStorage] for production.
func (s *MemStorage) BypassSemanticsSetBlobForTest(namespace, id string, ciphertext []byte) {
	k := mapKey{ns: namespace, id: id}
	s.m.Store(k, string(ciphertext))
}

// mapKey is the key type for [sync.Map] in [MemStorage].
type mapKey struct {
	ns, id string
}
