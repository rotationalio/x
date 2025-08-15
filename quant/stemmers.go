package quant

// ############################################################################
// Stemmer interface
// ############################################################################

// Interface for a stemming algorithm's implementation.
type Stemmer interface {
	// Returns the stem of the input word.
	Stem(word string) (stem string)
}

// ############################################################################
// NoOpStemmer
// ############################################################################

// Ensure [NoOpStemmer] meets the [Stemmer] interface requirements.
var _ Stemmer = &NoOpStemmer{}

// NoOpStemmer does no stemming and returns any input without changes.
type NoOpStemmer struct{}

// Returns the input without changes.
func (p *NoOpStemmer) Stem(word string) (stem string) {
	return word
}
