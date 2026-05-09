package vaulttest

// In-memory [vault.Storage] for tests ([MemStorage]).

import (
	"context"
	"sync"
	"testing"

	"go.rtnl.ai/x/vault"
)

//=============================================================================
// Types
//=============================================================================

// mapKey is the [sync.Map] key for one logical row (namespace, id).
type mapKey struct {
	ns, id string
}

// MemStorage is an in-memory [vault.Storage]. Ciphertext values are opaque bytes;
// internally they are stored as strings so [sync.Map.CompareAndSwap] compares
// blob content correctly (distinct []byte snapshots are not comparable via interface equality).
type MemStorage struct {
	m sync.Map // mapKey -> string (opaque blob)
}

//=============================================================================
// Constructors
//=============================================================================

// NewMemStorage returns an empty [MemStorage] ready for use.
func NewMemStorage() *MemStorage {
	return &MemStorage{}
}

// MapStorage returns an empty [MemStorage] and marks tb as a test helper.
func MapStorage(tb testing.TB) *MemStorage {
	tb.Helper()
	return NewMemStorage()
}

//=============================================================================
// vault.Storage
//=============================================================================

// Create inserts a new row; duplicate (namespace, id) returns [vault.ErrDuplicateKey].
func (s *MemStorage) Create(ctx context.Context, namespace, id string, ciphertext []byte) error {
	_ = ctx
	k := blobKey(namespace, id)
	val := string(append([]byte(nil), ciphertext...))
	_, loaded := s.m.LoadOrStore(k, val)
	if loaded {
		return vault.ErrDuplicateKey
	}
	return nil
}

// Get returns a fresh copy of the blob or [vault.ErrNotFound].
func (s *MemStorage) Get(ctx context.Context, namespace, id string) ([]byte, error) {
	_ = ctx
	k := blobKey(namespace, id)
	v, ok := s.m.Load(k)
	if !ok {
		return nil, vault.ErrNotFound
	}
	return blobCopy(v.(string)), nil
}

// Replace overwrites ciphertext for an existing row; missing row returns [vault.ErrNotFound].
func (s *MemStorage) Replace(ctx context.Context, namespace, id string, ciphertext []byte) error {
	_ = ctx
	k := blobKey(namespace, id)
	if _, ok := s.m.Load(k); !ok {
		return vault.ErrNotFound
	}
	s.m.Store(k, string(append([]byte(nil), ciphertext...)))
	return nil
}

// Delete removes a row if present; missing key is a no-op (always returns nil).
func (s *MemStorage) Delete(ctx context.Context, namespace, id string) error {
	_ = ctx
	s.m.Delete(blobKey(namespace, id))
	return nil
}

// CompareAndSwap sets newCiphertext only when the stored blob equals oldCiphertext.
// Wrong old value returns [vault.ErrCASFailed]; missing row returns [vault.ErrNotFound].
func (s *MemStorage) CompareAndSwap(ctx context.Context, namespace, id string, oldCiphertext, newCiphertext []byte) error {
	_ = ctx
	k := blobKey(namespace, id)
	wantOld := string(oldCiphertext)
	next := string(append([]byte(nil), newCiphertext...))

	for {
		v, ok := s.m.Load(k)
		if !ok {
			return vault.ErrNotFound
		}
		cur := v.(string)
		if cur != wantOld {
			return vault.ErrCASFailed
		}
		if s.m.CompareAndSwap(k, cur, next) {
			return nil
		}
		// Lost a race: retry (delete → NotFound; new value → CASFailed on next iter).
	}
}

//=============================================================================
// Test hooks
//=============================================================================

// SetTestBlob overwrites the stored blob for (namespace, id) without checking
// row existence or duplicate semantics; for tests that need a corrupt or
// synthetic ciphertext.
func (s *MemStorage) SetTestBlob(namespace, id string, ciphertext []byte) {
	k := blobKey(namespace, id)
	s.m.Store(k, string(append([]byte(nil), ciphertext...)))
}

//=============================================================================
// Helpers
//=============================================================================

// blobKey builds the map key for a row.
func blobKey(namespace, id string) mapKey {
	return mapKey{ns: namespace, id: id}
}

// blobCopy returns an independent []byte copy of the stored string payload.
func blobCopy(s string) []byte {
	return append([]byte(nil), s...)
}
