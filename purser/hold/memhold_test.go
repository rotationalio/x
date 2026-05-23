package hold_test

import (
	"context"
	"testing"

	"go.rtnl.ai/x/assert"
	perrors "go.rtnl.ai/x/purser/errors"
	"go.rtnl.ai/x/purser/hold"
	"go.rtnl.ai/x/purser/hold/holdtest"
	hexid "go.rtnl.ai/x/purser/hold/identifier/hex"
)

// TestNewMemHold_nilIdentifier rejects a nil identifier strategy.
func TestNewMemHold_nilIdentifier(t *testing.T) {
	_, err := hold.NewMemHold(nil)
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestMemHold_nilReceiver_create verifies Create on a nil MemHold returns ErrInvalidNewArgs.
func TestMemHold_nilReceiver_create(t *testing.T) {
	var h *hold.MemHold
	_, err := h.Create(context.Background(), "ns", []byte("x"))
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestMemHold_nilReceiver_get verifies Get on a nil MemHold returns ErrInvalidNewArgs.
func TestMemHold_nilReceiver_get(t *testing.T) {
	var h *hold.MemHold
	_, err := h.Get(context.Background(), "ns", "00112233445566778899aabbccddeeff")
	assert.ErrorIs(t, err, perrors.ErrInvalidNewArgs)
}

// TestMemHold_compliance runs holdtest.HoldConforms against hold.MemHold.
func TestMemHold_compliance(t *testing.T) {
	holdtest.HoldConforms(t, func(tb *testing.T) hold.Hold {
		tb.Helper()
		h, err := hold.NewMemHold(hexid.Identifier{})
		assert.Ok(tb, err, "new memhold")
		return h
	})
}
