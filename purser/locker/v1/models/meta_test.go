package models_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/models"
	"go.rtnl.ai/x/purser/locker/v1/suite"
)

// TestMeta_roundtrip checks [models.Meta.MarshalBinary] and [models.Meta.UnmarshalBinary] preserve fields.
func TestMeta_roundtrip(t *testing.T) {
	// A fully populated Meta covering all four fields: version, suite, key id, and namespace.
	m := models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          []byte{1, 2, 3},
		Namespace:      "ns-a",
	}

	// Marshal then unmarshal into a fresh value.
	b, err := m.MarshalBinary()
	assert.Ok(t, err)
	var got models.Meta

	// All four fields survive byte-for-byte. Field-by-field assertions (rather than
	// assert.Equal on the struct) make any future drift in a single field obvious.
	assert.Ok(t, got.UnmarshalBinary(b))
	assert.Equal(t, m.PackageVersion, got.PackageVersion)
	assert.Equal(t, m.SuiteID, got.SuiteID)
	assert.Equal(t, m.KeyID, got.KeyID)
	assert.Equal(t, m.Namespace, got.Namespace)
}

// TestMetaFromPrivKey verifies [models.MetaFromPrivKey] fills version, suite, and key id from an X25519 key.
func TestMetaFromPrivKey(t *testing.T) {
	// A fresh X25519 key — the only input the helper consumes.
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)

	// Build the template Meta from the private key.
	m, err := models.MetaFromPrivKey(priv)
	assert.Ok(t, err)

	// Version and suite are pinned to v1; key id matches the X25519 public key bytes;
	// namespace is left empty for callers to fill via WithNamespace at seal time.
	assert.Equal(t, constants.PackageVersion, m.PackageVersion)
	assert.Equal(t, suite.X25519HKDFSHA256AES256GCM, m.SuiteID)
	assert.Equal(t, priv.PublicKey().Bytes(), m.KeyID)
	assert.Equal(t, "", m.Namespace)
}

// TestMetaFromPrivKey_nil asserts a nil private key returns [verrors.ErrNilPrivateKey].
func TestMetaFromPrivKey_nil(t *testing.T) {
	_, err := models.MetaFromPrivKey(nil)
	assert.ErrorIs(t, err, verrors.ErrNilPrivateKey)
}

// TestMetaFromPrivKey_nonX25519 asserts a non-X25519 key returns [verrors.ErrInvalidWrappingKey].
func TestMetaFromPrivKey_nonX25519(t *testing.T) {
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	_, err = models.MetaFromPrivKey(priv)
	assert.ErrorIs(t, err, verrors.ErrInvalidWrappingKey)
}
