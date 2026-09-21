package password_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/password"
)

func TestShuffle(t *testing.T) {
	s := "qwghuiasmnzxARTPFCVB19834!$#"
	seen := make(map[string]struct{})

	for range 32 {
		shuffled := password.Shuffle(s)
		assert.Equal(t, len(s), len(shuffled), "the shuffled string should have the same length")
		assert.NotEqual(t, s, shuffled, "the shuffled string should not be the same as the original")
		seen[shuffled] = struct{}{}
	}

	assert.Len(t, seen, 32, "it is incredibly unlikely that in 32 attempts we would shuffle to the same string twice")
}
