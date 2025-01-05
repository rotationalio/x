/*
Assertion Helpers

Because this is a library, we prefer to have no dependencies including our usual test
dependencies (e.g. testify require). So we have some basic assertion helpers for tests.

See: https://github.com/benbjohnson/testing
*/
package assert

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
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

type BoolAssertion func(testing.TB, bool, ...interface{})

// True asserts that the condition is true.
func True(tb testing.TB, condition bool, msgAndArgs ...interface{}) {
	tb.Helper()
	msg := makeMessage("expected condition to be true", msgAndArgs...)
	Assert(tb, condition, msg)
}

// False asserts that the condition is false.
func False(tb testing.TB, condition bool, msgAndArgs ...interface{}) {
	tb.Helper()
	msg := makeMessage("expected condition to be false", msgAndArgs...)
	Assert(tb, !condition, msg)
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error, msgAndArgs ...interface{}) {
	tb.Helper()
	if err != nil {
		tb.Logf("\nunexpected error: %q\n", err.Error())
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}
}

// Equal fails the test if exp (expected) is not equal to act (actual).
func Equal(tb testing.TB, exp, act interface{}, msgAndArgs ...interface{}) {
	tb.Helper()
	if !reflect.DeepEqual(exp, act) {
		tb.Logf("\nactual value did not match expected:\n\n\t- exp: %#v\n\t- got: %#v\n", exp, act)
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}
}

// NotEqual fails the text if exp (expected) is equal to act (actual).
func NotEqual(tb testing.TB, exp, act interface{}, msgAndArgs ...interface{}) {
	tb.Helper()
	if reflect.DeepEqual(exp, act) {
		tb.Logf("\nactual value equals expected:\n\n\t- exp: %#v\n\t- got: %#v\n", exp, act)
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}
}

// ErrorIs fails the test if the err does not match the target.
func ErrorIs(tb testing.TB, err, target error, msgAndArgs ...interface{}) {
	tb.Helper()
	msg := makeMessage("expected target to be in error chain", msgAndArgs...)
	Assert(tb, errors.Is(err, target), msg)
}

// Len asserts that the specified container has specific length.
func Len(tb testing.TB, container interface{}, length int, msgAndArgs ...interface{}) {
	tb.Helper()
	l, ok := getLen(container)
	if !ok {
		tb.Logf("\n\"%v\" could not be applied to builtin len()\n", container)
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}
	if l != length {
		tb.Logf("\n\"%v\" should have %d item(s) but has %d\n", container, length, l)
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}
}

// Regexp asserts that a specified regular expression matches a string.
func Regexp(tb testing.TB, rx *regexp.Regexp, str string, msgAndArgs ...interface{}) {
	tb.Helper()
	msg := makeMessage("regular expression did not match target string", msgAndArgs...)
	Assert(tb, rx.MatchString(str), msg)
}

func makeMessage(msg string, msgAndArgs ...interface{}) string {
	switch len(msgAndArgs) {
	case 0:
		return msg
	case 1:
		return msg + "\n" + msgAndArgs[0].(string)
	default:
		return msg + "\n" + fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
}

func makeLogf(tb testing.TB, msgAndArgs ...interface{}) {
	switch len(msgAndArgs) {
	case 0:
		return
	case 1:
		tb.Log("\n" + msgAndArgs[0].(string) + "\n")
	default:
		tb.Logf("\n"+msgAndArgs[0].(string)+"\n", msgAndArgs[1:]...)
	}
}

// getLen tries to get the length of an object.
// It returns (0, false) if impossible.
func getLen(x interface{}) (length int, ok bool) {
	v := reflect.ValueOf(x)
	defer func() {
		ok = recover() == nil
	}()
	return v.Len(), true
}
