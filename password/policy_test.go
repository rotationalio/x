package password_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/password"
)

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
