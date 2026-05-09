package stringvault_test

// Tests for stringvault: UTF-8 string payloads layered on encrypted byte storage.

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault"
	"go.rtnl.ai/x/vault/stringvault"
	"go.rtnl.ai/x/vault/vaulttest"
)

// TestStringVault_roundtrip wires stringvault around MemStorage and verifies plain text
// survives encrypt/decrypt without corruption or accidental trimming.
func TestStringVault_roundtrip(t *testing.T) {
	// Prepare a 32-byte key and create a new Vault, backed by MemStorage and using HexIdentifier for IDs.
	key := make([]byte, 32)
	v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
	assert.Ok(t, err)

	// Wrap the vault with stringvault to use UTF-8 string payloads.
	w := stringvault.New(v)
	ctx := context.Background()

	// The plaintext we want to store and retrieve.
	const want = "hello"

	// Store the plaintext in the vault; receive an ID upon success.
	id, err := w.Store(ctx, "ns", want)
	assert.Ok(t, err)

	// Retrieve the plaintext using the returned ID.
	s, err := w.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)

	// Verify that what was retrieved matches what was originally stored.
	assert.Equal(t, want, s)
}
