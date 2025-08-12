package quant

// ############################################################################
// Stemmer interface
// ############################################################################

// Interface for a stemming algorithm's implementation.
type Stemmer interface {
	// Returns true if the language can be stemmed by the implemented Stemmer.
	CanStem(lang Language) (supported bool)

	// Returns the stem for the input word.
	Stem(word string) (stem string)
}

// ############################################################################
// NoOpStemmer
// ############################################################################

var _ Stemmer = &NoOpStemmer{}

// NoOpStemmer does no stemming and returns any input without changes.
type NoOpStemmer struct{}

// Supports all languages.
func (p *NoOpStemmer) CanStem(lang Language) bool {
	return true
}

// Returns the input without changes.
func (p *NoOpStemmer) Stem(word string) (stem string) {
	return word
}

// ############################################################################
// Porter2Stemmer
// ############################################################################

var _ Stemmer = &Porter2Stemmer{}

// Implements the Porter2 stemming algorithm.
type Porter2Stemmer struct{}

// Supported languages for the Porter2Stemmer are: [LanguageEnglish]
func (p *Porter2Stemmer) CanStem(lang Language) bool {
	if lang.In(LanuageEnglish) {
		return true
	}
	return false
}

// Supported languages for the Porter2Stemmer are: [LanguageEnglish]
func (p *Porter2Stemmer) Stem(word string) (stem string) {
	//TODO: implement it

	return stem
}
