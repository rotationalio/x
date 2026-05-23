/*
Package jsonpurser wraps [purser.Purser] with JSON instead of raw bytes for Store and Update.

Construct the inner purser with [purser.New], [purser.NewMemHold], [purser.NewMemring], and related
root aliases; build keys with [purser.NewPassword] and [purser.Keyring.Register]. Classify errors with purser/errors.
*/
package jsonpurser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
)

// Purser embeds a [purser.Purser] and exposes JSON-shaped Store, Update, and Retrieve.
type Purser struct {
	*purser.Purser
}

// New wraps a non-nil [purser.Purser].
func New(p *purser.Purser) *Purser {
	if p == nil {
		panic("purser/wrappers/json: New(nil)")
	}
	return &Purser{Purser: p}
}

// Store marshals value and stores the result via the inner [Purser.Store].
func (w *Purser) Store(ctx context.Context, namespace string, value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", errors.Join(perrors.ErrJSONMarshal, err)
	}
	res, err := w.Purser.Store(ctx, namespace, b)
	return res.ID, err
}

// Update marshals newValue and updates the row via the inner [Purser.Update].
func (w *Purser) Update(ctx context.Context, namespace, identifier string, newValue any) error {
	b, err := json.Marshal(newValue)
	if err != nil {
		return errors.Join(perrors.ErrJSONMarshal, err)
	}
	_, err = w.Purser.Update(ctx, namespace, identifier, b)
	return err
}

// CompareAndSwap replaces the row only if decrypted JSON plaintext matches currentPlain.
func (w *Purser) CompareAndSwap(ctx context.Context, namespace, identifier string, currentPlain, newPlain []byte) (purser.Result, error) {
	if len(currentPlain) > 0 && !json.Valid(currentPlain) {
		return purser.Result{}, errors.Join(perrors.ErrJSONUnmarshal, perrors.ErrInvalidJSON)
	}
	if len(newPlain) > 0 && !json.Valid(newPlain) {
		return purser.Result{}, errors.Join(perrors.ErrJSONUnmarshal, perrors.ErrInvalidJSON)
	}
	return w.Purser.CompareAndSwap(ctx, namespace, identifier, currentPlain, newPlain)
}

// Retrieve decrypts the row and unmarshals JSON into dst.
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

// EqualJSON reports whether a and b marshal to identical JSON bytes.
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
