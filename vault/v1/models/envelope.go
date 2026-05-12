package models

import (
	"go.rtnl.ai/x/vault/v1/constants"
	v1errs "go.rtnl.ai/x/vault/v1/errors"
)

// DekEnvelope is the ECDH/HKDF/AEAD-wrapped per-row DEK (fixed 92 bytes for the initial v1 suite).
type DekEnvelope struct {
	Pub     [constants.X25519PubBytes]byte
	Nonce   [constants.WrapNonceBytes]byte
	Payload [constants.DEKBytes + constants.GCMTagBytes]byte // 48: 32-byte DEK + 16-byte tag
}

// MarshalBinary encodes DekEnvelope.
func (d DekEnvelope) MarshalBinary() ([]byte, error) {
	out := make([]byte, 0, constants.DekEnvelopeBytes)
	out = append(out, d.Pub[:]...)
	out = append(out, d.Nonce[:]...)
	out = append(out, d.Payload[:]...)
	return out, nil
}

// UnmarshalBinary decodes DekEnvelope; v1 requires exactly [constants.DekEnvelopeBytes].
func (d *DekEnvelope) UnmarshalBinary(data []byte) error {
	if d == nil {
		return v1errs.ErrNilDekEnvelopePointer
	}
	if len(data) != constants.DekEnvelopeBytes {
		return v1errs.ErrMalformedWire
	}
	copy(d.Pub[:], data[:constants.X25519PubBytes])
	copy(d.Nonce[:], data[constants.X25519PubBytes:constants.X25519PubBytes+constants.WrapNonceBytes])
	copy(d.Payload[:], data[constants.X25519PubBytes+constants.WrapNonceBytes:])
	return nil
}
