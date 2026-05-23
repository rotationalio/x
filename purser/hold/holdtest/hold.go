// Package holdtest provides conformance helpers for hold and identifier implementations.
//
// HoldConforms stores nulllocker-shaped wire blobs so ciphertext is compatible with
// Purser key-id routing. Use [Ciphertext] for the same fixture in integration tests.

package holdtest

// Hold conformance helpers for tests.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/contract"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
)

//=============================================================================
// Public conformance suite
//=============================================================================

// HoldConforms runs every documented contract from the Hold interface against the
// implementation produced by newHold. Each check builds a fresh hold so checks
// remain independent. Stored ciphertext is nulllocker wire with an extractable key id.
func HoldConforms(t *testing.T, newHold func(*testing.T) hold.Hold) {
	t.Helper()
	ctx := context.Background()
	lck := newFixtureLocker(t)

	checks := []struct {
		name string
		fn   func(context.Context, hold.Hold, contract.Locker) error
	}{
		{"create_get_roundtrip", checkHoldCreateGetRoundtrip},
		{"create_duplicate", checkHoldCreateDuplicate},
		{"cas", checkHoldCompareAndSwap},
		{"replace_missing", checkHoldReplaceMissing},
		{"get_missing", checkHoldGetMissing},
		{"delete_idempotent", checkHoldDeleteIdempotent},
		{"cas_missing", checkHoldCASMissing},
		{"namespace_isolation", checkHoldNamespaceIsolation},
		{"replace_updates_data", checkHoldReplaceUpdatesData},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			assert.Ok(t, c.fn(ctx, newHold(t), lck))
		})
	}
}

//=============================================================================
// Checks (one invariant per function so failure messages pinpoint the contract)
//=============================================================================

// checkHoldCreateGetRoundtrip verifies Create then Get returns the same ciphertext.
func checkHoldCreateGetRoundtrip(ctx context.Context, h hold.Hold, lck contract.Locker) error {
	wire, err := seal(lck, "ns", []byte("cipher-a"))
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}
	id, err := h.Create(ctx, "ns", wire)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	got, err := h.Get(ctx, "ns", id)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	if !bytes.Equal(wire, got) {
		return fmt.Errorf("get: got %d bytes want %d bytes", len(got), len(wire))
	}
	return nil
}

// checkHoldCreateDuplicate verifies CreateWithIdentifier duplicate behavior.
func checkHoldCreateDuplicate(ctx context.Context, h hold.Hold, lck contract.Locker) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}
	wireX, err := seal(lck, "n", []byte("x"))
	if err != nil {
		return fmt.Errorf("seal x: %w", err)
	}
	if err := h.CreateWithIdentifier(ctx, "n", id, wireX); err != nil {
		return fmt.Errorf("first create: %w", err)
	}
	wireY, err := seal(lck, "n", []byte("y"))
	if err != nil {
		return fmt.Errorf("seal y: %w", err)
	}
	err = h.CreateWithIdentifier(ctx, "n", id, wireY)
	if err == nil {
		return fmt.Errorf("second create: got nil want error")
	}
	if !errors.Is(err, perrors.ErrDuplicateKey) {
		return fmt.Errorf("second create: got %v want %w", err, perrors.ErrDuplicateKey)
	}
	return nil
}

// checkHoldCompareAndSwap verifies CAS behavior.
func checkHoldCompareAndSwap(ctx context.Context, h hold.Hold, lck contract.Locker) error {
	oldWire, err := seal(lck, "n", []byte("old"))
	if err != nil {
		return fmt.Errorf("seal old: %w", err)
	}
	id, err := h.Create(ctx, "n", oldWire)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	newWire, err := seal(lck, "n", []byte("new"))
	if err != nil {
		return fmt.Errorf("seal new: %w", err)
	}
	if err := h.CompareAndSwap(ctx, "n", id, oldWire, newWire); err != nil {
		return fmt.Errorf("cas: %w", err)
	}
	badWire, err := seal(lck, "n", []byte("x"))
	if err != nil {
		return fmt.Errorf("seal x: %w", err)
	}
	err = h.CompareAndSwap(ctx, "n", id, oldWire, badWire)
	if !errors.Is(err, perrors.ErrCASFailed) {
		return fmt.Errorf("cas wrong old: got %v want %w", err, perrors.ErrCASFailed)
	}
	return nil
}

// checkHoldReplaceMissing verifies Replace on a missing row returns ErrNotFound.
func checkHoldReplaceMissing(ctx context.Context, h hold.Hold, lck contract.Locker) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}
	wire, err := seal(lck, "ns", []byte("data"))
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}
	err = h.Replace(ctx, "ns", id, wire)
	if err == nil {
		return fmt.Errorf("replace missing: got nil want error")
	}
	if !errors.Is(err, perrors.ErrNotFound) {
		return fmt.Errorf("replace missing: got %v want %w", err, perrors.ErrNotFound)
	}
	return nil
}

// checkHoldGetMissing verifies Get on a missing row returns ErrNotFound.
func checkHoldGetMissing(ctx context.Context, h hold.Hold, _ contract.Locker) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}
	_, err = h.Get(ctx, "ns", id)
	if err == nil {
		return fmt.Errorf("get missing: got nil want error")
	}
	if !errors.Is(err, perrors.ErrNotFound) {
		return fmt.Errorf("get missing: got %v want %w", err, perrors.ErrNotFound)
	}
	return nil
}

// checkHoldDeleteIdempotent verifies Delete on a missing row returns nil.
func checkHoldDeleteIdempotent(ctx context.Context, h hold.Hold, lck contract.Locker) error {
	wire, err := seal(lck, "ns", []byte("data"))
	if err != nil {
		return fmt.Errorf("seal: %w", err)
	}
	id, err := h.Create(ctx, "ns", wire)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if err := h.Delete(ctx, "ns", id); err != nil {
		return fmt.Errorf("first delete: %w", err)
	}
	if err := h.Delete(ctx, "ns", id); err != nil {
		return fmt.Errorf("second delete (should be nil): %w", err)
	}
	return nil
}

// checkHoldCASMissing verifies CAS on a missing row returns ErrNotFound.
func checkHoldCASMissing(ctx context.Context, h hold.Hold, lck contract.Locker) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}
	oldWire, err := seal(lck, "ns", []byte("old"))
	if err != nil {
		return fmt.Errorf("seal old: %w", err)
	}
	newWire, err := seal(lck, "ns", []byte("new"))
	if err != nil {
		return fmt.Errorf("seal new: %w", err)
	}
	err = h.CompareAndSwap(ctx, "ns", id, oldWire, newWire)
	if err == nil {
		return fmt.Errorf("cas missing: got nil want error")
	}
	if !errors.Is(err, perrors.ErrNotFound) {
		return fmt.Errorf("cas missing: got %v want %w", err, perrors.ErrNotFound)
	}
	return nil
}

// checkHoldNamespaceIsolation verifies that rows in different namespaces are independent.
func checkHoldNamespaceIsolation(ctx context.Context, h hold.Hold, lck contract.Locker) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}

	wireA, err := seal(lck, "ns-a", []byte("alpha"))
	if err != nil {
		return fmt.Errorf("seal ns-a: %w", err)
	}
	if err := h.CreateWithIdentifier(ctx, "ns-a", id, wireA); err != nil {
		return fmt.Errorf("create ns-a: %w", err)
	}
	wireB, err := seal(lck, "ns-b", []byte("beta"))
	if err != nil {
		return fmt.Errorf("seal ns-b: %w", err)
	}
	if err := h.CreateWithIdentifier(ctx, "ns-b", id, wireB); err != nil {
		return fmt.Errorf("create ns-b: %w", err)
	}

	gotA, err := h.Get(ctx, "ns-a", id)
	if err != nil {
		return fmt.Errorf("get ns-a: %w", err)
	}
	if !bytes.Equal(gotA, wireA) {
		return fmt.Errorf("ns-a: got %d bytes want %d bytes", len(gotA), len(wireA))
	}

	gotB, err := h.Get(ctx, "ns-b", id)
	if err != nil {
		return fmt.Errorf("get ns-b: %w", err)
	}
	if !bytes.Equal(gotB, wireB) {
		return fmt.Errorf("ns-b: got %d bytes want %d bytes", len(gotB), len(wireB))
	}

	if err := h.Delete(ctx, "ns-a", id); err != nil {
		return fmt.Errorf("delete ns-a: %w", err)
	}
	if _, err := h.Get(ctx, "ns-b", id); err != nil {
		return fmt.Errorf("get ns-b after delete ns-a: %w", err)
	}
	return nil
}

// checkHoldReplaceUpdatesData verifies Replace changes the stored data.
func checkHoldReplaceUpdatesData(ctx context.Context, h hold.Hold, lck contract.Locker) error {
	origWire, err := seal(lck, "ns", []byte("original"))
	if err != nil {
		return fmt.Errorf("seal original: %w", err)
	}
	id, err := h.Create(ctx, "ns", origWire)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	updatedWire, err := seal(lck, "ns", []byte("updated"))
	if err != nil {
		return fmt.Errorf("seal updated: %w", err)
	}
	if err := h.Replace(ctx, "ns", id, updatedWire); err != nil {
		return fmt.Errorf("replace: %w", err)
	}
	got, err := h.Get(ctx, "ns", id)
	if err != nil {
		return fmt.Errorf("get after replace: %w", err)
	}
	if !bytes.Equal(got, updatedWire) {
		return fmt.Errorf("get after replace: got %d bytes want %d bytes", len(got), len(updatedWire))
	}
	return nil
}
