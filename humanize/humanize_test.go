package humanize_test

import (
	"math/rand/v2"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/humanize"
)

func TestSingular(t *testing.T) {

	makeTest := func(n float64, require assert.BoolAssertion) func(t *testing.T) {
		return func(t *testing.T) {
			require(t, Singular(int(n)), "int failed")
			require(t, Singular(int8(n)), "int8 failed")
			require(t, Singular(int16(n)), "int16 failed")
			require(t, Singular(int32(n)), "int32 failed")
			require(t, Singular(int64(n)), "int64 failed")
			require(t, Singular(uint(n)), "uint failed")
			require(t, Singular(uint8(n)), "uint8 failed")
			require(t, Singular(uint16(n)), "uint16 failed")
			require(t, Singular(uint32(n)), "uint32 failed")
			require(t, Singular(uint64(n)), "uint64 failed")
			require(t, Singular(float32(n)), "float32 failed")
			require(t, Singular(float64(n)), "float64 failed")
		}
	}

	t.Run("One", makeTest(1, assert.True))

	t.Run("Two", makeTest(2, assert.False))

	t.Run("Zero", makeTest(0, assert.False))

	t.Run("RandInt", func(t *testing.T) {
		for i := 0; i < 32; i++ {
			i := rand.IntN(1000000) + 2
			assert.False(t, Singular(i))
			assert.False(t, Singular(i*-1))
		}
	})

	t.Run("RandFloat", func(t *testing.T) {
		for i := 0; i < 32; i++ {
			i := rand.Float64() + float64(rand.Int64N(1000000)) + 0.0000000001
			assert.False(t, Singular(i))
			assert.False(t, Singular(i*-1))
		}
	})
}

func TestMakePlural(t *testing.T) {
	// common case
	assert.Equal(t, "bytes", MakePlural(2, "byte", "", ""))
	assert.Equal(t, "byte", MakePlural(1, "byte", "", ""))

	// specify prefix case
	assert.Equal(t, "buses", MakePlural(2, "bus", "es", ""))
	assert.Equal(t, "bus", MakePlural(1, "bus", "es", ""))

	// irregular case
	assert.Equal(t, "geese", MakePlural(2, "goose", "", "geese"))
	assert.Equal(t, "goose", MakePlural(1, "goose", "", "geese"))

	// trim and replace
	assert.Equal(t, "wolves", MakePlural(2, "wolf", "ves", "f"))
	assert.Equal(t, "wolf", MakePlural(1, "wolf", "ves", "f"))

	// y to ies
	assert.Equal(t, "libraries", MakePlural(2, "library", "ies", "y"))
	assert.Equal(t, "library", MakePlural(1, "library", "ies", "y"))
}

func TestPlural(t *testing.T) {
	testCases := []struct {
		n        int64
		singular string
		plural   string
		expected string
	}{
		{1, "byte", "bytes", "byte"},
		{2, "byte", "bytes", "bytes"},
		{1, "bus", "buses", "bus"},
		{2, "bus", "buses", "buses"},
		{1, "goose", "geese", "goose"},
		{2, "goose", "geese", "geese"},
		{1, "", "", ""},
		{2, "", "", ""},
	}

	for i, tc := range testCases {
		assert.Equal(t, tc.expected, Plural(tc.n, tc.singular, tc.plural), "test case %d failed", i)
	}
}

func TestPluralize(t *testing.T) {
	testCases := []struct {
		unit     string
		expected string
	}{
		{"byte", "bytes"},
		{"bus", "buses"},
		{"oboe", "oboes"},
		{"library", "libraries"},
		{"wolf", "wolves"},
		{"life", "lives"},
		{"church", "churches"},
		{"box", "boxes"},
		{"city", "cities"},
		{"calf", "calves"},
		{"dog", "dogs"},
		{"criterion", "criteria"},
		{"chair", "chairs"},
		{"mango", "mangoes"},
	}

	for i, tc := range testCases {
		assert.Equal(t, tc.expected, Pluralize(tc.unit), "test case %d failed", i)
	}
}
