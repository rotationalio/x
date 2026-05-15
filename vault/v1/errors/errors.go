// Package errors defines stable sentinel errors for vault v1 wire decoding, [models], and [suite].
// Operational errors shared across versions live in [go.rtnl.ai/x/vault/errors]. Classify with the standard library errors.Is.
package errors

import stderrors "errors"

//=============================================================================
// Wire metadata, framing, and suite metadata on rows
//=============================================================================

var (
	// ErrNilInnerPointer means [*models.Inner.UnmarshalBinary] was called with a nil receiver.
	ErrNilInnerPointer = stderrors.New("vault/v1: nil Inner receiver")

	// ErrNilDekEnvelopePointer means [*models.DekEnvelope.UnmarshalBinary] was called with a nil receiver.
	ErrNilDekEnvelopePointer = stderrors.New("vault/v1: nil DekEnvelope receiver")

	// ErrMalformedWire means bytes are corrupt, truncated, or not a valid v1 wire layout for the operation.
	ErrMalformedWire = stderrors.New("vault/v1: malformed wire encoding")

	// ErrNilMetaPointer means [*models.Meta.UnmarshalBinary] was called with a nil receiver.
	ErrNilMetaPointer = stderrors.New("vault/v1: nil Meta")

	// ErrMetaKeyIDTooLarge means [models.Meta.KeyID] exceeds the wire limit ([constants.MaxKeyIDBytes]).
	ErrMetaKeyIDTooLarge = stderrors.New("vault/v1: meta key identifier exceeds limit")

	// ErrMetaNamespaceTooLarge means [models.Meta.Namespace] exceeds the wire limit ([constants.MaxNamespaceBytes]).
	ErrMetaNamespaceTooLarge = stderrors.New("vault/v1: namespace exceeds limit")

	// ErrNilSealedPointer means [*models.Sealed.UnmarshalBinary] was called with a nil receiver.
	ErrNilSealedPointer = stderrors.New("vault/v1: nil Sealed receiver")

	// ErrBadMagic means the wire blob does not begin with the expected v1 magic bytes ([constants.Magic]).
	ErrBadMagic = stderrors.New("vault/v1: bad magic")

	// ErrUnsupportedVersion means the row format version byte is not supported by this module.
	ErrUnsupportedVersion = stderrors.New("vault/v1: unsupported version")

	// ErrVersionMismatch means the outer format version disagrees with the decoded metadata version.
	ErrVersionMismatch = stderrors.New("vault/v1: unsupported format version")

	// ErrUnknownSuite means the metadata suite id is not a known v1 suite.
	ErrUnknownSuite = stderrors.New("vault/v1: unknown suite")

	// ErrNamespaceMismatch means the row was opened under a namespace that does not match the row metadata.
	ErrNamespaceMismatch = stderrors.New("vault/v1: namespace mismatch")
)

//=============================================================================
// Suite ID parse and marshal ([suite])
//=============================================================================

var (
	// ErrNilSuiteID means [*suite.ID.UnmarshalBinary] was called with a nil receiver.
	ErrNilSuiteID = stderrors.New("vault/v1/suite: nil ID receiver")

	// ErrInvalidSuiteWire means decoded suite bytes are not the expected length.
	ErrInvalidSuiteWire = stderrors.New("vault/v1/suite: invalid wire encoding")

	// ErrInvalidSuiteValue means a numeric suite id is not usable.
	ErrInvalidSuiteValue = stderrors.New("vault/v1/suite: invalid suite value")

	// ErrUnknownSuiteName means the string does not name a known suite.
	ErrUnknownSuiteName = stderrors.New("vault/v1/suite: unknown suite name")

	// ErrInvalidSuiteInput means the argument type is not supported for [suite.Parse].
	ErrInvalidSuiteInput = stderrors.New("vault/v1/suite: invalid input type")
)
