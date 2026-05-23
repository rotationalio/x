/*
Package jsonpurser wraps contract.Purser, exposing the same operation names with JSON instead of raw bytes:
Store and Update take any and marshal with encoding/json;
Retrieve unmarshals into dst; CompareAndSwap takes expected current and new JSON as []byte.
*/
package jsonpurser

// JSON-encoded payloads on top of contract.Purser using encoding/json.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
)

// Purser embeds a contract.Purser and exposes the same operation names, using JSON
// (any for store/update; CompareAndSwap for compare-and-swap on JSON bytes) instead of opaque plaintext bytes.
// MoveNamespace and Delete are promoted from the embedded purser.
type Purser struct {
	contract.Purser
}

// New wraps a non-nil contract.Purser.
func New(p contract.Purser) *Purser {
	if p == nil {
		panic("purser/wrappers/json: New(nil)")
	}
	return &Purser{Purser: p}
}

// Store marshals value with json.Marshal and stores the result via the inner contract.Purser.Store.
func (w *Purser) Store(ctx context.Context, namespace string, value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", errors.Join(perrors.ErrJSONMarshal, err)
	}
	return w.Purser.Store(ctx, namespace, b)
}

// Update marshals newValue and updates the row via the inner contract.Purser.Update.
func (w *Purser) Update(ctx context.Context, namespace, identifier string, newValue any) error {
	b, err := json.Marshal(newValue)
	if err != nil {
		return errors.Join(perrors.ErrJSONMarshal, err)
	}
	return w.Purser.Update(ctx, namespace, identifier, b)
}

// CompareAndSwap replaces the row only if decrypted JSON plaintext matches currentPlain, then stores newPlain.
// Non-empty currentPlain and newPlain must be valid JSON.
func (w *Purser) CompareAndSwap(ctx context.Context, namespace, identifier string, currentPlain, newPlain []byte) error {
	if len(currentPlain) > 0 && !json.Valid(currentPlain) {
		return errors.Join(perrors.ErrJSONUnmarshal, perrors.ErrInvalidJSON)
	}
	if len(newPlain) > 0 && !json.Valid(newPlain) {
		return errors.Join(perrors.ErrJSONUnmarshal, perrors.ErrInvalidJSON)
	}
	return w.Purser.CompareAndSwap(ctx, namespace, identifier, currentPlain, newPlain)
}

// Retrieve decrypts the row and unmarshals JSON into dst (dst must not be nil).
func (w *Purser) Retrieve(ctx context.Context, namespace, identifier string, dst any) error {
	if dst == nil {
		return perrors.ErrNilRetrieveDst
	}
	b, err := w.Purser.Retrieve(ctx, namespace, identifier)
	if err != nil {
		return err
	}
	if len(b) > 0 && !json.Valid(b) {
		return errors.Join(perrors.ErrJSONUnmarshal, perrors.ErrInvalidJSON)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return errors.Join(perrors.ErrJSONUnmarshal, err)
	}
	return nil
}

// EqualJSON reports whether a and b marshal to identical JSON bytes (canonical equality for CAS helpers).
func EqualJSON(a, b any) (bool, error) {
	ab, err := json.Marshal(a)
	if err != nil {
		return false, errors.Join(perrors.ErrJSONMarshal, err)
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false, errors.Join(perrors.ErrJSONMarshal, err)
	}
	return bytes.Equal(ab, bb), nil
}
