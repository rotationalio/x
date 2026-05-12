package storage_test

import (
	"testing"

	"go.rtnl.ai/x/vault/v1/identifier"
	"go.rtnl.ai/x/vault/v1/storage"
	"go.rtnl.ai/x/vault/v1/vaulttest"
)

// TestMemStorage_compliance runs [vaulttest.StorageConforms] against [storage.MemStorage]
// and [identifier.HexIdentifier].
func TestMemStorage_compliance(t *testing.T) {
	vaulttest.StorageConforms(t, identifier.HexIdentifier{}, func(tb *testing.T) storage.Storage {
		tb.Helper()
		return storage.NewMemStorage()
	})
}
