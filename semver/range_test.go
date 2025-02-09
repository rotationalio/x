package semver_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/semver"
)

func TestRange(t *testing.T) {
	makeRangeTest := func(rng string, cases []string, require assert.BoolAssertion) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()
			r, err := Range(rng)
			assert.Ok(t, err, "could not parse range")

			for i, v := range cases {
				require(t, r(v), "test case %d failed", i)
			}
		}
	}

	t.Run("EQ", makeRangeTest("=1.2.3", []string{"1.2.3"}, assert.True))
	t.Run("NE", makeRangeTest("=1.2.3", []string{"1.2.4", "1.8.3", "0.2.3", "2.2.3", "1.2.3-alpha.1"}, assert.False))
	t.Run("GT", makeRangeTest(">1.2.3", []string{"2.2.3", "1.3.3", "1.3.4", "1.3.4-alpha.1"}, assert.True))
	t.Run("NGT", makeRangeTest(">1.2.3", []string{"0.2.3", "1.1.3", "1.2.2", "1.2.3-alpha.4"}, assert.False))
	t.Run("LT", makeRangeTest("<1.2.3", []string{"0.2.3", "1.1.3", "1.2.2", "1.2.3-alpha.4"}, assert.True))
	t.Run("NLT", makeRangeTest("<1.2.3", []string{"2.2.3", "1.3.3", "1.3.4", "1.3.4-alpha.1"}, assert.False))
	t.Run("GTE", makeRangeTest(">=1.2.3", []string{"1.2.3", "2.2.3", "1.3.3", "1.3.4", "1.3.4-alpha.1"}, assert.True))
	t.Run("NGTE", makeRangeTest(">=1.2.3", []string{"0.2.3", "1.1.3", "1.2.2", "1.2.3-alpha.4"}, assert.False))
	t.Run("LTE", makeRangeTest("<=1.2.3", []string{"1.2.3", "0.2.3", "1.1.3", "1.2.2", "1.2.3-alpha.4"}, assert.True))
	t.Run("NLTE", makeRangeTest("<=1.2.3", []string{"2.2.3", "1.3.3", "1.3.4", "1.3.4-alpha.1"}, assert.False))
	t.Run("OR", makeRangeTest("<1.2.3 || >3.2.1", []string{"1.2.2", "1.1.3", "0.2.3", "4.2.1", "3.3.1", "3.2.2"}, assert.True))
	t.Run("NOR", makeRangeTest("<1.2.3 || >3.2.1", []string{"2.2.3", "1.3.3", "1.2.4", "2.2.1", "3.1.1", "3.2.0"}, assert.False))
	t.Run("AND", makeRangeTest(">1.2.3 <3.2.1", []string{"1.2.4", "1.3.3", "2.2.3", "2.2.1", "3.1.1", "3.2.0"}, assert.True))
	t.Run("NAND", makeRangeTest(">1.2.3 <3.2.1", []string{"0.2.3", "1.1.3", "1.2.2", "4.2.1", "3.3.1", "3.2.2"}, assert.False))
}

func TestSpecification(t *testing.T) {
	spec, err := Range(">=1.2.3")
	assert.Ok(t, err, "could not parse range")

	testCases := []struct {
		val     interface{}
		require assert.BoolAssertion
	}{
		{"1.2.3", assert.True},
		{Version{Major: 1, Minor: 2, Patch: 3}, assert.True},
		{&Version{Major: 1, Minor: 2, Patch: 3}, assert.True},
		{"1.D.3", assert.False},
		{42, assert.False},
		{false, assert.False},
	}

	for i, tc := range testCases {
		tc.require(t, spec(tc.val), "test case %d failed", i)
	}
}

func TestInvalidRange(t *testing.T) {
	testCases := []string{
		"foo",
		"!1.2.3",
		"=1.2.3 >2.3.4 foo",
		"=1.2.3 || >2.3.4 ||",
		"|| 1.2.3",
	}

	for _, tc := range testCases {
		spec, err := Range(tc)
		assert.ErrorIs(t, err, ErrInvalidRange)
		assert.Nil(t, spec)
	}
}
