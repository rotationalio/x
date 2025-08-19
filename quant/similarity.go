package quant

// ############################################################################
// Similarizer interface
// ############################################################################

type Similarizer interface {
	Similarity(a, b string) (similarity float64, err error)
}

// ############################################################################
// CosineSimilarizer
// ############################################################################

// CosineSimilarizer can be used to calculate the cosine similarity of two text
// chunks.
type CosineSimilarizer struct {
	vocab      map[string]int
	lang       Language
	tokenizer  Tokenizer
	vectorizer Vectorizer
}

// Returns a new [CosineSimilarizer] with the vocabulary and options set.
//
// Defaults:
//   - Lang: [LanguageEnglish]
//   - Tokenizer: [RegexTokenizer] with the Lang above and it's own defaults
//   - Vectorizer: [CountVectorizer] with the given vocabulary, Lang above, and
//     it's own defaults
func NewCosineSimilarizer(vocab map[string]int, opts ...CosineSimilarizerOption) (similarizer *CosineSimilarizer, err error) {
	// Set options
	similarizer = &CosineSimilarizer{}
	for _, fn := range opts {
		fn(similarizer)
	}

	// Set vocab (a required option)
	similarizer.vocab = vocab

	//Set defaults
	if similarizer.lang == LanguageUnknown {
		similarizer.lang = LanuageEnglish
	}
	if similarizer.tokenizer == nil {
		similarizer.tokenizer = NewRegexTokenizer(RegexTokenizerWithLanguage(similarizer.lang))
	}
	if similarizer.vectorizer == nil {
		if similarizer.vectorizer, err = NewCountVectorizer(similarizer.vocab, CountVectorizerWithLang(similarizer.lang)); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// Similarity returns a value in the range [-1.0, 1.0] that indicates if two
// strings are similar using the cosine similarity method.
func (s *CosineSimilarizer) Similarity(a, b string) (similarity float64, err error) {
	//Vectorize the strings
	var vecA, vecB []float64
	if vecA, err = s.vectorizer.Vectorize(a); err != nil {
		return 0.0, err
	}
	if vecB, err = s.vectorizer.Vectorize(b); err != nil {
		return 0.0, err
	}

	// Calculate the cosine of the angle between the vectors as the similarity
	if similarity, err = Cosine(vecA, vecB); err != nil {
		return 0.0, err
	}

	return similarity, nil
}

// ############################################################################
// SimilarityOption
// ############################################################################

// A CosineSimilarizerOption function sets options for a [CosineSimilarizer].
type CosineSimilarizerOption func(s *CosineSimilarizer)

// Returns a function which sets a [CosineSimilarizer]s [Language].
func CosineSimilarizerWithLanguage(lang Language) CosineSimilarizerOption {
	return func(s *CosineSimilarizer) {
		s.lang = lang
	}
}

// Returns a function which sets a [CosineSimilarizer]s [Tokenizer].
func CosineSimilarizerWithTokenizer(tokenizer Tokenizer) CosineSimilarizerOption {
	return func(s *CosineSimilarizer) {
		s.tokenizer = tokenizer
	}
}

// Returns a function which sets a [CosineSimilarizer]s [Vectorizer].
func CosineSimilarizerWithVectorizer(vectorizer Vectorizer) CosineSimilarizerOption {
	return func(s *CosineSimilarizer) {
		s.vectorizer = vectorizer
	}
}
