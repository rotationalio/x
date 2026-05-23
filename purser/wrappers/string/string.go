/*
Package stringpurser wraps contract.Purser with a string-shaped API: plaintext is
UTF-8 text (Store, Retrieve, Update, CompareAndSwap); bytes on the wire remain opaque.
*/
package stringpurser

// UTF-8 string payloads on top of contract.Purser; invalid UTF-8 returns errors.ErrInvalidUTF8.

import (
	"context"
	"unicode/utf8"

	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
)

// Purser embeds a contract.Purser and enforces UTF-8 on string plaintext at this API boundary.
// MoveNamespace and Delete are promoted from the embedded Purser.
type Purser struct {
	contract.Purser
}

// New wraps a non-nil contract.Purser.
func New(p contract.Purser) *Purser {
	if p == nil {
		panic("purser/wrappers/string: New(nil)")
	}
	return &Purser{Purser: p}
}

// Store rejects non-UTF-8 strings, then delegates to the inner contract.Purser.Store.
func (w *Purser) Store(ctx context.Context, namespace string, plaintext string) (string, error) {
	if !utf8.ValidString(plaintext) {
		return "", perrors.ErrInvalidUTF8
	}
	return w.Purser.Store(ctx, namespace, []byte(plaintext))
}

// Retrieve delegates to the inner contract.Purser.Retrieve and returns UTF-8 text, or ErrInvalidUTF8
// if the decrypted bytes are not valid UTF-8.
func (w *Purser) Retrieve(ctx context.Context, namespace, identifier string) (string, error) {
	b, err := w.Purser.Retrieve(ctx, namespace, identifier)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", perrors.ErrInvalidUTF8
	}
	return string(b), nil
}

// Update rejects non-UTF-8 strings, then delegates to the inner contract.Purser.Update.
func (w *Purser) Update(ctx context.Context, namespace, identifier string, plaintext string) error {
	if !utf8.ValidString(plaintext) {
		return perrors.ErrInvalidUTF8
	}
	return w.Purser.Update(ctx, namespace, identifier, []byte(plaintext))
}

// CompareAndSwap replaces the row only if decrypted plaintext matches currentPlain, then stores newPlain.
// currentPlain and newPlain must be valid UTF-8 strings.
func (w *Purser) CompareAndSwap(ctx context.Context, namespace, identifier string, currentPlain, newPlain string) error {
	if !utf8.ValidString(currentPlain) {
		return perrors.ErrInvalidUTF8
	}
	if !utf8.ValidString(newPlain) {
		return perrors.ErrInvalidUTF8
	}
	return w.Purser.CompareAndSwap(ctx, namespace, identifier, []byte(currentPlain), []byte(newPlain))
}
