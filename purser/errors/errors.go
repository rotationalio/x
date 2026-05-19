// Package errors defines stable operational and crypto-helper sentinel errors shared across vault
// versions. Classify with errors.Is from the standard library "errors" package. Wire-specific v1 errors live in
// [go.rtnl.ai/x/vault/v1/errors].
package errors

import stderrors "errors"

//=============================================================================
// Vault construction and receiver
//=============================================================================

var (
	// ErrNilPrivateKey means the private key is nil or missing where one is required.
	ErrNilPrivateKey = stderrors.New("vault: PrivateKey is required")

	// ErrInvalidWrappingKey means the long-term key is not usable with this module (for example, not X25519).
	ErrInvalidWrappingKey = stderrors.New("vault: wrapping key not supported")

	// ErrInvalidNewArgs means the vault constructor was called without required dependencies (storage or identifier).
	ErrInvalidNewArgs = stderrors.New("vault: storage and identifier are required")

	// ErrNilVault means a method was called on a nil Vault receiver (including a nil *vaulttest.TestVault).
	ErrNilVault = stderrors.New("vault: vault is nil")
)

//=============================================================================
// Cryptography (inner / wrap AEAD and helpers)
//=============================================================================

var (
	// ErrInvalidAEADKey means key material was rejected for AES-GCM construction (wrong length or cipher failure).
	ErrInvalidAEADKey = stderrors.New("vault: invalid AEAD key material")

	// ErrCiphertextTooShort means input bytes are shorter than required for the nonce prefix or ciphertext layout.
	ErrCiphertextTooShort = stderrors.New("vault: ciphertext too short")

	// ErrDecrypt means decryption or GCM authentication failed (wrong AAD, corrupt ciphertext, wrong key, etc.).
	ErrDecrypt = stderrors.New("vault: decrypt failed")

	// ErrSealFailed means sealing failed, for example when reading random bytes for a nonce.
	ErrSealFailed = stderrors.New("vault: seal failed")

	// ErrNilAEAD means a crypto helper received a nil AEAD implementation.
	ErrNilAEAD = stderrors.New("vault: nil AEAD")

	// ErrMalformedParameters means a gcm helper received invalid lengths, nonce size, or AEAD output layout.
	ErrMalformedParameters = stderrors.New("vault: malformed parameters")
)

//=============================================================================
// Identifiers and storage
//=============================================================================

var (
	// ErrInvalidHexID means the string is not a valid 32-character hex encoding of 16 bytes for hex row ids.
	ErrInvalidHexID = stderrors.New("vault: invalid hex identifier")

	// ErrInvalidIdentifier means the identifier implementation rejected the id (format or policy).
	ErrInvalidIdentifier = stderrors.New("vault: invalid identifier")

	// ErrDuplicateKey means a storage create or write conflicted with an existing row id in that namespace.
	ErrDuplicateKey = stderrors.New("vault: duplicate key")

	// ErrNotFound means no sealed row exists for the requested namespace and id.
	ErrNotFound = stderrors.New("vault: secret not found")

	// ErrCASFailed means CompareAndSwap lost the race: stored plaintext did not match currentPlain.
	ErrCASFailed = stderrors.New("vault: secret was modified concurrently; compare-and-swap lost")

	// ErrMoveNamespaceIncomplete means MoveNamespace could not finish moving every matching row.
	ErrMoveNamespaceIncomplete = stderrors.New("vault: namespace relocation incomplete")

	// ErrStorage means the underlying storage.Storage implementation returned a failure unrelated to vault logic.
	ErrStorage = stderrors.New("vault: storage operation failed")

	// ErrWrongCurrent means CompareAndSwap failed because stored plaintext did not equal expected currentPlain.
	ErrWrongCurrent = stderrors.New("vault: stored secret does not match expected plaintext")
)

//=============================================================================
// Keys ([keys] at go.rtnl.ai/x/vault/keys)
//=============================================================================

var (
	// ErrInvalidOut means the output buffer length is not valid for the requested operation.
	ErrInvalidOut = stderrors.New("vault/keys: invalid output length")

	// ErrInvalidSeed means the seed length is not valid for [keys.FromSeed].
	ErrInvalidSeed = stderrors.New("vault/keys: invalid seed")

	// ErrInvalidSalt means the salt length is not valid for [keys.Derive].
	ErrInvalidSalt = stderrors.New("vault/keys: invalid salt")

	// ErrNilPassword means [keys.Derive] received a nil password slice.
	ErrNilPassword = stderrors.New("vault/keys: nil password")

	// ErrRandSalt means reading random bytes for a new salt failed.
	ErrRandSalt = stderrors.New("vault/keys: failed to read random salt")
)

//=============================================================================
// JSON and UTF-8 wrappers ([jsonvault], [stringvault])
//=============================================================================

var (
	// ErrJSONMarshal means JSON encoding of a store payload failed before encryption.
	ErrJSONMarshal = stderrors.New("vault: json marshal failed")

	// ErrNilRetrieveDst means [jsonvault.Vault.Retrieve] was called with a nil dst (see [encoding/json.Unmarshal] for valid dst shapes).
	ErrNilRetrieveDst = stderrors.New("vault: json retrieve destination is nil")

	// ErrJSONUnmarshal means JSON decoding of decrypted bytes into the retrieve destination failed.
	ErrJSONUnmarshal = stderrors.New("vault: json unmarshal failed")

	// ErrInvalidJSON means decrypted plaintext is non-empty and not valid JSON per [encoding/json.Valid].
	ErrInvalidJSON = stderrors.New("vault: json plaintext is not valid JSON")

	// ErrInvalidUTF8 means a string payload is not valid UTF-8 (store input or decrypted bytes).
	ErrInvalidUTF8 = stderrors.New("vault: plain text is not valid UTF-8")
)
