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
	p.Step_0_English()
	p.Step_1a_English()
	p.Step_1b_English()
	p.Step_1c_English()
	p.Step_2_English()
	p.Step_3_English()
	p.Step_4_English()
	p.Step_5_English()

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
func (p *Porter2Stemmer) Step_0_English() {
	// Remove the longest suffix found in the suffixes
	p.removeSuffix(len(p.longestMatchingSuffix(0, len(p.word), "'", "'s", "'s'")))
}

// Performs step 1a of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) Step_1a_English() {
	// Find the longest suffix then perform the operation for it
	longest := p.longestMatchingSuffix(0, len(p.word),
		"sses",
		"ied",
		"ies",
		"us",
		"ss",
		"s",
	)

	// Perform the operation
	switch longest {
	case "":
		//No match

	case "sses":
		// Replace "sses" with "ss"
		p.replaceSuffix(len(longest), "ss")

	case "ied", "ies":
		if 4 < len(p.word) {
			// Replace by "i" if preceeded by more than one letter
			p.replaceSuffix(len(longest), "i")
		} else {
			// Replace with "ie" for words < 5 runes
			p.replaceSuffix(len(longest), "ie")
		}

	case "us", "ss":
		// Do nothing

	case "s":
		// Delete if the preceding word part contains a vowel not immediately
		// before the s
		for i := range p.word[:len(p.word)-2] {
			if p.isVowel(i) {
				p.removeSuffix(len(longest))
				return
			}
		}
	}
}

// Performs step 1b of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) Step_1b_English() {
	// Find the longest suffix then perform the operation for it
	longest := p.longestMatchingSuffix(0, len(p.word),
		"eed",
		"eedly",
		"ed",
		"edly",
		"ing",
		"ingly",
	)

	// Perform the operation
	switch longest {
	case "":
		//No match

	case "eed", "eedly":
		// Replace by "ee" if in R1
		if len(longest) < (len(p.word) - p.p1) {
			p.replaceSuffix(len(longest), "ee")
		}

	case "ed", "edly", "ing", "ingly":
		// For "ing", check if the word before the suffix is exactly one of the
		// exceptional cases
		if longest == "ing" {
			// If it's a non-vowel followed by "y", replace "y" and "ing" with
			// "ie", then go to step 1c.
			if len(p.word) == 5 && !p.isVowel(0) && p.word[1] == 'y' {
				p.replaceSuffix(4, "ie")
				return
			}
			// If it's exactly one of "inn", "out", "cann", "herr", "earr" or
			// "even".
			if 0 < len(p.longestMatchingSuffix(0, len(p.word)-3,
				"inn",  // inning
				"out",  // outing
				"cann", // canning
				"herr", // herring
				"earr", // earring
				"even", // evening
			)) {
				// Do nothing
				return
			}
		}

		// Delete if the preceeding word part contains a vowel
		for i := range p.word[:len(p.word)-len(longest)] {
			if p.isVowel(i) {
				p.removeSuffix(len(longest))
			}
		}

		// If the word ends "at", "bl" or "iz" add "e"
		if 0 < len(p.longestMatchingSuffix(0, len(p.word),
			"at",
			"bl",
			"iz",
		)) {
			p.appendSuffix("e")
			return
		}

		// If the word ends with a double preceded by something other than
		// exactly "a", "e", or "o" then remove the last letter
		if 3 <= len(p.word) {
			i := len(p.word) - 2
			preceeding_aeo := p.word[i-1] == 'a' || p.word[i-1] == 'e' || p.word[i-1] == 'o'
			if p.isDouble(i) && !preceeding_aeo {
				p.removeSuffix(1)
				return
			}
		}

		// If the word does not end with a double and is short, add "e"
		if p.isShortWord() {
			p.appendSuffix("e")
		}
	}
}

// Performs step 1c of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) Step_1c_English() {
	// Do nothing to 2 letter words here
	if len(p.word) == 2 {
		return
	}

	// Replace suffix "y" or "Y" by "i" if preceded by a non-vowel
	idx := len(p.word) - 1
	if (p.word[idx] == 'y' || p.word[idx] == 'Y') && !p.isVowel(idx-1) {
		p.replaceSuffix(1, "i")
	}
}

// Performs step 2 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) Step_2_English() {
	//TODO
}

// Performs step 3 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) Step_3_English() {
	//TODO
}

// Performs step 4 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) Step_4_English() {
	//TODO
}

// Performs step 5 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) Step_5_English() {
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

// Returns true if the suffix of the word buffer slice [start:end] matches the
// suffix runes provided.
func (p *Porter2Stemmer) hasSuffix(start, end int, suffix string) (matches bool) {
	// If the suffix is longer than the word range, it cannot match.
	if len(p.word[start:end]) < len(suffix) {
		return false
	}

	return slices.Equal(p.word[end-len(suffix):end], []rune(suffix))
}

// Returns the longest matching suffix of the word buffer slice [start:end].
func (p *Porter2Stemmer) longestMatchingSuffix(start, end int, suffixes ...string) (longest string) {
	for _, suffix := range suffixes {
		if p.hasSuffix(start, end, suffix) && len(longest) < len(suffix) {
			longest = suffix
		}
	}
	return longest
}

// Removes the last n runes from the word buffer.
func (p *Porter2Stemmer) removeSuffix(n int) {
	if n <= 0 {
		return
	}
	p.word = p.word[:len(p.word)-n]
	p.setRegions()
}

// Replaces the last n runes of the word buffer with the provided suffix.
func (p *Porter2Stemmer) replaceSuffix(n int, suffix string) {
	p.word = append(p.word[:len(p.word)-n], []rune(suffix)...)
	p.setRegions()
}

// Appends the word buffer with the provided suffix.
func (p *Porter2Stemmer) appendSuffix(suffix string) {
	p.word = append(p.word, []rune(suffix)...)
	p.setRegions()
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

// Returns true if the word buffer slice [:i] ends in a short syllable.
func (p *Porter2Stemmer) endsShortSyllable(i int) bool {
	switch p.lang {
	case LanuageEnglish:
		// Not enough runes to be a short syllable
		if i < 2 {
			return false
		}

		// When at the beginning of the word: a vowel followed by a non-vowel
		if i == 2 {
			return p.isVowel(0) && !p.isVowel(1)
		}

		// i >= 3:

		// If it has a "past" suffix then it is a short syllable
		if p.hasSuffix(0, i, "past") {
			return true
		}

		// A vowel, followed by a non-vowel other than w, x or Y, and preceded by a non-vowel
		endIs_wxY := p.word[i-1] == 'w' || p.word[i-1] == 'x' || p.word[i-1] == 'Y'
		return p.isVowel(i-2) && (!p.isVowel(i-1) && !endIs_wxY) && !p.isVowel(i-3)
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

		// A word is short if it ends in a short syllable
		return p.endsShortSyllable(len(p.word))
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
