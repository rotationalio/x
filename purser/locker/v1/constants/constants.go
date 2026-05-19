// Package constants holds normative wire sizes, magic, and version bytes for vault v1 sealed rows.
package constants

const (
	// PackageVersion is the supported v1 package and wire format version.
	PackageVersion uint8 = 1

	// Magic is the four-byte preamble for sealed rows (wire normative).
	Magic = "VLT1"

	// MaxNamespaceBytes is the maximum number of bytes allowed for a namespace identifier on the wire.
	MaxNamespaceBytes = 255

	// MaxKeyIDBytes is the maximum number of bytes allowed for a key identifier on the wire.
	MaxKeyIDBytes = 32

	// InnerNonceBytes is the inner AES-GCM nonce size in bytes.
	InnerNonceBytes = 12

	// WrapNonceBytes is the DEK-wrap AES-GCM nonce size in bytes.
	WrapNonceBytes = 12

	// X25519PubBytes is the length in bytes of an X25519 public key on the wire.
	X25519PubBytes = 32

	// DEKBytes is the per-row data encryption key length in bytes.
	DEKBytes = 32

	// WrapKeyBytes is the HKDF-derived AES-256 wrap key length in bytes.
	WrapKeyBytes = 32

	// GCMTagBytes is the AES-GCM authentication tag size in bytes.
	GCMTagBytes = 16

	// DekEnvelopeBytes is the fixed on-wire size for the initial v1 suite (32+12+32+16).
	DekEnvelopeBytes = X25519PubBytes + WrapNonceBytes + DEKBytes + GCMTagBytes

	// MaxMetaWireBytes is the largest possible v1 Meta encoding (bounded decode).
	MaxMetaWireBytes = 1 + 1 + 1 + MaxKeyIDBytes + 1 + MaxNamespaceBytes
)
