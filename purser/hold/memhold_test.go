package hold_test

import (
	"testing"

	storage "go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/hold/holdtest"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
)

// TestMemStorage_compliance runs [holdtest.StorageConforms] against [storage.MemStorage]
// and [hexid.Identifier].
func TestMemStorage_compliance(t *testing.T) {
	holdtest.StorageConforms(t, hexid.Identifier{}, func(tb *testing.T) storage.Storage {
		tb.Helper()
		return storage.NewMemStorage()
	})
}
