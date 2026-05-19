package models_test

import (
	"bytes"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/models"
)

// TestInner_roundtrip checks [models.Inner.MarshalBinary] and [models.Inner.UnmarshalBinary] preserve nonce and payload.
func TestInner_roundtrip(t *testing.T) {
	// A representative Inner: a deterministic 12-byte nonce and a payload sized to the
	// minimum we expect (the GCM tag length).
	in := models.Inner{
		Nonce:   [constants.InnerNonceBytes]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Payload: bytes.Repeat([]byte{'x'}, constants.GCMTagBytes),
	}

	// Marshal to wire, then unmarshal into a fresh value.
	raw, err := in.MarshalBinary()
	assert.Ok(t, err)
	var got models.Inner
	assert.Ok(t, got.UnmarshalBinary(raw))

	// Both fields survive the round-trip — no field reordering or length confusion.
	assert.Equal(t, in.Nonce, got.Nonce)
	assert.Equal(t, in.Payload, got.Payload)
}

// TestInner_unmarshal_nil_receiver asserts [*models.Inner.UnmarshalBinary] on a nil receiver returns [perrors.ErrNilInnerPointer].
func TestInner_unmarshal_nil_receiver(t *testing.T) {
	var p *models.Inner
	assert.ErrorIs(t, p.UnmarshalBinary(nil), perrors.ErrNilInnerPointer)
}
