package lorem_test

import (
	"regexp"
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/lorem"
)

// sentenceRegexp matches a sentence that starts with an uppercase letter and ends with
// one of the allowed terminal punctuation marks.
var sentenceRegexp = regexp.MustCompile(`^[A-Z].*[.?!]$`)

func TestDeterministic(t *testing.T) {
	// Two generators with the same seed must produce identical output.
	a := lorem.New(42)
	b := lorem.New(42)
	assert.Equal(t, a.Document(), b.Document())

	// Different seeds should (overwhelmingly likely) produce different output.
	c := lorem.New(99)
	assert.NotEqual(t, a.Document(), c.Document())
}

func TestWords(t *testing.T) {
	ello := lorem.New(42)
	for i := 0; i < 100; i++ {
		words := ello.Words()
		assert.GreaterEqual(t, 4, len(words))
		assert.LessEqual(t, 12, len(words))
		for _, w := range words {
			assert.True(t, w != "", "word should not be empty")
		}
	}

	// Random output should vary across calls.
	assert.True(t, varies(func() string {
		return strings.Join(ello.Words(), " ")
	}), "expected Words output to vary")
}

func TestSentences(t *testing.T) {
	ello := lorem.New(42)
	for i := 0; i < 100; i++ {
		sentences := ello.Sentences()
		assert.GreaterEqual(t, 3, len(sentences))
		assert.LessEqual(t, 8, len(sentences))
		for _, s := range sentences {
			assert.Regexp(t, sentenceRegexp, s)

			// Each sentence should contain between MinWords and MaxWords words.
			n := len(strings.Fields(s))
			assert.GreaterEqual(t, 4, n)
			assert.LessEqual(t, 12, n)
		}
	}

	assert.True(t, varies(func() string {
		return strings.Join(ello.Sentences(), " ")
	}), "expected Sentences output to vary")
}

func TestParagraphs(t *testing.T) {
	ello := lorem.New(42)
	for i := 0; i < 100; i++ {
		paragraphs := ello.Paragraphs()
		assert.GreaterEqual(t, 2, len(paragraphs))
		assert.LessEqual(t, 5, len(paragraphs))
		for _, p := range paragraphs {
			assert.True(t, p != "", "paragraph should not be empty")
		}
	}

	assert.True(t, varies(func() string {
		return strings.Join(ello.Paragraphs(), "\n")
	}), "expected Paragraphs output to vary")
}

func TestDocument(t *testing.T) {
	ello := lorem.New(42)
	for i := 0; i < 100; i++ {
		doc := ello.Document()
		assert.True(t, doc != "", "document should not be empty")

		// A document is one or more paragraphs joined by blank lines.
		paragraphs := strings.Split(doc, "\n\n")
		assert.GreaterEqual(t, 2, len(paragraphs))
		assert.LessEqual(t, 5, len(paragraphs))
	}

	assert.True(t, varies(func() string {
		return ello.Document()
	}), "expected Document output to vary")
}

func TestCustomConfig(t *testing.T) {
	ello := &lorem.Ipsum{
		MinWords: 2,
		MaxWords: 3,
		MinSents: 1,
		MaxSents: 1,
		MinParas: 1,
		MaxParas: 1,
	}

	for i := 0; i < 100; i++ {
		words := ello.Words()
		assert.GreaterEqual(t, 2, len(words))
		assert.LessEqual(t, 3, len(words))

		assert.Len(t, ello.Sentences(), 1)
		assert.Len(t, ello.Paragraphs(), 1)
	}
}

func TestZeroValue(t *testing.T) {
	// A zero-value Ipsum should apply defaults and seed itself without panicking.
	ello := &lorem.Ipsum{}
	assert.GreaterEqual(t, 4, len(ello.Words()))
	assert.LessEqual(t, 12, len(ello.Words()))
	assert.GreaterEqual(t, 3, len(ello.Sentences()))
	assert.GreaterEqual(t, 2, len(ello.Paragraphs()))
	assert.True(t, ello.Document() != "", "document should not be empty")
}

func TestPackageLevel(t *testing.T) {
	assert.True(t, len(lorem.Words()) > 0, "expected words")
	assert.True(t, len(lorem.Sentences()) > 0, "expected sentences")
	assert.True(t, len(lorem.Paragraphs()) > 0, "expected paragraphs")
	assert.True(t, lorem.Document() != "", "expected document")
}

// varies returns true if calling fn repeatedly produces at least two distinct results,
// confirming that random output is actually being generated.
func varies(fn func() string) bool {
	first := fn()
	for i := 0; i < 100; i++ {
		if fn() != first {
			return true
		}
	}
	return false
}
