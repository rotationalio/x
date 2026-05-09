// Package stringvault provides a thin UTF-8 wrapper around [go.rtnl.ai/x/vault.Vault].
package stringvault

import (
	"context"
	"errors"
	"unicode/utf8"

	"go.rtnl.ai/x/vault"
)

// ErrInvalidUTF8 is returned when a string argument is not valid UTF-8.
var ErrInvalidUTF8 = errors.New("stringvault: plain text is not valid UTF-8")

// Vault wraps [*vault.Vault] with string payloads (UTF-8 bytes on the wire).
type Vault struct {
	V *vault.Vault
}

// New wraps an existing [*vault.Vault]. V must not be nil.
func New(v *vault.Vault) *Vault {
	return &Vault{V: v}
}

// Store encodes plain as UTF-8 bytes and delegates to [vault.Vault.Store].
func (w *Vault) Store(ctx context.Context, namespace, plain string) (id string, err error) {
	if !utf8.ValidString(plain) {
		return "", ErrInvalidUTF8
	}
	return w.V.Store(ctx, namespace, []byte(plain))
}

// Update delegates to [vault.Vault.Update].
func (w *Vault) Update(ctx context.Context, namespace, id, newPlain string) error {
	if !utf8.ValidString(newPlain) {
		return ErrInvalidUTF8
	}
	return w.V.Update(ctx, namespace, id, []byte(newPlain))
}

// AtomicUpdate delegates to [vault.Vault.AtomicUpdate].
func (w *Vault) AtomicUpdate(ctx context.Context, namespace, id, currentPlain, newPlain string) error {
	if !utf8.ValidString(currentPlain) || !utf8.ValidString(newPlain) {
		return ErrInvalidUTF8
	}
	return w.V.AtomicUpdate(ctx, namespace, id, []byte(currentPlain), []byte(newPlain))
}

// Retrieve returns UTF-8 plaintext from [vault.Vault.Retrieve].
func (w *Vault) Retrieve(ctx context.Context, namespace, id string) (string, error) {
	b, err := w.V.Retrieve(ctx, namespace, id)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// MoveNamespace delegates to [vault.Vault.MoveNamespace].
func (w *Vault) MoveNamespace(ctx context.Context, fromNamespace, toNamespace, id string) error {
	return w.V.MoveNamespace(ctx, fromNamespace, toNamespace, id)
}

// Delete delegates to [vault.Vault.Delete].
func (w *Vault) Delete(ctx context.Context, namespace, id string) error {
	return w.V.Delete(ctx, namespace, id)
}
