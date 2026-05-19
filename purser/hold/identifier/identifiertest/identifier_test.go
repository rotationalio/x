package identifiertest

// Negative conformance tests for the Identifier contract (see [identifier.Identifier]).
//
// The helpers in identifier.go (CheckIdentifier…, IdentifierConforms) encode
// contracts that real identifiers such as [hexid.Identifier] must satisfy. Each subtest
// here wires a deliberately broken fake into one of those checks and asserts the check returns a
// non-nil error—proving the check would fail a non-conforming implementation.

import (
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/hold/identifier"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
)

//=============================================================================
// Tests: negative identifier conformance
//=============================================================================

// TestIdentifierConformance_negative runs table-style subtests; each subtest pairs one broken
// fake with one check that should detect the defect.
func TestIdentifierConformance_negative(t *testing.T) {
	t.Run("new_many_distinct_duplicate_ids", func(t *testing.T) {
		// Real identifiers must mint unique ids; a generator that always returns the same id must
		// fail [checkIdentifierNewManyDistinct] once the duplicate is seen.
		err := checkIdentifierNewManyDistinct(idAlwaysSame{}, 100)
		assert.Error(t, err, "CheckIdentifierNewManyDistinct: expected error for duplicate ids")
	})

	t.Run("new_many_distinct_varying_length", func(t *testing.T) {
		// Hold and storage treat the id string as an opaque key, but Parse and wire framing assume
		// a fixed canonical length; oscillating lengths between mints must fail the distinct check.
		err := checkIdentifierNewManyDistinct(&idOscillatingLen{}, 10)
		assert.Error(t, err, "CheckIdentifierNewManyDistinct: expected error for varying id lengths")
	})

	t.Run("parse_rejects_wrong_length", func(t *testing.T) {
		// Parse must reject ids whose length differs from the canonical form; a permissive Parse
		// that accepts any string breaks [checkIdentifierParseRejectsWrongLength].
		err := checkIdentifierParseRejectsWrongLength(idParsePermissive{})
		assert.Error(t, err, "CheckIdentifierParseRejectsWrongLength: expected error when Parse accepts wrong length")
	})

	t.Run("marshal_binary_roundtrip", func(t *testing.T) {
		// MarshalBinary/UnmarshalBinary must be lossless for the canonical id string; truncating on
		// unmarshal must fail [checkIdentifierMarshalBinaryRoundtrip].
		err := checkIdentifierMarshalBinaryRoundtrip(idTruncateUnmarshal{})
		assert.Error(t, err, "CheckIdentifierMarshalBinaryRoundtrip: expected error for lossy UnmarshalBinary")
	})

	t.Run("new_returns_empty", func(t *testing.T) {
		// New must return a non-empty id; an implementation that always returns "" must fail.
		err := checkIdentifierNewNonEmpty(idNewEmpty{})
		assert.Error(t, err, "CheckIdentifierNewNonEmpty: expected error for empty id")
	})

	t.Run("marshal_binary_returns_empty", func(t *testing.T) {
		// MarshalBinary must return non-empty bytes; an implementation returning nil fails.
		err := checkIdentifierMarshalBinaryNonEmpty(idMarshalEmpty{})
		assert.Error(t, err, "CheckIdentifierMarshalBinaryNonEmpty: expected error for empty marshal")
	})

	t.Run("parse_rejects_valid", func(t *testing.T) {
		// Parse must accept ids that New produces; an implementation rejecting everything fails.
		err := checkIdentifierParseAcceptsMinted(idParseRejectsAll{})
		assert.Error(t, err, "CheckIdentifierParseAcceptsMinted: expected error for rejecting Parse")
	})
}

//=============================================================================
// Broken identifier fakes (used only by tests above)
//=============================================================================

// idAlwaysSame implements [identifier.Identifier] but returns the same id on every New call, violating
// uniqueness required by [checkIdentifierNewManyDistinct]. Parse and binary marshal
// delegate to [hexid.Identifier] so only New is defective.
type idAlwaysSame struct{}

var _ identifier.Identifier = idAlwaysSame{}

func (idAlwaysSame) New() (string, error) { return "0123456789abcdef0123456789abcdef", nil }

func (idAlwaysSame) Parse(id string) error { return hexid.Identifier{}.Parse(id) }

func (idAlwaysSame) MarshalBinary(id string) ([]byte, error) {
	return hexid.Identifier{}.MarshalBinary(id)
}

func (idAlwaysSame) UnmarshalBinary(b []byte) (string, error) {
	return hexid.Identifier{}.UnmarshalBinary(b)
}

// idOscillatingLen alternates between two different string lengths on each New call, violating
// the stable-length requirement enforced by [checkIdentifierNewManyDistinct].
type idOscillatingLen struct{ n int }

func (g *idOscillatingLen) New() (string, error) {
	g.n++
	if g.n%2 == 1 {
		return "aaa", nil
	}
	return "bbbbb", nil
}

func (g *idOscillatingLen) Parse(id string) error {
	if len(id) != 3 && len(id) != 5 {
		return fmt.Errorf("bad length")
	}
	return nil
}

func (g *idOscillatingLen) MarshalBinary(id string) ([]byte, error) { return []byte(id), nil }

func (g *idOscillatingLen) UnmarshalBinary(b []byte) (string, error) { return string(b), nil }

// idParsePermissive embeds [hexid.Identifier] for New/Marshal/Unmarshal but overrides
// Parse to accept any string, defeating length validation.
type idParsePermissive struct{ hexid.Identifier }

func (idParsePermissive) Parse(string) error { return nil }

// idTruncateUnmarshal delegates New/Parse/Marshal to [hexid.Identifier] but drops the last
// byte on UnmarshalBinary, breaking the round-trip contract checked by
// [checkIdentifierMarshalBinaryRoundtrip].
type idTruncateUnmarshal struct{}

func (idTruncateUnmarshal) New() (string, error) { return hexid.Identifier{}.New() }

func (idTruncateUnmarshal) Parse(id string) error { return hexid.Identifier{}.Parse(id) }

func (idTruncateUnmarshal) MarshalBinary(id string) ([]byte, error) {
	return hexid.Identifier{}.MarshalBinary(id)
}

func (idTruncateUnmarshal) UnmarshalBinary(b []byte) (string, error) {
	if len(b) < 1 {
		return "", fmt.Errorf("empty")
	}
	return string(b[:len(b)-1]), nil
}

// idNewEmpty always returns an empty string from New, violating the non-empty contract.
type idNewEmpty struct{ hexid.Identifier }

func (idNewEmpty) New() (string, error) { return "", nil }

// idMarshalEmpty returns empty bytes from MarshalBinary, violating the non-empty contract.
type idMarshalEmpty struct{ hexid.Identifier }

func (idMarshalEmpty) MarshalBinary(string) ([]byte, error) { return nil, nil }

// idParseRejectsAll rejects every id from Parse, violating the "Parse accepts New output" contract.
type idParseRejectsAll struct{ hexid.Identifier }

func (idParseRejectsAll) Parse(string) error { return fmt.Errorf("always rejects") }
