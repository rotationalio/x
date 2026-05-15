package vaulttest_test

// Negative conformance tests for the vault Storage contract (see [storage.Storage]).
//
// The helpers in storage.go (CheckStorage…, StorageConforms) encode row semantics
// the vault depends on. Each table row pairs one deliberately broken in-memory fake with the
// specific CheckStorage… function that should reject it. If a check returned nil, the suite would
// not catch that class of storage bug.

import (
	"bytes"
	"context"
	"errors"
	"testing"

	verrors "go.rtnl.ai/x/vault/errors"
	"go.rtnl.ai/x/vault/identifier"
	"go.rtnl.ai/x/vault/storage"
	"go.rtnl.ai/x/vault/vaulttest"
)

//=============================================================================
// Tests: negative storage conformance
//=============================================================================

// TestStorageConformance_negative table-drives broken [storage.Storage] implementations against the
// exported check that is designed to catch each defect.
func TestStorageConformance_negative(t *testing.T) {
	ctx := context.Background()
	idGen := identifier.HexIdentifier{}

	cases := []struct {
		name string
		st   storage.Storage
		fn   func(context.Context, storage.Storage, identifier.Identifier) error
	}{
		{
			name: "create_get_roundtrip",
			st:   newStorGetWrong(),
			fn:   vaulttest.CheckStorageCreateGetRoundtrip,
		},
		{
			name: "create_duplicate",
			st:   newStorAllowDup(),
			fn:   vaulttest.CheckStorageCreateDuplicate,
		},
		{
			// Duplicate create must return errors.Is(err, verrors.ErrDuplicateKey), not a generic error.
			name: "create_duplicate_wrapped_err",
			st:   newStorWrongDupErr(),
			fn:   vaulttest.CheckStorageCreateDuplicate,
		},
		{
			name: "namespace_isolation",
			st:   newStorNSCollide(),
			fn:   vaulttest.CheckStorageNamespaceIsolation,
		},
		{
			name: "get_missing",
			st:   storGetMissingWrong{},
			fn:   vaulttest.CheckStorageGetMissing,
		},
		{
			name: "replace_success",
			st:   newStorReplaceNoop(),
			fn:   vaulttest.CheckStorageReplaceSuccess,
		},
		{
			name: "replace_missing",
			st:   storReplaceMissingOK{},
			fn:   vaulttest.CheckStorageReplaceMissing,
		},
		{
			name: "delete_idempotent",
			st:   storDeleteErrMissing{},
			fn:   vaulttest.CheckStorageDeleteIdempotent,
		},
		{
			name: "delete_existing_then_get_missing",
			st:   newStorDeleteNoop(),
			fn:   vaulttest.CheckStorageDeleteExistingThenGetMissing,
		},
		{
			name: "cas_success_and_conflict",
			st:   newStorCASBlind(),
			fn:   vaulttest.CheckStorageCompareAndSwapSuccessAndConflict,
		},
		{
			name: "cas_missing_row",
			st:   storCASMissingWrong{},
			fn:   vaulttest.CheckStorageCompareAndSwapMissingRow,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Each fake breaks exactly one storage contract; the paired check must return an error.
			if err := tc.fn(ctx, tc.st, idGen); err == nil {
				t.Fatalf("expected non-nil error from check %s", tc.name)
			}
		})
	}
}

//=============================================================================
// In-memory map key layout (correct fakes use this; broken fakes may omit namespace)
//=============================================================================

// storK builds a composite map key so correct implementations never collide across namespaces.
// Broken fakes that key only by id (ignoring ns) are detected by namespace isolation checks.
func storK(ns, id string) string { return ns + "\x00" + id }

//=============================================================================
// Broken storage fakes — each violates one contract checked by CheckStorage…
//=============================================================================

// storAllowDup implements Create as blind insert: duplicate keys overwrite instead of returning
// [verrors.ErrDuplicateKey], so [vaulttest.CheckStorageCreateDuplicate] fails.
type storAllowDup struct{ m map[string][]byte }

func newStorAllowDup() *storAllowDup { return &storAllowDup{m: make(map[string][]byte)} }

func (s *storAllowDup) Create(_ context.Context, ns, id string, ct []byte) error {
	s.m[storK(ns, id)] = append([]byte(nil), ct...)
	return nil
}

func (s *storAllowDup) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := s.m[storK(ns, id)]
	if !ok {
		return nil, verrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (s *storAllowDup) Replace(_ context.Context, ns, id string, ct []byte) error {
	k := storK(ns, id)
	if _, ok := s.m[k]; !ok {
		return verrors.ErrNotFound
	}
	s.m[k] = append([]byte(nil), ct...)
	return nil
}

func (s *storAllowDup) Delete(_ context.Context, ns, id string) error {
	delete(s.m, storK(ns, id))
	return nil
}

func (s *storAllowDup) CompareAndSwap(_ context.Context, ns, id string, old, newCt []byte) error {
	k := storK(ns, id)
	cur, ok := s.m[k]
	if !ok {
		return verrors.ErrNotFound
	}
	if !bytes.Equal(cur, old) {
		return verrors.ErrCASFailed
	}
	s.m[k] = append([]byte(nil), newCt...)
	return nil
}

// storWrongDupErr rejects duplicate creates with a generic error instead of [verrors.ErrDuplicateKey],
// breaking errors.Is classification required by [vaulttest.CheckStorageCreateDuplicate].
type storWrongDupErr struct{ m map[string][]byte }

func newStorWrongDupErr() *storWrongDupErr { return &storWrongDupErr{m: make(map[string][]byte)} }

func (s *storWrongDupErr) Create(_ context.Context, ns, id string, ct []byte) error {
	k := storK(ns, id)
	if _, dup := s.m[k]; dup {
		return errors.New("not a duplicate key error")
	}
	s.m[k] = append([]byte(nil), ct...)
	return nil
}

func (s *storWrongDupErr) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := s.m[storK(ns, id)]
	if !ok {
		return nil, verrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (s *storWrongDupErr) Replace(_ context.Context, ns, id string, ct []byte) error {
	k := storK(ns, id)
	if _, ok := s.m[k]; !ok {
		return verrors.ErrNotFound
	}
	s.m[k] = append([]byte(nil), ct...)
	return nil
}

func (s *storWrongDupErr) Delete(_ context.Context, ns, id string) error {
	delete(s.m, storK(ns, id))
	return nil
}

func (s *storWrongDupErr) CompareAndSwap(_ context.Context, ns, id string, old, newCt []byte) error {
	k := storK(ns, id)
	cur, ok := s.m[k]
	if !ok {
		return verrors.ErrNotFound
	}
	if !bytes.Equal(cur, old) {
		return verrors.ErrCASFailed
	}
	s.m[k] = append([]byte(nil), newCt...)
	return nil
}

// storGetWrong returns a wrong constant payload on Get, failing [vaulttest.CheckStorageCreateGetRoundtrip].
type storGetWrong struct{ m map[string][]byte }

func newStorGetWrong() *storGetWrong { return &storGetWrong{m: make(map[string][]byte)} }

func (s *storGetWrong) Create(_ context.Context, ns, id string, ct []byte) error {
	s.m[storK(ns, id)] = append([]byte(nil), ct...)
	return nil
}

func (s *storGetWrong) Get(_ context.Context, ns, id string) ([]byte, error) {
	if _, ok := s.m[storK(ns, id)]; !ok {
		return nil, verrors.ErrNotFound
	}
	return []byte("wrong"), nil
}

func (s *storGetWrong) Replace(_ context.Context, ns, id string, ct []byte) error {
	k := storK(ns, id)
	if _, ok := s.m[k]; !ok {
		return verrors.ErrNotFound
	}
	s.m[k] = append([]byte(nil), ct...)
	return nil
}

func (s *storGetWrong) Delete(_ context.Context, ns, id string) error {
	delete(s.m, storK(ns, id))
	return nil
}

func (s *storGetWrong) CompareAndSwap(_ context.Context, ns, id string, old, newCt []byte) error {
	k := storK(ns, id)
	cur, ok := s.m[k]
	if !ok {
		return verrors.ErrNotFound
	}
	if !bytes.Equal(cur, old) {
		return verrors.ErrCASFailed
	}
	s.m[k] = append([]byte(nil), newCt...)
	return nil
}

// storNSCollide keys rows by id only, ignoring namespace, so the second namespace overwrites the
// first—[vaulttest.CheckStorageNamespaceIsolation] must fail.
type storNSCollide struct{ m map[string][]byte }

func newStorNSCollide() *storNSCollide { return &storNSCollide{m: make(map[string][]byte)} }

func (s *storNSCollide) Create(_ context.Context, ns, id string, ct []byte) error {
	_ = ns
	s.m[id] = append([]byte(nil), ct...)
	return nil
}

func (s *storNSCollide) Get(_ context.Context, ns, id string) ([]byte, error) {
	_ = ns
	v, ok := s.m[id]
	if !ok {
		return nil, verrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (s *storNSCollide) Replace(_ context.Context, ns, id string, ct []byte) error {
	_ = ns
	if _, ok := s.m[id]; !ok {
		return verrors.ErrNotFound
	}
	s.m[id] = append([]byte(nil), ct...)
	return nil
}

func (s *storNSCollide) Delete(_ context.Context, ns, id string) error {
	_ = ns
	delete(s.m, id)
	return nil
}

func (s *storNSCollide) CompareAndSwap(_ context.Context, ns, id string, old, newCt []byte) error {
	_ = ns
	cur, ok := s.m[id]
	if !ok {
		return verrors.ErrNotFound
	}
	if !bytes.Equal(cur, old) {
		return verrors.ErrCASFailed
	}
	s.m[id] = append([]byte(nil), newCt...)
	return nil
}

// storGetMissingWrong returns (nil, nil) for missing rows instead of [verrors.ErrNotFound], defeating
// [vaulttest.CheckStorageGetMissing].
type storGetMissingWrong struct{}

func (storGetMissingWrong) Create(context.Context, string, string, []byte) error { return nil }

func (storGetMissingWrong) Get(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (storGetMissingWrong) Replace(context.Context, string, string, []byte) error {
	return verrors.ErrNotFound
}

func (storGetMissingWrong) Delete(context.Context, string, string) error { return nil }

func (storGetMissingWrong) CompareAndSwap(context.Context, string, string, []byte, []byte) error {
	return verrors.ErrNotFound
}

// storReplaceNoop is a no-op Replace: ciphertext never updates, so [vaulttest.CheckStorageReplaceSuccess] fails.
type storReplaceNoop struct{ m map[string][]byte }

func newStorReplaceNoop() *storReplaceNoop { return &storReplaceNoop{m: make(map[string][]byte)} }

func (s *storReplaceNoop) Create(_ context.Context, ns, id string, ct []byte) error {
	s.m[storK(ns, id)] = append([]byte(nil), ct...)
	return nil
}

func (s *storReplaceNoop) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := s.m[storK(ns, id)]
	if !ok {
		return nil, verrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (s *storReplaceNoop) Replace(context.Context, string, string, []byte) error { return nil }

func (s *storReplaceNoop) Delete(_ context.Context, ns, id string) error {
	delete(s.m, storK(ns, id))
	return nil
}

func (s *storReplaceNoop) CompareAndSwap(_ context.Context, ns, id string, old, newCt []byte) error {
	k := storK(ns, id)
	cur, ok := s.m[k]
	if !ok {
		return verrors.ErrNotFound
	}
	if !bytes.Equal(cur, old) {
		return verrors.ErrCASFailed
	}
	s.m[k] = append([]byte(nil), newCt...)
	return nil
}

// storReplaceMissingOK returns nil on Replace for a missing row instead of [verrors.ErrNotFound].
type storReplaceMissingOK struct{}

func (storReplaceMissingOK) Create(context.Context, string, string, []byte) error { return nil }

func (storReplaceMissingOK) Get(context.Context, string, string) ([]byte, error) {
	return nil, verrors.ErrNotFound
}

func (storReplaceMissingOK) Replace(context.Context, string, string, []byte) error { return nil }

func (storReplaceMissingOK) Delete(context.Context, string, string) error { return nil }

func (storReplaceMissingOK) CompareAndSwap(context.Context, string, string, []byte, []byte) error {
	return verrors.ErrNotFound
}

// storDeleteErrMissing returns an error on Delete for a missing row; vault requires idempotent delete.
type storDeleteErrMissing struct{}

func (storDeleteErrMissing) Create(context.Context, string, string, []byte) error { return nil }

func (storDeleteErrMissing) Get(context.Context, string, string) ([]byte, error) {
	return nil, verrors.ErrNotFound
}

func (storDeleteErrMissing) Replace(context.Context, string, string, []byte) error {
	return verrors.ErrNotFound
}

func (storDeleteErrMissing) Delete(context.Context, string, string) error {
	return errors.New("delete missing not allowed")
}

func (storDeleteErrMissing) CompareAndSwap(context.Context, string, string, []byte, []byte) error {
	return verrors.ErrNotFound
}

// storDeleteNoop pretends Delete succeeded but leaves the row in the map, so Get still succeeds.
type storDeleteNoop struct{ m map[string][]byte }

func newStorDeleteNoop() *storDeleteNoop { return &storDeleteNoop{m: make(map[string][]byte)} }

func (s *storDeleteNoop) Create(_ context.Context, ns, id string, ct []byte) error {
	s.m[storK(ns, id)] = append([]byte(nil), ct...)
	return nil
}

func (s *storDeleteNoop) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := s.m[storK(ns, id)]
	if !ok {
		return nil, verrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (s *storDeleteNoop) Replace(_ context.Context, ns, id string, ct []byte) error {
	k := storK(ns, id)
	if _, ok := s.m[k]; !ok {
		return verrors.ErrNotFound
	}
	s.m[k] = append([]byte(nil), ct...)
	return nil
}

func (s *storDeleteNoop) Delete(context.Context, string, string) error { return nil }

func (s *storDeleteNoop) CompareAndSwap(_ context.Context, ns, id string, old, newCt []byte) error {
	k := storK(ns, id)
	cur, ok := s.m[k]
	if !ok {
		return verrors.ErrNotFound
	}
	if !bytes.Equal(cur, old) {
		return verrors.ErrCASFailed
	}
	s.m[k] = append([]byte(nil), newCt...)
	return nil
}

// storCASBlind ignores the old ciphertext and always applies the new value, so a stale CAS cannot
// return [verrors.ErrCASFailed]—[vaulttest.CheckStorageCompareAndSwapSuccessAndConflict] fails.
type storCASBlind struct{ m map[string][]byte }

func newStorCASBlind() *storCASBlind { return &storCASBlind{m: make(map[string][]byte)} }

func (s *storCASBlind) Create(_ context.Context, ns, id string, ct []byte) error {
	s.m[storK(ns, id)] = append([]byte(nil), ct...)
	return nil
}

func (s *storCASBlind) Get(_ context.Context, ns, id string) ([]byte, error) {
	v, ok := s.m[storK(ns, id)]
	if !ok {
		return nil, verrors.ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (s *storCASBlind) Replace(_ context.Context, ns, id string, ct []byte) error {
	k := storK(ns, id)
	if _, ok := s.m[k]; !ok {
		return verrors.ErrNotFound
	}
	s.m[k] = append([]byte(nil), ct...)
	return nil
}

func (s *storCASBlind) Delete(_ context.Context, ns, id string) error {
	delete(s.m, storK(ns, id))
	return nil
}

func (s *storCASBlind) CompareAndSwap(_ context.Context, ns, id string, _ []byte, newCt []byte) error {
	k := storK(ns, id)
	if _, ok := s.m[k]; !ok {
		return verrors.ErrNotFound
	}
	s.m[k] = append([]byte(nil), newCt...)
	return nil
}

// storCASMissingWrong returns [verrors.ErrCASFailed] instead of [verrors.ErrNotFound] for CAS on a missing row.
type storCASMissingWrong struct{}

func (storCASMissingWrong) Create(context.Context, string, string, []byte) error { return nil }

func (storCASMissingWrong) Get(context.Context, string, string) ([]byte, error) {
	return nil, verrors.ErrNotFound
}

func (storCASMissingWrong) Replace(context.Context, string, string, []byte) error {
	return verrors.ErrNotFound
}

func (storCASMissingWrong) Delete(context.Context, string, string) error { return nil }

func (storCASMissingWrong) CompareAndSwap(context.Context, string, string, []byte, []byte) error {
	return verrors.ErrCASFailed
}
