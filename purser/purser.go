/*
Package purser wires Hold persistence, Keyring routing, and Locker crypto into row operations.

Single-import clients use aliases in aliases.go: [Locker], [Keyring], [Hold],
[EditionV1], [HexIdentifier], [NewMemHold], [NewMemring], [Params],
[NewPassword], and related symbols. Use keyring/registry for FromPassword and ParseKeyID.
*/
package purser

import (
	"bytes"
	"context"
	"errors"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/internal/memzero"
	"go.rtnl.ai/x/purser/keyring"
	"go.rtnl.ai/x/purser/locker"
)

// Purser seals and opens rows via Hold and Keyring.
type Purser struct {
	h  hold.Hold
	kr keyring.Keyring
}

// New constructs a Purser from Hold and Keyring.
func New(h hold.Hold, kr keyring.Keyring) (*Purser, error) {
	if h == nil || kr == nil {
		return nil, perrors.ErrInvalidNewArgs
	}
	return &Purser{h: h, kr: kr}, nil
}

// Keyring returns the keyring passed to New (mutable).
func (p *Purser) Keyring() keyring.Keyring {
	return p.kr
}

// Store seals plaintext and persists a new secret.
func (p *Purser) Store(ctx context.Context, namespace string, plaintext []byte) (Result, error) {
	lck, wire, err := p.seal(namespace, plaintext)
	if err != nil {
		return Result{}, err
	}

	id, err := p.h.Create(ctx, namespace, wire)
	if err != nil {
		return Result{}, errors.Join(perrors.ErrHold, err)
	}
	return newResult(namespace, id, lck), nil
}

// Retrieve loads and opens a secret.
func (p *Purser) Retrieve(ctx context.Context, namespace, identifier string) ([]byte, error) {
	wire, err := p.h.Get(ctx, namespace, identifier)
	if err != nil {
		return nil, errors.Join(perrors.ErrHold, err)
	}
	return p.open(namespace, wire)
}

// Update replaces the secret with new plaintext.
func (p *Purser) Update(ctx context.Context, namespace, identifier string, plaintext []byte) (Result, error) {
	lck, wire, err := p.seal(namespace, plaintext)
	if err != nil {
		return Result{}, err
	}
	if err = p.h.Replace(ctx, namespace, identifier, wire); err != nil {
		return Result{}, errors.Join(perrors.ErrHold, err)
	}
	return newResult(namespace, identifier, lck), nil
}

// CompareAndSwap replaces the secret atomically when the current plaintext matches.
func (p *Purser) CompareAndSwap(ctx context.Context, namespace, identifier string, currentPlain, newPlain []byte) (Result, error) {
	oldWire, err := p.h.Get(ctx, namespace, identifier)
	if err != nil {
		return Result{}, errors.Join(perrors.ErrHold, err)
	}

	plain, err := p.open(namespace, oldWire)
	if err != nil {
		return Result{}, err
	}
	defer memzero.Zero(plain)

	if !bytes.Equal(plain, currentPlain) {
		return Result{}, perrors.ErrWrongCurrent
	}

	lck, newWire, err := p.seal(namespace, newPlain)
	if err != nil {
		return Result{}, err
	}
	if err = p.h.CompareAndSwap(ctx, namespace, identifier, oldWire, newWire); err != nil {
		return Result{}, errors.Join(perrors.ErrHold, err)
	}
	return newResult(namespace, identifier, lck), nil
}

// MoveNamespace re-seals under newNamespace and deletes the old secret.
func (p *Purser) MoveNamespace(ctx context.Context, oldNamespace, newNamespace, identifier string) error {
	if oldNamespace == newNamespace {
		return nil
	}

	oldWire, err := p.h.Get(ctx, oldNamespace, identifier)
	if err != nil {
		return errors.Join(perrors.ErrHold, err)
	}

	plain, err := p.open(oldNamespace, oldWire)
	if err != nil {
		return err
	}
	defer memzero.Zero(plain)

	_, newWire, err := p.seal(newNamespace, plain)
	if err != nil {
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

// Delete removes a secret. Is idempotent.
func (p *Purser) Delete(ctx context.Context, namespace, identifier string) error {
	if err := p.h.Delete(ctx, namespace, identifier); err != nil {
		return errors.Join(perrors.ErrHold, err)
	}
	return nil
}

//=============================================================================
// Helpers
//=============================================================================

// seal picks the namespace locker and returns sealed wire.
func (p *Purser) seal(namespace string, plaintext []byte) (locker.Locker, []byte, error) {
	lck, err := p.kr.LockerFor(namespace)
	if err != nil {
		return nil, nil, err
	}
	wire, err := lck.Seal(namespace, plaintext)
	if err != nil {
		return nil, nil, err
	}
	return lck, wire, nil
}

// open routes wire to a locker and decrypts under namespace.
func (p *Purser) open(namespace string, wire []byte) ([]byte, error) {
	lck, err := p.kr.Route(wire)
	if err != nil {
		return nil, err
	}
	return lck.Open(namespace, wire)
}

//=============================================================================
// Result
//=============================================================================

// Result carries non-secret metadata from [Purser.Store], [Purser.Update],
// and [Purser.CompareAndSwap] on success. Failed CompareAndSwap calls return a zero Result.
type Result struct {
	ID        string
	Namespace string
	KeyID     []byte
	Edition   string
}

// newResult builds Result metadata from a sealed row.
func newResult(namespace, id string, lck locker.Locker) Result {
	return Result{
		ID:        id,
		Namespace: namespace,
		KeyID:     append([]byte(nil), lck.KeyID()...),
		Edition:   lck.Edition(),
	}
}
