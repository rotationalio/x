package gcm_test

import (
	"bytes"
	cryptorand "crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/gcm"
)

// TestWrap_sealOpen_roundtrip seals a DEK with random wrap nonce and opens with the same AEAD and AAD.
func TestWrap_sealOpen_roundtrip(t *testing.T) {
	// Deterministic wrap key, DEK, ephemeral pubkey, and AAD so the test is reproducible
	// (the wrap nonce itself is still random).
	wrapKey := bytes.Repeat([]byte{3}, 32)
	dek := bytes.Repeat([]byte{7}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	copy(pub[:], bytes.Repeat([]byte{9}, 32))
	wrapAAD := []byte("meta-aad-for-wrap")

	// Wrap the DEK, then immediately unwrap with matching AEAD and AAD.
	wrapped, err := gcm.SealWrappedDEK(pub, aead, wrapAAD, dek)
	assert.Ok(t, err)
	got, err := gcm.OpenWrappedDEK(aead, wrapAAD, wrapped)

	// Round-trip recovers the exact DEK bytes — confirming the wrap layer is lossless
	// when AAD and key match.
	assert.Ok(t, err)
	assert.Equal(t, dek, got)
}

// TestWrap_sealOpen_withFixedNonce exercises [gcm.SealWrappedDEKWithNonce] (deterministic nonce path used by golden vectors).
func TestWrap_sealOpen_withFixedNonce(t *testing.T) {
	// Deterministic inputs including a pinned nonce — golden vectors rely on this path
	// to produce reproducible wire bytes.
	wrapKey := bytes.Repeat([]byte{0x22}, 32)
	dek := bytes.Repeat([]byte{0x33}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	var nonce [constants.WrapNonceBytes]byte
	nonce[0] = 7
	wrapAAD := []byte("fixed-nonce-aad")

	// Wrap with the explicit nonce, then unwrap with matching AAD.
	wrapped, err := gcm.SealWrappedDEKWithNonce(pub, aead, wrapAAD, dek, nonce)
	assert.Ok(t, err)
	got, err := gcm.OpenWrappedDEK(aead, wrapAAD, wrapped)

	// Round-trip is lossless even when the nonce is caller-supplied.
	assert.Ok(t, err)
	assert.Equal(t, dek, got)
}

// TestWrap_wrongAAD_failsAuth checks Open with a different AAD than Seal yields [verrors.ErrDecrypt].
func TestWrap_wrongAAD_failsAuth(t *testing.T) {
	// Wrap a DEK with AAD "aad-a".
	wrapKey := bytes.Repeat([]byte{1}, 32)
	dek := bytes.Repeat([]byte{2}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	wrapped, err := gcm.SealWrappedDEK(pub, aead, []byte("aad-a"), dek)
	assert.Ok(t, err)

	// Attempt to unwrap with a different AAD ("aad-b"); GCM authentication must fail.
	_, err = gcm.OpenWrappedDEK(aead, []byte("aad-b"), wrapped)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestWrap_wrongWrapKey_failsAuth seals with one wrap key and opens with another.
func TestWrap_wrongWrapKey_failsAuth(t *testing.T) {
	// Two distinct wrap keys backing two AEADs; the same DEK is wrapped under one and
	// then we attempt to unwrap with the other.
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

	// Open with the wrong wrap key — GCM tag verification must fail.
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
	// Wrap a DEK normally, then mutate the last byte of the payload — that byte falls
	// inside the trailing GCM authentication tag.
	wrapKey := bytes.Repeat([]byte{11}, 32)
	dek := bytes.Repeat([]byte{12}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	wrapAAD := []byte("aad")
	wrapped, err := gcm.SealWrappedDEK(pub, aead, wrapAAD, dek)
	assert.Ok(t, err)
	wrapped.Payload[len(wrapped.Payload)-1] ^= 0xff

	// Open must reject the tampered blob with ErrDecrypt — proving the wrap layer
	// authenticates the full payload.
	_, err = gcm.OpenWrappedDEK(aead, wrapAAD, wrapped)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestWrap_seal_randFailure checks [gcm.SealWrappedDEK] maps [crypto/rand.Reader] failure to [verrors.ErrSealFailed].
func TestWrap_seal_randFailure(t *testing.T) {
	// Construct the wrap AEAD normally so the only failure surface is entropy.
	wrapKey := bytes.Repeat([]byte{13}, 32)
	dek := bytes.Repeat([]byte{14}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte

	// Replace the package rand.Reader with one that always returns EOF; cleanup restores
	// it after the test.
	orig := cryptorand.Reader
	t.Cleanup(func() { cryptorand.Reader = orig })
	cryptorand.Reader = eofReader{}

	// The seal path needs entropy for the wrap nonce and must report ErrSealFailed
	// rather than leaking the raw io error.
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
	// One fixed shared secret is all we need — HKDF must be a pure function.
	secret := bytes.Repeat([]byte{0xab}, 32)

	// Derive the wrap key twice from identical inputs.
	k1, err := gcm.DeriveWrapKey(secret)
	assert.Ok(t, err)
	k2, err := gcm.DeriveWrapKey(secret)
	assert.Ok(t, err)

	// Both derivations must produce identical bytes; otherwise Seal and Open paths in
	// the envelope flow could disagree on the wrap key.
	assert.Equal(t, k1, k2)
}

// TestWrap_truncatedPayload_openFails exercises Open on a ciphertext shorter than DEK+tag.
func TestWrap_truncatedPayload_openFails(t *testing.T) {
	// Wrap a DEK normally to obtain a well-formed wrapped DEK.
	wrapKey := bytes.Repeat([]byte{0x60}, 32)
	dek := bytes.Repeat([]byte{0x61}, 32)
	aead, err := gcm.NewWrapAEAD(wrapKey)
	assert.Ok(t, err)
	var pub [32]byte
	wrapped, err := gcm.SealWrappedDEK(pub, aead, []byte("aad"), dek)
	assert.Ok(t, err)

	// Build a tampered version with the last byte of the payload dropped (truncated tag).
	trunc := wrapped.Payload[:len(wrapped.Payload)-1]
	tampered := gcm.WrappedDEK{Pub: wrapped.Pub, Nonce: wrapped.Nonce}
	copy(tampered.Payload[:], trunc)

	// Open must fail with ErrDecrypt — truncated ciphertext cannot satisfy GCM auth.
	_, err = gcm.OpenWrappedDEK(aead, []byte("aad"), tampered)
	assert.ErrorIs(t, err, verrors.ErrDecrypt)
}

// TestWrap_distinctNoncesPerSeal expects two [gcm.SealWrappedDEK] calls with the same inputs to differ (random wrap nonces).
func TestWrap_distinctNoncesPerSeal(t *testing.T) {
	// Identical wrap key, DEK, ephemeral pub, and AAD across both seals — only the nonce
	// (drawn from rand.Reader) is allowed to differ.
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

	// Different nonces (or therefore different payloads) prove the seal path samples a
	// fresh wrap nonce per call — required to avoid GCM nonce reuse.
	assert.True(t, w1.Nonce != w2.Nonce || w1.Payload != w2.Payload, "expected distinct wrap seals (nonce randomness)")
}
