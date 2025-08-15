package quant

import (
	"slices"
	"strings"
)

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

// Returns the stem for the selected word using the language-specific
// implementation set with [NewPorter2Stemmer].
func (p *Porter2Stemmer) Stem(word string) (stem string) {
	// The implementation was set in [NewPorter2Stemmer]
	return p.impl(word)
}

// ############################################################################
// Porter2Stemmer English Steps
// ############################################################################

// Returns the stem for the selected English word using the English Porter2
// algorithm. Whitespace will be trimmed and the stem will be returned in
// all lowercase.
func (p *Porter2Stemmer) StemEnglish(word string) (stem string) {
	// Lowercase and remove any whitespace
	word = strings.TrimSpace(strings.ToLower(word))

	// Remove initial apostrophes in word
	for p.isApostrophe(0) && len(p.word) != 0 {
		p.word = p.word[1:]
	}

	// If the word has two letters or less, leave it as it is
	if len(word) <= 2 {
		return word
	}

	// Return exceptions immediately
	if stem = p.porter2Exceptions(word); stem != "" {
		return stem
	}

	// Put the word into the word buffer
	p.word = []rune(word)

	// Set initial y, or y after a vowel, to Y
	for i := range p.word {
		if p.word[i] == 'y' && (i == 0 || !p.isVowel(i-1)) {
			p.word[i] = 'Y'
		}
	}

	// Setup the regions R1 and R2
	p.setRegions()

	// Run Porter2 algorithm steps in order
	p.step_0_English()
	p.step_1a_English()
	p.step_1b_English()
	p.step_1c_English()
	p.step_2_English()
	p.step_3_English()
	p.step_4_English()
	p.step_5_English()

	// TODO: post processing

	// Uncapitalize Y's
	for i := range p.word {
		if p.word[i] == 'Y' {
			p.word[i] = 'y'
		}
	}

	// Return the completed stem as a string
	return string(p.word)
}

// Performs step 0 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_0_English() {
	//TODO
}

// Performs step 1a of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_1a_English() {
	//TODO
}

// Performs step 1b of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_1b_English() {
	//TODO
}

// Performs step 1c of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_1c_English() {
	//TODO
}

// Performs step 2 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_2_English() {
	//TODO
}

// Performs step 3 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_3_English() {
	//TODO
}

// Performs step 4 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_4_English() {
	//TODO
}

// Performs step 5 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_5_English() {
	//TODO
}

// ############################################################################
// Porter2Stemmer Helpers
// NOTE: Functions which take indexes into the word are not guaranteed to check
//       the index is valid, so the indexes must be checked before passing!
// ############################################################################

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
	// Ensure the start index is at least the penultimate rune
	if start < len(p.word)-2 {
		// Find the word's next non-vowel after a vowel starting at the index given
		for i := range p.word[start:] {
			if p.isVowel(i) && !p.isVowel(i+1) {
				// The region starts at i+2, after the pattern 'Vnv'
				// Note: this may be the "null" region, which is fine
				return start + i + 2
			}
		}
	}

	// Defaults to "null" region after the word
	return len(p.word)
}

// Returns true if rune in the word buffer at index i is a vowel as defined in
// the Porter2 algorithm.
func (p *Porter2Stemmer) isVowel(i int) bool {
	switch p.lang {
	case LanuageEnglish:
		switch p.word[i] {
		// We will only count lower case vowels because capital Y indicates
		// that particular Y is defined as a non-vowel.
		case 'a', 'e', 'i', 'o', 'u', 'y':
			return true
		}
	}
	return false
}

// Returns true if runes in the word buffer slice [i:i+1] are a double as
// defined in the Porter2 algorithm.
func (p *Porter2Stemmer) isDouble(i int) bool {
	switch p.lang {
	case LanuageEnglish:
		switch p.word[i] {
		case 'b', 'd', 'f', 'g', 'm', 'n', 'p', 'r', 't':
			// It's a double if i and i+1 are both the same and one of above
			if p.word[i] == p.word[i+1] {
				return true
			}
		}
	}
	return false
}

// Returns true if the rune in the word buffer at index i is a valid li-ending
// as defined in the Porter2 algorithm.
func (p *Porter2Stemmer) isValidLiEnding(i int) bool {
	switch p.lang {
	case LanuageEnglish:
		switch p.word[i] {
		case 'c', 'd', 'e', 'g', 'h', 'k', 'm', 'n', 'r', 't':
			// It's a valid li- ending if it's one of the runes above
			return true
		}
	}
	return false
}

// Returns true if runes in the word buffer slice [i-1:i+1] (if i is 0 then
// [i:i+1]) are a short syllable as defined in the Porter2 algorithm. This
// function does not include the case in part (c) for "past", which is covered
// sepatately in the function [Porter2Stemmer.isShortSyllablePast] and must
// be called separately to cover both cases for ease of implementation.
func (p *Porter2Stemmer) isShortSyllable2Rune(i int) bool {
	switch p.lang {
	case LanuageEnglish:
		// The rune at i must be a vowel
		iIsV := p.isVowel(i)

		// The rune at i+1 must be a non-vowel that isn't w, x, or Y
		iPlusIsV := p.isVowel(i + 1)
		iPlusIs_wxY := p.word[i+1] == 'w' || p.word[i+1] == 'x' || p.word[i+1] == 'Y'

		// The run at i-1 must be a non-vowel, or i must be at the start of the
		// word
		iMinusIsV := false // if it's at the start of the word
		if i != 0 {
			iMinusIsV = p.isVowel(i - 1)
		}

		return iIsV && !iPlusIsV && !iPlusIs_wxY && !iMinusIsV
	}
	return false
}

// Returns true if runes in the word buffer slice [i:i+3] equal "past", which
// covers part (c) of the short syllable definition according to the Porter2
// algorithm.
func (p *Porter2Stemmer) isShortSyllablePast(i int) bool {
	switch p.lang {
	case LanuageEnglish:
		return slices.Equal(p.word[i:], []rune("past"))
	}
	return false
}

// Returns true if the word is a short word as defined in the Porter2 algorithm.
func (p *Porter2Stemmer) isShortWord() bool {
	switch p.lang {
	case LanuageEnglish:
		// If R1 is not the "null" region at the end of the word, it is not a
		// short word
		if p.p1 != len(p.word) {
			return false
		}

		// Check if the word ends in "past"
		endsPast := false
		if 4 <= len(p.word) {
			endsPast = p.isShortSyllablePast(len(p.word) - 4)
		}

		// A word is short if it ends in a short syllable
		return p.isShortSyllable2Rune(len(p.word)-2) || endsPast
	}
	return false

}

// Returns true if the rune at word index i is an apostrophe of some type.
// Defined as one of: '\u0027', '\u2019', '\u2018', '\u201B'
func (p *Porter2Stemmer) isApostrophe(i int) bool {
	switch p.word[i] {
	case '\u0027', '\u2019', '\u2018', '\u201B':
		return true
	}
	return false
}

// If word is an exception, then a stem for it will be returned otherwise the
// null string indicates it is not exceptional.
func (p *Porter2Stemmer) porter2Exceptions(word string) string {
	switch p.lang {
	case LanuageEnglish:
		switch word {
		// Porter2 exceptions:
		case "skis":
			return "ski"
		case "skies":
			return "sky"
		case "idly":
			return "idl"
		case "gently":
			return "gentl"
		case "ugly":
			return "ugli"
		case "early":
			return "earli"
		case "only":
			return "onli"
		case "singly":
			return "singl"
		//Porter2 invariants:
		case "sky", "news", "howe", "atlas", "cosmos", "bias", "andes":
			return word
		}
	}
	return ""
}
