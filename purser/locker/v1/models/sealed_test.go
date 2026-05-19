package models_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/gcm"
	"go.rtnl.ai/x/purser/locker/v1/models"
	"go.rtnl.ai/x/purser/locker/v1/suite"
)

// TestSealed_roundtrip builds a full [models.Sealed] row with real inner and wrap crypto, marshals wire bytes,
// unmarshals, and opens the inner payload with the same DEK.
func TestSealed_roundtrip(t *testing.T) {
	// Long-term X25519 key (acts as the locker's wrap key) and the per-row metadata.
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)

	kid := append([]byte(nil), priv.PublicKey().Bytes()...)
	if len(kid) > constants.MaxKeyIDBytes {
		kid = kid[:constants.MaxKeyIDBytes]
	}
	meta := models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          kid,
		Namespace:      "app",
	}

	// Deterministic DEK so the failure mode (if any) is reproducible.
	dek := make([]byte, constants.DEKBytes)
	for i := range dek {
		dek[i] = byte(i + 11)
	}
	innerAEAD, err := gcm.NewInnerAEAD(dek)
	assert.Ok(t, err)
	metaRaw, err := meta.MarshalBinary()
	assert.Ok(t, err)

	// Inner-payload seal: encrypt the user plaintext under the DEK with metaRaw as AAD.
	nonce, payload, err := gcm.SealInner(innerAEAD, metaRaw, []byte("hello-plain"))
	assert.Ok(t, err)
	body := models.Inner{Nonce: nonce, Payload: payload}

	// Envelope: derive a wrap key from ephemeral X25519 ↔ long-term X25519 ECDH, then
	// wrap the DEK so the recipient can recover it without the original DEK ever
	// hitting the wire.
	eph, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	shared, err := eph.ECDH(priv.PublicKey())
	assert.Ok(t, err)
	wk, err := gcm.DeriveWrapKey(shared)
	assert.Ok(t, err)
	wrapAEAD, err := gcm.NewWrapAEAD(wk)
	assert.Ok(t, err)
	var pub [constants.X25519PubBytes]byte
	copy(pub[:], eph.PublicKey().Bytes())

	wrapAAD := append([]byte(gcm.WrapAADPrefix), metaRaw...)
	dekWire, err := gcm.SealWrappedDEK(pub, wrapAEAD, wrapAAD, dek)
	assert.Ok(t, err)
	dekEnv := models.DekEnvelope{Pub: dekWire.Pub, Nonce: dekWire.Nonce, Payload: dekWire.Payload}

	// Assemble the Sealed row and serialize it to wire bytes.
	s := models.Sealed{
		FormatVersion: constants.PackageVersion,
		Meta:          meta,
		Dek:           dekEnv,
		Body:          body,
	}
	wire, err := s.MarshalBinary()
	assert.Ok(t, err)

	// Unmarshal the wire bytes back into a fresh Sealed and decrypt the inner body
	// using the (still in-memory) DEK and the same AAD.
	var opened models.Sealed
	assert.Ok(t, opened.UnmarshalBinary(wire))
	plain, err := gcm.OpenInner(innerAEAD, metaRaw, opened.Body.Nonce, opened.Body.Payload)

	// Recovered plaintext matches the original — confirming the full Sealed wire
	// (meta + dek envelope + body) round-trips and the inner payload survives intact.
	assert.Ok(t, err)
	assert.Equal(t, []byte("hello-plain"), plain)
}
