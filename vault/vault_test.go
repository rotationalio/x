// Package vault_test contains external, black-box tests for go.rtnl.ai/x/vault.
// These tests thoroughly cover:
//
//   - Argument and key validation for Vault construction and usage.
//   - Vault API methods in logical order (see vault.go), including correct binding of
//     namespaces as GCM Additional Authenticated Data (AAD).
//   - Handling and propagation of identifier and storage errors.
//   - Cryptographic behaviors and edge cases in GCM sealing/opening, via exported
//     test-only helpers (vault.ExportGCM*) from export_test.go.
//
// File organization:
// - vault_test.go: End-to-end API surface tests, helper functions, and test doubles (bottom).
// - gcm_test.go: Cryptographic primitive tests (seal/open/newAEAD).
package vault_test

import (
	"context"
	"errors"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault"
	"go.rtnl.ai/x/vault/vaulttest"
)

//=============================================================================
// New
//=============================================================================

// TestNew checks vault.New for argument validation and AES-256 key length.
func TestNew(t *testing.T) {
	// Setup: baseline 32-byte key, mem storage, and hex identifier for rows that need them.
	key := testKey()
	st := vaulttest.NewMemStorage()
	id := vaulttest.HexIdentifier{}

	tests := []struct {
		name    string
		key     []byte
		st      vault.Storage
		id      vault.Identifier
		wantErr error
	}{
		{name: "nil_storage", key: key, st: nil, id: id, wantErr: vault.ErrInvalidNewArgs},
		{name: "nil_identifier", key: key, st: st, id: nil, wantErr: vault.ErrInvalidNewArgs},
		{name: "key_16", key: make([]byte, 16), st: st, id: id, wantErr: vault.ErrInvalidKeyLength},
		{name: "key_24", key: make([]byte, 24), st: st, id: id, wantErr: vault.ErrInvalidKeyLength},
		{name: "key_31", key: make([]byte, 31), st: st, id: id, wantErr: vault.ErrInvalidKeyLength},
		{name: "key_32_ok", key: key, st: st, id: id},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Exercise: construct vault from table row.
			v, err := vault.New(tt.key, tt.st, tt.id)

			if tt.wantErr != nil {
				// Expect: error chain includes wantErr; no vault on failure.
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, v)
				return
			}

			// Expect: vault constructed.
			assert.Ok(t, err)
			assert.NotNil(t, v)
		})
	}
}

//=============================================================================
// Store
//=============================================================================

// TestVault_Store checks Vault.Store: basic usage, error on id mint, and duplicate id.
func TestVault_Store(t *testing.T) {
	ctx := context.Background()
	key := testKey()

	t.Run("happy", func(t *testing.T) {
		// Setup: standard vault.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise: Store seals plaintext and Create's a row.
		id, err := v.Store(ctx, "ns", []byte("plain"))
		assert.Ok(t, err)

		// Expect: identifier produced a non-empty id.
		assert.True(t, len(id) > 0)
	})

	t.Run("identifier_New_fails", func(t *testing.T) {
		// Setup: identifier that cannot mint ids.
		v, err := vault.New(key, vaulttest.NewMemStorage(), errNewIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect: Store surfaces ErrInvalidIdentifier.
		_, err = v.Store(ctx, "ns", []byte("x"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrInvalidIdentifier)
	})

	t.Run("duplicate_id_second_store", func(t *testing.T) {
		// Setup: identifier always returns the same id.
		v, err := vault.New(key, vaulttest.NewMemStorage(), dupNewIdentifier{})
		assert.Ok(t, err)

		// Exercise: first Store succeeds (Create).
		_, err = v.Store(ctx, "ns", []byte("first"))
		assert.Ok(t, err)

		// Exercise / expect: second Store hits duplicate key in storage.
		_, err = v.Store(ctx, "ns", []byte("second"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrStorage)
		assert.ErrorIs(t, err, vault.ErrDuplicateKey)
	})
}

//=============================================================================
// Update
//=============================================================================

// TestVault_Update checks Vault.Update for normal update, invalid id, and missing record.
func TestVault_Update(t *testing.T) {
	ctx := context.Background()
	key := testKey()

	t.Run("happy", func(t *testing.T) {
		// Setup: vault and initial ciphertext for v1.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "ns", []byte("v1"))
		assert.Ok(t, err)

		// Exercise: blind Replace with re-sealed v2.
		err = v.Update(ctx, "ns", id, []byte("v2"))
		assert.Ok(t, err)

		// Expect: decrypt shows v2.
		got, err := v.Retrieve(ctx, "ns", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("v2"), got)
	})

	t.Run("invalid_id", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect: Parse fails before storage.
		err = v.Update(ctx, "ns", "bad-id", []byte("z"))
		assert.ErrorIs(t, err, vault.ErrInvalidIdentifier)
	})

	t.Run("missing_row", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect: Replace on missing row → ErrNotFound via ErrStorage.
		err = v.Update(ctx, "ns", "0123456789abcdef0123456789abcdef", []byte("z"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrStorage)
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})
}

//=============================================================================
// AtomicUpdate
//=============================================================================

// TestVault_AtomicUpdate covers CAS-based update logic and error conditions.
func TestVault_AtomicUpdate(t *testing.T) {
	ctx := context.Background()
	key := testKey()

	t.Run("happy", func(t *testing.T) {
		// Setup: secret with plaintext "alpha".
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "ns", []byte("alpha"))
		assert.Ok(t, err)

		// Exercise: current matches decrypted row; swap to "beta" via CAS path.
		err = v.AtomicUpdate(ctx, "ns", id, []byte("alpha"), []byte("beta"))
		assert.Ok(t, err)

		// Expect.
		got, err := v.Retrieve(ctx, "ns", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("beta"), got)
	})

	t.Run("wrong_current_plaintext", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "ns", []byte("stored"))
		assert.Ok(t, err)

		// Exercise / expect: mismatch before CAS → ErrWrongCurrent.
		err = v.AtomicUpdate(ctx, "ns", id, []byte("not-stored"), []byte("new"))
		assert.ErrorIs(t, err, vault.ErrWrongCurrent)
	})

	t.Run("invalid_id", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect.
		err = v.AtomicUpdate(ctx, "ns", "bad", []byte("a"), []byte("b"))
		assert.ErrorIs(t, err, vault.ErrInvalidIdentifier)
	})

	t.Run("missing_row", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect: Get misses before open.
		err = v.AtomicUpdate(ctx, "ns", "0123456789abcdef0123456789abcdef", []byte("a"), []byte("b"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrStorage)
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("decrypt_fails_corrupt_ciphertext", func(t *testing.T) {
		// Setup: vault backed by shared MemStorage; store a valid blob.
		st := vaulttest.NewMemStorage()
		v, err := vault.New(key, st, vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "good-ns", []byte("payload"))
		assert.Ok(t, err)

		// Arrange: replace ciphertext with junk (still same row key).
		st.SetTestBlob("good-ns", id, []byte{9, 9, 9})

		// Exercise / expect: open fails → ErrDecrypt before compare.
		err = v.AtomicUpdate(ctx, "good-ns", id, []byte("payload"), []byte("x"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDecrypt)
	})

	t.Run("CAS_lost", func(t *testing.T) {
		// Setup: storage wrapper that always rejects CompareAndSwap.
		base := vaulttest.NewMemStorage()
		st := &casFailStorage{MemStorage: base}
		v, err := vault.New(key, st, vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "ns", []byte("cur"))
		assert.Ok(t, err)

		// Exercise / expect: plaintext step ok, CAS returns ErrCASFailed.
		err = v.AtomicUpdate(ctx, "ns", id, []byte("cur"), []byte("next"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrStorage)
		assert.ErrorIs(t, err, vault.ErrCASFailed)
	})
}

//=============================================================================
// Retrieve
//=============================================================================

// TestVault_Retrieve checks roundtrip, bad namespace, missing row, id, and corrupt data.
func TestVault_Retrieve(t *testing.T) {
	ctx := context.Background()
	key := testKey()

	t.Run("happy", func(t *testing.T) {
		// Create vault with MemStorage and HexIdentifier.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Store secret payload and get generated id.
		want := []byte("secret-bytes")
		id, err := v.Store(ctx, "ns-a", want)
		assert.Ok(t, err)

		// Retrieve payload by id and namespace.
		got, err := v.Retrieve(ctx, "ns-a", id)
		assert.Ok(t, err)

		// Check retrieved payload matches what was stored.
		assert.Equal(t, want, got)
	})

	t.Run("wrong_namespace_AAD", func(t *testing.T) {
		// Create vault and back it by MemStorage.
		st := vaulttest.NewMemStorage()
		v, err := vault.New(key, st, vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Store payload under one namespace.
		id, err := v.Store(ctx, "ns-sealed", []byte("data"))
		assert.Ok(t, err)

		// Get ciphertext blob from storage.
		blob, err := st.Get(ctx, "ns-sealed", id)
		assert.Ok(t, err)

		// Copy to different namespace (wrong AAD).
		assert.Ok(t, st.Create(ctx, "ns-other", id, blob))

		// Attempt retrieval with wrong namespace; expect auth failure.
		_, err = v.Retrieve(ctx, "ns-other", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDecrypt)
	})

	t.Run("missing_row", func(t *testing.T) {
		// Create vault with standard test backing.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Attempt retrieve using a valid id that doesn't exist in storage.
		_, err = v.Retrieve(ctx, "ns", "0123456789abcdef0123456789abcdef")
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrStorage)
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("invalid_id", func(t *testing.T) {
		// Create vault for test.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Attempt retrieve using an invalid id string.
		_, err = v.Retrieve(ctx, "ns", "not-a-valid-hex-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrInvalidIdentifier)
	})

	t.Run("truncated_ciphertext", func(t *testing.T) {
		// Create vault and get storage handle.
		st := vaulttest.NewMemStorage()
		v, err := vault.New(key, st, vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Store a valid secret to get a real id.
		id, err := v.Store(ctx, "ns", []byte("ok"))
		assert.Ok(t, err)

		// Overwrite ciphertext with too-short/corrupt blob.
		st.SetTestBlob("ns", id, []byte{1, 2, 3})

		// Attempt retrieve; expect decryption and length error.
		_, err = v.Retrieve(ctx, "ns", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDecrypt)
		assert.ErrorIs(t, err, vault.ErrCiphertextTooShort)
	})
}

//=============================================================================
// MoveNamespace
//=============================================================================

// TestVault_MoveNamespace exercises Vault.MoveNamespace for noop, move, errors.
func TestVault_MoveNamespace(t *testing.T) {
	ctx := context.Background()
	key := testKey()

	t.Run("noop_equal_namespaces", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "same", []byte("x"))
		assert.Ok(t, err)

		// Exercise: from == to is a fast path (no reloc).
		err = v.MoveNamespace(ctx, "same", "same", id)
		assert.Ok(t, err)

		// Expect: row unchanged.
		got, err := v.Retrieve(ctx, "same", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("x"), got)
	})

	t.Run("happy", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "from", []byte("payload"))
		assert.Ok(t, err)

		// Exercise: decrypt→reseal→Create(to)→Delete(from).
		err = v.MoveNamespace(ctx, "from", "to", id)
		assert.Ok(t, err)

		// Expect: readable under "to", gone from "from".
		got, err := v.Retrieve(ctx, "to", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("payload"), got)

		_, err = v.Retrieve(ctx, "from", id)
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("invalid_id", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect.
		err = v.MoveNamespace(ctx, "a", "b", "bad-id")
		assert.ErrorIs(t, err, vault.ErrInvalidIdentifier)
	})

	t.Run("missing_source", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect: Get(from) empty.
		err = v.MoveNamespace(ctx, "from", "to", "0123456789abcdef0123456789abcdef")
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrStorage)
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("decrypt_source_fails", func(t *testing.T) {
		// Setup: corrupt source blob after Store.
		st := vaulttest.NewMemStorage()
		v, err := vault.New(key, st, vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "from", []byte("ok"))
		assert.Ok(t, err)

		st.SetTestBlob("from", id, []byte{1, 2, 3})

		// Exercise / expect: open(from) fails before Create(to).
		err = v.MoveNamespace(ctx, "from", "to", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDecrypt)
	})

	t.Run("duplicate_destination", func(t *testing.T) {
		// Setup: destination slot pre-occupied.
		st := vaulttest.NewMemStorage()
		v, err := vault.New(key, st, vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "from", []byte("a"))
		assert.Ok(t, err)

		assert.Ok(t, st.Create(ctx, "to", id, []byte("occupies")))

		// Exercise / expect: Create(to) returns duplicate.
		err = v.MoveNamespace(ctx, "from", "to", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrStorage)
		assert.ErrorIs(t, err, vault.ErrDuplicateKey)
	})

	t.Run("incomplete_after_create_delete_fails", func(t *testing.T) {
		// Setup: Delete on "from" fails after successful Create on "to".
		base := vaulttest.NewMemStorage()
		st := &deleteFailsOnNs{MemStorage: base, ns: "from"}
		v, err := vault.New(key, st, vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "from", []byte("data"))
		assert.Ok(t, err)

		// Exercise / expect: ErrMoveNamespaceIncomplete + both rows may exist.
		err = v.MoveNamespace(ctx, "from", "to", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrMoveNamespaceIncomplete)
		assert.ErrorIs(t, err, vault.ErrStorage)

		_, err = v.Retrieve(ctx, "to", id)
		assert.Ok(t, err)

		_, err = v.Retrieve(ctx, "from", id)
		assert.Ok(t, err)
	})
}

//=============================================================================
// Delete
//=============================================================================

// TestVault_Delete checks Delete on present and missing rows, invalid id, and backend error.
func TestVault_Delete(t *testing.T) {
	ctx := context.Background()
	key := testKey()

	t.Run("happy", func(t *testing.T) {
		// Setup: one row.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "ns", []byte("x"))
		assert.Ok(t, err)

		// Exercise: delete row.
		err = v.Delete(ctx, "ns", id)
		assert.Ok(t, err)

		// Expect: subsequent read is not found.
		_, err = v.Retrieve(ctx, "ns", id)
		assert.ErrorIs(t, err, vault.ErrNotFound)
	})

	t.Run("idempotent_missing", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect: missing row still returns nil from Delete.
		err = v.Delete(ctx, "ns", "0123456789abcdef0123456789abcdef")
		assert.Ok(t, err)
	})

	t.Run("invalid_id", func(t *testing.T) {
		// Setup.
		v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		// Exercise / expect.
		err = v.Delete(ctx, "ns", "bad-id")
		assert.ErrorIs(t, err, vault.ErrInvalidIdentifier)
	})

	t.Run("delete_storage_error", func(t *testing.T) {
		// Setup: storage that errors on Delete for "ns".
		base := vaulttest.NewMemStorage()
		st := &deleteFailsOnNs{MemStorage: base, ns: "ns"}
		v, err := vault.New(key, st, vaulttest.HexIdentifier{})
		assert.Ok(t, err)

		id, err := v.Store(ctx, "ns", []byte("x"))
		assert.Ok(t, err)

		// Exercise / expect: error wrapped in ErrStorage.
		err = v.Delete(ctx, "ns", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrStorage)
	})
}

//=============================================================================
// Shared test helpers and stubs
//=============================================================================

// testKey returns a deterministic 32-byte AES-256 key for test use.
func testKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}

// errNewIdentifier causes Identifier.New to always fail.
type errNewIdentifier struct{ vaulttest.HexIdentifier }

func (errNewIdentifier) New() (string, error) {
	return "", errors.New("identifier mint failed")
}

// dupNewIdentifier always returns the same id for collision testing.
type dupNewIdentifier struct{ vaulttest.HexIdentifier }

func (dupNewIdentifier) New() (string, error) {
	return "0123456789abcdef0123456789abcdef", nil
}

// casFailStorage always returns ErrCASFailed from CompareAndSwap.
type casFailStorage struct{ *vaulttest.MemStorage }

func (*casFailStorage) CompareAndSwap(ctx context.Context, namespace, id string, oldCipher, newCipher []byte) error {
	_ = ctx
	_ = namespace
	_ = id
	_ = oldCipher
	_ = newCipher
	return vault.ErrCASFailed
}

// deleteFailsOnNs fails Delete for a specific namespace.
type deleteFailsOnNs struct {
	*vaulttest.MemStorage
	ns string
}

func (s *deleteFailsOnNs) Delete(ctx context.Context, namespace, id string) error {
	if namespace == s.ns {
		return errors.New("simulated delete failure")
	}
	return s.MemStorage.Delete(ctx, namespace, id)
}
