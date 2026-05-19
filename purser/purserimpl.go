package purser

// This file defines the New constructor and row operation methods that connect
// locker crypto to hold persistence and key routing.

import (
	"bytes"
	"context"
	"errors"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
)

//=============================================================================
// Purser implementation
//=============================================================================

// purserImpl implements Purser using a Hold and Keyring.
type purserImpl struct {
	h  hold.Hold
	kr Keyring
}

// Ensure purserImpl implements Purser.
var _ Purser = (*purserImpl)(nil)

// New constructs a Purser from a Hold and Keyring.
func New(h hold.Hold, kr Keyring) (Purser, error) {
	if h == nil || kr == nil || kr.Active() == nil {
		return nil, perrors.ErrInvalidNewArgs
	}
	return &purserImpl{h: h, kr: kr}, nil
}

// Store encrypts plaintext with the active locker and persists a new row.
func (p *purserImpl) Store(ctx context.Context, namespace string, plaintext []byte) (identifier string, err error) {
	if p == nil {
		return "", perrors.ErrNilPurser
	}

	var wire []byte
	wire, err = p.kr.Active().Seal(namespace, plaintext)
	if err != nil {
		return "", err
	}

	if identifier, err = p.h.Create(ctx, namespace, wire); err != nil {
		return "", errors.Join(perrors.ErrHold, err)
	}

	return identifier, nil
}

// Retrieve loads ciphertext for (namespace, identifier), routes by key id, and decrypts to plaintext.
func (p *purserImpl) Retrieve(ctx context.Context, namespace, identifier string) (plaintext []byte, err error) {
	if p == nil {
		return nil, perrors.ErrNilPurser
	}

	var wire []byte
	if wire, err = p.h.Get(ctx, namespace, identifier); err != nil {
		return nil, errors.Join(perrors.ErrHold, err)
	}

	return p.openForNamespace(namespace, wire)
}

// Update replaces plaintext for an existing row using active-key re-encryption.
func (p *purserImpl) Update(ctx context.Context, namespace, identifier string, plaintext []byte) error {
	if p == nil {
		return perrors.ErrNilPurser
	}

	var wire []byte
	var err error

	if wire, err = p.kr.Active().Seal(namespace, plaintext); err != nil {
		return err
	}

	if err = p.h.Replace(ctx, namespace, identifier, wire); err != nil {
		return errors.Join(perrors.ErrHold, err)
	}

	return nil
}

// CompareAndSwap decrypts current ciphertext, checks currentPlain, then writes newPlain
// with the active locker on success.
func (p *purserImpl) CompareAndSwap(ctx context.Context, namespace, identifier string, currentPlain, newPlain []byte) error {
	if p == nil {
		return perrors.ErrNilPurser
	}

	var oldWire []byte
	var plain []byte
	var newWire []byte
	var err error

	if oldWire, err = p.h.Get(ctx, namespace, identifier); err != nil {
		return errors.Join(perrors.ErrHold, err)
	}

	if plain, err = p.openForNamespace(namespace, oldWire); err != nil {
		return err
	}
	defer Zero(plain)

	if !bytes.Equal(plain, currentPlain) {
		return perrors.ErrWrongCurrent
	}

	if newWire, err = p.kr.Active().Seal(namespace, newPlain); err != nil {
		return err
	}

	if err = p.h.CompareAndSwap(ctx, namespace, identifier, oldWire, newWire); err != nil {
		return errors.Join(perrors.ErrHold, err)
	}

	return nil
}

// MoveNamespace decrypts under oldNamespace, re-seals under newNamespace with the active locker,
// writes the new row, and then deletes the old row.
func (p *purserImpl) MoveNamespace(ctx context.Context, oldNamespace, newNamespace, identifier string) error {
	if p == nil {
		return perrors.ErrNilPurser
	}

	if oldNamespace == newNamespace {
		return nil
	}

	var oldWire []byte
	var plain []byte
	var newWire []byte
	var err error

	if oldWire, err = p.h.Get(ctx, oldNamespace, identifier); err != nil {
		return errors.Join(perrors.ErrHold, err)
	}

	if plain, err = p.openForNamespace(oldNamespace, oldWire); err != nil {
		return err
	}
	defer Zero(plain)

	if newWire, err = p.kr.Active().Seal(newNamespace, plain); err != nil {
		return err
	}

	if err = p.h.CreateWithIdentifier(ctx, newNamespace, identifier, newWire); err != nil {
		return errors.Join(perrors.ErrHold, err)
	}

	if err = p.h.Delete(ctx, oldNamespace, identifier); err != nil {
		return errors.Join(perrors.ErrMoveNamespaceIncomplete, perrors.ErrHold, err)
	}

	return nil
}

// Delete removes a row for (namespace, identifier).
func (p *purserImpl) Delete(ctx context.Context, namespace, identifier string) error {
	if p == nil {
		return perrors.ErrNilPurser
	}

	if err := p.h.Delete(ctx, namespace, identifier); err != nil {
		return errors.Join(perrors.ErrHold, err)
	}
	return nil
}

// openForNamespace routes to the correct locker via the keyring and opens ciphertext.
// RouteKeyID tries each registered locker's ParseKeyID so that multiple wire formats
// (different locker versions) can coexist in one keyring.
func (p *purserImpl) openForNamespace(namespace string, wire []byte) (plain []byte, err error) {
	var lck Locker
	if lck, err = p.kr.RouteKeyID(wire); err != nil {
		return nil, err
	}
	return lck.Open(namespace, wire)
}
