/*
Package locker implements purser.Locker for the v1 row format.

Role

  - Long-term X25519 private key identifies the locker ([envLocker.KeyID] is the public key bytes).
  - Each [envLocker.Seal] generates a fresh ephemeral X25519 keypair for that row only.

Seal and Open

  - A shared secret is computed using ECDH (ephemeral private with long-term public key on seal,
    long-term private with ephemeral public key from the wire on open).
  - The shared secret is expanded with HKDF-SHA256 (info "purser/v1/x25519-hkdf-sha256-aes256gcm/data-key", 32 bytes).
  - Plaintext is encrypted with AES-256-GCM; additional authenticated data is the marshaled per-row
    [models.Meta] (namespace, suite, key id, format version).
  - [envLocker.Open] validates the requested namespace from metadata before decrypting.

Wire layout

  - magic (4) || format version (1) || meta length u16 BE (2) || Meta (variable)
  - || ephemeral X25519 public key (32, fixed) || inner nonce (12) || ciphertext+tag (variable)

The ephemeral public key is the only envelope field besides Meta and the encrypted inner ciphertext.

Key construction

  - [New] from an X25519 [*ecdh.PrivateKey]
  - [FromSeed], [FromPassword], [FromPKCS8], and [FromKey] in keys.go (X25519 only for [FromKey])

Subpackages

  - models: wire framing (Sealed, Meta, EphPub, Inner)
  - gcm: HKDF data-key derivation and AES-GCM seal/open
  - constants: sizes, magic, format version
  - suite: recipe id x25519_hkdf_sha256_aes256_gcm
*/
package locker

// New, Seal, Open, and ParseKeyID for the v1 envelope locker.

import (
	"crypto/ecdh"
	"crypto/rand"
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
	// Generate a fresh nonce for the payload.
	var innerNonce [constants.InnerNonceBytes]byte
	if _, err := io.ReadFull(rand.Reader, innerNonce[:]); err != nil {
		return nil, perrors.ErrSealFailed
	}

	// Generate a fresh ephemeral key pair for the payload.
	ephPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, perrors.ErrSealFailed
	}

	// Compute the shared secret between the ephemeral key and the long-term private key.
	shared, err := ephPriv.ECDH(l.priv.PublicKey())
	if err != nil {
		return nil, err
	}
	defer purser.Zero(shared)

	// Derive the data key from the shared secret.
	dataKey, err := pgcm.DeriveDataKey(shared)
	if err != nil {
		return nil, err
	}
	defer purser.Zero(dataKey)

	// Construct the AEAD to seal the plaintext with the data key.
	innerAEAD, err := pgcm.NewInnerAEAD(dataKey)
	if err != nil {
		return nil, err
	}

	// Construct the metadata for the row with the namespace.
	row, err := l.template.WithNamespace(namespace)
	if err != nil {
		return nil, err
	}

	// Marshal the metadata for the row.
	metaRaw, err := row.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Seal the plaintext with the AEAD and the nonce.
	nonce, payload, err := pgcm.SealInnerWithNonce(innerAEAD, metaRaw, plaintext, innerNonce)
	if err != nil {
		return nil, err
	}

	// Construct the sealed row with the metadata, ephemeral public key, and nonce/payload.
	var ephPub models.EphPub
	copy(ephPub[:], ephPriv.PublicKey().Bytes())
	sealed := models.Sealed{
		FormatVersion: constants.PackageVersion,
		Meta:          row,
		Eph:           ephPub,
		Body:          models.Inner{Nonce: nonce, Payload: payload},
	}

	// Marshal the sealed row to the wire.
	return sealed.MarshalBinary()
}

// Open parses wire, derives the row key, verifies plaintext, and checks namespace matches requestedNS.
func (l *envLocker) Open(requestedNS string, wire []byte) ([]byte, error) {
	// Unmarshal the sealed row from the wire.
	var msg models.Sealed
	if err := msg.UnmarshalBinary(wire); err != nil {
		return nil, err
	}

	// Verify the namespace matches the requested namespace.
	if msg.Meta.Namespace != requestedNS {
		return nil, perrors.ErrNamespaceMismatch
	}

	// Construct the ephemeral public key from the wire.
	epub, err := ecdh.X25519().NewPublicKey(msg.Eph[:])
	if err != nil {
		return nil, perrors.ErrDecrypt
	}

	// Compute the shared secret between the long-term private key and the ephemeral public key.
	shared, err := l.priv.ECDH(epub)
	if err != nil {
		return nil, perrors.ErrDecrypt
	}
	defer purser.Zero(shared)

	// Derive the data key from the shared secret.
	dataKey, err := pgcm.DeriveDataKey(shared)
	if err != nil {
		return nil, err
	}
	defer purser.Zero(dataKey)

	// Construct the AEAD to open the ciphertext with the data key.
	innerAEAD, err := pgcm.NewInnerAEAD(dataKey)
	if err != nil {
		return nil, err
	}

	// Marshal the metadata for the additional authenticated data.
	metaRaw, err := msg.Meta.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Open the ciphertext with the AEAD and the nonce.
	return pgcm.OpenInner(innerAEAD, metaRaw, msg.Body.Nonce, msg.Body.Payload)
}

// ParseKeyID parses ciphertext metadata and returns the key identifier without decrypting.
func (l *envLocker) ParseKeyID(ciphertext []byte) (keyID []byte, err error) {
	var msg models.Sealed
	if err = msg.UnmarshalBinary(ciphertext); err != nil {
		return nil, err
	}
	return append([]byte(nil), msg.Meta.KeyID...), nil
}
