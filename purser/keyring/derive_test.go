package keyring_test

import (
	cryptorand "crypto/rand"
	"io"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/purser/errors"
	keyring "go.rtnl.ai/x/purser/keyring"
	v1 "go.rtnl.ai/x/purser/locker/v1"
)

// TestDerive_rejectsNilPassword ensures a nil password slice returns before Argon2.
func TestDerive_rejectsNilPassword(t *testing.T) {
	salt := make([]byte, keyring.SaltBytes)
	_, err := keyring.Derive(nil, salt, keyring.MemoryConstrainedParams(), v1.SeedBytes)
	assert.ErrorIs(t, err, verrors.ErrNilPassword)
}

// TestDerive_rejectsBadSaltLength ensures only [keyring.SaltBytes]-length salts are accepted.
func TestDerive_rejectsBadSaltLength(t *testing.T) {
	_, err := keyring.Derive([]byte("pw"), make([]byte, keyring.SaltBytes-1), keyring.MemoryConstrainedParams(), v1.SeedBytes)
	assert.ErrorIs(t, err, verrors.ErrInvalidSalt)
}

// TestDerive_rejectsNonPositiveOutLen ensures outLen must be positive.
func TestDerive_rejectsNonPositiveOutLen(t *testing.T) {
	salt := make([]byte, keyring.SaltBytes)
	_, err := keyring.Derive([]byte("pw"), salt, keyring.MemoryConstrainedParams(), 0)
	assert.ErrorIs(t, err, verrors.ErrInvalidOut)
}

// TestDerive_FromSeed_roundtrip runs a tiny Argon2 profile then maps the seed to a v1 locker key.
func TestDerive_FromSeed_roundtrip(t *testing.T) {
	// Tiny Argon2id parameters keep the test fast; production callers should use
	// DefaultParams or MemoryConstrainedParams.
	p := keyring.Params{Iterations: 1, MemoryKiB: 32, Threads: 1}
	salt, err := keyring.RandSalt()
	assert.Ok(t, err)

	// Derive a 32-byte seed, then map it to a v1 X25519 private key.
	seed, err := keyring.Derive([]byte("unit-test-password"), salt, p, v1.SeedBytes)
	assert.Ok(t, err)
	priv, err := v1.FromSeed(seed)

	// Both calls succeed and the derived private key is non-nil — proving the
	// password→seed→key path produces a usable locker key.
	assert.Ok(t, err)
	assert.NotNil(t, priv)
}

// TestFromSeed_rejectsWrongLength ensures only a v1 seed-length buffer is accepted.
func TestFromSeed_rejectsWrongLength(t *testing.T) {
	_, err := v1.FromSeed(make([]byte, v1.SeedBytes-1))
	assert.ErrorIs(t, err, verrors.ErrInvalidSeed)
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

// TestRandSalt_propagatesReaderFailure ensures entropy read errors surface as [verrors.ErrRandSalt].
func TestRandSalt_propagatesReaderFailure(t *testing.T) {
	// Replace the package-level rand.Reader with one that always returns io.EOF;
	// cleanup restores the original after the test.
	orig := cryptorand.Reader
	t.Cleanup(func() { cryptorand.Reader = orig })
	cryptorand.Reader = eofReader{}

	// RandSalt must surface the entropy failure as ErrRandSalt rather than leaking the
	// raw io error, since callers identify this failure by the sentinel.
	_, err := keyring.RandSalt()
	assert.ErrorIs(t, err, verrors.ErrRandSalt)
}
