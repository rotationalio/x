package backoff_test

import (
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/backoff"
)

func TestZeroBackOff(t *testing.T) {
	subtestNextBackOff(t, 0, new(ZeroBackOff))
}

func TestStopBackOff(t *testing.T) {
	subtestNextBackOff(t, Stop, new(StopBackOff))
}

func TestConstantBackOff(t *testing.T) {
	subtestNextBackOff(t, 100*time.Millisecond, NewConstantBackOff(100*time.Millisecond))
}

func subtestNextBackOff(t *testing.T, expected time.Duration, policy BackOff) {
	for i := 0; i < 16; i++ {
		actual := policy.NextBackOff()
		assert.Equal(t, expected, actual)
	}
}
