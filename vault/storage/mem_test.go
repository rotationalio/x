package storage_test

import (
	"testing"

	"go.rtnl.ai/x/vault/identifier"
	"go.rtnl.ai/x/vault/storage"
	"go.rtnl.ai/x/vault/vaulttest"
)

// TestMemStorage_compliance runs [vaulttest.StorageConforms] against [storage.MemStorage]
// and [identifier.HexIdentifier].
func TestMemStorage_compliance(t *testing.T) {
	vaulttest.StorageConforms(t, identifier.HexIdentifier{}, func(tb *testing.T) storage.Storage {
		tb.Helper()
		return storage.NewMemStorage()
	})
}
