package v1_test

// Tests for [v1.New] and [v1.Vault].

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"go.rtnl.ai/x/assert"
	v1 "go.rtnl.ai/x/vault/v1"
	verrors "go.rtnl.ai/x/vault/v1/errors"
	"go.rtnl.ai/x/vault/v1/identifier"
	"go.rtnl.ai/x/vault/v1/models"
	"go.rtnl.ai/x/vault/v1/storage"
)

//=============================================================================
// Tests: New
//=============================================================================

// TestNew_nilStorage verifies New rejects nil storage.
func TestNew_nilStorage(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	_, err = v1.New(priv, nil, identifier.HexIdentifier{})
	assert.ErrorIs(t, err, verrors.ErrInvalidNewArgs)
}

// TestNew_nilIdentifier verifies New rejects a nil Identifier.
func TestNew_nilIdentifier(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	_, err = v1.New(priv, storage.NewMemStorage(), nil)
	assert.ErrorIs(t, err, verrors.ErrInvalidNewArgs)
}

// TestNew_nilPrivateKey verifies New rejects a nil wrapping key.
func TestNew_nilPrivateKey(t *testing.T) {
	_, err := v1.New(nil, storage.NewMemStorage(), identifier.HexIdentifier{})
	assert.ErrorIs(t, err, verrors.ErrNilPrivateKey)
}

// TestNew_ok verifies a minimal valid New succeeds.
func TestNew_ok(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	v, err := v1.New(priv, storage.NewMemStorage(), identifier.HexIdentifier{})
	assert.Ok(t, err)
	assert.NotNil(t, v)
}

//=============================================================================
// Tests: Vault (envelope)
//=============================================================================

// TestVault_envelope_store_retrieve exercises a minimal Store then Retrieve on a real
// [v1.Vault] with in-memory storage and hex ids.
func TestVault_envelope_store_retrieve(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	v, err := v1.New(priv, storage.NewMemStorage(), identifier.HexIdentifier{})
	assert.Ok(t, err)
	ctx := context.Background()

	// End-to-end envelope path: store ciphertext, retrieve decrypts to same bytes.
	id, err := v.Store(ctx, "ns1", []byte("payload"))
	assert.Ok(t, err)
	got, err := v.Retrieve(ctx, "ns1", id)
	assert.Ok(t, err)
	assert.Equal(t, []byte("payload"), got)
}

// TestVault_envelope_wrong_namespace ensures ciphertext copied to another namespace key
// fails open with [verrors.ErrNamespaceMismatch].
func TestVault_envelope_wrong_namespace(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	st := storage.NewMemStorage()
	v, err := v1.New(priv, st, identifier.HexIdentifier{})
	assert.Ok(t, err)
	ctx := context.Background()

	id, err := v.Store(ctx, "ns1", []byte("x"))
	assert.Ok(t, err)
	blob, err := st.Get(ctx, "ns1", id)
	assert.Ok(t, err)

	// Same ciphertext under another namespace key must fail namespace binding.
	assert.Ok(t, st.Create(ctx, "ns2", id, blob))

	_, err = v.Retrieve(ctx, "ns2", id)
	assert.ErrorIs(t, err, verrors.ErrNamespaceMismatch)
}

// TestVault_Store_sealEntropyFailure verifies [v1.Vault.Store] maps [crypto/rand.Reader] failures
// during envelope sealing (DEK, inner nonce, ephemeral key, or wrap nonce reads) to [verrors.ErrSealFailed].
func TestVault_Store_sealEntropyFailure(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	v, err := v1.New(priv, storage.NewMemStorage(), dupNewIdentifier{})
	assert.Ok(t, err)
	ctx := context.Background()

	// dupNewIdentifier avoids reading [rand.Reader] during id mint so the patched reader fails inside seal only.
	orig := rand.Reader
	t.Cleanup(func() { rand.Reader = orig })
	rand.Reader = vaultEOFReader{}

	_, err = v.Store(ctx, "ns", []byte("payload"))
	assert.ErrorIs(t, err, verrors.ErrSealFailed)
}

// TestVault_Retrieve_badEphemeralPubKey ensures garbage [models.DekEnvelope.Pub] bytes so
// [ecdh.X25519.NewPublicKey] (or subsequent ECDH) fails open with [verrors.ErrDecrypt].
func TestVault_Retrieve_badEphemeralPubKey(t *testing.T) {
	ctx := context.Background()
	st := storage.NewMemStorage()
	v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
	id, err := v.Store(ctx, "ns", []byte("secret"))
	assert.Ok(t, err)
	wire, err := st.Get(ctx, "ns", id)
	assert.Ok(t, err)

	var msg models.Sealed
	assert.Ok(t, msg.UnmarshalBinary(wire))
	for i := range msg.Dek.Pub {
		msg.Dek.Pub[i] = 0xff
	}
	bad, err := msg.MarshalBinary()
	assert.Ok(t, err)
	assert.Ok(t, st.Replace(ctx, "ns", id, bad))

	_, err = v.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestVault_retrieve_wrongLongTermKey ensures ciphertext sealed under one X25519 wrapping key
// cannot be decrypted by a [v1.Vault] built from a different long-term key ([verrors.ErrDecrypt]).
func TestVault_retrieve_wrongLongTermKey(t *testing.T) {
	ctx := context.Background()
	st := storage.NewMemStorage()
	privAlice, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	privBob, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	vAlice, err := v1.New(privAlice, st, identifier.HexIdentifier{})
	assert.Ok(t, err)
	id, err := vAlice.Store(ctx, "ns", []byte("secret"))
	assert.Ok(t, err)

	vBob, err := v1.New(privBob, st, identifier.HexIdentifier{})
	assert.Ok(t, err)
	_, err = vBob.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestVault_nilReceiver_contract asserts every [v1.Vault] method on a nil [*sealedVault] returns [verrors.ErrNilVault].
func TestVault_nilReceiver_contract(t *testing.T) {
	ctx := context.Background()
	nv := v1.NilSealedVault
	const id = "0123456789abcdef0123456789abcdef"

	_, err := nv.Store(ctx, "ns", []byte("x"))
	assert.ErrorIs(t, err, verrors.ErrNilVault)

	_, err = nv.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, verrors.ErrNilVault)

	assert.ErrorIs(t, nv.Update(ctx, "ns", id, []byte("z")), verrors.ErrNilVault)

	assert.ErrorIs(t, nv.CompareAndSwap(ctx, "ns", id, []byte("a"), []byte("b")), verrors.ErrNilVault)

	assert.ErrorIs(t, nv.MoveNamespace(ctx, "from", "to", id), verrors.ErrNilVault)

	assert.ErrorIs(t, nv.Delete(ctx, "ns", id), verrors.ErrNilVault)
}

// TestVault_Store covers Store success, identifier errors, and duplicate ids.
func TestVault_Store(t *testing.T) {
	ctx := context.Background()

	t.Run("happy", func(t *testing.T) {

		// Normal insert: minted id is non-empty and ciphertext lands in storage.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns", []byte("payload"))
		assert.Ok(t, err)
		assert.True(t, len(id) > 0)
	})

	t.Run("identifier_New_fails", func(t *testing.T) {

		// Store must surface identifier mint errors joined with ErrInvalidIdentifier.
		v := testEnvelopeVault(t, storage.NewMemStorage(), errNewIdentifier{})
		_, err := v.Store(ctx, "ns", []byte("x"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrInvalidIdentifier)
	})

	t.Run("duplicate_id_second_store", func(t *testing.T) {

		// Fixed id from Identifier.New makes the second Create hit ErrDuplicateKey.
		v := testEnvelopeVault(t, storage.NewMemStorage(), dupNewIdentifier{})
		_, err := v.Store(ctx, "ns", []byte("first"))
		assert.Ok(t, err)
		_, err = v.Store(ctx, "ns", []byte("second"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrStorage)
		assert.ErrorIs(t, err, verrors.ErrDuplicateKey)
	})
}

// TestVault_Retrieve covers happy path, namespace mismatch, missing rows, and corrupt wire.
func TestVault_Retrieve(t *testing.T) {
	ctx := context.Background()

	t.Run("happy", func(t *testing.T) {

		// Round-trip seal and open under the same namespace.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		want := []byte("secret-bytes")
		id, err := v.Store(ctx, "ns-a", want)
		assert.Ok(t, err)
		got, err := v.Retrieve(ctx, "ns-a", id)
		assert.Ok(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("wrong_namespace_metadata", func(t *testing.T) {

		// Same ciphertext blob copied to another namespace key must fail AAD/namespacing checks.
		st := storage.NewMemStorage()
		v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns-sealed", []byte("data"))
		assert.Ok(t, err)
		blob, err := st.Get(ctx, "ns-sealed", id)
		assert.Ok(t, err)
		assert.Ok(t, st.Create(ctx, "ns-other", id, blob))
		_, err = v.Retrieve(ctx, "ns-other", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrNamespaceMismatch)
	})

	t.Run("missing_row", func(t *testing.T) {

		// Retrieve on unknown id yields ErrNotFound from storage.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		_, err := v.Retrieve(ctx, "ns", "0123456789abcdef0123456789abcdef")
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrStorage)
		assert.ErrorIs(t, err, verrors.ErrNotFound)
	})

	t.Run("invalid_id", func(t *testing.T) {

		// Parse rejects non-canonical id strings.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		_, err := v.Retrieve(ctx, "ns", "not-a-valid-hex-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrInvalidIdentifier)
	})

	t.Run("truncated_ciphertext", func(t *testing.T) {

		// Too-short blob cannot carry a valid nonce+tag layout.
		st := storage.NewMemStorage()
		v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns", []byte("ok"))
		assert.Ok(t, err)
		st.BypassSemanticsSetBlobForTest("ns", id, []byte{1, 2, 3})
		_, err = v.Retrieve(ctx, "ns", id)
		assert.Error(t, err)
	})
}

// TestVault_Update covers Update success, invalid ids, and missing rows.
func TestVault_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("happy", func(t *testing.T) {

		// Blind replace: id exists, new plaintext round-trips through Retrieve.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns", []byte("v1"))
		assert.Ok(t, err)
		assert.Ok(t, v.Update(ctx, "ns", id, []byte("v2")))
		got, err := v.Retrieve(ctx, "ns", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("v2"), got)
	})

	t.Run("invalid_id", func(t *testing.T) {

		// Parse fails before storage is touched.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		err := v.Update(ctx, "ns", "bad-id", []byte("z"))
		assert.ErrorIs(t, err, verrors.ErrInvalidIdentifier)
	})

	t.Run("missing_row", func(t *testing.T) {

		// Replace on a well-formed but unknown id returns ErrNotFound from storage.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		err := v.Update(ctx, "ns", "0123456789abcdef0123456789abcdef", []byte("z"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrStorage)
		assert.ErrorIs(t, err, verrors.ErrNotFound)
	})
}

// TestVault_CompareAndSwap exercises compare-and-swap on explicit current and new plaintext.
func TestVault_CompareAndSwap(t *testing.T) {
	ctx := context.Background()

	t.Run("happy", func(t *testing.T) {

		// currentPlain matches decrypted value, so CAS writes newPlain and read sees it.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns", []byte("alpha"))
		assert.Ok(t, err)
		err = v.CompareAndSwap(ctx, "ns", id, []byte("alpha"), []byte("beta"))
		assert.Ok(t, err)
		got, err := v.Retrieve(ctx, "ns", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("beta"), got)
	})

	t.Run("wrong_current_plaintext", func(t *testing.T) {

		// Mismatch before CAS: row must not change, caller gets ErrWrongCurrent.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns", []byte("stored"))
		assert.Ok(t, err)
		err = v.CompareAndSwap(ctx, "ns", id, []byte("not-stored"), []byte("new"))
		assert.ErrorIs(t, err, verrors.ErrWrongCurrent)
	})

	t.Run("invalid_id", func(t *testing.T) {

		// Parse rejects id before any read.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		err := v.CompareAndSwap(ctx, "ns", "bad", []byte("a"), []byte("b"))
		assert.ErrorIs(t, err, verrors.ErrInvalidIdentifier)
	})

	t.Run("missing_row", func(t *testing.T) {

		// Get fails for unknown id before open/compare.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		err := v.CompareAndSwap(ctx, "ns", "0123456789abcdef0123456789abcdef", []byte("a"), []byte("b"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrStorage)
		assert.ErrorIs(t, err, verrors.ErrNotFound)
	})

	t.Run("decrypt_fails_corrupt_ciphertext", func(t *testing.T) {

		// Truncated or random bytes under the id make open fail before plaintext compare.
		st := storage.NewMemStorage()
		v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
		id, err := v.Store(ctx, "good-ns", []byte("payload"))
		assert.Ok(t, err)
		st.BypassSemanticsSetBlobForTest("good-ns", id, []byte{9, 9, 9})
		err = v.CompareAndSwap(ctx, "good-ns", id, []byte("payload"), []byte("x"))
		assert.Error(t, err)
	})

	t.Run("CAS_lost", func(t *testing.T) {

		// Storage always loses CAS so the vault surfaces ErrCASFailed even when current matches.
		base := storage.NewMemStorage()
		st := &casFailStorage{MemStorage: base}
		v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns", []byte("cur"))
		assert.Ok(t, err)
		err = v.CompareAndSwap(ctx, "ns", id, []byte("cur"), []byte("next"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrStorage)
		assert.ErrorIs(t, err, verrors.ErrCASFailed)
	})
}

// TestVault_MoveNamespace covers re-sealing across namespaces and failure modes.
func TestVault_MoveNamespace(t *testing.T) {
	ctx := context.Background()

	t.Run("noop_equal_namespaces", func(t *testing.T) {

		// Same src/dst is a no-op but must still succeed and leave data readable.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		id, err := v.Store(ctx, "same", []byte("x"))
		assert.Ok(t, err)
		assert.Ok(t, v.MoveNamespace(ctx, "same", "same", id))
		got, err := v.Retrieve(ctx, "same", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("x"), got)
	})

	t.Run("happy", func(t *testing.T) {

		// Destination holds plaintext; source row is removed after successful re-seal.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		id, err := v.Store(ctx, "from", []byte("payload"))
		assert.Ok(t, err)
		assert.Ok(t, v.MoveNamespace(ctx, "from", "to", id))
		got, err := v.Retrieve(ctx, "to", id)
		assert.Ok(t, err)
		assert.Equal(t, []byte("payload"), got)
		_, err = v.Retrieve(ctx, "from", id)
		assert.ErrorIs(t, err, verrors.ErrNotFound)
	})

	t.Run("invalid_id", func(t *testing.T) {

		// Identifier parse fails before touching storage.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		err := v.MoveNamespace(ctx, "a", "b", "bad-id")
		assert.ErrorIs(t, err, verrors.ErrInvalidIdentifier)
	})

	t.Run("missing_source", func(t *testing.T) {

		// No row at source id surfaces ErrNotFound.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		err := v.MoveNamespace(ctx, "from", "to", "0123456789abcdef0123456789abcdef")
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrStorage)
		assert.ErrorIs(t, err, verrors.ErrNotFound)
	})

	t.Run("decrypt_source_fails", func(t *testing.T) {

		// Corrupt source blob cannot be opened, so move aborts before writing destination.
		st := storage.NewMemStorage()
		v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
		id, err := v.Store(ctx, "from", []byte("ok"))
		assert.Ok(t, err)
		st.BypassSemanticsSetBlobForTest("from", id, []byte{1, 2, 3})
		err = v.MoveNamespace(ctx, "from", "to", id)
		assert.Error(t, err)
	})

	t.Run("duplicate_destination", func(t *testing.T) {

		// Destination key already exists: Create must fail with ErrDuplicateKey.
		st := storage.NewMemStorage()
		v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
		id, err := v.Store(ctx, "from", []byte("a"))
		assert.Ok(t, err)
		assert.Ok(t, st.Create(ctx, "to", id, []byte("occupies")))
		err = v.MoveNamespace(ctx, "from", "to", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrStorage)
		assert.ErrorIs(t, err, verrors.ErrDuplicateKey)
	})

	t.Run("incomplete_after_create_delete_fails", func(t *testing.T) {

		// Destination write succeeded but source delete failed: both sides still readable, ErrMoveNamespaceIncomplete.
		base := storage.NewMemStorage()
		st := &deleteFailsOnNs{MemStorage: base, ns: "from"}
		v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
		id, err := v.Store(ctx, "from", []byte("data"))
		assert.Ok(t, err)
		err = v.MoveNamespace(ctx, "from", "to", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrMoveNamespaceIncomplete)
		assert.ErrorIs(t, err, verrors.ErrStorage)
		_, err = v.Retrieve(ctx, "to", id)
		assert.Ok(t, err)
		_, err = v.Retrieve(ctx, "from", id)
		assert.Ok(t, err)
	})
}

// TestVault_Delete covers successful delete, idempotency, invalid ids, and storage errors.
func TestVault_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("happy", func(t *testing.T) {

		// After delete, retrieve must see the row as gone.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns", []byte("x"))
		assert.Ok(t, err)
		assert.Ok(t, v.Delete(ctx, "ns", id))
		_, err = v.Retrieve(ctx, "ns", id)
		assert.ErrorIs(t, err, verrors.ErrNotFound)
	})

	t.Run("idempotent_missing", func(t *testing.T) {

		// Missing row delete is a no-op success (idempotent).
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		assert.Ok(t, v.Delete(ctx, "ns", "0123456789abcdef0123456789abcdef"))
	})

	t.Run("invalid_id", func(t *testing.T) {

		// Bad id format rejected before storage delete.
		v := testEnvelopeVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
		err := v.Delete(ctx, "ns", "bad-id")
		assert.ErrorIs(t, err, verrors.ErrInvalidIdentifier)
	})

	t.Run("delete_storage_error", func(t *testing.T) {

		// Propagate storage delete failures as ErrStorage.
		base := storage.NewMemStorage()
		st := &deleteFailsOnNs{MemStorage: base, ns: "ns"}
		v := testEnvelopeVault(t, st, identifier.HexIdentifier{})
		id, err := v.Store(ctx, "ns", []byte("x"))
		assert.Ok(t, err)
		err = v.Delete(ctx, "ns", id)
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrStorage)
	})
}

//=============================================================================
// Test helpers and fakes
//=============================================================================

// testEnvelopeVault returns a [v1.Vault] backed by st and id using a fresh X25519 wrapping key.
func testEnvelopeVault(tb testing.TB, st storage.Storage, id identifier.Identifier) v1.Vault {
	tb.Helper()
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(tb, err)
	v, err := v1.New(priv, st, id)
	assert.Ok(tb, err)
	return v
}

type errNewIdentifier struct{ identifier.HexIdentifier }

// vaultEOFReader simulates entropy source failure for seal path tests.
type vaultEOFReader struct{}

func (vaultEOFReader) Read([]byte) (int, error) { return 0, io.EOF }

// New returns an error so tests can exercise identifier mint failures.
func (errNewIdentifier) New() (string, error) {
	return "", errors.New("identifier mint failed")
}

type dupNewIdentifier struct{ identifier.HexIdentifier }

// New returns a fixed id so a second Store hits ErrDuplicateKey.
func (dupNewIdentifier) New() (string, error) {
	return "0123456789abcdef0123456789abcdef", nil
}

type casFailStorage struct {
	*storage.MemStorage
}

// CompareAndSwap always reports a lost compare-and-swap race.
func (*casFailStorage) CompareAndSwap(ctx context.Context, namespace, id string, oldCipher, newCipher []byte) error {
	_ = ctx
	_ = namespace
	_ = id
	_ = oldCipher
	_ = newCipher
	return verrors.ErrCASFailed
}

type deleteFailsOnNs struct {
	*storage.MemStorage
	ns string
}

// Delete simulates a backend failure for namespace ns.
func (s *deleteFailsOnNs) Delete(ctx context.Context, namespace, id string) error {
	if namespace == s.ns {
		return errors.New("simulated delete failure")
	}
	return s.MemStorage.Delete(ctx, namespace, id)
}
