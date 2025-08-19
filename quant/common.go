package quant

import (
	"slices"
	"strings"
)

// ############################################################################
// Language "enum"
// ############################################################################

type Language uint16

const (
	LanguageUnknown = iota
	LanuageEnglish
)

// Returns True if the argument [Language]s contains this language.
func (l Language) In(langs ...Language) bool {
	return slices.Contains(langs, l)
}

// ############################################################################
// Helpers
// ############################################################################

// Sorts the slice of strings by length (longest to shortest) and sorts
// lexicographically for equal length strings.
func SortLengthAndLexicographically(l []string) {
	slices.SortFunc(l, func(a, b string) int {
		if len(a) == len(b) {
			return strings.Compare(a, b)
		}
		return len(b) - len(a)
	})
}

// ############################################################################
// Stop Words
// ############################################################################

// Returns true if word is a stop word for the given language. If the language
// is not implemented this function will return false.
func IsStopWord(word string, lang Language) bool {
	switch lang {
	case LanuageEnglish:
		return slices.Contains(StopWordsEnglish[:], word)
	}
	return false
}

// Stop words for [LanguageEnglish]. Extracted from the [snowballstem.org stop
// words list on GitHub], which is BSD 3-clause licensed.
//
// [snowballstem.org stop words list on GitHub]: https://github.com/snowballstem/snowball-website/blob/1f7a15d10924fb38f90519d95a875342cf3e87ba/algorithms/english/stop.txt
var StopWordsEnglish = [174]string{
	"a", "about", "above", "after", "again", "against", "all", "am", "an",
	"and", "any", "are", "aren't", "as", "at", "be", "because", "been",
	"before", "being", "below", "between", "both", "but", "by", "can't",
	"cannot", "could", "couldn't", "did", "didn't", "do", "does", "doesn't",
	"doing", "don't", "down", "during", "each", "few", "for", "from", "further",
	"had", "hadn't", "has", "hasn't", "have", "haven't", "having", "he", "he'd",
	"he'll", "he's", "her", "here", "here's", "hers", "herself", "him",
	"himself", "his", "how", "how's", "i", "i'd", "i'll", "i'm", "i've", "if",
	"in", "into", "is", "isn't", "it", "it's", "its", "itself", "let's", "me",
	"more", "most", "mustn't", "my", "myself", "no", "nor", "not", "of", "off",
	"on", "once", "only", "or", "other", "ought", "our", "ours", "ourselves",
	"out", "over", "own", "same", "shan't", "she", "she'd", "she'll", "she's",
	"should", "shouldn't", "so", "some", "such", "than", "that", "that's",
	"the", "their", "theirs", "them", "themselves", "then", "there", "there's",
	"these", "they", "they'd", "they'll", "they're", "they've", "this",
	"those", "through", "to", "too", "under", "until", "up", "very", "was",
	"wasn't", "we", "we'd", "we'll", "we're", "we've", "were", "weren't",
	"what", "what's", "when", "when's", "where", "where's", "which", "while",
	"who", "who's", "whom", "why", "why's", "with", "won't", "would",
	"wouldn't", "you", "you'd", "you'll", "you're", "you've", "your", "yours",
	"yourself", "yourselves",
}
