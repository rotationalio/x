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
// Tests: Update / CompareAndSwap
//=============================================================================

// TestStringPurser_update covers UTF-8 enforcement on Update and a happy-path delegation smoke test.
func TestStringPurser_update(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", "v1")
	assert.Ok(t, err)

	assert.ErrorIs(t, w.Update(ctx, "ns", id, string([]byte{0xff, 0xfe})), perrors.ErrInvalidUTF8)

	assert.Ok(t, w.Update(ctx, "ns", id, "v2"))
	got, err := w.Retrieve(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, "v2", got)
}

// TestStringPurser_compareAndSwap rejects invalid UTF-8 and delegates a successful swap.
func TestStringPurser_compareAndSwap(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	res, err := w.Purser.Store(ctx, "ns", []byte("v1"))
	assert.Ok(t, err)

	casRes, err := w.CompareAndSwap(ctx, "ns", res.ID, string([]byte{0xff}), "v2")
	assert.Equal(t, purser.Result{}, casRes)
	assert.ErrorIs(t, err, perrors.ErrInvalidUTF8)

	casRes, err = w.CompareAndSwap(ctx, "ns", res.ID, "v1", "v2")
	assert.Equal(t, res.ID, casRes.ID)
	assert.Ok(t, err)
	got, err := w.Retrieve(ctx, "ns", res.ID)
	assert.Ok(t, err)
	assert.Equal(t, "v2", got)
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
