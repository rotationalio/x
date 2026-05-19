/*
Package v1 implements purser.Locker for the v1 envelope format: per-row metadata,
an ephemeral X25519 exchange, HKDF-derived keys, a wrapped per-row data key, and an
inner AES-256-GCM payload.
*/
package v1

// This file defines New and the envelope seal/open helpers for locker/v1.

import (
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"

	"go.rtnl.ai/x/purser"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	pgcm "go.rtnl.ai/x/purser/locker/v1/gcm"
	"go.rtnl.ai/x/purser/locker/v1/models"
	"go.rtnl.ai/x/purser/locker/v1/suite"
)

//=============================================================================
// Locker
//=============================================================================

// envLocker implements purser.Locker using an X25519 private key and envelope encryption.
type envLocker struct {
	priv     *ecdh.PrivateKey
	template models.Meta // namespace is set on each Seal call
}

// Ensure locker implements purser.Locker.
var _ purser.Locker = (*envLocker)(nil)

// New constructs a purser.Locker for the v1 envelope suite from an X25519 private key.
func New(priv *ecdh.PrivateKey) (purser.Locker, error) {
	if priv == nil {
		return nil, perrors.ErrNilPrivateKey
	}
	if priv.Curve() != ecdh.X25519() {
		return nil, perrors.ErrInvalidWrappingKey
	}

	kid := priv.PublicKey().Bytes()
	if len(kid) > constants.MaxKeyIDBytes {
		return nil, perrors.ErrMetaKeyIDTooLarge
	}

	meta := models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          append([]byte(nil), kid...),
		Namespace:      "",
	}
	if _, err := meta.MarshalBinary(); err != nil {
		return nil, err
	}

	return &envLocker{priv: priv, template: meta}, nil
}

// KeyID returns a defensive copy of the locker key id.
func (l *envLocker) KeyID() []byte {
	if l == nil {
		return nil
	}
	return append([]byte(nil), l.template.KeyID...)
}

// Seal encrypts plaintext under namespace and returns the v1 ciphertext wire blob.
func (l *envLocker) Seal(namespace string, plaintext []byte) ([]byte, error) {
	// Generate a fresh, random Data Encryption Key (DEK) for this row.
	dek := make([]byte, constants.DEKBytes)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, perrors.ErrSealFailed
	}
	defer purser.Zero(dek)

	// Generate a unique nonce for the inner AEAD encryption.
	var innerNonce [constants.InnerNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, innerNonce[:]); err != nil {
		return nil, perrors.ErrSealFailed
	}

	// Generate ephemeral X25519 key for envelope wrapping.
	ephPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, perrors.ErrSealFailed
	}

	// Generate a nonce for the envelope (wrapping) AEAD.
	var wrapNonce [constants.WrapNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, wrapNonce[:]); err != nil {
		return nil, perrors.ErrSealFailed
	}

	// Seal the plaintext and return the complete envelope using fresh keys/nonces.
	return l.sealWith(namespace, plaintext, dek, innerNonce, ephPriv, wrapNonce)
}

// Open parses wire, unwraps keys, verifies plaintext, and checks namespace matches requestedNS.
func (l *envLocker) Open(requestedNS string, wire []byte) ([]byte, error) {
	// Unmarshal the sealed wire into the Sealed structure.
	var msg models.Sealed
	if err := msg.UnmarshalBinary(wire); err != nil {
		return nil, err
	}

	// Ensure that the namespace matches the one requested.
	if msg.Meta.Namespace != requestedNS {
		return nil, perrors.ErrNamespaceMismatch
	}

	// wrapAAD is prefix||metaRaw and metaRaw aliases the trailing bytes of wrapAAD.
	wrapAAD, metaRaw, err := buildWrapAADAndMeta(msg.Meta)
	if err != nil {
		return nil, err
	}

	// Reconstruct the ephemeral public key for ECDH.
	epub, err := ecdh.X25519().NewPublicKey(msg.Dek.Pub[:])
	if err != nil {
		return nil, perrors.ErrDecrypt
	}

	// Perform ECDH with our private key and the ephemeral public key.
	shared, err := l.priv.ECDH(epub)
	if err != nil {
		return nil, perrors.ErrDecrypt
	}
	defer purser.Zero(shared)

	// Derive the wrapping key from the shared secret.
	wrapKey, err := pgcm.DeriveWrapKey(shared)
	if err != nil {
		return nil, err
	}
	defer purser.Zero(wrapKey)

	// Build AEAD for unwrapping the DEK.
	wrapAEAD, err := pgcm.NewWrapAEAD(wrapKey)
	if err != nil {
		return nil, err
	}

	// Unwrap and authenticate the DEK using the AEAD and metadata.
	wrapped := pgcm.WrappedDEK{Pub: msg.Dek.Pub, Nonce: msg.Dek.Nonce, Payload: msg.Dek.Payload}
	dek, err := pgcm.OpenWrappedDEK(wrapAEAD, wrapAAD, wrapped)
	if err != nil {
		return nil, err
	}
	defer purser.Zero(dek)

	// Build AEAD for decrypting the inner ciphertext.
	innerAEAD, err := pgcm.NewInnerAEAD(dek)
	if err != nil {
		return nil, err
	}

	// Open and verify the inner ciphertext with the decrypted DEK and metadata.
	plain, err := pgcm.OpenInner(innerAEAD, metaRaw, msg.Body.Nonce, msg.Body.Payload)
	if err != nil {
		return nil, err
	}

	return plain, nil
}

// ParseKeyID parses ciphertext metadata and returns the key identifier without decrypting.
func (l *envLocker) ParseKeyID(ciphertext []byte) (keyID []byte, err error) {
	var msg models.Sealed
	if err = msg.UnmarshalBinary(ciphertext); err != nil {
		return nil, err
	}
	return append([]byte(nil), msg.Meta.KeyID...), nil
}

//=============================================================================
// Envelope seal
//=============================================================================

// sealWith seals plaintext using fixed DEK, nonces, and ephemeral key.
func (l *envLocker) sealWith(namespace string, plaintext, dek []byte, innerNonce [constants.InnerNonceBytes]byte, ephPriv *ecdh.PrivateKey, wrapNonce [constants.WrapNonceBytes]byte) ([]byte, error) {
	defer purser.Zero(dek)

	// Prepare per-row metadata, copying the template and injecting this operation's namespace.
	row, err := l.template.WithNamespace(namespace)
	if err != nil {
		return nil, err
	}

	// wrapAAD is prefix||metaRaw and metaRaw aliases the trailing bytes of wrapAAD.
	wrapAAD, metaRaw, err := buildWrapAADAndMeta(row)
	if err != nil {
		return nil, err
	}

	// Build the AEAD used to encrypt the user's data (inner payload).
	innerAEAD, err := pgcm.NewInnerAEAD(dek)
	if err != nil {
		return nil, err
	}

	// Encrypt the plaintext (sealing the data and binding metadata as AAD).
	nonce, payload, err := pgcm.SealInnerWithNonce(innerAEAD, metaRaw, plaintext, innerNonce)
	if err != nil {
		return nil, err
	}
	body := models.Inner{Nonce: nonce, Payload: payload}

	// ECDH: derive a shared secret from ephemeral private and long-term public key.
	shared, err := ephPriv.ECDH(l.priv.PublicKey())
	if err != nil {
		return nil, err
	}
	defer purser.Zero(shared)

	// Stretch the shared secret into an envelope wrapping key.
	wrapKey, err := pgcm.DeriveWrapKey(shared)
	if err != nil {
		return nil, err
	}
	defer purser.Zero(wrapKey)

	// Build AEAD for the envelope (to wrap the DEK).
	wrapAEAD, err := pgcm.NewWrapAEAD(wrapKey)
	if err != nil {
		return nil, err
	}

	// Prepare the ephemeral public key to include in the wire format.
	var ephPub [constants.X25519PubBytes]byte
	copy(ephPub[:], ephPriv.PublicKey().Bytes())

	// Encrypt (wrap) the DEK for transport, sealing it with envelope AEAD and AAD (metadata).
	dekWire, err := pgcm.SealWrappedDEKWithNonce(ephPub, wrapAEAD, wrapAAD, dek, wrapNonce)
	if err != nil {
		return nil, err
	}
	dekEnv := models.DekEnvelope{Pub: dekWire.Pub, Nonce: dekWire.Nonce, Payload: dekWire.Payload}

	// Assemble the complete sealed wire, including all envelope components.
	sealed := models.Sealed{
		FormatVersion: constants.PackageVersion,
		Meta:          row,
		Dek:           dekEnv,
		Body:          body,
	}

	// Marshal the final sealed row as a single wire blob.
	return sealed.MarshalBinary()
}

// ExportTestBuildSealedRow builds deterministic wire bytes for golden tests.
func ExportTestBuildSealedRow(l purser.Locker, namespace string, plaintext, dek []byte, innerNonce [constants.InnerNonceBytes]byte, ephPriv *ecdh.PrivateKey, wrapNonce [constants.WrapNonceBytes]byte) ([]byte, error) {
	impl, ok := l.(*envLocker)
	if !ok {
		return nil, errors.New("locker/v1: invalid locker implementation")
	}
	return impl.sealWith(namespace, plaintext, dek, innerNonce, ephPriv, wrapNonce)
}

// buildWrapAADAndMeta marshals meta into a single buffer laid out as
// [pgcm.WrapAADPrefix][meta]. wrapAAD is the full buffer (AAD for DEK
// wrapping); metaRaw aliases the trailing meta bytes (AAD for the inner AEAD).
func buildWrapAADAndMeta(meta models.Meta) (wrapAAD, metaRaw []byte, err error) {
	metaSize, err := meta.MarshalBinarySize()
	if err != nil {
		return nil, nil, err
	}

	prefixLen := len(pgcm.WrapAADPrefix)
	buf := make([]byte, prefixLen+metaSize)
	copy(buf, pgcm.WrapAADPrefix)
	if _, err := meta.MarshalBinaryTo(buf[prefixLen:]); err != nil {
		return nil, nil, err
	}
	return buf, buf[prefixLen:], nil
}
