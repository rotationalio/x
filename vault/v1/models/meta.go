package models

// Meta wire encoding: caps validation, deterministic marshal, and authenticated row metadata.

import (
	"crypto/ecdh"

	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/v1/errors"
	"go.rtnl.ai/x/vault/v1/suite"
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
	// Copy-by-value so the template Meta on the vault is never mutated in place.
	out := m
	out.Namespace = namespace

	// Enforce wire caps before any marshal or AEAD that would embed this metadata.
	if err := validateMetaCaps(out); err != nil {
		return Meta{}, err
	}
	return out, nil
}

// MarshalBinary encodes Meta in deterministic v1 layout.
func (m Meta) MarshalBinary() ([]byte, error) {
	// Check if the metadata meets the wire caps.
	if err := validateMetaCaps(m); err != nil {
		return nil, err
	}
	if m.PackageVersion != constants.PackageVersion {
		return nil, verrors.ErrUnsupportedVersion
	}
	if !m.SuiteID.Valid() {
		return nil, verrors.ErrUnknownSuite
	}
	ns := []byte(m.Namespace)

	// Deterministic layout (no padding): version | suite byte | keyID len | keyID | namespace len | namespace UTF-8.
	// Length bytes are single uint8; KeyID and namespace lengths are bounded by constants.
	out := make([]byte, 0, 1+1+1+len(m.KeyID)+1+len(ns))
	out = append(out, m.PackageVersion)
	out = append(out, byte(m.SuiteID))
	out = append(out, byte(len(m.KeyID)))
	out = append(out, m.KeyID...)
	out = append(out, byte(len(ns)))
	out = append(out, ns...)
	return out, nil
}

// UnmarshalBinary decodes Meta; rejects trailing bytes and invalid wire.
func (m *Meta) UnmarshalBinary(data []byte) error {
	if m == nil {
		return verrors.ErrNilMetaPointer
	}

	// Need at least: version, suite, keyLen, nsLen — four bytes before any variable payload.
	if len(data) < 4 {
		return verrors.ErrMalformedWire
	}
	off := 0
	m.PackageVersion = data[off]
	off++
	if m.PackageVersion != constants.PackageVersion {
		return verrors.ErrUnsupportedVersion
	}
	m.SuiteID = suite.ID(data[off])
	off++
	if !m.SuiteID.Valid() {
		return verrors.ErrUnknownSuite
	}

	// Read KeyID with explicit bounds so a corrupt length cannot read past the buffer end.
	lk := int(data[off])
	off++
	if lk > constants.MaxKeyIDBytes || off+lk > len(data) {
		return verrors.ErrMalformedWire
	}
	m.KeyID = append([]byte(nil), data[off:off+lk]...)
	off += lk

	// After KeyID we must still have the namespace length byte.
	if off >= len(data) {
		return verrors.ErrMalformedWire
	}
	ln := int(data[off])
	off++
	if ln > constants.MaxNamespaceBytes || off+ln > len(data) {
		return verrors.ErrMalformedWire
	}
	m.Namespace = string(data[off : off+ln])
	off += ln

	// Trailing bytes would mean the encoder and decoder disagree on layout; reject rather than ignore.
	if off != len(data) {
		return verrors.ErrMalformedWire
	}
	return nil
}

// validateMetaCaps checks KeyID and Namespace are within their byte-length caps.
func validateMetaCaps(m Meta) error {
	if len(m.KeyID) > constants.MaxKeyIDBytes {
		return verrors.ErrMetaKeyIDTooLarge
	}
	if len([]byte(m.Namespace)) > constants.MaxNamespaceBytes {
		return verrors.ErrMetaNamespaceTooLarge
	}
	return nil
}

// MetaFromPrivKey builds the default wire [Meta] template for priv. Namespace is empty;
// each Store/Update seals the per-call namespace into row metadata.
func MetaFromPrivKey(priv *ecdh.PrivateKey) (Meta, error) {
	if priv == nil {
		return Meta{}, verrors.ErrNilPrivateKey
	}

	// v1 envelope is defined only for X25519 long-term keys; other curves cannot derive the same suite semantics.
	if priv.Curve() != ecdh.X25519() {
		return Meta{}, verrors.ErrInvalidWrappingKey
	}
	kid := priv.PublicKey().Bytes()

	// Wire caps: if the public encoding ever exceeded MaxKeyIDBytes, we could not store this key id on the row.
	if len(kid) > constants.MaxKeyIDBytes {
		return Meta{}, verrors.ErrMetaKeyIDTooLarge
	}
	m := Meta{
		PackageVersion: constants.PackageVersion,
		SuiteID:        suite.X25519HKDFSHA256AES256GCM,
		KeyID:          kid,
		Namespace:      "",
	}

	// Marshal as a dry run: catches suite/version/keyid combinations that cannot be encoded.
	if _, err := m.MarshalBinary(); err != nil {
		return Meta{}, err
	}
	return m, nil
}
