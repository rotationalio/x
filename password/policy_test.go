package password_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/password"
)

func TestPolicy_Charset(t *testing.T) {
	p := &Policy{
		Define: map[string]string{
			"alphabet": "qrstuv",
			"numbers":  "1234",
		},
	}
	assert.Equal(t, p.Charset("alphabet"), "qrstuv", "policy defined charset")
	assert.Equal(t, p.Charset("numbers"), "1234", "overridden charset")
	assert.Equal(t, p.Charset("digits"), Charset("digits"), "default charset")
	assert.Equal(t, p.Charset("zephyr"), "", "unknown charset")
}

func TestPolicy_Check(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		p := &Policy{
			Strength: Moderate,
			Length:   &Range{Min: 12},
			Require:  []string{"lowercase", "uppercase", "digits", "symbols"},
		}
		err := p.Check("R4d!shRaNs0m3$")
		assert.Ok(t, err, "expected the password to pass the policy check")
	})

	t.Run("Invalid", func(t *testing.T) {
		t.Run("Strength", func(t *testing.T) {
			p := &Policy{
				Strength: Hard,
			}
			err := p.Check("supersecretpassword")
			assert.Error(t, err, "expected the password to fail the policy check")
			assert.Equal(t, "password strength is weak, minimum required strength is hard", err.Error(), "the expected error message did not match")
		})

		t.Run("TooShort", func(t *testing.T) {
			p := &Policy{
				Length: &Range{
					Min: 10,
				},
			}
			err := p.Check("123456789")
			assert.Error(t, err, "expected the password to fail the policy check")
			assert.Equal(t, "password length is 9, minimum required length is 10", err.Error(), "the expected error message did not match")
		})

		t.Run("TooLong", func(t *testing.T) {
			p := &Policy{
				Length: &Range{
					Min: 5,
					Max: 10,
				},
			}
			err := p.Check("abcdef1234567890")
			assert.Error(t, err, "expected the password to fail the policy check")
			assert.Equal(t, "password length is 16, maximum allowed length is 10", err.Error(), "the expected error message did not match")
		})

		t.Run("MissingCharset", func(t *testing.T) {
			p := &Policy{
				Require: []string{"lowercase", "uppercase", "digits"},
			}
			err := p.Check("lower$123")
			assert.Error(t, err, "expected the password to fail the policy check")
		})
	})
}

func TestRange_JSON(t *testing.T) {
	tests := []struct {
		rng      *Range
		expected string
	}{
		{
			rng: &Range{
				Min: 8,
				Max: 16,
			},
			expected: `{"max":16,"min":8}`,
		},
		{
			rng: &Range{
				Min: 14,
				Max: 14,
			},
			expected: `14`,
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("Test Case %d", i+1), func(t *testing.T) {
			data, err := json.Marshal(tc.rng)
			assert.Ok(t, err)
			assert.Equal(t, tc.expected, string(data))

			var cmpt *Range
			err = json.Unmarshal(data, &cmpt)
			assert.Ok(t, err)
			assert.Equal(t, tc.rng, cmpt)
		})
	}
}

func TestCharSelect_JSON(t *testing.T) {
	tests := []struct {
		cs       *CharSelect
		expected string
	}{
		{
			cs: &CharSelect{
				Name: "alpha",
				Prob: 0.5,
			},
			expected: `{"name":"alpha","prob":0.5}`,
		},
		{
			cs: &CharSelect{
				Name: "numeric",
			},
			expected: `"numeric"`,
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("Test Case %d", i+1), func(t *testing.T) {
			data, err := json.Marshal(tc.cs)
			assert.Ok(t, err)
			assert.Equal(t, tc.expected, string(data))

			var cmpt *CharSelect
			err = json.Unmarshal(data, &cmpt)
			assert.Ok(t, err)
			assert.Equal(t, tc.cs, cmpt)
		})
	}
}
