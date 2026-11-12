package humanize

import (
	"strconv"
	"strings"
)

func stripTrailingZeros(s string) string {
	if !strings.ContainsRune(s, '.') {
		return s
	}

	offset := len(s) - 1
	for offset > 0 {
		if s[offset] == '.' {
			offset--
			break
		}
		if s[offset] != '0' {
			break
		}
		offset--
	}

	return s[:offset+1]
}

func stripTrailingDigits(s string, n int) string {
	if i := strings.IndexRune(s, '.'); i >= 0 {
		if n <= 0 {
			return s[:i]
		}

		i++
		if n >= len(s)-i {
			return s
		}

		return s[:i+n]
	}
	return s
}

// Removes the negative sign from a string that represents zero; e.g. a negative zero
// or a negative number that has been truncated to zero such as "-0" or "-0.00".
func stripNegativeZero(s string) string {
	if strings.HasPrefix(s, "-") && strings.Trim(s[1:], "0.") == "" {
		return s[1:]
	}
	return s
}

// Ftoa converts a float value to a string with the specified number of decimal places,
// truncating or padding with zeros as necessary.
func Ftoa[F Float](n F, precision int) string {
	return stripNegativeZero(stripTrailingZeros(stripTrailingDigits(strconv.FormatFloat(float64(n), 'f', 6, 64), precision)))
}
