package models

// Wire framing for the full v1 sealed row: magic, format version, meta length, [Meta],
// [DekEnvelope], [Inner].

import (
	"encoding/binary"

	"go.rtnl.ai/x/vault/v1/constants"
	verrors "go.rtnl.ai/x/vault/v1/errors"
)

// sealedPreambleBytes is the fixed header before variable-length meta: magic(4) + formatVersion(1) + lenMeta u16 BE(2).
const sealedPreambleBytes = 4 + 1 + 2

// Sealed is the full stored row: preamble, Meta, Dek, Body.
type Sealed struct {
	FormatVersion uint8
	Meta          Meta
	Dek           DekEnvelope
	Body          Inner
}

// MarshalBinary encodes the full v1 wire row.
func (s Sealed) MarshalBinary() ([]byte, error) {
	// Meta is length-prefixed on the outer row; marshal it first so we know lenMeta for the header.
	metaRaw, err := s.Meta.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if len(metaRaw) > constants.MaxMetaWireBytes {
		return nil, verrors.ErrMalformedWire
	}
	dekRaw, err := s.Dek.MarshalBinary()
	if err != nil {
		return nil, err
	}
	innerRaw, err := s.Body.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Full row: magic | outer formatVersion | big-endian meta length | meta bytes | fixed-size DekEnvelope | inner blob.
	out := make([]byte, 0, sealedPreambleBytes+len(metaRaw)+len(dekRaw)+len(innerRaw))
	out = append(out, constants.Magic...)
	out = append(out, s.FormatVersion)
	var lenMeta [2]byte
	binary.BigEndian.PutUint16(lenMeta[:], uint16(len(metaRaw)))
	out = append(out, lenMeta[:]...)
	out = append(out, metaRaw...)
	out = append(out, dekRaw...)
	out = append(out, innerRaw...)
	return out, nil
}

// UnmarshalBinary parses magic, dual version checks, framed meta, Dek, Body.
func (s *Sealed) UnmarshalBinary(data []byte) error {
	if s == nil {
		return verrors.ErrNilSealedPointer
	}
	if len(data) < sealedPreambleBytes {
		return verrors.ErrMalformedWire
	}

	// Magic is ASCII so we compare as bytes without accepting odd UTF-8 interpretations.
	if string(data[0:4]) != constants.Magic {
		return verrors.ErrBadMagic
	}
	s.FormatVersion = data[4]
	lenMeta := int(binary.BigEndian.Uint16(data[5:7]))

	// lenMeta must cover at least a minimal valid Meta and stay within the decoder's worst-case bound.
	if lenMeta < 4 || lenMeta > constants.MaxMetaWireBytes {
		return verrors.ErrMalformedWire
	}

	// Ensure the slice is long enough for preamble + meta + fixed DekEnvelope + minimal inner (nonce + tag).
	if len(data) < sealedPreambleBytes+lenMeta+constants.DekEnvelopeBytes+constants.InnerNonceBytes+constants.GCMTagBytes {
		return verrors.ErrMalformedWire
	}
	off := sealedPreambleBytes
	metaSlice := data[off : off+lenMeta]
	off += lenMeta
	if err := s.Meta.UnmarshalBinary(metaSlice); err != nil {
		return err
	}

	// Outer row format byte must match the metadata block's package version (defends against spliced blobs).
	if s.FormatVersion != s.Meta.PackageVersion {
		return verrors.ErrVersionMismatch
	}
	if s.FormatVersion != constants.PackageVersion {
		return verrors.ErrUnsupportedVersion
	}

	// DekEnvelope is fixed width for v1 suite; no length prefix between meta and inner.
	if err := s.Dek.UnmarshalBinary(data[off : off+constants.DekEnvelopeBytes]); err != nil {
		return err
	}
	off += constants.DekEnvelopeBytes

	// Remainder is the inner structure (nonce + ciphertext+tag); length varies only with plaintext size inside Inner.
	if err := s.Body.UnmarshalBinary(data[off:]); err != nil {
		return err
	}
	return nil
}
