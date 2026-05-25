package fuzz

// Tests that fuzz edition registration stays aligned with [registry.Editions].

import (
	"slices"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/keyring/registry"
)

//=============================================================================
// Tests: edition seed registration
//=============================================================================

// TestEditionSeedsCoverRegistry fails when [registry.Editions] lacks a fuzz seed row.
func TestEditionSeedsCoverRegistry(t *testing.T) {
	registered := registry.Editions()
	for edition := range editionSeeds {
		assert.True(t, slices.Contains(registered, edition), edition)
	}
}
