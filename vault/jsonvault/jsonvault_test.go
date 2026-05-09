// External tests for go.rtnl.ai/x/vault/jsonvault: JSON marshal/unmarshal around an encrypted
// core [vault.Vault], sentinel errors on encoding failures, and [EqualJSON] helpers.
package jsonvault_test

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault"
	"go.rtnl.ai/x/vault/jsonvault"
	"go.rtnl.ai/x/vault/vaulttest"
)

// TestJSONVault_roundtrip ensures Store JSON-encodes payloads and Retrieve[T] decodes symmetrically
// after vault encrypt/decrypt at rest.
func TestJSONVault_roundtrip(t *testing.T) {
	// Setup: AES vault over memory + jsonvault wrapper.
	key := make([]byte, 32)
	v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
	assert.Ok(t, err)
	w := jsonvault.New(v)
	ctx := context.Background()

	type payload struct {
		A int `json:"a"`
	}

	// Exercise: Store marshals struct → vault seals JSON bytes.
	id, err := w.Store(ctx, "ns", payload{A: 42})
	assert.Ok(t, err)

	// Exercise / expect: Retrieve unmarshals back into the same logical value.
	got, err := jsonvault.Retrieve[payload](w, ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, 42, got.A)
}

// TestJSONVault_Store_marshal_failure ensures non-JSON-marshalable values surface ErrJSONMarshal.
func TestJSONVault_Store_marshal_failure(t *testing.T) {
	// Setup.
	key := make([]byte, 32)
	v, err := vault.New(key, vaulttest.NewMemStorage(), vaulttest.HexIdentifier{})
	assert.Ok(t, err)
	w := jsonvault.New(v)
	ctx := context.Background()

	ch := make(chan int)

	// Exercise: channels cannot be encoded by encoding/json.
	_, err = w.Store(ctx, "ns", ch)

	// Expect: sentinel join so callers can errors.Is(ErrJSONMarshal).
	assert.ErrorIs(t, err, jsonvault.ErrJSONMarshal)
}

// TestJSONVault_Retrieve_unmarshal_failure ensures corrupt JSON plaintext yields ErrJSONUnmarshal
// after successful decrypt.
func TestJSONVault_Retrieve_unmarshal_failure(t *testing.T) {
	// Setup: shared MemStorage so we rely on a known row key after Store.
	key := make([]byte, 32)
	st := vaulttest.NewMemStorage()
	v, err := vault.New(key, st, vaulttest.HexIdentifier{})
	assert.Ok(t, err)
	w := jsonvault.New(v)
	ctx := context.Background()

	type payload struct {
		A int `json:"a"`
	}

	id, err := w.Store(ctx, "ns", payload{A: 1})
	assert.Ok(t, err)

	// Arrange: replace vault plaintext with truncated JSON (vault.Update re-seals raw bytes; skips jsonvault marshaling).
	assert.Ok(t, w.V.Update(ctx, "ns", id, []byte(`{"a":`)))

	// Exercise: decrypt succeeds, Unmarshal fails.
	_, err = jsonvault.Retrieve[payload](w, ctx, "ns", id)

	// Expect.
	assert.ErrorIs(t, err, jsonvault.ErrJSONUnmarshal)
}

// TestEqualJSON_marshal_failure ensures EqualJSON reports ErrJSONMarshal when either argument cannot marshal.
func TestEqualJSON_marshal_failure(t *testing.T) {
	// Exercise: left-hand value is not JSON-marshalable.
	_, err := jsonvault.EqualJSON(make(chan int), 1)

	// Expect.
	assert.ErrorIs(t, err, jsonvault.ErrJSONMarshal)
}
