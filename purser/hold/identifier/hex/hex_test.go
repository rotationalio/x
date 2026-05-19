package hex_test

import (
	"testing"

	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
	"go.rtnl.ai/x/purser/hold/identifier/identifiertest"
)

// TestHexIdentifier_compliance runs [identifiertest.IdentifierConforms] against [hexid.Identifier].
func TestHexIdentifier_compliance(t *testing.T) {
	identifiertest.IdentifierConforms(t, hexid.Identifier{})
}
