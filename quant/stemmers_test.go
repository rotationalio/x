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

func TestStemmer(t *testing.T) {
	testcases := []struct {
		Name         string
		Stemmer      quant.Stemmer
		InputPath    string
		ExpectedPath string
	}{
		{
			Name:         "NoOpStemmer",
			Stemmer:      &quant.NoOpStemmer{},
			InputPath:    "testdata/NoOpStemmer/voc.txt",
			ExpectedPath: "testdata/NoOpStemmer/voc.txt", // same as input
		},
		{
			Name:         "Porter2Stemmer",
			Stemmer:      &quant.Porter2Stemmer{},
			InputPath:    "testdata/Porter2Stemmer/voc.txt",
			ExpectedPath: "testdata/Porter2Stemmer/output.txt",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// Load 'input' data
			input, err := os.Open(tc.InputPath)
			assert.NotNil(t, input, "unexpected nil input file")
			assert.Nil(t, err, "error opening 'input' file")
			defer input.Close()

			// Load 'expected' data
			expected, err := os.Open(tc.InputPath)
			assert.NotNil(t, expected, "unexpected nil 'expected' file")
			assert.Nil(t, err, "error opening 'expected' file")
			defer expected.Close()

			// Scan each line of the input and compare to the output of the stemmer
			inputScanner := bufio.NewScanner(input)
			expectedScanner := bufio.NewScanner(expected)
			for inputScanner.Scan() && expectedScanner.Scan() {
				exp := expectedScanner.Text()
				act := tc.Stemmer.Stem(inputScanner.Text())
				assert.Equal(t, exp, act, fmt.Sprintf("wrong stem: expected '%s', got '%s'", exp, act))
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
