package stringpurser_test

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/pursertest"
	stringpurser "go.rtnl.ai/x/purser/wrappers/string"
)

//=============================================================================
// Tests: roundtrip across UTF-8 corner cases
//=============================================================================

// TestStringPurser_roundtrip exercises Store/Retrieve across a small but representative
// set of UTF-8 strings so any boundary bug (empty, ASCII, multibyte, combining marks)
// shows up in one place.
func TestStringPurser_roundtrip(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"ascii", "hello"},
		{"multibyte", "héllo"},
		{"emoji", "smile: 🙂"},
		{"combining", "café"}, // e + combining acute
	}
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := w.Store(ctx, "ns", tc.in)
			assert.Ok(t, err)
			got, err := w.Retrieve(ctx, "ns", id)
			assert.Ok(t, err)
			assert.Equal(t, tc.in, got)
		})
	}
}

//=============================================================================
// Tests: constructor and nil receiver
//=============================================================================

// TestStringPurser_newNil rejects a nil inner purser.
func TestStringPurser_newNil(t *testing.T) {
	_, err := stringpurser.New(nil)
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestStringPurser_nilReceiver_store verifies Store on a nil wrapper returns ErrNilPurser.
func TestStringPurser_nilReceiver_store(t *testing.T) {
	var w *stringpurser.Purser
	_, err := w.Store(context.Background(), "ns", "hello")
	assert.ErrorIs(t, err, perrors.ErrNilPurser)
}

//=============================================================================
// Tests: invalid UTF-8 contracts
//=============================================================================

// TestStringPurser_invalidUTF8 checks invalid UTF-8 is rejected on Store before any
// hold I/O happens.
func TestStringPurser_invalidUTF8(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	// 0xff 0xfe is not a valid UTF-8 sequence.
	invalid := string([]byte{0xff, 0xfe})

	_, err := w.Store(ctx, "ns", invalid)
	assert.ErrorIs(t, err, perrors.ErrInvalidUTF8)
}

// TestStringPurser_invalidUTF8CorruptRow ensures the post-decrypt UTF-8 check fires
// when a row's plaintext is replaced with invalid bytes through the same locker, and
// confirms the corrupted wire is left in place (no auto-repair).
func TestStringPurser_invalidUTF8CorruptRow(t *testing.T) {
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)
	p, lck := pursertest.NewTestPurserWithLocker(t, h)
	w, err := stringpurser.New(p)
	assert.Ok(t, err)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", "good")
	assert.Ok(t, err)
	corruptWire, err := lck.Seal("ns", []byte{0xff, 0xfe})
	assert.Ok(t, err)
	h.BypassSemanticsSetBlobForTest(t, "ns", id, corruptWire)

	// Retrieve must surface ErrInvalidUTF8.
	_, err = w.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, perrors.ErrInvalidUTF8)

	// And the wrapper must not have silently rewritten or deleted the corrupt row;
	// re-reading via the hold returns the same corrupted bytes we injected.
	stillCorrupt, err := h.Get(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, corruptWire, stillCorrupt)
}

//=============================================================================
// Tests: Update / CompareAndSwap / MoveNamespace / Delete
//=============================================================================

// TestStringPurser_update covers UTF-8 enforcement on Update, happy-path replacement,
// and propagation of missing-row errors.
func TestStringPurser_update(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", "v1")
	assert.Ok(t, err)

	// Invalid UTF-8 rejected before reaching the inner purser.
	assert.ErrorIs(t, w.Update(ctx, "ns", id, string([]byte{0xff, 0xfe})), perrors.ErrInvalidUTF8)

	// Happy-path replacement.
	assert.Ok(t, w.Update(ctx, "ns", id, "v2"))
	got, err := w.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, "v2", got)

	// Update on a missing row surfaces ErrNotFound.
	err = w.Update(ctx, "ns", "00112233445566778899aabbccddeeff", "v3")
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestStringPurser_compareAndSwap covers UTF-8 enforcement on both arguments, the
// wrong-current path, the success path, and the missing-row path.
func TestStringPurser_compareAndSwap(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	res, err := w.Purser.Store(ctx, "ns", []byte("v1"))
	assert.Ok(t, err)

	// Invalid UTF-8 in either argument rejected before reaching the inner purser.
	casRes, err := w.CompareAndSwap(ctx, "ns", res.ID, string([]byte{0xff}), "v2")
	assert.Equal(t, purser.Result{}, casRes)
	assert.ErrorIs(t, err, perrors.ErrInvalidUTF8)
	casRes, err = w.CompareAndSwap(ctx, "ns", res.ID, "v1", string([]byte{0xff}))
	assert.Equal(t, purser.Result{}, casRes)
	assert.ErrorIs(t, err, perrors.ErrInvalidUTF8)

	// Wrong current — refuses to swap and leaves the row at "v1".
	casRes, err = w.CompareAndSwap(ctx, "ns", res.ID, "wrong", "v2")
	assert.Equal(t, purser.Result{}, casRes)
	assert.ErrorIs(t, err, perrors.ErrWrongCurrent)
	got, err := w.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, "v1", got)

	// Correct current — swap succeeds.
	casRes, err = w.CompareAndSwap(ctx, "ns", res.ID, "v1", "v2")
	assert.Equal(t, "ns", casRes.Namespace)
	assert.Equal(t, res.KeyID, casRes.KeyID)
	assert.Equal(t, res.Edition, casRes.Edition)
	assert.Equal(t, res.ID, casRes.ID)
	assert.Ok(t, err)
	got, err = w.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, "v2", got)

	// CAS on missing identifier — ErrNotFound bubbles through the wrapper.
	casRes, err = w.CompareAndSwap(ctx, "ns", "aabbccddeeff00112233445566778899", "a", "b")
	assert.Equal(t, purser.Result{}, casRes)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestStringPurser_moveNamespace ensures the embedded [purser.Purser.MoveNamespace] is reachable
// through the wrapper and works end-to-end on UTF-8 data.
func TestStringPurser_moveNamespace(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns-a", "value")
	assert.Ok(t, err)

	assert.Ok(t, w.MoveNamespace(ctx, "ns-a", "ns-b", id))
	got, err := w.Retrieve(ctx, "ns-b", id)
	assert.Ok(t, err)
	assert.Equal(t, "value", got)

	_, err = w.Retrieve(ctx, "ns-a", id)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestStringPurser_delete ensures Delete is reachable through the wrapper and removes
// the row.
func TestStringPurser_delete(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", "value")
	assert.Ok(t, err)

	assert.Ok(t, w.Delete(ctx, "ns", id))
	_, err = w.Retrieve(ctx, "ns", id)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

//=============================================================================
// Helpers
//=============================================================================

// newWrappedPurser builds a string-wrapped Purser backed by a null locker via the
// real [purser.New] orchestration. The hold is returned for tests that need to inject
// or read raw wire blobs.
func newWrappedPurser(tb testing.TB) (*stringpurser.Purser, *hold.MemHold) {
	tb.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(tb, err)
	w, err := stringpurser.New(pursertest.NewTestPurser(tb, h))
	assert.Ok(tb, err)
	return w, h
}
