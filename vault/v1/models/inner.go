package models

import (
	"go.rtnl.ai/x/vault/v1/constants"
	v1errs "go.rtnl.ai/x/vault/v1/errors"
)

// Inner is nonce plus inner ciphertext+tag. GCM additional data is the marshaled row [Meta]
// (see [Sealed.Meta]).
type Inner struct {
	Nonce   [constants.InnerNonceBytes]byte
	Payload []byte // inner ciphertext including GCM tag ([constants.GCMTagBytes] bytes).
}

// MarshalBinary encodes Inner as nonce||payload.
func (i Inner) MarshalBinary() ([]byte, error) {
	out := make([]byte, 0, len(i.Nonce)+len(i.Payload))
	out = append(out, i.Nonce[:]...)
	out = append(out, i.Payload...)
	return out, nil
}

// UnmarshalBinary decodes Inner; consumes the full slice.
func (i *Inner) UnmarshalBinary(data []byte) error {
	// Check if the receiver is nil.
	if i == nil {
		return v1errs.ErrNilInnerPointer
	}

	// Minimum wire is 12-byte nonce plus a GCM tag (empty plaintext still produces ciphertext length 0 + tag).
	if len(data) < constants.InnerNonceBytes+constants.GCMTagBytes {
		return v1errs.ErrMalformedWire
	}

	// Split fixed prefix (nonce) from tail (everything the inner AEAD produced).
	copy(i.Nonce[:], data[:constants.InnerNonceBytes])
	i.Payload = append([]byte(nil), data[constants.InnerNonceBytes:]...)
	return nil
}
