package humanize

import (
	"bytes"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Produces a string of the numeric value in base 10 with commas after every three orders of magnitude.
func Comma[N Numeric](n N) string {
	switch v := reflect.ValueOf(n); {
	case v.CanInt():
		return CommaInt(v.Int())
	case v.CanUint():
		return CommaUint(v.Uint())
	case v.CanFloat():
		return CommaFloat(v.Float())
	default:
		panic("unreachable: N is constrained to integer or float types")
	}
}

// Produces a string of the float value in base 10 with commas after every three orders
// of magnitude, constraining the the output to the specified number of decimal places,
// truncating or padding with zeros as necessary.
func Commaf[F Float](n F, precision int) string {
	// Special values have no decimal places to truncate or pad
	f := float64(n)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return CommaFloat(f)
	}

	s := stripNegativeZero(stripTrailingDigits(CommaFloat(f), precision))
	if precision <= 0 {
		return s
	}

	if dot := strings.IndexRune(s, '.'); dot < 0 {
		s += "." + strings.Repeat("0", precision)
	} else if have := len(s) - dot - 1; have < precision {
		s += strings.Repeat("0", precision-have)
	}

	return s
}

// Converts an int64 value to a string with commas after every three orders of magnitude.
func CommaInt(n int64) string {
	// Shortcut case for small numbers
	if n&^0b111 == 0 {
		return string([]byte{byte(n) + 48})
	}

	// Handle smallest int64 special case
	if n == math.MinInt64 {
		return "-9,223,372,036,854,775,808"
	}

	// Count the number of digits
	var count byte = 0
	for d := n; d != 0; d = d / 10 {
		count++
	}

	// Add the number of commas to the count
	count += (count - 1) / 3

	// Add the negative sign to the count
	if n < 0 {
		n = 0 - n
		count++
	}

	// Create a string buffer with the output size
	out := make([]byte, count)

	// String loop book keeping
	var counter byte = 0
	var j int = len(out) - 1

	for n > 9 {
		out[j] = byte(n%10) + 48
		n = n / 10
		j--

		if counter == 2 {
			counter = 0
			out[j] = ','
			j--
		} else {
			counter++
		}
	}

	out[j] = byte(n) + 48
	if j == 1 {
		out[0] = '-'
	}

	return string(out)
}

// Converts a uint64 value to a string with commas after every three orders of magnitude.
func CommaUint(n uint64) string {
	// Shortcut case for small numbers
	if n&^0b111 == 0 {
		return string([]byte{byte(n) + 48})
	}

	// Count the number of digits
	var count byte = 0
	for d := n; d != 0; d = d / 10 {
		count++
	}

	// Add the number of commas to the count
	count += (count - 1) / 3

	// Create a string buffer with the output size
	out := make([]byte, count)

	// String loop book keeping
	var counter byte = 0
	var j int = len(out) - 1

	for n > 9 {
		out[j] = byte(n%10) + 48
		n = n / 10
		j--

		if counter == 2 {
			counter = 0
			out[j] = ','
			j--
		} else {
			counter++
		}
	}

	out[j] = byte(n) + 48
	return string(out)
}

// Converts a float64 value to a string with commas after every three orders of magnitude.
func CommaFloat(n float64) string {
	// Handle special cases
	switch {
	case math.IsNaN(n):
		return "NaN"
	case math.IsInf(n, -1):
		return "-Inf"
	case math.IsInf(n, 1):
		return "+Inf"
	case n == 0:
		// Handles both positive and negative zero
		return "0"
	}

	buf := &bytes.Buffer{}
	if n < 0 {
		buf.Write([]byte{'-'})
		n = 0 - n
	}

	comma := []byte{','}
	parts := strings.Split(strconv.FormatFloat(n, 'f', -1, 64), ".")
	pos := 0

	if len(parts[0])%3 != 0 {
		pos += len(parts[0]) % 3
		buf.WriteString(parts[0][:pos])
		buf.Write(comma)
	}

	for ; pos < len(parts[0]); pos += 3 {
		buf.WriteString(parts[0][pos : pos+3])
		buf.Write(comma)
	}

	buf.Truncate(buf.Len() - 1)

	if len(parts) > 1 {
		buf.Write([]byte{'.'})
		buf.WriteString(parts[1])
	}

	return buf.String()
}

// Parses a string produced by the Comma function into an integer value.
func ParseComma(s string) (int64, error) {
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseInt(s, 10, 64)
}

// Parses a string produced by the Comma function into a float value.
func ParseCommaf(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseFloat(s, 64)
}
