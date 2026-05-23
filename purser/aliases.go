package purser

// Type aliases and forwarded constructors for single-import clients.
//
// Import go.rtnl.ai/x/purser for row operations and the symbols defined here. Locker
// construction and routing helpers that are not aliased live in keyring/registry and
// keyring/kdf; errors live in purser/errors.

import (
	"go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/hold/identifier"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/keyring"
	"go.rtnl.ai/x/purser/keyring/kdf"
	"go.rtnl.ai/x/purser/keyring/keyspec"
	"go.rtnl.ai/x/purser/keyring/memring"
	"go.rtnl.ai/x/purser/locker"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
)

//=============================================================================
// Editions
//=============================================================================

// EditionV1 is the production locker edition string ("v1") for keyring/registry and
// [NewPassword]. X25519 long-term keys with per-row ephemeral X25519 keypairs.
const EditionV1 = constv1.Edition

//=============================================================================
// Locker
//=============================================================================

// Locker types.
type (
	// Locker exposes secret sealing, opening, and key id/metadata reporting.
	// Composed of [Sealer], [Labeler], and [Keyer].
	Locker = locker.Locker

	// Sealer supports encrypting and decrypting wire-format blobs.
	Sealer = locker.Sealer

	// Labeler exposes the locker’s stable edition string for wire compatibility.
	Labeler = locker.Labeler

	// Keyer exposes a binary key identifier for lock/unlock/route operations.
	Keyer = locker.Keyer
)

//=============================================================================
// Keyring
//=============================================================================

// Keyring types.
type (
	// Keyring manages locker registration, namespace binding, and routing.
	// Composed of [Registrator], [Namespacer], and [Router].
	Keyring = keyring.Keyring

	// Registrator manages locker registration and default assignment.
	Registrator = keyring.Registrator

	// Namespacer maps namespaces to lockers.
	Namespacer = keyring.Namespacer

	// Router enables locker selection and routing by ciphertext.
	Router = keyring.Router

	// KeySpec is a parsed key and its parameters, suitable for constructing a
	// locker. Use [NewPassword], [NewSeed], [NewPKCS8], or [NewPrivateKey] to
	// construct a KeySpec.
	KeySpec = keyspec.KeySpec

	// Memring is an in-memory implementation of [Keyring]. Useful for testing
	// and non-persistent storage. Use [NewMemring] to construct a Memring.
	Memring = memring.Memring
)

// KeySpec constructors.
var (
	// NewPassword derives a KeySpec from a password with optional parameters.
	NewPassword = keyspec.NewPassword

	// NewSeed builds a KeySpec from raw key material (as from a CSPRNG or KDF).
	NewSeed = keyspec.NewSeed

	// NewPKCS8 parses a PKCS#8-formatted private key into a KeySpec.
	NewPKCS8 = keyspec.NewPKCS8

	// NewPrivateKey parses a private key (any supported format) into a KeySpec.
	NewPrivateKey = keyspec.NewPrivateKey
)

//=============================================================================
// Hold
//=============================================================================

// Hold types.
type (
	// Hold provides persistent storage and retrieval of wire-format secret blobs.
	Hold = hold.Hold

	// MemHold is the in-memory [Hold] implementation. Use [NewMemHold] to construct one.
	MemHold = hold.MemHold

	// Identifier provides a stable, canonical key for storage and retrieval.
	Identifier = identifier.Identifier

	// HexIdentifier implements [Identifier] using 16-byte random ids encoded as hex (32 chars).
	HexIdentifier = hexid.Identifier
)

// NewMemring returns an empty in-memory keyring.
func NewMemring() *Memring {
	return memring.New()
}

// NewMemHold returns a new in-memory Hold identified by the given Identifier.
func NewMemHold(id Identifier) (*MemHold, error) {
	return hold.NewMemHold(id)
}

//=============================================================================
// KDF
//=============================================================================

// Params specifies memory/cpu parameters for password-based key derivation (KDF).
type Params = kdf.Params

// KDF helpers.
var (
	// RandSalt returns a new random salt for Argon2id (16 bytes).
	RandSalt = kdf.RandSalt

	// DefaultParams is the RFC 9106 first recommended Argon2id profile.
	DefaultParams = kdf.DefaultParams

	// MemoryConstrainedParams is the RFC 9106 second recommended Argon2id profile.
	MemoryConstrainedParams = kdf.MemoryConstrainedParams
)
