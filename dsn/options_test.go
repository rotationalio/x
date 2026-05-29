package dsn_test

import (
	"fmt"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/dsn"
)

func TestOptions(t *testing.T) {
	t.Run("ReadOnly", func(t *testing.T) {
		testCases := []struct {
			opts   dsn.Options
			assert assert.BoolAssertion
		}{
			{dsn.Options{"readonly": "true"}, assert.True},
			{dsn.Options{"readonly": "false"}, assert.False},
			{dsn.Options{"readonly": "1"}, assert.True},
			{dsn.Options{"readonly": "0"}, assert.False},
			{dsn.Options{"readonly": "true", "foo": "bar"}, assert.True},
			{dsn.Options{"readonly": "invalid"}, assert.False},
			{dsn.Options{"readonly": ""}, assert.False},
			{dsn.Options{"foo": "true"}, assert.False},
		}

		for i, tc := range testCases {
			tc.assert(t, tc.opts.ReadOnly(), "test case %d failed", i)
		}
	})

	t.Run("Get", func(t *testing.T) {
		testCases := []struct {
			opts     dsn.Options
			key      string
			expected string
			ok       bool
		}{
			{dsn.Options{"foo": "bar"}, "foo", "bar", true},
			{dsn.Options{"foo": "bar"}, "Foo", "bar", true},
			{dsn.Options{"foo": "bar"}, "FOO", "bar", true},
			{dsn.Options{"foo": "bar"}, "bar", "", false},
			{dsn.Options{"FOO": "bar"}, "foo", "bar", true},
			{dsn.Options{"FOO": "bar"}, "Foo", "bar", true},
			{dsn.Options{"FOO": "bar"}, "FOO", "bar", true},
			{dsn.Options{"FOO": "bar"}, "bar", "", false},
			{dsn.Options{"Foo": "bar"}, "foo", "", false},
			{dsn.Options{"Foo": "bar"}, "Foo", "bar", true},
			{dsn.Options{"Foo": "bar"}, "FOO", "", false},
			{dsn.Options{"Foo": "bar"}, "bar", "", false},
		}

		for i, tc := range testCases {
			actual, ok := tc.opts.Get(tc.key)
			assert.Equal(t, tc.expected, actual, "test case %d failed, value mismatch", i)
			assert.Equal(t, tc.ok, ok, "test case %d failed, ok mismatch", i)
		}
	})

	t.Run("Bool", func(t *testing.T) {
		testCases := []struct {
			opts     dsn.Options
			key      string
			expected bool
			err      error
		}{
			{dsn.Options{"foo": "true"}, "foo", true, nil},
			{dsn.Options{"foo": "false"}, "foo", false, nil},
			{dsn.Options{"foo": "1"}, "foo", true, nil},
			{dsn.Options{"foo": "0"}, "foo", false, nil},
			{dsn.Options{"foo": "true", "bar": "bar"}, "foo", true, nil},
			{dsn.Options{"foo": "invalid"}, "foo", false, fmt.Errorf("could not parse %q as bool: strconv.ParseBool: parsing \"invalid\": invalid syntax", "foo")},
			{dsn.Options{"foo": ""}, "foo", false, fmt.Errorf("could not parse %q as bool: strconv.ParseBool: parsing \"\": invalid syntax", "foo")},
			{dsn.Options{"foo": "true", "zap": "bar"}, "bar", false, fmt.Errorf("option %q not found", "bar")},
		}

		for i, tc := range testCases {
			actual, err := tc.opts.Bool(tc.key)
			assert.Equal(t, tc.expected, actual, "test case %d failed, value mismatch", i)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "test case %d failed, error mismatch", i)
			} else {
				assert.Ok(t, err, "test case %d failed, error mismatch", i)
			}
		}
	})

	t.Run("Int64", func(t *testing.T) {
		testCases := []struct {
			opts     dsn.Options
			key      string
			expected int64
			err      error
		}{
			{dsn.Options{"foo": "17234"}, "foo", 17234, nil},
			{dsn.Options{"foo": "0"}, "foo", 0, nil},
			{dsn.Options{"foo": "-313451"}, "foo", -313451, nil},
			{dsn.Options{"foo": "invalid"}, "foo", 0, fmt.Errorf("could not parse %q as int64: strconv.ParseInt: parsing \"invalid\": invalid syntax", "foo")},
			{dsn.Options{"foo": ""}, "foo", 0, fmt.Errorf("could not parse %q as int64: strconv.ParseInt: parsing \"\": invalid syntax", "foo")},
			{dsn.Options{"foo": "true", "zap": "bar"}, "bar", 0, fmt.Errorf("option %q not found", "bar")},
		}

		for i, tc := range testCases {
			actual, err := tc.opts.Int64(tc.key)
			assert.Equal(t, tc.expected, actual, "test case %d failed, value mismatch", i)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "test case %d failed, error mismatch", i)
			} else {
				assert.Ok(t, err, "test case %d failed, error mismatch", i)
			}
		}
	})

	t.Run("Float64", func(t *testing.T) {
		testCases := []struct {
			opts     dsn.Options
			key      string
			expected float64
			err      error
		}{
			{dsn.Options{"foo": "17234.56789"}, "foo", 17234.56789, nil},
			{dsn.Options{"foo": "0"}, "foo", 0, nil},
			{dsn.Options{"foo": "-313451.123456789"}, "foo", -313451.123456789, nil},
			{dsn.Options{"foo": "invalid"}, "foo", 0, fmt.Errorf("could not parse %q as float64: strconv.ParseFloat: parsing \"invalid\": invalid syntax", "foo")},
			{dsn.Options{"foo": ""}, "foo", 0, fmt.Errorf("could not parse %q as float64: strconv.ParseFloat: parsing \"\": invalid syntax", "foo")},
			{dsn.Options{"foo": "true", "zap": "bar"}, "bar", 0, fmt.Errorf("option %q not found", "bar")},
		}

		for i, tc := range testCases {
			actual, err := tc.opts.Float64(tc.key)
			assert.Equal(t, tc.expected, actual, "test case %d failed, value mismatch", i)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "test case %d failed, error mismatch", i)
			} else {
				assert.Ok(t, err, "test case %d failed, error mismatch", i)
			}
		}
	})

	t.Run("Duration", func(t *testing.T) {
		testCases := []struct {
			opts     dsn.Options
			key      string
			expected time.Duration
			err      error
		}{
			{dsn.Options{"foo": "17234s"}, "foo", 17234 * time.Second, nil},
			{dsn.Options{"foo": "0"}, "foo", 0, nil},
			{dsn.Options{"foo": "-313451h"}, "foo", -313451 * time.Hour, nil},
			{dsn.Options{"foo": "invalid"}, "foo", 0, fmt.Errorf("could not parse %q as duration: time: invalid duration %q", "foo", "invalid")},
			{dsn.Options{"foo": ""}, "foo", 0, fmt.Errorf("could not parse %q as duration: time: invalid duration %q", "foo", "")},
			{dsn.Options{"foo": "true", "zap": "bar"}, "bar", 0, fmt.Errorf("option %q not found", "bar")},
		}

		for i, tc := range testCases {
			actual, err := tc.opts.Duration(tc.key)
			assert.Equal(t, tc.expected, actual, "test case %d failed, value mismatch", i)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "test case %d failed, error mismatch", i)
			} else {
				assert.Ok(t, err, "test case %d failed, error mismatch", i)
			}
		}
	})
}
