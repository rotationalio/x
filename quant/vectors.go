package quant

import (
	"math"
)

// ############################################################################
// Vector Math Functions
// ############################################################################

// Cosine returns the cosine of the angle between two vectors; which can be used
// as a similarity metric (as defined by SLP 3rd Edition section 6.4 fig 6.10).
// If the vectors do not have the same number of elements, an error will be
// returned.
func Cosine(a, b []float64) (cosine float64, err error) {
	// Ensure vectors have the same number of elements
	if len(a) != len(b) {
		return 0.0, ErrUnequalLengthVectors
	}

	var (
		dp, vla, vlb float64
	)
	if dp, err = DotProduct(a, b); err != nil {
		return 0.0, err
	}
	vla = VectorLength(a)
	vlb = VectorLength(b)
	return dp / (vla * vlb), nil
}

// DotProduct returns the dot product of the two vectors (as defined by SLP 3rd
// Edition section 6.4 fig 6.7). If the vectors do not have the same number
// of elements, an error will be returned.
func DotProduct(a, b []float64) (product float64, err error) {
	// Ensure vectors have the same number of elements
	if len(a) != len(b) {
		return 0.0, ErrUnequalLengthVectors
	}

	for i := range a {
		product += a[i] * b[i]
	}
	return product, nil
}

// VectorLength returns the vector length (as defined by SLP 3rd Edition section
// 6.4 fig 6.8).
func VectorLength(v []float64) (length float64) {
	for _, e := range v {
		length += e * e
	}
	return math.Sqrt(length)
}

// ############################################################################
// Vectorizer interface
// ############################################################################

type Vectorizer interface {
	Vectorize(chunk string) (vector []float64, err error)
}

// ############################################################################
// CountVectorizer
// ############################################################################

// CountVectorizer can be used to vectorize text using the frequency or one-hot
// text vectorization algorithms.
type CountVectorizer struct {
	vocab            map[string]int
	lang             Language
	tokenizer        Tokenizer
	stemmer          Stemmer
	typeCounter      *TypeCounter
	method           VectorizationMethod
	excludeStopWords bool
}

// Returns a new [CountVectorizer] instance.
//
// Defaults:
//   - Lang: [LanguageEnglish]
//   - Tokenizer: [RegexTokenizer] using Lang above and it's own defaults
//   - Stemmer: [Porter2Stemmer] using Lang above and it's own defaults
//   - TypeCounter: [TypeCounter] using Lang, Stemmer, and Tokenizer above
//   - Method: [VectorizeOneHot]
//   - ExcludeStopWords: [true] (uses the stop words list for Lang above)
func NewCountVectorizer(vocab map[string]int, opts ...CountVectorizerOption) (vectorizer *CountVectorizer, err error) {
	// Set options
	vectorizer = &CountVectorizer{excludeStopWords: true}
	for _, fn := range opts {
		fn(vectorizer)
	}

	// Set vocab (a required option)
	vectorizer.vocab = vocab

	// Set defaults
	if vectorizer.lang == LanguageUnknown {
		vectorizer.lang = LanuageEnglish
	}
	if vectorizer.tokenizer == nil {
		// create with default regex
		vectorizer.tokenizer = NewRegexTokenizer(RegexTokenizerWithLanguage(vectorizer.lang))
	}
	if vectorizer.stemmer == nil {
		if vectorizer.stemmer, err = NewPorter2Stemmer(vectorizer.lang); err != nil {
			return nil, err
		}
	}
	if !vectorizer.typeCounter.initialized {
		if vectorizer.typeCounter, err = NewTypeCounter(
			TypeCounterWithLanguage(vectorizer.lang),
			TypeCounterWithTokenizer(vectorizer.tokenizer),
			TypeCounterWithStemmer(vectorizer.stemmer),
		); err != nil {
			return nil, err
		}
	}
	if vectorizer.method == VectorizeUnknown {
		vectorizer.method = VectorizeOneHot
	}

	return vectorizer, nil
}

// Returns the [CountVectorizer]s configured vocabulary.
func (c *CountVectorizer) Vocab() map[string]int {
	return c.vocab
}

// Returns the [CountVectorizer]s configured [Language].
func (c *CountVectorizer) Language() Language {
	return c.lang
}

// Returns the [CountVectorizer]s configured [Tokenizer].
func (c *CountVectorizer) Tokenizer() Tokenizer {
	return c.tokenizer
}

// Returns the [CountVectorizer]s configured [Stemmer].
func (c *CountVectorizer) Stemmer() Stemmer {
	return c.stemmer
}

// Returns the [CountVectorizer]s configured [TypeCounter].
func (c *CountVectorizer) TypeCounter() *TypeCounter {
	return c.typeCounter
}

// Returns the [CountVectorizer]s configured [VectorizationMethod].
func (c *CountVectorizer) Method() VectorizationMethod {
	return c.method
}

// Returns whther the [CountVectorizer] is configured to exclude stop words.
func (c *CountVectorizer) ExcludeStopWords() bool {
	return c.excludeStopWords
}

// Vectorizes the chunk of text.
func (v *CountVectorizer) Vectorize(chunk string) (vector []float64, err error) {
	switch v.method {
	case VectorizeOneHot:
		return v.VectorizeOneHot(chunk)
	case VectorizeFrequency:
		return v.VectorizeFrequency(chunk)
	}
	return nil, ErrMethodNotSupported
}

// VectorizeFrequency returns a frequency (count) encoding vector for the given
// chunk of text and given vocabulary map. The vector returned has a value of
// the count of word instances within the chunk for each vocabulary word index.
func (v *CountVectorizer) VectorizeFrequency(chunk string) (vector []float64, err error) {
	// Type count the chunk
	var types map[string]int
	if types, err = v.typeCounter.TypeCount(chunk); err != nil {
		return nil, err
	}

	// Create the vector from the vocabulary
	vector = make([]float64, len(v.vocab))
	for word, i := range v.vocab {
		if count, ok := types[word]; ok {
			vector[i] = float64(count)
		}
	}

	return vector, nil
}

// VectorizeOneHot returns a one-hot encoding vector for the given chunk of text
// and given vocabulary map. The vector returned has a value of 1 for each
// vocabulary word index if it is present within the chunk of text and 0
// otherwise.
func (v *CountVectorizer) VectorizeOneHot(chunk string) (vector []float64, err error) {
	// Get the frequency encoding
	if vector, err = v.VectorizeFrequency(chunk); err != nil {
		return nil, err
	}

	// Then convert it to a one-hot encoding
	for i, e := range vector {
		if e != 0.0 {
			vector[i] = 1
		}
	}

	return vector, nil
}

// ############################################################################
// VectorizationMethod "enum"
// ############################################################################

type VectorizationMethod uint8

const (
	VectorizeUnknown = iota
	VectorizeOneHot
	VectorizeFrequency
)

// ############################################################################
// CountVectorizerOption
// ############################################################################

// TypeCounterOption functions modify a [CountVectorizer].
type CountVectorizerOption func(c *CountVectorizer)

// CountVectorizerWithLang sets the [Language] to use with the
// [CountVectorizer].
func CountVectorizerWithLang(lang Language) CountVectorizerOption {
	return func(c *CountVectorizer) {
		c.lang = lang
	}
}

// CountVectorizerWithTokenizer sets the [Tokenizer] to use with the
// [CountVectorizer].
func CountVectorizerWithTokenizer(tokenizer Tokenizer) CountVectorizerOption {
	return func(c *CountVectorizer) {
		c.tokenizer = tokenizer
	}
}

// CountVectorizerWithStemmer sets the [Stemmer] to use with the
// [CountVectorizer].
func CountVectorizerWithStemmer(stemmer Stemmer) CountVectorizerOption {
	return func(c *CountVectorizer) {
		c.stemmer = stemmer
	}
}

// CountVectorizerWithTypeCounter sets the [TypeCounter] to use with the
// [CountVectorizer].
func CountVectorizerWithTypeCounter(typecounter *TypeCounter) CountVectorizerOption {
	return func(c *CountVectorizer) {
		c.typeCounter = typecounter
	}
}

// CountVectorizerWithMethod sets the [VectorizationMethod] to use with the
// [CountVectorizer].
func CountVectorizerWithMethod(method VectorizationMethod) CountVectorizerOption {
	return func(c *CountVectorizer) {
		c.method = method
	}
}

// CountVectorizerWithExcludeStopWords sets whether the [CountVectorizer] will
// exclude stop words.
func CountVectorizerWithExcludeStopWords(exclude bool) CountVectorizerOption {
	return func(c *CountVectorizer) {
		c.excludeStopWords = exclude
	}
}
