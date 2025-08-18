package quant

// ############################################################################
// TypeCounter
// ############################################################################

// TypeCounter can be used to perform type counting on text; create with [NewTypeCounter].
type TypeCounter struct {
	lang      Language
	tokenizer Tokenizer
	stemmer   Stemmer
}

// Returns a new [TypeCounter] instance. Defaults to the default [RegexTokenizer] and
// [Stemmer] options. Modified by passing [TypeCounterOption] functions into
// relevant function calls.
//
// Defaults:
//   - Language: [LanguageEnglish]
//   - Tokenizer: [RegexTokenizer]
//   - Stemmer: [Porter2Stemmer]
func NewTypeCounter(opts ...TypeCounterOption) (tc *TypeCounter, err error) {
	// Set options
	tc = &TypeCounter{}
	for _, fn := range opts {
		fn(tc)
	}

	// Set defaults
	if tc.lang == LanguageUnknown {
		tc.lang = LanuageEnglish
	}
	if tc.tokenizer == nil {
		tc.tokenizer = NewRegexTokenizer()
	}
	if tc.stemmer == nil {
		if tc.stemmer, err = NewPorter2Stemmer(tc.lang); err != nil {
			return nil, err
		}
	}

	return tc, nil
}

// Returns a map of type strings and their counts. For each token, all of the
// modifiers provided will be performed before counting. An example of a
// [StringModifier] would be the function [strings.ToLower] or many others in
// the Go [strings] package. Defaults to the default [Tokenizer] and [Stemmer]
// default options if none are provided.
func (c *TypeCounter) TypeCount(chunk string) (types map[string]int, err error) {
	// Tokenize
	var tokens []string
	if tokens, err = c.tokenizer.Tokenize(chunk); err != nil {
		return nil, err
	}

	// Stem
	for i, tok := range tokens {
		tokens[i] = c.stemmer.Stem(tok)
	}

	// Count
	return c.CountTypes(tokens), nil
}

// CountTypes returns a the count of each type (unique word) in the given token
// list.
func (c *TypeCounter) CountTypes(tokens []string) (types map[string]int) {
	sz := len(tokens) / 5 // map size selected arbitrarily
	types = make(map[string]int, sz)
	for _, tok := range tokens {
		types[tok] += 1
	}
	return types
}

// ############################################################################
// TypeCounterOptions
// ############################################################################

// TypeCounterOption functions modify a [TypeCounter].
type TypeCounterOption func(t *TypeCounter)

// WithTokenizer sets the [Tokenizer] to be used for the [TypeCounter].
func WithTokenizer(tokenizer Tokenizer) TypeCounterOption {
	return func(t *TypeCounter) {
		t.tokenizer = tokenizer
	}
}

// WithStemmer sets the [Stemmer] to be used for the [TypeCounter].
func WithStemmer(stemmer Stemmer) TypeCounterOption {
	return func(t *TypeCounter) {
		t.stemmer = stemmer
	}
}
