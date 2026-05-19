package models

// Inner wire encoding: nonce and GCM ciphertext+tag for the user's plaintext payload.

import (
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
)

// Inner is nonce plus inner ciphertext+tag. GCM additional data is the marshaled row [Meta]
// (see [Sealed.Meta]).
type Inner struct {
	Nonce   [constants.InnerNonceBytes]byte
	Payload []byte // inner ciphertext including GCM tag ([constants.GCMTagBytes] bytes).
}

// MarshalBinarySize returns the encoded byte length of Inner.
func (i Inner) MarshalBinarySize() int {
	return len(i.Nonce) + len(i.Payload)
}

// MarshalBinaryTo encodes Inner into dst and returns written bytes.
func (i Inner) MarshalBinaryTo(dst []byte) (int, error) {
	need := i.MarshalBinarySize()

	// Validate output capacity before writing fixed and variable segments.
	if len(dst) < need {
		return 0, perrors.ErrMalformedWire
	}

	// Layout is nonce||payload with no additional framing.
	copy(dst[:constants.InnerNonceBytes], i.Nonce[:])
	copy(dst[constants.InnerNonceBytes:need], i.Payload)

	return need, nil
}

// MarshalBinary encodes Inner as nonce||payload.
func (i Inner) MarshalBinary() ([]byte, error) {
	out := make([]byte, i.MarshalBinarySize())
	_, err := i.MarshalBinaryTo(out)
	return out, err
}

// UnmarshalBinary decodes Inner; consumes the full slice.
func (i *Inner) UnmarshalBinary(data []byte) error {
	if i == nil {
		return perrors.ErrNilInnerPointer
	}
	if len(data) < constants.InnerNonceBytes+constants.GCMTagBytes {
		return perrors.ErrMalformedWire
	}

	copy(i.Nonce[:], data[:constants.InnerNonceBytes])
	i.Payload = append([]byte(nil), data[constants.InnerNonceBytes:]...)
	return nil
}
