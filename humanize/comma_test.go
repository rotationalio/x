package humanize_test

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/humanize"
)

// Named numeric types to ensure the ~ constraints are exercised.
type (
	Altitude int32
	Port     uint16
	Ratio    float32
)

func TestComma(t *testing.T) {
	t.Run("Int", func(t *testing.T) {
		testCases := []struct {
			n        int64
			expected string
		}{
			{0, "0"},
			{1, "1"},
			{7, "7"},
			{8, "8"},
			{9, "9"},
			{10, "10"},
			{99, "99"},
			{100, "100"},
			{999, "999"},
			{1000, "1,000"},
			{1001, "1,001"},
			{9999, "9,999"},
			{10000, "10,000"},
			{100000, "100,000"},
			{999999, "999,999"},
			{1000000, "1,000,000"},
			{1234567, "1,234,567"},
			{1234567890, "1,234,567,890"},
			{math.MaxInt64, "9,223,372,036,854,775,807"},
			{-1, "-1"},
			{-7, "-7"},
			{-8, "-8"},
			{-10, "-10"},
			{-100, "-100"},
			{-999, "-999"},
			{-1000, "-1,000"},
			{-1234, "-1,234"},
			{-999999, "-999,999"},
			{-1000000, "-1,000,000"},
			{-1234567, "-1,234,567"},
			{math.MinInt64 + 1, "-9,223,372,036,854,775,807"},
			{math.MinInt64, "-9,223,372,036,854,775,808"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Comma(tc.n), "test case %d failed", i)
		}
	})

	t.Run("Uint", func(t *testing.T) {
		testCases := []struct {
			n        uint64
			expected string
		}{
			{0, "0"},
			{7, "7"},
			{8, "8"},
			{999, "999"},
			{1000, "1,000"},
			{1234567, "1,234,567"},
			{math.MaxInt64, "9,223,372,036,854,775,807"},
			{math.MaxInt64 + 1, "9,223,372,036,854,775,808"},
			{math.MaxUint64, "18,446,744,073,709,551,615"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Comma(tc.n), "test case %d failed", i)
		}
	})

	t.Run("Float", func(t *testing.T) {
		testCases := []struct {
			n        float64
			expected string
		}{
			{0, "0"},
			{0.5, "0.5"},
			{1, "1"},
			{999, "999"},
			{999.999, "999.999"},
			{1000, "1,000"},
			{1234.5678, "1,234.5678"},
			{1000000, "1,000,000"},
			{1234567.891, "1,234,567.891"},
			{123456789.125, "123,456,789.125"},
			{-0.5, "-0.5"},
			{-999, "-999"},
			{-1234.5678, "-1,234.5678"},
			{-1000000.25, "-1,000,000.25"},

			// Small and large values are not rendered in scientific notation
			{0.0000001, "0.0000001"},
			{1e21, "1,000,000,000,000,000,000,000"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Comma(tc.n), "test case %d failed", i)
		}
	})

	t.Run("Types", func(t *testing.T) {
		// Signed integers
		assert.Equal(t, "-1,234,567", Comma(int(-1234567)), "int failed")
		assert.Equal(t, "-128", Comma(int8(math.MinInt8)), "int8 failed")
		assert.Equal(t, "-32,768", Comma(int16(math.MinInt16)), "int16 failed")
		assert.Equal(t, "-2,147,483,648", Comma(int32(math.MinInt32)), "int32 failed")
		assert.Equal(t, "-9,223,372,036,854,775,808", Comma(int64(math.MinInt64)), "int64 failed")

		// Unsigned integers
		assert.Equal(t, "1,234,567", Comma(uint(1234567)), "uint failed")
		assert.Equal(t, "255", Comma(uint8(math.MaxUint8)), "uint8 failed")
		assert.Equal(t, "65,535", Comma(uint16(math.MaxUint16)), "uint16 failed")
		assert.Equal(t, "4,294,967,295", Comma(uint32(math.MaxUint32)), "uint32 failed")
		assert.Equal(t, "18,446,744,073,709,551,615", Comma(uint64(math.MaxUint64)), "uint64 failed")
		assert.Equal(t, "1,234,567", Comma(uintptr(1234567)), "uintptr failed")

		// Floats
		assert.Equal(t, "1,234.5", Comma(float32(1234.5)), "float32 failed")
		assert.Equal(t, "1,234.5678", Comma(float64(1234.5678)), "float64 failed")

		// Untyped constants
		assert.Equal(t, "1,000", Comma(1000), "untyped int failed")
		assert.Equal(t, "1,000.5", Comma(1000.5), "untyped float failed")
	})

	t.Run("NamedTypes", func(t *testing.T) {
		assert.Equal(t, "-10,994", Comma(Altitude(-10994)))
		assert.Equal(t, "65,535", Comma(Port(65535)))
		assert.Equal(t, "1,024.25", Comma(Ratio(1024.25)))
	})

	t.Run("Special", func(t *testing.T) {
		assert.Equal(t, "NaN", Comma(math.NaN()))
		assert.Equal(t, "+Inf", Comma(math.Inf(1)))
		assert.Equal(t, "-Inf", Comma(math.Inf(-1)))
		assert.Equal(t, "+Inf", Comma(float32(math.Inf(1))))
	})

	t.Run("NegativeZero", func(t *testing.T) {
		assert.Equal(t, "0", Comma(math.Copysign(0, -1)))
		assert.Equal(t, "0", Comma(float32(math.Copysign(0, -1))))
	})

	t.Run("Boundaries", func(t *testing.T) {
		for _, n := range int64Boundaries() {
			assert.Equal(t, insertCommas(strconv.FormatInt(n, 10)), Comma(n), "int64 %d failed", n)
		}

		for _, n := range uint64Boundaries() {
			assert.Equal(t, insertCommas(strconv.FormatUint(n, 10)), Comma(n), "uint64 %d failed", n)
		}
	})

	t.Run("Random", func(t *testing.T) {
		rng := newRand()
		for i := 0; i < 256; i++ {
			// Shift by a random amount so that all orders of magnitude are tested.
			n := int64(rng.Uint64()) >> rng.UintN(64)
			assert.Equal(t, insertCommas(strconv.FormatInt(n, 10)), Comma(n), "int64 %d failed", n)

			u := rng.Uint64() >> rng.UintN(64)
			assert.Equal(t, insertCommas(strconv.FormatUint(u, 10)), Comma(u), "uint64 %d failed", u)
		}
	})
}

func TestCommaf(t *testing.T) {
	t.Run("Float64", func(t *testing.T) {
		testCases := []struct {
			n         float64
			precision int
			expected  string
		}{
			// Fewer decimal places than the precision are padded with zeros
			{0, 2, "0.00"},
			{1, 2, "1.00"},
			{0.5, 3, "0.500"},
			{1234, 2, "1,234.00"},
			{1234.5, 2, "1,234.50"},
			{1234.5, 4, "1,234.5000"},
			{1.5, 8, "1.50000000"},
			{1000000, 1, "1,000,000.0"},

			// Exactly the right number of decimal places are unchanged
			{0.125, 3, "0.125"},
			{1234.56, 2, "1,234.56"},
			{0.1234567891, 10, "0.1234567891"},

			// Extra decimal places are truncated, not rounded
			{1.999, 2, "1.99"},
			{2.675, 2, "2.67"},
			{999.999, 1, "999.9"},
			{1234.5678, 2, "1,234.56"},
			{1234.5678, 3, "1,234.567"},
			{1234567.891, 1, "1,234,567.8"},

			// Zero or negative precision returns only the integer part
			{0.999, 0, "0"},
			{999.999, 0, "999"},
			{1234, 0, "1,234"},
			{1234.5678, 0, "1,234"},
			{1234.5678, -1, "1,234"},
			{1234567.891, -10, "1,234,567"},

			// Negative values
			{-1, 2, "-1.00"},
			{-1234, 2, "-1,234.00"},
			{-1234.5678, 2, "-1,234.56"},
			{-1000000.5, 3, "-1,000,000.500"},
			{-999.999, 0, "-999"},

			// Large values are not rendered in scientific notation
			{123456789.125, 2, "123,456,789.12"},
			{1e21, 2, "1,000,000,000,000,000,000,000.00"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Commaf(tc.n, tc.precision), "test case %d failed", i)
		}
	})

	t.Run("Float32", func(t *testing.T) {
		// These values are all exactly representable as a float32.
		testCases := []struct {
			n         float32
			precision int
			expected  string
		}{
			{0.25, 1, "0.2"},
			{0.25, 4, "0.2500"},
			{1234.5, 2, "1,234.50"},
			{-2048.75, 1, "-2,048.7"},
			{1000000, 0, "1,000,000"},
			{16777216, 2, "16,777,216.00"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Commaf(tc.n, tc.precision), "test case %d failed", i)
		}
	})

	t.Run("NamedTypes", func(t *testing.T) {
		assert.Equal(t, "1,024.2", Commaf(Ratio(1024.25), 1))
		assert.Equal(t, "1,024.250", Commaf(Ratio(1024.25), 3))
		assert.Equal(t, "-273.15", Commaf(Celsius(-273.15), 2))
	})

	t.Run("Special", func(t *testing.T) {
		// Special values are never padded with zeros regardless of the precision
		for _, precision := range []int{-1, 0, 2, 6} {
			assert.Equal(t, "NaN", Commaf(math.NaN(), precision), "precision %d failed", precision)
			assert.Equal(t, "+Inf", Commaf(math.Inf(1), precision), "precision %d failed", precision)
			assert.Equal(t, "-Inf", Commaf(math.Inf(-1), precision), "precision %d failed", precision)
		}

		assert.Equal(t, "NaN", Commaf(float32(math.NaN()), 2))
		assert.Equal(t, "+Inf", Commaf(float32(math.Inf(1)), 2))
		assert.Equal(t, "-Inf", Commaf(float32(math.Inf(-1)), 2))
	})

	t.Run("NegativeZero", func(t *testing.T) {
		testCases := []struct {
			n         float64
			precision int
			expected  string
		}{
			// Negative values that truncate to zero do not keep the negative sign
			{math.Copysign(0, -1), 0, "0"},
			{math.Copysign(0, -1), 2, "0.00"},
			{-0.4, 0, "0"},
			{-0.999, -1, "0"},
			{-0.001, 2, "0.00"},
			{-0.0001, 3, "0.000"},

			// Negative values that do not truncate to zero keep the negative sign
			{-0.5, 1, "-0.5"},
			{-0.01, 2, "-0.01"},
			{-0.001, 3, "-0.001"},
			{-0.001, 5, "-0.00100"},
			{-1.001, 0, "-1"},
			{-10.001, 2, "-10.00"},
			{-1000.001, 2, "-1,000.00"},
			{-1000000.4, 0, "-1,000,000"},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, Commaf(tc.n, tc.precision), "test case %d failed", i)
		}
	})
}

func TestParseComma(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			s        string
			expected int64
		}{
			{"0", 0},
			{"7", 7},
			{"999", 999},
			{"1,000", 1000},
			{"1,234,567", 1234567},
			{"9,223,372,036,854,775,807", math.MaxInt64},
			{"-1", -1},
			{"-999", -999},
			{"-1,000", -1000},
			{"-1,234,567", -1234567},
			{"-9,223,372,036,854,775,808", math.MinInt64},

			// Commas are optional
			{"1234567", 1234567},
			{"-1234567", -1234567},
		}

		for i, tc := range testCases {
			actual, err := ParseComma(tc.s)
			assert.Ok(t, err, "test case %d errored", i)
			assert.Equal(t, tc.expected, actual, "test case %d failed", i)
		}
	})

	t.Run("Syntax", func(t *testing.T) {
		testCases := []string{
			"",
			",",
			"-",
			"abc",
			"1,234abc",
			"1,234.56",
			" 1,234",
			"1 234",
			"NaN",
		}

		for i, tc := range testCases {
			_, err := ParseComma(tc)
			assert.ErrorIs(t, err, strconv.ErrSyntax, "test case %d failed", i)
		}
	})

	t.Run("Range", func(t *testing.T) {
		testCases := []string{
			"9,223,372,036,854,775,808",
			"-9,223,372,036,854,775,809",
			"18,446,744,073,709,551,615",
		}

		for i, tc := range testCases {
			_, err := ParseComma(tc)
			assert.ErrorIs(t, err, strconv.ErrRange, "test case %d failed", i)
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		for _, n := range int64Boundaries() {
			actual, err := ParseComma(Comma(n))
			assert.Ok(t, err, "%d errored", n)
			assert.Equal(t, n, actual, "%d failed", n)
		}

		rng := newRand()
		for i := 0; i < 256; i++ {
			n := int64(rng.Uint64()) >> rng.UintN(64)
			actual, err := ParseComma(Comma(n))
			assert.Ok(t, err, "%d errored", n)
			assert.Equal(t, n, actual, "%d failed", n)
		}
	})
}

func TestParseCommaf(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			s        string
			expected float64
		}{
			{"0", 0},
			{"0.5", 0.5},
			{"0.00", 0},
			{"999.999", 999.999},
			{"1,234", 1234},
			{"1,234.56", 1234.56},
			{"1,234.50", 1234.5},
			{"1,234,567.891", 1234567.891},
			{"1,000,000,000,000,000,000,000", 1e21},
			{"-0.5", -0.5},
			{"-1,234.5678", -1234.5678},
			{"-1,000,000.00", -1000000},

			// Commas are optional
			{"1234.5", 1234.5},
			{"-1234567.891", -1234567.891},
		}

		for i, tc := range testCases {
			actual, err := ParseCommaf(tc.s)
			assert.Ok(t, err, "test case %d errored", i)
			assert.Equal(t, tc.expected, actual, "test case %d failed", i)
		}
	})

	t.Run("Special", func(t *testing.T) {
		// The special values produced by Comma must be parseable.
		actual, err := ParseCommaf(Comma(math.NaN()))
		assert.Ok(t, err)
		assert.True(t, math.IsNaN(actual), "expected NaN")

		actual, err = ParseCommaf(Comma(math.Inf(1)))
		assert.Ok(t, err)
		assert.True(t, math.IsInf(actual, 1), "expected +Inf")

		actual, err = ParseCommaf(Comma(math.Inf(-1)))
		assert.Ok(t, err)
		assert.True(t, math.IsInf(actual, -1), "expected -Inf")
	})

	t.Run("Syntax", func(t *testing.T) {
		testCases := []string{
			"",
			",",
			"-",
			".",
			"abc",
			"1,234abc",
			"1,234.5.6",
			" 1,234.5",
			"1 234.5",
		}

		for i, tc := range testCases {
			_, err := ParseCommaf(tc)
			assert.ErrorIs(t, err, strconv.ErrSyntax, "test case %d failed", i)
		}
	})

	t.Run("Range", func(t *testing.T) {
		// A comma separated integer that is too large to be represented by a float64.
		huge := "1" + strings.Repeat(",000", 103)

		_, err := ParseCommaf(huge)
		assert.ErrorIs(t, err, strconv.ErrRange)

		_, err = ParseCommaf("-" + huge)
		assert.ErrorIs(t, err, strconv.ErrRange)
	})

	t.Run("RoundTrip", func(t *testing.T) {
		rng := newRand()
		for i := 0; i < 256; i++ {
			// Scale by a random power of ten so that all orders of magnitude are tested.
			n := (rng.Float64()*2 - 1) * math.Pow10(rng.IntN(16))

			// Comma does not lose any precision so the value must match exactly.
			actual, err := ParseCommaf(Comma(n))
			assert.Ok(t, err, "%f errored", n)
			assert.Equal(t, n, actual, "%f failed", n)

			// Commaf truncates so the value must be within the truncated precision.
			actual, err = ParseCommaf(Commaf(n, 2))
			assert.Ok(t, err, "%f errored", n)
			assert.InDelta(t, n, actual, 0.01, "%f failed", n)
		}
	})
}

//===========================================================================
// Helpers
//===========================================================================

// A seeded random number generator so that failures are reproducible.
func newRand() *rand.Rand {
	return rand.New(rand.NewPCG(42, 1024))
}

// Reference implementation to compare the optimized Comma functions against: inserts
// commas into a string of base 10 digits that may be prefixed with a negative sign.
func insertCommas(digits string) string {
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}

	sb := &strings.Builder{}
	for i := 0; i < len(digits); i++ {
		if i > 0 && (len(digits)-i)%3 == 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte(digits[i])
	}

	return sign + sb.String()
}

// Returns the positive and negative values on either side of every power of ten that
// can be represented by an int64, as well as the limits of the int64 range; e.g. all
// of the values where the number of digits or commas in the output changes.
func int64Boundaries() []int64 {
	vals := []int64{math.MinInt64, math.MinInt64 + 1, math.MaxInt64 - 1, math.MaxInt64}
	for p, k := int64(1), 0; k <= 18; k++ {
		vals = append(vals, p-1, p, p+1, -(p - 1), -p, -(p + 1))
		if k < 18 {
			p *= 10
		}
	}
	return vals
}

// Returns the values on either side of every power of ten that can be represented by
// a uint64, as well as the limits of the uint64 range.
func uint64Boundaries() []uint64 {
	vals := []uint64{math.MaxInt64, math.MaxInt64 + 1, math.MaxUint64 - 1, math.MaxUint64}
	for p, k := uint64(1), 0; k <= 19; k++ {
		vals = append(vals, p-1, p, p+1)
		if k < 19 {
			p *= 10
		}
	}
	return vals
}
