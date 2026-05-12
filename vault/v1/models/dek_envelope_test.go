package models_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/v1/errors"
	"go.rtnl.ai/x/vault/v1/models"
)

// TestDekEnvelope_roundtrip checks [models.DekEnvelope.MarshalBinary] wire size and unmarshal round-trip.
func TestDekEnvelope_roundtrip(t *testing.T) {
	var d models.DekEnvelope
	for i := range d.Pub {
		d.Pub[i] = byte(i)
	}
	for i := range d.Nonce {
		d.Nonce[i] = byte(i + 1)
	}
	for i := range d.Payload {
		d.Payload[i] = byte(i + 2)
	}
	raw, err := d.MarshalBinary()
	assert.Ok(t, err)
	assert.Equal(t, constants.DekEnvelopeBytes, len(raw))
	var got models.DekEnvelope
	assert.Ok(t, got.UnmarshalBinary(raw))
	assert.Equal(t, d, got)
}

// TestDekEnvelope_unmarshal_errors covers truncated wire and a nil [models.DekEnvelope] receiver.
func TestDekEnvelope_unmarshal_errors(t *testing.T) {
	var d models.DekEnvelope
	assert.ErrorIs(t, d.UnmarshalBinary(make([]byte, constants.DekEnvelopeBytes-1)), verrors.ErrMalformedWire)
	var p *models.DekEnvelope
	assert.ErrorIs(t, p.UnmarshalBinary(make([]byte, constants.DekEnvelopeBytes)), verrors.ErrNilDekEnvelopePointer)
}
