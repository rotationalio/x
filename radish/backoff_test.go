package radish_test

import (
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/radish"
)

func TestConstantBackOff(t *testing.T) {
	// Create a constant backoff with a 1 second interval
	backoff := radish.NewConstantBackOff(1 * time.Second)

	// Check that the next interval is always 1 second
	for range 5 {
		assert.Equal(t, 1*time.Second, backoff.Next(), "expected the same interval for constant backoff")
	}
}

func TestZeroBackOff(t *testing.T) {
	// Create a zero backoff
	backoff := radish.NewZeroBackOff()

	// Check that the next interval is always 0
	for range 5 {
		assert.Equal(t, 0*time.Second, backoff.Next(), "expected zero interval for zero backoff")
	}
}
