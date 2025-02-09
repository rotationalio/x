package semver

import "errors"

var (
	ErrInvalidSemVer = errors.New("invalid semantic version")
	ErrInvalidRange  = errors.New("invalid semantic range")
	ErrScanValue     = errors.New("could not scan source value")
)
