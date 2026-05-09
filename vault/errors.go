package vault

// Sentinel errors for [errors.Is].

import "errors"

var (
	// ---- Key/cipher errors ----

	// ErrAEADSetup means [New] could not construct the AEAD after key length checks.
	ErrAEADSetup = errors.New("vault: AEAD setup failed")

	// ErrInvalidAESKeySize means newAEAD received a key not of length 16, 24, or 32.
	ErrInvalidAESKeySize = errors.New("vault: invalid AES key size for newAEAD")

	// ErrInvalidKeyLength means New received a key that is not 32 bytes.
	ErrInvalidKeyLength = errors.New("vault: invalid AES key length for Vault (want 32 bytes for AES-256)")

	// ---- Encryption/decryption errors ----

	// ErrCiphertextTooShort means stored bytes are shorter than the nonce prefix.
	ErrCiphertextTooShort = errors.New("vault: ciphertext too short")

	// ErrDecrypt means decrypt or authenticate failed (wrong AAD, corrupt blob, etc.).
	ErrDecrypt = errors.New("vault: decrypt failed")

	// ErrSealFailed means sealing failed (e.g. reading entropy for the nonce).
	ErrSealFailed = errors.New("vault: seal failed")

	// ---- Vault/identifier/storage errors ----

	// ErrCASFailed means a compare-and-swap lost due to concurrent modification.
	ErrCASFailed = errors.New("vault: secret was modified concurrently; compare-and-swap lost")

	// ErrDuplicateKey means [Storage.Create] saw an existing (namespace, id).
	ErrDuplicateKey = errors.New("vault: duplicate key")

	// ErrInvalidIdentifier means [Identifier.New] or [Identifier.Parse] rejected input.
	ErrInvalidIdentifier = errors.New("vault: invalid identifier")

	// ErrInvalidNewArgs means New was called with nil storage or identifier.
	ErrInvalidNewArgs = errors.New("vault: storage and identifier are required")

	// ErrMoveNamespaceIncomplete means destination row was created but source delete failed.
	ErrMoveNamespaceIncomplete = errors.New("vault: MoveNamespace incomplete: destination created but source not deleted")

	// ErrNotFound means no secret exists at (namespace, id).
	ErrNotFound = errors.New("vault: secret not found")

	// ErrStorage means a [Storage] operation failed (often wrapping the backend error).
	ErrStorage = errors.New("vault: storage operation failed")

	// ErrWrongCurrent means AtomicUpdate expected plaintext did not match storage.
	ErrWrongCurrent = errors.New("vault: stored secret does not match expected plaintext")
)
