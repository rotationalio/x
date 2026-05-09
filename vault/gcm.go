package vault

// AES-GCM seal and open helpers used by [Vault] and any caller with an
// [cipher.AEAD]. Additional authenticated data (AAD) is supplied by the caller
// as opaque bytes.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// newAEAD constructs and returns an AEAD cipher (AES-GCM) using the provided
// key. Keys of 16, 24, or 32 bytes select AES-128, AES-192, and AES-256
// respectively.
func newAEAD(key []byte) (cipher.AEAD, error) {
	// Validate key length.
	switch len(key) {
	case 16, 24, 32:
		// Valid key length.
	default:
		return nil, errors.Join(ErrInvalidAESKeySize, fmt.Errorf("got %d bytes", len(key)))
	}

	// Instantiate the block cipher from the key.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Wrap the block cipher into an AEAD (Galois/Counter Mode).
	return cipher.NewGCM(block)
}

// seal encrypts plain using aead, with a randomly generated unique nonce. The
// output format is [nonce | ciphertext]. aad is GCM additional authenticated
// data; the same bytes must be supplied to [open].
func seal(aead cipher.AEAD, aad, plain []byte) ([]byte, error) {
	// Allocate nonce buffer.
	ns := aead.NonceSize()
	nonce := make([]byte, ns)

	// Fill nonce with secure random bytes.
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Join(ErrSealFailed, err)
	}

	// AEAD.Seal appends the encrypted+MAC'd data to the provided dst.
	ct := aead.Seal(nil, nonce, plain, aad)

	// Create output: [nonce | ciphertext]
	out := make([]byte, ns+len(ct))
	copy(out, nonce)
	copy(out[ns:], ct)

	return out, nil
}

// open decrypts a blob produced by [seal]. aad must match the value used when
// sealing. Layout: nonce prefix, then ciphertext+tag.
func open(aead cipher.AEAD, aad, blob []byte) ([]byte, error) {
	// Check blob is long enough to contain at least the nonce.
	ns := aead.NonceSize()
	if len(blob) < ns {
		return nil, errors.Join(ErrDecrypt, ErrCiphertextTooShort, fmt.Errorf("got %d bytes", len(blob)))
	}

	// Extract nonce and ciphertext.
	nonce := blob[:ns]
	ct := blob[ns:]

	// Authenticate and decrypt the ciphertext.
	plain, err := aead.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, errors.Join(ErrDecrypt, err)
	}
	return plain, nil
}
