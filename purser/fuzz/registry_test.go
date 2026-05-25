package fuzz

// Fuzz targets for [registry.ParseKeyID] edition dispatch.

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/keyring/registry"
)

//=============================================================================
// Fuzz: registry.ParseKeyID
//=============================================================================

// FuzzRegistry_ParseKeyID exercises [registry.ParseKeyID] edition dispatch on random wire.
// Invariants: no panic; success implies non-empty key id for keyring routing.
func FuzzRegistry_ParseKeyID(f *testing.F) {
	AddEditionWireSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		var (
			kid []byte
			err error
		)

		kid, err = registry.ParseKeyID(data)
		if err != nil {
			return
		}
		assert.True(t, len(kid) > 0, "registry.ParseKeyID returned nil error with empty key id")
	})
}
