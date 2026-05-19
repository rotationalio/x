package models_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/models"
)

// TestDekEnvelope_roundtrip checks [models.DekEnvelope.MarshalBinary] wire size and unmarshal round-trip.
func TestDekEnvelope_roundtrip(t *testing.T) {
	// Build a DekEnvelope with deterministic but distinct values for each field so a
	// boundary error in MarshalBinary/UnmarshalBinary would leak into the diff.
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

	// Marshal to wire bytes.
	raw, err := d.MarshalBinary()
	assert.Ok(t, err)

	// Wire size must equal the documented constant — anything else means the framing
	// drifted from the v1 envelope contract.
	assert.Equal(t, constants.DekEnvelopeBytes, len(raw))

	// Unmarshal into a fresh value and assert the full struct equals the original.
	var got models.DekEnvelope
	assert.Ok(t, got.UnmarshalBinary(raw))
	assert.Equal(t, d, got)
}

// TestDekEnvelope_unmarshal_errors covers truncated wire and a nil [models.DekEnvelope] receiver.
func TestDekEnvelope_unmarshal_errors(t *testing.T) {
	var d models.DekEnvelope
	assert.ErrorIs(t, d.UnmarshalBinary(make([]byte, constants.DekEnvelopeBytes-1)), perrors.ErrMalformedWire)
	var p *models.DekEnvelope
	assert.ErrorIs(t, p.UnmarshalBinary(make([]byte, constants.DekEnvelopeBytes)), perrors.ErrNilDekEnvelopePointer)
}
