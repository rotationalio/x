package gcm

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/v1/errors"
)

// NewInnerAEAD constructs inner payload AEAD (AES-256-GCM) for a 32-byte DEK.
func NewInnerAEAD(dek []byte) (cipher.AEAD, error) {
	// Inner AEAD is always AES-256 in v1; reject any other key length before touching the cipher.
	if len(dek) != constants.DEKBytes {
		return nil, verrors.ErrMalformedParameters
	}
	return newAEAD(dek)
}

// SealInner encrypts plaintext with aad as GCM additional data using a random nonce.
func SealInner(aead cipher.AEAD, aad, plaintext []byte) ([constants.InnerNonceBytes]byte, []byte, error) {
	// Check if the AEAD instance is nil.
	if aead == nil {
		return [constants.InnerNonceBytes]byte{}, nil, verrors.ErrNilAEAD
	}

	// v1 fixes the inner nonce at 12 bytes (AES-GCM standard IV size in this module). If someone
	// passed an AEAD from another construction, nonce size would not match and wire layout would break.
	if aead.NonceSize() != constants.InnerNonceBytes {
		return [constants.InnerNonceBytes]byte{}, nil, verrors.ErrMalformedParameters
	}

	// Fresh random nonce per seal; must not repeat for the same DEK under GCM.
	var nonce [constants.InnerNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return [constants.InnerNonceBytes]byte{}, nil, verrors.ErrSealFailed
	}

	// Delegate to the nonce-explicit path so tests and golden vectors can pin nonces.
	return SealInnerWithNonce(aead, aad, plaintext, nonce)
}

// SealInnerWithNonce encrypts plaintext with the given nonce (random in [SealInner]).
func SealInnerWithNonce(aead cipher.AEAD, aad, plaintext []byte, nonce [constants.InnerNonceBytes]byte) ([constants.InnerNonceBytes]byte, []byte, error) {
	// Check if the AEAD instance is nil.
	if aead == nil {
		return [constants.InnerNonceBytes]byte{}, nil, verrors.ErrNilAEAD
	}

	// Ensure the AEAD's nonce size matches the expected size for AES-GCM.
	if aead.NonceSize() != constants.InnerNonceBytes {
		return [constants.InnerNonceBytes]byte{}, nil, verrors.ErrMalformedParameters
	}

	// Encrypt the plaintext with the provided nonce and additional authenticated data (aad).
	ct := aead.Seal(nil, nonce[:], plaintext, aad)
	return nonce, ct, nil
}

// OpenInner decrypts ciphertext+tag with aad as GCM additional data.
func OpenInner(aead cipher.AEAD, aad []byte, nonce [constants.InnerNonceBytes]byte, payload []byte) ([]byte, error) {
	// Check if the AEAD instance is nil.
	if aead == nil {
		return nil, verrors.ErrNilAEAD
	}

	// Attempt to decrypt payload (ciphertext + tag) using the provided nonce and AAD.
	plain, err := aead.Open(nil, nonce[:], payload, aad)
	if err != nil {
		return nil, verrors.ErrDecrypt
	}
	return plain, nil
}
