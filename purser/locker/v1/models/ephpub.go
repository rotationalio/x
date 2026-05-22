package models

// EphPub wire encoding: per-row ephemeral X25519 public key (fixed 32 bytes).

import (
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
)

// EphPub is the ephemeral X25519 public key sent with each sealed row.
type EphPub [constants.EphPubBytes]byte

// MarshalBinarySize returns the encoded byte length of EphPub.
func (e EphPub) MarshalBinarySize() int {
	return constants.EphPubBytes
}

// MarshalBinaryTo encodes EphPub into dst and returns written bytes.
func (e EphPub) MarshalBinaryTo(dst []byte) (int, error) {
	if len(dst) < constants.EphPubBytes {
		return 0, perrors.ErrMalformedWire
	}
	copy(dst[:constants.EphPubBytes], e[:])
	return constants.EphPubBytes, nil
}

// MarshalBinary encodes EphPub.
func (e EphPub) MarshalBinary() ([]byte, error) {
	out := make([]byte, constants.EphPubBytes)
	_, err := e.MarshalBinaryTo(out)
	return out, err
}

// UnmarshalBinary decodes EphPub; v1 requires exactly [constants.EphPubBytes].
func (e *EphPub) UnmarshalBinary(data []byte) error {
	if e == nil {
		return perrors.ErrNilEphPubPointer
	}
	if len(data) != constants.EphPubBytes {
		return perrors.ErrMalformedWire
	}
	copy(e[:], data)
	return nil
}
