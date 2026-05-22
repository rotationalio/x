// Package constants holds normative wire sizes, magic, and version bytes for purser locker v1 rows.
package constants

const (
	// PackageVersion is the supported v1 package and wire format version.
	PackageVersion uint8 = 1

	// Magic is the four-byte preamble for sealed rows (wire normative).
	Magic = "ARR1"

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

	// DataKeyBytes is the HKDF-derived AES-256 data key length in bytes.
	DataKeyBytes = 32

	// GCMTagBytes is the AES-GCM authentication tag size in bytes.
	GCMTagBytes = 16

	// MaxMetaWireBytes is the largest possible v1 Meta encoding (bounded decode).
	MaxMetaWireBytes = 1 + 1 + 1 + MaxKeyIDBytes + 1 + MaxNamespaceBytes
)
