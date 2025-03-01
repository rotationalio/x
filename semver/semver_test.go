package semver_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/semver"
)

func TestValid(t *testing.T) {
	tests := []struct {
		version string
		assert  assert.BoolAssertion
	}{
		{"1.2.3", assert.True},
		{"v1.2.3", assert.True},
		{"v10.2.3", assert.True},
		{"10.2.3", assert.True},
		{"1.202.3", assert.True},
		{"1.2.3312", assert.True},
		{"1.0.0-alpha", assert.True},
		{"1.0.0-alpha.1", assert.True},
		{"1.0.0-alpha.beta", assert.True},
		{"1.0.0-0.3.7", assert.True},
		{"1.0.0-x.7.z.92", assert.True},
		{"1.0.0-x-y-z.--", assert.True},
		{"1.0.0-alpha+001", assert.True},
		{"1.0.0+20130313144700", assert.True},
		{"1.0.0-beta+exp.sha.5114f85", assert.True},
		{"1.0.0+21AF26D3----117B344092BD", assert.True},
		{"f1.2.3", assert.False},
		{"01.2.3", assert.False},
		{"v01.2.3", assert.False},
		{"1.02.3", assert.False},
		{"1.2.03", assert.False},
		{"1.2", assert.False},
		{"1", assert.False},
		{"1.a.3", assert.False},
		{"3.4.c", assert.False},
		{"", assert.False},
		{"foo", assert.False},
		{"v", assert.False},
	}

	for i, tc := range tests {
		tc.assert(t, Valid(tc.version), "test case %d failed", i)
	}
}

func TestParse(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []string{
			"1.2.3",
			"10.2.3",
			"1.202.3",
			"1.2.3312",
			"1.0.0-alpha",
			"1.0.0-alpha.1",
			"1.0.0-alpha.beta",
			"1.0.0-0.3.7",
			"1.0.0-x.7.z.92",
			"1.0.0-x-y-z.--",
			"1.0.0-alpha+001",
			"1.0.0+20130313144700",
			"1.0.0-beta+exp.sha.5114f85",
			"1.0.0+21AF26D3----117B344092BD",
		}

		for _, tc := range tests {
			v, err := Parse(tc)
			assert.Ok(t, err, "expected parse to succeed")
			assert.Equal(t, tc, v.String(), "expected parsed version string to exactly match input")
		}
	})

	t.Run("Prefix", func(t *testing.T) {
		tests := []string{
			"v1.2.3",
			"v10.2.3",
		}

		for _, tc := range tests {
			v, err := Parse(tc)
			assert.Ok(t, err, "expected parse to succeed")
			assert.Equal(t, tc[1:], v.String(), "expected parsed version string to match input without prefix")
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []string{
			"f1.2.3",
			"01.2.3",
			"v01.2.3",
			"1.02.3",
			"1.2.03",
			"1.2",
			"1",
			"1.a.3",
			"3.4.c",
			"",
			"foo",
			"v",
		}

		for _, tc := range tests {
			v, err := Parse(tc)
			assert.ErrorIs(t, err, ErrInvalidSemVer, "expected parse to fail")
			assert.True(t, v.IsZero(), "expected parsed version to be zero value")
		}
	})
}

func TestIsZero(t *testing.T) {
	v := Version{}
	assert.True(t, v.IsZero(), "expected zero value to be zero")
	assert.False(t, Version{Major: 1}.IsZero(), "expected non-zero value to not be zero")
}

func TestShort(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha.4", BuildMeta: "exp.sha.5114f85"}
	assert.Equal(t, "1.2.3-alpha.4+exp.sha.5114f85", v.String(), "expected string representation to match")
	assert.Equal(t, "1.2.3", v.Short(), "expected short string representation to match")
}

func TestSpecifies(t *testing.T) {
	// This test is covered more extensively in range tests.
	ver := MustParse("1.2.3")

	t.Run("True", func(t *testing.T) {
		assert.True(t, ver.Satisfies(Specifies(func(v Version) bool {
			return true
		})))
	})

	t.Run("False", func(t *testing.T) {
		assert.False(t, ver.Satisfies(Specifies(func(v Version) bool {
			return false
		})))
	})
}

func TestMarshal(t *testing.T) {
	t.Run("Text", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			a := randVersion()
			text, err := a.MarshalText()
			assert.Ok(t, err)
			assert.True(t, Valid(string(text)))

			b := Version{}
			err = b.UnmarshalText(text)
			assert.Ok(t, err)

			assert.Equal(t, a, b)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		a := randVersion()
		text, err := json.Marshal(a)
		assert.Ok(t, err)

		var s string
		assert.Ok(t, json.Unmarshal(text, &s))
		assert.True(t, Valid(s))

		b := Version{}
		err = json.Unmarshal(text, &b)
		assert.Ok(t, err)

		assert.Equal(t, a, b)
	})
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.3.0", "2.3.0", 0},
		{"3.1.4", "3.1.4", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.0", "1.1.0", -1},
		{"2.1.1", "2.1.0", 1},
		{"2.1.0", "2.0.0", 1},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"1.0.0-alpha.beta", "1.0.0-beta", -1},
		{"1.0.0-beta", "1.0.0-beta.2", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-beta.11", "1.0.0-rc.1", -1},
		{"1.0.0-rc.1", "1.0.0", -1},
		{"1.0.0", "1.0.0+20130313144700", 0},
		{"1.0.0+20130313144700", "1.0.0+exp.sha.5114f85", 0},
	}

	for i, tc := range tests {
		a := MustParse(tc.a)
		b := MustParse(tc.b)
		assert.Equal(t, tc.expected, a.Compare(b), "test case %d failed", i)
		assert.Equal(t, -1*tc.expected, Compare(b, a), "test case %d failed", i)
	}
}

func TestSQL(t *testing.T) {
	t.Run("Scan", func(t *testing.T) {
		a := Version{}
		assert.Ok(t, a.Scan("6.1.2"))
		assert.Equal(t, Version{Major: 6, Minor: 1, Patch: 2}, a)

		b := Version{}
		assert.Ok(t, b.Scan([]byte("6.1.2")))
		assert.Equal(t, Version{Major: 6, Minor: 1, Patch: 2}, b)

		c := Version{}
		assert.Ok(t, c.Scan(nil))
		assert.True(t, c.IsZero())

		assert.ErrorIs(t, c.Scan(42), ErrScanValue)
	})

	t.Run("Value", func(t *testing.T) {
		v := randVersion()
		val, err := v.Value()
		assert.Ok(t, err)
		assert.Equal(t, v.String(), val)
	})
}

func randVersion() Version {
	return Version{
		Major:      uint16(rand.IntN(100) - 1),
		Minor:      uint16(rand.IntN(100) - 1),
		Patch:      uint16(rand.IntN(100) - 1),
		PreRelease: randPreRelease(),
		BuildMeta:  randBuildMeta(),
	}
}

func randPreRelease() (s string) {
	if rand.Float32() >= 0.8 {
		s = []string{"alpha", "beta", "final", "post", "rc"}[rand.IntN(5)]
		if rand.Float32() >= 0.5 {
			s += fmt.Sprintf(".%d", rand.IntN(10))
		}
	}
	return s
}

func randBuildMeta() string {
	if rand.Float32() >= 0.8 {
		return fmt.Sprintf("%x", rand.Int64())
	}
	return ""
}
