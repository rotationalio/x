package kdf_test

// Tests for Argon2id Derive, RandSalt, and parameter validation.

import (
	cryptorand "crypto/rand"
	"io"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/keyring/kdf"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
)

//=============================================================================
// Derive validation
//=============================================================================

// TestDerive_rejectsNilPassword ensures a nil password slice returns before Argon2.
func TestDerive_rejectsNilPassword(t *testing.T) {
	salt := make([]byte, kdf.SaltBytes)
	_, err := kdf.Derive(nil, salt, kdf.MemoryConstrainedParams, lockerv1.SeedBytes)
	assert.ErrorIs(t, err, perrors.ErrNilPassword)
}

// TestDerive_rejectsBadSaltLength ensures only kdf.SaltBytes-length salts are accepted.
func TestDerive_rejectsBadSaltLength(t *testing.T) {
	_, err := kdf.Derive([]byte("pw"), make([]byte, kdf.SaltBytes-1), kdf.MemoryConstrainedParams, lockerv1.SeedBytes)
	assert.ErrorIs(t, err, perrors.ErrInvalidSalt)
}

// TestDerive_rejectsNonPositiveOutLen ensures outLen must be positive.
func TestDerive_rejectsNonPositiveOutLen(t *testing.T) {
	salt := make([]byte, kdf.SaltBytes)
	_, err := kdf.Derive([]byte("pw"), salt, kdf.MemoryConstrainedParams, 0)
	assert.ErrorIs(t, err, perrors.ErrInvalidOut)
}

//=============================================================================
// Derive round-trip
//=============================================================================

// TestDerive_FromSeed_roundtrip runs a tiny Argon2 profile then maps the seed to a v1 locker.
func TestDerive_FromSeed_roundtrip(t *testing.T) {
	// Tiny Argon2id parameters keep the test fast; production callers should use
	// DefaultParams or MemoryConstrainedParams.
	p := kdf.Params{Iterations: 1, MemoryKiB: 32, Threads: 1}
	salt, err := kdf.RandSalt()
	assert.Ok(t, err)

	// Derive a 32-byte seed, then map it to a v1 X25519 private key.
	seed, err := kdf.Derive([]byte("unit-test-password"), salt, p, lockerv1.SeedBytes)
	assert.Ok(t, err)
	lck, err := lockerv1.FromSeed(seed)
	assert.Ok(t, err)
	assert.NotNil(t, lck)
	assert.True(t, len(lck.KeyID()) > 0)
}

// TestRandSalt_propagatesReaderFailure ensures entropy read errors surface as ErrRandSalt.
func TestRandSalt_propagatesReaderFailure(t *testing.T) {
	// Replace crypto/rand.Reader with one that always returns io.EOF; cleanup restores it.
	orig := cryptorand.Reader
	t.Cleanup(func() { cryptorand.Reader = orig })
	cryptorand.Reader = eofReader{}

	// RandSalt must surface the entropy failure as ErrRandSalt rather than leaking the
	// raw io error, since callers identify this failure by the sentinel.
	_, err := kdf.RandSalt()
	assert.ErrorIs(t, err, perrors.ErrRandSalt)
}

// eofReader is a test double that always returns io.EOF from Read.
type eofReader struct{}

// Read implements io.Reader for eofReader.
func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }
