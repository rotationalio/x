package models_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault/v1/constants"
	"go.rtnl.ai/x/vault/v1/gcm"
	"go.rtnl.ai/x/vault/v1/models"
	"go.rtnl.ai/x/vault/v1/suite"
)

// TestSealed_roundtrip builds a full [models.Sealed] row with real inner and wrap crypto, marshals wire bytes,
// unmarshals, and opens the inner payload with the same DEK.
func TestSealed_roundtrip(t *testing.T) {
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

	dek := make([]byte, constants.DEKBytes)
	for i := range dek {
		dek[i] = byte(i + 11)
	}
	innerAEAD, err := gcm.NewInnerAEAD(dek)
	assert.Ok(t, err)
	metaRaw, err := meta.MarshalBinary()
	assert.Ok(t, err)

	nonce, payload, err := gcm.SealInner(innerAEAD, metaRaw, []byte("hello-plain"))
	assert.Ok(t, err)
	body := models.Inner{Nonce: nonce, Payload: payload}

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

	dekWire, err := gcm.SealWrappedDEK(pub, wrapAEAD, gcm.WrapAAD(metaRaw), dek)
	assert.Ok(t, err)
	dekEnv := models.DekEnvelope{Pub: dekWire.Pub, Nonce: dekWire.Nonce, Payload: dekWire.Payload}

	s := models.Sealed{
		FormatVersion: constants.PackageVersion,
		Meta:          meta,
		Dek:           dekEnv,
		Body:          body,
	}
	wire, err := s.MarshalBinary()
	assert.Ok(t, err)

	var opened models.Sealed
	assert.Ok(t, opened.UnmarshalBinary(wire))
	plain, err := gcm.OpenInner(innerAEAD, metaRaw, opened.Body.Nonce, opened.Body.Payload)
	assert.Ok(t, err)
	assert.Equal(t, []byte("hello-plain"), plain)
}
