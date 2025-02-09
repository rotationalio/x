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
