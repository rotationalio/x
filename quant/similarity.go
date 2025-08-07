package quant

/*
similarity.go provides similarity metrics on strings.

TODO: finalize this documentation block

Types:
* None

Functions:
* Similarity(a, b string]) (similarity float64, err error)
*/

// ############################################################################
// Similarity
// ############################################################################

//TODO: SimilarityOptions struct and SimilarityModifier function type

// CosineSimilarity returns a value in the range [-1, 1] that indicates if two
// strings are similar.
// TODO: document the defaults here
func CosineSimilarity(a, b string, opt ...Options) (similarity float64, err error) {
	//TODO: vectorize the strings then do the Cosine to return the similarity between them
	vecA := make([]float64, 0)
	vecB := make([]float64, 0)

	// Calculate the cosine of the angle between the vectors as the similarity
	if similarity, err = Cosine(vecA, vecB); err != nil {
		return 0.0, err
	}

	return similarity, nil
}
