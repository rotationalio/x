package holdtest

// Negative conformance tests for the Hold contract (see [hold.Hold]).
//
// The helpers in hold.go (checkHold…, HoldConforms) encode contracts that real Hold
// implementations such as MemHold must satisfy. Each subtest here wires a deliberately
// broken fake into one of those checks and asserts the check returns a non-nil error —
// proving the check would catch a non-conforming implementation.

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/hold/identifier"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"

	perrors "go.rtnl.ai/x/purser/errors"
)

//=============================================================================
// Tests: negative Hold conformance
//=============================================================================

// TestHoldConformance_negative runs table-style subtests; each pairs a deliberately broken
// hold implementation with a conformance check that should detect the defect.
func TestHoldConformance_negative(t *testing.T) {
	ctx := context.Background()
	lck := newFixtureLocker(t)

	t.Run("get_returns_wrong_data", func(t *testing.T) {
		// A hold that returns "wrong" for every Get must fail the round-trip check.
		err := checkHoldCreateGetRoundtrip(ctx, newGetWrongHold(), lck)
		assert.Error(t, err, "expected conformance check to fail")
	})

	t.Run("create_duplicate_no_error", func(t *testing.T) {
		// A hold that silently overwrites on duplicate create must fail the duplicate check.
		err := checkHoldCreateDuplicate(ctx, newDuplicatePermissiveHold(), lck)
		assert.Error(t, err, "expected conformance check to fail for permissive duplicate")
	})

	t.Run("cas_wrong_old_no_error", func(t *testing.T) {
		// A hold whose CAS always succeeds even with wrong old value must fail the CAS check.
		err := checkHoldCompareAndSwap(ctx, newCASAlwaysSucceedHold(), lck)
		assert.Error(t, err, "expected conformance check to fail for permissive CAS")
	})

	t.Run("replace_missing_no_error", func(t *testing.T) {
		// A hold that doesn't return ErrNotFound on replace of missing row must fail.
		err := checkHoldReplaceMissing(ctx, newReplacePermissiveHold(), lck)
		assert.Error(t, err, "expected conformance check to fail for permissive Replace")
	})

	t.Run("get_missing_no_error", func(t *testing.T) {
		// A hold that returns nil,nil for missing rows must fail.
		err := checkHoldGetMissing(ctx, newGetMissingPermissiveHold(), lck)
		assert.Error(t, err, "expected conformance check to fail for permissive Get")
	})

	t.Run("delete_idempotent_fails", func(t *testing.T) {
		// A hold that returns an error on double-delete must fail the delete idempotency check.
		err := checkHoldDeleteIdempotent(ctx, newDeleteNonIdempotentHold(), lck)
		assert.Error(t, err, "expected conformance check to fail for non-idempotent Delete")
	})

	t.Run("cas_missing_no_error", func(t *testing.T) {
		// A hold whose CAS on a missing key returns nil must fail.
		err := checkHoldCASMissing(ctx, newCASMissingPermissiveHold(), lck)
		assert.Error(t, err, "expected conformance check to fail for permissive CAS on missing")
	})

	t.Run("namespace_isolation", func(t *testing.T) {
		// A hold that ignores namespaces (keys by ID only) must fail the isolation check.
		err := checkHoldNamespaceIsolation(ctx, newNamespaceBlindHold(), lck)
		assert.Error(t, err, "expected conformance check to fail for namespace-blind hold")
	})

	t.Run("replace_updates_data", func(t *testing.T) {
		// A hold that ignores Replace (always returns old data) must fail.
		err := checkHoldReplaceUpdatesData(ctx, newReplaceNoopHold(), lck)
		assert.Error(t, err, "expected conformance check to fail for no-op Replace")
	})
}

//=============================================================================
// Broken hold fakes
//=============================================================================

// getWrongHold always returns "wrong" from Get regardless of what was stored.
type getWrongHold struct{ m map[string][]byte }

func newGetWrongHold() *getWrongHold { return &getWrongHold{m: map[string][]byte{}} }

func (h *getWrongHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *getWrongHold) Create(_ context.Context, ns string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *getWrongHold) CreateWithIdentifier(_ context.Context, ns, id string, ct []byte) error {
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *getWrongHold) Get(_ context.Context, _, _ string) ([]byte, error) {
	return []byte("wrong"), nil
}

func (h *getWrongHold) Replace(_ context.Context, ns, id string, ct []byte) error {
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *getWrongHold) Delete(_ context.Context, ns, id string) error {
	delete(h.m, ns+":"+id)
	return nil
}

func (h *getWrongHold) CompareAndSwap(_ context.Context, ns, id string, oldCt, newCt []byte) error {
	k := ns + ":" + id
	if !bytes.Equal(h.m[k], oldCt) {
		return nil
	}
	h.m[k] = append([]byte(nil), newCt...)
	return nil
}

// duplicatePermissiveHold silently overwrites on duplicate CreateWithIdentifier.
type duplicatePermissiveHold struct{ m map[string][]byte }

func newDuplicatePermissiveHold() *duplicatePermissiveHold {
	return &duplicatePermissiveHold{m: map[string][]byte{}}
}

func (h *duplicatePermissiveHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *duplicatePermissiveHold) Create(_ context.Context, ns string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *duplicatePermissiveHold) CreateWithIdentifier(_ context.Context, ns, id string, ct []byte) error {
	// Broken: silently overwrites instead of returning ErrDuplicateKey.
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *duplicatePermissiveHold) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := h.m[ns+":"+id]
	if !ok {
		return nil, perrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (h *duplicatePermissiveHold) Replace(_ context.Context, ns, id string, ct []byte) error {
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *duplicatePermissiveHold) Delete(_ context.Context, ns, id string) error {
	delete(h.m, ns+":"+id)
	return nil
}

func (h *duplicatePermissiveHold) CompareAndSwap(_ context.Context, ns, id string, oldCt, newCt []byte) error {
	k := ns + ":" + id
	if !bytes.Equal(h.m[k], oldCt) {
		return perrors.ErrCASFailed
	}
	h.m[k] = append([]byte(nil), newCt...)
	return nil
}

// casAlwaysSucceedHold CAS always succeeds even with wrong old value.
type casAlwaysSucceedHold struct{ m map[string][]byte }

func newCASAlwaysSucceedHold() *casAlwaysSucceedHold {
	return &casAlwaysSucceedHold{m: map[string][]byte{}}
}

func (h *casAlwaysSucceedHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *casAlwaysSucceedHold) Create(_ context.Context, ns string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *casAlwaysSucceedHold) CreateWithIdentifier(_ context.Context, ns, id string, ct []byte) error {
	k := ns + ":" + id
	if _, exists := h.m[k]; exists {
		return perrors.ErrDuplicateKey
	}
	h.m[k] = append([]byte(nil), ct...)
	return nil
}

func (h *casAlwaysSucceedHold) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := h.m[ns+":"+id]
	if !ok {
		return nil, perrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (h *casAlwaysSucceedHold) Replace(_ context.Context, ns, id string, ct []byte) error {
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *casAlwaysSucceedHold) Delete(_ context.Context, ns, id string) error {
	delete(h.m, ns+":"+id)
	return nil
}

func (h *casAlwaysSucceedHold) CompareAndSwap(_ context.Context, ns, id string, _, newCt []byte) error {
	// Broken: always succeeds regardless of old value.
	h.m[ns+":"+id] = append([]byte(nil), newCt...)
	return nil
}

// replacePermissiveHold does not return ErrNotFound when replacing a missing row.
type replacePermissiveHold struct{ m map[string][]byte }

func newReplacePermissiveHold() *replacePermissiveHold {
	return &replacePermissiveHold{m: map[string][]byte{}}
}

func (h *replacePermissiveHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *replacePermissiveHold) Create(_ context.Context, ns string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *replacePermissiveHold) CreateWithIdentifier(_ context.Context, ns, id string, ct []byte) error {
	k := ns + ":" + id
	if _, exists := h.m[k]; exists {
		return perrors.ErrDuplicateKey
	}
	h.m[k] = append([]byte(nil), ct...)
	return nil
}

func (h *replacePermissiveHold) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := h.m[ns+":"+id]
	if !ok {
		return nil, perrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (h *replacePermissiveHold) Replace(_ context.Context, ns, id string, ct []byte) error {
	// Broken: silently creates if missing instead of returning ErrNotFound.
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *replacePermissiveHold) Delete(_ context.Context, ns, id string) error {
	delete(h.m, ns+":"+id)
	return nil
}

func (h *replacePermissiveHold) CompareAndSwap(_ context.Context, ns, id string, oldCt, newCt []byte) error {
	k := ns + ":" + id
	if !bytes.Equal(h.m[k], oldCt) {
		return perrors.ErrCASFailed
	}
	h.m[k] = append([]byte(nil), newCt...)
	return nil
}

// getMissingPermissiveHold returns nil,nil instead of ErrNotFound for missing rows.
type getMissingPermissiveHold struct{ m map[string][]byte }

func newGetMissingPermissiveHold() *getMissingPermissiveHold {
	return &getMissingPermissiveHold{m: map[string][]byte{}}
}

func (h *getMissingPermissiveHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *getMissingPermissiveHold) Create(_ context.Context, ns string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *getMissingPermissiveHold) CreateWithIdentifier(_ context.Context, ns, id string, ct []byte) error {
	k := ns + ":" + id
	if _, exists := h.m[k]; exists {
		return perrors.ErrDuplicateKey
	}
	h.m[k] = append([]byte(nil), ct...)
	return nil
}

func (h *getMissingPermissiveHold) Get(_ context.Context, ns, id string) ([]byte, error) {
	v := h.m[ns+":"+id]
	// Broken: returns nil, nil when key is missing.
	return v, nil
}

func (h *getMissingPermissiveHold) Replace(_ context.Context, ns, id string, ct []byte) error {
	if _, ok := h.m[ns+":"+id]; !ok {
		return perrors.ErrNotFound
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *getMissingPermissiveHold) Delete(_ context.Context, ns, id string) error {
	delete(h.m, ns+":"+id)
	return nil
}

func (h *getMissingPermissiveHold) CompareAndSwap(_ context.Context, ns, id string, oldCt, newCt []byte) error {
	k := ns + ":" + id
	if !bytes.Equal(h.m[k], oldCt) {
		return perrors.ErrCASFailed
	}
	h.m[k] = append([]byte(nil), newCt...)
	return nil
}

// deleteNonIdempotentHold returns an error on double-delete instead of nil.
type deleteNonIdempotentHold struct{ m map[string][]byte }

func newDeleteNonIdempotentHold() *deleteNonIdempotentHold {
	return &deleteNonIdempotentHold{m: map[string][]byte{}}
}

func (h *deleteNonIdempotentHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *deleteNonIdempotentHold) Create(_ context.Context, ns string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *deleteNonIdempotentHold) CreateWithIdentifier(_ context.Context, ns, id string, ct []byte) error {
	k := ns + ":" + id
	if _, exists := h.m[k]; exists {
		return perrors.ErrDuplicateKey
	}
	h.m[k] = append([]byte(nil), ct...)
	return nil
}

func (h *deleteNonIdempotentHold) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := h.m[ns+":"+id]
	if !ok {
		return nil, perrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (h *deleteNonIdempotentHold) Replace(_ context.Context, ns, id string, ct []byte) error {
	if _, ok := h.m[ns+":"+id]; !ok {
		return perrors.ErrNotFound
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *deleteNonIdempotentHold) Delete(_ context.Context, ns, id string) error {
	k := ns + ":" + id
	if _, ok := h.m[k]; !ok {
		// Broken: returns error on missing row instead of nil (not idempotent).
		return errors.New("not found")
	}
	delete(h.m, k)
	return nil
}

func (h *deleteNonIdempotentHold) CompareAndSwap(_ context.Context, ns, id string, oldCt, newCt []byte) error {
	k := ns + ":" + id
	if !bytes.Equal(h.m[k], oldCt) {
		return perrors.ErrCASFailed
	}
	h.m[k] = append([]byte(nil), newCt...)
	return nil
}

// casMissingPermissiveHold CAS on a missing key returns nil instead of ErrNotFound.
type casMissingPermissiveHold struct{ m map[string][]byte }

func newCASMissingPermissiveHold() *casMissingPermissiveHold {
	return &casMissingPermissiveHold{m: map[string][]byte{}}
}

func (h *casMissingPermissiveHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *casMissingPermissiveHold) Create(_ context.Context, ns string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *casMissingPermissiveHold) CreateWithIdentifier(_ context.Context, ns, id string, ct []byte) error {
	k := ns + ":" + id
	if _, exists := h.m[k]; exists {
		return perrors.ErrDuplicateKey
	}
	h.m[k] = append([]byte(nil), ct...)
	return nil
}

func (h *casMissingPermissiveHold) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := h.m[ns+":"+id]
	if !ok {
		return nil, perrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (h *casMissingPermissiveHold) Replace(_ context.Context, ns, id string, ct []byte) error {
	if _, ok := h.m[ns+":"+id]; !ok {
		return perrors.ErrNotFound
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return nil
}

func (h *casMissingPermissiveHold) Delete(_ context.Context, ns, id string) error {
	delete(h.m, ns+":"+id)
	return nil
}

func (h *casMissingPermissiveHold) CompareAndSwap(_ context.Context, ns, id string, _, newCt []byte) error {
	// Broken: silently succeeds even when the key is missing.
	h.m[ns+":"+id] = append([]byte(nil), newCt...)
	return nil
}

// namespaceBlindHold ignores the namespace in all operations, keying by ID only.
type namespaceBlindHold struct{ m map[string][]byte }

func newNamespaceBlindHold() *namespaceBlindHold {
	return &namespaceBlindHold{m: map[string][]byte{}}
}

func (h *namespaceBlindHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *namespaceBlindHold) Create(_ context.Context, _ string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	// Broken: ignores namespace.
	h.m[id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *namespaceBlindHold) CreateWithIdentifier(_ context.Context, _, id string, ct []byte) error {
	if _, exists := h.m[id]; exists {
		return perrors.ErrDuplicateKey
	}
	h.m[id] = append([]byte(nil), ct...)
	return nil
}

func (h *namespaceBlindHold) Get(_ context.Context, _, id string) ([]byte, error) {
	v, ok := h.m[id]
	if !ok {
		return nil, perrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (h *namespaceBlindHold) Replace(_ context.Context, _, id string, ct []byte) error {
	if _, ok := h.m[id]; !ok {
		return perrors.ErrNotFound
	}
	h.m[id] = append([]byte(nil), ct...)
	return nil
}

func (h *namespaceBlindHold) Delete(_ context.Context, _, id string) error {
	delete(h.m, id)
	return nil
}

func (h *namespaceBlindHold) CompareAndSwap(_ context.Context, _, id string, oldCt, newCt []byte) error {
	if !bytes.Equal(h.m[id], oldCt) {
		return perrors.ErrCASFailed
	}
	h.m[id] = append([]byte(nil), newCt...)
	return nil
}

// replaceNoopHold ignores Replace calls—stored data never changes.
type replaceNoopHold struct{ m map[string][]byte }

func newReplaceNoopHold() *replaceNoopHold {
	return &replaceNoopHold{m: map[string][]byte{}}
}

func (h *replaceNoopHold) Identifier() identifier.Identifier { return hexid.Identifier{} }

func (h *replaceNoopHold) Create(_ context.Context, ns string, ct []byte) (string, error) {
	id, err := h.Identifier().New()
	if err != nil {
		return "", err
	}
	h.m[ns+":"+id] = append([]byte(nil), ct...)
	return id, nil
}

func (h *replaceNoopHold) CreateWithIdentifier(_ context.Context, ns, id string, ct []byte) error {
	k := ns + ":" + id
	if _, exists := h.m[k]; exists {
		return perrors.ErrDuplicateKey
	}
	h.m[k] = append([]byte(nil), ct...)
	return nil
}

func (h *replaceNoopHold) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := h.m[ns+":"+id]
	if !ok {
		return nil, perrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (h *replaceNoopHold) Replace(_ context.Context, _, _ string, _ []byte) error {
	// Broken: silently ignores the replacement.
	return nil
}

func (h *replaceNoopHold) Delete(_ context.Context, ns, id string) error {
	delete(h.m, ns+":"+id)
	return nil
}

func (h *replaceNoopHold) CompareAndSwap(_ context.Context, ns, id string, oldCt, newCt []byte) error {
	k := ns + ":" + id
	if !bytes.Equal(h.m[k], oldCt) {
		return perrors.ErrCASFailed
	}
	h.m[k] = append([]byte(nil), newCt...)
	return nil
}
