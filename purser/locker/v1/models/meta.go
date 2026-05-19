package models

// Meta wire encoding: caps validation, deterministic marshal, and authenticated row metadata.

import (
	"crypto/ecdh"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/suite"
)

// Meta is authenticated metadata carried on the wire; trust fields only after AEAD verify.
type Meta struct {
	PackageVersion uint8
	SuiteID        suite.ID
	KeyID          []byte
	Namespace      string // per-operation; raw []byte(Namespace) participates in caps and AAD
}

// WithNamespace returns a copy of [Meta] with [Meta.Namespace] set to namespace.
func (m Meta) WithNamespace(namespace string) (Meta, error) {
	out := m
	out.Namespace = namespace
	if err := validateMetaCaps(out); err != nil {
		return Meta{}, err
	}
	return out, nil
}

// MarshalBinarySize returns the encoded byte length of Meta.
func (m Meta) MarshalBinarySize() (int, error) {
	if err := validateMetaCaps(m); err != nil {
		return 0, err
	}
	if m.PackageVersion != constants.PackageVersion {
		return 0, perrors.ErrUnsupportedVersion
	}
	if !m.SuiteID.Valid() {
		return 0, perrors.ErrUnknownSuite
	}
	return 1 + 1 + 1 + len(m.KeyID) + 1 + len(m.Namespace), nil
}

// MarshalBinaryTo encodes Meta into dst and returns written bytes.
func (m Meta) MarshalBinaryTo(dst []byte) (int, error) {
	need, err := m.MarshalBinarySize()
	if err != nil {
		return 0, err
	}

	// Reject undersized destinations up front so callers can pre-size once and reuse buffers.
	if len(dst) < need {
		return 0, perrors.ErrMalformedWire
	}

	off := 0

	// Encode fixed one-byte header fields first so variable fields can stream after them.
	dst[off] = m.PackageVersion
	off++
	dst[off] = byte(m.SuiteID)
	off++
	dst[off] = byte(len(m.KeyID))
	off++

	// KeyID occupies exactly the declared length and is copied verbatim.
	copy(dst[off:off+len(m.KeyID)], m.KeyID)
	off += len(m.KeyID)

	// Namespace follows as a length-prefixed UTF-8 byte slice.
	nsLen := len(m.Namespace)
	dst[off] = byte(nsLen)
	off++
	copy(dst[off:off+nsLen], m.Namespace)
	off += nsLen
	return off, nil
}

// MarshalBinary encodes Meta in deterministic v1 layout.
func (m Meta) MarshalBinary() ([]byte, error) {
	need, err := m.MarshalBinarySize()
	if err != nil {
		return nil, err
	}
	out := make([]byte, need)
	_, err = m.MarshalBinaryTo(out)
	return out, err
}

// UnmarshalBinary decodes Meta; rejects trailing bytes and invalid wire.
func (m *Meta) UnmarshalBinary(data []byte) error {
	if m == nil {
		return perrors.ErrNilMetaPointer
	}

	// Minimum framing is version, suite, key-length, and namespace-length bytes.
	if len(data) < 4 {
		return perrors.ErrMalformedWire
	}
	off := 0

	// Parse and validate package version early so downstream parsing can assume known layout.
	m.PackageVersion = data[off]
	off++
	if m.PackageVersion != constants.PackageVersion {
		return perrors.ErrUnsupportedVersion
	}

	// Suite ID gates key-agreement and AEAD behavior; unknown suites are hard failures.
	m.SuiteID = suite.ID(data[off])
	off++
	if !m.SuiteID.Valid() {
		return perrors.ErrUnknownSuite
	}

	// Read the declared KeyID length and ensure the slice stays in-bounds before copying.
	lk := int(data[off])
	off++
	if lk > constants.MaxKeyIDBytes || off+lk > len(data) {
		return perrors.ErrMalformedWire
	}
	m.KeyID = append([]byte(nil), data[off:off+lk]...)
	off += lk

	// A namespace length byte must exist even when namespace itself is empty.
	if off >= len(data) {
		return perrors.ErrMalformedWire
	}

	// Parse namespace length and verify the declared bytes are available.
	ln := int(data[off])
	off++
	if ln > constants.MaxNamespaceBytes || off+ln > len(data) {
		return perrors.ErrMalformedWire
	}
	m.Namespace = string(data[off : off+ln])
	off += ln

	// v1 metadata decoding is strict: trailing bytes indicate malformed framing.
	if off != len(data) {
		return perrors.ErrMalformedWire
	}
	return nil
}

// validateMetaCaps checks KeyID and Namespace are within their byte-length caps.
func validateMetaCaps(m Meta) error {
	// KeyID is serialized behind a single-byte length field and capped by constants.
	if len(m.KeyID) > constants.MaxKeyIDBytes {
		return perrors.ErrMetaKeyIDTooLarge
	}

	// Namespace cap is enforced in bytes to match on-wire framing.
	if len(m.Namespace) > constants.MaxNamespaceBytes {
		return perrors.ErrMetaNamespaceTooLarge
	}
	return nil
}

// MetaFromPrivKey builds the default wire [Meta] template for priv. Namespace is empty.
func MetaFromPrivKey(priv *ecdh.PrivateKey) (Meta, error) {
	if priv == nil {
		return Meta{}, perrors.ErrNilPrivateKey
	}

	// v1 currently supports only X25519 wrapping keys.
	if priv.Curve() != ecdh.X25519() {
		return Meta{}, perrors.ErrInvalidWrappingKey
	}

	// The default KeyID is the encoded X25519 public key bytes.
	kid := priv.PublicKey().Bytes()
	if len(kid) > constants.MaxKeyIDBytes {
		return Meta{}, perrors.ErrMetaKeyIDTooLarge
	}
	m := Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          kid,
		Namespace:      "",
	}
	// Re-encode once to validate all invariants through the same public wire path.
	if _, err := m.MarshalBinary(); err != nil {
		return Meta{}, err
	}
	return m, nil
}
