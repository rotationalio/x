// Package errors defines stable operational and crypto-helper sentinel errors shared across purser
// versions. Classify with [errors.Is] from the standard library "errors" package.
package errors

import stderrors "errors"

//=============================================================================
// Purser construction and receiver
//=============================================================================

var (
	// ErrNilPrivateKey means the private key is nil or missing where one is required.
	ErrNilPrivateKey = stderrors.New("purser: PrivateKey is required")

	// ErrInvalidWrappingKey means the long-term key is not usable with this module (for example, not X25519).
	ErrInvalidWrappingKey = stderrors.New("purser: wrapping key not supported")

	// ErrInvalidNewArgs means a constructor was called without required dependencies.
	ErrInvalidNewArgs = stderrors.New("purser: required dependency is nil")

	// ErrNilPurser means a method was called on a nil Purser receiver.
	ErrNilPurser = stderrors.New("purser: purser is nil")
)

//=============================================================================
// Cryptography (inner / wrap AEAD and helpers)
//=============================================================================

var (
	// ErrInvalidAEADKey means key material was rejected for AES-GCM construction.
	ErrInvalidAEADKey = stderrors.New("purser: invalid AEAD key material")

	// ErrDecrypt means decryption or GCM authentication failed.
	ErrDecrypt = stderrors.New("purser: decrypt failed")

	// ErrSealFailed means sealing failed, for example when reading random bytes for a nonce.
	ErrSealFailed = stderrors.New("purser: seal failed")

	// ErrNilAEAD means a crypto helper received a nil AEAD implementation.
	ErrNilAEAD = stderrors.New("purser: nil AEAD")

	// ErrMalformedParameters means a gcm helper received invalid lengths, nonce size, or AEAD output layout.
	ErrMalformedParameters = stderrors.New("purser: malformed parameters")
)

//=============================================================================
// Identifiers, lockers, and hold
//=============================================================================

var (
	// ErrInvalidHexIdentifier means the string is not a valid 32-character hex encoding of 16 bytes.
	ErrInvalidHexIdentifier = stderrors.New("purser: invalid hex identifier")

	// ErrInvalidIdentifier means the identifier implementation rejected the id.
	ErrInvalidIdentifier = stderrors.New("purser: invalid identifier")

	// ErrNoLocker means no locker is registered for the key identifier found in a ciphertext blob.
	ErrNoLocker = stderrors.New("purser: no locker registered for key identifier")

	// ErrDuplicateKeyID means a locker with the same key identifier is already registered in the keyring.
	ErrDuplicateKeyID = stderrors.New("purser: duplicate key identifier in keyring")

	// ErrDuplicateKey means a hold create or write conflicted with an existing row id in that namespace.
	ErrDuplicateKey = stderrors.New("purser: duplicate key")

	// ErrNotFound means no sealed row exists for the requested namespace and id.
	ErrNotFound = stderrors.New("purser: secret not found")

	// ErrCASFailed means CompareAndSwap lost the race.
	ErrCASFailed = stderrors.New("purser: secret was modified concurrently; compare-and-swap lost")

	// ErrMoveNamespaceIncomplete means MoveNamespace could not finish moving every matching row.
	ErrMoveNamespaceIncomplete = stderrors.New("purser: namespace relocation incomplete")

	// ErrHold means the underlying hold implementation returned a failure unrelated to purser logic.
	ErrHold = stderrors.New("purser: hold operation failed")

	// ErrWrongCurrent means CompareAndSwap failed because stored plaintext did not equal expected currentPlain.
	ErrWrongCurrent = stderrors.New("purser: stored secret does not match expected plaintext")
)

//=============================================================================
// Keys ([keyring] at go.rtnl.ai/x/purser/keyring)
//=============================================================================

var (
	// ErrInvalidOut means the output buffer length is not valid for the requested operation.
	ErrInvalidOut = stderrors.New("purser/keyring: invalid output length")

	// ErrInvalidSeed means the seed length is not valid for the target locker's FromSeed.
	ErrInvalidSeed = stderrors.New("purser/keyring: invalid seed")

	// ErrInvalidSalt means the salt length is not valid for [keyring.Derive].
	ErrInvalidSalt = stderrors.New("purser/keyring: invalid salt")

	// ErrNilPassword means [keyring.Derive] received a nil password slice.
	ErrNilPassword = stderrors.New("purser/keyring: nil password")

	// ErrRandSalt means reading random bytes for a new salt failed.
	ErrRandSalt = stderrors.New("purser/keyring: failed to read random salt")
)

//=============================================================================
// JSON and UTF-8 wrappers ([wrappers/json], [wrappers/string])
//=============================================================================

var (
	// ErrJSONMarshal means JSON encoding of a store payload failed before encryption.
	ErrJSONMarshal = stderrors.New("purser: json marshal failed")

	// ErrNilRetrieveDst means wrappers/json.Retrieve was called with a nil dst.
	ErrNilRetrieveDst = stderrors.New("purser: json retrieve destination is nil")

	// ErrJSONUnmarshal means JSON decoding of decrypted bytes into the retrieve destination failed.
	ErrJSONUnmarshal = stderrors.New("purser: json unmarshal failed")

	// ErrInvalidJSON means decrypted plaintext is non-empty and not valid JSON.
	ErrInvalidJSON = stderrors.New("purser: json plaintext is not valid JSON")

	// ErrInvalidUTF8 means a string payload is not valid UTF-8.
	ErrInvalidUTF8 = stderrors.New("purser: plain text is not valid UTF-8")
)

//=============================================================================
// locker/v1 wire metadata, framing, and suite metadata on rows
//=============================================================================

var (
	// ErrNilInnerPointer means [*models.Inner.UnmarshalBinary] was called with a nil receiver.
	ErrNilInnerPointer = stderrors.New("purser/locker/v1: nil Inner receiver")

	// ErrNilEphPubPointer means [*models.EphPub.UnmarshalBinary] was called with a nil receiver.
	ErrNilEphPubPointer = stderrors.New("purser/locker: nil EphPub receiver")

	// ErrMalformedWire means bytes are corrupt, truncated, or not a valid v1 wire layout for the operation.
	ErrMalformedWire = stderrors.New("purser/locker/v1: malformed wire encoding")

	// ErrNilMetaPointer means [*models.Meta.UnmarshalBinary] was called with a nil receiver.
	ErrNilMetaPointer = stderrors.New("purser/locker/v1: nil Meta")

	// ErrMetaKeyIDTooLarge means [models.Meta.KeyID] exceeds the wire limit.
	ErrMetaKeyIDTooLarge = stderrors.New("purser/locker/v1: meta key identifier exceeds limit")

	// ErrMetaNamespaceTooLarge means [models.Meta.Namespace] exceeds the wire limit.
	ErrMetaNamespaceTooLarge = stderrors.New("purser/locker/v1: namespace exceeds limit")

	// ErrNilSealedPointer means [*models.Sealed.UnmarshalBinary] was called with a nil receiver.
	ErrNilSealedPointer = stderrors.New("purser/locker/v1: nil Sealed receiver")

	// ErrBadMagic means the wire blob does not begin with the expected v1 magic bytes.
	ErrBadMagic = stderrors.New("purser/locker/v1: bad magic")

	// ErrUnsupportedVersion means the row format version byte is not supported by this module.
	ErrUnsupportedVersion = stderrors.New("purser/locker/v1: unsupported version")

	// ErrVersionMismatch means the outer format version disagrees with the decoded metadata version.
	ErrVersionMismatch = stderrors.New("purser/locker/v1: unsupported format version")

	// ErrUnknownSuite means the metadata suite id is not a known v1 suite.
	ErrUnknownSuite = stderrors.New("purser/locker/v1: unknown suite")

	// ErrNamespaceMismatch means the row was opened under a namespace that does not match the row metadata.
	ErrNamespaceMismatch = stderrors.New("purser/locker/v1: namespace mismatch")
)

//=============================================================================
// locker/v1 suite ID parse and marshal ([locker/v1/suite])
//=============================================================================

var (
	// ErrNilSuiteID means [*suite.ID.UnmarshalBinary] was called with a nil receiver.
	ErrNilSuiteID = stderrors.New("purser/locker/v1/suite: nil ID receiver")

	// ErrInvalidSuiteWire means decoded suite bytes are not the expected length.
	ErrInvalidSuiteWire = stderrors.New("purser/locker/v1/suite: invalid wire encoding")

	// ErrInvalidSuiteValue means a numeric suite id is not usable.
	ErrInvalidSuiteValue = stderrors.New("purser/locker/v1/suite: invalid suite value")

	// ErrUnknownSuiteName means the string does not name a known suite.
	ErrUnknownSuiteName = stderrors.New("purser/locker/v1/suite: unknown suite name")

	// ErrInvalidSuiteInput means the argument type is not supported for suite.Parse.
	ErrInvalidSuiteInput = stderrors.New("purser/locker/v1/suite: invalid input type")
)
