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

import "regexp"

// See: https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
var re = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Valid checks a version string against the regular expression specified by semver.org.
func Valid(v string) bool {
	return re.MatchString(v)
}

type Version struct {
	Major      uint16
	Minor      uint16
	Patch      uint16
	PreRelease string
	BuildMeta  string
}
