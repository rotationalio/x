package quant

// ############################################################################
// SimilarityOption
// ############################################################################

// TODO docs and name?
// TODO: interface instead?
type Similarizer struct {
	vocab      map[string]int //TODO SimilarityOption
	tokenizer  Tokenizer      //TODO SimilarityOption
	vectorizer Vectorizer     //TODO SimilarityOption
}

// A SimilarityOption function sets options for a Similarity metric function.
type SimilarityOption func(args ...any) Similarizer //TODO: fix args?

// ############################################################################
// CosineSimilarity
// ############################################################################

// CosineSimilarity returns a value in the range [-1, 1] that indicates if two
// strings are similar. // TODO: document the defaults for the options
func (s *Similarizer) CosineSimilarity(a, b string, opts ...SimilarityOption) (similarity float64, err error) {
	// TODO: set options (in a function)

	// Type count both strings
	// TODO only do this if we need a vocab

	//Vectorize the strings
	vecA := make([]float64, 0) //TODO
	vecB := make([]float64, 0) //TODO

	// Calculate the cosine of the angle between the vectors as the similarity
	if similarity, err = Cosine(vecA, vecB); err != nil {
		return 0.0, err
	}

	return similarity, nil
}
