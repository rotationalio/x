package locker_test

// Direct tests for locker seal, open, key-id parsing, and constructor validation.

import (
	"crypto/ecdh"
	crand "crypto/rand"
	"crypto/x509"
	"io"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/keyring/kdf"
	"go.rtnl.ai/x/purser/locker"
	"go.rtnl.ai/x/purser/locker/lockertest"
	lockerv1 "go.rtnl.ai/x/purser/locker/v1"
	"go.rtnl.ai/x/purser/locker/v1/constants"
)

//=============================================================================
// Tests: conformance
//=============================================================================

// TestLocker_conforms runs the shared locker conformance suite against locker.
func TestLocker_conforms(t *testing.T) {
	err := lockertest.LockerConforms(func() (locker.Locker, error) {
		priv, err := ecdh.X25519().GenerateKey(crand.Reader)
		if err != nil {
			return nil, err
		}
		return lockerv1.New(priv)
	})
	assert.Ok(t, err)
}

//=============================================================================
// Tests: New
//=============================================================================

// TestNew_nilKey verifies New rejects a nil private key.
func TestNew_nilKey(t *testing.T) {
	_, err := lockerv1.New(nil)
	assert.ErrorIs(t, err, perrors.ErrNilPrivateKey)
}

// TestNew_nonX25519Key verifies New rejects a P-256 key.
func TestNew_nonX25519Key(t *testing.T) {
	priv, err := ecdh.P256().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	_, err = lockerv1.New(priv)
	assert.ErrorIs(t, err, perrors.ErrInvalidWrappingKey)
}

// TestNew_ok verifies New succeeds with a valid X25519 key.
func TestNew_ok(t *testing.T) {
	_, lck := freshLocker(t)
	assert.NotNil(t, lck)
}

// TestLocker_edition verifies Version, Edition, Recipe, and Context match v1 constants.
func TestLocker_edition(t *testing.T) {
	_, lck := freshLocker(t)
	l, ok := lck.(locker.Locker)
	assert.True(t, ok)
	assert.Equal(t, constants.Version, l.Version())
	assert.Equal(t, constants.Edition, l.Edition())
	assert.Equal(t, constants.Recipe, l.Recipe())
	assert.Equal(t, constants.Context, l.Context())
}

//=============================================================================
// Tests: KeyID
//=============================================================================

// TestKeyID_matchesPublicKey verifies KeyID returns the X25519 public key bytes.
func TestKeyID_matchesPublicKey(t *testing.T) {
	priv, lck := freshLocker(t)
	assert.Equal(t, priv.PublicKey().Bytes(), lck.KeyID())
}

// TestKeyID_defensiveCopy verifies mutating the returned key ID does not affect the locker.
func TestKeyID_defensiveCopy(t *testing.T) {
	// Take a copy of the key id, then deliberately corrupt the first byte.
	_, lck := freshLocker(t)
	kid1 := lck.KeyID()
	kid1[0] ^= 0xff

	// Ask for the key id again — if KeyID returns its internal slice, both copies
	// would now share the corrupted byte.
	kid2 := lck.KeyID()

	// The second copy must still hold the original value, proving KeyID returns a
	// defensive copy rather than the locker's internal slice.
	assert.True(t, kid1[0] != kid2[0])
}

//=============================================================================
// Tests: Seal / Open round-trip
//=============================================================================

// TestSealOpen_roundtrip verifies plaintext survives a Seal/Open round-trip.
func TestSealOpen_roundtrip(t *testing.T) {
	// A fresh locker and a known plaintext.
	_, lck := freshLocker(t)
	plain := []byte("hello v1 locker")

	// Seal then immediately open under the same namespace.
	wire, err := lck.Seal("ns", plain)
	assert.Ok(t, err)
	got, err := lck.Open("ns", wire)

	// The decrypted bytes match the original plaintext exactly.
	assert.Ok(t, err)
	assert.Equal(t, plain, got)
}

// TestSealOpen_emptyPlaintext verifies both an explicit empty slice and a nil slice
// round-trip correctly (Go's AEAD treats them identically; one test covers both).
func TestSealOpen_emptyPlaintext(t *testing.T) {
	_, lck := freshLocker(t)

	for _, plain := range [][]byte{nil, {}} {
		wire, err := lck.Seal("ns", plain)
		assert.Ok(t, err)
		got, err := lck.Open("ns", wire)
		assert.Ok(t, err)
		assert.Equal(t, 0, len(got))
	}
}

// TestSealOpen_emptyNamespace verifies an empty namespace is valid.
func TestSealOpen_emptyNamespace(t *testing.T) {
	_, lck := freshLocker(t)

	wire, err := lck.Seal("", []byte("data"))
	assert.Ok(t, err)
	got, err := lck.Open("", wire)
	assert.Ok(t, err)
	assert.Equal(t, []byte("data"), got)
}

// TestSeal_uniqueCiphertexts verifies repeated seals of identical plaintext produce
// distinct wire blobs. Eight iterations is overkill statistically but cheap; matching
// wire bytes across any pair would indicate DEK or nonce reuse, which is catastrophic
// for AES-GCM.
func TestSeal_uniqueCiphertexts(t *testing.T) {
	_, lck := freshLocker(t)

	const n = 8
	seen := make(map[string]struct{}, n)
	for range n {
		w, err := lck.Seal("ns", []byte("same"))
		assert.Ok(t, err)
		key := string(w)
		_, dup := seen[key]
		assert.True(t, !dup, "duplicate ciphertext across %d Seal calls", n)
		seen[key] = struct{}{}
	}
}

//=============================================================================
// Tests: Seal negative cases
//=============================================================================

// sealedPreambleBytes is magic(4) + formatVersion(1) + metaLen u16 BE(2).
const sealedPreambleBytes = 4 + 1 + 2

// TestSeal_entropyFailure maps [crypto/rand.Reader] failures during Seal to [perrors.ErrSealFailed].
func TestSeal_entropyFailure(t *testing.T) {
	_, lck := freshLocker(t)
	withFailingEntropy(t)
	_, err := lck.Seal("ns", []byte("x"))
	assert.ErrorIs(t, err, perrors.ErrSealFailed)
}

// TestSeal_oversizedNamespace rejects namespaces longer than the wire cap.
func TestSeal_oversizedNamespace(t *testing.T) {
	_, lck := freshLocker(t)
	big := string(make([]byte, constants.MaxNamespaceBytes+1))
	_, err := lck.Seal(big, []byte("data"))
	assert.ErrorIs(t, err, perrors.ErrMetaNamespaceTooLarge)
}

//=============================================================================
// Tests: Open negative cases
//=============================================================================

// TestOpen_wrongNamespace ensures namespace mismatch returns ErrNamespaceMismatch.
func TestOpen_wrongNamespace(t *testing.T) {
	// Seal under "ns-a".
	_, lck := freshLocker(t)
	wire, err := lck.Seal("ns-a", []byte("data"))
	assert.Ok(t, err)

	// Attempt to open the same wire under a different namespace ("ns-b") and assert the
	// locker rejects it with the namespace-binding sentinel.
	_, err = lck.Open("ns-b", wire)
	assert.ErrorIs(t, err, perrors.ErrNamespaceMismatch)
}

// TestOpen_wrongKey ensures opening with a different key fails.
func TestOpen_wrongKey(t *testing.T) {
	// Two independent lockers; the second has no relation to the first's keypair.
	_, lckA := freshLocker(t)
	_, lckB := freshLocker(t)

	// Seal with lckA, then attempt to open with lckB.
	wire, err := lckA.Seal("ns", []byte("data"))
	assert.Ok(t, err)
	_, err = lckB.Open("ns", wire)

	// Open must fail — lckB cannot derive the same row key from the ephemeral pubkey.
	assert.Error(t, err)
}

// TestOpen_tamperedMeta flips a byte in the framed Meta region; inner GCM AAD no longer matches.
func TestOpen_tamperedMeta(t *testing.T) {
	_, lck := freshLocker(t)
	wire, err := lck.Seal("ns", []byte("data"))
	assert.Ok(t, err)

	tampered := append([]byte(nil), wire...)
	// Meta layout: version, suite, keyID len, keyID… — flip the first KeyID byte.
	tampered[sealedPreambleBytes+4] ^= 0xff

	_, err = lck.Open("ns", tampered)
	assert.Error(t, err)
	assert.ErrorIs(t, err, perrors.ErrDecrypt)
}

// TestOpen_truncatedWire ensures truncated ciphertext returns an error.
func TestOpen_truncatedWire(t *testing.T) {
	_, lck := freshLocker(t)

	wire, err := lck.Seal("ns", []byte("data"))
	assert.Ok(t, err)

	_, err = lck.Open("ns", wire[:10])
	assert.Error(t, err)
}

// TestOpen_tamperedWire ensures tampered ciphertext fails AEAD authentication.
func TestOpen_tamperedWire(t *testing.T) {
	// Seal a row, then flip the last byte (which lives inside the inner GCM tag region)
	// to simulate an attacker mutating ciphertext in transit or at rest.
	_, lck := freshLocker(t)
	wire, err := lck.Seal("ns", []byte("data"))
	assert.Ok(t, err)
	wire[len(wire)-1] ^= 0xff

	// Open must fail because GCM's authentication tag no longer verifies.
	_, err = lck.Open("ns", wire)
	assert.Error(t, err)
}

// TestOpen_emptyWire returns an error on empty input.
func TestOpen_emptyWire(t *testing.T) {
	_, lck := freshLocker(t)
	_, err := lck.Open("ns", nil)
	assert.Error(t, err)
}

// TestOpen_badMagic returns ErrBadMagic when the magic prefix is wrong.
func TestOpen_badMagic(t *testing.T) {
	// Seal a row, then overwrite the wire's magic prefix so the framing layer cannot
	// recognize the blob as a v1 envelope.
	_, lck := freshLocker(t)
	wire, err := lck.Seal("ns", []byte("data"))
	assert.Ok(t, err)
	wire[0] = 'X'
	wire[1] = 'X'

	// Open returns ErrBadMagic before any AEAD work — the framing check rejects early.
	_, err = lck.Open("ns", wire)
	assert.ErrorIs(t, err, perrors.ErrBadMagic)
}

//=============================================================================
// Tests: ParseKeyID
//=============================================================================

// TestParseKeyID_matchesKeyID extracts the correct key ID from sealed wire.
func TestParseKeyID_matchesKeyID(t *testing.T) {
	// Produce a sealed wire blob with a known key id.
	_, lck := freshLocker(t)
	wire, err := lck.Seal("ns", []byte("data"))
	assert.Ok(t, err)

	// Extract the key id directly from the wire (no decryption performed).
	kid, err := lck.ParseKeyID(wire)

	// The parsed key id matches the locker's reported key id, which is what the keyring
	// uses to route ciphertext during multi-version Open.
	assert.Ok(t, err)
	assert.Equal(t, lck.KeyID(), kid)
}

// TestParseKeyID_malformed returns an error for garbage input.
func TestParseKeyID_malformed(t *testing.T) {
	_, lck := freshLocker(t)
	_, err := lck.ParseKeyID([]byte("too-short"))
	assert.Error(t, err)
}

// TestParseKeyID_truncated returns an error for truncated wire.
func TestParseKeyID_truncated(t *testing.T) {
	_, lck := freshLocker(t)

	wire, err := lck.Seal("ns", []byte("data"))
	assert.Ok(t, err)

	_, err = lck.ParseKeyID(wire[:5])
	assert.Error(t, err)
}

//=============================================================================
// Tests: FromSeed
//=============================================================================

// TestFromSeed_ok verifies a 32-byte seed produces a valid locker.
func TestFromSeed_ok(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	for i := range seed {
		seed[i] = byte(i)
	}

	lck, err := lockerv1.FromSeed(seed)
	assert.Ok(t, err)
	assert.True(t, len(lck.KeyID()) > 0)
}

// TestFromSeed_wrongLength rejects seeds that are not exactly 32 bytes.
func TestFromSeed_wrongLength(t *testing.T) {
	_, err := lockerv1.FromSeed(make([]byte, 31))
	assert.ErrorIs(t, err, perrors.ErrInvalidSeed)

	_, err = lockerv1.FromSeed(make([]byte, 33))
	assert.ErrorIs(t, err, perrors.ErrInvalidSeed)
}

// TestFromSeed_deterministic verifies the same seed always produces the same key ID.
func TestFromSeed_deterministic(t *testing.T) {
	seed := make([]byte, lockerv1.SeedBytes)
	for i := range seed {
		seed[i] = byte(i + 42)
	}

	lck1, err := lockerv1.FromSeed(seed)
	assert.Ok(t, err)
	lck2, err := lockerv1.FromSeed(seed)
	assert.Ok(t, err)
	assert.Equal(t, lck1.KeyID(), lck2.KeyID())
}

//=============================================================================
// Tests: FromPKCS8 / FromKey
//=============================================================================

// TestFromKey_ok accepts a raw X25519 private key.
func TestFromKey_ok(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	lck, err := lockerv1.FromKey(priv)
	assert.Ok(t, err)
	assert.Equal(t, priv.PublicKey().Bytes(), lck.KeyID())
}

// TestFromPKCS8_roundtrip marshals an X25519 key to PKCS#8 and loads a locker from it.
func TestFromPKCS8_roundtrip(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	assert.Ok(t, err)

	lck, err := lockerv1.FromPKCS8(der)
	assert.Ok(t, err)
	assert.Equal(t, priv.PublicKey().Bytes(), lck.KeyID())
}

// TestFromPKCS8_invalidDER rejects malformed PKCS#8 input.
func TestFromPKCS8_invalidDER(t *testing.T) {
	_, err := lockerv1.FromPKCS8([]byte{0x30, 0x01, 0x02})
	assert.Error(t, err)
}

// TestFromPKCS8_rejectsP256 rejects PKCS#8 material for a non-X25519 key.
func TestFromPKCS8_rejectsP256(t *testing.T) {
	priv, err := ecdh.P256().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	assert.Ok(t, err)
	_, err = lockerv1.FromPKCS8(der)
	assert.ErrorIs(t, err, perrors.ErrInvalidWrappingKey)
}

// TestFromKey_rejectsP256 rejects a non-X25519 ECDH private key.
func TestFromKey_rejectsP256(t *testing.T) {
	priv, err := ecdh.P256().GenerateKey(crand.Reader)
	assert.Ok(t, err)
	_, err = lockerv1.FromKey(priv)
	assert.ErrorIs(t, err, perrors.ErrInvalidWrappingKey)
}

// TestFromKey_rejectsWrongType rejects non-ECDH key material.
func TestFromKey_rejectsWrongType(t *testing.T) {
	_, err := lockerv1.FromKey("not-a-key")
	assert.ErrorIs(t, err, perrors.ErrInvalidWrappingKey)
}

//=============================================================================
// Tests: FromPassword
//=============================================================================

// TestFromPassword_roundtrip derives a locker from a password and performs a seal/open cycle.
func TestFromPassword_roundtrip(t *testing.T) {
	// A random salt and the memory-constrained Argon2id profile (cheap enough for tests).
	salt, err := kdf.RandSalt()
	assert.Ok(t, err)

	// Derive a locker from the password+salt, then seal and open a small payload.
	lck, err := lockerv1.FromPassword([]byte("test-password"), salt, kdf.MemoryConstrainedParams)
	assert.Ok(t, err)
	assert.True(t, len(lck.KeyID()) > 0)

	wire, err := lck.Seal("ns", []byte("hello"))
	assert.Ok(t, err)
	got, err := lck.Open("ns", wire)

	// Round-trip is lossless — confirming password derivation produces a usable locker.
	assert.Ok(t, err)
	assert.Equal(t, []byte("hello"), got)
}

// TestFromPassword_deterministic verifies the same password+salt always yields the same key ID.
func TestFromPassword_deterministic(t *testing.T) {
	// One shared salt; both derivations use identical password and Argon2id parameters.
	salt, err := kdf.RandSalt()
	assert.Ok(t, err)

	// Derive twice from the same inputs.
	lck1, err := lockerv1.FromPassword([]byte("pw"), salt, kdf.MemoryConstrainedParams)
	assert.Ok(t, err)
	lck2, err := lockerv1.FromPassword([]byte("pw"), salt, kdf.MemoryConstrainedParams)
	assert.Ok(t, err)

	// Key IDs match, demonstrating the derive→key step is deterministic for fixed
	// (password, salt, params) — required so callers can re-derive the same locker.
	assert.Equal(t, lck1.KeyID(), lck2.KeyID())
}

// TestFromPassword_nilPassword propagates the ErrNilPassword sentinel.
func TestFromPassword_nilPassword(t *testing.T) {
	salt := make([]byte, kdf.SaltBytes)
	_, err := lockerv1.FromPassword(nil, salt, kdf.MemoryConstrainedParams)
	assert.ErrorIs(t, err, perrors.ErrNilPassword)
}

// TestFromPassword_badSalt propagates the ErrInvalidSalt sentinel.
func TestFromPassword_badSalt(t *testing.T) {
	_, err := lockerv1.FromPassword([]byte("pw"), make([]byte, 3), kdf.MemoryConstrainedParams)
	assert.ErrorIs(t, err, perrors.ErrInvalidSalt)
}

//=============================================================================
// Fuzz: ParseKeyID
//=============================================================================

// FuzzParseKeyID exercises [locker.Locker.ParseKeyID] on the v1 envelope locker
// against semi-random ciphertext blobs. Invariants:
//
//   - The parser must not panic on any input.
//   - A nil-error result must yield a non-empty key id (the keyring relies on
//     this to route ciphertext; an empty id would collide on every lookup).
func FuzzParseKeyID(f *testing.F) {
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(f, err, "seed key")
	lck, err := lockerv1.New(priv)
	assert.Ok(f, err, "seed locker")
	wire, err := lck.Seal("ns", []byte("plain"))
	assert.Ok(f, err, "seed seal")

	f.Add(wire)
	f.Add([]byte{})
	f.Add([]byte("PURS"))
	f.Add(append([]byte(nil), wire[:len(wire)-1]...))

	f.Fuzz(func(t *testing.T, data []byte) {
		kid, err := lck.ParseKeyID(data)
		if err != nil {
			return
		}
		assert.True(t, len(kid) > 0, "ParseKeyID returned nil error with empty key id")
	})
}

//=============================================================================
// Helpers
//=============================================================================

// freshLocker returns a locker.Locker and the associated private key.
func freshLocker(tb testing.TB) (*ecdh.PrivateKey, interface {
	KeyID() []byte
	Seal(string, []byte) ([]byte, error)
	Open(string, []byte) ([]byte, error)
	ParseKeyID([]byte) ([]byte, error)
}) {
	tb.Helper()
	priv, err := ecdh.X25519().GenerateKey(crand.Reader)
	assert.Ok(tb, err)
	lck, err := lockerv1.New(priv)
	assert.Ok(tb, err)
	return priv, lck
}

// withFailingEntropy swaps [crypto/rand.Reader] for one that always returns [io.EOF].
func withFailingEntropy(t *testing.T) {
	t.Helper()
	orig := crand.Reader
	t.Cleanup(func() { crand.Reader = orig })
	crand.Reader = eofReader{}
}

// eofReader is a [crypto/rand.Reader] substitute that always returns [io.EOF].
type eofReader struct{}

// Read implements [io.Reader].
func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }
