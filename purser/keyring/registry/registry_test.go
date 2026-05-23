package registry_test

// Tests for edition-dispatched locker construction and ParseKeyID.

import (
	"crypto/ecdh"
	crand "crypto/rand"
	"crypto/x509"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold/holdtest"
	"go.rtnl.ai/x/purser/keyring/kdf"
	"go.rtnl.ai/x/purser/keyring/registry"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
)

// TestEditions_includesV1 verifies the built-in registry lists locker v1.
func TestEditions_includesV1(t *testing.T) {
	got := registry.Editions()
	found := false
	for _, e := range got {
		if e == constv1.Edition {
			found = true
		}
	}
	assert.True(t, found)
}

// TestFromSeed_v1_roundtrip verifies v1 seed dispatch produces a usable locker.
func TestFromSeed_v1_roundtrip(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	for i := range seed {
		seed[i] = byte(i)
	}
	lck, err := registry.FromSeed(constv1.Edition, seed)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

// TestFromSeed_unsupportedEdition rejects unknown edition strings.
func TestFromSeed_unsupportedEdition(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	unsupported := []string{
		"v0",        // completely unknown
		"",          // empty string
		"V1",        // wrong case
		"v1beta1",   // plausible but not registered
		"edition-X", // random
	}
	for _, ed := range unsupported {
		_, err := registry.FromSeed(ed, seed)
		assert.ErrorIs(t, err, perrors.ErrUnsupportedLockerVersion, ed)
	}
}

// TestFromPassword_v1_roundtrip verifies password-based v1 locker construction.
func TestFromPassword_v1_roundtrip(t *testing.T) {
	salt, err := kdf.RandSalt()
	assert.Ok(t, err)
	lck, err := registry.FromPassword(constv1.Edition, []byte("pw"), salt, kdf.MemoryConstrainedParams)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

// TestParseKeyID_v1Wire extracts the sealing key id from v1 ciphertext.
func TestParseKeyID_v1Wire(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	_, err := crand.Read(seed)
	assert.Ok(t, err)
	lck, err := registry.FromSeed(constv1.Edition, seed)
	assert.Ok(t, err)
	wire, err := lck.Seal("ns", []byte("data"))
	assert.Ok(t, err)
	kid, err := registry.ParseKeyID(wire)
	assert.Ok(t, err)
	assert.Equal(t, lck.KeyID(), kid)
}

// TestParseKeyID_unrecognized rejects garbage and nulllocker holdtest wire.
func TestParseKeyID_unrecognized(t *testing.T) {
	_, err := registry.ParseKeyID([]byte("too-short"))
	assert.ErrorIs(t, err, perrors.ErrUnrecognizedCiphertext)

	wire := holdtest.Ciphertext(t, "ns", []byte("plain"))
	_, err = registry.ParseKeyID(wire)
	assert.ErrorIs(t, err, perrors.ErrUnrecognizedCiphertext)
}

// TestFromPKCS8_roundtrip loads a locker from PKCS#8 DER via edition dispatch.
func TestFromPKCS8_roundtrip(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	assert.Ok(t, err)
	lck, err := registry.FromPKCS8(der)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

// TestFromKey_roundtrip loads a locker from an X25519 private key.
func TestFromKey_roundtrip(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lck, err := registry.FromKey(priv)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}
