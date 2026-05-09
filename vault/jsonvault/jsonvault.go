// Package jsonvault wraps [go.rtnl.ai/x/vault.Vault] with encoding/json for values.
package jsonvault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"go.rtnl.ai/x/vault"
)

// Sentinel errors for [errors.Is]; JSON failures often wrap encoding/json errors with
// [errors.Join](sentinel, underlying).
var (
	// ErrJSONMarshal means [json.Marshal] failed (Store, Update, AtomicUpdate current/new, or [EqualJSON]).
	ErrJSONMarshal = errors.New("jsonvault: json marshal failed")

	// ErrJSONUnmarshal means [json.Unmarshal] failed in [Retrieve] (corrupt payload, wrong shape, etc.).
	ErrJSONUnmarshal = errors.New("jsonvault: json unmarshal failed")
)

// Vault wraps [*vault.Vault] with JSON-encoded payloads.
type Vault struct {
	V *vault.Vault
}

// New wraps an existing [*vault.Vault].
func New(v *vault.Vault) *Vault {
	return &Vault{V: v}
}

// Store marshals value and delegates to [vault.Vault.Store].
func (w *Vault) Store(ctx context.Context, namespace string, value any) (id string, err error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", errors.Join(ErrJSONMarshal, err)
	}
	return w.V.Store(ctx, namespace, b)
}

// Update marshals newValue and delegates to [vault.Vault.Update].
func (w *Vault) Update(ctx context.Context, namespace, id string, newValue any) error {
	b, err := json.Marshal(newValue)
	if err != nil {
		return errors.Join(ErrJSONMarshal, err)
	}
	return w.V.Update(ctx, namespace, id, b)
}

// AtomicUpdate compares the stored JSON plaintext to the marshaled currentValue (exact bytes),
// then replaces with marshaled newValue. Semantic mismatches only matter through JSON serialization:
// use the same shapes/types you expect to retrieve. Use [EqualJSON] to compare currentValue and newValue before CAS.
func (w *Vault) AtomicUpdate(ctx context.Context, namespace, id string, currentValue, newValue any) error {
	cur, err := json.Marshal(currentValue)
	if err != nil {
		return errors.Join(ErrJSONMarshal, err)
	}
	nxt, err := json.Marshal(newValue)
	if err != nil {
		return errors.Join(ErrJSONMarshal, err)
	}
	return w.V.AtomicUpdate(ctx, namespace, id, cur, nxt)
}

// Retrieve unmarshals into T using [json.Unmarshal].
func Retrieve[T any](w *Vault, ctx context.Context, namespace, id string) (T, error) {
	var z T
	b, err := w.V.Retrieve(ctx, namespace, id)
	if err != nil {
		return z, err
	}
	if err := json.Unmarshal(b, &z); err != nil {
		return z, errors.Join(ErrJSONUnmarshal, err)
	}
	return z, nil
}

// MoveNamespace delegates to [vault.Vault.MoveNamespace].
func (w *Vault) MoveNamespace(ctx context.Context, fromNamespace, toNamespace, id string) error {
	return w.V.MoveNamespace(ctx, fromNamespace, toNamespace, id)
}

// Delete delegates to [vault.Vault.Delete].
func (w *Vault) Delete(ctx context.Context, namespace, id string) error {
	return w.V.Delete(ctx, namespace, id)
}

// EqualJSON reports whether a and b marshal to identical JSON bytes (canonical equality for CAS helpers).
func EqualJSON(a, b any) (bool, error) {
	ab, err := json.Marshal(a)
	if err != nil {
		return false, errors.Join(ErrJSONMarshal, err)
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false, errors.Join(ErrJSONMarshal, err)
	}
	return bytes.Equal(ab, bb), nil
}
