// Package hex implements 16-byte random identifiers as 32-character lowercase hex strings.
package hex

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold/identifier"
)

// Identifier implements [identifier.Identifier] using 16-byte random ids encoded as hex (32 chars).
// Single-import clients use [purser.HexIdentifier].
type Identifier struct{}

// Identifier implements [identifier.Identifier].
var _ identifier.Identifier = Identifier{}

// New mints a random 16-byte value encoded as 32 hex characters.
func (Identifier) New() (string, error) {
	var (
		b   [16]byte
		out [32]byte
	)
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "", err
	}
	hex.Encode(out[:], b[:])
	return string(out[:]), nil
}

// Parse accepts exactly 32 hex characters decoding to 16 bytes.
func (Identifier) Parse(id string) error {
	if len(id) != 32 || !isHex32(id) {
		return perrors.ErrInvalidHexIdentifier
	}
	return nil
}

// MarshalBinary decodes a hex id string to raw bytes (inverse of [Identifier.New] encoding).
func (Identifier) MarshalBinary(id string) ([]byte, error) {
	var b [16]byte
	if len(id) != 32 || !isHex32(id) {
		return nil, perrors.ErrInvalidHexIdentifier
	}
	hex.Decode(b[:], []byte(id))
	return append([]byte(nil), b[:]...), nil
}

// UnmarshalBinary hex-encodes arbitrary bytes (caller should pass 16-byte ids for stable keys).
func (Identifier) UnmarshalBinary(src []byte) (string, error) {
	if len(src) == 16 {
		var out [32]byte
		hex.Encode(out[:], src)
		return string(out[:]), nil
	}
	return hex.EncodeToString(src), nil
}

// isHex32 reports whether s is exactly 32 ASCII hex digits.
func isHex32(s string) bool {
	for i := range 32 {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}
