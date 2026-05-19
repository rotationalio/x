package gcm_test

import (
	"bytes"
	cryptorand "crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/errors"
	"go.rtnl.ai/x/vault/v1/gcm"
)

// TestWrap_sealOpen_roundtrip seals a DEK with random wrap nonce and opens with the same AEAD and AAD.
func TestWrap_sealOpen_roundtrip(t *testing.T) {
	wrapKey := bytes.Repeat([]byte{3}, 32)
	dek := bytes.Repeat([]byte{7}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	copy(pub[:], bytes.Repeat([]byte{9}, 32))
	wrapAAD := []byte("meta-aad-for-wrap")

	wrapped, err := gcm.SealWrappedDEK(pub, aead, wrapAAD, dek)
	assert.Ok(t, err)

	got, err := gcm.OpenWrappedDEK(aead, wrapAAD, wrapped)
	assert.Ok(t, err)
	assert.Equal(t, dek, got)
}

// TestWrap_sealOpen_withFixedNonce exercises [gcm.SealWrappedDEKWithNonce] (deterministic nonce path used by golden vectors).
func TestWrap_sealOpen_withFixedNonce(t *testing.T) {
	wrapKey := bytes.Repeat([]byte{0x22}, 32)
	dek := bytes.Repeat([]byte{0x33}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	var nonce [constants.WrapNonceBytes]byte
	nonce[0] = 7
	wrapAAD := []byte("fixed-nonce-aad")

	wrapped, err := gcm.SealWrappedDEKWithNonce(pub, aead, wrapAAD, dek, nonce)
	assert.Ok(t, err)

	got, err := gcm.OpenWrappedDEK(aead, wrapAAD, wrapped)
	assert.Ok(t, err)
	assert.Equal(t, dek, got)
}

// TestWrap_wrongAAD_failsAuth checks Open with a different AAD than Seal yields [verrors.ErrDecrypt].
func TestWrap_wrongAAD_failsAuth(t *testing.T) {
	wrapKey := bytes.Repeat([]byte{1}, 32)
	dek := bytes.Repeat([]byte{2}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	wrapped, err := gcm.SealWrappedDEK(pub, aead, []byte("aad-a"), dek)
	assert.Ok(t, err)

	_, err = gcm.OpenWrappedDEK(aead, []byte("aad-b"), wrapped)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestWrap_wrongWrapKey_failsAuth seals with one wrap key and opens with another.
func TestWrap_wrongWrapKey_failsAuth(t *testing.T) {
	keySeal := bytes.Repeat([]byte{4}, 32)
	keyOpen := bytes.Repeat([]byte{5}, 32)
	dek := bytes.Repeat([]byte{6}, 32)
	aeadSeal, err := gcm.NewWrapAEAD(keySeal)
	assert.Ok(t, err)
	aeadOpen, err := gcm.NewWrapAEAD(keyOpen)
	assert.Ok(t, err)
	var pub [32]byte
	wrapAAD := []byte("same-aad")
	wrapped, err := gcm.SealWrappedDEK(pub, aeadSeal, wrapAAD, dek)
	assert.Ok(t, err)

	_, err = gcm.OpenWrappedDEK(aeadOpen, wrapAAD, wrapped)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestWrap_nilAEAD_sealAndOpen rejects a nil AEAD on seal and open paths.
func TestWrap_nilAEAD_sealAndOpen(t *testing.T) {
	var pub [32]byte
	dek := bytes.Repeat([]byte{8}, 32)
	_, err := gcm.SealWrappedDEK(pub, nil, []byte("aad"), dek)
	assert.ErrorIs(t, err, verrors.ErrNilAEAD)

	var w gcm.WrappedDEK
	_, err = gcm.OpenWrappedDEK(nil, []byte("aad"), w)
	assert.ErrorIs(t, err, verrors.ErrNilAEAD)
}

// TestWrap_shortDEK_sealFails ensures DEK length must be exactly 32 bytes.
func TestWrap_shortDEK_sealFails(t *testing.T) {
	wrapKey := bytes.Repeat([]byte{2}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	_, err = gcm.SealWrappedDEK(pub, aead, []byte("aad"), make([]byte, 31))
	assert.ErrorIs(t, err, verrors.ErrMalformedParameters)
}

// TestWrap_tamperedPayload_openFails flips a byte in the GCM tag region so Open fails auth.
func TestWrap_tamperedPayload_openFails(t *testing.T) {
	wrapKey := bytes.Repeat([]byte{11}, 32)
	dek := bytes.Repeat([]byte{12}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	wrapAAD := []byte("aad")
	wrapped, err := gcm.SealWrappedDEK(pub, aead, wrapAAD, dek)
	assert.Ok(t, err)

	wrapped.Payload[len(wrapped.Payload)-1] ^= 0xff
	_, err = gcm.OpenWrappedDEK(aead, wrapAAD, wrapped)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestWrap_seal_randFailure checks [gcm.SealWrappedDEK] maps [crypto/rand.Reader] failure to [verrors.ErrSealFailed].
func TestWrap_seal_randFailure(t *testing.T) {
	wrapKey := bytes.Repeat([]byte{13}, 32)
	dek := bytes.Repeat([]byte{14}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte

	orig := cryptorand.Reader
	t.Cleanup(func() { cryptorand.Reader = orig })
	cryptorand.Reader = eofReader{}

	_, err = gcm.SealWrappedDEK(pub, aead, []byte("aad"), dek)
	assert.ErrorIs(t, err, verrors.ErrSealFailed)
}

// TestNewWrapAEAD_rejectsBadKeyLength ensures only 32-byte wrap keys are accepted.
func TestNewWrapAEAD_rejectsBadKeyLength(t *testing.T) {
	_, err := gcm.NewWrapAEAD(make([]byte, 31))
	assert.ErrorIs(t, err, verrors.ErrMalformedParameters)
	_, err = gcm.NewWrapAEAD(make([]byte, 33))
	assert.ErrorIs(t, err, verrors.ErrMalformedParameters)
}

// TestDeriveWrapKey_deterministic asserts HKDF output is stable for the same X25519 shared secret.
func TestDeriveWrapKey_deterministic(t *testing.T) {
	secret := bytes.Repeat([]byte{0xab}, 32)
	k1, err := gcm.DeriveWrapKey(secret)
	assert.Ok(t, err)
	k2, err := gcm.DeriveWrapKey(secret)
	assert.Ok(t, err)
	assert.Equal(t, k1, k2)
}

// TestWrap_truncatedPayload_openFails exercises Open on a ciphertext shorter than DEK+tag.
func TestWrap_truncatedPayload_openFails(t *testing.T) {
	wrapKey := bytes.Repeat([]byte{0x60}, 32)
	dek := bytes.Repeat([]byte{0x61}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	wrapped, err := gcm.SealWrappedDEK(pub, aead, []byte("aad"), dek)
	assert.Ok(t, err)

	trunc := wrapped.Payload[:len(wrapped.Payload)-1]
	tampered := gcm.WrappedDEK{Pub: wrapped.Pub, Nonce: wrapped.Nonce}
	copy(tampered.Payload[:], trunc)
	_, err = gcm.OpenWrappedDEK(aead, []byte("aad"), tampered)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestWrap_distinctNoncesPerSeal expects two [gcm.SealWrappedDEK] calls with the same inputs to differ (random wrap nonces).
func TestWrap_distinctNoncesPerSeal(t *testing.T) {
	wrapKey := bytes.Repeat([]byte{0x70}, 32)
	dek := bytes.Repeat([]byte{0x71}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	aad := []byte("ns")
	w1, err := gcm.SealWrappedDEK(pub, aead, aad, dek)
	assert.Ok(t, err)
	w2, err := gcm.SealWrappedDEK(pub, aead, aad, dek)
	assert.Ok(t, err)
	if w1.Nonce == w2.Nonce && w1.Payload == w2.Payload {
		t.Fatal("expected distinct wrap seals (nonce randomness)")
	}
}
