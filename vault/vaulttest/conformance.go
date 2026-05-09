package vaulttest

// Storage conformance helpers for [vault.Storage] implementations ([Run]).

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault"
)

// Run exercises Create, Get, Replace, Delete, and CompareAndSwap. newStorage must return
// an empty [vault.Storage] for each subtest (parallel-safe isolation).
func Run(t *testing.T, newStorage func(*testing.T) vault.Storage) {
	t.Helper()

	t.Run("create_get_roundtrip", func(t *testing.T) {
		// Setup: empty storage and opaque ciphertext blob.
		st := newStorage(t)
		ctx := context.Background()
		const ns = "ns"
		payload := []byte("cipher-a")

		// Exercise: persist one row under (ns, id1).
		assert.Ok(t, st.Create(ctx, ns, "id1", payload))

		// Exercise: read same key back.
		got, err := st.Get(ctx, ns, "id1")
		assert.Ok(t, err)

		// Expect: ciphertext bytes match (Storage contract: opaque roundtrip equality).
		assert.Equal(t, payload, got)
	})

	t.Run("create_duplicate", func(t *testing.T) {
		// Setup: empty storage.
		st := newStorage(t)
		ctx := context.Background()

		// Exercise: first Create succeeds for (n,id).
		assert.Ok(t, st.Create(ctx, "n", "id", []byte("x")))

		// Exercise: second Create must fail — duplicate composite key (namespace,id).
		err := st.Create(ctx, "n", "id", []byte("y"))

		// Expect: sentinel ErrDuplicateKey for callers discriminating duplicates.
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDuplicateKey)
	})

	t.Run("namespace_isolation", func(t *testing.T) {
		// Setup: same logical id string in two namespaces must be independent rows.
		st := newStorage(t)
		ctx := context.Background()
		const sharedID = "same-id"

		// Exercise: two creates differing only by namespace.
		assert.Ok(t, st.Create(ctx, "ns-a", sharedID, []byte("blob-a")))
		assert.Ok(t, st.Create(ctx, "ns-b", sharedID, []byte("blob-b")))

		// Exercise / expect: each namespace returns its own ciphertext.
		gotA, err := st.Get(ctx, "ns-a", sharedID)
		assert.Ok(t, err)
		assert.Equal(t, []byte("blob-a"), gotA)

		gotB, err := st.Get(ctx, "ns-b", sharedID)
		assert.Ok(t, err)
		assert.Equal(t, []byte("blob-b"), gotB)

		// Expect: repeating Create inside one namespace still fails (row already exists).
		err = st.Create(ctx, "ns-a", sharedID, []byte("clash"))
		assert.ErrorIs(t, err, vault.ErrDuplicateKey)
	})

	t.Run("get_missing", func(t *testing.T) {
		// Setup: no rows.
		st := newStorage(t)
		ctx := context.Background()

		// Exercise: Get absent key.
		_, err := st.Get(ctx, "n", "missing")

		// Expect: ErrNotFound (not silently empty slice).
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("replace_success", func(t *testing.T) {
		// Setup: existing row with v1 bytes.
		st := newStorage(t)
		ctx := context.Background()

		assert.Ok(t, st.Create(ctx, "n", "id", []byte("v1")))

		// Exercise: Replace overwrites ciphertext for existing (n,id).
		assert.Ok(t, st.Replace(ctx, "n", "id", []byte("v2")))

		// Expect: Get returns new blob only (no stale v1).
		got, err := st.Get(ctx, "n", "id")
		assert.Ok(t, err)
		assert.Equal(t, []byte("v2"), got)
	})

	t.Run("replace_missing", func(t *testing.T) {
		// Setup: empty storage.
		st := newStorage(t)
		ctx := context.Background()

		// Exercise: Replace targets missing row.
		err := st.Replace(ctx, "n", "id", []byte("z"))

		// Expect: ErrNotFound (cannot replace missing).
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("delete_idempotent", func(t *testing.T) {
		// Setup: guaranteed absent key without creating first.
		st := newStorage(t)
		ctx := context.Background()

		// Exercise: Delete when row does not exist.
		err := st.Delete(ctx, "n", "gone")

		// Expect: idempotent semantics — nil error.
		assert.Ok(t, err)
	})

	t.Run("delete_existing_then_get_missing", func(t *testing.T) {
		// Setup: persisted row under (n,id).
		st := newStorage(t)
		ctx := context.Background()

		assert.Ok(t, st.Create(ctx, "n", "id", []byte("data")))

		// Exercise: remove that row explicitly.
		assert.Ok(t, st.Delete(ctx, "n", "id"))

		// Expect: reads fail with ErrNotFound (row truly gone).
		_, err := st.Get(ctx, "n", "id")
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("cas_success_and_conflict", func(t *testing.T) {
		// Setup: one row holding "old".
		st := newStorage(t)
		ctx := context.Background()
		old := []byte("old")

		assert.Ok(t, st.Create(ctx, "n", "k", old))

		newb := []byte("new")

		// Exercise: CAS when supplied old ciphertext matches stored value.
		assert.Ok(t, st.CompareAndSwap(ctx, "n", "k", old, newb))

		// Expect: persisted value updated to "new".
		got, err := st.Get(ctx, "n", "k")
		assert.Ok(t, err)
		assert.Equal(t, newb, got)

		// Exercise: CAS again using stale expectation (still "old" but storage has "new").
		err = st.CompareAndSwap(ctx, "n", "k", old, []byte("x"))

		// Expect: compare fails with ErrCASFailed; value untouched.
		assert.ErrorIs(t, err, vault.ErrCASFailed)
		gotAfter, err := st.Get(ctx, "n", "k")
		assert.Ok(t, err)
		assert.Equal(t, newb, gotAfter)
	})

	t.Run("cas_missing_row", func(t *testing.T) {
		// Setup: empty storage (no (n,k) row).
		st := newStorage(t)
		ctx := context.Background()

		// Exercise: CAS with no backing row.
		err := st.CompareAndSwap(ctx, "n", "k", []byte("a"), []byte("b"))

		// Expect: same as Get — ErrNotFound, not CASFailed (nothing to compare).
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})
}
