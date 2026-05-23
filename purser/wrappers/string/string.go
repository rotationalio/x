/*
Package stringpurser wraps [purser.Purser] with a string-shaped UTF-8 API.

Construct the inner purser with [purser.New], [purser.NewMemHold], [purser.NewMemring], and related
root aliases; build keys with [purser.NewPassword] and [purser.Keyring.Register]. Classify errors with purser/errors.
*/
package stringpurser

import (
	"context"
	"unicode/utf8"

	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
)

// Purser embeds a [purser.Purser] and enforces UTF-8 on string plaintext.
type Purser struct {
	*purser.Purser
}

// New wraps a non-nil [purser.Purser].
func New(p *purser.Purser) (*Purser, error) {
	if p == nil {
		return nil, perrors.ErrInvalidNewArgs
	}
	return &Purser{Purser: p}, nil
}

// Store rejects non-UTF-8 strings, then delegates to the inner [Purser.Store].
func (w *Purser) Store(ctx context.Context, namespace string, plaintext string) (string, error) {
	if !utf8.ValidString(plaintext) {
		return "", perrors.ErrInvalidUTF8
	}

	p, err := w.inner()
	if err != nil {
		return "", err
	}

	res, err := p.Store(ctx, namespace, []byte(plaintext))
	return res.ID, err
}

// Retrieve returns decrypted plaintext as a UTF-8 string.
func (w *Purser) Retrieve(ctx context.Context, namespace, identifier string) (string, error) {
	p, err := w.inner()
	if err != nil {
		return "", err
	}

	b, err := p.Retrieve(ctx, namespace, identifier)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", perrors.ErrInvalidUTF8
	}
	return string(b), nil
}

// Update rejects non-UTF-8 strings, then delegates to the inner [Purser.Update].
func (w *Purser) Update(ctx context.Context, namespace, identifier string, plaintext string) error {
	if !utf8.ValidString(plaintext) {
		return perrors.ErrInvalidUTF8
	}

	p, err := w.inner()
	if err != nil {
		return err
	}

	_, err = p.Update(ctx, namespace, identifier, []byte(plaintext))
	return err
}

// CompareAndSwap requires UTF-8 current and new plaintext strings.
func (w *Purser) CompareAndSwap(ctx context.Context, namespace, identifier string, currentPlain, newPlain string) (purser.Result, error) {
	if !utf8.ValidString(currentPlain) {
		return purser.Result{}, perrors.ErrInvalidUTF8
	}
	if !utf8.ValidString(newPlain) {
		return purser.Result{}, perrors.ErrInvalidUTF8
	}

	p, err := w.inner()
	if err != nil {
		return purser.Result{}, err
	}
	return p.CompareAndSwap(ctx, namespace, identifier, []byte(currentPlain), []byte(newPlain))
}

// inner returns the embedded purser or an error when the wrapper or inner pointer is nil.
func (w *Purser) inner() (*purser.Purser, error) {
	if w == nil || w.Purser == nil {
		return nil, perrors.ErrNilPurser
	}
	return w.Purser, nil
}
