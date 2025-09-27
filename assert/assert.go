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
	"math"
	"reflect"
	"regexp"
	"testing"
	"time"
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

// LessEqual fails the test if act (actual) is not less than or equal to exp (expected).
func LessEqual(tb testing.TB, exp, act interface{}, msgAndArgs ...interface{}) {
	tb.Helper()
	expectedValue := reflect.ValueOf(exp)
	actualValue := reflect.ValueOf(act)
	if expectedValue.Kind() != actualValue.Kind() {
		tb.Logf("\nexpected value type %s does not match actual value type %s\n", expectedValue.Kind(), actualValue.Kind())
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}

	switch actualValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		Assert(tb, actualValue.Kind() == expectedValue.Kind(), "expected value type %s does not match actual value type %s", expectedValue.Kind(), actualValue.Kind())
		Assert(tb, actualValue.Int() <= expectedValue.Int(), "expected %d to be less than or equal to %d", actualValue.Int(), expectedValue.Int())
	case reflect.Float32, reflect.Float64:
		Assert(tb, actualValue.Kind() == expectedValue.Kind(), "expected value type %s does not match actual value type %s", expectedValue.Kind(), actualValue.Kind())
		Assert(tb, actualValue.Float() <= expectedValue.Float(), "expected %f to be less than or equal to %f", actualValue.Float(), expectedValue.Float())
	default:
		tb.Logf("\nunsupported kind for comparison: %s\n", actualValue.Kind())
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}
}

// GreaterEqual fails the test if act (actual) is not greater than or equal to exp (expected).
func GreaterEqual(tb testing.TB, exp, act interface{}, msgAndArgs ...interface{}) {
	tb.Helper()
	expectedValue := reflect.ValueOf(exp)
	actualValue := reflect.ValueOf(act)
	if expectedValue.Kind() != actualValue.Kind() {
		tb.Logf("\nexpected value type %s does not match actual value type %s\n", expectedValue.Kind(), actualValue.Kind())
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}

	switch actualValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		Assert(tb, actualValue.Kind() == expectedValue.Kind(), "expected value type %s does not match actual value type %s", expectedValue.Kind(), actualValue.Kind())
		Assert(tb, actualValue.Int() >= expectedValue.Int(), "expected %d to be greater than or equal to %d", actualValue.Int(), expectedValue.Int())
	case reflect.Float32, reflect.Float64:
		Assert(tb, actualValue.Kind() == expectedValue.Kind(), "expected value type %s does not match actual value type %s", expectedValue.Kind(), actualValue.Kind())
		Assert(tb, actualValue.Float() >= expectedValue.Float(), "expected %f to be greater than or equal to %f", actualValue.Float(), expectedValue.Float())
	default:
		tb.Logf("\nunsupported kind for comparison: %s\n", actualValue.Kind())
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

// EqualError fails the test if the error message does not match the expected message.
func EqualError(tb testing.TB, err error, expected string, msgAndArgs ...interface{}) {
	tb.Helper()
	if err == nil {
		tb.Logf("\nexpected error but got nil\n")
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}
	if err.Error() != expected {
		tb.Logf("\nexpected error message to be %q but got %q\n", expected, err.Error())
		makeLogf(tb, msgAndArgs...)
		tb.FailNow()
	}
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

// IsType asserts that the specified object is of the expected type.
func IsType(tb testing.TB, expectedType interface{}, object interface{}, msgAndArgs ...interface{}) {
	tb.Helper()
	expectedTypeName := reflect.TypeOf(expectedType).String()
	actualTypeName := reflect.TypeOf(object).String()
	if expectedTypeName != actualTypeName {
		tb.Logf("\nexpected type %q but got %q\n", expectedTypeName, actualTypeName)
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

// Nil asserts that the specified object is nil.
func Nil(tb testing.TB, object interface{}, msgAndArgs ...interface{}) {
	tb.Helper()
	msg := makeMessage("expected nil, but got non-nil", msgAndArgs...)
	Assert(tb, object == nil || reflect.ValueOf(object).IsNil(), msg)
}

// NotNil asserts that the specified object is not nil.
func NotNil(tb testing.TB, object interface{}, msgAndArgs ...interface{}) {
	tb.Helper()
	msg := makeMessage("expected non-nil, but got nil", msgAndArgs...)
	Assert(tb, object != nil && !reflect.ValueOf(object).IsNil(), msg)
}

// InDelta asserts that the two numerals are within delta of each other.
func InDelta(tb testing.TB, expected, actual any, delta float64, msgAndArgs ...interface{}) {
	tb.Helper()

	af, aok := toFloat(expected)
	bf, bok := toFloat(actual)

	Assert(tb, aok && bok, makeMessage("parameters must be numbers", msgAndArgs...))

	// If both expected and actual are NaN, we consider them equal.
	if math.IsNaN(af) && math.IsNaN(bf) {
		return
	}

	Assert(tb, !math.IsNaN(af), makeMessage("expected must not be NaN", msgAndArgs...))
	Assert(tb, !math.IsNaN(bf), makeMessage(fmt.Sprintf("expected %v with delta %v, but was NaN", expected, delta), msgAndArgs...))

	d := af - bf
	Assert(tb, !(d < -delta || d > delta), makeMessage(fmt.Sprintf("expected %v to be within %v of %v", actual, delta, expected), msgAndArgs...))
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

// toFloat tries to convert an interface to a float64.
// It returns (0, false) if impossible.
func toFloat(x interface{}) (float64, bool) {
	var v float64
	ok := true

	switch n := x.(type) {
	case uint:
		v = float64(n)
	case uint8:
		v = float64(n)
	case uint16:
		v = float64(n)
	case uint32:
		v = float64(n)
	case uint64:
		v = float64(n)
	case int:
		v = float64(n)
	case int8:
		v = float64(n)
	case int16:
		v = float64(n)
	case int32:
		v = float64(n)
	case int64:
		v = float64(n)
	case float32:
		v = float64(n)
	case float64:
		v = n
	case time.Duration:
		v = float64(n)
	default:
		ok = false
	}

	return v, ok
}
