// Package models defines v1 locker wire types and framing for sealed rows.
package models

// This file defines the v1 sealed-row wire format and three ways to read or write it.
//
// On-disk / on-the-wire layout (see [wire.PreambleBytes] and [Meta.MarshalBinary]):
//
//	PURS | formatVersion(1) | metaLen u16 BE | Meta bytes | Eph(32) | inner nonce(12) | ciphertext+tag
//
// The Meta section is authenticated as GCM additional data (AAD) for the inner ciphertext.
// At seal time, locker builds meta once, encrypts with those bytes as AAD, then copies the
// same bytes into the row via [Sealed.BindMetaWire] before [Sealed.MarshalBinary]. At open time,
// [ParseOpenWire] passes a subslice of those bytes to gcm.OpenInner—never re-marshaling from a [Meta] struct.
//
// API choice:
//   - [Sealed.UnmarshalBinary]: tests, tooling, or callers that need typed [Meta], [EphPub], [Inner].
//     Not used on locker decrypt (see [ParseOpenWire]). Allocates KeyID, namespace string, and a
//     full copy of inner ciphertext—by design for a defensive, owned [Sealed] value.
//   - [ParseKeyIDFromSealed]: registry/keyring routing; reads preamble + meta only (one key-id copy).
//   - [ParseOpenWire]: locker decrypt; subslices into the input buffer, namespace checked before AEAD.
//
// Shared parser: [parseSealedFrame] splits preamble+meta from Eph||Inner. Fast paths alias the
// caller's buffer; do not mutate wire bytes until crypto completes.

import (
	"bytes"
	"encoding/binary"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/wire"
)

// sealedPreambleBytes is the fixed header before variable-length meta ([wire.PreambleBytes]).
const sealedPreambleBytes = wire.PreambleBytes

// Sealed is the full stored row: preamble, Meta, Eph, Body.
type Sealed struct {
	FormatVersion uint8
	Meta          Meta
	Eph           EphPub
	Body          Inner

	// metaWire, when set by [Sealed.BindMetaWire], is used by marshal methods
	// instead of re-encoding [Meta].
	metaWire []byte
}

//=============================================================================
// Sealed row API
//=============================================================================

// BindMetaWire pins on-wire meta bytes for [Sealed.MarshalBinary] (same bytes
// as GCM AAD when sealing). Caller must not mutate meta until marshaling
// finishes.
func (s *Sealed) BindMetaWire(meta []byte) {
	s.metaWire = meta
}

//=============================================================================
// Sealed Marshal/Unmarshal API
//=============================================================================

// MarshalBinarySize returns the encoded byte length of Sealed.
func (s Sealed) MarshalBinarySize() (int, error) {
	var (
		metaSize int
		err      error
	)

	metaSize, err = s.sealedMetaLen()
	if err != nil {
		return 0, err
	}
	return sealedWireSize(metaSize, s.Eph, s.Body), nil
}

// MarshalBinaryTo encodes Sealed into dst and returns written bytes.
func (s Sealed) MarshalBinaryTo(dst []byte) (int, error) {
	var (
		meta []byte
		err  error
	)

	meta, err = s.sealedMetaBytes()
	if err != nil {
		return 0, err
	}
	return writeSealedWire(dst, s.FormatVersion, meta, s.Eph, s.Body)
}

// MarshalBinary encodes the full v1 wire row.
func (s Sealed) MarshalBinary() ([]byte, error) {
	var (
		meta []byte
		out  []byte
		err  error
	)

	meta, err = s.sealedMetaBytes()
	if err != nil {
		return nil, err
	}
	out = make([]byte, sealedWireSize(len(meta), s.Eph, s.Body))

	_, err = writeSealedWire(out, s.FormatVersion, meta, s.Eph, s.Body)
	return out, err
}

// UnmarshalBinary parses magic, dual version checks, framed meta, Eph, and Body.
// This is the allocating decode path; production decrypt uses [ParseOpenWire] instead.
func (s *Sealed) UnmarshalBinary(data []byte) error {
	var (
		meta []byte
		rest []byte
		err  error
	)

	if s == nil {
		return perrors.ErrNilSealedPointer
	}

	// Same frame split as [parseSealedFrame] / [ParseOpenWire] (validates magic and lengths).
	s.FormatVersion, meta, rest, err = parseSealedFrame(data)
	if err != nil {
		return err
	}

	// Fill [Meta] (KeyID copy + namespace string); cross-check preamble vs meta version.
	if err = unmarshalMetaInto(&s.Meta, meta, s.FormatVersion); err != nil {
		return err
	}

	// Eph is fixed width; inner decode copies nonce and ciphertext+tag into [Body].
	copy(s.Eph[:], rest[:constants.EphPubBytes])
	return unmarshalInnerInto(&s.Body, rest[constants.EphPubBytes:])
}

//=============================================================================
// Sealed wire parse
//=============================================================================

// ParseKeyIDFromSealed returns the sealing key id without unmarshaling Eph or Inner.
func ParseKeyIDFromSealed(data []byte) ([]byte, error) {
	var (
		formatVersion uint8
		meta          []byte
		err           error
	)

	formatVersion, meta, _, err = parseSealedFrame(data)
	if err != nil {
		return nil, err
	}
	if formatVersion != constants.Version {
		return nil, perrors.ErrUnsupportedVersion
	}
	return keyIDFromMetaWire(meta)
}

// ParseOpenWire validates sealed wire and returns subslices for decrypt.
// requestedNS is compared to meta before any AEAD work. Returned slices alias data.
func ParseOpenWire(data []byte, requestedNS string) (
	formatVersion uint8,
	metaAAD []byte,
	eph []byte,
	nonce [constants.InnerNonceBytes]byte,
	payload []byte,
	err error,
) {
	var (
		meta []byte
		rest []byte
	)

	formatVersion, meta, rest, err = parseSealedFrame(data)
	if err != nil {
		return 0, nil, nil, nonce, nil, err
	}
	if formatVersion != constants.Version {
		return 0, nil, nil, nonce, nil, perrors.ErrUnsupportedVersion
	}

	if err = metaNamespaceMatches(meta, formatVersion, requestedNS); err != nil {
		return 0, nil, nil, nonce, nil, err
	}

	// [parseSealedFrame] already enforced minimum Eph||Inner size.
	metaAAD = meta
	eph = rest[:constants.EphPubBytes]
	rest = rest[constants.EphPubBytes:]
	copy(nonce[:], rest[:constants.InnerNonceBytes])
	payload = rest[constants.InnerNonceBytes:]
	return formatVersion, metaAAD, eph, nonce, payload, nil
}

//=============================================================================
// Sealed meta marshal
//=============================================================================

// sealedMetaLen returns the on-wire meta byte length (bound metaWire or [Meta] encoding).
func (s Sealed) sealedMetaLen() (int, error) {
	if len(s.metaWire) > 0 {
		if err := validateMetaWireLen(len(s.metaWire)); err != nil {
			return 0, err
		}
		return len(s.metaWire), nil
	}

	var (
		metaSize int
		err      error
	)

	metaSize, err = s.Meta.MarshalBinarySize()
	if err != nil {
		return 0, err
	}
	if metaSize > constants.MaxMetaWireBytes {
		return 0, perrors.ErrMalformedWire
	}
	return metaSize, nil
}

// sealedMetaBytes returns on-wire meta bytes (bound metaWire or [Meta.MarshalBinary]).
func (s Sealed) sealedMetaBytes() ([]byte, error) {
	if len(s.metaWire) > 0 {
		if err := validateMetaWireLen(len(s.metaWire)); err != nil {
			return nil, err
		}
		return s.metaWire, nil
	}
	return s.Meta.MarshalBinary()
}

// validateMetaWireLen rejects meta lengths outside the v1 wire bounds.
func validateMetaWireLen(metaLen int) error {
	if metaLen < 4 || metaLen > constants.MaxMetaWireBytes {
		return perrors.ErrMalformedWire
	}
	return nil
}

//=============================================================================
// Sealed wire encode
//=============================================================================

// writeSealedWire writes preamble, meta copy, ephemeral pubkey, and inner ciphertext.
func writeSealedWire(dst []byte, formatVersion uint8, meta []byte, eph EphPub, body Inner) (int, error) {
	var (
		off int
		n   int
		err error
	)

	if err = validateMetaWireLen(len(meta)); err != nil {
		return 0, err
	}
	if len(dst) < sealedWireSize(len(meta), eph, body) {
		return 0, perrors.ErrMalformedWire
	}

	off = writeSealedPreamble(dst, formatVersion, len(meta))
	copy(dst[off:off+len(meta)], meta)
	off += len(meta)

	n, err = writeSealedTail(dst[off:], eph, body)
	if err != nil {
		return 0, err
	}
	return off + n, nil
}

// writeSealedPreamble writes magic, format version, and meta length; returns bytes written.
func writeSealedPreamble(dst []byte, formatVersion uint8, metaLen int) int {
	copy(dst[:wire.MagicLen], wire.Magic)
	dst[wire.MagicLen] = formatVersion
	binary.BigEndian.PutUint16(dst[wire.VersionOffset+1:wire.PreambleBytes], uint16(metaLen))
	return sealedPreambleBytes
}

// writeSealedTail writes Eph and Body; returns bytes written.
func writeSealedTail(dst []byte, eph EphPub, body Inner) (int, error) {
	var (
		ephSize  = eph.MarshalBinarySize()
		bodySize = body.MarshalBinarySize()
		off      int
		n        int
		err      error
	)

	if len(dst) < ephSize+bodySize {
		return 0, perrors.ErrMalformedWire
	}

	n, err = eph.MarshalBinaryTo(dst[:ephSize])
	if err != nil {
		return 0, err
	}
	off = n

	n, err = body.MarshalBinaryTo(dst[off : off+bodySize])
	if err != nil {
		return 0, err
	}
	return off + n, nil
}

// sealedWireSize returns the encoded byte length for meta, Eph, and Body sections.
func sealedWireSize(metaLen int, eph EphPub, body Inner) int {
	return sealedPreambleBytes + metaLen + eph.MarshalBinarySize() + body.MarshalBinarySize()
}

//=============================================================================
// Sealed wire parse
//=============================================================================

// parseSealedFrame splits preamble and meta from the trailing Eph||Inner section.
// Returned slices alias data; do not mutate data until crypto finishes.
func parseSealedFrame(data []byte) (formatVersion uint8, meta []byte, rest []byte, err error) {
	var (
		lenMeta   int
		metaStart int
		metaEnd   int
	)

	if len(data) < sealedPreambleBytes {
		return 0, nil, nil, perrors.ErrMalformedWire
	}
	if !bytes.Equal(data[:wire.MagicLen], wire.MagicBytes) {
		return 0, nil, nil, perrors.ErrBadMagic
	}

	formatVersion = data[wire.VersionOffset]
	lenMeta = int(binary.BigEndian.Uint16(data[wire.VersionOffset+1 : wire.PreambleBytes]))
	if lenMeta < 4 || lenMeta > constants.MaxMetaWireBytes {
		return 0, nil, nil, perrors.ErrMalformedWire
	}

	metaStart = sealedPreambleBytes
	metaEnd = metaStart + lenMeta
	if len(data) < metaEnd+constants.EphPubBytes+constants.InnerNonceBytes+constants.GCMTagBytes {
		return 0, nil, nil, perrors.ErrMalformedWire
	}
	return formatVersion, data[metaStart:metaEnd], data[metaEnd:], nil
}
