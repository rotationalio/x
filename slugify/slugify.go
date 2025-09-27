package slugify

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	ErrEmpty   = errors.New("slug cannot be empty")
	ErrInvalid = errors.New("slug contains invalid characters")
	ErrDashes  = errors.New("slug cannot start or end with a dash or contain multiple consecutive dashes")
)

var skip = []*unicode.RangeTable{
	unicode.Mark,
	unicode.Sk,
	unicode.Lm,
}

var safe = []*unicode.RangeTable{
	unicode.Letter,
	unicode.Number,
}

var cmap = map[rune]string{
	'&': "and",
	'@': "at",
	'%': "percent",
}

func Slugify(text string) string {
	buf := make([]rune, 0, len(text))
	dash := false

	for _, r := range norm.NFKD.String(text) {
		if rep, ok := cmap[r]; ok {
			if dash {
				buf = append(buf, '-')
			}

			for _, rr := range rep {
				buf = append(buf, rr)
			}

			buf = append(buf, '-')
			dash = false
			continue
		}

		switch {
		case unicode.IsOneOf(safe, r):
			buf = append(buf, unicode.ToLower(r))
			dash = true
		case unicode.IsOneOf(skip, r):
		case dash:
			buf = append(buf, '-')
			dash = false
		}
	}

	// Remove leading slashes
	if i := len(buf) - 1; i >= 0 && buf[i] == '-' {
		buf = buf[:i]
	}
	return string(buf)
}

func Slugifyf(format string, a ...any) string {
	return Slugify(fmt.Sprintf(format, a...))
}

func Validate(slug string) error {
	if slug == "" {
		return ErrEmpty
	}

	for i, r := range slug {
		if !(unicode.IsOneOf(safe, r) || r == '-') || !nfkd(r) {
			return ErrInvalid
		}

		if r == '-' {
			if i == 0 || i == len(slug)-1 || slug[i-1] == '-' {
				return ErrDashes
			}
		}
	}

	return nil
}

// Returns true if the rune is in its NFKD form or has no compatibility mapping.
func nfkd(r rune) bool {
	b := make([]byte, 4)
	n := utf8.EncodeRune(b, r)
	return norm.NFKD.IsNormal(b[:n])
}
