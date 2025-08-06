package quant

import (
	"math"
)

/*
vector.go provides vector-related functionality with `[]float64`.

Types:
* None

Functions:
* DotProduct(a, b []float64]) (product float64, err error)
* VectorLength(v []float64]) (length float64)
* CosineSimilarity(a, b []float64]) (similarity float64, err error)
*/

// ############################################################################
// DotProduct
// ############################################################################

// VectorLength returns the dot product of the two vectors (as defined by SLP
// 3rd Edition section 6.4 fig 6.7). If the vectors do not have the same number
// of elements, an error will be returned.
var DotProduct func(a, b []float64) (product float64, err error) = DotProduct_impl_1

// First Implementation of DotProduct, using a 'for range' loop.
func DotProduct_impl_1(a, b []float64) (product float64, err error) {
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
// VectorLength
// ############################################################################

// VectorLength returns the vector length (as defined by SLP 3rd Edition section
// 6.4 fig 6.8).
var VectorLength func(v []float64) (length float64) = VectorLength_impl_1

// First Implementation of VectorLength, using a 'for range' loop.
func VectorLength_impl_1(v []float64) (length float64) {
	for _, e := range v {
		length += e * e
	}
	return math.Sqrt(length)
}

// ############################################################################
// CosineSimilarity
// ############################################################################

// CosineSimilarity returns the cosine similarity (as defined by SLP 3rd Edition
// section 6.4 fig 6.10). If the vectors do not have the same number of elements,
// an error will be returned.
func CosineSimilarity(a, b []float64) (similarity float64, err error) {
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

func TextSimilarity(a, b string, opt ...Options) (similarity float64, err error) {
	//TODO:
	return 0.0, nil
}
