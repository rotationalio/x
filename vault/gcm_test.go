package vault_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"io"
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault"
)

//=============================================================================
// newAEAD
//=============================================================================

// TestGCM_newAEAD checks ExportGCMNewAEAD rejects invalid key lengths with
// ErrInvalidAESKeySize and builds GCM for 16/24/32-byte AES keys with nonce sizes
// matching cipher.NewGCM.
func TestGCM_newAEAD(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		wantSuccess bool
	}{
		{name: "reject_empty", key: nil, wantSuccess: false},
		{name: "reject_15", key: make([]byte, 15), wantSuccess: false},
		{name: "reject_33", key: make([]byte, 33), wantSuccess: false},
		{name: "aes128", key: bytes.Repeat([]byte{1}, 16), wantSuccess: true},
		{name: "aes192", key: bytes.Repeat([]byte{2}, 24), wantSuccess: true},
		{name: "aes256", key: bytes.Repeat([]byte{3}, 32), wantSuccess: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aead, err := vault.ExportGCMNewAEAD(tt.key)
			if tt.wantSuccess {
				assert.Ok(t, err)
				assert.NotNil(t, aead)
				// Sanity: matches stdlib GCM nonce size for this key length.
				block, err := aes.NewCipher(tt.key)
				assert.Ok(t, err)
				want, err := cipher.NewGCM(block)
				assert.Ok(t, err)
				assert.Equal(t, want.NonceSize(), aead.NonceSize())
			} else {
				assert.Error(t, err)
				assert.Nil(t, aead)
				assert.ErrorIs(t, err, vault.ErrInvalidAESKeySize)
			}
		})
	}
}

//=============================================================================
// seal / open
//=============================================================================

// TestGCM_seal_open_roundtrip verifies ExportGCMSeal output layout nonce|ciphertext,
// ExportGCMOpen roundtrip for several AAD/plaintext shapes, and two seals yield distinct blobs.
func TestGCM_seal_open_roundtrip(t *testing.T) {
	// Fixed AES-256 key and shared AEAD across cases (plaintext/AAD shapes vary only).
	key := bytes.Repeat([]byte{7}, 32)
	aead, err := vault.ExportGCMNewAEAD(key)
	assert.Ok(t, err)

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
			// Seal the plaintext
			blob, err := vault.ExportGCMSeal(aead, tt.aad, tt.plain)
			assert.Ok(t, err)
			ns := aead.NonceSize()
			assert.True(t, len(blob) >= ns)

			// Open the blob
			got, err := vault.ExportGCMOpen(aead, tt.aad, blob)
			assert.Ok(t, err)
			assert.Equal(t, tt.plain, got)

			// Nonces are random; two seals must not produce identical wire bytes.
			b2, err := vault.ExportGCMSeal(aead, tt.aad, tt.plain)
			assert.Ok(t, err)
			if bytes.Equal(blob, b2) {
				t.Fatal("expected distinct ciphertexts (nonce randomness)")
			}
		})
	}
}

// TestGCM_open_wrong_AAD ensures ExportGCMOpen with a different AAD than ExportGCMSeal used
// fails with ErrDecrypt.
func TestGCM_open_wrong_AAD(t *testing.T) {
	// Generate a 32-byte AES-256 key filled with the value 9.
	key := bytes.Repeat([]byte{9}, 32)

	// Construct a GCM AEAD instance using the generated key.
	aead, err := vault.ExportGCMNewAEAD(key)
	assert.Ok(t, err)

	// Seal the plaintext "secret" with the AAD "seal-ns".
	blob, err := vault.ExportGCMSeal(aead, []byte("seal-ns"), []byte("secret"))
	assert.Ok(t, err)

	// Try to open the sealed blob with a different AAD ("other-ns"), which should fail
	// because authentication will not match the AAD used for sealing.
	_, err = vault.ExportGCMOpen(aead, []byte("other-ns"), blob)
	assert.Error(t, err)
	assert.ErrorIs(t, err, vault.ErrDecrypt)
}

// TestGCM_open_truncated_and_flipped exercises ExportGCMOpen on blobs shorter than the nonce,
// truncated ciphertext after the nonce, and a bit flip in the tag region.
func TestGCM_open_truncated_and_flipped(t *testing.T) {
	// Fixed key
	key := bytes.Repeat([]byte{5}, 32)
	aead, err := vault.ExportGCMNewAEAD(key)
	assert.Ok(t, err)

	// Fixed payload
	blob, err := vault.ExportGCMSeal(aead, []byte("aad"), []byte("payload"))
	assert.Ok(t, err)

	t.Run("shorter_than_nonce", func(t *testing.T) {
		// Blob is shorter than length of nonce; should fail with both ErrDecrypt and ErrCiphertextTooShort.
		short := blob[:aead.NonceSize()-1]
		_, err := vault.ExportGCMOpen(aead, []byte("aad"), short)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDecrypt)
		assert.ErrorIs(t, err, vault.ErrCiphertextTooShort)
	})

	t.Run("truncated_after_nonce", func(t *testing.T) {
		ns := aead.NonceSize()
		// Blob is nonce plus just one byte (incomplete tag/ciphertext); should return ErrDecrypt.
		trunc := blob[:ns+1]
		_, err := vault.ExportGCMOpen(aead, []byte("aad"), trunc)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDecrypt)
	})

	t.Run("bit_flip_in_tag", func(t *testing.T) {
		// Tamper with the last (tag) byte of the blob and expect ErrDecrypt on open.
		tampered := append([]byte(nil), blob...)
		tampered[len(tampered)-1] ^= 0xff
		_, err := vault.ExportGCMOpen(aead, []byte("aad"), tampered)
		assert.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrDecrypt)
	})
}

// TestGCM_seal_rand_failure checks ExportGCMSeal wraps Rand.Reader failure with ErrSealFailed.
func TestGCM_seal_rand_failure(t *testing.T) {
	// Fixed key
	key := bytes.Repeat([]byte{2}, 32)
	aead, err := vault.ExportGCMNewAEAD(key)
	assert.Ok(t, err)

	// Save original cryptorand.Reader and restore after test.
	orig := cryptorand.Reader
	t.Cleanup(func() { cryptorand.Reader = orig })
	cryptorand.Reader = eofReader{}

	// Attempt sealing; should return ErrSealFailed due to nonce read failure.
	_, err = vault.ExportGCMSeal(aead, []byte("aad"), []byte("x"))
	assert.Error(t, err)
	assert.ErrorIs(t, err, vault.ErrSealFailed)
}

//=============================================================================
// Test helpers
//=============================================================================

// eofReader always returns io.EOF so io.ReadFull for the nonce fails immediately.
type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }
