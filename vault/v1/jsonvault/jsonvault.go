/*
Package jsonvault wraps [v1.Vault], exposing the same operation names with JSON instead of raw bytes:
[Store] and [Update] take [any] and marshal with [encoding/json];
[Retrieve] unmarshals into dst; [CompareAndSwap] takes expected current and new JSON as []byte (validated with [encoding/json.Valid] when non-empty) and delegates to the embedded vault. Rows remain opaque ciphertext in the storage backend; the
embedded [v1.Vault] is also available as the struct field Vault (e.g. [v1.Vault.Update] with raw bytes in tests).
*/
package jsonvault

// JSON-encoded payloads on top of [v1.Vault] using encoding/json.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	v1 "go.rtnl.ai/x/vault/v1"
	verrors "go.rtnl.ai/x/vault/v1/errors"
)

// Vault embeds a [v1.Vault] and exposes the same operation names as [v1.Vault], using JSON
// ([any] for store/update; [CompareAndSwap] for compare-and-swap on JSON bytes) instead of opaque plaintext bytes.
// [MoveNamespace] and [Delete] are promoted from the embedded vault.
type Vault struct {
	v1.Vault
}

// New wraps a non-nil [v1.Vault] (for example from [v1.New]).
func New(v v1.Vault) *Vault {
	if v == nil {
		panic("jsonvault: New(nil)")
	}
	return &Vault{Vault: v}
}

// Store marshals value with [json.Marshal] and stores the result via the inner [v1.Vault.Store].
func (w *Vault) Store(ctx context.Context, namespace string, value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", errors.Join(verrors.ErrJSONMarshal, err)
	}
	return w.Vault.Store(ctx, namespace, b)
}

// Update marshals newValue and updates the row via the inner [v1.Vault.Update].
func (w *Vault) Update(ctx context.Context, namespace, id string, newValue any) error {
	b, err := json.Marshal(newValue)
	if err != nil {
		return errors.Join(verrors.ErrJSONMarshal, err)
	}
	return w.Vault.Update(ctx, namespace, id, b)
}

// CompareAndSwap replaces the row only if decrypted JSON plaintext matches currentPlain, then stores newPlain.
// Non-empty currentPlain and newPlain must be valid JSON ([encoding/json.Valid]).
func (w *Vault) CompareAndSwap(ctx context.Context, namespace, id string, currentPlain, newPlain []byte) error {
	if len(currentPlain) > 0 && !json.Valid(currentPlain) {
		return errors.Join(verrors.ErrJSONUnmarshal, verrors.ErrInvalidJSON)
	}
	if len(newPlain) > 0 && !json.Valid(newPlain) {
		return errors.Join(verrors.ErrJSONUnmarshal, verrors.ErrInvalidJSON)
	}
	return w.Vault.CompareAndSwap(ctx, namespace, id, currentPlain, newPlain)
}

// Retrieve decrypts the row and unmarshals JSON into dst (dst must not be nil;
// same constraints as [encoding/json.Unmarshal]). A nil dst returns [verrors.ErrNilRetrieveDst].
// Non-empty stored bytes that are not valid JSON return [errors.Join] of [verrors.ErrJSONUnmarshal] and
// [verrors.ErrInvalidJSON] before unmarshal.
func (w *Vault) Retrieve(ctx context.Context, namespace, id string, dst any) error {
	if dst == nil {
		return verrors.ErrNilRetrieveDst
	}
	b, err := w.Vault.Retrieve(ctx, namespace, id)
	if err != nil {
		return err
	}
	if len(b) > 0 && !json.Valid(b) {
		return errors.Join(verrors.ErrJSONUnmarshal, verrors.ErrInvalidJSON)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return errors.Join(verrors.ErrJSONUnmarshal, err)
	}
	return nil
}

// EqualJSON reports whether a and b marshal to identical JSON bytes (canonical equality for CAS helpers).
func EqualJSON(a, b any) (bool, error) {
	ab, err := json.Marshal(a)
	if err != nil {
		return false, errors.Join(verrors.ErrJSONMarshal, err)
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false, errors.Join(verrors.ErrJSONMarshal, err)
	}
	return bytes.Equal(ab, bb), nil
}
