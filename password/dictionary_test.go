package password

import (
	"testing"

	"go.rtnl.ai/x/assert"
)

func TestIsDictionaryWord(t *testing.T) {
	tests := []struct {
		word   string
		assert assert.BoolAssertion
	}{
		// Words in the dictionary.
		{word: "password", assert: assert.True},
		{word: "absolutely", assert: assert.True},
		{word: "musician", assert: assert.True},
		{word: "offshore", assert: assert.True},
		{word: "landmark", assert: assert.True},
		{word: "insertion", assert: assert.True},
		{word: "letsdoit", assert: assert.True},
		{word: "edmonton", assert: assert.True},
		{word: "hurricane", assert: assert.True},
		{word: "contortionist", assert: assert.True},
		{word: "jordan23", assert: assert.True},
		{word: "newyork1", assert: assert.True},

		// The dictionary is lowercase, so lookups must fold case.
		{word: "PASSWORD", assert: assert.True},
		{word: "Musician", assert: assert.True},
		{word: "HuRrIcAnE", assert: assert.True},

		// Words of eight or more characters that are not in the dictionary.
		{word: "zebrafish", assert: assert.False},
		{word: "granulated", assert: assert.False},
		{word: "hyphenation", assert: assert.False},
		{word: "marmalade", assert: assert.False},
		{word: "vestibule", assert: assert.False},
		{word: "wavelength", assert: assert.False},
		{word: "sandpiper", assert: assert.False},
		{word: "bluehorizon", assert: assert.False},
		{word: "quixotic8", assert: assert.False},
		{word: "telescope4", assert: assert.False},
	}

	for i, tc := range tests {
		tc.assert(t, IsDictionaryWord(tc.word), "test %d: %q", i, tc.word)
	}
}
