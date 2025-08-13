package quant

import "strings"

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
// Porter2Stemmer General
// ############################################################################

// Ensure [Porter2Stemmer] meets the [Stemmer] interface requirements.
var _ Stemmer = &Porter2Stemmer{}

// Implements the Porter2 stemming algorithm.
type Porter2Stemmer struct {
	// Language for this stemmer (set in [NewPorter2Stemmer])
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

	//ENGLISH
	case LanuageEnglish:
		// 30 runes is long enough for most English words
		stemmer.word = make([]rune, 30)

		// Use English stemmer
		stemmer.impl = stemmer.StemEnglish

	//UNSUPPORTED
	default:
		return nil, ErrLanguageNotSupported
	}

	return stemmer, nil
}

// Returns the stem for the selected word using the language-specific
// implementation set with [NewPorter2Stemmer].
func (p *Porter2Stemmer) Stem(word string) (stem string) {
	// The implementation was set in [NewPorter2Stemmer]
	return p.impl(word)
}

// ############################################################################
// Porter2Stemmer English Steps
// ############################################################################

// Returns the stem for the selected English word.
func (p *Porter2Stemmer) StemEnglish(word string) (stem string) {
	// Return 2-letter words immediately
	if len(word) <= 2 {
		return word
	}

	// Return stop words immediately
	//TODO

	// Return special cases immediately
	//TODO

	// Setup for stemming
	p.setupEnglish(word)

	// Run Porter2 algorithm steps in order
	//TODO

	// Return the completed stem as a string
	return string(p.word)
}

// ############################################################################
// Porter2Stemmer English Helpers
// ############################################################################

// Sets up the English [Porter2Stemmer] with the given word.
func (p *Porter2Stemmer) setupEnglish(word string) {
	// Put the word into the word buffer (lowercase and remove whitespace)
	p.word = []rune(strings.TrimSpace(strings.ToLower(word)))

	// Setup the regions R1 and R2
	p.setRegions()
}

// Returns true if rune in the word buffer at index i is a vowel as defined in
// the Porter2 algorithm.
func (p *Porter2Stemmer) isVowel(i int) bool {
	switch p.word[i] {
	case 'a', 'e', 'i', 'o', 'u', 'y':
		return true
	default:
		return false
	}
}

// Returns true if runes in the word buffer slice [i:i+1] are a double as
// defined in the Porter2 algorithm.
func (p *Porter2Stemmer) isDouble(i int) bool {
	switch p.word[i] {
	case 'b', 'd', 'f', 'g', 'm', 'n', 'p', 'r', 't':
		if p.word[i] == p.word[i+1] {
			// It's a double!
			return true
		}
		// Runes were not a pair
		return false
	default:
		// Not the correct rune(s) for a double
		return false
	}
}

// Returns true if the rune in the word buffer at index i is a valid li-ending
// as defined in the Porter2 algorithm.
func (p *Porter2Stemmer) isValidLiEnding(i int) bool {
	switch p.word[i] {
	case 'c', 'd', 'e', 'g', 'h', 'k', 'm', 'n', 'r', 't':
		return true
	default:
		return false
	}
}

// TODO: docs
func (p *Porter2Stemmer) isShortSyllable(i int) bool {
	// TODO: "Define a short syllable in a word as either..."
	return true
}

// Sets the R1 and R2 region pointers for the current word buffer.
func (p *Porter2Stemmer) setRegions() {
	// Find R1
	p.p1 = p.findRegionStart(0)
	// Find R2
	p.p2 = p.findRegionStart(p.p1)
}

// Finds the start of the next region from the start index in the word buffer.
// Regions are defined in the Porter2 stemmer algorithm as the region after the
// next vowel followed by a non-vowel, or the end of the word.
func (p *Porter2Stemmer) findRegionStart(start int) int {
	// The word end is maximum return value
	wordEnd := len(p.word) - 1
	regionStart := wordEnd

	// Find the next non-vowel after a vowel starting at the index given
	for i := range p.word[start:] {
		if p.isVowel(i) && !p.isVowel(i+1) {
			// The region starts AFTER the pattern above
			regionStart = start + i + 2

			// Region start cannot be after the end of the word
			if wordEnd < regionStart {
				return wordEnd
			}
			return regionStart
		}
	}

	// Defaults to the end of the word
	return wordEnd
}
