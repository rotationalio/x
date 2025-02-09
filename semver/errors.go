package semver

import "errors"

var (
	ErrInvalidSemVer = errors.New("invalid semantic version")
	ErrScanValue     = errors.New("could not scan source value")
)
