package quant

import (
	"regexp"
)

// ############################################################################
// Tokenizer interface
// ############################################################################

type Tokenizer interface {
	Tokenize(chunk string) (tokens []string, err error)
}

// ############################################################################
// Regex Expressions for Tokenizing
// ############################################################################

// Basic English alphanumeric tokenization. Does not account for special number
// formats or any words with punctuation in them.
//
//	`A-Za-z0-9`
const REGEX_ENGLISH_ALPHANUMERIC = `A-Za-z0-9`

// ############################################################################
// RegexTokenizer
// ############################################################################

// RegexTokenizer can be used to tokenize text; create with [NewRegexTokenizer].
type RegexTokenizer struct {
	// The [Language] to use for the [Tokenizer].
	lang Language
	// The regular expression to use for the [Tokenizer].
	regex string
}

// Returns a new [RegexTokenizer] instance. Defaults to [LanuageEnglish] and
// alphanumeric tokenization. Modified by passing [RegexTokenizerOption] functions
// into relevant function calls.
//
// Defaults:
// * Language: [LanguageEnglish]
// * Regex: [REGEX_ENGLISH_ALPHANUMERIC]
func NewRegexTokenizer(opts ...RegexTokenizerOption) *RegexTokenizer {
	// Set defaults
	tokenizer := &RegexTokenizer{
		lang:  LanuageEnglish,
		regex: REGEX_ENGLISH_ALPHANUMERIC,
	}

	// Set options
	for _, fn := range opts {
		fn(tokenizer)
	}

	return tokenizer
}

// Tokenizes a chunk of text using [regexp.Regexp.FindAllString].
func (t *RegexTokenizer) Tokenize(chunk string) (tokens []string, err error) {
	// Compile regexp
	var r *regexp.Regexp
	if r, err = regexp.Compile(t.regex); err != nil {
		return nil, err
	}

	// Tokenize with regex
	tokens = r.FindAllString(chunk, -1)

	return tokens, nil
}

// ############################################################################
// RegexTokenizerOption
// ############################################################################

// RegexTokenizerOption functions modify a [RegexTokenizer].
type RegexTokenizerOption func(t *RegexTokenizer)

// Returns a function which sets the [Language] to use with the [RegexTokenizer].
func WithLanguage(lang Language) RegexTokenizerOption {
	return func(t *RegexTokenizer) {
		t.lang = lang
	}
}

// Returns a function which sets the regular expression to use with the
// [RegexTokenizer].
func WithRegex(regex string) RegexTokenizerOption {
	return func(t *RegexTokenizer) {
		t.regex = regex
	}
}
