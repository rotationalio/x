// Utilities for identifying, parsing, and comparing semantic version strings as
// specified by Semantic Versioning 2.0.0 (https://semver.org/).
//
// Summary:
//
// Given a version number MAJOR.MINOR.PATCH, increment the:
//
// 1. MAJOR version when you make incompatible API changes
// 2. MINOR version when you add functionality in a backward compatible manner
// 3. PATCH version when you make backward compatible bug fixes
//
// Additional labels for pre-release and build metadata are available as extensions
// to the MAJOR.MINOR.PATCH format.
package semver

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// See: https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
var re = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Valid checks a version string against the regular expression specified by semver.org.
func Valid(v string) bool {
	return re.MatchString(v)
}

// Parse the semantic version using the regular expression specified by semver.org.
func Parse(vers string) (v Version, err error) {
	match := re.FindStringSubmatch(vers)
	if match == nil {
		return v, ErrInvalidSemVer
	}

	v.Major = parseNumeric(match[1])
	v.Minor = parseNumeric(match[2])
	v.Patch = parseNumeric(match[3])
	v.PreRelease = match[4]
	v.BuildMeta = match[5]

	return v, nil
}

func parseNumeric(v string) uint16 {
	n, _ := strconv.ParseUint(v, 10, 16)
	return uint16(n)
}

// MustParse panics if the version string is invalid.
func MustParse(vers string) (v Version) {
	var err error
	if v, err = Parse(vers); err != nil {
		panic(err)
	}
	return v
}

// Version represents the parsed components of a semantic version string.
type Version struct {
	Major      uint16
	Minor      uint16
	Patch      uint16
	PreRelease string
	BuildMeta  string
}

func (v Version) String() string {
	vers := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)

	if v.PreRelease != "" {
		vers += "-" + v.PreRelease
	}

	if v.BuildMeta != "" {
		vers += "+" + v.BuildMeta
	}

	return vers
}

func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0 && v.PreRelease == "" && v.BuildMeta == ""
}

func (v Version) Satisfies(spec Specifies) bool {
	return spec(v)
}

//===========================================================================
// Comparison
//===========================================================================

// Compare returns the precedence of version v compared to version o. See Compare
// for more details about how precedence is determined.
func (v Version) Compare(o Version) int {
	return Compare(v, o)
}

// Compare returns the precedence of version a compared to version b. If a has a higher
// precedence than b (a > b), the result is 1. If a has a lower precedence than b (a < b),
// the result is -1. If they have the same precedence (a == b), the result is 0.
// Note that BuildMeta is not factored into precedence so it is possible to have two
// semantically equivalent versions with different string values.
//
// Precedence is determined by the first difference when comparing each of these
// identifiers from left to right as follows: Major, minor, and patch versions are
// always compared numerically.
//
// When major, minor, and patch are equal, a pre-release version has lower precedence
// than a normal version.
//
// Precedence for two pre-release versions with the same major, minor, and patch version
// MUST be determined by comparing each dot separated identifier from left to right
// until a difference is found as follows:
//
//  1. Identifiers consisting of only digits are compared numerically.
//  2. Identifiers with letters or hyphens are compared lexically in ASCII sort order.
//  3. Numeric identifiers always have lower precedence than non-numeric identifiers.
//  4. A larger set of pre-release fields has a higher precedence than a smaller set,
//     if all of the preceding identifiers are equal.
func Compare(a, b Version) int {
	if a.Major != b.Major {
		if a.Major > b.Major {
			return 1
		}
		return -1
	}

	if a.Minor != b.Minor {
		if a.Minor > b.Minor {
			return 1
		}
		return -1
	}

	if a.Patch != b.Patch {
		if a.Patch > b.Patch {
			return 1
		}
		return -1
	}

	if a.PreRelease != b.PreRelease {
		// a doesn't have a prerelease so it has a higher precedence
		if a.PreRelease == "" {
			return 1
		}

		// b doesn't have a prerelease so it has a higher precedence
		if b.PreRelease == "" {
			return -1
		}

		ar := strings.Split(a.PreRelease, ".")
		br := strings.Split(b.PreRelease, ".")

		if len(ar) >= len(br) {
			for i, bv := range br {
				av := ar[i]
				if precedence := compareIdentifier(av, bv); precedence != 0 {
					return precedence
				}
			}

			// a has more pre-release fields than b and all are equal so a has a higher precedence
			// if len(ar) == len(br) then the first difference would have been found in the loop
			return 1
		} else {
			for i, av := range ar {
				bv := br[i]
				if precedence := compareIdentifier(av, bv); precedence != 0 {
					return precedence
				}
			}

			// b has more pre-release fields than a and all are equal so b has a higher precedence
			return -1
		}
	}

	// At this point we know the versions are semantically equivalent.
	return 0
}

func compareIdentifier(a, b string) int {
	if a == b {
		return 0
	}

	// Determine if a and b are numeric or non-numeric.
	na, aIsNumeric := isNumeric(a)
	nb, bIsNumeric := isNumeric(b)

	switch {
	// If both are numeric, compare numerically.
	case aIsNumeric && bIsNumeric:
		if na > nb {
			return 1
		}
		return -1
	// If one is numeric and the other is not, the numeric one has lower precedence.
	case aIsNumeric:
		return -1
	case bIsNumeric:
		return 1
	default:
		// Compare lexicographically in ASCII sort order.
		return strings.Compare(a, b)
	}
}

func isNumeric(s string) (v uint16, ok bool) {
	n, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, false
	}
	return uint16(n), true
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (v Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *Version) UnmarshalText(data []byte) error {
	parsed, err := Parse(string(data))
	if err != nil {
		return err
	}

	*v = parsed
	return nil
}

func (v Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

func (v *Version) UnmarshalJSON(data []byte) error {
	var vers string
	if err := json.Unmarshal(data, &vers); err != nil {
		return err
	}

	parsed, err := Parse(vers)
	if err != nil {
		return err
	}

	*v = parsed
	return nil
}

//===========================================================================
// SQL Interfaces
//===========================================================================

func (v *Version) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil
	case []byte:
		return v.UnmarshalText(src)
	case string:
		return v.UnmarshalText([]byte(src))
	default:
		return ErrScanValue
	}
}

func (v Version) Value() (interface{}, error) {
	return v.String(), nil
}
