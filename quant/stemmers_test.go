package quant_test

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/quant"
)

// ############################################################################
// Tests
// ############################################################################

func TestStemmers(t *testing.T) {
	testcases := []struct {
		TestName     string
		Stemmer      quant.Stemmer
		InputPath    string
		ExpectedPath string
	}{
		{
			TestName:     "NoOpStemmer",
			Stemmer:      &quant.NoOpStemmer{},
			InputPath:    "testdata/NoOpStemmer/voc.txt",
			ExpectedPath: "testdata/NoOpStemmer/voc.txt", // same as input
		},
		{
			TestName:     "Porter2Stemmer [English]",
			Stemmer:      mustNewPorter2Stemmer(quant.LanuageEnglish),
			InputPath:    "testdata/Porter2Stemmer/voc.txt",
			ExpectedPath: "testdata/Porter2Stemmer/output.txt",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.TestName, func(t *testing.T) {
			// Load 'input' data
			input, err := os.Open(tc.InputPath)
			assert.NotNil(t, input, "unexpected nil input file")
			assert.Nil(t, err, "error opening 'input' file")
			defer input.Close()

			// Load 'expected' data
			expected, err := os.Open(tc.ExpectedPath)
			assert.NotNil(t, expected, "unexpected nil 'expected' file")
			assert.Nil(t, err, "error opening 'expected' file")
			defer expected.Close()

			// Scan each line of the input and compare to the output of the stemmer
			inputScanner := bufio.NewScanner(input)
			expectedScanner := bufio.NewScanner(expected)
			for inputScanner.Scan() && expectedScanner.Scan() {
				in := inputScanner.Text()
				// NOTE: Uncomment below to see the 'in' word to debug panics
				// fmt.Printf("IN: %s\n", in)
				exp := expectedScanner.Text()
				act := tc.Stemmer.Stem(in)
				assert.Equal(t, exp, act, fmt.Sprintf("wrong stem for |%s|: expected |%s|, got |%s|", in, exp, act))
			}
			// Ensure there were no scanning errors
			assert.Nil(t, inputScanner.Err(), "error scanning 'input'")
			assert.Nil(t, expectedScanner.Err(), "error scanning 'expected'")
		})
	}
}

// ############################################################################
// Benchmarking
// ############################################################################

//TODO: benchmark Porter2Stemmer

// ############################################################################
// Helpers
// ############################################################################

// Returns a new [quant.Porter2Stemmer] which supports the [quant.Language]
// given or panics on an error.
func mustNewPorter2Stemmer(lang quant.Language) (stemmer *quant.Porter2Stemmer) {
	var err error
	if stemmer, err = quant.NewPorter2Stemmer(lang); err != nil {
		panic(err)
	}
	return stemmer
}
