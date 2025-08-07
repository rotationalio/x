package quant

import "slices"

/*
sharedtypes.go provides a location for types that are useful for more than one
context.

Types:
* Language
* StringModifier

Functions:
* Language.In(langs ...Language) bool
*/

// ############################################################################
// Language type enumeration
// ############################################################################

type Language uint16

const (
	LanguageUnknown = iota
	LanuageEnglish
)

// Returns True if the argument [Language]s contains this language.
func (l Language) In(langs ...Language) bool {
	return slices.Contains(langs, l)
}

// ############################################################################
// StringModifier function type
// ############################################################################

// A StringModifier is a function which takes a string and returns a string.
type StringModifier func(string) string
