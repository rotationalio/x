package quant

import "errors"

var (
	ErrUnequalLengthVectors = errors.New("this operation requires vectors with the same number of elements")
	ErrLanguageNotSupported = errors.New("the selected language is not supported by this operation")
)
