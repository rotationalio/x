package quant

// ############################################################################
// TypeCounter
// ############################################################################

// TypeCounter can be used to perform type counting on text; create with [NewTypeCounter].
// TODO: interface instead?
type TypeCounter struct {
	tokenizer *Tokenizer
	stemmer   *Stemmer
}

// Returns a new [TypeCounter] instance. Defaults to the default [Tokenizer] and
// [Stemmer] options. Modified by passing [TypeCounterOption] functions into
// relevant function calls.
func NewTypeCounter() *TypeCounter {
	return &TypeCounter{
		tokenizer: NewTokenizer(),
		//TODO stemmer:   NewStemmer(),
	}
}

// Returns a map of type strings and their counts. For each token, all of the
// modifiers provided will be performed before counting. An example of a
// [StringModifier] would be the function [strings.ToLower] or many others in
// the Go [strings] package. Defaults to the default [Tokenizer] and [Stemmer]
// default options if none are provided.
func (c *TypeCounter) TypeCount(chunk string, opts ...TypeCounterOption) (types map[string]int64, err error) {
	// Set TypeCounter options
	for _, fn := range opts {
		fn(c)
	}

	// Tokenizing
	var tokens []string
	if tokens, err = c.tokenizer.Tokenize(chunk); err != nil {
		return nil, err
	}

	// Stemming
	//FIXME:
	// for i, tok := range tokens {
	// 	tokens[i] = c.stemmer.Stem(tok)
	// }

	// Counting
	return c.CountTypes(tokens), nil
}

// CountTypes returns a the count of each type (unique word) in the given token
// list.
func (c *TypeCounter) CountTypes(tokens []string) (types map[string]int64) {
	sz := len(tokens) / 5 // map size selected arbitrarily
	types = make(map[string]int64, sz)
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
func WithTokenizer(tokenizer *Tokenizer) TypeCounterOption {
	return func(t *TypeCounter) {
		t.tokenizer = tokenizer
	}
}

// WithStemmer sets the [Stemmer] to be used for the [TypeCounter].
func WithStemmer(stemmer *Stemmer) TypeCounterOption {
	return func(t *TypeCounter) {
		t.stemmer = stemmer
	}
}
