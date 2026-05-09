package vaulttest

// Storage conformance helpers for [vault.Storage] implementations ([Run]).

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault"
)

// mustNewID mints one string id via identifier.New(); callers reuse that string across all
// operations in a subtest that share one logical row key.
func mustNewID(t *testing.T, identifier vault.Identifier) string {
	t.Helper()
	s, err := identifier.New()
	assert.Ok(t, err)
	return s
}

// Run exercises Create, Get, Replace, Delete, and CompareAndSwap using ids minted by identifier.
// Pass the same [vault.Identifier] your production [vault.Vault] uses so Storage implementations
// that validate ids (e.g. after [vault.Identifier.Parse]) are exercised correctly.
//
// newStorage must return an empty [vault.Storage] for each subtest (parallel-safe isolation).
func Run(t *testing.T, identifier vault.Identifier, newStorage func(*testing.T) vault.Storage) {
	t.Helper()

	t.Run("create_get_roundtrip", func(t *testing.T) {
		// Setup: empty storage, opaque ciphertext blob, and one minted id from identifier.
		st := newStorage(t)
		ctx := context.Background()
		const ns = "ns"
		payload := []byte("cipher-a")
		id := mustNewID(t, identifier)

		// Exercise: persist one row under (ns, id).
		assert.Ok(t, st.Create(ctx, ns, id, payload))

		// Exercise: read same key back.
		got, err := st.Get(ctx, ns, id)
		assert.Ok(t, err)

		// Expect: ciphertext bytes match (Storage contract: opaque roundtrip equality).
		assert.Equal(t, payload, got)
	})

	t.Run("create_duplicate", func(t *testing.T) {
		// Setup: empty storage and one minted id.
		st := newStorage(t)
		ctx := context.Background()
		id := mustNewID(t, identifier)

		// Exercise: first Create succeeds for (n,id).
		assert.Ok(t, st.Create(ctx, "n", id, []byte("x")))

		// Exercise: second Create must fail — duplicate composite key (namespace,id).
		err := st.Create(ctx, "n", id, []byte("y"))

		// Expect: sentinel ErrDuplicateKey for callers discriminating duplicates.
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDuplicateKey)
	})

	t.Run("namespace_isolation", func(t *testing.T) {
		// Setup: same logical id string in two namespaces must be independent rows.
		st := newStorage(t)
		ctx := context.Background()
		sharedID := mustNewID(t, identifier)

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
		// Setup: no rows; id is minted but never inserted (absent key).
		st := newStorage(t)
		ctx := context.Background()
		id := mustNewID(t, identifier)

		// Exercise: Get absent key.
		_, err := st.Get(ctx, "n", id)

		// Expect: ErrNotFound (not silently empty slice).
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("replace_success", func(t *testing.T) {
		// Setup: existing row with v1 bytes.
		st := newStorage(t)
		ctx := context.Background()
		id := mustNewID(t, identifier)

		assert.Ok(t, st.Create(ctx, "n", id, []byte("v1")))

		// Exercise: Replace overwrites ciphertext for existing (n,id).
		assert.Ok(t, st.Replace(ctx, "n", id, []byte("v2")))

		// Expect: Get returns new blob only (no stale v1).
		got, err := st.Get(ctx, "n", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("v2"), got)
	})

	t.Run("replace_missing", func(t *testing.T) {
		// Setup: empty storage; id minted but row never created.
		st := newStorage(t)
		ctx := context.Background()
		id := mustNewID(t, identifier)

		// Exercise: Replace targets missing row.
		err := st.Replace(ctx, "n", id, []byte("z"))

		// Expect: ErrNotFound (cannot replace missing).
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("delete_idempotent", func(t *testing.T) {
		// Setup: guaranteed absent key without creating first (minted id never inserted).
		st := newStorage(t)
		ctx := context.Background()
		id := mustNewID(t, identifier)

		// Exercise: Delete when row does not exist.
		err := st.Delete(ctx, "n", id)

		// Expect: idempotent semantics — nil error.
		assert.Ok(t, err)
	})

	t.Run("delete_existing_then_get_missing", func(t *testing.T) {
		// Setup: persisted row under (n,id).
		st := newStorage(t)
		ctx := context.Background()
		id := mustNewID(t, identifier)

		assert.Ok(t, st.Create(ctx, "n", id, []byte("data")))

		// Exercise: remove that row explicitly.
		assert.Ok(t, st.Delete(ctx, "n", id))

		// Expect: reads fail with ErrNotFound (row truly gone).
		_, err := st.Get(ctx, "n", id)
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("cas_success_and_conflict", func(t *testing.T) {
		// Setup: one row holding "old".
		st := newStorage(t)
		ctx := context.Background()
		old := []byte("old")
		id := mustNewID(t, identifier)

		assert.Ok(t, st.Create(ctx, "n", id, old))

		newb := []byte("new")

		// Exercise: CAS when supplied old ciphertext matches stored value.
		assert.Ok(t, st.CompareAndSwap(ctx, "n", id, old, newb))

		// Expect: persisted value updated to "new".
		got, err := st.Get(ctx, "n", id)
		assert.Ok(t, err)
		assert.Equal(t, newb, got)

		// Exercise: CAS again using stale expectation (still "old" but storage has "new").
		err = st.CompareAndSwap(ctx, "n", id, old, []byte("x"))

		// Expect: compare fails with ErrCASFailed; value untouched.
		assert.ErrorIs(t, err, vault.ErrCASFailed)
		gotAfter, err := st.Get(ctx, "n", id)
		assert.Ok(t, err)
		assert.Equal(t, newb, gotAfter)
	})

	t.Run("cas_missing_row", func(t *testing.T) {
		// Setup: empty storage (no row for minted id).
		st := newStorage(t)
		ctx := context.Background()
		id := mustNewID(t, identifier)

		// Exercise: CAS with no backing row.
		err := st.CompareAndSwap(ctx, "n", id, []byte("a"), []byte("b"))

		// Expect: same as Get — ErrNotFound, not CASFailed (nothing to compare).
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})
}
