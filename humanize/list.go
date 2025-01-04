package humanize

import "strings"

// Returns a list of strings as a human-readable string with commas and "and"
func AndList[T String](t []T) string {
	strs := make([]string, 0, len(t))
	for _, s := range t {
		strs = append(strs, string(s))
	}
	return NaturalList(strs, "and")
}

// Returns a list of strings as a human-readable string with commas and "or"
func OrList[T String](t []T) string {
	strs := make([]string, 0, len(t))
	for _, s := range t {
		strs = append(strs, string(s))
	}
	return NaturalList(strs, "or")
}

// Returns a list of strings with commas and the final string separated by the conjunction
func NaturalList(strs []string, conjunction string) string {
	// Ensure there is only one space before and after conjunction
	conjunction = " " + strings.TrimSpace(conjunction) + " "

	switch len(strs) {
	case 0:
		return ""
	case 1:
		return strs[0]
	case 2:
		return strs[0] + conjunction + strs[1]
	default:
		return strings.Join(strs[:len(strs)-1], ", ") + conjunction + strs[len(strs)-1]
	}
}
