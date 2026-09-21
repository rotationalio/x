package password_test

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/password"
)

func TestPolicy_Charset(t *testing.T) {
	// cSpell:ignore qrstuv
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

	t.Run("URLEncoded", func(t *testing.T) {
		p := &Policy{
			Strength:  Moderate,
			Length:    &Range{Min: 12},
			Require:   []string{"lowercase", "uppercase", "digits", "symbols"},
			URLEncode: true,
		}
		err := p.Check("R4d%21shRaNs0m3%24")
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

func TestPolicy_Generate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		p := &Policy{
			Charsets: []*CharSelect{
				{Name: "lowercase", Prob: 0.35},
				{Name: "uppercase", Prob: 0.35},
				{Name: "digits", Prob: 0.2},
				{Name: "symbols", Prob: 0.1},
			},
			Length:  &Range{Min: 14, Max: 18},
			Require: []string{"lowercase", "uppercase", "digits", "symbols"},
		}

		seen := make(map[string]struct{})
		for range 32 {
			password, err := p.Generate()
			assert.Ok(t, err, "expected the password to be generated")
			seen[password] = struct{}{}

			// Expect the password to be between 14 and 18 characters long.
			assert.GreaterEqual(t, 14, len(password))
			assert.LessEqual(t, 18, len(password))

			// Expect the password to contain all of the required character sets.
			assert.True(t, Contains(password, Charset("lowercase")), "the password should contain lowercase characters")
			assert.True(t, Contains(password, Charset("uppercase")), "the password should contain uppercase characters")
			assert.True(t, Contains(password, Charset("digits")), "the password should contain digits")
			assert.True(t, Contains(password, Charset("symbols")), "the password should contain symbols")
		}
		assert.Len(t, seen, 32, "it is incredibly unlikely that in 32 attempts we would generate the same password twice")
	})

	t.Run("DefaultLength", func(t *testing.T) {
		p := &Policy{
			Charsets: []*CharSelect{
				{Name: "lowercase", Prob: 0.35},
				{Name: "uppercase", Prob: 0.35},
				{Name: "digits", Prob: 0.2},
				{Name: "symbols", Prob: 0.1},
			},
		}
		password, err := p.Generate()
		assert.Ok(t, err, "expected the password to be generated")
		assert.Equal(t, 14, len(password), "the password should be 14 characters long")
	})

	t.Run("URLEncoded", func(t *testing.T) {
		p := &Policy{
			URLEncode: true,
			Charsets: []*CharSelect{
				{Name: "lowercase", Prob: 0.25},
				{Name: "uppercase", Prob: 0.25},
				{Name: "digits", Prob: 0.2},
				{Name: "symbols", Prob: 0.3},
			},
			Length:  &Range{Min: 15, Max: 15},
			Require: []string{"symbols"},
		}
		password, err := p.Generate()
		assert.Ok(t, err, "expected the password to be generated")

		decoded, err := url.PathUnescape(password)
		assert.Ok(t, err, "expected the password to be unescaped")
		assert.Len(t, decoded, 15, "the password should be 15 characters long")
	})

	t.Run("UndefinedCharset", func(t *testing.T) {
		p := &Policy{
			Charsets: []*CharSelect{
				{Name: "lowercase", Prob: 0.35},
				{Name: "uppercase", Prob: 0.35},
				{Name: "zephyr", Prob: 0.2},
				{Name: "symbols", Prob: 0.1},
			},
		}

		_, err := p.Generate()
		assert.Error(t, err, "expected the password to fail to be generated")
		assert.Equal(t, "unknown or undefined character set \"zephyr\" is not defined", err.Error(), "the expected error message did not match")
	})

	t.Run("MissingRequired", func(t *testing.T) {
		p := &Policy{
			Charsets: []*CharSelect{
				{Name: "lowercase", Prob: 0.35},
				{Name: "uppercase", Prob: 0.35},
				{Name: "digits", Prob: 0.2},
			},
			Require: []string{"lowercase", "uppercase", "zephyr", "symbols"},
		}
		_, err := p.Generate()
		assert.Error(t, err, "expected the password to fail to be generated")
		assert.Equal(t, "required character set \"zephyr\" is not defined", err.Error(), "the expected error message did not match")
	})

	t.Run("Pathological", func(t *testing.T) {
		p := &Policy{
			Strength: Robust,
			Length:   &Range{Min: 10, Max: 10},
			Charsets: []*CharSelect{
				{Name: "lowercase", Prob: 0.4985},
				{Name: "uppercase", Prob: 0.4985},
				{Name: "digits", Prob: 0.002},
				{Name: "symbols", Prob: 0.001},
			},
			Require: []string{"lowercase", "uppercase", "digits", "symbols"},
		}

		// We assume it is very difficult to generate a password that meets the policy
		// requirements -- so it will fail to generate a password in most cases.
		_, err := p.Generate()
		assert.Error(t, err, "expected the password to fail to be generated")
	})
}

func TestRange_Get(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		rng := &Range{}
		assert.Equal(t, 14, rng.Get(), "the default password length should be returned")
	})

	t.Run("Equal", func(t *testing.T) {
		rng := &Range{
			Min: 18,
			Max: 18,
		}
		assert.Equal(t, 18, rng.Get(), "the password length should be returned")
	})

	t.Run("Random", func(t *testing.T) {
		rng := &Range{
			Min: 8,
			Max: 20,
		}
		for range 32 {
			num := rng.Get()
			assert.GreaterEqual(t, 8, num)
			assert.LessEqual(t, 20, num)
		}
	})

	t.Run("OddBall", func(t *testing.T) {
		rng := &Range{
			Min: 15,
			Max: 0,
		}
		assert.Equal(t, 15, rng.Get(), "the minimum length should be returned")
	})

}

func TestCharsetCounts(t *testing.T) {
	t.Run("Probabilities", func(t *testing.T) {
		dist := []*CharSelect{
			{Name: "uppercase", Prob: 0.35},
			{Name: "lowercase", Prob: 0.35},
			{Name: "digits", Prob: 0.2},
			{Name: "symbols", Prob: 0.1},
		}

		for range 128 {
			n := rand.Intn(32) + 1
			counts := CharsetCounts(n, dist)

			total := 0
			for name, count := range counts {
				assert.True(t, name == "uppercase" || name == "lowercase" || name == "digits" || name == "symbols", "the name should be a valid charset")
				total += count
			}
			assert.Equal(t, n, total, "the total count should be the same as the input")
		}
	})

	t.Run("Integers", func(t *testing.T) {
		t.Run("Zero", func(t *testing.T) {
			dist := []*CharSelect{
				{Name: "uppercase", Prob: 0},
				{Name: "lowercase", Prob: 0},
				{Name: "digits", Prob: 0},
				{Name: "symbols", Prob: 2},
			}
			counts := CharsetCounts(0, dist)
			total := 0
			for name, count := range counts {
				assert.True(t, name == "uppercase" || name == "lowercase" || name == "digits" || name == "symbols", "the name should be a valid charset")
				total += count
			}
			assert.Equal(t, 0, total, "the total count should be 0")
		})

		t.Run("Empty", func(t *testing.T) {
			dist := []*CharSelect{
				{Name: "uppercase", Prob: 0},
				{Name: "lowercase", Prob: 0},
				{Name: "digits", Prob: 0},
				{Name: "symbols", Prob: 0},
			}
			counts := CharsetCounts(14, dist)
			assert.Len(t, counts, len(dist), "the counts should be returned with the same length")
			assert.Equal(t, 4, counts["uppercase"], "the uppercase count should be 4")
			assert.Equal(t, 4, counts["lowercase"], "the lowercase count should be 4")
			assert.Equal(t, 3, counts["digits"], "the digits count should be 3")
			assert.Equal(t, 3, counts["symbols"], "the symbols count should be 2")
		})

		t.Run("Fuzz", func(t *testing.T) {
			for range 128 {
				// Create random distributions of charactersets with integers.
				dist := []*CharSelect{}
				for _, name := range []string{"uppercase", "lowercase", "digits", "symbols"} {
					if rand.Float64() < 0.2 {
						// 20% chance of skipping the charset.
						continue
					}
					dist = append(dist, &CharSelect{Name: name, Prob: float64(rand.Intn(7))})
				}

				if len(dist) == 0 {
					continue
				}

				n := rand.Intn(32) + 1
				counts := CharsetCounts(n, dist)

				total := 0
				for name, count := range counts {
					assert.True(t, count > 0, "the count should be greater than 0 for each charset")
					assert.True(t, name == "uppercase" || name == "lowercase" || name == "digits" || name == "symbols", "the name should be a valid charset")
					total += count
				}
				assert.Equal(t, n, total, "the total count should be the same as the input")
			}
		})

		t.Run("Exact", func(t *testing.T) {
			dist := []*CharSelect{
				{Name: "uppercase", Prob: 5.0},
				{Name: "lowercase", Prob: 5.0},
				{Name: "digits", Prob: 3.0},
				{Name: "symbols", Prob: 2.0},
			}

			counts := CharsetCounts(15, dist)
			assert.Len(t, counts, len(dist), "the counts should be returned with the same length")
			assert.Equal(t, 5, counts["uppercase"], "the uppercase count should be 5")
			assert.Equal(t, 5, counts["lowercase"], "the lowercase count should be 5")
			assert.Equal(t, 3, counts["digits"], "the digits count should be 3")
			assert.Equal(t, 2, counts["symbols"], "the symbols count should be 2")
		})

		t.Run("Unused", func(t *testing.T) {
			dist := []*CharSelect{
				{Name: "uppercase"},
				{Name: "lowercase"},
				{Name: "digits", Prob: 3.0},
				{Name: "symbols", Prob: 2.0},
			}
			counts := CharsetCounts(15, dist)
			assert.Len(t, counts, len(dist), "the counts should be returned with the same length")
			assert.Equal(t, 5, counts["uppercase"], "the uppercase count should be 5")
			assert.Equal(t, 5, counts["lowercase"], "the lowercase count should be 5")
			assert.Equal(t, 3, counts["digits"], "the digits count should be 3")
			assert.Equal(t, 2, counts["symbols"], "the symbols count should be 2")
		})

		t.Run("Overused", func(t *testing.T) {
			dist := []*CharSelect{
				{Name: "uppercase", Prob: 5.0},
				{Name: "lowercase", Prob: 5.0},
				{Name: "digits", Prob: 3.0},
				{Name: "symbols", Prob: 2.0},
			}
			counts := CharsetCounts(14, dist)
			assert.Len(t, counts, len(dist), "the counts should be returned with the same length")
			assert.Equal(t, 5, counts["uppercase"], "the uppercase count should be 5")
			assert.Equal(t, 5, counts["lowercase"], "the lowercase count should be 5")
			assert.Equal(t, 3, counts["digits"], "the digits count should be 3")
			assert.Equal(t, 1, counts["symbols"], "the symbols count should be 2")
		})

		t.Run("Underused", func(t *testing.T) {
			dist := []*CharSelect{
				{Name: "uppercase", Prob: 3.0},
				{Name: "lowercase", Prob: 4.0},
				{Name: "digits", Prob: 3.0},
				{Name: "symbols", Prob: 2.0},
			}
			counts := CharsetCounts(16, dist)
			assert.Len(t, counts, len(dist), "the counts should be returned with the same length")
			assert.Equal(t, 4, counts["uppercase"], "the uppercase count should be 4")
			assert.Equal(t, 5, counts["lowercase"], "the lowercase count should be 5")
			assert.Equal(t, 4, counts["digits"], "the digits count should be 4")
			assert.Equal(t, 3, counts["symbols"], "the symbols count should be 3")
		})

		t.Run("Remainder", func(t *testing.T) {
			dist := []*CharSelect{
				{Name: "uppercase"},
				{Name: "lowercase"},
				{Name: "digits", Prob: 3.0},
				{Name: "symbols", Prob: 2.0},
			}
			counts := CharsetCounts(12, dist)
			assert.Len(t, counts, len(dist), "the counts should be returned with the same length")
			assert.Equal(t, 4, counts["uppercase"], "the uppercase count should be 4")
			assert.Equal(t, 3, counts["lowercase"], "the lowercase count should be 4")
			assert.Equal(t, 3, counts["digits"], "the digits count should be 3")
			assert.Equal(t, 2, counts["symbols"], "the symbols count should be 1")
		})
	})
}

func TestNormalizeCharDist(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		dist := NormalizeCharDist(nil)
		assert.Len(t, dist, 4, "the default character distribution should be returned")
		probs := 0.0
		for _, charset := range dist {
			probs += charset.Prob
		}
		assert.Equal(t, 1.0, round(probs), "the total probability should be 1.0")
	})

	t.Run("Normalized", func(t *testing.T) {
		dist := []*CharSelect{
			{Name: "uppercase", Prob: 0.45},
			{Name: "lowercase", Prob: 0.4},
			{Name: "digits", Prob: 0.15},
		}
		assert.Equal(t, dist, NormalizeCharDist(dist), "the character distribution should be returned as is")
	})

	t.Run("Remainder", func(t *testing.T) {
		dist := []*CharSelect{
			{Name: "uppercase"},
			{Name: "lowercase"},
			{Name: "digits", Prob: 0.3},
			{Name: "symbols", Prob: 0.1},
		}
		norm := NormalizeCharDist(dist)
		assert.Len(t, norm, 4, "the character distribution should be returned with the same length")
		assert.Equal(t, 0.3, norm[0].Prob, "the uppercase probability should be 0.3")
		assert.Equal(t, 0.3, norm[1].Prob, "the lowercase probability should be 0.3")
		assert.Equal(t, dist[2].Prob, norm[2].Prob, "the digits probability should be unchanged")
		assert.Equal(t, dist[3].Prob, norm[3].Prob, "the symbols probability should be unchanged")
	})

	t.Run("Redistribute", func(t *testing.T) {
		dist := []*CharSelect{
			{Name: "uppercase", Prob: 0.3},
			{Name: "lowercase", Prob: 0.3},
			{Name: "digits", Prob: 0.1},
			{Name: "symbols", Prob: 0.1},
		}
		norm := NormalizeCharDist(dist)
		assert.Len(t, norm, 4, "the character distribution should be returned with the same length")
		assert.Equal(t, 0.35, norm[0].Prob, "the uppercase probability should be 0.3")
		assert.Equal(t, 0.35, norm[1].Prob, "the lowercase probability should be 0.35")
		assert.Equal(t, 0.15, norm[2].Prob, "the digits probability should be unchanged")
		assert.Equal(t, 0.15, norm[3].Prob, "the symbols probability should be unchanged")
	})

	t.Run("Integers", func(t *testing.T) {
		dist := []*CharSelect{
			{Name: "uppercase", Prob: 8},
			{Name: "lowercase", Prob: 10},
			{Name: "digits", Prob: 1},
			{Name: "symbols", Prob: 2},
		}
		assert.Equal(t, dist, NormalizeCharDist(dist), "the character distribution should be returned as is")
	})

	t.Run("IntegersWithZero", func(t *testing.T) {
		dist := []*CharSelect{
			{Name: "uppercase", Prob: 0},
			{Name: "lowercase", Prob: 0},
			{Name: "digits", Prob: 0},
			{Name: "symbols", Prob: 2},
		}
		assert.Equal(t, dist, NormalizeCharDist(dist), "the character distribution should be returned as is")
	})

	t.Run("Pathological", func(t *testing.T) {
		dist := []*CharSelect{
			{Name: "uppercase", Prob: 0.3},
			{Name: "lowercase", Prob: 4.8},
			{Name: "digits", Prob: 2.1},
			{Name: "symbols", Prob: 0},
		}
		norm := NormalizeCharDist(dist)
		assert.Len(t, norm, 4, "the character distribution should be returned with the same length")
		assert.Equal(t, 1.0, norm[0].Prob, "the uppercase probability should be 1")
		assert.Equal(t, 4.0, norm[1].Prob, "the lowercase probability should be 4")
		assert.Equal(t, 2.0, norm[2].Prob, "the digits probability should be 1")
		assert.Equal(t, 1.0, norm[3].Prob, "the symbols probability should be 1")
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

func round(f float64) float64 {
	ratio := math.Pow(10, 4)
	return math.Round(f*ratio) / ratio
}
