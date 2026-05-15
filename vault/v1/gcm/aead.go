package gcm

import (
	"crypto/aes"
	"crypto/cipher"

	verrors "go.rtnl.ai/x/vault/errors"
)

// newAEAD constructs an AES-GCM AEAD for key material of an allowed size.
func newAEAD(key []byte) (cipher.AEAD, error) {
	switch len(key) {
	case 16, 24, 32:
		// Valid key sizes for AES-128, AES-192, and AES-256.
	default:
		return nil, verrors.ErrInvalidAEADKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, verrors.ErrInvalidAEADKey
	}

	return cipher.NewGCM(block)
}
