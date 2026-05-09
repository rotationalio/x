// Package vaulttest provides test doubles for [go.rtnl.ai/x/vault]: in-memory [MemStorage],
// [HexIdentifier], and plaintext [TestVault] (no AES).
//
// Construct a real AES vault over memory for unit tests:
//
//	st := vaulttest.NewMemStorage()
//	v, err := vault.New(key32, st, vaulttest.HexIdentifier{})
//	id, err := v.Store(ctx, "ns", []byte("secret"))
//
// Exercise [vault.Storage] semantics without crypto (same id/namespace API, plaintext bytes):
//
//	tv := vaulttest.NewTestVault(t, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
//	id, err := tv.Store(ctx, "ns", []byte("plain"))
//	got, err := tv.Retrieve(ctx, "ns", id)
//
// Use [MemStorage] directly when a test only needs a map-like backend:
//
//	st := vaulttest.NewMemStorage()
//	_ = st.Create(ctx, "n", "id1", []byte("blob"))
//	b, err := st.Get(ctx, "n", "id1")
//
// Use [Run] to conformance-test any [vault.Storage] (fresh instance per subtest):
//
//	vaulttest.Run(t, func(tb *testing.T) vault.Storage {
//		tb.Helper()
//		return NewMemStorage() // or your backend
//	})
package vaulttest

import (
	"bytes"
	"context"
	"testing"

	"go.rtnl.ai/x/vault"
)

//=============================================================================
// TestVault (plaintext-through-storage)
//=============================================================================

// TestVault mirrors [vault.Vault] but stores plaintext through [vault.Storage] (no AES).
type TestVault struct {
	St vault.Storage
	Id vault.Identifier
}

// NewTestVault returns a plaintext-through-storage vault for tests.
func NewTestVault(tb testing.TB, st vault.Storage, id vault.Identifier) *TestVault {
	tb.Helper()
	if st == nil || id == nil {
		tb.Fatal("vaulttest: storage and identifier required")
	}
	return &TestVault{St: st, Id: id}
}

// Store mints an id and inserts plaintext with [vault.Storage.Create].
func (tv *TestVault) Store(ctx context.Context, namespace string, plain []byte) (string, error) {
	id, err := tv.Id.New()
	if err != nil {
		return "", err
	}
	if err := tv.St.Create(ctx, namespace, id, plain); err != nil {
		return "", err
	}
	return id, nil
}

// Update replaces stored plaintext with [vault.Storage.Replace].
func (tv *TestVault) Update(ctx context.Context, namespace, id string, newPlain []byte) error {
	if err := tv.Id.Parse(id); err != nil {
		return err
	}
	return tv.St.Replace(ctx, namespace, id, newPlain)
}

// AtomicUpdate verifies plaintext with [vault.Storage.Get] then swaps with [vault.Storage.CompareAndSwap].
func (tv *TestVault) AtomicUpdate(ctx context.Context, namespace, id string, currentPlain, newPlain []byte) error {
	if err := tv.Id.Parse(id); err != nil {
		return err
	}
	old, err := tv.St.Get(ctx, namespace, id)
	if err != nil {
		return err
	}
	if !bytes.Equal(old, currentPlain) {
		return vault.ErrWrongCurrent
	}
	return tv.St.CompareAndSwap(ctx, namespace, id, old, newPlain)
}

// Retrieve returns plaintext from [vault.Storage.Get].
func (tv *TestVault) Retrieve(ctx context.Context, namespace, id string) ([]byte, error) {
	if err := tv.Id.Parse(id); err != nil {
		return nil, err
	}
	return tv.St.Get(ctx, namespace, id)
}

// Delete removes the row with [vault.Storage.Delete].
func (tv *TestVault) Delete(ctx context.Context, namespace, id string) error {
	if err := tv.Id.Parse(id); err != nil {
		return err
	}
	return tv.St.Delete(ctx, namespace, id)
}
