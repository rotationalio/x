package identifier

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	verrors "go.rtnl.ai/x/vault/v1/errors"
)

// HexIdentifier implements [Identifier] using 16-byte random ids encoded as hex (32 chars).
type HexIdentifier struct{}

// HexIdentifier implements [Identifier].
var _ Identifier = HexIdentifier{}

// New mints a random 16-byte value encoded as 32 hex characters.
func (HexIdentifier) New() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Parse accepts exactly 32 hex characters decoding to 16 bytes.
func (HexIdentifier) Parse(id string) error {
	b, err := hex.DecodeString(id)
	if err != nil || len(b) != 16 {
		return verrors.ErrInvalidHexID
	}
	return nil
}

// MarshalBinary decodes a hex id string to raw bytes (inverse of [HexIdentifier.New] encoding).
func (HexIdentifier) MarshalBinary(id string) ([]byte, error) {
	b, err := hex.DecodeString(id)
	if err != nil || len(b) != 16 {
		return nil, verrors.ErrInvalidHexID
	}
	return b, nil
}

// UnmarshalBinary hex-encodes arbitrary bytes (caller should pass 16-byte ids for stable keys).
func (HexIdentifier) UnmarshalBinary(src []byte) (string, error) {
	return hex.EncodeToString(src), nil
}
