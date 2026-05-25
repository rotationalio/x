/*
Package nulllocker provides a null-encryption locker.Locker for testing multi-version
keyring dispatch, golden-test fixture generation, and wire-format parsing without real
cryptographic overhead.

This package lives under purser/internal/ so only purser subtree packages can import it.
Every constructor requires a non-nil testing.TB to prevent accidental production use.

Each nullLocker is parameterized by a Variant byte (0x00..0x02) that changes the wire
prefix length and key-id format. This lets tests register several null lockers in a
single keyring and verify that ParseKeyID routes ciphertext to the correct locker.

Wire format (all variants):

	[magic 'N' 'U' 'L' 'L'] [variant byte] [key-id-length byte] [key-id bytes]
	[namespace-length uint16 BE] [namespace bytes] [plaintext]
	[sha256 checksum (32 bytes)]

The trailing 32-byte SHA-256 is computed over every byte preceding it and is verified by
Open before any field parsing, providing tamper detection for golden-file fixtures.

Different variants use different key-id sizes (4, 8, 16 bytes) so ParseKeyID must handle
variable-length extraction and the keyring can distinguish between lockers.
*/
package nulllocker

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/locker"
)

// checksumBytes is the trailing SHA-256 checksum length for tamper detection.
const checksumBytes = 32

// magic is the four-byte preamble identifying null-locker wire blobs.
var magic = [4]byte{'N', 'U', 'L', 'L'}

// preambleBytes is the fixed portion before variable-length fields: magic(4) + variant(1) + keyIDLen(1).
const preambleBytes = 6

// Variant selects the key-id size and wire personality of a null locker.
type Variant byte

const (
	// VariantA uses a 4-byte key ID.
	VariantA Variant = 0x00
	// VariantB uses an 8-byte key ID.
	VariantB Variant = 0x01
	// VariantC uses a 16-byte key ID.
	VariantC Variant = 0x02
)

// keyIDSize returns the key-id byte length for a variant.
func (v Variant) keyIDSize() int {
	switch v {
	case VariantA:
		return 4
	case VariantB:
		return 8
	case VariantC:
		return 16
	default:
		return 4
	}
}

// nullLocker implements locker.Locker with null encryption.
type nullLocker struct {
	variant Variant
	keyID   []byte
}

// Ensure nullLocker satisfies locker.Locker at compile time.
var _ locker.Locker = (*nullLocker)(nil)

// New constructs a null-encryption locker for the given variant with a deterministic key ID
// derived from seed bytes. The seed is truncated or repeated to fill the variant's key-id
// size. A non-nil testing.TB is required to prevent production use.
func New(tb testing.TB, variant Variant, seed []byte) (locker.Locker, error) {
	if tb == nil {
		return nil, perrors.ErrInvalidNewArgs
	}
	tb.Helper()

	if seed == nil {
		return nil, perrors.ErrInvalidSeed
	}

	size := variant.keyIDSize()
	kid := make([]byte, size)
	for i := range kid {
		kid[i] = seed[i%len(seed)]
	}

	return &nullLocker{variant: variant, keyID: kid}, nil
}

// FromSeed maps arbitrary seed bytes to a null locker key ID for the given variant.
// It truncates or repeats the seed to match the variant's key-id size.
func FromSeed(tb testing.TB, variant Variant, seed []byte) (locker.Locker, error) {
	return New(tb, variant, seed)
}

// Version returns 0 (null locker is not a purser wire edition).
func (l *nullLocker) Version() uint8 { return 0 }

// Edition returns "null".
func (l *nullLocker) Edition() string { return "null" }

// Recipe returns "null".
func (l *nullLocker) Recipe() string { return "null" }

// Context returns empty (null locker does not derive keys).
func (l *nullLocker) Context() string { return "" }

// KeyID returns a defensive copy of the locker's key identifier.
func (l *nullLocker) KeyID() []byte {
	if l == nil {
		return nil
	}
	return append([]byte(nil), l.keyID...)
}

// Seal produces a null-encrypted wire blob: the plaintext is stored verbatim after
// structured metadata so ParseKeyID can extract the key ID without decryption. A trailing
// SHA-256 checksum enables tamper detection in golden tests.
func (l *nullLocker) Seal(namespace string, plaintext []byte) ([]byte, error) {
	if l == nil {
		return nil, perrors.ErrSealFailed
	}

	ns := []byte(namespace)
	kidLen := len(l.keyID)

	// Total wire: magic(4) + variant(1) + kidLen(1) + keyID + nsLen(2) + ns + plaintext + sha256(32).
	total := preambleBytes + kidLen + 2 + len(ns) + len(plaintext) + checksumBytes
	wire := make([]byte, total)

	off := 0

	// Write magic preamble.
	copy(wire[off:off+4], magic[:])
	off += 4

	// Variant byte determines how parsers interpret the remaining fields.
	wire[off] = byte(l.variant)
	off++

	// Key-id length prefix followed by key-id bytes.
	wire[off] = byte(kidLen)
	off++
	copy(wire[off:off+kidLen], l.keyID)
	off += kidLen

	// Namespace as big-endian uint16 length followed by UTF-8 bytes.
	binary.BigEndian.PutUint16(wire[off:off+2], uint16(len(ns)))
	off += 2
	copy(wire[off:off+len(ns)], ns)
	off += len(ns)

	// Plaintext stored verbatim (null encryption).
	copy(wire[off:off+len(plaintext)], plaintext)
	off += len(plaintext)

	// Trailing SHA-256 over everything before the checksum for tamper detection.
	sum := sha256.Sum256(wire[:off])
	copy(wire[off:], sum[:])
	return wire, nil
}

// Open parses the wire blob, verifies the checksum and namespace, and returns the plaintext.
func (l *nullLocker) Open(namespace string, ciphertext []byte) ([]byte, error) {
	if l == nil {
		return nil, perrors.ErrDecrypt
	}

	kid, ns, plain, err := parseWire(ciphertext)
	if err != nil {
		return nil, err
	}

	// Verify the key ID belongs to this locker.
	if !bytesEqual(kid, l.keyID) {
		return nil, perrors.ErrNoLocker
	}

	// Namespace binding: the stored namespace must match the requested one.
	if ns != namespace {
		return nil, perrors.ErrNamespaceMismatch
	}

	return append([]byte(nil), plain...), nil
}

// ParseKeyID extracts the key identifier from the wire blob without checksum verification
// or full decryption. Only the magic prefix and key-id framing are validated.
func (l *nullLocker) ParseKeyID(ciphertext []byte) ([]byte, error) {
	kid, err := parseKeyIDOnly(ciphertext)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), kid...), nil
}

// parseKeyIDOnly extracts just the key ID from a null-locker wire blob without verifying
// the checksum. This is used by ParseKeyID for fast routing.
func parseKeyIDOnly(data []byte) ([]byte, error) {
	if len(data) < preambleBytes {
		return nil, perrors.ErrMalformedWire
	}

	// Verify the null-locker magic prefix.
	if data[0] != magic[0] || data[1] != magic[1] || data[2] != magic[2] || data[3] != magic[3] {
		return nil, perrors.ErrBadMagic
	}

	off := 5 // skip magic(4) + variant(1)
	kidLen := int(data[off])
	off++
	if off+kidLen > len(data) {
		return nil, perrors.ErrMalformedWire
	}
	return data[off : off+kidLen], nil
}

// parseWire extracts key-id, namespace, and plaintext from a null-locker wire blob.
// The last checksumBytes of the blob are a SHA-256 over everything before them.
func parseWire(data []byte) (keyID []byte, namespace string, plaintext []byte, err error) {
	if len(data) < preambleBytes+checksumBytes {
		return nil, "", nil, perrors.ErrMalformedWire
	}

	// Verify the null-locker magic prefix.
	if data[0] != magic[0] || data[1] != magic[1] || data[2] != magic[2] || data[3] != magic[3] {
		return nil, "", nil, perrors.ErrBadMagic
	}

	// Split trailing checksum and verify integrity before parsing fields.
	body := data[:len(data)-checksumBytes]
	storedSum := data[len(data)-checksumBytes:]
	computed := sha256.Sum256(body)
	if !bytesEqual(computed[:], storedSum) {
		return nil, "", nil, perrors.ErrDecrypt
	}

	off := 5 // skip magic(4) + variant(1)

	// Read key-id length and key-id bytes.
	kidLen := int(body[off])
	off++
	if off+kidLen > len(body) {
		return nil, "", nil, perrors.ErrMalformedWire
	}
	keyID = body[off : off+kidLen]
	off += kidLen

	// Read namespace length (uint16 BE) and namespace bytes.
	if off+2 > len(body) {
		return nil, "", nil, perrors.ErrMalformedWire
	}
	nsLen := int(binary.BigEndian.Uint16(body[off : off+2]))
	off += 2
	if off+nsLen > len(body) {
		return nil, "", nil, perrors.ErrMalformedWire
	}
	namespace = string(body[off : off+nsLen])
	off += nsLen

	// Remaining body bytes (before checksum) are the plaintext.
	plaintext = body[off:]
	return keyID, namespace, plaintext, nil
}

// bytesEqual is a constant-time-irrelevant byte comparison for test key IDs.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
