package validation

import (
	"fmt"
	"strings"
)

// Error appends field validation errors into a single error chain.
func Error(err error, errs ...*FieldError) error {
	var verr Errors
	if err == nil {
		verr = make(Errors, 0, len(errs))
	} else {
		var ok bool
		if verr, ok = err.(Errors); !ok {
			verr = make(Errors, 0, len(errs)+1)

			if ferr, ok := err.(*FieldError); ok {
				verr = append(verr, ferr)
			} else {
				verr = append(verr, Incorrect("input", err.Error()))
			}
		}
	}

	for _, e := range errs {
		if e != nil {
			verr = append(verr, e)
		}
	}

	if len(verr) == 0 {
		return nil
	}

	return verr
}

// If you have a child field that returns validation errors, then this method will join
// the child errors into a single validation error for the parent field. It will also
// ensure that errors are prefixed with the parent field's name.
func SubfieldError(err, suberr error, field string) error {
	// If no suberr occurred, then do not modify the parent error.
	if suberr == nil {
		return err
	}

	switch child := suberr.(type) {
	case *FieldError:
		return Error(err, child.Subfield(field))
	case Errors:
		// Convert all of the field errors into subfield errors.
		for _, fieldError := range child {
			err = Error(err, fieldError.Subfield(field))
		}
		return err
	default:
		return Error(err, Incorrect(field, suberr.Error()))
	}
}

//============================================================================
// Validation Errors Type
//============================================================================

// Aggregates field-level validation issues.
type Errors []*FieldError

// Implements the error interface.
func (e Errors) Error() string {
	if len(e) == 1 {
		return e[0].Error()
	}

	sb := &strings.Builder{}
	fmt.Fprintf(sb, "%d validation errors occurred:\n", len(e))
	for i, err := range e {
		fmt.Fprintf(sb, "  %s", err)
		if i < len(e)-1 {
			sb.WriteRune('\n')
		}
	}
	return sb.String()
}

// Prefix a field path to every validation field.
func (e Errors) Prefix(parent string) Errors {
	if parent == "" {
		return e
	}

	for i, err := range e {
		e[i] = err.Subfield(parent)
	}
	return e
}

// Converts validation errors into a map of field path to error string. Useful for
// sending field level validation errors in an API response.
func (e Errors) Map() map[string]string {
	errs := make(map[string]string, len(e))
	for _, err := range e {
		errs[err.Field()] = err.Error()
	}
	return errs
}
