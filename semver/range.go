package semver

import (
	"fmt"
	"strings"
	"unicode"
)

func Range(s string) (_ Specifies, err error) {
	return parseRange(s)
}

type Specifies func(Version) bool

func (s Specifies) Or(o Specifies) Specifies {
	return func(v Version) bool {
		return s(v) || o(v)
	}
}

func (s Specifies) And(o Specifies) Specifies {
	return func(v Version) bool {
		return s(v) && o(v)
	}
}

//===========================================================================
// Comparison Operators
//===========================================================================

type operator rune

const (
	EQ  operator = '='
	GT  operator = '>'
	GTE operator = '≥'
	LT  operator = '<'
	LTE operator = '≤'
)

func compare(v Version, op operator) Specifies {
	switch op {
	case EQ:
		return func(o Version) bool {
			return Compare(o, v) == 0
		}
	case GT:
		return func(o Version) bool {
			return Compare(o, v) > 0
		}
	case GTE:
		return func(o Version) bool {
			return Compare(o, v) >= 0
		}
	case LT:
		return func(o Version) bool {
			return Compare(o, v) < 0
		}
	case LTE:
		return func(o Version) bool {
			return Compare(o, v) <= 0
		}
	default:
		panic(fmt.Errorf("unknown operator %q", op))
	}
}

//===========================================================================
// Parsing
//===========================================================================

func parseRange(s string) (_ Specifies, err error) {
	parts := split(s)
	orParts, err := splitOR(parts)
	if err != nil {
		return nil, err
	}

	// TODO: expand wildcards

	var orFn Specifies
	for _, part := range orParts {
		var andFn Specifies
		for _, and := range part {
			op, vers, err := parseComparitor(and)
			if err != nil {
				return nil, err
			}

			// TODO: build range functions

			// Set function
			if andFn == nil {
				andFn = compare(vers, op)
			} else {
				andFn = andFn.And(compare(vers, op))
			}
		}

		if orFn == nil {
			orFn = andFn
		} else {
			orFn = orFn.Or(andFn)
		}
	}

	return orFn, nil
}

func parseComparitor(s string) (op operator, v Version, err error) {
	i := strings.IndexFunc(s, unicode.IsDigit)
	if i == -1 {
		return 0, Version{}, ErrInvalidRange
	}

	// Split the operator from the version
	ops, vers := s[0:i], s[i:]

	// Parse the version number
	if v, err = Parse(vers); err != nil {
		return 0, Version{}, err
	}

	// Parse the operator
	switch ops {
	case "=", "==":
		return EQ, v, nil
	case ">":
		return GT, v, nil
	case ">=":
		return GTE, v, nil
	case "<":
		return LT, v, nil
	case "<=":
		return LTE, v, nil
	default:
		return 0, Version{}, ErrInvalidRange
	}
}

func split(s string) (result []string) {
	last := 0
	var lastChar byte
	exclude := []byte{'>', '<', '='}

	for i := 0; i < len(s); i++ {
		if s[i] == ' ' && !inArray(lastChar, exclude) {
			if last < i-1 {
				result = append(result, s[last:i])
			}
			last = i + 1
		} else if s[i] != ' ' {
			lastChar = s[i]
		}
	}

	if last < len(s)-1 {
		result = append(result, s[last:])
	}

	for i, v := range result {
		result[i] = strings.Replace(v, " ", "", -1)
	}

	return result
}

func splitOR(parts []string) (result [][]string, err error) {
	last := 0
	for i, part := range parts {
		if part == "||" {
			if i == 0 {
				return nil, ErrInvalidRange
			}

			result = append(result, parts[last:i])
			last = i + 1
		}
	}

	if last == len(parts) {
		return nil, ErrInvalidRange
	}

	result = append(result, parts[last:])
	return result, nil
}

func inArray(s byte, list []byte) bool {
	for _, el := range list {
		if el == s {
			return true
		}
	}
	return false
}
