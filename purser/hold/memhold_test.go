package hold_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/hold/holdtest"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
)

// TestMemHold_compliance runs holdtest.HoldConforms against hold.MemHold.
func TestMemHold_compliance(t *testing.T) {
	holdtest.HoldConforms(t, func(tb *testing.T) hold.Hold {
		tb.Helper()
		h, err := hold.NewMemHold(hexid.Identifier{})
		assert.Ok(tb, err, "new memhold")
		return h
	})
}
