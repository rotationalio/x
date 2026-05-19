package jsonpurser_test

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/pursertest"
	jsonpurser "go.rtnl.ai/x/purser/wrappers/json"
)

//=============================================================================
// Tests: roundtrip across representative JSON values
//=============================================================================

// TestJSONPurser_roundtrip exercises Store/Retrieve across a small set of
// representative JSON payloads (scalars, struct, nested struct, empty struct).
func TestJSONPurser_roundtrip(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	t.Run("scalar_int", func(t *testing.T) {
		id, err := w.Store(ctx, "ns", 42)
		assert.Ok(t, err)
		var got int
		assert.Ok(t, w.Retrieve(ctx, "ns", id, &got))
		assert.Equal(t, 42, got)
	})

	t.Run("simple_struct", func(t *testing.T) {
		want := payload{A: 7, B: "hello"}
		id, err := w.Store(ctx, "ns", want)
		assert.Ok(t, err)
		var got payload
		assert.Ok(t, w.Retrieve(ctx, "ns", id, &got))
		assert.Equal(t, want, got)
	})

	t.Run("nested_struct", func(t *testing.T) {
		want := nested{Outer: "o", Inner: payload{A: 9, B: "y"}}
		id, err := w.Store(ctx, "ns", want)
		assert.Ok(t, err)
		var got nested
		assert.Ok(t, w.Retrieve(ctx, "ns", id, &got))
		assert.Equal(t, want, got)
	})

	t.Run("empty_struct", func(t *testing.T) {
		want := payload{}
		id, err := w.Store(ctx, "ns", want)
		assert.Ok(t, err)
		var got payload
		assert.Ok(t, w.Retrieve(ctx, "ns", id, &got))
		assert.Equal(t, want, got)
	})
}

//=============================================================================
// Tests: marshal / unmarshal failure paths
//=============================================================================

// TestJSONPurser_retrieveNilDst checks Retrieve rejects a nil destination upfront.
func TestJSONPurser_retrieveNilDst(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", 1)
	assert.Ok(t, err)
	assert.ErrorIs(t, w.Retrieve(ctx, "ns", id, nil), perrors.ErrNilRetrieveDst)
}

// TestJSONPurser_storeMarshalFailure ensures non-marshalable values return ErrJSONMarshal
// before any hold I/O happens.
func TestJSONPurser_storeMarshalFailure(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()
	ch := make(chan int)

	_, err := w.Store(ctx, "ns", ch)
	assert.ErrorIs(t, err, perrors.ErrJSONMarshal)
}

// TestJSONPurser_retrieveUnmarshalFailure ensures corrupt JSON yields ErrJSONUnmarshal
// joined with ErrInvalidJSON, and that the corrupt wire is left in place.
func TestJSONPurser_retrieveUnmarshalFailure(t *testing.T) {
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(t, err)
	p, lck := pursertest.NewTestPurserWithLocker(t, h)
	w := jsonpurser.New(p)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", payload{A: 1})
	assert.Ok(t, err)
	corruptWire, err := lck.Seal("ns", []byte(`{"a":`))
	assert.Ok(t, err)
	h.BypassSemanticsSetBlobForTest(t, "ns", id, corruptWire)

	err = w.Retrieve(ctx, "ns", id, new(payload))
	assert.ErrorIs(t, err, perrors.ErrJSONUnmarshal)
	assert.ErrorIs(t, err, perrors.ErrInvalidJSON)

	// The wrapper does not auto-repair the row.
	stillCorrupt, err := h.Get(ctx, "ns", id)
	assert.Ok(t, err)
	assert.Equal(t, corruptWire, stillCorrupt)
}

// TestJSONPurser_equalJSONMarshalFailure ensures EqualJSON surfaces ErrJSONMarshal.
func TestJSONPurser_equalJSONMarshalFailure(t *testing.T) {
	_, err := jsonpurser.EqualJSON(make(chan int), 1)
	assert.ErrorIs(t, err, perrors.ErrJSONMarshal)
}

//=============================================================================
// Tests: Update / CompareAndSwap / MoveNamespace / Delete
//=============================================================================

// TestJSONPurser_update covers happy-path replacement, marshal-failure rejection, and
// missing-row propagation.
func TestJSONPurser_update(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", payload{A: 1})
	assert.Ok(t, err)

	// Marshal failure rejected up front.
	assert.ErrorIs(t, w.Update(ctx, "ns", id, make(chan int)), perrors.ErrJSONMarshal)

	// Happy path.
	assert.Ok(t, w.Update(ctx, "ns", id, payload{A: 2}))
	var got payload
	assert.Ok(t, w.Retrieve(ctx, "ns", id, &got))
	assert.Equal(t, payload{A: 2}, got)

	// Missing row surfaces ErrNotFound.
	err = w.Update(ctx, "ns", "00112233445566778899aabbccddeeff", payload{A: 3})
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestJSONPurser_compareAndSwap covers JSON validation on both arguments, wrong-current,
// success, and missing-row paths.
func TestJSONPurser_compareAndSwap(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", payload{A: 1})
	assert.Ok(t, err)

	// Invalid JSON in either argument rejected before reaching the inner purser.
	err = w.CompareAndSwap(ctx, "ns", id, []byte(`{"a":`), []byte(`{"a":2}`))
	assert.ErrorIs(t, err, perrors.ErrJSONUnmarshal)
	assert.ErrorIs(t, err, perrors.ErrInvalidJSON)
	err = w.CompareAndSwap(ctx, "ns", id, []byte(`{"a":1}`), []byte(`{"a":`))
	assert.ErrorIs(t, err, perrors.ErrJSONUnmarshal)
	assert.ErrorIs(t, err, perrors.ErrInvalidJSON)

	// Wrong current — refuses to swap and leaves the row at A=1.
	assert.ErrorIs(t, w.CompareAndSwap(ctx, "ns", id, []byte(`{"a":99}`), []byte(`{"a":2}`)), perrors.ErrWrongCurrent)
	var got payload
	assert.Ok(t, w.Retrieve(ctx, "ns", id, &got))
	assert.Equal(t, payload{A: 1}, got)

	// Correct current — swap succeeds.
	assert.Ok(t, w.CompareAndSwap(ctx, "ns", id, []byte(`{"a":1}`), []byte(`{"a":2}`)))
	assert.Ok(t, w.Retrieve(ctx, "ns", id, &got))
	assert.Equal(t, payload{A: 2}, got)

	// Missing row.
	err = w.CompareAndSwap(ctx, "ns", "aabbccddeeff00112233445566778899", []byte(`{"a":1}`), []byte(`{"a":2}`))
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestJSONPurser_moveNamespace ensures embedded MoveNamespace works end-to-end.
func TestJSONPurser_moveNamespace(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns-a", payload{A: 1})
	assert.Ok(t, err)

	assert.Ok(t, w.MoveNamespace(ctx, "ns-a", "ns-b", id))
	var got payload
	assert.Ok(t, w.Retrieve(ctx, "ns-b", id, &got))
	assert.Equal(t, payload{A: 1}, got)

	err = w.Retrieve(ctx, "ns-a", id, &got)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

// TestJSONPurser_delete ensures embedded Delete is reachable and removes the row.
func TestJSONPurser_delete(t *testing.T) {
	w, _ := newWrappedPurser(t)
	ctx := context.Background()

	id, err := w.Store(ctx, "ns", payload{A: 1})
	assert.Ok(t, err)

	assert.Ok(t, w.Delete(ctx, "ns", id))
	var got payload
	err = w.Retrieve(ctx, "ns", id, &got)
	assert.ErrorIs(t, err, perrors.ErrNotFound)
}

//=============================================================================
// Helpers and shared types
//=============================================================================

// payload is a small JSON-encodable struct used across the round-trip tests.
type payload struct {
	A int    `json:"a"`
	B string `json:"b,omitempty"`
}

// nested exercises a struct-within-struct case so any field-level mishandling shows up.
type nested struct {
	Outer string  `json:"outer"`
	Inner payload `json:"inner"`
}

// newWrappedPurser builds a JSON-wrapped Purser backed by a null locker via the real
// purser.New orchestration.
func newWrappedPurser(tb testing.TB) (*jsonpurser.Purser, *hold.MemHold) {
	tb.Helper()
	h, err := hold.NewMemHold(hexid.Identifier{})
	assert.Ok(tb, err)
	return jsonpurser.New(pursertest.NewTestPurser(tb, h)), h
}
