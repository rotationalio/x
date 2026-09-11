package validation

import (
	"fmt"
	"strings"
)

//============================================================================
// Field Error Type
//============================================================================

// FieldError represents a single validation issue for one field.
type FieldError struct {
	verb  string
	field string
	issue string
}

//============================================================================
// Field Error Methods
//============================================================================

// Implements the error interface.
func (e *FieldError) Error() string {
	return fmt.Sprintf("%s %s: %s", e.verb, e.field, e.issue)
}

// Field returns the field path associated with the error in . notation.
func (e *FieldError) Field() string {
	return e.field
}

// Subfield prefixes the current field name with a parent field.
func (e *FieldError) Subfield(parent string) *FieldError {
	e.field = parent + "." + e.field
	return e
}

// SubfieldArray prefixes the current field name with a parent field and array index.
func (e *FieldError) SubfieldArray(parent string, index int) *FieldError {
	e.field = fmt.Sprintf("%s[%d].%s", parent, index, e.field)
	return e
}

// Equal implements the Equaler interface.
func (e *FieldError) Equal(other *FieldError) bool {
	return e.verb == other.verb && e.field == other.field && e.issue == other.issue
}

//============================================================================
// Field Error Constructors
//============================================================================

// Indicates the field is zero valued or empty but is required.
func Missing(field string) *FieldError {
	return &FieldError{verb: "missing", field: field, issue: "this field is required"}
}

// Aliases for Missing
var (
	MissingField  = Missing
	Required      = Missing
	RequiredField = Missing
)

// Indicates that the value of the field is invalid or incorrect.
func Incorrect(field, issue string) *FieldError {
	return &FieldError{verb: "invalid", field: field, issue: issue}
}

// Aliases for Incorrect
var (
	IncorrectField = Incorrect
	InvalidField   = Incorrect
	Invalid        = Incorrect
)

// Indicates that the field is read-only and cannot be modified by the user.
func ReadOnly(field string) *FieldError {
	return &FieldError{verb: "read-only", field: field, issue: "this field cannot be modified by the user"}
}

// Aliases for ReadOnly
var (
	ReadOnlyField  = ReadOnly
	Immutable      = ReadOnly
	ImmutableField = ReadOnly
)

// Indicates the value for the field must be unique and already exists. Omit the value
// if you do not want the error to indicate what the duplicate value is.
func Duplicate(field, value string) *FieldError {
	e := &FieldError{verb: "duplicate", field: field}
	if value != "" {
		e.issue = fmt.Sprintf("the value %q already exists", value)
	} else {
		e.issue = "this value for this field must be unique"
	}
	return e
}

// Aliases for Duplicate
var (
	DuplicateField     = Duplicate
	AlreadyExists      = Duplicate
	AlreadyExistsField = Duplicate
)

// At least one of the specified fields must be non-zero valued but isn't.
func OneOfMissing(fields ...string) *FieldError {
	switch len(fields) {
	case 0:
		return nil
	case 1:
		return Missing(fields[0])
	default:
		return &FieldError{
			verb:  "missing one of",
			issue: "at least one of these fields is required",
			field: fieldList(fields...),
		}
	}
}

// Aliases for OneOfMissing
var (
	OneOfMissingField  = OneOfMissing
	OneOfRequired      = OneOfMissing
	OneOfRequiredField = OneOfMissing
)

// At most one of the specified fields is required and more are specified.
func OneOfTooMany(fields ...string) *FieldError {
	if len(fields) < 2 {
		return nil
	}
	return &FieldError{
		verb:  "specify only one of",
		issue: "at most only one of these fields can be set",
		field: fieldList(fields...),
	}
}

// Aliases for OneOfTooMany
var (
	OneOfTooManyField = OneOfTooMany
	OneOfOnlyOne      = OneOfTooMany
	OneOfOnlyOneField = OneOfTooMany
)

// ============================================================================
// Helpers
// ============================================================================

// fieldList joins field names for one-of style validation messages.
func fieldList(fields ...string) string {
	switch len(fields) {
	case 0:
		return ""
	case 1:
		return fields[0]
	case 2:
		return fmt.Sprintf("%s or %s", fields[0], fields[1])
	default:
		last := len(fields) - 1
		return fmt.Sprintf("%s, or %s", strings.Join(fields[0:last], ", "), fields[last])
	}
}
