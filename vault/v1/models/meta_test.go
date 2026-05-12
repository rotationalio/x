package models_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/v1/errors"
	"go.rtnl.ai/x/vault/v1/models"
	"go.rtnl.ai/x/vault/v1/suite"
)

// TestMeta_roundtrip checks [models.Meta.MarshalBinary] and [models.Meta.UnmarshalBinary] preserve fields.
func TestMeta_roundtrip(t *testing.T) {
	m := models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          []byte{1, 2, 3},
		Namespace:      "ns-a",
	}
	b, err := m.MarshalBinary()
	assert.Ok(t, err)
	var got models.Meta
	assert.Ok(t, got.UnmarshalBinary(b))
	assert.Equal(t, m.PackageVersion, got.PackageVersion)
	assert.Equal(t, m.SuiteID, got.SuiteID)
	assert.Equal(t, m.KeyID, got.KeyID)
	assert.Equal(t, m.Namespace, got.Namespace)
}

// TestMetaFromPrivKey verifies [models.MetaFromPrivKey] fills version, suite, and key id from an X25519 key.
func TestMetaFromPrivKey(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	m, err := models.MetaFromPrivKey(priv)
	assert.Ok(t, err)
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
