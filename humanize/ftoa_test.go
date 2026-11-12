package humanize_test

import (
	"math"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/humanize"
)

// Named float type to ensure the ~float64 constraint is exercised.
type Celsius float64

func TestFtoa(t *testing.T) {
	t.Run("Float64", func(t *testing.T) {
		testCases := []struct {
			n         float64
			precision int
			expected  string
		}{
			// Whole numbers never render a decimal point or lose integer zeros
			{0, 0, "0"},
			{0, 2, "0"},
			{1, 2, "1"},
			{42, 6, "42"},
			{100, 0, "100"},
			{100, 2, "100"},
			{1200, 3, "1200"},

			// Fewer decimal places than the precision are not padded
			{1.5, 3, "1.5"},
			{10.1, 2, "10.1"},
			{0.25, 6, "0.25"},

			// Trailing zeros left after truncation are stripped
			{1.5001, 2, "1.5"},
			{3.001, 2, "3"},
			{10.101, 2, "10.1"},

			// Extra decimal places are truncated, not rounded
			{3.14159, 2, "3.14"},
			{3.14159, 4, "3.1415"},
			{1.999, 2, "1.99"},
			{2.675, 2, "2.67"},
			{0.999, 0, "0"},
			{123.456789, 3, "123.456"},

			// Zero or negative precision returns only the integer part
			{3.99, 0, "3"},
			{3.99, -1, "3"},
			{123.456, -10, "123"},

			// Precision is capped at six decimal places; the value is rounded
			// at the sixth place before truncation.
			{123.456789, 6, "123.456789"},
			{123.456789, 10, "123.456789"},
			{1.23456789, 8, "1.234568"},
			{0.000001, 6, "0.000001"},
			{0.0000001, 6, "0"},
			{0.9999999, 2, "1"},

			// Negative values
			{-1, 2, "-1"},
			{-1.5, 1, "-1.5"},
			{-123.456, 2, "-123.45"},
			{-3.99, 0, "-3"},
			{-100, 2, "-100"},

			// Large values are not rendered in scientific notation
			{1e15, 2, "1000000000000000"},
			{123456789.125, 2, "123456789.12"},
			{1e21, 2, "1000000000000000000000"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Ftoa(tc.n, tc.precision), "test case %d failed", i)
		}
	})

	t.Run("Float32", func(t *testing.T) {
		testCases := []struct {
			n         float32
			precision int
			expected  string
		}{
			// float32 representation error must not leak into the output
			{0.1, 6, "0.1"},
			{0.3, 6, "0.3"},
			{1.1, 2, "1.1"},
			{2.5, 2, "2.5"},
			{3.14159, 4, "3.1415"},
			{1.999, 2, "1.99"},
			{-7.25, 1, "-7.2"},
			{16777216, 2, "16777216"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Ftoa(tc.n, tc.precision), "test case %d failed", i)
		}
	})

	t.Run("NamedType", func(t *testing.T) {
		assert.Equal(t, "36.6", Ftoa(Celsius(36.666), 1))
		assert.Equal(t, "-40", Ftoa(Celsius(-40), 2))
	})

	t.Run("Special", func(t *testing.T) {
		assert.Equal(t, "NaN", Ftoa(math.NaN(), 2))
		assert.Equal(t, "+Inf", Ftoa(math.Inf(1), 2))
		assert.Equal(t, "-Inf", Ftoa(math.Inf(-1), 2))
		assert.Equal(t, "+Inf", Ftoa(float32(math.Inf(1)), 0))
	})

	t.Run("NegativeZero", func(t *testing.T) {
		testCases := []struct {
			n         float64
			precision int
			expected  string
		}{
			// Negative values that truncate to zero do not keep the negative sign
			{math.Copysign(0, -1), 0, "0"},
			{math.Copysign(0, -1), 2, "0"},
			{-0.4, 0, "0"},
			{-0.999, -1, "0"},
			{-0.001, 2, "0"},
			{-0.0000001, 6, "0"},

			// Negative values that do not truncate to zero keep the negative sign
			{-0.5, 1, "-0.5"},
			{-0.01, 2, "-0.01"},
			{-0.001, 3, "-0.001"},
			{-1.001, 0, "-1"},
			{-10.001, 2, "-10"},
			{-1000.001, 2, "-1000"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Ftoa(tc.n, tc.precision), "test case %d failed", i)
		}
	})
}
