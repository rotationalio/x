package quant

import (
	"slices"
	"strings"
)

// ############################################################################
// Porter2Stemmer English Steps
// ############################################################################

// Returns the stem for the selected English word using the English Porter2
// algorithm. Whitespace will be trimmed and the stem will be returned in
// all lowercase.
func (p *Porter2Stemmer) StemEnglish(word string) (stem string) {
	// Lowercase and remove any whitespace
	word = strings.TrimSpace(strings.ToLower(word))

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

	// Remove initial apostrophe if present
	if p.isApostrophe(0) {
		p.word = p.word[1:]
	}

	// Set initial y, or y after a vowel, to Y
	for i := range p.word {
		if p.word[i] == 'y' && (i == 0 || p.isVowel(i-1)) {
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

	// Uncapitalize Y's
	for i := range p.word {
		if p.word[i] == 'Y' {
			p.word[i] = 'y'
		}
	}

	// Return the completed stem as a string
	return string(p.word)
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

// Performs step 0 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_0_English() {
	// Remove the longest suffix found in the suffixes
	p.removeSuffix(len(p.longestMatchingSuffix(0, len(p.word), "'", "'s", "'s'")))
}

// Performs step 1a of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_1a_English() {
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
		chkEnd := len(p.word) - 2
		if 0 < chkEnd {
			for i := range p.word[:chkEnd] {
				if p.isVowel(i) {
					p.removeSuffix(len(longest))
					return
				}
			}
		}
	}
}

// Performs step 1b of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_1b_English() {
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
		// Exceptions for "eed" words; if found do not perform any action
		if pfx := p.hasAnyPrefix(
			"proc", // proceed
			"exc",  // exceed
			"succ", // succeed
		); pfx != nil {
			// Do nothing
			return
		}

		// Replace by "ee" if in R1
		if len(longest) <= (len(p.word) - p.p1) {
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

			// Invariant exceptions
			longIng := p.longestMatchingSuffix(0, len(p.word)-len(longest),
				"inn",  // inning
				"out",  // outing
				"cann", // canning
				"herr", // herring
				"earr", // earring
				"even", // evening
			)
			// Must be EXACTLY one of the above
			if longIng != "" && p.hasSuffix(0, len(longIng), longIng) {
				// Do nothing
				return
			}
		}

		// Delete if the preceeding word part contains a vowel
		idxEnd := len(p.word) - len(longest)
		hadVowel := false
		for i := range p.word[:idxEnd] {
			if p.isVowel(i) {
				p.removeSuffix(len(longest))
				hadVowel = true
				break
			}
		}

		// Then, after the deletion:
		if hadVowel {
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
			idxDbl := len(p.word) - 2
			if p.isDouble(idxDbl) {
				if len(p.word) == 3 {
					if p.word[0] != 'a' && p.word[0] != 'e' && p.word[0] != 'o' {
						p.removeSuffix(1)
					}
				} else if 3 < len(p.word) {
					p.removeSuffix(1)
				}
				return
			}

			// If the word does not end with a double and is short, add "e"
			if p.isShortWord() {
				p.appendSuffix("e")
			}
		}
	}
}

// Performs step 1c of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_1c_English() {
	// Do nothing to 2 letter words here
	if len(p.word) == 2 {
		return
	}

	// Replace suffix "y" or "Y" by "i" if preceded by a non-vowel
	end := len(p.word) - 1
	if (p.word[end] == 'y' || p.word[end] == 'Y') && !p.isVowel(end-1) {
		p.replaceSuffix(1, "i")
	}
}

// Performs step 2 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_2_English() {
	// Search for the longest among the following suffixes
	longest := p.longestMatchingSuffix(0, len(p.word),
		"tional",
		"enci",
		"anci",
		"abli",
		"entli",
		"izer", "ization",
		"ational", "ation", "ator",
		"alism", "aliti", "alli",
		"fulness",
		"ousli", "ousness",
		"iveness", "iviti",
		"biliti", "bli",
		"ogist",
		"ogi",
		"fulli",
		"lessli",
		"li",
	)

	// If a suffix is not found or it is not in R1, do nothing
	if longest == "" || !p.hasSuffix(p.p1, len(p.word), longest) {
		return
	}

	// Perform the operation
	switch longest {
	case "tional":
		p.replaceSuffix(len(longest), "tion")

	case "enci":
		p.replaceSuffix(len(longest), "ence")

	case "anci":
		p.replaceSuffix(len(longest), "ance")

	case "abli":
		p.replaceSuffix(len(longest), "able")

	case "entli":
		p.replaceSuffix(len(longest), "ent")

	case "izer", "ization":
		p.replaceSuffix(len(longest), "ize")

	case "ational", "ation", "ator":
		p.replaceSuffix(len(longest), "ate")

	case "alism", "aliti", "alli":
		p.replaceSuffix(len(longest), "al")

	case "fulness":
		p.replaceSuffix(len(longest), "ful")

	case "ousli", "ousness":
		p.replaceSuffix(len(longest), "ous")

	case "iveness", "iviti":
		p.replaceSuffix(len(longest), "ive")

	case "biliti", "bli":
		p.replaceSuffix(len(longest), "ble")

	case "ogist":
		p.replaceSuffix(len(longest), "og")

	case "ogi":
		// Only if preceded by "l"
		idx := len(p.word) - len(longest) - 1
		if 4 <= len(p.word) && p.word[idx] == 'l' {
			p.replaceSuffix(len(longest), "og")
		}

	case "fulli":
		p.replaceSuffix(len(longest), "ful")

	case "lessli":
		p.replaceSuffix(len(longest), "less")

	case "li":
		// Only if preceded by a valid "li-ending"
		idx := len(p.word) - len(longest) - 1
		if 0 <= idx && p.isValidLiEnding(idx) {
			p.removeSuffix(len(longest))
		}
	}
}

// Performs step 3 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_3_English() {
	// Search for the longest among the following suffixes
	longest := p.longestMatchingSuffix(0, len(p.word),
		"tional",
		"ational",
		"alize",
		"icate", "iciti", "ical",
		"ful", "ness",
		"ative",
	)

	// If a suffix is not found or it is not in R1, do nothing
	if longest == "" || !p.hasSuffix(p.p1, len(p.word), longest) {
		return
	}

	// Perform the operation
	switch longest {
	case "tional":
		p.replaceSuffix(len(longest), "tion")

	case "ational":
		p.replaceSuffix(len(longest), "ate")

	case "alize":
		p.replaceSuffix(len(longest), "al")

	case "icate", "iciti", "ical":
		p.replaceSuffix(len(longest), "ic")

	case "ful", "ness":
		p.removeSuffix(len(longest))

	case "ative":
		// Remove only if in R2
		if p.hasSuffix(p.p2, len(p.word), longest) {
			p.removeSuffix(len(longest))
		}
	}
}

// Performs step 4 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_4_English() {
	// Search for the longest among the following suffixes
	longest := p.longestMatchingSuffix(0, len(p.word),
		"al", "ance", "ence", "er", "ic", "able", "ible", "ant", "ement", "ment", "ent", "ism", "ate", "iti", "ous", "ive", "ize",
		"ion",
	)

	// If a suffix is not found or it is not in R2, do nothing
	if longest == "" || !p.hasSuffix(p.p2, len(p.word), longest) {
		return
	}

	// Perform the operation
	switch longest {
	case "al", "ance", "ence", "er", "ic", "able", "ible", "ant", "ement", "ment", "ent", "ism", "ate", "iti", "ous", "ive", "ize":
		p.removeSuffix(len(longest))

	case "ion":
		// Delete if precedded by "s" or "t"
		idx := len(p.word) - len(longest) - 1
		if 0 <= idx && (p.word[idx] == 's' || p.word[idx] == 't') {
			p.removeSuffix(len(longest))
		}
	}
}

// Performs step 5 of the Porter2 English stemmer algorithm on the word buffer.
func (p *Porter2Stemmer) step_5_English() {
	// Delete "e" if in R2
	if p.hasSuffix(p.p2, len(p.word), "e") {
		p.removeSuffix(1)
		return
	}

	// Delete "e" if in R1 and not preceeded by a short syllable
	if p.hasSuffix(p.p1, len(p.word), "e") && !p.endsShortSyllable(len(p.word)-1) {
		p.removeSuffix(1)
		return
	}

	// Delete "l" if in R2 and preceeded by an "l"
	if p.hasSuffix(p.p2-1, len(p.word), "ll") {
		p.removeSuffix(1)
		return
	}
}

// ############################################################################
// Porter2Stemmer Helpers
// ############################################################################

// Sets the R1 and R2 region pointers for the current word buffer.
func (p *Porter2Stemmer) setRegions() {
	// Find R1
	p.p1 = p.findRegionStart(0)

	// R1 exceptions for over-stemmed words; if found then set R1 to the end of
	// the prefix found
	if pfx := p.hasAnyPrefix(
		"gener",   // generate/general/generic/generous
		"commun",  // communication/communism/community
		"arsen",   // arsenic/arsenal
		"past",    // past/paste
		"univers", // universe/universal/university
		"later",   // lateral/later
		"emerg",   // emerge/emergency
		"organ",   // organ/organic/organize
	); pfx != nil {
		p.p1 = len(pfx)
	}

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
		for i := range p.word[start : len(p.word)-1] {
			if p.isVowel(start+i) && !p.isVowel(start+i+1) {
				// The region starts at i+2, after the pattern 'Vnv'
				// Note: this may be the "null" region, which is fine
				return start + i + 2
			}
		}
	}

	// Defaults to "null" region after the word
	return len(p.word)
}

// Returns the first prefix argument that matches the word buffer prefix. If
// none are found, returns nil.
func (p *Porter2Stemmer) hasAnyPrefix(prefixes ...string) (prefix []rune) {
	for _, pfx := range prefixes {
		if len(pfx) <= len(p.word) {
			prefix := []rune(pfx)
			if slices.Equal(p.word[:len(prefix)], prefix) {
				return prefix
			}
		}
	}
	return nil
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
func (p *Porter2Stemmer) longestMatchingSuffix(start, end int, suffixes ...string) string {
	// There can't be a match
	if start == end {
		return ""
	}

	// Sort the suffixes by length then lexicographically
	slices.SortFunc(suffixes, func(a, b string) int {
		if len(a) == len(b) {
			return strings.Compare(a, b)
		}
		return len(b) - len(a)
	})

	// Find the first matching suffix
	for _, suffix := range suffixes {
		if p.hasSuffix(start, end, suffix) {
			return suffix
		}
	}
	return ""
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

// ############################################################################
// Porter2Stemmer Rule Definitions
// ############################################################################

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
