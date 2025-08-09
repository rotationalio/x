package quant

import (
	"regexp"
)

/*
tokens.go provides tokenization functionality.

Types:
* Tokenizer struct
* TokenizerOption func(t *Tokenizer)

Functions:
* NewTokenizer() *Tokenizer
* Tokenize(corpus string, lang Language) (tokens []string, err error)
* WithLanguage(lang Language) TokenizerOption
* WithRegex(regex string) TokenizerOption
*/

// ############################################################################
// Regex Expressions for Tokenizing
// ############################################################################

// 26 uppercase, 26 lowercase, and 10 digits
const REGEX_ENGLISH_ALPHANUMERIC = `A-Za-z0-9`

// ############################################################################
// Tokenizer
// ############################################################################

// Tokenizer can be used to tokenize text; create with [NewTokenizer].
type Tokenizer struct {
	// The [Language] to use for the [Tokenizer].
	lang Language
	// The regular expression to use for the [Tokenizer].
	regex string
}

// Returns a new [Tokenizer] instance. Defaults to [LanuageEnglish] and
// alphanumeric tokenization. Modified by passing [TokenizerOption] functions
// into relevant function calls.
func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		lang:  LanuageEnglish,
		regex: REGEX_ENGLISH_ALPHANUMERIC,
	}
}

// Tokenizes a string. Does not modify the text chunk before tokenizing.
func (t *Tokenizer) Tokenize(chunk string, opts ...TokenizerOption) (tokens []string, err error) {
	// Set Tokenizer options
	for _, fn := range opts {
		fn(t)
	}

	// Compile and tokenize
	var r *regexp.Regexp
	if r, err = regexp.Compile(t.regex); err != nil {
		return nil, err
	}
	tokens = r.FindAllString(chunk, -1)

	return tokens, nil
}

// ############################################################################
// TokenizerOptions
// ############################################################################

// TokenizerOption functions modify a [Tokenizer].
type TokenizerOption func(t *Tokenizer)

// Returns a function which sets the [Language] to use with the [Tokenizer].
func WithLanguage(lang Language) TokenizerOption {
	return func(t *Tokenizer) {
		t.lang = lang
	}
}

// Returns a function which sets the regular expression to use with the
// [Tokenizer].
func WithRegex(regex string) TokenizerOption {
	return func(t *Tokenizer) {
		t.regex = regex
	}
}
