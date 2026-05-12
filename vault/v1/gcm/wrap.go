package gcm

import (
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/errors"
)

// WrappedDEK is the fixed-layout DEK wrap segment (pub, nonce, ciphertext+tag).
// It matches [models.DekEnvelope] field-for-field for easy copying.
//
// Layout on the wire: Pub is the ephemeral X25519 public key (not encrypted). Nonce is the
// wrap-AEAD nonce. Payload is exactly DEK ciphertext plus GCM tag so total Payload length is
// DEKBytes + GCMTagBytes.
type WrappedDEK struct {
	Pub     [constants.X25519PubBytes]byte
	Nonce   [constants.WrapNonceBytes]byte
	Payload [constants.DEKBytes + constants.GCMTagBytes]byte
}

// NewWrapAEAD constructs DEK-wrap AEAD (AES-256-GCM) for a 32-byte wrap key.
func NewWrapAEAD(wrapKey []byte) (cipher.AEAD, error) {
	if len(wrapKey) != constants.WrapKeyBytes {
		return nil, verrors.ErrMalformedParameters
	}
	return newAEAD(wrapKey)
}

// SealWrappedDEK wraps dek with wrapAAD using ephemeral X25519 public key pub.
func SealWrappedDEK(pub [constants.X25519PubBytes]byte, aead cipher.AEAD, wrapAAD, dek []byte) (WrappedDEK, error) {
	// Reject nil AEAD instance.
	if aead == nil {
		return WrappedDEK{}, verrors.ErrNilAEAD
	}

	// Require exactly 32 bytes for the DEK; ensures fit in the fixed Payload array.
	if len(dek) != constants.DEKBytes {
		return WrappedDEK{}, verrors.ErrMalformedParameters
	}

	// Enforce the AEAD uses the standard 12-byte (GCM) nonce before reading entropy.
	if aead.NonceSize() != constants.WrapNonceBytes {
		return WrappedDEK{}, verrors.ErrMalformedParameters
	}

	// Generate a random nonce for this DEK wrap operation.
	var nonce [constants.WrapNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return WrappedDEK{}, verrors.ErrSealFailed
	}

	// Delegate to the nonce-explicit path for deterministic/golden vector testing convenience.
	return SealWrappedDEKWithNonce(pub, aead, wrapAAD, dek, nonce)
}

// SealWrappedDEKWithNonce wraps dek using the given nonce (random in [SealWrappedDEK]).
// NOTE: this is separated from SealWrappedDEK so we can generate fixed golden vector tests
// easily.
func SealWrappedDEKWithNonce(pub [constants.X25519PubBytes]byte, aead cipher.AEAD, wrapAAD, dek []byte, nonce [constants.WrapNonceBytes]byte) (WrappedDEK, error) {
	// Validate inputs: reject nil AEAD, check DEK length, and confirm AEAD nonce size.
	if aead == nil {
		return WrappedDEK{}, verrors.ErrNilAEAD
	}
	if len(dek) != constants.DEKBytes {
		return WrappedDEK{}, verrors.ErrMalformedParameters
	}
	if aead.NonceSize() != constants.WrapNonceBytes {
		return WrappedDEK{}, verrors.ErrMalformedParameters
	}

	// Seal the DEK bytes with the provided nonce and additional authenticated data.
	// The output (ciphertext + tag) must fit exactly in the [Payload] field.
	ct := aead.Seal(nil, nonce[:], dek, wrapAAD)
	if len(ct) != constants.DEKBytes+constants.GCMTagBytes {
		return WrappedDEK{}, verrors.ErrMalformedParameters
	}

	// Materialize the wrapped DEK (ephemeral pub, nonce, ciphertext+tag) in a stack-allocated struct.
	var out WrappedDEK
	out.Pub = pub
	out.Nonce = nonce
	copy(out.Payload[:], ct)

	return out, nil
}

// OpenWrappedDEK unwraps DEK bytes using wrapAAD.
func OpenWrappedDEK(aead cipher.AEAD, wrapAAD []byte, dek WrappedDEK) ([]byte, error) {
	// Check if the AEAD instance is nil.
	if aead == nil {
		return nil, verrors.ErrNilAEAD
	}

	// Attempt to decrypt the wrapped DEK using the same wrapAAD as at seal time
	// (usually prefix || metaRaw from [WrapAAD]).
	plain, err := aead.Open(nil, dek.Nonce[:], dek.Payload[:], wrapAAD)
	if err != nil {
		return nil, verrors.ErrDecrypt
	}

	// After successful open, ensure that the plaintext length matches the expected DEK size
	// (GCM should always return DEKBytes for v1 envelopes).
	if len(plain) != constants.DEKBytes {
		return nil, verrors.ErrMalformedParameters
	}

	// Copy the DEK plaintext to a new slice so callers can zero sensitive material
	// without mutating the stack-backed struct fields.
	out := make([]byte, constants.DEKBytes)
	copy(out, plain)
	return out, nil
}

//=============================================================================
// Helpers: wrap-key derivation and DEK-wrap AAD
//=============================================================================

// wrapAADPrefix binds DEK-wrap AEAD to the v1 envelope (prefix || metaRaw).
const wrapAADPrefix = "vault-wrap-dek-v1"

// hkdfWrapInfo is the HKDF context string for stretching the X25519 shared secret into the wrap key.
// It must stay stable across releases that read the same wire format.
const hkdfWrapInfo = "vault/v1/x25519-hkdf-sha256-aes256gcm/wrap-key"

// WrapAAD prefixes meta-derived AAD for DEK wrapping so it cannot be confused
// with other uses of the same key material.
func WrapAAD(metaRaw []byte) []byte {
	out := make([]byte, 0, len(wrapAADPrefix)+len(metaRaw))
	out = append(out, wrapAADPrefix...)
	out = append(out, metaRaw...)
	return out
}

// DeriveWrapKey derives the AES-256 wrap key from an ECDH shared secret using
// HKDF-SHA256.
func DeriveWrapKey(sharedSecret []byte) ([]byte, error) {
	// No salt: shared secret is already high-entropy.
	return hkdf.Key(sha256.New, sharedSecret, nil, hkdfWrapInfo, constants.WrapKeyBytes)
}
