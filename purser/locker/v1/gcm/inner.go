package gcm

// Inner-payload AES-256-GCM seal and open helpers for per-row data encryption.

import (
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"io"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
)

// hkdfDataKeyInfo is the HKDF context string for stretching an X25519 shared secret into the row data key.
const hkdfDataKeyInfo = "purser/v1/x25519-hkdf-sha256-aes256gcm/data-key"

// DeriveDataKey derives the AES-256 row data key from an ECDH shared secret using HKDF-SHA256.
func DeriveDataKey(sharedSecret []byte) ([]byte, error) {
	return hkdf.Key(sha256.New, sharedSecret, nil, hkdfDataKeyInfo, constants.DataKeyBytes)
}

// NewInnerAEAD constructs inner payload AEAD (AES-256-GCM) for a 32-byte data key.
func NewInnerAEAD(key []byte) (cipher.AEAD, error) {
	// Inner AEAD is always AES-256 in v1; reject any other key length before touching the cipher.
	if len(key) != constants.DataKeyBytes {
		return nil, perrors.ErrMalformedParameters
	}
	return newAEAD(key)
}

// SealInner encrypts plaintext with aad as GCM additional data using a random nonce.
func SealInner(aead cipher.AEAD, aad, plaintext []byte) ([constants.InnerNonceBytes]byte, []byte, error) {
	// Check if the AEAD instance is nil.
	if aead == nil {
		return [constants.InnerNonceBytes]byte{}, nil, perrors.ErrNilAEAD
	}

	// v1 fixes the inner nonce at 12 bytes (AES-GCM standard IV size in this module). If someone
	// passed an AEAD from another construction, nonce size would not match and wire layout would break.
	if aead.NonceSize() != constants.InnerNonceBytes {
		return [constants.InnerNonceBytes]byte{}, nil, perrors.ErrMalformedParameters
	}

	// Fresh random nonce per seal; must not repeat for the same key under GCM.
	var nonce [constants.InnerNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return [constants.InnerNonceBytes]byte{}, nil, perrors.ErrSealFailed
	}

	// Delegate to the nonce-explicit path so tests and golden vectors can pin nonces.
	return SealInnerWithNonce(aead, aad, plaintext, nonce)
}

// SealInnerWithNonce encrypts plaintext with the given nonce (random in [SealInner]).
func SealInnerWithNonce(aead cipher.AEAD, aad, plaintext []byte, nonce [constants.InnerNonceBytes]byte) ([constants.InnerNonceBytes]byte, []byte, error) {
	// Check if the AEAD instance is nil.
	if aead == nil {
		return [constants.InnerNonceBytes]byte{}, nil, perrors.ErrNilAEAD
	}

	// Ensure the AEAD's nonce size matches the expected size for AES-GCM.
	if aead.NonceSize() != constants.InnerNonceBytes {
		return [constants.InnerNonceBytes]byte{}, nil, perrors.ErrMalformedParameters
	}

	// Encrypt the plaintext with the provided nonce and additional authenticated data (aad).
	ct := aead.Seal(nil, nonce[:], plaintext, aad)
	return nonce, ct, nil
}

// OpenInner decrypts ciphertext+tag with aad as GCM additional data.
func OpenInner(aead cipher.AEAD, aad []byte, nonce [constants.InnerNonceBytes]byte, payload []byte) ([]byte, error) {
	// Check if the AEAD instance is nil.
	if aead == nil {
		return nil, perrors.ErrNilAEAD
	}

	// Attempt to decrypt payload (ciphertext + tag) using the provided nonce and AAD.
	plain, err := aead.Open(nil, nonce[:], payload, aad)
	if err != nil {
		return nil, perrors.ErrDecrypt
	}
	return plain, nil
}
