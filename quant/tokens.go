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
* TokenizeStringNaive(corpus string, lang Language) (tokens []string, err error)
* TypeCountStringTokens(tokens []string, tokenModifiers ...StringModifier) (types map[string]int64)
* TypeCount(chunk string, options ...Options)
*/

// ############################################################################
// Tokenize
// ############################################################################

//TODO: Tokenizer struct and TokenizerOption

// Tokenizes a string (naively) by grouping alphanumeric characters, ignoring
// non-alphanumeric characters. Does not modify the corpus before tokenizing.
// TODO: replace lang with options
func Tokenize(chunk string, lang Language) (tokens []string, err error) {
	var (
		expr string
		r    *regexp.Regexp
	)

	// Define the regexp expression by language
	switch lang {
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
	tokens = r.FindAllString(corpus, -1)

	return tokens, nil
}

// ############################################################################
// TypeCountStringTokens
// ############################################################################

// Returns a map of type strings and their counts. For each token, all of the
// modifiers provided will be performed before counting. An example of a
// [StringModifier] would be the function [strings.ToLower] or many others in
// the Go [strings] package.
// TODO: replace the modifiers with an options functions thing like patrick did in radish
func TypeCountTokens(tokens []string, tokenModifiers ...StringModifier) (types map[string]int64, err error) {
	// Make the types map (variable sz was selected arbitrarily)
	sz := len(tokens) / 4
	types = make(map[string]int64, sz)

	// Modify and count the tokens
	for _, t := range tokens {
		// Apply all token modifiers to the token
		for _, modFn := range tokenModifiers {
			t = modFn(t)
		}

		// Count the token
		types[t] += 1
	}

	return types
}

// ############################################################################
// TypeCount
// ############################################################################

type Options func(something any) any //FIXME: make this something useful not just a type filler for now

// Returns a map of type strings and their counts. For each token, all of the
// modifiers provided will be performed before counting. An example of a
// [StringModifier] would be the function [strings.ToLower] or many others in
// the Go [strings] package.
// TODO: chunk implies it's a piece; later we can chunk a whole corpus into parts perhaps
func TypeCount(chunk string, options ...Options) (types map[string]int64, err error) {
	//TODO: type count a string with different tokenizers and other stuff using options functions like patrick did for radish
	// Make the types map (variable sz was selected arbitrarily)
	return make(map[string]int64), nil
}

// ############################################################################
// Stemmer
// ############################################################################

//TODO: stemmer struct, options, and functions
