package gcm_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"io"
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/errors"
	"go.rtnl.ai/x/vault/v1/gcm"
)

//=============================================================================
// newAEAD
//=============================================================================

// TestGCM_newAEAD checks ExportNewAEAD rejects invalid key material with [verrors.ErrInvalidAEADKey].
func TestGCM_newAEAD(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		wantSuccess bool
	}{
		// Each test case covers a different key size: too short, too long, correct sizes.
		{name: "reject_empty", key: nil, wantSuccess: false},
		{name: "reject_15", key: make([]byte, 15), wantSuccess: false},
		{name: "reject_33", key: make([]byte, 33), wantSuccess: false},
		{name: "aes128", key: bytes.Repeat([]byte{1}, 16), wantSuccess: true},
		{name: "aes192", key: bytes.Repeat([]byte{2}, 24), wantSuccess: true},
		{name: "aes256", key: bytes.Repeat([]byte{3}, 32), wantSuccess: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aead, err := gcm.ExportNewAEAD(tt.key)
			if tt.wantSuccess {
				// Should succeed: check for no error and a non-nil AEAD implementation.
				assert.Ok(t, err)
				assert.NotNil(t, aead)

				// AEAD constructed directly (acts as a reference) must match returned AEAD.
				block, err := aes.NewCipher(tt.key)
				assert.Ok(t, err)
				want, err := cipher.NewGCM(block)
				assert.Ok(t, err)
				assert.Equal(t, want.NonceSize(), aead.NonceSize())
			} else {
				// Keys that are invalid should return correct error and nil AEAD.
				assert.Error(t, err)
				assert.Nil(t, aead)
				assert.ErrorIs(t, err, verrors.ErrInvalidAEADKey)
			}
		})
	}
}

//=============================================================================
// SealInner / OpenInner
//=============================================================================

// TestGCM_inner_roundtrip verifies random inner seal, distinct ciphertexts, and OpenInner.
func TestGCM_inner_roundtrip(t *testing.T) {
	// Use a deterministic 32-byte key to construct an AEAD.
	key := bytes.Repeat([]byte{7}, 32)
	aead, err := gcm.NewInnerAEAD(key)
	assert.Ok(t, err)

	// Define variant test cases for AAD/plaintext combinations, including corner cases.
	tests := []struct {
		name  string
		aad   []byte
		plain []byte
	}{
		{name: "empty_plain", aad: []byte("ns-a"), plain: nil},
		{name: "short", aad: []byte("x"), plain: []byte("hello")},
		{name: "long_aad", aad: []byte(strings.Repeat("n", 500)), plain: []byte("p")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Seal with input, get ciphertext and nonce.
			n1, p1, err := gcm.SealInner(aead, tt.aad, tt.plain)
			assert.Ok(t, err)

			// Decrypt with matching params: should recover the original plaintext.
			got, err := gcm.OpenInner(aead, tt.aad, n1, p1)
			assert.Ok(t, err)
			assert.Equal(t, tt.plain, got)

			// Seal again (same AAD/plain) - should be randomized (nonce).
			n2, p2, err := gcm.SealInner(aead, tt.aad, tt.plain)
			assert.Ok(t, err)

			// Ensure distinct ciphertext outputs for each seal operation.
			if bytes.Equal(n1[:], n2[:]) && bytes.Equal(p1, p2) {
				t.Fatal("expected distinct inner seals (nonce randomness)")
			}
		})
	}
}

// TestGCM_inner_sealOpen_withFixedNonce exercises [gcm.SealInnerWithNonce] (deterministic nonce path).
func TestGCM_inner_sealOpen_withFixedNonce(t *testing.T) {
	// Use a static 32-byte key.
	key := bytes.Repeat([]byte{0x33}, 32)
	aead, err := gcm.NewInnerAEAD(key)
	assert.Ok(t, err)

	// Use a fixed nonce to force deterministic encryption.
	var nonce [constants.InnerNonceBytes]byte
	nonce[0] = 3

	// Seal and decrypt using the fixed nonce.
	n, p, err := gcm.SealInnerWithNonce(aead, []byte("aad"), []byte("hello"), nonce)
	assert.Ok(t, err)
	got, err := gcm.OpenInner(aead, []byte("aad"), n, p)
	assert.Ok(t, err)

	// Expect exact recovery of plaintext.
	assert.Equal(t, []byte("hello"), got)
}

// TestGCM_inner_wrong_AAD ensures [gcm.OpenInner] with a different AAD than [gcm.SealInner] fails with [verrors.ErrDecrypt].
func TestGCM_inner_wrong_AAD(t *testing.T) {
	// Construct an AEAD and seal using one AAD.
	key := bytes.Repeat([]byte{9}, 32)
	aead, err := gcm.NewInnerAEAD(key)
	assert.Ok(t, err)

	// Seal with one AAD.
	n, p, err := gcm.SealInner(aead, []byte("seal-ns"), []byte("secret"))
	assert.Ok(t, err)

	// Attempt to open with a different AAD: should fail with ErrDecrypt.
	_, err = gcm.OpenInner(aead, []byte("other-ns"), n, p)
	assert.Error(t, err)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestGCM_inner_truncated_and_flipped exercises OpenInner on truncated ciphertext and a bit flip in the tag region.
func TestGCM_inner_truncated_and_flipped(t *testing.T) {
	// Create AEAD using a static test key.
	key := bytes.Repeat([]byte{5}, 32)
	aead, err := gcm.NewInnerAEAD(key)
	assert.Ok(t, err)

	// Seal with one AAD.
	n, p, err := gcm.SealInner(aead, []byte("aad"), []byte("payload"))
	assert.Ok(t, err)

	t.Run("truncated_payload", func(t *testing.T) {
		// Use a ciphertext with one byte missing from the end.
		trunc := p[:len(p)-1]
		_, err := gcm.OpenInner(aead, []byte("aad"), n, trunc)

		// Should fail integrity/auth check and return ErrDecrypt.
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrDecrypt)
	})

	t.Run("bit_flip_in_tag", func(t *testing.T) {
		// Flip a bit at the end of the ciphertext (tag region).
		tampered := append([]byte(nil), p...)
		tampered[len(tampered)-1] ^= 0xff
		_, err := gcm.OpenInner(aead, []byte("aad"), n, tampered)

		// Should also fail authenticity.
		assert.Error(t, err)
		assert.ErrorIs(t, err, verrors.ErrDecrypt)
	})
}

// TestGCM_inner_seal_rand_failure checks [gcm.SealInner] wraps [crypto/rand.Reader] failure with [verrors.ErrSealFailed].
func TestGCM_inner_seal_rand_failure(t *testing.T) {
	// Create AEAD using a static test key.
	key := bytes.Repeat([]byte{2}, 32)
	aead, err := gcm.NewInnerAEAD(key)
	assert.Ok(t, err)

	// Save and replace cryptorand.Reader with an EOFing reader to simulate failure.
	orig := cryptorand.Reader
	t.Cleanup(func() { cryptorand.Reader = orig })
	cryptorand.Reader = eofReader{}

	// Call SealInner; expects an error wrapping EOF as ErrSealFailed.
	_, _, err = gcm.SealInner(aead, []byte("aad"), []byte("x"))
	assert.Error(t, err)
	assert.ErrorIs(t, err, verrors.ErrSealFailed)
}

// TestGCM_inner_nilAEAD_sealAndOpen rejects nil AEAD on [gcm.SealInner] and [gcm.OpenInner].
func TestGCM_inner_nilAEAD_sealAndOpen(t *testing.T) {
	// Attempting to SealInner with a nil AEAD should return ErrNilAEAD.
	_, _, err := gcm.SealInner(nil, []byte("aad"), []byte("x"))
	assert.ErrorIs(t, err, verrors.ErrNilAEAD)

	// Attempting to OpenInner with a nil AEAD should also return ErrNilAEAD.
	var zeroNonce [constants.InnerNonceBytes]byte
	_, err = gcm.OpenInner(nil, []byte("aad"), zeroNonce, []byte{1, 2, 3})
	assert.ErrorIs(t, err, verrors.ErrNilAEAD)
}

// TestGCM_NewInnerAEAD_rejectsBadDEKLength ensures only a 32-byte DEK is accepted for inner AEAD.
func TestGCM_NewInnerAEAD_rejectsBadDEKLength(t *testing.T) {
	_, err := gcm.NewInnerAEAD(make([]byte, constants.DEKBytes-1))
	assert.ErrorIs(t, err, verrors.ErrMalformedParameters)
}

// TestGCM_inner_wrongDEK_failsAuth opens with a different inner key than was used to seal.
func TestGCM_inner_wrongDEK_failsAuth(t *testing.T) {
	// Seal with one key.
	keySeal := bytes.Repeat([]byte{0x40}, 32)

	// Open with a different key.
	keyOpen := bytes.Repeat([]byte{0x41}, 32)

	aeadSeal, err := gcm.NewInnerAEAD(keySeal)
	assert.Ok(t, err)
	aeadOpen, err := gcm.NewInnerAEAD(keyOpen)
	assert.Ok(t, err)

	// Produce ciphertext.
	n, p, err := gcm.SealInner(aeadSeal, []byte("aad"), []byte("payload"))
	assert.Ok(t, err)

	// Attempt to open ciphertext with a different key: should fail authentication.
	_, err = gcm.OpenInner(aeadOpen, []byte("aad"), n, p)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

//=============================================================================
// Test helpers
//=============================================================================

// Implements io.Reader, always returns EOF (used to simulate random failure in tests).
type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }
