package fuzz

// Fuzz targets for v1 sealed-row wire parsers ([models.Sealed], [models.ParseOpenWire]).

import (
	"testing"

	"go.rtnl.ai/x/assert"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/models"
)

//=============================================================================
// Fuzz: Sealed wire parse
//=============================================================================

// FuzzSealed_unmarshal exercises [models.Sealed.UnmarshalBinary] on random wire.
// Invariants: no panic; success implies canonical re-marshal.
func FuzzSealed_unmarshal(f *testing.F) {
	AddV1WireSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		var (
			s   models.Sealed
			out []byte
			err error
		)

		if err = s.UnmarshalBinary(data); err != nil {
			return
		}
		out, err = s.MarshalBinary()
		assert.Ok(t, err, "unmarshal succeeded but re-marshal failed")
		assert.Equal(t, string(data), string(out), "round-trip mismatch")
	})
}

// FuzzParseOpenWire exercises [models.ParseOpenWire] on random wire and namespaces.
// Invariants: no panic; success implies valid slice lengths and non-empty routing key id.
func FuzzParseOpenWire(f *testing.F) {
	AddV1OpenWireSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte, requestedNS string) {
		var (
			formatVersion uint8
			eph           []byte
			payload       []byte
			kid           []byte
			err           error
		)

		formatVersion, _, eph, _, payload, err = models.ParseOpenWire(data, requestedNS)
		if err != nil {
			return
		}
		if formatVersion != constv1.Version {
			t.Fatalf("ParseOpenWire: format version %d want %d", formatVersion, constv1.Version)
		}
		if len(eph) != constv1.EphPubBytes {
			t.Fatalf("ParseOpenWire: eph len %d want %d", len(eph), constv1.EphPubBytes)
		}
		if len(payload) < constv1.GCMTagBytes {
			t.Fatalf("ParseOpenWire: payload len %d want at least %d", len(payload), constv1.GCMTagBytes)
		}

		kid, err = models.ParseKeyIDFromSealed(data)
		assert.Ok(t, err, "ParseOpenWire ok but ParseKeyIDFromSealed failed")
		assert.True(t, len(kid) > 0, "ParseOpenWire ok but key id empty")
	})
}

// FuzzParseKeyID_agreesWithUnmarshal checks fast and full sealed parsers stay aligned.
// Invariants: no panic; both succeed or both fail; on success key ids match and are non-empty.
func FuzzParseKeyID_agreesWithUnmarshal(f *testing.F) {
	AddV1WireSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		var (
			kid     []byte
			s       models.Sealed
			errFast error
			errFull error
		)

		kid, errFast = models.ParseKeyIDFromSealed(data)
		errFull = s.UnmarshalBinary(data)

		if errFast != nil && errFull != nil {
			return
		}
		if errFast == nil && errFull != nil {
			t.Fatalf("ParseKeyIDFromSealed ok but UnmarshalBinary failed: %v", errFull)
		}
		if errFast != nil && errFull == nil {
			t.Fatalf("UnmarshalBinary ok but ParseKeyIDFromSealed failed: %v", errFast)
		}
		assert.True(t, len(kid) > 0, "both parsers ok but key id empty")
		assert.Equal(t, s.Meta.KeyID, kid, "ParseKeyIDFromSealed disagrees with UnmarshalBinary")
	})
}
