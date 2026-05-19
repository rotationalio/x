package gcm

// AES-GCM AEAD construction shared by inner-payload and DEK-wrap paths.

import (
	"crypto/aes"
	"crypto/cipher"

	perrors "go.rtnl.ai/x/purser/errors"
)

// newAEAD constructs an AES-GCM AEAD for key material of an allowed size.
func newAEAD(key []byte) (cipher.AEAD, error) {
	switch len(key) {
	case 16, 24, 32:
		// Valid key sizes for AES-128, AES-192, and AES-256.
	default:
		return nil, perrors.ErrInvalidAEADKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, perrors.ErrInvalidAEADKey
	}

	return cipher.NewGCM(block)
}
