package vaulttest

// Stdlib-only hex identifiers for vault tests (see [HexIdentifier]).

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// HexIdentifier implements [vault.Identifier] using 16-byte random ids encoded as hex (32 chars).
type HexIdentifier struct{}

// New mints a random 16-byte value encoded as 32 hex characters.
func (HexIdentifier) New() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Parse accepts exactly 32 hex characters decoding to 16 bytes.
func (HexIdentifier) Parse(id string) error {
	b, err := hex.DecodeString(id)
	if err != nil || len(b) != 16 {
		return fmt.Errorf("vaulttest: invalid id")
	}
	return nil
}

// IDFromBytes hex-encodes arbitrary bytes (caller should pass 16-byte ids for stable keys).
func (HexIdentifier) IDFromBytes(src []byte) (string, error) {
	return hex.EncodeToString(src), nil
}

// BytesFromID decodes a hex string from [HexIdentifier.New].
func (HexIdentifier) BytesFromID(id string) ([]byte, error) {
	return hex.DecodeString(id)
}
