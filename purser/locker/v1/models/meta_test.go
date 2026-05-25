package models_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"go.rtnl.ai/x/assert"
	verrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/models"
)

// TestMeta_roundtrip checks MarshalBinary and UnmarshalBinary preserve fields.
func TestMeta_roundtrip(t *testing.T) {
	m := models.Meta{
		Version:   constants.Version,
		KeyID:     []byte{1, 2, 3},
		Namespace: "ns-a",
	}
	b, err := m.MarshalBinary()
	assert.Ok(t, err)
	var got models.Meta
	assert.Ok(t, got.UnmarshalBinary(b))
	assert.Equal(t, m.Version, got.Version)
	assert.Equal(t, m.KeyID, got.KeyID)
	assert.Equal(t, m.Namespace, got.Namespace)
}

// TestMetaFromPrivKey verifies MetaFromPrivKey fills version and key id from X25519.
func TestMetaFromPrivKey(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	m, err := models.MetaFromPrivKey(priv)
	assert.Ok(t, err)
	assert.Equal(t, constants.Version, m.Version)
	assert.Equal(t, priv.PublicKey().Bytes(), m.KeyID)
	assert.Equal(t, "", m.Namespace)
}

// TestMetaFromPrivKey_nil asserts nil private key returns ErrNilPrivateKey.
func TestMetaFromPrivKey_nil(t *testing.T) {
	_, err := models.MetaFromPrivKey(nil)
	assert.ErrorIs(t, err, verrors.ErrNilPrivateKey)
}

// TestMetaFromPrivKey_nonX25519 asserts non-X25519 key returns ErrInvalidWrappingKey.
func TestMetaFromPrivKey_nonX25519(t *testing.T) {
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	assert.Ok(t, err)
	_, err = models.MetaFromPrivKey(priv)
	assert.ErrorIs(t, err, verrors.ErrInvalidWrappingKey)
}

// TestMeta_withNamespace covers namespace replacement and oversized namespace rejection.
func TestMeta_withNamespace(t *testing.T) {
	base := models.Meta{
		Version:   constants.Version,
		KeyID:     []byte{1, 2, 3},
		Namespace: "",
	}
	t.Run("sets_namespace", func(t *testing.T) {
		got, err := base.WithNamespace("app")
		assert.Ok(t, err)
		assert.Equal(t, "app", got.Namespace)
		assert.Equal(t, "", base.Namespace)
	})
	t.Run("oversized_namespace_rejected", func(t *testing.T) {
		big := string(make([]byte, constants.MaxNamespaceBytes+1))
		_, err := base.WithNamespace(big)
		assert.ErrorIs(t, err, verrors.ErrMetaNamespaceTooLarge)
	})
}

// TestMeta_marshalCaps asserts marshal rejects oversize fields and bad version.
func TestMeta_marshalCaps(t *testing.T) {
	t.Run("empty_keyid", func(t *testing.T) {
		m := models.Meta{Version: constants.Version, KeyID: nil, Namespace: "ns"}
		_, err := m.MarshalBinarySize()
		assert.ErrorIs(t, err, verrors.ErrMalformedWire)
	})
	t.Run("oversized_keyid", func(t *testing.T) {
		m := models.Meta{
			Version:   constants.Version,
			KeyID:     make([]byte, constants.MaxKeyIDBytes+1),
			Namespace: "",
		}
		_, err := m.MarshalBinarySize()
		assert.ErrorIs(t, err, verrors.ErrMetaKeyIDTooLarge)
	})
	t.Run("oversized_namespace", func(t *testing.T) {
		m := models.Meta{
			Version:   constants.Version,
			KeyID:     []byte{1, 2, 3},
			Namespace: string(make([]byte, constants.MaxNamespaceBytes+1)),
		}
		_, err := m.MarshalBinary()
		assert.ErrorIs(t, err, verrors.ErrMetaNamespaceTooLarge)
	})
	t.Run("unsupported_version", func(t *testing.T) {
		m := models.Meta{
			Version:   constants.Version + 1,
			KeyID:     []byte{1, 2, 3},
			Namespace: "ns",
		}
		_, err := m.MarshalBinarySize()
		assert.ErrorIs(t, err, verrors.ErrUnsupportedVersion)
	})
}

// TestMeta_marshalBinaryToUndersized confirms MarshalBinaryTo rejects short dst.
func TestMeta_marshalBinaryToUndersized(t *testing.T) {
	m := models.Meta{
		Version:   constants.Version,
		KeyID:     []byte{1, 2, 3},
		Namespace: "ns",
	}
	size, err := m.MarshalBinarySize()
	assert.Ok(t, err)
	dst := make([]byte, size-1)
	_, err = m.MarshalBinaryTo(dst)
	assert.ErrorIs(t, err, verrors.ErrMalformedWire)
}

// TestMeta_unmarshalBinaryRejects covers strict-framing branches of UnmarshalBinary.
func TestMeta_unmarshalBinaryRejects(t *testing.T) {
	good, err := (models.Meta{
		Version:   constants.Version,
		KeyID:     []byte{1, 2, 3},
		Namespace: "ns",
	}).MarshalBinary()
	assert.Ok(t, err)

	t.Run("nil_receiver", func(t *testing.T) {
		var m *models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(good), verrors.ErrNilMetaPointer)
	})
	t.Run("short_input", func(t *testing.T) {
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary([]byte{1, 2}), verrors.ErrMalformedWire)
	})
	t.Run("unsupported_version", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[0] = constants.Version + 1
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(bad), verrors.ErrUnsupportedVersion)
	})
	t.Run("keyid_length_overruns_buffer", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[1] = byte(len(bad))
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(bad), verrors.ErrMalformedWire)
	})
	t.Run("trailing_bytes_rejected", func(t *testing.T) {
		bad := append(append([]byte(nil), good...), 0xff)
		var m models.Meta
		assert.ErrorIs(t, m.UnmarshalBinary(bad), verrors.ErrMalformedWire)
	})
}

// FuzzMeta_unmarshal exercises Meta.UnmarshalBinary on random inputs.
func FuzzMeta_unmarshal(f *testing.F) {
	good, err := (models.Meta{
		Version:   constants.Version,
		KeyID:     []byte{1, 2, 3},
		Namespace: "ns",
	}).MarshalBinary()
	assert.Ok(f, err, "seed marshal")
	f.Add(good)
	f.Add([]byte{})
	f.Add([]byte{constants.Version, 0, 0})
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
