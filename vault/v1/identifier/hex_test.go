package identifier_test

import (
	"testing"

	"go.rtnl.ai/x/vault/v1/identifier"
	"go.rtnl.ai/x/vault/v1/vaulttest"
)

// TestHexIdentifier_compliance runs [vaulttest.IdentifierConforms] against [identifier.HexIdentifier].
func TestHexIdentifier_compliance(t *testing.T) {
	vaulttest.IdentifierConforms(t, identifier.HexIdentifier{})
}
