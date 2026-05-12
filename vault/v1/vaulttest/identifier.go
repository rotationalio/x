package vaulttest

// Identifier conformance helpers for tests.
//
// These functions are not registered as Go tests by name: call [IdentifierConforms] or the
// individual [CheckIdentifier…] helpers from your *_test.go files. They are safe to ship in
// non-test sources.
//
// They encode what the root vault expects from [identifier.Identifier]: stable string length across mints,
// uniqueness of ids, Parse rejecting off-by-one lengths, and lossless binary marshal round-trips
// for the canonical string form.

import (
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/vault/v1/identifier"
)

// DefaultIdentifierDistinctSamples is the sample count used by [IdentifierConforms] for the
// distinct-id stress check. Exported [CheckIdentifierNewManyDistinct] uses the same default when
// passed this value.
const DefaultIdentifierDistinctSamples = 10_000

// CheckIdentifierNewManyDistinct verifies that New returns many distinct ids of stable length.
//
// Every sample must be unique; length must match the first non-empty id for all subsequent samples.
func CheckIdentifierNewManyDistinct(idGen identifier.Identifier, samples int) error {
	seen := make(map[string]struct{}, samples)
	var wantLen int

	// Mint many ids: lengths must stay stable and every id must be unique.
	for i := range samples {
		idStr, err := idGen.New()
		if err != nil {
			return fmt.Errorf("sample %d New: %w", i, err)
		}
		if i == 0 {
			if len(idStr) == 0 {
				return fmt.Errorf("sample 0: New returned empty id")
			}
			wantLen = len(idStr)
		} else if len(idStr) != wantLen {
			return fmt.Errorf("sample %d: id length %d want %d (id %q)", i, len(idStr), wantLen, idStr)
		}
		if _, dup := seen[idStr]; dup {
			return fmt.Errorf("duplicate id after %d samples: %q", len(seen), idStr)
		}
		seen[idStr] = struct{}{}
	}

	if len(seen) != samples {
		return fmt.Errorf("map size %d want %d", len(seen), samples)
	}
	return nil
}

// CheckIdentifierParseAcceptsMinted verifies Parse accepts ids returned by New.
func CheckIdentifierParseAcceptsMinted(idGen identifier.Identifier) error {
	idStr, err := idGen.New()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	if err := idGen.Parse(idStr); err != nil {
		return fmt.Errorf("parse(minted): %w", err)
	}
	return nil
}

// CheckIdentifierParseRejectsWrongLength verifies Parse rejects ids one byte shorter or longer.
//
// The "long by one" case appends a duplicate of the last byte so hex parsers cannot reject on
// charset alone—length must be enforced.
func CheckIdentifierParseRejectsWrongLength(idGen identifier.Identifier) error {
	idStr, err := idGen.New()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	n := len(idStr)
	if n == 0 {
		return fmt.Errorf("new returned empty id")
	}
	if err := idGen.Parse(idStr[:n-1]); err == nil {
		return fmt.Errorf("parse(short by one): got nil want error")
	}
	last := idStr[n-1]
	if err := idGen.Parse(idStr + string(last)); err == nil {
		return fmt.Errorf("parse(long by one): got nil want error")
	}
	return nil
}

// CheckIdentifierMarshalBinaryRoundtrip verifies MarshalBinary / UnmarshalBinary round-trip.
func CheckIdentifierMarshalBinaryRoundtrip(idGen identifier.Identifier) error {
	idStr, err := idGen.New()
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	raw, err := idGen.MarshalBinary(idStr)
	if err != nil {
		return fmt.Errorf("marshalBinary: %w", err)
	}
	back, err := idGen.UnmarshalBinary(raw)
	if err != nil {
		return fmt.Errorf("unmarshalBinary: %w", err)
	}
	if back != idStr {
		return fmt.Errorf("round-trip: got %q want %q", back, idStr)
	}
	return nil
}

// IdentifierConforms runs minimal subtests that an implementation of [identifier.Identifier] should satisfy
// for use with the sealed vault: many minted ids stay unique and share one canonical byte length, minted
// ids parse cleanly, Parse rejects ids one byte shorter or one byte longer than that length (last
// byte duplicated), and MarshalBinary/UnmarshalBinary round-trip the canonical string form.
func IdentifierConforms(t *testing.T, idGen identifier.Identifier) {
	t.Helper()

	t.Run("new_many_distinct", func(t *testing.T) {
		assert.Ok(t, CheckIdentifierNewManyDistinct(idGen, DefaultIdentifierDistinctSamples))
	})

	t.Run("parse_accepts_minted", func(t *testing.T) {
		assert.Ok(t, CheckIdentifierParseAcceptsMinted(idGen))
	})

	t.Run("parse_rejects_wrong_length", func(t *testing.T) {
		assert.Ok(t, CheckIdentifierParseRejectsWrongLength(idGen))
	})

	t.Run("marshal_binary_roundtrip", func(t *testing.T) {
		assert.Ok(t, CheckIdentifierMarshalBinaryRoundtrip(idGen))
	})
}
