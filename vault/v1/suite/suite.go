/*
Package suite defines numeric suite identifiers and stable wire names for vault v1 metadata.
*/
package suite

import (
	"strconv"

	verrors "go.rtnl.ai/x/vault/v1/errors"
)

// ID selects the full crypto recipe (wrap + KDF context + inner AEAD).
type ID uint8

const (
	Unknown ID = iota
	X25519HKDFSHA256AES256GCM
)

// Names maps [ID] to a stable wire/debug name (index must match ID).
var Names = []string{
	"unknown",
	"x25519_hkdf_sha256_aes256_gcm",
}

// Valid reports whether id names a supported v1 suite.
func (id ID) Valid() bool {
	switch id {
	case X25519HKDFSHA256AES256GCM:
		return true
	default:
		return false
	}
}

// String returns a stable name for id.
func (id ID) String() string {
	if int(id) >= 0 && int(id) < len(Names) {
		return Names[id]
	}
	return Names[Unknown]
}

// Parse coerces v into an [ID].
func Parse(v any) (ID, error) {
	switch t := v.(type) {
	case ID:
		return t, nil
	case uint8:
		return ID(t), nil
	case int:
		if t < 0 || t > 255 {
			return Unknown, verrors.ErrInvalidSuiteValue
		}
		return ID(t), nil
	case int64:
		if t < 0 || t > 255 {
			return Unknown, verrors.ErrInvalidSuiteValue
		}
		return ID(t), nil
	case string:
		for i, name := range Names {
			if name == t {
				return ID(i), nil
			}
		}
		n, err := strconv.ParseUint(t, 10, 8)
		if err != nil {
			return Unknown, verrors.ErrUnknownSuiteName
		}
		return ID(n), nil
	default:
		return Unknown, verrors.ErrInvalidSuiteInput
	}
}

// MarshalBinary encodes id as a single byte.
func (id ID) MarshalBinary() ([]byte, error) {
	return []byte{byte(id)}, nil
}

// UnmarshalBinary decodes a single-byte suite selector.
func (id *ID) UnmarshalBinary(data []byte) error {
	if id == nil {
		return verrors.ErrNilSuiteID
	}
	if len(data) != 1 {
		return verrors.ErrInvalidSuiteWire
	}
	*id = ID(data[0])
	return nil
}
