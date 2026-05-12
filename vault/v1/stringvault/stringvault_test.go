package stringvault_test

// Tests stringvault UTF-8 string payloads on top of [vaulttest.TestVault].

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/vault/v1/errors"
	"go.rtnl.ai/x/vault/v1/identifier"
	"go.rtnl.ai/x/vault/v1/storage"
	"go.rtnl.ai/x/vault/v1/stringvault"
	"go.rtnl.ai/x/vault/v1/vaulttest"
)

// TestStringVault_roundtrip checks [stringvault.Vault.Store] and [stringvault.Vault.Retrieve] preserve a UTF-8 string.
func TestStringVault_roundtrip(t *testing.T) {
	v := vaulttest.NewTestVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
	w := stringvault.New(v)
	ctx := context.Background()

	const want = "hello"

	// Store encodes UTF-8; retrieve decodes back to the same string.
	id, err := w.Store(ctx, "ns", want)
	assert.Ok(t, err)

	got, err := w.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)

	assert.Equal(t, want, got)
}

// TestStringVault_invalidUTF8 checks that storing invalid UTF-8 returns [verrors.ErrInvalidUTF8].
func TestStringVault_invalidUTF8(t *testing.T) {
	st := storage.NewMemStorage()
	v := vaulttest.NewTestVault(t, st, identifier.HexIdentifier{})
	w := stringvault.New(v)
	ctx := context.Background()

	// Invalid UTF-8 string: 0xff is not legal in UTF-8.
	invalid := string([]byte{0xff, 0xfe})

	// Store should reject invalid UTF-8 input.
	_, err := w.Store(ctx, "ns", invalid)
	assert.ErrorIs(t, err, verrors.ErrInvalidUTF8)
}

// TestStringVault_invalidUTF8_corrupt_row checks [stringvault.Vault.Retrieve] rejects invalid UTF-8 when the
// underlying vault stores raw bytes and storage is corrupted, returning [verrors.ErrInvalidUTF8].
func TestStringVault_invalidUTF8_corrupt_row(t *testing.T) {
	st := storage.NewMemStorage()
	v := vaulttest.NewTestVault(t, st, identifier.HexIdentifier{})
	w := stringvault.New(v)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", "good")
	assert.Ok(t, err)

	st.BypassSemanticsSetBlobForTest("ns", id, []byte{0xff, 0xfe})

	_, err = w.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, verrors.ErrInvalidUTF8)
}
