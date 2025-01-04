/*
Assertion Helpers

Because this is a library, we prefer to have no dependencies including our usual test
dependencies (e.g. testify require). So we have some basic assertion helpers for tests.

See: https://github.com/benbjohnson/testing
*/
package assert

import (
	"errors"
	"reflect"
	"testing"
)

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	tb.Helper()
	if !condition {
		tb.Logf("\n"+msg+"\n", v...)
		tb.FailNow()
	}
}

type BoolAssertion func(testing.TB, bool)

// True asserts that the condition is true.
func True(tb testing.TB, condition bool) {
	tb.Helper()
	Assert(tb, condition, "expected condition to be true")
}

// False asserts that the condition is false.
func False(tb testing.TB, condition bool) {
	tb.Helper()
	Assert(tb, !condition, "expected condition to be false")
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Logf("\nunexpected error: %q\n", err.Error())
		tb.FailNow()
	}
}

// Equals fails the test if exp (expected) is not equal to act (actual).
func Equals(tb testing.TB, exp, act interface{}) {
	tb.Helper()
	if !reflect.DeepEqual(exp, act) {
		tb.Logf("\nactual value did not match expected:\n\n\t- exp: %#v\n\t- got: %#v\n", exp, act)
		tb.FailNow()
	}
}

// ErrorIs fails the test if the err does not match the target.
func ErrorIs(tb testing.TB, err, target error) {
	tb.Helper()
	Assert(tb, errors.Is(err, target), "expected target to be in error chain")
}
