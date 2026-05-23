package models

// Meta wire encoding: caps validation, deterministic marshal, and authenticated row metadata.

import (
	"crypto/ecdh"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
)

// Meta is authenticated metadata carried on the wire; trust fields only after AEAD verify.
type Meta struct {
	Version   uint8
	KeyID     []byte
	Namespace string // per-operation; raw []byte(Namespace) participates in caps and AAD
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
	if m.Version != constants.Version {
		return 0, perrors.ErrUnsupportedVersion
	}
	return 1 + 1 + len(m.KeyID) + 1 + len(m.Namespace), nil
}

// MarshalBinaryTo encodes Meta into dst and returns written bytes.
func (m Meta) MarshalBinaryTo(dst []byte) (int, error) {
	need, err := m.MarshalBinarySize()
	if err != nil {
		return 0, err
	}

	if len(dst) < need {
		return 0, perrors.ErrMalformedWire
	}

	off := 0
	dst[off] = m.Version
	off++
	dst[off] = byte(len(m.KeyID))
	off++
	copy(dst[off:off+len(m.KeyID)], m.KeyID)
	off += len(m.KeyID)

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

	if len(data) < 3 {
		return perrors.ErrMalformedWire
	}
	off := 0

	m.Version = data[off]
	off++
	if m.Version != constants.Version {
		return perrors.ErrUnsupportedVersion
	}

	lk := int(data[off])
	off++
	if lk > constants.MaxKeyIDBytes || off+lk > len(data) {
		return perrors.ErrMalformedWire
	}
	m.KeyID = append([]byte(nil), data[off:off+lk]...)
	off += lk

	if off >= len(data) {
		return perrors.ErrMalformedWire
	}

	ln := int(data[off])
	off++
	if ln > constants.MaxNamespaceBytes || off+ln > len(data) {
		return perrors.ErrMalformedWire
	}
	m.Namespace = string(data[off : off+ln])
	off += ln

	if off != len(data) {
		return perrors.ErrMalformedWire
	}
	return nil
}

// validateMetaCaps checks KeyID and Namespace are within their byte-length caps.
func validateMetaCaps(m Meta) error {
	if len(m.KeyID) > constants.MaxKeyIDBytes {
		return perrors.ErrMetaKeyIDTooLarge
	}
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
	if priv.Curve() != ecdh.X25519() {
		return Meta{}, perrors.ErrInvalidWrappingKey
	}

	kid := priv.PublicKey().Bytes()
	if len(kid) > constants.MaxKeyIDBytes {
		return Meta{}, perrors.ErrMetaKeyIDTooLarge
	}
	m := Meta{
		Version:   constants.Version,
		KeyID:     kid,
		Namespace: "",
	}
	if _, err := m.MarshalBinary(); err != nil {
		return Meta{}, err
	}
	return m, nil
}
