// Package holdtest provides conformance helpers for hold and identifier implementations.

package holdtest

// Hold conformance helpers for tests.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
)

//=============================================================================
// Public conformance suite
//=============================================================================

// HoldConforms runs every documented contract from the Hold interface against the
// implementation produced by newHold. Each check builds a fresh hold so checks
// remain independent.
func HoldConforms(t *testing.T, newHold func(*testing.T) hold.Hold) {
	t.Helper()
	ctx := context.Background()

	checks := []struct {
		name string
		fn   func(context.Context, hold.Hold) error
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
			assert.Ok(t, c.fn(ctx, newHold(t)))
		})
	}
}

//=============================================================================
// Checks (one invariant per function so failure messages pinpoint the contract)
//=============================================================================

// checkHoldCreateGetRoundtrip verifies Create then Get returns the same ciphertext.
func checkHoldCreateGetRoundtrip(ctx context.Context, h hold.Hold) error {
	id, err := h.Create(ctx, "ns", []byte("cipher-a"))
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	got, err := h.Get(ctx, "ns", id)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	if !bytes.Equal([]byte("cipher-a"), got) {
		return fmt.Errorf("get: got %q want %q", got, []byte("cipher-a"))
	}
	return nil
}

// checkHoldCreateDuplicate verifies CreateWithIdentifier duplicate behavior.
func checkHoldCreateDuplicate(ctx context.Context, h hold.Hold) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}
	if err := h.CreateWithIdentifier(ctx, "n", id, []byte("x")); err != nil {
		return fmt.Errorf("first create: %w", err)
	}
	err = h.CreateWithIdentifier(ctx, "n", id, []byte("y"))
	if err == nil {
		return fmt.Errorf("second create: got nil want error")
	}
	if !errors.Is(err, perrors.ErrDuplicateKey) {
		return fmt.Errorf("second create: got %v want %w", err, perrors.ErrDuplicateKey)
	}
	return nil
}

// checkHoldCompareAndSwap verifies CAS behavior.
func checkHoldCompareAndSwap(ctx context.Context, h hold.Hold) error {
	id, err := h.Create(ctx, "n", []byte("old"))
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if err := h.CompareAndSwap(ctx, "n", id, []byte("old"), []byte("new")); err != nil {
		return fmt.Errorf("cas: %w", err)
	}
	err = h.CompareAndSwap(ctx, "n", id, []byte("old"), []byte("x"))
	if !errors.Is(err, perrors.ErrCASFailed) {
		return fmt.Errorf("cas wrong old: got %v want %w", err, perrors.ErrCASFailed)
	}
	return nil
}

// checkHoldReplaceMissing verifies Replace on a missing row returns ErrNotFound.
func checkHoldReplaceMissing(ctx context.Context, h hold.Hold) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}
	err = h.Replace(ctx, "ns", id, []byte("data"))
	if err == nil {
		return fmt.Errorf("replace missing: got nil want error")
	}
	if !errors.Is(err, perrors.ErrNotFound) {
		return fmt.Errorf("replace missing: got %v want %w", err, perrors.ErrNotFound)
	}
	return nil
}

// checkHoldGetMissing verifies Get on a missing row returns ErrNotFound.
func checkHoldGetMissing(ctx context.Context, h hold.Hold) error {
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
func checkHoldDeleteIdempotent(ctx context.Context, h hold.Hold) error {
	id, err := h.Create(ctx, "ns", []byte("data"))
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
func checkHoldCASMissing(ctx context.Context, h hold.Hold) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}
	err = h.CompareAndSwap(ctx, "ns", id, []byte("old"), []byte("new"))
	if err == nil {
		return fmt.Errorf("cas missing: got nil want error")
	}
	if !errors.Is(err, perrors.ErrNotFound) {
		return fmt.Errorf("cas missing: got %v want %w", err, perrors.ErrNotFound)
	}
	return nil
}

// checkHoldNamespaceIsolation verifies that rows in different namespaces are independent.
func checkHoldNamespaceIsolation(ctx context.Context, h hold.Hold) error {
	id, err := h.Identifier().New()
	if err != nil {
		return fmt.Errorf("identifier.New: %w", err)
	}

	// Create the same identifier in two different namespaces.
	if err := h.CreateWithIdentifier(ctx, "ns-a", id, []byte("alpha")); err != nil {
		return fmt.Errorf("create ns-a: %w", err)
	}
	if err := h.CreateWithIdentifier(ctx, "ns-b", id, []byte("beta")); err != nil {
		return fmt.Errorf("create ns-b: %w", err)
	}

	// Verify each namespace returns its own data.
	gotA, err := h.Get(ctx, "ns-a", id)
	if err != nil {
		return fmt.Errorf("get ns-a: %w", err)
	}
	if !bytes.Equal(gotA, []byte("alpha")) {
		return fmt.Errorf("ns-a: got %q want %q", gotA, "alpha")
	}

	gotB, err := h.Get(ctx, "ns-b", id)
	if err != nil {
		return fmt.Errorf("get ns-b: %w", err)
	}
	if !bytes.Equal(gotB, []byte("beta")) {
		return fmt.Errorf("ns-b: got %q want %q", gotB, "beta")
	}

	// Deleting from one namespace must not affect the other.
	if err := h.Delete(ctx, "ns-a", id); err != nil {
		return fmt.Errorf("delete ns-a: %w", err)
	}
	if _, err := h.Get(ctx, "ns-b", id); err != nil {
		return fmt.Errorf("get ns-b after delete ns-a: %w", err)
	}
	return nil
}

// checkHoldReplaceUpdatesData verifies Replace changes the stored data.
func checkHoldReplaceUpdatesData(ctx context.Context, h hold.Hold) error {
	id, err := h.Create(ctx, "ns", []byte("original"))
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if err := h.Replace(ctx, "ns", id, []byte("updated")); err != nil {
		return fmt.Errorf("replace: %w", err)
	}
	got, err := h.Get(ctx, "ns", id)
	if err != nil {
		return fmt.Errorf("get after replace: %w", err)
	}
	if !bytes.Equal(got, []byte("updated")) {
		return fmt.Errorf("get after replace: got %q want %q", got, "updated")
	}
	return nil
}
