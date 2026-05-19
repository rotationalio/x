package models

// DekEnvelope wire encoding: ephemeral X25519 public key, wrap nonce, and wrapped DEK payload.

import (
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
)

// DekEnvelope is the ECDH/HKDF/AEAD-wrapped per-row DEK (fixed 92 bytes for the initial v1 suite).
type DekEnvelope struct {
	Pub     [constants.X25519PubBytes]byte
	Nonce   [constants.WrapNonceBytes]byte
	Payload [constants.DEKBytes + constants.GCMTagBytes]byte // 48: 32-byte DEK + 16-byte tag
}

// MarshalBinarySize returns the encoded byte length of DekEnvelope.
func (d DekEnvelope) MarshalBinarySize() int {
	return constants.DekEnvelopeBytes
}

// MarshalBinaryTo encodes DekEnvelope into dst and returns written bytes.
func (d DekEnvelope) MarshalBinaryTo(dst []byte) (int, error) {
	// Envelope layout is fixed-width; caller must provide a full-size destination.
	if len(dst) < constants.DekEnvelopeBytes {
		return 0, perrors.ErrMalformedWire
	}

	// Copy contiguous fixed segments in wire order: ephemeral pubkey, wrap nonce, wrapped DEK payload.
	copy(dst[:constants.X25519PubBytes], d.Pub[:])
	copy(dst[constants.X25519PubBytes:constants.X25519PubBytes+constants.WrapNonceBytes], d.Nonce[:])
	copy(dst[constants.X25519PubBytes+constants.WrapNonceBytes:constants.DekEnvelopeBytes], d.Payload[:])
	return constants.DekEnvelopeBytes, nil
}

// MarshalBinary encodes DekEnvelope.
func (d DekEnvelope) MarshalBinary() ([]byte, error) {
	out := make([]byte, constants.DekEnvelopeBytes)
	_, err := d.MarshalBinaryTo(out)
	return out, err
}

// UnmarshalBinary decodes DekEnvelope; v1 requires exactly [constants.DekEnvelopeBytes].
func (d *DekEnvelope) UnmarshalBinary(data []byte) error {
	if d == nil {
		return perrors.ErrNilDekEnvelopePointer
	}

	// v1 envelope framing is exact-width; trailing or short data is malformed.
	if len(data) != constants.DekEnvelopeBytes {
		return perrors.ErrMalformedWire
	}

	// Decode using the same field boundaries as MarshalBinaryTo.
	copy(d.Pub[:], data[:constants.X25519PubBytes])
	copy(d.Nonce[:], data[constants.X25519PubBytes:constants.X25519PubBytes+constants.WrapNonceBytes])
	copy(d.Payload[:], data[constants.X25519PubBytes+constants.WrapNonceBytes:])
	return nil
}
