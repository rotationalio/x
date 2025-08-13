package quant

import "slices"

/*
sharedtypes.go provides a location for shared types.

Types:
* Language

Functions:
* (l Language) In(langs ...Language) bool
*/

// ############################################################################
// Language (a lightweight enumeration)
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
