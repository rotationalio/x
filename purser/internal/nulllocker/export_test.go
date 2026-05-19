package nulllocker

// Test-only exports. Visible to nulllocker_test in this package and to external
// tests in package nulllocker_test, but invisible outside test builds.

import "go.rtnl.ai/x/purser"

// NewNilLocker returns a typed-nil *nullLocker wrapped in the Locker interface.
// External tests use this to exercise nil-receiver contracts on the unexported
// concrete type without exposing it in the production API.
func NewNilLocker() purser.Locker {
	var l *nullLocker
	return l
}
