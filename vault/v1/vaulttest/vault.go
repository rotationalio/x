/*
Package vaulttest provides a [TestVault] mock for testing without crypto, and
helpers for testing [storage.Storage] and [identifier.Identifier] implementations for contract conformance.

Storage checks ([StorageConforms] and [CheckStorageCreateGetRoundtrip], etc.) live in
storage.go. Identifier checks ([IdentifierConforms] and [CheckIdentifierNewManyDistinct], etc.)
live in identifier.go.
*/
package vaulttest

// Plaintext [TestVault] stores bytes through [storage.Storage] with no envelope crypto, implementing [v1.Vault].

import (
	"bytes"
	"context"
	"errors"
	"testing"

	v1 "go.rtnl.ai/x/vault/v1"
	verrors "go.rtnl.ai/x/vault/v1/errors"
	"go.rtnl.ai/x/vault/v1/identifier"
	"go.rtnl.ai/x/vault/v1/storage"
)

// TestVault stores plaintext through [storage.Storage] using [identifier.Identifier]; it implements [v1.Vault]
// for tests and as the inner [v1.Vault] for stringvault and jsonvault wrappers.
type TestVault struct {
	St storage.Storage
	Id identifier.Identifier
}

// TestVault implements [v1.Vault].
var _ v1.Vault = (*TestVault)(nil)

// NewTestVault returns a plaintext-through-storage vault for tests.
func NewTestVault(tb testing.TB, st storage.Storage, id identifier.Identifier) *TestVault {
	tb.Helper()
	if st == nil || id == nil {
		tb.Fatal("vaulttest: storage and identifier required")
	}
	return &TestVault{St: st, Id: id}
}

// Store mints an id and inserts plaintext with [storage.Storage.Create].
func (tv *TestVault) Store(ctx context.Context, namespace string, plain []byte) (string, error) {
	if tv == nil {
		return "", verrors.ErrNilVault
	}
	id, err := tv.Id.New()
	if err != nil {
		return "", err
	}
	if err := tv.St.Create(ctx, namespace, id, plain); err != nil {
		return "", err
	}
	return id, nil
}

// Update replaces stored plaintext with [storage.Storage.Replace].
func (tv *TestVault) Update(ctx context.Context, namespace, id string, newPlain []byte) error {
	if tv == nil {
		return verrors.ErrNilVault
	}
	if err := tv.Id.Parse(id); err != nil {
		return err
	}
	return tv.St.Replace(ctx, namespace, id, newPlain)
}

// CompareAndSwap replaces stored plaintext with newPlain only if it byte-equals currentPlain, using [storage.Storage.CompareAndSwap].
func (tv *TestVault) CompareAndSwap(ctx context.Context, namespace, id string, currentPlain, newPlain []byte) error {
	if tv == nil {
		return verrors.ErrNilVault
	}
	if err := tv.Id.Parse(id); err != nil {
		return err
	}

	old, err := tv.St.Get(ctx, namespace, id)
	if err != nil {
		return err
	}

	if !bytes.Equal(old, currentPlain) {
		return verrors.ErrWrongCurrent
	}

	return tv.St.CompareAndSwap(ctx, namespace, id, old, newPlain)
}

// Retrieve returns plaintext from [storage.Storage.Get].
func (tv *TestVault) Retrieve(ctx context.Context, namespace, id string) ([]byte, error) {
	if tv == nil {
		return nil, verrors.ErrNilVault
	}
	if err := tv.Id.Parse(id); err != nil {
		return nil, err
	}
	return tv.St.Get(ctx, namespace, id)
}

// MoveNamespace moves plaintext from fromNamespace to toNamespace.
func (tv *TestVault) MoveNamespace(ctx context.Context, fromNamespace, toNamespace, id string) error {
	if tv == nil {
		return verrors.ErrNilVault
	}
	if fromNamespace == toNamespace {
		return nil
	}
	if err := tv.Id.Parse(id); err != nil {
		return err
	}

	oldPlain, err := tv.St.Get(ctx, fromNamespace, id)
	if err != nil {
		return err
	}

	if err := tv.St.Create(ctx, toNamespace, id, oldPlain); err != nil {
		return err
	}

	// Delete source only after destination row exists (same ordering as [v1.Vault.MoveNamespace]).
	if err := tv.St.Delete(ctx, fromNamespace, id); err != nil {
		return errors.Join(verrors.ErrMoveNamespaceIncomplete, verrors.ErrStorage, err)
	}
	return nil
}

// Delete removes the row with [storage.Storage.Delete].
func (tv *TestVault) Delete(ctx context.Context, namespace, id string) error {
	if tv == nil {
		return verrors.ErrNilVault
	}
	if err := tv.Id.Parse(id); err != nil {
		return err
	}
	return tv.St.Delete(ctx, namespace, id)
}
