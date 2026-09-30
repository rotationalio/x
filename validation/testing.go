package validation

import (
	"errors"
	"slices"
	"testing"

	"go.rtnl.ai/x/assert"
)

// RequireValidationFields asserts that err contains exactly the specified validation fields.
func RequireValidationFields(t *testing.T, err error, fields ...string) {
	t.Helper()

	assert.Error(t, err)
	var verr Errors
	assert.True(t, errors.As(err, &verr), "expected error to contain validation.Errors")
	assert.Len(t, verr, len(fields))

	got := verr.Map()
	for _, field := range fields {
		_, ok := got[field]
		assert.True(t, ok, "expected validation error for field %s", field)
	}
}

// RequireValidation asserts that err contains a validation error equal to target.
func RequireValidation(t *testing.T, err error, target *FieldError, msgAndArgs ...any) {
	t.Helper()

	assert.Error(t, err, msgAndArgs...)
	var verr Errors
	assert.True(t, errors.As(err, &verr), msgAndArgs...)

	found := slices.ContainsFunc(verr, func(e *FieldError) bool {
		return target.Equal(e)
	})
	if len(msgAndArgs) > 0 {
		assert.True(t, found, msgAndArgs...)
		return
	}
	assert.True(t, found, "expected validation error for field "+target.Field())
}
