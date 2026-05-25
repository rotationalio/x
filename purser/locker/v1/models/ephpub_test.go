package models_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/models"
)

// TestEphPub_roundtrip checks [models.EphPub.MarshalBinary] wire size and unmarshal round-trip.
func TestEphPub_roundtrip(t *testing.T) {
	var e models.EphPub
	for i := range e {
		e[i] = byte(i + 1)
	}

	raw, err := e.MarshalBinary()
	assert.Ok(t, err)
	assert.Equal(t, constants.EphPubBytes, len(raw))

	var got models.EphPub
	assert.Ok(t, got.UnmarshalBinary(raw))
	assert.Equal(t, e, got)
}

// TestEphPub_unmarshal_errors covers truncated wire and a nil [models.EphPub] receiver.
func TestEphPub_unmarshal_errors(t *testing.T) {
	var e models.EphPub
	assert.ErrorIs(t, e.UnmarshalBinary(make([]byte, constants.EphPubBytes-1)), perrors.ErrMalformedWire)
	var p *models.EphPub
	assert.ErrorIs(t, p.UnmarshalBinary(make([]byte, constants.EphPubBytes)), perrors.ErrNilEphPubPointer)
}
