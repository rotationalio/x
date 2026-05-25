package fuzz

// Fuzz targets for v1 [models.Meta] wire framing.

import (
	"testing"

	"go.rtnl.ai/x/assert"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
	"go.rtnl.ai/x/purser/locker/v1/models"
)

//=============================================================================
// Fuzz: Meta.UnmarshalBinary
//=============================================================================

// FuzzMeta_unmarshal exercises [models.Meta.UnmarshalBinary] on random inputs.
// Invariants: no panic; success implies canonical re-marshal.
func FuzzMeta_unmarshal(f *testing.F) {
	var (
		good []byte
		err  error
	)

	good, err = (models.Meta{
		Version:   constv1.Version,
		KeyID:     []byte{1, 2, 3},
		Namespace: "ns",
	}).MarshalBinary()
	assert.Ok(f, err, "seed marshal")

	f.Add(good)
	f.Add([]byte{})
	f.Add([]byte{constv1.Version, 0, 0})
	f.Add(append([]byte(nil), good[:len(good)-1]...))

	f.Fuzz(func(t *testing.T, data []byte) {
		var (
			m   models.Meta
			out []byte
		)

		if err = m.UnmarshalBinary(data); err != nil {
			return
		}
		out, err = m.MarshalBinary()
		assert.Ok(t, err, "unmarshal succeeded but re-marshal failed")
		assert.Equal(t, string(data), string(out), "round-trip mismatch")
	})
}
