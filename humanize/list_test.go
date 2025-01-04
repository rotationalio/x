package humanize_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/humanize"
)

func TestList(t *testing.T) {
	t.Run("And", func(t *testing.T) {
		testCases := []struct {
			strs     []string
			expected string
		}{
			{
				[]string{},
				"",
			},
			{
				[]string{"apples"},
				"apples",
			},
			{
				[]string{"apples", "oranges"},
				"apples and oranges",
			},
			{
				[]string{"apples", "oranges", "berries"},
				"apples, oranges and berries",
			},
			{
				[]string{"apples", "oranges", "lemons", "limes"},
				"apples, oranges, lemons and limes",
			},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, humanize.AndList(tc.strs), "test case %d failed", i)
		}
	})

	t.Run("Or", func(t *testing.T) {
		testCases := []struct {
			strs     []string
			expected string
		}{
			{
				[]string{},
				"",
			},
			{
				[]string{"apples"},
				"apples",
			},
			{
				[]string{"apples", "oranges"},
				"apples or oranges",
			},
			{
				[]string{"apples", "oranges", "pear"},
				"apples, oranges or pear",
			},
			{
				[]string{"apples", "oranges", "pear", "figs"},
				"apples, oranges, pear or figs",
			},
		}

		for i, tc := range testCases {
			assert.Equal(t, tc.expected, humanize.OrList(tc.strs), "test case %d failed", i)
		}
	})
}
