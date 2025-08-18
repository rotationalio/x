package quant_test

import (
	"fmt"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/quant"
)

// Implemented by [TestStemmers] in 'stemmers_test.go'

// Use the following test to test a single word stem, for debugging.
func TestPorter2Single(t *testing.T) {
	// NOTE: skipping this test unless we're debugging a word:
	t.SkipNow()

	// Debug a single, specific word
	in := "seaweed"
	exp := "seawe"
	act := quant.MustNewPorter2Stemmer(quant.LanuageEnglish).Stem(in)
	assert.Equal(t, exp, act, fmt.Sprintf("wrong stem for |%s|: expected |%s|, got |%s|", in, exp, act))
}
