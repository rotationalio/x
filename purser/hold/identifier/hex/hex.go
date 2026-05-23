package hex

// 16-byte random identifiers encoded as 32-character lowercase hex strings.

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
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Parse accepts exactly 32 hex characters decoding to 16 bytes.
func (Identifier) Parse(id string) error {
	b, err := hex.DecodeString(id)
	if err != nil || len(b) != 16 {
		return perrors.ErrInvalidHexIdentifier
	}
	return nil
}

// MarshalBinary decodes a hex id string to raw bytes (inverse of [Identifier.New] encoding).
func (Identifier) MarshalBinary(id string) ([]byte, error) {
	b, err := hex.DecodeString(id)
	if err != nil || len(b) != 16 {
		return nil, perrors.ErrInvalidHexIdentifier
	}
	return b, nil
}

// UnmarshalBinary hex-encodes arbitrary bytes (caller should pass 16-byte ids for stable keys).
func (Identifier) UnmarshalBinary(src []byte) (string, error) {
	return hex.EncodeToString(src), nil
}
