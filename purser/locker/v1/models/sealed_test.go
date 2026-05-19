package models_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/binary"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/gcm"
	"go.rtnl.ai/x/purser/locker/v1/models"
	"go.rtnl.ai/x/purser/locker/v1/suite"
)

// sealedPreambleBytes mirrors models.sealedPreambleBytes for offset math in tests.
const sealedPreambleBytes = 4 + 1 + 2

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

//=============================================================================
// Tests: Sealed.UnmarshalBinary negative cases
//=============================================================================

// TestSealed_unmarshalMalformedWire exercises the negative branches of
// [models.Sealed.UnmarshalBinary] over a deliberately broken wire matrix. Each
// case is a precise mutation of a known-good wire blob so a failure points at
// exactly one parser invariant.
func TestSealed_unmarshalMalformedWire(t *testing.T) {
	good := newValidSealedWire(t)

	t.Run("nil_receiver", func(t *testing.T) {
		var s *models.Sealed
		err := s.UnmarshalBinary(good)
		assert.ErrorIs(t, err, verrors.ErrNilSealedPointer)
	})

	t.Run("short_preamble", func(t *testing.T) {
		var got models.Sealed
		err := got.UnmarshalBinary(good[:sealedPreambleBytes-1])
		assert.ErrorIs(t, err, verrors.ErrMalformedWire)
	})

	t.Run("bad_magic", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[0], bad[1], bad[2], bad[3] = 'X', 'X', 'X', 'X'
		var got models.Sealed
		err := got.UnmarshalBinary(bad)
		assert.ErrorIs(t, err, verrors.ErrBadMagic)
	})

	t.Run("meta_length_too_small", func(t *testing.T) {
		// Minimum valid meta length is 4 (version + suite + keyIDLen + nsLen).
		// Anything smaller must be rejected before the parser touches Meta bytes.
		bad := append([]byte(nil), good...)
		binary.BigEndian.PutUint16(bad[5:7], 3)
		var got models.Sealed
		err := got.UnmarshalBinary(bad)
		assert.ErrorIs(t, err, verrors.ErrMalformedWire)
	})

	t.Run("meta_length_too_large", func(t *testing.T) {
		// Declared meta length exceeds the wire cap — the parser must refuse it
		// even though the slice itself might be physically large enough.
		bad := append([]byte(nil), good...)
		binary.BigEndian.PutUint16(bad[5:7], uint16(constants.MaxMetaWireBytes+1))
		var got models.Sealed
		err := got.UnmarshalBinary(bad)
		assert.ErrorIs(t, err, verrors.ErrMalformedWire)
	})

	t.Run("truncated_after_meta", func(t *testing.T) {
		// Drop the last byte of the DEK envelope to make the post-meta region
		// shorter than the required DEK+inner-nonce+tag minimum.
		var got models.Sealed
		err := got.UnmarshalBinary(good[:len(good)-1-constants.GCMTagBytes-constants.InnerNonceBytes])
		assert.ErrorIs(t, err, verrors.ErrMalformedWire)
	})

	t.Run("truncated_inner_body", func(t *testing.T) {
		// Drop one byte off the very end so the inner body falls below the
		// nonce + GCM tag minimum and the inner unmarshal layer rejects it.
		var got models.Sealed
		err := got.UnmarshalBinary(good[:len(good)-(constants.GCMTagBytes+1)])
		assert.ErrorIs(t, err, verrors.ErrMalformedWire)
	})
}

// TestSealed_unmarshalVersionMismatch confirms the outer format version and the
// embedded [models.Meta.PackageVersion] are cross-checked, and that an unsupported
// version flips them both to is surfaced as [verrors.ErrUnsupportedVersion].
func TestSealed_unmarshalVersionMismatch(t *testing.T) {
	good := newValidSealedWire(t)

	t.Run("outer_disagrees_with_meta", func(t *testing.T) {
		// Flip ONLY the outer FormatVersion. The embedded meta still reports v1
		// so meta unmarshal succeeds; the Sealed-layer cross-check rejects the row.
		bad := append([]byte(nil), good...)
		bad[4] = constants.PackageVersion + 1

		var got models.Sealed
		err := got.UnmarshalBinary(bad)
		assert.ErrorIs(t, err, verrors.ErrVersionMismatch)
	})

	t.Run("unsupported_version", func(t *testing.T) {
		// Flip BOTH outer and embedded version bytes. Meta unmarshal rejects the
		// embedded byte first and surfaces ErrUnsupportedVersion — callers see the
		// same sentinel as the Sealed-layer defensive check.
		bad := append([]byte(nil), good...)
		bad[4] = 99                   // outer FormatVersion
		bad[sealedPreambleBytes] = 99 // meta PackageVersion is the first byte after preamble

		var got models.Sealed
		err := got.UnmarshalBinary(bad)
		assert.ErrorIs(t, err, verrors.ErrUnsupportedVersion)
	})
}

//=============================================================================
// Helpers
//=============================================================================

// newValidSealedWire constructs a fully-formed v1 Sealed wire blob suitable for
// negative-test mutation. Tests own the resulting slice and may modify it freely.
func newValidSealedWire(tb testing.TB) []byte {
	tb.Helper()

	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(tb, err)

	kid := append([]byte(nil), priv.PublicKey().Bytes()...)
	if len(kid) > constants.MaxKeyIDBytes {
		kid = kid[:constants.MaxKeyIDBytes]
	}
	meta := models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          kid,
		Namespace:      "ns",
	}

	dek := make([]byte, constants.DEKBytes)
	for i := range dek {
		dek[i] = byte(i + 3)
	}
	innerAEAD, err := gcm.NewInnerAEAD(dek)
	assert.Ok(tb, err)

	metaRaw, err := meta.MarshalBinary()
	assert.Ok(tb, err)
	nonce, payload, err := gcm.SealInner(innerAEAD, metaRaw, []byte("plain"))
	assert.Ok(tb, err)

	eph, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(tb, err)
	shared, err := eph.ECDH(priv.PublicKey())
	assert.Ok(tb, err)
	wk, err := gcm.DeriveWrapKey(shared)
	assert.Ok(tb, err)
	wrapAEAD, err := gcm.NewWrapAEAD(wk)
	assert.Ok(tb, err)
	var pub [constants.X25519PubBytes]byte
	copy(pub[:], eph.PublicKey().Bytes())

	wrapAAD := append([]byte(gcm.WrapAADPrefix), metaRaw...)
	dekWire, err := gcm.SealWrappedDEK(pub, wrapAEAD, wrapAAD, dek)
	assert.Ok(tb, err)

	s := models.Sealed{
		FormatVersion: constants.PackageVersion,
		Meta:          meta,
		Dek:           models.DekEnvelope{Pub: dekWire.Pub, Nonce: dekWire.Nonce, Payload: dekWire.Payload},
		Body:          models.Inner{Nonce: nonce, Payload: payload},
	}
	wire, err := s.MarshalBinary()
	assert.Ok(tb, err)
	return wire
}

//=============================================================================
// Fuzz: Sealed.UnmarshalBinary
//=============================================================================

// FuzzSealed_unmarshal exercises [models.Sealed.UnmarshalBinary] across many
// mutated wire blobs. Invariants:
//
//   - The parser must not panic on any input.
//   - A successful unmarshal must re-marshal to identical bytes — the v1 wire
//     framing is canonical, so any deviation indicates a parser/encoder skew.
func FuzzSealed_unmarshal(f *testing.F) {
	good := newValidSealedWire(f)
	f.Add(good)
	f.Add([]byte{})
	f.Add(good[:sealedPreambleBytes])
	f.Add(append([]byte(nil), good[:len(good)-1]...))

	f.Fuzz(func(t *testing.T, data []byte) {
		var s models.Sealed
		if err := s.UnmarshalBinary(data); err != nil {
			return
		}
		out, err := s.MarshalBinary()
		assert.Ok(t, err, "unmarshal succeeded but re-marshal failed")
		assert.Equal(t, string(data), string(out), "round-trip mismatch")
	})
}
