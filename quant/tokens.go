package quant

import (
	"regexp"
)

/*
tokens.go provides tokenization, stemming/lemmaization, and count functionality.

TODO: finalize this documentation block

Types:
* None

Functions:
* `Tokenize(corpus string, lang Language) (tokens []string, err error)`
* `TypeCount(chunk string, options ...Options)`
* `TypeCountTokens(tokens []string, tokenModifiers ...StringModifier) (types map[string]int64)`
*/

// ############################################################################
// Tokenize
// ############################################################################

// TODO: docs
type Tokenizer struct {
	lang      Language //TODO: TokenizerOption
	regexExpr string   //TODO: TokenizerOption
}

// TODO docs
type TokenizerOption func(args ...any) Tokenizer //TODO: fix args?

// Tokenizes a string (naively) by grouping alphanumeric characters, ignoring
// non-alphanumeric characters. Does not modify the corpus before tokenizing.
// TODO: document defaults here
func (t *Tokenizer) Tokenize(chunk string, opts ...TokenizerOption) (tokens []string, err error) {
	//TODO: do opts

	var (
		expr string
		r    *regexp.Regexp
	)

	// Define the regexp expression by language
	switch t.lang {
	case LanuageEnglish:
		// 26 uppercase, 26 lowercase, and 10 digits
		expr = `A-Za-z0-9` //TODO: regex or function for tokenization is provided as option?
	default:
		// Unsupported language
		return nil, ErrLanguageNotSupported
	}

	// Compile and tokenize
	if r, err = regexp.Compile(expr); err != nil {
		return nil, err
	}
	tokens = r.FindAllString(chunk, -1)

	return tokens, nil
}

// ############################################################################
// TypeCountTokens
// ############################################################################

// Returns a map of type strings and their counts. For each token, all of the
// modifiers provided will be performed before counting. An example of a
// [StringModifier] would be the function [strings.ToLower] or many others in
// the Go [strings] package.
// TODO document defaults here
func (t *Tokenizer) TypeCountTokens(tokens []string, opts ...TokenizerOption) (types map[string]int64, err error) {
	// Make the types map (variable sz was selected arbitrarily)
	sz := len(tokens) / 4
	types = make(map[string]int64, sz)

	// TODO: count the tokens

	return types, nil
}

// ############################################################################
// TypeCount
// ############################################################################

// Returns a map of type strings and their counts. For each token, all of the
// modifiers provided will be performed before counting. An example of a
// [StringModifier] would be the function [strings.ToLower] or many others in
// the Go [strings] package.
// TODO document defaults here
func (t *Tokenizer) TypeCount(chunk string, opts ...TokenizerOption) (types map[string]int64, err error) {
	//TODO: type count a string with different tokenizers and other stuff using options functions like patrick did for radish
	// Make the types map (variable sz was selected arbitrarily)
	return make(map[string]int64), nil
}

// ############################################################################
// Stemmer
// ############################################################################

//TODO: stemmer struct, options, and functions
