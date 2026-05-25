package fuzz

// Fuzz targets for [locker.Locker.ParseKeyID] using [AddEditionWireSeeds].

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/locker"
	constv1 "go.rtnl.ai/x/purser/locker/v1/constants"
)

//=============================================================================
// Fuzz: locker.ParseKeyID
//=============================================================================

// FuzzParseKeyID exercises v1 [locker.Locker.ParseKeyID] on random ciphertext wire.
// Invariants: no panic; success implies non-empty key id for keyring routing.
func FuzzParseKeyID(f *testing.F) {
	var lck locker.Locker

	lck = NewLocker(f, constv1.Edition)
	AddEditionWireSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		var (
			kid []byte
			err error
		)

		kid, err = lck.ParseKeyID(data)
		if err != nil {
			return
		}
		assert.True(t, len(kid) > 0, "ParseKeyID returned nil error with empty key id")
	})
}
