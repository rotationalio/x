package vaulttest_test

// Negative conformance tests for the vault Identifier contract (see [identifier.Identifier]).
//
// The helpers in identifier.go (CheckIdentifier…, IdentifierConforms) encode
// contracts that real identifiers such as [identifier.HexIdentifier] must satisfy. Each subtest
// here wires a deliberately broken fake into one of those checks and asserts the check returns a
// non-nil error—proving the check would fail a non-conforming implementation.

import (
	"fmt"
	"testing"

	_ "go.rtnl.ai/x/vault/v1"
	"go.rtnl.ai/x/vault/identifier"
	"go.rtnl.ai/x/vault/vaulttest"
)

//=============================================================================
// Tests: negative identifier conformance
//=============================================================================

// TestIdentifierConformance_negative runs table-style subtests; each subtest pairs one broken
// fake with one exported check that should detect the defect.
func TestIdentifierConformance_negative(t *testing.T) {
	t.Run("new_many_distinct_duplicate_ids", func(t *testing.T) {
		// Real identifiers must mint unique ids; a generator that always returns the same id must
		// fail [vaulttest.CheckIdentifierNewManyDistinct] once the duplicate is seen.
		if err := vaulttest.CheckIdentifierNewManyDistinct(idAlwaysSame{}, 100); err == nil {
			t.Fatal("CheckIdentifierNewManyDistinct: expected error for duplicate ids")
		}
	})

	t.Run("new_many_distinct_varying_length", func(t *testing.T) {
		// Vault and storage treat the id string as an opaque key, but Parse and wire framing assume
		// a fixed canonical length; oscillating lengths between mints must fail the distinct check.
		if err := vaulttest.CheckIdentifierNewManyDistinct(&idOscillatingLen{}, 10); err == nil {
			t.Fatal("CheckIdentifierNewManyDistinct: expected error for varying id lengths")
		}
	})

	t.Run("parse_rejects_wrong_length", func(t *testing.T) {
		// Parse must reject ids whose length differs from the canonical form; a permissive Parse
		// that accepts any string breaks [vaulttest.CheckIdentifierParseRejectsWrongLength].
		if err := vaulttest.CheckIdentifierParseRejectsWrongLength(idParsePermissive{}); err == nil {
			t.Fatal("CheckIdentifierParseRejectsWrongLength: expected error when Parse accepts wrong length")
		}
	})

	t.Run("marshal_binary_roundtrip", func(t *testing.T) {
		// MarshalBinary/UnmarshalBinary must be lossless for the canonical id string; truncating on
		// unmarshal must fail [vaulttest.CheckIdentifierMarshalBinaryRoundtrip].
		if err := vaulttest.CheckIdentifierMarshalBinaryRoundtrip(idTruncateUnmarshal{}); err == nil {
			t.Fatal("CheckIdentifierMarshalBinaryRoundtrip: expected error for lossy UnmarshalBinary")
		}
	})
}

//=============================================================================
// Broken identifier fakes (used only by tests above)
//=============================================================================

// idAlwaysSame implements [identifier.Identifier] but returns the same id on every New call, violating
// uniqueness required by [vaulttest.CheckIdentifierNewManyDistinct]. Parse and binary marshal
// delegate to [identifier.HexIdentifier] so only New is defective.
type idAlwaysSame struct{}

var _ identifier.Identifier = idAlwaysSame{}

func (idAlwaysSame) New() (string, error) { return "0123456789abcdef0123456789abcdef", nil }

func (idAlwaysSame) Parse(id string) error { return identifier.HexIdentifier{}.Parse(id) }

func (idAlwaysSame) MarshalBinary(id string) ([]byte, error) {
	return identifier.HexIdentifier{}.MarshalBinary(id)
}

func (idAlwaysSame) UnmarshalBinary(b []byte) (string, error) {
	return identifier.HexIdentifier{}.UnmarshalBinary(b)
}

// idOscillatingLen alternates between two different string lengths on each New call, violating
// the stable-length requirement enforced by [vaulttest.CheckIdentifierNewManyDistinct].
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

// idParsePermissive embeds [identifier.HexIdentifier] for New/Marshal/Unmarshal but overrides
// Parse to accept any string, defeating length validation.
type idParsePermissive struct{ identifier.HexIdentifier }

func (idParsePermissive) Parse(string) error { return nil }

// idTruncateUnmarshal delegates New/Parse/Marshal to [identifier.HexIdentifier] but drops the last
// byte on UnmarshalBinary, breaking the round-trip contract checked by
// [vaulttest.CheckIdentifierMarshalBinaryRoundtrip].
type idTruncateUnmarshal struct{}

func (idTruncateUnmarshal) New() (string, error) { return identifier.HexIdentifier{}.New() }

func (idTruncateUnmarshal) Parse(id string) error { return identifier.HexIdentifier{}.Parse(id) }

func (idTruncateUnmarshal) MarshalBinary(id string) ([]byte, error) {
	return identifier.HexIdentifier{}.MarshalBinary(id)
}

func (idTruncateUnmarshal) UnmarshalBinary(b []byte) (string, error) {
	if len(b) < 1 {
		return "", fmt.Errorf("empty")
	}
	return string(b[:len(b)-1]), nil
}
