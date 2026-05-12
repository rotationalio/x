/*
Package stringvault wraps [v1.Vault] with a string-shaped API: plaintext is
UTF-8 text ([Store], [Retrieve], [Update], [CompareAndSwap]); bytes on the wire remain opaque to your storage implementation via the embedded [v1.Vault].
*/
package stringvault

// UTF-8 string payloads on top of [v1.Vault]; invalid UTF-8 returns [verrors.ErrInvalidUTF8].

import (
	"context"
	"unicode/utf8"

	v1 "go.rtnl.ai/x/vault/v1"
	verrors "go.rtnl.ai/x/vault/v1/errors"
)

// Vault embeds a [v1.Vault] and enforces UTF-8 on string plaintext at this API boundary.
// [MoveNamespace] and [Delete] are promoted from the embedded vault.
type Vault struct {
	v1.Vault
}

// New wraps a non-nil [v1.Vault] (for example from [v1.New]).
func New(v v1.Vault) *Vault {
	if v == nil {
		panic("stringvault: New(nil)")
	}
	return &Vault{Vault: v}
}

// Store rejects non-UTF-8 strings, then delegates to the inner [v1.Vault.Store].
func (w *Vault) Store(ctx context.Context, namespace string, plaintext string) (string, error) {
	if !utf8.ValidString(plaintext) {
		return "", verrors.ErrInvalidUTF8
	}
	return w.Vault.Store(ctx, namespace, []byte(plaintext))
}

// Retrieve delegates to the inner [v1.Vault.Retrieve] and returns UTF-8 text, or [verrors.ErrInvalidUTF8]
// if the decrypted bytes are not valid UTF-8.
func (w *Vault) Retrieve(ctx context.Context, namespace, id string) (string, error) {
	b, err := w.Vault.Retrieve(ctx, namespace, id)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", verrors.ErrInvalidUTF8
	}
	return string(b), nil
}

// Update rejects non-UTF-8 strings, then delegates to the inner [v1.Vault.Update].
func (w *Vault) Update(ctx context.Context, namespace, id string, plaintext string) error {
	if !utf8.ValidString(plaintext) {
		return verrors.ErrInvalidUTF8
	}
	return w.Vault.Update(ctx, namespace, id, []byte(plaintext))
}

// CompareAndSwap replaces the row only if decrypted plaintext matches currentPlain, then stores newPlain.
// currentPlain and newPlain must be valid UTF-8 strings.
func (w *Vault) CompareAndSwap(ctx context.Context, namespace, id string, currentPlain, newPlain string) error {
	if !utf8.ValidString(currentPlain) {
		return verrors.ErrInvalidUTF8
	}
	if !utf8.ValidString(newPlain) {
		return verrors.ErrInvalidUTF8
	}
	return w.Vault.CompareAndSwap(ctx, namespace, id, []byte(currentPlain), []byte(newPlain))
}
