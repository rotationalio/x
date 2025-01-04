/*
Utilities to convert various types into more human readable strings.
*/
package humanize

import "strings"

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Integer interface {
	Signed | Unsigned
}

type Float interface {
	~float32 | ~float64
}

// Expresses a quantity of something
type Numeric interface {
	Integer | Float
}

type String interface {
	~string
}

// True if the numeric value is 1
func Singular[N Numeric](n N) bool {
	return n == 1.0
}

// Return the singular or pluralized form of one unit if n is singular or not. The
// plural form is determined by the following rules:
//
// If suffix is not specified and replace is not specified: append "s"
// If suffix is not specified and replace is: return replace (e.g. for irregular verbs)
// If suffix is specified and replace is not: apppend suffix
// If suffix and replace are specified: trim replace suffix and append suffix
func MakePlural[N Numeric](n N, unit, suffix, replace string) string {
	if Singular(n) {
		return unit
	}

	switch {
	case suffix == "" && replace == "":
		// Just append an s in the default case
		return unit + "s"
	case suffix == "" && replace != "":
		// Irregular verbs case
		return replace
	case suffix != "" && replace == "":
		// Append the suffix (e.g. for -es)
		return unit + suffix
	case suffix != "" && replace != "":
		unit, _ = strings.CutSuffix(unit, replace)
		return unit + suffix
	default:
		panic("unknown pluralization case")
	}
}

// Return the singular or pluralized form of the unit based on the value of n.
func Plural[N Numeric](n N, singular, plural string) string {
	if Singular(n) {
		return singular
	}
	return plural
}

var esSuffixes = [6]string{"s", "z", "ch", "sh", "x", "o"}

// Best effort attempt to pluralize the unit based on the suffix using English rules.
// Appends an -s or -es depending on the clashing suffix; converts words that end in
// "f" to -ves and words that end in "y" to "ies". Cannot handle irregular plurals and
// this is not guanteed to be correct.
func Pluralize(unit string) string {
	l := len(unit) - 1
	switch {
	case unit[l] == 'y':
		return unit[:l] + "ies"
	case unit[l] == 'f':
		return unit[:l] + "ves"
	case strings.HasSuffix(unit, "fe"):
		return unit[:len(unit)-2] + "ves"
	case strings.HasSuffix(unit, "on"):
		return unit[:len(unit)-2] + "a"
	default:
		for _, suffix := range esSuffixes {
			if strings.HasSuffix(unit, suffix) {
				return unit + "es"
			}
		}
		return unit + "s"
	}
}
