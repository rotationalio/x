package backoff_test

import (
	"math"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/backoff"
)

func TestBackOff(t *testing.T) {
	var (
		testInitialInterval     = 500 * time.Millisecond
		testRandomizationFactor = 0.1
		testMultiplier          = 2.0
		testMaxInterval         = 5 * time.Second
	)

	exp := NewExponentialBackOff()
	exp.InitialInterval = testInitialInterval
	exp.RandomizationFactor = testRandomizationFactor
	exp.Multiplier = testMultiplier
	exp.MaxInterval = testMaxInterval
	exp.Reset()

	var expectedResults = []time.Duration{500, 1000, 2000, 4000, 5000, 5000, 5000, 5000, 5000, 5000}
	for i, d := range expectedResults {
		expectedResults[i] = d * time.Millisecond
	}

	for _, expected := range expectedResults {
		assert.Equal(t, expected, exp.Current())

		// Assert that the next backoff falls in the expected range
		minval := expected - time.Duration(testRandomizationFactor*float64(expected))
		maxval := expected + time.Duration(testRandomizationFactor*float64(expected))
		actual := exp.NextBackOff()
		assert.True(t, minval <= actual && actual <= maxval, "Expected %v to be in range [%v, %v]", actual, minval, maxval)
	}
}

func TestBackOffOverflow(t *testing.T) {
	var (
		testInitialInterval time.Duration = math.MaxInt64 / 2
		testMaxInterval     time.Duration = math.MaxInt64
		testMultiplier                    = 2.1
	)

	exp := NewExponentialBackOff()
	exp.InitialInterval = testInitialInterval
	exp.Multiplier = testMultiplier
	exp.MaxInterval = testMaxInterval
	exp.Reset()

	exp.NextBackOff()
	// Assert that when an overflow is possible, the current interval time.Duration is set to the max interval time.Duration.
	assert.Equal(t, testMaxInterval, exp.Current())
}

func TestRandomizedInterval(t *testing.T) {
	// 33% chance of being 1
	assert.Equal(t, time.Duration(1), RandomizedInterval(0.5, 0, 2))
	assert.Equal(t, time.Duration(1), RandomizedInterval(0.5, 0.33, 2))

	// 33% chance of being 2
	assert.Equal(t, time.Duration(2), RandomizedInterval(0.5, 0.34, 2))
	assert.Equal(t, time.Duration(2), RandomizedInterval(0.5, 0.66, 2))

	// 33% chance of being 3
	assert.Equal(t, time.Duration(3), RandomizedInterval(0.5, 0.67, 2))
	assert.Equal(t, time.Duration(3), RandomizedInterval(0.5, 0.99, 2))
}
