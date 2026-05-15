package vaulttest

// Storage conformance helpers for tests.
//
// These functions are not registered as Go tests by name: call [StorageConforms] or the individual
// [CheckStorage…] helpers from your *_test.go files. They are safe to ship in non-test sources.
//
// They encode the semantics the vault expects from [storage.Storage]: namespace-scoped rows, correct
// [verrors.ErrDuplicateKey] / [verrors.ErrNotFound] / [verrors.ErrCASFailed] sentinels, idempotent Delete on
// missing rows, and Compare-and-swap that compares full ciphertext blobs.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/vault/errors"
	"go.rtnl.ai/x/vault/identifier"
	"go.rtnl.ai/x/vault/storage"
)

// CheckStorageCreateGetRoundtrip verifies Create then Get returns the same ciphertext.
//
// Uses a fresh namespace string and a newly minted id so the row cannot collide with prior tests.
func CheckStorageCreateGetRoundtrip(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	const ns = "ns"
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	payload := []byte("cipher-a")
	if err := st.Create(ctx, ns, id, payload); err != nil {
		return fmt.Errorf("create: %w", err)
	}
	got, err := st.Get(ctx, ns, id)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	if !bytes.Equal(payload, got) {
		return fmt.Errorf("get: got %q want %q", got, payload)
	}
	return nil
}

// CheckStorageCreateDuplicate verifies a second Create with the same key returns [verrors.ErrDuplicateKey].
//
// The duplicate must surface as errors.Is(err, verrors.ErrDuplicateKey) so vault logic can classify it.
func CheckStorageCreateDuplicate(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	if err := st.Create(ctx, "n", id, []byte("x")); err != nil {
		return fmt.Errorf("first create: %w", err)
	}
	err = st.Create(ctx, "n", id, []byte("y"))
	if err == nil {
		return fmt.Errorf("second create: got nil want error")
	}
	if !errors.Is(err, verrors.ErrDuplicateKey) {
		return fmt.Errorf("second create: got %v want %w", err, verrors.ErrDuplicateKey)
	}
	return nil
}

// CheckStorageNamespaceIsolation verifies the same id in different namespaces holds distinct blobs.
//
// After creating "blob-a" and "blob-b" under the same logical id in two namespaces, reads must
// not cross namespaces. A second Create in the first namespace must still report ErrDuplicateKey.
func CheckStorageNamespaceIsolation(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	sharedID, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}

	// Same logical id in two namespaces must hold independent blobs.
	if err := st.Create(ctx, "ns-a", sharedID, []byte("blob-a")); err != nil {
		return fmt.Errorf("create ns-a: %w", err)
	}
	if err := st.Create(ctx, "ns-b", sharedID, []byte("blob-b")); err != nil {
		return fmt.Errorf("create ns-b: %w", err)
	}

	gotA, err := st.Get(ctx, "ns-a", sharedID)
	if err != nil {
		return fmt.Errorf("get ns-a: %w", err)
	}
	if !bytes.Equal([]byte("blob-a"), gotA) {
		return fmt.Errorf("get ns-a: got %q want blob-a", gotA)
	}

	gotB, err := st.Get(ctx, "ns-b", sharedID)
	if err != nil {
		return fmt.Errorf("get ns-b: %w", err)
	}
	if !bytes.Equal([]byte("blob-b"), gotB) {
		return fmt.Errorf("get ns-b: got %q want blob-b", gotB)
	}

	// Second create in the same namespace must still be rejected.
	err = st.Create(ctx, "ns-a", sharedID, []byte("clash"))
	if !errors.Is(err, verrors.ErrDuplicateKey) {
		return fmt.Errorf("duplicate create ns-a: got %v want %w", err, verrors.ErrDuplicateKey)
	}
	return nil
}

// CheckStorageGetMissing verifies Get on a missing row returns [verrors.ErrNotFound].
func CheckStorageGetMissing(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	_, err = st.Get(ctx, "n", id)
	if !errors.Is(err, verrors.ErrNotFound) {
		return fmt.Errorf("get missing: got %v want %w", err, verrors.ErrNotFound)
	}
	return nil
}

// CheckStorageReplaceSuccess verifies Replace updates ciphertext for an existing row.
func CheckStorageReplaceSuccess(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	if err := st.Create(ctx, "n", id, []byte("v1")); err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if err := st.Replace(ctx, "n", id, []byte("v2")); err != nil {
		return fmt.Errorf("replace: %w", err)
	}
	got, err := st.Get(ctx, "n", id)
	if err != nil {
		return fmt.Errorf("get after Replace: %w", err)
	}
	if !bytes.Equal([]byte("v2"), got) {
		return fmt.Errorf("get: got %q want v2", got)
	}
	return nil
}

// CheckStorageReplaceMissing verifies Replace on a missing row returns [verrors.ErrNotFound].
func CheckStorageReplaceMissing(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	err = st.Replace(ctx, "n", id, []byte("z"))
	if !errors.Is(err, verrors.ErrNotFound) {
		return fmt.Errorf("replace missing: got %v want %w", err, verrors.ErrNotFound)
	}
	return nil
}

// CheckStorageDeleteIdempotent verifies Delete on a missing row returns nil.
//
// Vault may delete rows that were never written; storage must not require a pre-existing row.
func CheckStorageDeleteIdempotent(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	if err := st.Delete(ctx, "n", id); err != nil {
		return fmt.Errorf("delete missing: %w", err)
	}
	return nil
}

// CheckStorageDeleteExistingThenGetMissing verifies Delete removes a row.
func CheckStorageDeleteExistingThenGetMissing(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	if err := st.Create(ctx, "n", id, []byte("data")); err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if err := st.Delete(ctx, "n", id); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	_, err = st.Get(ctx, "n", id)
	if !errors.Is(err, verrors.ErrNotFound) {
		return fmt.Errorf("get after Delete: got %v want %w", err, verrors.ErrNotFound)
	}
	return nil
}

// CheckStorageCompareAndSwapSuccessAndConflict verifies CAS updates when old matches and rejects wrong old.
//
// After a successful swap from "old" to "new", a second CAS with stale "old" must return
// [verrors.ErrCASFailed] and the stored blob must still be "new".
func CheckStorageCompareAndSwapSuccessAndConflict(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	old := []byte("old")
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	if err := st.Create(ctx, "n", id, old); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	newb := []byte("new")
	if err := st.CompareAndSwap(ctx, "n", id, old, newb); err != nil {
		return fmt.Errorf("cas: %w", err)
	}

	got, err := st.Get(ctx, "n", id)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	if !bytes.Equal(newb, got) {
		return fmt.Errorf("get after CAS: got %q want new", got)
	}

	// Wrong old ciphertext must not change the row.
	err = st.CompareAndSwap(ctx, "n", id, old, []byte("x"))
	if !errors.Is(err, verrors.ErrCASFailed) {
		return fmt.Errorf("cas wrong old: got %v want %w", err, verrors.ErrCASFailed)
	}

	gotAfter, err := st.Get(ctx, "n", id)
	if err != nil {
		return fmt.Errorf("get after failed CAS: %w", err)
	}
	if !bytes.Equal(newb, gotAfter) {
		return fmt.Errorf("get after failed CAS: got %q want new", gotAfter)
	}
	return nil
}

// CheckStorageCompareAndSwapMissingRow verifies CAS on a missing row returns [verrors.ErrNotFound].
//
// A missing row is not the same as a CAS conflict; callers distinguish via errors.Is.
func CheckStorageCompareAndSwapMissingRow(ctx context.Context, st storage.Storage, idGen identifier.Identifier) error {
	id, err := idGen.New()
	if err != nil {
		return fmt.Errorf("idGen.New: %w", err)
	}
	err = st.CompareAndSwap(ctx, "n", id, []byte("a"), []byte("b"))
	if !errors.Is(err, verrors.ErrNotFound) {
		return fmt.Errorf("cas missing: got %v want %w", err, verrors.ErrNotFound)
	}
	return nil
}

// StorageConforms runs subtests that verify newStorage(t) implements [storage.Storage] semantics, using
// ids minted by idGen. It is not a Go test function; call it from your own Test_* in *_test.go.
//
// newStorage must return an isolated backend for each invocation (typically a new in-memory map) so
// subtests do not share mutable state.
func StorageConforms(t *testing.T, idGen identifier.Identifier, newStorage func(*testing.T) storage.Storage) {
	t.Helper()
	ctx := context.Background()

	t.Run("create_get_roundtrip", func(t *testing.T) {
		assert.Ok(t, CheckStorageCreateGetRoundtrip(ctx, newStorage(t), idGen))
	})

	t.Run("create_duplicate", func(t *testing.T) {
		assert.Ok(t, CheckStorageCreateDuplicate(ctx, newStorage(t), idGen))
	})

	t.Run("namespace_isolation", func(t *testing.T) {
		assert.Ok(t, CheckStorageNamespaceIsolation(ctx, newStorage(t), idGen))
	})

	t.Run("get_missing", func(t *testing.T) {
		assert.Ok(t, CheckStorageGetMissing(ctx, newStorage(t), idGen))
	})

	t.Run("replace_success", func(t *testing.T) {
		assert.Ok(t, CheckStorageReplaceSuccess(ctx, newStorage(t), idGen))
	})

	t.Run("replace_missing", func(t *testing.T) {
		assert.Ok(t, CheckStorageReplaceMissing(ctx, newStorage(t), idGen))
	})

	t.Run("delete_idempotent", func(t *testing.T) {
		assert.Ok(t, CheckStorageDeleteIdempotent(ctx, newStorage(t), idGen))
	})

	t.Run("delete_existing_then_get_missing", func(t *testing.T) {
		assert.Ok(t, CheckStorageDeleteExistingThenGetMissing(ctx, newStorage(t), idGen))
	})

	t.Run("cas_success_and_conflict", func(t *testing.T) {
		assert.Ok(t, CheckStorageCompareAndSwapSuccessAndConflict(ctx, newStorage(t), idGen))
	})

	t.Run("cas_missing_row", func(t *testing.T) {
		assert.Ok(t, CheckStorageCompareAndSwapMissingRow(ctx, newStorage(t), idGen))
	})
}
