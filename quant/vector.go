/*
vector.go provides vector-related functionality.
*/

package quant

import (
	"math"
)

// ############################################################################
// Vector64 type
// ############################################################################

// Vector64 is a vector of 64 bit floats.
type Vector64 []float64

// Create a new Vector64 from a slice of 64 bit floats.
func NewVector64(vals ...float64) Vector64 {
	// Vector64 is currently implemented as a slice of 64 bit floats.
	return vals
}

// Returns the number of elements in the Vector64.
func (v Vector64) Len() int {
	// Vector64 is currently implemented as a slice of 64 bit floats.
	return len(v)
}

// ############################################################################
// DotProduct
// ############################################################################

// VectorLength returns the dot product of the two vectors (as defined by SLP
// 3rd Edition section 6.4 fig 6.7). If the vectors do not have the same number
// of elements, an error will be returned.
var DotProduct func(a, b Vector64) (product float64, err error) = DotProduct_impl_1

// First Implementation of DotProduct, using a 'for range' loop.
func DotProduct_impl_1(a, b Vector64) (product float64, err error) {
	// Ensure vectors have the same number of elements
	if a.Len() != b.Len() {
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
var VectorLength func(v Vector64) (length float64) = VectorLength_impl_1

// First Implementation of VectorLength, using a 'for range' loop.
func VectorLength_impl_1(v Vector64) (length float64) {
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
var CosineSimilarity func(a, b Vector64) (similarity float64, err error) = CosineSimilarity_impl_1

// First Implementation of CosineSimilarity, using the DotProduct and
// VectorLength functions.
func CosineSimilarity_impl_1(a, b Vector64) (similarity float64, err error) {
	// Ensure vectors have the same number of elements
	if a.Len() != b.Len() {
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
