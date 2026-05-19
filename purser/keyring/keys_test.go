package keys_test

import (
	cryptorand "crypto/rand"
	"io"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/vault/errors"
	"go.rtnl.ai/x/vault/keys"
)

// TestDerive_rejectsNilPassword ensures a nil password slice returns before Argon2.
func TestDerive_rejectsNilPassword(t *testing.T) {
	salt := make([]byte, keys.SaltBytes)
	_, err := keys.Derive(nil, salt, keys.MemoryConstrainedParams(), keys.DerivedKeyBytes)
	assert.ErrorIs(t, err, verrors.ErrNilPassword)
}

// TestDerive_rejectsBadSaltLength ensures only [keys.SaltBytes]-length salts are accepted.
func TestDerive_rejectsBadSaltLength(t *testing.T) {
	_, err := keys.Derive([]byte("pw"), make([]byte, keys.SaltBytes-1), keys.MemoryConstrainedParams(), keys.DerivedKeyBytes)
	assert.ErrorIs(t, err, verrors.ErrInvalidSalt)
}

// TestDerive_rejectsNonPositiveOutLen ensures outLen must be positive.
func TestDerive_rejectsNonPositiveOutLen(t *testing.T) {
	salt := make([]byte, keys.SaltBytes)
	_, err := keys.Derive([]byte("pw"), salt, keys.MemoryConstrainedParams(), 0)
	assert.ErrorIs(t, err, verrors.ErrInvalidOut)
}

// TestDerive_FromSeed_roundtrip runs a tiny Argon2 profile then maps the seed to an X25519 key.
func TestDerive_FromSeed_roundtrip(t *testing.T) {
	// Small memory so the test stays fast; not a production profile.
	p := keys.Params{Iterations: 1, MemoryKiB: 32, Threads: 1}
	salt, err := keys.RandSalt()
	assert.Ok(t, err)
	seed, err := keys.Derive([]byte("unit-test-password"), salt, p, keys.DerivedKeyBytes)
	assert.Ok(t, err)
	priv, err := keys.FromSeed(seed)
	assert.Ok(t, err)
	assert.NotNil(t, priv)
}

// TestFromSeed_rejectsWrongLength ensures only a 32-byte seed is accepted.
func TestFromSeed_rejectsWrongLength(t *testing.T) {
	_, err := keys.FromSeed(make([]byte, keys.DerivedKeyBytes-1))
	assert.ErrorIs(t, err, verrors.ErrInvalidSeed)
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

// TestRandSalt_propagatesReaderFailure ensures entropy read errors surface as [verrors.ErrRandSalt].
func TestRandSalt_propagatesReaderFailure(t *testing.T) {
	orig := cryptorand.Reader
	t.Cleanup(func() { cryptorand.Reader = orig })
	cryptorand.Reader = eofReader{}
	_, err := keys.RandSalt()
	assert.ErrorIs(t, err, verrors.ErrRandSalt)
}
