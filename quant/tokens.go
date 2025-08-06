package quant

import (
	"regexp"
)

/*
tokens.go provides tokenization and counting functionality.

Types:
* None

Functions:
* TokenizeStringNaive(corpus string, lang Language) (tokens []string, err error)
* TypeCountStringTokens(tokens []string, tokenModifiers ...StringModifier) (types map[string]int64)
*/

// ############################################################################
// TokenizeStringNaive
// ############################################################################

// Tokenizes a string (naively) by grouping alphanumeric characters, ignoring
// non-alphanumeric characters. Does not modify the corpus before tokenizing.
func TokenizeStringNaive(corpus string, lang Language) (tokens []string, err error) {
	var (
		expr string
		r    *regexp.Regexp
	)

	// Define the regexp expression by language
	switch lang {
	case LanuageEnglish:
		// 26 uppercase, 26 lowercase, and 10 digits
		expr = `A-Za-z0-9`
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
func TypeCountStringTokens(tokens []string, tokenModifiers ...StringModifier) (types map[string]int64) {
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
