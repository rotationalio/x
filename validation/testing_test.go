package validation_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/validation"
)

// A wrapped validation error passes when its field set matches exactly.
func TestRequireValidationFields_Success(t *testing.T) {
	err := validation.Error(nil,
		validation.Missing("email"),
		validation.Incorrect("age", "must not be negative"),
	)
	validation.RequireValidationFields(t, fmt.Errorf("wrapped: %w", err), "email", "age")
}

// Missing errors, non-validation errors, and incorrect field sets fail assertions.
func TestRequireValidationFields_Failures(t *testing.T) {
	tests := []struct {
		name       string
		wantOutput string
	}{
		{name: "fields-missing-error", wantOutput: "expected error but got nil"},
		{name: "fields-non-validation-error", wantOutput: "expected error to contain validation.Errors"},
		{name: "fields-wrong-count", wantOutput: "should have 2 item(s) but has 1"},
		{name: "fields-wrong-set", wantOutput: "expected validation error for field name"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runFailureProbe(t, tc.name, tc.wantOutput)
		})
	}
}

// A matching field error passes even when aggregated with other errors and wrapped.
func TestRequireValidation_Success(t *testing.T) {
	target := validation.Missing("email")
	err := validation.Error(nil, validation.ReadOnly("id"), validation.Missing("email"))
	validation.RequireValidation(t, fmt.Errorf("wrapped: %w", err), target)
}

// Missing errors, non-validation errors, and unmatched targets fail with optional context.
func TestRequireValidation_Failures(t *testing.T) {
	tests := []struct {
		name       string
		wantOutput string
	}{
		{name: "target-missing-error", wantOutput: "expected error but got nil"},
		{name: "target-non-validation-error", wantOutput: "custom context: details"},
		{name: "target-non-matching", wantOutput: "custom context: details"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runFailureProbe(t, tc.name, tc.wantOutput)
		})
	}
}

// Runs an expected assertion failure in a separate process so it cannot fail its parent test.
func runFailureProbe(t *testing.T, probe, wantOutput string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestValidationHelperFailureProbe$")
	cmd.Env = append(os.Environ(), "X_VALIDATION_HELPER_FAILURE_PROBE="+probe)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "failure probe %q unexpectedly passed", probe)
	assert.Contains(t, string(output), wantOutput)
}

// Dispatches a single failure case when launched by the parent process.
func TestValidationHelperFailureProbe(t *testing.T) {
	switch os.Getenv("X_VALIDATION_HELPER_FAILURE_PROBE") {
	case "":
		// Skip when the test binary is running without a requested probe.
		return
	case "fields-missing-error":
		// A nil error must fail the required-error assertion.
		validation.RequireValidationFields(t, nil, "email")
	case "fields-non-validation-error":
		// A regular error must fail the validation-error assertion.
		validation.RequireValidationFields(t, errors.New("something broke"), "input")
	case "fields-wrong-count":
		// The number of validation errors must match the expected field count.
		validation.RequireValidationFields(t, validation.Error(nil, validation.Missing("email")), "email", "age")
	case "fields-wrong-set":
		// Every expected field must be present in the validation errors.
		err := validation.Error(nil, validation.Missing("email"), validation.Missing("age"))
		validation.RequireValidationFields(t, err, "email", "name")
	case "target-missing-error":
		// A nil error must fail the required-error assertion.
		validation.RequireValidation(t, nil, validation.Missing("name"))
	case "target-non-validation-error":
		// A regular error must fail the validation-error assertion.
		validation.RequireValidation(t, errors.New("something broke"), validation.Missing("name"), "custom context: %s", "details")
	case "target-non-matching":
		// A validation error unequal to the target must fail the match assertion.
		err := validation.Error(nil, validation.Missing("email"))
		validation.RequireValidation(t, err, validation.Missing("name"), "custom context: %s", "details")
	default:
		// Catch misspelled or unsupported probe selectors.
		t.Fatalf("unknown validation helper failure probe %q", os.Getenv("X_VALIDATION_HELPER_FAILURE_PROBE"))
	}
}
