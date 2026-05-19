package gcm

// DEK-wrap AEAD, HKDF key derivation, and AAD construction for envelope encryption.

import (
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
)

// WrappedDEK is the fixed-layout DEK wrap segment (pub, nonce, ciphertext+tag).
// It matches [models.DekEnvelope] field-for-field for easy copying.
type WrappedDEK struct {
	Pub     [constants.X25519PubBytes]byte
	Nonce   [constants.WrapNonceBytes]byte
	Payload [constants.DEKBytes + constants.GCMTagBytes]byte
}

// NewWrapAEAD constructs DEK-wrap AEAD (AES-256-GCM) for a 32-byte wrap key.
func NewWrapAEAD(wrapKey []byte) (cipher.AEAD, error) {
	if len(wrapKey) != constants.WrapKeyBytes {
		return nil, perrors.ErrMalformedParameters
	}
	return newAEAD(wrapKey)
}

// SealWrappedDEK wraps dek with wrapAAD using ephemeral X25519 public key pub.
func SealWrappedDEK(pub [constants.X25519PubBytes]byte, aead cipher.AEAD, wrapAAD, dek []byte) (WrappedDEK, error) {
	// Reject nil AEAD instance.
	if aead == nil {
		return WrappedDEK{}, perrors.ErrNilAEAD
	}
	if len(dek) != constants.DEKBytes {
		return WrappedDEK{}, perrors.ErrMalformedParameters
	}
	if aead.NonceSize() != constants.WrapNonceBytes {
		return WrappedDEK{}, perrors.ErrMalformedParameters
	}

	var nonce [constants.WrapNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return WrappedDEK{}, perrors.ErrSealFailed
	}
	return SealWrappedDEKWithNonce(pub, aead, wrapAAD, dek, nonce)
}

// SealWrappedDEKWithNonce wraps dek using the given nonce (random in [SealWrappedDEK]).
func SealWrappedDEKWithNonce(pub [constants.X25519PubBytes]byte, aead cipher.AEAD, wrapAAD, dek []byte, nonce [constants.WrapNonceBytes]byte) (WrappedDEK, error) {
	if aead == nil {
		return WrappedDEK{}, perrors.ErrNilAEAD
	}
	if len(dek) != constants.DEKBytes {
		return WrappedDEK{}, perrors.ErrMalformedParameters
	}
	if aead.NonceSize() != constants.WrapNonceBytes {
		return WrappedDEK{}, perrors.ErrMalformedParameters
	}

	ct := aead.Seal(nil, nonce[:], dek, wrapAAD)
	if len(ct) != constants.DEKBytes+constants.GCMTagBytes {
		return WrappedDEK{}, perrors.ErrMalformedParameters
	}

	var out WrappedDEK
	out.Pub = pub
	out.Nonce = nonce
	copy(out.Payload[:], ct)
	return out, nil
}

// OpenWrappedDEK unwraps DEK bytes using wrapAAD. The returned slice owns its memory; callers
// should [purser.Zero] it after use to scrub the DEK from the heap.
func OpenWrappedDEK(aead cipher.AEAD, wrapAAD []byte, dek WrappedDEK) ([]byte, error) {
	if aead == nil {
		return nil, perrors.ErrNilAEAD
	}

	dst := make([]byte, 0, constants.DEKBytes)
	plain, err := aead.Open(dst, dek.Nonce[:], dek.Payload[:], wrapAAD)
	if err != nil {
		return nil, perrors.ErrDecrypt
	}
	if len(plain) != constants.DEKBytes {
		// Length mismatch must never leak DEK bytes to the caller.
		purser.Zero(plain)
		return nil, perrors.ErrMalformedParameters
	}
	return plain, nil
}

//=============================================================================
// Helpers: wrap-key derivation and DEK-wrap AAD
//=============================================================================

// WrapAADPrefix binds DEK-wrap AEAD to the v1 envelope; the full AAD is WrapAADPrefix || metaRaw.
const WrapAADPrefix = "purser-wrap-dek-v1"

// hkdfWrapInfo is the HKDF context string for stretching the X25519 shared secret into the wrap key.
const hkdfWrapInfo = "purser/v1/x25519-hkdf-sha256-aes256gcm/wrap-key"

// DeriveWrapKey derives the AES-256 wrap key from an ECDH shared secret using HKDF-SHA256.
func DeriveWrapKey(sharedSecret []byte) ([]byte, error) {
	return hkdf.Key(sha256.New, sharedSecret, nil, hkdfWrapInfo, constants.WrapKeyBytes)
}
