package hex_test

import (
	"testing"

	"go.rtnl.ai/x/purser/hold/holdtest"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
)

// TestHexIdentifier_compliance runs [holdtest.IdentifierConforms] against [hexid.Identifier].
func TestHexIdentifier_compliance(t *testing.T) {
	holdtest.IdentifierConforms(t, hexid.Identifier{})
}
