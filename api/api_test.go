package api_test

import (
	"testing"

	"go.rtnl.ai/x/api"
	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/validation"
)

func TestSearchQuery(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		q := &api.SearchQuery{Query: "coinbase", Limit: 10}
		assert.Ok(t, q.Validate())

		q = &api.SearchQuery{Query: "coinbase", Limit: 0}
		assert.Ok(t, q.Validate())
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			q   *api.SearchQuery
			err error
		}{
			{
				&api.SearchQuery{Limit: 12},
				validation.Missing("query"),
			},
			{
				&api.SearchQuery{Query: "coinbase", Limit: -14},
				validation.Incorrect("limit", "limit cannot be less than zero"),
			},
			{
				&api.SearchQuery{Query: "coinbase", Limit: 100},
				validation.Incorrect("limit", "maximum number of search results that can be returned is 50"),
			},
		}

		for i, tc := range tests {
			assert.Equal(t, tc.q.Validate().Error(), tc.err.Error(), "test case %d failed", i)
		}
	})
}
