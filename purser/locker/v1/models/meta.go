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

//=============================================================================
// Meta API
//=============================================================================

// WithNamespace returns a copy of [Meta] with [Meta.Namespace] set to namespace.
func (m Meta) WithNamespace(namespace string) (Meta, error) {
	out := m
	out.Namespace = namespace
	if err := validateMetaCaps(out); err != nil {
		return Meta{}, err
	}
	return out, nil
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

	return Meta{
		Version:   constants.Version,
		KeyID:     kid,
		Namespace: "",
	}, nil
}

//=============================================================================
// Meta Marshal/Unmarshal API
//=============================================================================

// MarshalBinary encodes Meta in deterministic v1 layout.
func (m Meta) MarshalBinary() ([]byte, error) {
	var (
		need int
		out  []byte
		err  error
	)

	need, err = m.MarshalBinarySize()
	if err != nil {
		return nil, err
	}
	out = make([]byte, need)
	_, err = m.MarshalBinaryTo(out)
	return out, err
}

// UnmarshalBinary decodes Meta; rejects trailing bytes and invalid wire.
func (m *Meta) UnmarshalBinary(data []byte) error {
	if len(data) < 1 {
		return perrors.ErrMalformedWire
	}
	return unmarshalMetaInto(m, data, data[0])
}

// MarshalBinaryTo encodes Meta into dst and returns written bytes.
func (m Meta) MarshalBinaryTo(dst []byte) (int, error) {
	var (
		need  int
		err   error
		off   int
		nsLen int
	)

	need, err = m.MarshalBinarySize()
	if err != nil {
		return 0, err
	}
	if len(dst) < need {
		return 0, perrors.ErrMalformedWire
	}

	// Layout: version | keyID len | keyID | namespace len | namespace.
	off = 0
	dst[off] = m.Version
	off++
	dst[off] = byte(len(m.KeyID))
	off++
	copy(dst[off:off+len(m.KeyID)], m.KeyID)
	off += len(m.KeyID)

	nsLen = len(m.Namespace)
	dst[off] = byte(nsLen)
	off++
	copy(dst[off:off+nsLen], m.Namespace)
	off += nsLen
	return off, nil
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

//=============================================================================
// Meta validation
//=============================================================================

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

//=============================================================================
// Meta wire parse
//=============================================================================

// validateMetaWireLayout checks meta wire: version, key id, namespace, no trailing bytes.
func validateMetaWireLayout(meta []byte) error {
	var (
		off int
		lk  int
		ln  int
	)

	if len(meta) < 3 {
		return perrors.ErrMalformedWire
	}
	if meta[0] != constants.Version {
		return perrors.ErrUnsupportedVersion
	}

	// Length-prefixed key id.
	off = 1
	lk = int(meta[off])
	off++
	if lk > constants.MaxKeyIDBytes || off+lk > len(meta) {
		return perrors.ErrMalformedWire
	}
	off += lk

	// Length-prefixed namespace.
	if off >= len(meta) {
		return perrors.ErrMalformedWire
	}
	ln = int(meta[off])
	off++
	if ln > constants.MaxNamespaceBytes || off+ln > len(meta) {
		return perrors.ErrMalformedWire
	}
	off += ln

	if off != len(meta) {
		return perrors.ErrMalformedWire
	}
	return nil
}

// unmarshalMetaInto decodes meta wire into m; checks row and meta version bytes match.
func unmarshalMetaInto(m *Meta, meta []byte, preambleVersion uint8) error {
	var (
		off int
		lk  int
		ln  int
		err error
	)

	if m == nil {
		return perrors.ErrNilMetaPointer
	}
	if err = validateMetaWireLayout(meta); err != nil {
		return err
	}
	if meta[0] != preambleVersion {
		return perrors.ErrVersionMismatch
	}
	if preambleVersion != constants.Version {
		return perrors.ErrUnsupportedVersion
	}

	off = 1
	lk = int(meta[off])
	off++
	m.Version = meta[0]
	m.KeyID = append([]byte(nil), meta[off:off+lk]...)
	off += lk

	ln = int(meta[off])
	off++
	m.Namespace = string(meta[off : off+ln])
	return nil
}

// keyIDFromMetaWire returns a defensive copy of the key id from meta wire bytes.
func keyIDFromMetaWire(meta []byte) ([]byte, error) {
	var (
		lk  int
		err error
	)

	if err = validateMetaWireLayout(meta); err != nil {
		return nil, err
	}
	lk = int(meta[1])
	return append([]byte(nil), meta[2:2+lk]...), nil
}

// metaNamespaceMatches reports whether requestedNS equals the namespace in meta wire.
func metaNamespaceMatches(meta []byte, requestedNS string) error {
	var (
		off int
		ln  int
		err error
	)

	if err = validateMetaWireLayout(meta); err != nil {
		return err
	}

	// Skip version(1), lk(1), key id; ln byte starts namespace length.
	off = 2 + int(meta[1])
	ln = int(meta[off])
	off++
	ns := meta[off : off+ln]
	if len(requestedNS) != ln {
		return perrors.ErrNamespaceMismatch
	}
	for i := range ln {
		if requestedNS[i] != ns[i] {
			return perrors.ErrNamespaceMismatch
		}
	}
	return nil
}
