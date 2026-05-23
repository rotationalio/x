// Package constants holds normative wire sizes, magic, and edition metadata for locker v1.
package constants

const (
	// Edition is the app-facing locker id passed to registry.FromSeed and FromPassword.
	// Single-import clients use [purser.EditionV1] as the edition argument.
	Edition = "v1"

	// Version is the wire format byte (sealed preamble and meta).
	Version uint8 = 1

	// Recipe is the short crypto recipe name for this edition.
	Recipe = "x25519_hkdf_sha256_aes256_gcm"

	// Context is the KDF/context string for row key derivation (HKDF info in v1).
	Context = "purser/v1/x25519_hkdf_sha256_aes256_gcm"

	// MaxNamespaceBytes is the maximum number of bytes allowed for a namespace identifier on the wire.
	MaxNamespaceBytes = 255

	// MaxKeyIDBytes is the maximum number of bytes allowed for a key identifier on the wire.
	MaxKeyIDBytes = 32

	// InnerNonceBytes is the inner AES-GCM nonce size in bytes.
	InnerNonceBytes = 12

	// X25519PubBytes is the length in bytes of an X25519 public key on the wire.
	X25519PubBytes = 32

	// EphPubBytes is the fixed on-wire size of the per-row ephemeral X25519 public key.
	EphPubBytes = X25519PubBytes

	// DataKeyBytes is the derived AES-256 data key length in bytes.
	DataKeyBytes = 32

	// GCMTagBytes is the AES-GCM authentication tag size in bytes.
	GCMTagBytes = 16

	// MaxMetaWireBytes is the largest possible v1 Meta encoding (bounded decode).
	MaxMetaWireBytes = 1 + 1 + MaxKeyIDBytes + 1 + MaxNamespaceBytes
)
