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

// ############################################################################
// Porter2Stemmer
// ############################################################################

// Ensure [Porter2Stemmer] meets the [Stemmer] interface requirements.
var _ Stemmer = &Porter2Stemmer{}

// Implements the Porter2 stemming algorithm.
type Porter2Stemmer struct {
	lang Language

	// Implementation function (set in [NewPorter2Stemmer])
	impl func(string) string
	// Word buffer
	word []rune
	// Pointer to the start of the word region R1
	p1 int
	// Pointer to the start of the word region R2
	p2 int
}

// Returns a new [Porter2Stemmer] which supports the [Language] given, or an
// error if the language is not supported.
func NewPorter2Stemmer(lang Language) (stemmer *Porter2Stemmer, err error) {
	// Setup the stemmer for the selected language
	stemmer = &Porter2Stemmer{lang: lang}
	switch stemmer.lang {
	case LanuageEnglish:
		// 30 runes is long enough for most English words
		stemmer.word = make([]rune, 30)
		// Use English stemmer
		stemmer.impl = stemmer.StemEnglish

	default: // unsupported language
		return nil, ErrLanguageNotSupported
	}

	return stemmer, nil
}

// Returns the [Porter2Stemmer]s configured [Language].
func (p *Porter2Stemmer) Language() Language {
	return p.lang
}

// Returns the stem for the selected word using the language-specific
// implementation set with [NewPorter2Stemmer].
func (p *Porter2Stemmer) Stem(word string) (stem string) {
	// The implementation was set in [NewPorter2Stemmer]
	return p.impl(word)
}
