package models

// Wire framing for the full v1 sealed row: magic, format version, meta length, [Meta],
// [EphPub], [Inner].

import (
	"encoding/binary"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
)

// sealedPreambleBytes is the fixed header before variable-length meta: magic(4) + formatVersion(1) + lenMeta u16 BE(2).
const sealedPreambleBytes = 4 + 1 + 2

// Sealed is the full stored row: preamble, Meta, Eph, Body.
type Sealed struct {
	FormatVersion uint8
	Meta          Meta
	Eph           EphPub
	Body          Inner
}

// MarshalBinarySize returns the encoded byte length of Sealed.
func (s Sealed) MarshalBinarySize() (int, error) {
	metaSize, err := s.Meta.MarshalBinarySize()
	if err != nil {
		return 0, err
	}
	if metaSize > constants.MaxMetaWireBytes {
		return 0, perrors.ErrMalformedWire
	}
	return sealedPreambleBytes + metaSize + s.Eph.MarshalBinarySize() + s.Body.MarshalBinarySize(), nil
}

// MarshalBinaryTo encodes Sealed into dst and returns written bytes.
func (s Sealed) MarshalBinaryTo(dst []byte) (int, error) {
	metaSize, err := s.Meta.MarshalBinarySize()
	if err != nil {
		return 0, err
	}
	if metaSize > constants.MaxMetaWireBytes {
		return 0, perrors.ErrMalformedWire
	}
	return s.marshalBinaryToWithMetaSize(dst, metaSize)
}

// MarshalBinary encodes the full v1 wire row.
func (s Sealed) MarshalBinary() ([]byte, error) {
	metaSize, err := s.Meta.MarshalBinarySize()
	if err != nil {
		return nil, err
	}
	if metaSize > constants.MaxMetaWireBytes {
		return nil, perrors.ErrMalformedWire
	}
	need := sealedPreambleBytes + metaSize + s.Eph.MarshalBinarySize() + s.Body.MarshalBinarySize()
	out := make([]byte, need)
	_, err = s.marshalBinaryToWithMetaSize(out, metaSize)
	return out, err
}

// marshalBinaryToWithMetaSize writes the wire bytes into dst using a precomputed metaSize.
func (s Sealed) marshalBinaryToWithMetaSize(dst []byte, metaSize int) (int, error) {
	ephSize := s.Eph.MarshalBinarySize()
	bodySize := s.Body.MarshalBinarySize()
	need := sealedPreambleBytes + metaSize + ephSize + bodySize
	if len(dst) < need {
		return 0, perrors.ErrMalformedWire
	}

	off := 0

	// Preamble layout is magic||formatVersion||metaLen(u16 big-endian).
	copy(dst[off:off+4], constants.Magic)
	off += 4
	dst[off] = s.FormatVersion
	off++
	binary.BigEndian.PutUint16(dst[off:off+2], uint16(metaSize))
	off += 2

	var n int
	var err error
	// Marshal framed sections in wire order: Meta then ephemeral pubkey then Inner payload.
	n, err = s.Meta.MarshalBinaryTo(dst[off : off+metaSize])
	if err != nil {
		return 0, err
	}
	off += n

	n, err = s.Eph.MarshalBinaryTo(dst[off : off+ephSize])
	if err != nil {
		return 0, err
	}
	off += n

	n, err = s.Body.MarshalBinaryTo(dst[off : off+bodySize])
	if err != nil {
		return 0, err
	}
	off += n
	return off, nil
}

// UnmarshalBinary parses magic, dual version checks, framed meta, Eph, Body.
func (s *Sealed) UnmarshalBinary(data []byte) error {
	if s == nil {
		return perrors.ErrNilSealedPointer
	}

	// Ensure preamble is present before reading magic or framed lengths.
	if len(data) < sealedPreambleBytes {
		return perrors.ErrMalformedWire
	}
	if string(data[0:4]) != constants.Magic {
		return perrors.ErrBadMagic
	}

	// Parse top-level format version and declared metadata length.
	s.FormatVersion = data[4]
	lenMeta := int(binary.BigEndian.Uint16(data[5:7]))
	if lenMeta < 4 || lenMeta > constants.MaxMetaWireBytes {
		return perrors.ErrMalformedWire
	}

	// Enforce minimum remaining bytes for ephemeral pubkey and inner nonce+tag.
	if len(data) < sealedPreambleBytes+lenMeta+constants.EphPubBytes+constants.InnerNonceBytes+constants.GCMTagBytes {
		return perrors.ErrMalformedWire
	}
	off := sealedPreambleBytes
	metaSlice := data[off : off+lenMeta]
	off += lenMeta
	if err := s.Meta.UnmarshalBinary(metaSlice); err != nil {
		return err
	}

	// Require all version sentinels to agree: row preamble, encoded meta, and package constant.
	if s.FormatVersion != s.Meta.PackageVersion {
		return perrors.ErrVersionMismatch
	}
	if s.FormatVersion != constants.PackageVersion {
		return perrors.ErrUnsupportedVersion
	}
	if err := s.Eph.UnmarshalBinary(data[off : off+constants.EphPubBytes]); err != nil {
		return err
	}
	off += constants.EphPubBytes
	return s.Body.UnmarshalBinary(data[off:])
}
