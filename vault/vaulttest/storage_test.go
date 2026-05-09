package vaulttest

import (
	"testing"

	"go.rtnl.ai/x/vault"
)

// TestMemStorage_conformance runs [Run] against [NewMemStorage] so MemStorage stays
// aligned with the [vault.Storage] contract.
func TestMemStorage_conformance(t *testing.T) {
	Run(t, HexIdentifier{}, func(tb *testing.T) vault.Storage {
		tb.Helper()
		return NewMemStorage()
	})
}
