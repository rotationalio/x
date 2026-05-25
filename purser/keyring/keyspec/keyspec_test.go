package keyspec_test

// Tests for KeySpec constructors, Locker lifecycle, and edition guards.

import (
	"crypto/ecdh"
	crand "crypto/rand"
	"crypto/x509"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/keyring/kdf"
	"go.rtnl.ai/x/purser/keyring/keyspec"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
)

//=============================================================================
// Constructor validation
//=============================================================================

// TestNewPassword_rejectsEmptyPassword ensures a zero-length password is rejected.
func TestNewPassword_rejectsEmptyPassword(t *testing.T) {
	salt := make([]byte, kdf.SaltBytes)
	_, err := keyspec.NewPassword(nil, salt, kdf.MemoryConstrainedParams, constv1.Edition)
	assert.ErrorIs(t, err, perrors.ErrNilPassword)
}

// TestNewPassword_rejectsBadSaltLength ensures only kdf.SaltBytes-length salts are accepted.
func TestNewPassword_rejectsBadSaltLength(t *testing.T) {
	_, err := keyspec.NewPassword([]byte("pw"), make([]byte, kdf.SaltBytes-1), kdf.MemoryConstrainedParams, constv1.Edition)
	assert.ErrorIs(t, err, perrors.ErrInvalidSalt)
}

// TestNewSeed_rejectsWrongLength ensures seed length must match v1 SeedBytes.
func TestNewSeed_rejectsWrongLength(t *testing.T) {
	_, err := keyspec.NewSeed(make([]byte, lockerv1.SeedBytes-1), constv1.Edition)
	assert.ErrorIs(t, err, perrors.ErrInvalidSeed)
}

// TestNewPKCS8_rejectsEmptyDER ensures empty PKCS#8 input is rejected.
func TestNewPKCS8_rejectsEmptyDER(t *testing.T) {
	_, err := keyspec.NewPKCS8(nil, constv1.Edition)
	assert.ErrorIs(t, err, perrors.ErrInvalidKeySpec)
}

// TestNewPrivateKey_rejectsNil ensures a nil private key is rejected.
func TestNewPrivateKey_rejectsNil(t *testing.T) {
	_, err := keyspec.NewPrivateKey(nil, constv1.Edition)
	assert.ErrorIs(t, err, perrors.ErrInvalidKeySpec)
}

// TestKeySpec_rejectsEmptyEdition ensures every constructor requires a non-empty edition string.
func TestKeySpec_rejectsEmptyEdition(t *testing.T) {
	salt := make([]byte, kdf.SaltBytes)
	seed := make([]byte, lockerv1.SeedBytes)

	_, err := keyspec.NewPassword([]byte("pw"), salt, kdf.MemoryConstrainedParams, "")
	assert.ErrorIs(t, err, perrors.ErrInvalidEdition)

	_, err = keyspec.NewSeed(seed, "")
	assert.ErrorIs(t, err, perrors.ErrInvalidEdition)

	_, err = keyspec.NewPKCS8([]byte{0x30, 0x01, 0x02}, "")
	assert.ErrorIs(t, err, perrors.ErrInvalidEdition)

	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	_, err = keyspec.NewPrivateKey(priv, "")
	assert.ErrorIs(t, err, perrors.ErrInvalidEdition)
}

//=============================================================================
// Locker happy paths
//=============================================================================

// TestLocker_fromSeed builds a v1 locker from a fixed seed spec.
func TestLocker_fromSeed(t *testing.T) {
	seed := testSeed(t)
	spec, err := keyspec.NewSeed(seed, constv1.Edition)
	assert.Ok(t, err)

	lck, err := spec.Locker()
	assert.Ok(t, err)
	assert.NotNil(t, lck)
	assert.True(t, len(lck.KeyID()) > 0)
	assert.Equal(t, constv1.Edition, lck.Edition())
}

// TestLocker_fromPassword builds a v1 locker from password material with tiny Argon2 params.
func TestLocker_fromPassword(t *testing.T) {
	p := kdf.Params{Iterations: 1, MemoryKiB: 32, Threads: 1}
	salt, err := kdf.RandSalt()
	assert.Ok(t, err)

	spec, err := keyspec.NewPassword([]byte("unit-test-password"), salt, p, constv1.Edition)
	assert.Ok(t, err)

	lck, err := spec.Locker()
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

// TestLocker_fromPKCS8 builds a v1 locker from PKCS#8 DER.
func TestLocker_fromPKCS8(t *testing.T) {
	der := testPKCS8DER(t)
	spec, err := keyspec.NewPKCS8(der, constv1.Edition)
	assert.Ok(t, err)

	lck, err := spec.Locker()
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

// TestLocker_fromPrivateKey builds a v1 locker from an in-process X25519 key.
func TestLocker_fromPrivateKey(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)

	spec, err := keyspec.NewPrivateKey(priv, constv1.Edition)
	assert.Ok(t, err)

	lck, err := spec.Locker()
	assert.Ok(t, err)
	assert.NotNil(t, lck)
}

//=============================================================================
// Single-use wipe
//=============================================================================

// TestLocker_singleUseWipe rejects a second Locker call after the spec is consumed.
func TestLocker_singleUseWipe(t *testing.T) {
	spec, err := keyspec.NewSeed(testSeed(t), constv1.Edition)
	assert.Ok(t, err)

	_, err = spec.Locker()
	assert.Ok(t, err)

	_, err = spec.Locker()
	assert.ErrorIs(t, err, perrors.ErrInvalidKeySpec)
}

//=============================================================================
// Edition guards
//=============================================================================

// TestLocker_pkcs8EditionMismatch rejects a spec edition that does not match the parsed locker.
func TestLocker_pkcs8EditionMismatch(t *testing.T) {
	der := testPKCS8DER(t)
	spec, err := keyspec.NewPKCS8(der, "v0")
	assert.Ok(t, err)

	_, err = spec.Locker()
	assert.ErrorIs(t, err, perrors.ErrUnsupportedLockerVersion)
}

// TestLocker_privateKeyEditionMismatch rejects a spec edition that does not match the parsed locker.
func TestLocker_privateKeyEditionMismatch(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)

	spec, err := keyspec.NewPrivateKey(priv, "v0")
	assert.Ok(t, err)

	_, err = spec.Locker()
	assert.ErrorIs(t, err, perrors.ErrUnsupportedLockerVersion)
}

//=============================================================================
// Helpers
//=============================================================================

// testSeed returns a deterministic 32-byte v1 seed for unit tests.
func testSeed(t *testing.T) []byte {
	t.Helper()
	seed := make([]byte, lockerv1.SeedBytes)
	for i := range seed {
		seed[i] = byte(i)
	}
	return seed
}

// testPKCS8DER returns PKCS#8 DER for a generated X25519 private key.
func testPKCS8DER(t *testing.T) []byte {
	t.Helper()
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	assert.Ok(t, err)
	return der
}
