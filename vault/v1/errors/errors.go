// Package errors defines stable sentinel errors for vault v1, [models], [gcm],
// [suite], and [keys]. Use [errors.Is] against these variables for classification.
package errors

import stderrors "errors"

//=============================================================================
// Vault construction and receiver
//=============================================================================

var (
	// ErrNilPrivateKey means the private key is nil or missing where one is required.
	ErrNilPrivateKey = stderrors.New("vault/v1: PrivateKey is required")

	// ErrInvalidWrappingKey means the long-term key is not usable with this module (for example, not X25519).
	ErrInvalidWrappingKey = stderrors.New("vault/v1: wrapping key not supported")

	// ErrInvalidNewArgs means [v1.New] was called without required dependencies (storage or identifier).
	ErrInvalidNewArgs = stderrors.New("vault/v1: storage and identifier are required")

	// ErrNilVault means a method was called on a nil [v1.Vault] receiver (including a nil *vaulttest.TestVault).
	ErrNilVault = stderrors.New("vault/v1: vault is nil")
)

//=============================================================================
// Cryptography (inner / wrap AEAD and helpers)
//=============================================================================

var (
	// ErrInvalidAEADKey means key material was rejected for AES-GCM construction (wrong length or cipher failure).
	ErrInvalidAEADKey = stderrors.New("vault/v1: invalid AEAD key material")

	// ErrCiphertextTooShort means input bytes are shorter than required for the nonce prefix or ciphertext layout.
	ErrCiphertextTooShort = stderrors.New("vault/v1: ciphertext too short")

	// ErrDecrypt means decryption or GCM authentication failed (wrong AAD, corrupt ciphertext, wrong key, etc.).
	ErrDecrypt = stderrors.New("vault/v1: decrypt failed")

	// ErrSealFailed means sealing failed, for example when reading random bytes for a nonce.
	ErrSealFailed = stderrors.New("vault/v1: seal failed")

	// ErrNilAEAD means a crypto helper received a nil AEAD implementation.
	ErrNilAEAD = stderrors.New("vault/v1: nil AEAD")

	// ErrMalformedParameters means a gcm helper received invalid lengths, nonce size, or AEAD output layout.
	ErrMalformedParameters = stderrors.New("vault/v1: malformed parameters")
)

//=============================================================================
// Identifiers and storage
//=============================================================================

var (
	// ErrInvalidHexID means the string is not a valid 32-character hex encoding of 16 bytes for hex row ids.
	ErrInvalidHexID = stderrors.New("vault/v1: invalid hex identifier")

	// ErrInvalidIdentifier means the identifier implementation rejected the id (format or policy).
	ErrInvalidIdentifier = stderrors.New("vault/v1: invalid identifier")

	// ErrDuplicateKey means a storage create or write conflicted with an existing row id in that namespace.
	ErrDuplicateKey = stderrors.New("vault/v1: duplicate key")

	// ErrNotFound means no sealed row exists for the requested namespace and id.
	ErrNotFound = stderrors.New("vault/v1: secret not found")

	// ErrCASFailed means [v1.Vault.CompareAndSwap] lost the race: stored plaintext did not match currentPlain.
	ErrCASFailed = stderrors.New("vault/v1: secret was modified concurrently; compare-and-swap lost")

	// ErrMoveNamespaceIncomplete means [v1.Vault.MoveNamespace] could not finish moving every matching row.
	ErrMoveNamespaceIncomplete = stderrors.New("vault/v1: namespace relocation incomplete")

	// ErrStorage means the underlying [storage.Storage] implementation returned a failure unrelated to vault logic.
	ErrStorage = stderrors.New("vault/v1: storage operation failed")

	// ErrWrongCurrent means [v1.Vault.CompareAndSwap] failed because stored plaintext did not equal expected currentPlain.
	ErrWrongCurrent = stderrors.New("vault/v1: stored secret does not match expected plaintext")
)

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

//=============================================================================
// Keys ([keys])
//=============================================================================

var (
	// ErrInvalidOut means the output buffer length is not valid for the requested operation.
	ErrInvalidOut = stderrors.New("vault/v1/keys: invalid output length")

	// ErrInvalidSeed means the seed length is not valid for [keys.FromSeed].
	ErrInvalidSeed = stderrors.New("vault/v1/keys: invalid seed")

	// ErrInvalidSalt means the salt length is not valid for [keys.Derive].
	ErrInvalidSalt = stderrors.New("vault/v1/keys: invalid salt")

	// ErrNilPassword means [keys.Derive] received a nil password slice.
	ErrNilPassword = stderrors.New("vault/v1/keys: nil password")

	// ErrRandSalt means reading random bytes for a new salt failed.
	ErrRandSalt = stderrors.New("vault/v1/keys: failed to read random salt")
)

//=============================================================================
// JSON and UTF-8 wrappers ([jsonvault], [stringvault])
//=============================================================================

var (
	// ErrJSONMarshal means JSON encoding of a store payload failed before encryption.
	ErrJSONMarshal = stderrors.New("vault/v1: json marshal failed")

	// ErrNilRetrieveDst means [jsonvault.Vault.Retrieve] was called with a nil dst (see [encoding/json.Unmarshal] for valid dst shapes).
	ErrNilRetrieveDst = stderrors.New("vault/v1: json retrieve destination is nil")

	// ErrJSONUnmarshal means JSON decoding of decrypted bytes into the retrieve destination failed.
	ErrJSONUnmarshal = stderrors.New("vault/v1: json unmarshal failed")

	// ErrInvalidJSON means decrypted plaintext is non-empty and not valid JSON per [encoding/json.Valid].
	ErrInvalidJSON = stderrors.New("vault/v1: json plaintext is not valid JSON")

	// ErrInvalidUTF8 means a string payload is not valid UTF-8 (store input or decrypted bytes).
	ErrInvalidUTF8 = stderrors.New("vault/v1: plain text is not valid UTF-8")
)
