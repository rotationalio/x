package models_test

import (
	"bytes"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/v1/errors"
	"go.rtnl.ai/x/vault/v1/models"
)

// TestInner_roundtrip checks [models.Inner.MarshalBinary] and [models.Inner.UnmarshalBinary] preserve nonce and payload.
func TestInner_roundtrip(t *testing.T) {
	in := models.Inner{
		Nonce:   [constants.InnerNonceBytes]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Payload: bytes.Repeat([]byte{'x'}, constants.GCMTagBytes),
	}
	raw, err := in.MarshalBinary()
	assert.Ok(t, err)
	var got models.Inner
	assert.Ok(t, got.UnmarshalBinary(raw))
	assert.Equal(t, in.Nonce, got.Nonce)
	assert.Equal(t, in.Payload, got.Payload)
}

// TestInner_unmarshal_nil_receiver asserts [*models.Inner.UnmarshalBinary] on a nil receiver returns [verrors.ErrNilInnerPointer].
func TestInner_unmarshal_nil_receiver(t *testing.T) {
	var p *models.Inner
	assert.ErrorIs(t, p.UnmarshalBinary(nil), verrors.ErrNilInnerPointer)
}
