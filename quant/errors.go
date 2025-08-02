package quant

import "errors"

var (
	ErrUnequalLengthVectors = errors.New("this operation requires vectors with the same number of elements")
)
