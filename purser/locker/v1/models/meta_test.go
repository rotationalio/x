package models_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/models"
	"go.rtnl.ai/x/purser/locker/v1/suite"
)

// TestMeta_roundtrip checks [models.Meta.MarshalBinary] and [models.Meta.UnmarshalBinary] preserve fields.
func TestMeta_roundtrip(t *testing.T) {
	// A fully populated Meta covering all four fields: version, suite, key id, and namespace.
	m := models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          []byte{1, 2, 3},
		Namespace:      "ns-a",
	}

	// Marshal then unmarshal into a fresh value.
	b, err := m.MarshalBinary()
	assert.Ok(t, err)
	var got models.Meta

	// All four fields survive byte-for-byte. Field-by-field assertions (rather than
	// assert.Equal on the struct) make any future drift in a single field obvious.
	assert.Ok(t, got.UnmarshalBinary(b))
	assert.Equal(t, m.PackageVersion, got.PackageVersion)
	assert.Equal(t, m.SuiteID, got.SuiteID)
	assert.Equal(t, m.KeyID, got.KeyID)
	assert.Equal(t, m.Namespace, got.Namespace)
}

// TestMetaFromPrivKey verifies [models.MetaFromPrivKey] fills version, suite, and key id from an X25519 key.
func TestMetaFromPrivKey(t *testing.T) {
	// A fresh X25519 key — the only input the helper consumes.
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)

	// Build the template Meta from the private key.
	m, err := models.MetaFromPrivKey(priv)
	assert.Ok(t, err)

	// Version and suite are pinned to v1; key id matches the X25519 public key bytes;
	// namespace is left empty for callers to fill via WithNamespace at seal time.
	assert.Equal(t, constants.PackageVersion, m.PackageVersion)
	assert.Equal(t, suite.X25519HKDFSHA256AES256GCM, m.SuiteID)
	assert.Equal(t, priv.PublicKey().Bytes(), m.KeyID)
	assert.Equal(t, "", m.Namespace)
}

// TestMetaFromPrivKey_nil asserts a nil private key returns [verrors.ErrNilPrivateKey].
func TestMetaFromPrivKey_nil(t *testing.T) {
	_, err := models.MetaFromPrivKey(nil)
	assert.ErrorIs(t, err, verrors.ErrNilPrivateKey)
}

// TestMetaFromPrivKey_nonX25519 asserts a non-X25519 key returns [verrors.ErrInvalidWrappingKey].
func TestMetaFromPrivKey_nonX25519(t *testing.T) {
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	_, err = models.MetaFromPrivKey(priv)
	assert.ErrorIs(t, err, verrors.ErrInvalidWrappingKey)
}

//=============================================================================
// Tests: Meta.WithNamespace
//=============================================================================

// TestMeta_withNamespace covers the happy-path namespace replacement and the
// oversized-namespace cap rejection.
func TestMeta_withNamespace(t *testing.T) {
	base := models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          []byte{1, 2, 3},
		Namespace:      "",
	}

	t.Run("sets_namespace", func(t *testing.T) {
		got, err := base.WithNamespace("app")
		assert.Ok(t, err)
		assert.Equal(t, "app", got.Namespace)
		assert.Equal(t, "", base.Namespace, "WithNamespace must not mutate the receiver")
	})

	t.Run("oversized_namespace_rejected", func(t *testing.T) {
		big := string(make([]byte, constants.MaxNamespaceBytes+1))
		_, err := base.WithNamespace(big)
		assert.ErrorIs(t, err, verrors.ErrMetaNamespaceTooLarge)
	})
}

//=============================================================================
// Tests: Meta marshal caps and buffer sizing
//=============================================================================

// TestMeta_marshalCaps asserts the size and write paths reject inputs that exceed
// declared on-wire caps for KeyID and Namespace.
func TestMeta_marshalCaps(t *testing.T) {
	t.Run("oversized_keyid", func(t *testing.T) {
		m := models.Meta{
			PackageVersion: constants.PackageVersion,
			SuiteID:        suite.X25519HKDFSHA256AES256GCM,
			KeyID:          make([]byte, constants.MaxKeyIDBytes+1),
			Namespace:      "",
		}
		_, err := m.MarshalBinarySize()
		assert.ErrorIs(t, err, verrors.ErrMetaKeyIDTooLarge)
		_, err = m.MarshalBinary()
		assert.ErrorIs(t, err, verrors.ErrMetaKeyIDTooLarge)
	})

	t.Run("oversized_namespace", func(t *testing.T) {
		m := models.Meta{
			PackageVersion: constants.PackageVersion,
			SuiteID:        suite.X25519HKDFSHA256AES256GCM,
			KeyID:          []byte{1, 2, 3},
			Namespace:      string(make([]byte, constants.MaxNamespaceBytes+1)),
		}
		_, err := m.MarshalBinary()
		assert.ErrorIs(t, err, verrors.ErrMetaNamespaceTooLarge)
	})

	t.Run("unsupported_version", func(t *testing.T) {
		m := models.Meta{
			PackageVersion: constants.PackageVersion + 1,
			SuiteID:        suite.X25519HKDFSHA256AES256GCM,
			KeyID:          []byte{1, 2, 3},
			Namespace:      "ns",
		}
		_, err := m.MarshalBinarySize()
		assert.ErrorIs(t, err, verrors.ErrUnsupportedVersion)
	})

	t.Run("unknown_suite", func(t *testing.T) {
		m := models.Meta{
			PackageVersion: constants.PackageVersion,
			SuiteID:        suite.Unknown,
			KeyID:          []byte{1, 2, 3},
			Namespace:      "ns",
		}
		_, err := m.MarshalBinarySize()
		assert.ErrorIs(t, err, verrors.ErrUnknownSuite)
	})
}

// TestMeta_marshalBinaryToUndersized confirms MarshalBinaryTo refuses to write
// into a destination smaller than [models.Meta.MarshalBinarySize] reports.
func TestMeta_marshalBinaryToUndersized(t *testing.T) {
	m := models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          []byte{1, 2, 3},
		Namespace:      "ns",
	}
	size, err := m.MarshalBinarySize()
	assert.Ok(t, err)

	dst := make([]byte, size-1)
	_, err = m.MarshalBinaryTo(dst)
	assert.ErrorIs(t, err, verrors.ErrMalformedWire)
}

//=============================================================================
// Tests: Meta.UnmarshalBinary negative branches
//=============================================================================

// TestMeta_unmarshalBinaryRejects covers the strict-framing branches of
// [models.Meta.UnmarshalBinary]: nil receiver, short input, unsupported version,
// unknown suite, declared KeyID length past end-of-buffer, missing namespace
// length byte, and trailing junk after a well-formed Meta.
func TestMeta_unmarshalBinaryRejects(t *testing.T) {
	good, err := (models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          []byte{1, 2, 3},
		Namespace:      "ns",
	}).MarshalBinary()
	assert.Ok(t, err)

	t.Run("nil_receiver", func(t *testing.T) {
		var m *models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(good), verrors.ErrNilMetaPointer)
	})

	t.Run("short_input", func(t *testing.T) {
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary([]byte{1, 2, 3}), verrors.ErrMalformedWire)
	})

	t.Run("unsupported_version", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[0] = constants.PackageVersion + 1
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(bad), verrors.ErrUnsupportedVersion)
	})

	t.Run("unknown_suite", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[1] = byte(suite.Unknown)
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(bad), verrors.ErrUnknownSuite)
	})

	t.Run("keyid_length_overruns_buffer", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[2] = byte(len(bad)) // declared KeyID length larger than remaining bytes
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(bad), verrors.ErrMalformedWire)
	})

	t.Run("trailing_bytes_rejected", func(t *testing.T) {
		bad := append(append([]byte(nil), good...), 0xff)
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(bad), verrors.ErrMalformedWire)
	})
}

//=============================================================================
// Fuzz: Meta.UnmarshalBinary
//=============================================================================

// FuzzMeta_unmarshal exercises [models.Meta.UnmarshalBinary] against semi-random
// inputs derived from a small good/bad seed corpus. The invariants checked are:
//
//   - The parser must not panic on any input.
//   - A successful unmarshal must round-trip back to identical bytes
//     (Meta's wire format is strictly canonical: trailing bytes are rejected
//     and field order is fixed).
func FuzzMeta_unmarshal(f *testing.F) {
	good, err := (models.Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          []byte{1, 2, 3},
		Namespace:      "ns",
	}).MarshalBinary()
	assert.Ok(f, err, "seed marshal")
	f.Add(good)
	f.Add([]byte{})
	f.Add([]byte{constants.PackageVersion, byte(suite.X25519HKDFSHA256AES256GCM), 0, 0})
	f.Add(append([]byte(nil), good[:len(good)-1]...))

	f.Fuzz(func(t *testing.T, data []byte) {
		var m models.Meta
		if err := m.UnmarshalBinary(data); err != nil {
			return
		}
		out, err := m.MarshalBinary()
		assert.Ok(t, err, "unmarshal succeeded but re-marshal failed")
		assert.Equal(t, string(data), string(out), "round-trip mismatch")
	})
}
