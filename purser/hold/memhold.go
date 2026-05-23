package hold

// In-memory Hold using sync.Map for concurrent tests and examples.

import (
	"context"
	"errors"
	"sync"
	"testing"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold/identifier"
)

// MemHold is an in-memory Hold. Ciphertext values are opaque bytes;
// internally they are stored as strings so [sync.Map.CompareAndSwap] compares
// blob content correctly (distinct []byte snapshots are not comparable via
// interface equality).
type MemHold struct {
	i identifier.Identifier
	m sync.Map // mapKey -> string (opaque blob)
}

// MemHold implements Hold.
var _ Hold = (*MemHold)(nil)

// NewMemHold returns an empty MemHold ready for use. Single-import clients call [purser.NewMemHold].
func NewMemHold(i identifier.Identifier) (*MemHold, error) {
	if i == nil {
		return nil, perrors.ErrInvalidNewArgs
	}
	return &MemHold{i: i}, nil
}

// Identifier returns the configured identifier strategy.
func (h *MemHold) Identifier() identifier.Identifier {
	if h == nil {
		return nil
	}
	return h.i
}

// Create inserts a new row with a generated identifier.
func (h *MemHold) Create(ctx context.Context, namespace string, ciphertext []byte) (identifier string, err error) {
	if h == nil || h.i == nil {
		return "", perrors.ErrInvalidNewArgs
	}
	identifier, err = h.i.New()
	if err != nil {
		return "", errors.Join(perrors.ErrInvalidIdentifier, err)
	}
	err = h.CreateWithIdentifier(ctx, namespace, identifier, ciphertext)
	return identifier, err
}

// CreateWithIdentifier inserts a new row; duplicate (namespace, identifier) returns ErrDuplicateKey.
func (h *MemHold) CreateWithIdentifier(ctx context.Context, namespace, identifier string, ciphertext []byte) error {
	_ = ctx
	if h == nil || h.i == nil {
		return perrors.ErrInvalidNewArgs
	}
	if err := h.i.Parse(identifier); err != nil {
		return errors.Join(perrors.ErrInvalidIdentifier, err)
	}
	k := mapKey{ns: namespace, id: identifier}
	val := string(ciphertext)
	_, loaded := h.m.LoadOrStore(k, val)
	if loaded {
		return perrors.ErrDuplicateKey
	}
	return nil
}

// Get returns a fresh copy of the blob or ErrNotFound.
func (h *MemHold) Get(ctx context.Context, namespace, identifier string) ([]byte, error) {
	_ = ctx
	if h == nil || h.i == nil {
		return nil, perrors.ErrInvalidNewArgs
	}
	if err := h.i.Parse(identifier); err != nil {
		return nil, errors.Join(perrors.ErrInvalidIdentifier, err)
	}
	k := mapKey{ns: namespace, id: identifier}
	v, ok := h.m.Load(k)
	if !ok {
		return nil, perrors.ErrNotFound
	}
	return []byte(v.(string)), nil
}

// Replace overwrites ciphertext for an existing row; missing row returns ErrNotFound.
func (h *MemHold) Replace(ctx context.Context, namespace, identifier string, ciphertext []byte) error {
	_ = ctx
	if h == nil || h.i == nil {
		return perrors.ErrInvalidNewArgs
	}
	if err := h.i.Parse(identifier); err != nil {
		return errors.Join(perrors.ErrInvalidIdentifier, err)
	}
	k := mapKey{ns: namespace, id: identifier}
	if _, ok := h.m.Load(k); !ok {
		return perrors.ErrNotFound
	}
	h.m.Store(k, string(ciphertext))
	return nil
}

// Delete removes a row if present; missing key is a no-op (implementation always returns nil).
func (h *MemHold) Delete(ctx context.Context, namespace, identifier string) error {
	_ = ctx
	if h == nil || h.i == nil {
		return perrors.ErrInvalidNewArgs
	}
	if err := h.i.Parse(identifier); err != nil {
		return errors.Join(perrors.ErrInvalidIdentifier, err)
	}
	h.m.Delete(mapKey{ns: namespace, id: identifier})
	return nil
}

// CompareAndSwap sets newCiphertext only when the stored blob equals oldCiphertext.
// Wrong old value returns ErrCASFailed; missing row returns ErrNotFound.
func (h *MemHold) CompareAndSwap(ctx context.Context, namespace, identifier string, oldCiphertext, newCiphertext []byte) error {
	_ = ctx
	if h == nil || h.i == nil {
		return perrors.ErrInvalidNewArgs
	}
	if err := h.i.Parse(identifier); err != nil {
		return errors.Join(perrors.ErrInvalidIdentifier, err)
	}
	k := mapKey{ns: namespace, id: identifier}
	wantOld := string(oldCiphertext)
	next := string(newCiphertext)

	// sync.Map compares the stored string to wantOld; distinct []byte snapshots
	// are not comparable as interface values.
	if h.m.CompareAndSwap(k, wantOld, next) {
		return nil
	}

	// If the CAS failed determine for which reason.
	if _, ok := h.m.Load(k); !ok {
		return perrors.ErrNotFound
	}
	return perrors.ErrCASFailed
}

// BypassSemanticsSetBlobForTest overwrites the stored blob for (namespace, identifier)
// without checking row existence or duplicate semantics; for tests that need a
// corrupt or synthetic ciphertext.
func (h *MemHold) BypassSemanticsSetBlobForTest(tb testing.TB, namespace, identifier string, ciphertext []byte) {
	tb.Helper()

	k := mapKey{ns: namespace, id: identifier}
	h.m.Store(k, string(ciphertext))
}

// mapKey is the key type for sync.Map in MemHold.
type mapKey struct {
	ns, id string
}
