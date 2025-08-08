package quant

import (
	"math"
)

/*
vectors.go provides vector-related functionality.

Types:
* None

Functions:
* `Cosine(a, b []float64]) (cosine float64, err error)`
* `DotProduct(a, b []float64]) (product float64, err error)`
* `VectorizeFrequency(chunk string, vocab map[string]int) (vector []float64, err error)`
* `VectorizeOneHot(chunk string, vocab map[string]int) (vector []float64, err error)`
* `VectorLength(v []float64]) (length float64)`
*/

// ############################################################################
// Vectorizer
// ############################################################################

// TODO: docs
type Vectorizer struct {
	tokenizer Tokenizer //TODO: VectorizeOption function
}

// TODO: docs
type VectorizeOption func(args ...any) Vectorizer //TODO: fix args?

// ############################################################################
// Cosine
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

// ############################################################################
// DotProduct
// ############################################################################

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

// ############################################################################
// VectorizeFrequency
// ############################################################################

// VectorizeFrequency returns a frequency (count) encoding vector for the given
// chunk of text and given vocabulary map. The vector returned has a value of
// the count of word instances within the chunk for each vocabulary word index.
func (v *Vectorizer) VectorizeFrequency(chunk string, vocab map[string]int) (vector []float64, err error) {
	// Type count the chunk
	var types map[string]int64
	if types, err = v.tokenizer.TypeCount(chunk); err != nil {
		return nil, err
	}

	// Create the vector from the vocabulary
	vector = make([]float64, len(vocab))
	for word, i := range vocab {
		if count, ok := types[word]; ok {
			vector[i] = float64(count)
		}
	}

	return vector, nil
}

// ############################################################################
// VectorizeOneHot
// ############################################################################

// VectorizeOneHot returns a one-hot encoding vector for the given chunk of text
// and given vocabulary map. The vector returned has a value of 1 for each
// vocabulary word index if it is present within the chunk of text and 0
// otherwise.
func (v *Vectorizer) VectorizeOneHot(chunk string, vocab map[string]int) (vector []float64, err error) {
	// Get the frequency encoding first...
	if vector, err = v.VectorizeFrequency(chunk, vocab); err != nil {
		return nil, err
	}

	// ...then convert it to a one-hot encoding
	for i, e := range vector {
		if e != 0.0 {
			vector[i] = 1
		}
	}

	return vector, nil
}

// ############################################################################
// VectorLength
// ############################################################################

// VectorLength returns the vector length (as defined by SLP 3rd Edition section
// 6.4 fig 6.8).
func VectorLength(v []float64) (length float64) {
	for _, e := range v {
		length += e * e
	}
	return math.Sqrt(length)
}
