package jsonvault_test

// Tests jsonvault JSON encoding on top of [vaulttest.TestVault].

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/vault/errors"
	"go.rtnl.ai/x/vault/identifier"
	"go.rtnl.ai/x/vault/jsonvault"
	"go.rtnl.ai/x/vault/storage"
	"go.rtnl.ai/x/vault/vaulttest"
)

// TestJSONVault_roundtrip ensures Store and Retrieve round-trip JSON through the vault.
func TestJSONVault_roundtrip(t *testing.T) {
	v := vaulttest.NewTestVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
	w := jsonvault.New(v)
	ctx := context.Background()

	type payload struct {
		A int `json:"a"`
	}

	// JSON encode on store, decode into dst on retrieve.
	id, err := w.Store(ctx, "ns", payload{A: 42})
	assert.Ok(t, err)

	var got payload
	assert.Ok(t, w.Retrieve(ctx, "ns", id, &got))
	assert.Equal(t, 42, got.A)
}

// TestJSONVault_Store_marshal_failure ensures non-marshalable values return ErrJSONMarshal.
func TestJSONVault_Store_marshal_failure(t *testing.T) {
	v := vaulttest.NewTestVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
	w := jsonvault.New(v)
	ctx := context.Background()

	ch := make(chan int)

	// Channels are not JSON-serializable; Store must fail before touching storage.
	_, err := w.Store(ctx, "ns", ch)

	assert.ErrorIs(t, err, verrors.ErrJSONMarshal)
}

// TestJSONVault_Retrieve_unmarshal_failure ensures corrupt JSON yields [verrors.ErrJSONUnmarshal]
// and [verrors.ErrInvalidJSON] after decrypt.
func TestJSONVault_Retrieve_unmarshal_failure(t *testing.T) {
	st := storage.NewMemStorage()
	v := vaulttest.NewTestVault(t, st, identifier.HexIdentifier{})
	w := jsonvault.New(v)
	ctx := context.Background()

	type payload struct {
		A int `json:"a"`
	}

	id, err := w.Store(ctx, "ns", payload{A: 1})
	assert.Ok(t, err)

	// Corrupt JSON in the vault row after a valid store; Retrieve must surface unmarshal errors.
	assert.Ok(t, w.Vault.Update(ctx, "ns", id, []byte(`{"a":`)))

	err = w.Retrieve(ctx, "ns", id, new(payload))

	assert.ErrorIs(t, err, verrors.ErrJSONUnmarshal)
	assert.ErrorIs(t, err, verrors.ErrInvalidJSON)
}

// TestEqualJSON_marshal_failure ensures [jsonvault.EqualJSON] surfaces [verrors.ErrJSONMarshal] when marshal fails.
func TestEqualJSON_marshal_failure(t *testing.T) {
	_, err := jsonvault.EqualJSON(make(chan int), 1)

	assert.ErrorIs(t, err, verrors.ErrJSONMarshal)
}

// TestJSONVault_Retrieve_nil_dst ensures [jsonvault.Vault.Retrieve] rejects a nil dst with [verrors.ErrNilRetrieveDst]
// and that a non-pointer dst fails with [verrors.ErrJSONUnmarshal] from [encoding/json.Unmarshal].
func TestJSONVault_Retrieve_nil_dst(t *testing.T) {
	v := vaulttest.NewTestVault(t, storage.NewMemStorage(), identifier.HexIdentifier{})
	w := jsonvault.New(v)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", 1)
	assert.Ok(t, err)

	assert.ErrorIs(t, w.Retrieve(ctx, "ns", id, nil), verrors.ErrNilRetrieveDst)
	err = w.Retrieve(ctx, "ns", id, 0)
	assert.Error(t, err)
	assert.ErrorIs(t, err, verrors.ErrJSONUnmarshal)
}
