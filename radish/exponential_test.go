package radish_test

import (
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/radish"
)

func TestExponentialBackOff(t *testing.T) {
	t.Run("TestDoublingIntervals", func(t *testing.T) {
		// Intervals should double starting from 1 second
		// Ensure randomization factor is 0 to ensure these are predictable
		// intervals.
		backoff := NewExponentialBackOff(
			WithInitialInterval(1*time.Second),
			WithMultiplier(2.0),
			WithMaxInterval(10*time.Second),
			WithMaxElapsedTime(10*time.Second),
			WithRandomizationFactor(0),
		)

		// Check that the next intervals are as expected
		expectedIntervals := []time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
			8 * time.Second,
			StopBackOff,
		}

		for _, expected := range expectedIntervals {
			assert.Equal(t, expected, backoff.Next(), "expected the next interval to match")
		}
	})

	t.Run("TestMaxInterval", func(t *testing.T) {
		// Set multiplier high to ensure max interval is hit
		backoff := NewExponentialBackOff(
			WithInitialInterval(1*time.Second),
			WithMultiplier(10.0),
			WithMaxInterval(5*time.Second),
			WithRandomizationFactor(0),
		)

		// Intervals should not exceed the max interval
		expectedIntervals := []time.Duration{
			1 * time.Second,
			5 * time.Second,
			5 * time.Second,
			5 * time.Second,
		}

		for _, expected := range expectedIntervals {
			assert.Equal(t, expected, backoff.Next(), "expected the next interval to match")
		}
	})

	t.Run("TestRandomFactor", func(t *testing.T) {
		// Set randomization factor for variability
		backoff := NewExponentialBackOff(
			WithInitialInterval(1*time.Second),
			WithMultiplier(1.0),
			WithMaxInterval(10*time.Second),
			WithRandomizationFactor(0.5),
		)

		// With r=0.5 the next interval should be between 0.5x and 1.5x of the
		// initial interval
		initial := 1 * time.Second
		minInterval := time.Duration(float64(initial) * 0.5)
		maxInterval := time.Duration(float64(initial) * 1.5)

		for range 10 {
			next := backoff.Next()
			assert.GreaterEqual(t, minInterval, next, "next interval should be at least half of the initial interval")
			assert.LessEqual(t, maxInterval, next, "next interval should be at most 1.5 times the initial interval")
		}
	})
}
