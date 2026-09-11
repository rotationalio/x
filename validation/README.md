# Validation

This package provides error helpers for validating structs, particularly to produce
consistent, field-level validation errors in JSON and REST API responses. Rather than
returning a single opaque error, it lets you accumulate one error per invalid field,
attach a machine-readable field path to each, aggregate them into a single error value,
and render them as a `map[string]string` suitable for an API response body.

There are two core types:

- `FieldError`: a single validation issue for one field (a verb, a field path, and an issue description).
- `Errors`: a slice of `*FieldError` that aggregates all field-level issues into one `error`.

## Field Errors

A `FieldError` is constructed with one of the constructors below. Each renders with the
form `<verb> <field>: <issue>`, for example `missing email: this field is required`.

| Constructor | Verb | Meaning |
| --- | --- | --- |
| `Missing(field)` | `missing` | The field is required but zero-valued or empty. |
| `Incorrect(field, issue)` | `invalid` | The field's value is invalid; `issue` describes why. |
| `ReadOnly(field)` | `read-only` | The field cannot be modified by the user. |
| `Duplicate(field, value)` | `duplicate` | The field must be unique and `value` already exists (pass `""` to omit the value). |
| `OneOfMissing(fields...)` | `missing one of` | At least one of the listed fields is required but none are set. |
| `OneOfTooMany(fields...)` | `specify only one of` | At most one of the listed fields may be set, but more than one is. |

Each constructor has aliases so you can pick the name that reads best at the call site:

- `Missing`: `MissingField`, `Required`, `RequiredField`
- `Incorrect`: `IncorrectField`, `InvalidField`, `Invalid`
- `ReadOnly`: `ReadOnlyField`, `Immutable`, `ImmutableField`
- `Duplicate`: `DuplicateField`, `AlreadyExists`, `AlreadyExistsField`
- `OneOfMissing`: `OneOfMissingField`, `OneOfRequired`, `OneOfRequiredField`
- `OneOfTooMany`: `OneOfTooManyField`, `OneOfOnlyOne`, `OneOfOnlyOneField`

Note that `OneOfMissing` returns `nil` when no fields are supplied, and `OneOfTooMany`
returns `nil` when fewer than two fields are supplied, so these are safe to call
unconditionally.

## Aggregating Errors

Use `Error` to collect field errors into a single `Errors` value while validating a
struct. It accepts an existing error (which may be `nil`, a `*FieldError`, or an
`Errors`) plus any number of additional `*FieldError` values, skipping `nil` entries. If
nothing is wrong, it returns `nil`.

```go
func (u *User) Validate() error {
	var err error

	if u.Email == "" {
		err = validation.Error(err, validation.Missing("email"))
	}

	if u.Age < 0 {
		err = validation.Error(err, validation.Incorrect("age", "must not be negative"))
	}

	if u.ID != "" {
		err = validation.Error(err, validation.ReadOnly("id"))
	}

	return err
}
```

Because `Error` returns `nil` when there are no field errors, the common pattern above
returns a valid struct as `nil` and an aggregated `Errors` otherwise.

## Nested Structs and Field Paths

Field paths use dot notation (and `[index]` for array elements) so that clients can map
each error back to the exact field that produced it.

- `FieldError.Subfield(parent)` prefixes the field with a parent name (`email` becomes `contact.email`).
- `FieldError.SubfieldArray(parent, index)` prefixes with a parent and array index (`street` becomes `addresses[2].street`).
- `Errors.Prefix(parent)` applies a parent prefix to every error in the slice.

When a child value has its own `Validate` method, use `SubfieldError` to fold its errors
into the parent under the child's field name. It handles a `nil` suberror, a single
`*FieldError`, an `Errors` slice, or any other `error` (wrapped as an `Incorrect`
error):

```go
func (o *Order) Validate() error {
	var err error

	if o.Total < 0 {
		err = validation.Error(err, validation.Incorrect("total", "must not be negative"))
	}

	// Fold the customer's validation errors in under the "customer" field.
	err = validation.SubfieldError(err, o.Customer.Validate(), "customer")

	return err
}
```

## Rendering for API Responses

`Errors` implements the `error` interface. A single error renders as just that error's
message; multiple errors render as a numbered summary followed by each error on its own
line.

For structured API responses, `Errors.Map` converts the aggregate into a
`map[string]string` keyed by field path:

```go
if err := user.Validate(); err != nil {
	if verrs, ok := err.(validation.Errors); ok {
		// e.g. {"email": "missing email: this field is required"}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": verrs.Map()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	return
}
```

## Comparing Errors

`FieldError.Equal` reports whether two field errors have the same verb, field, and
issue, which is handy for table-driven tests that assert exact validation output.

`Errors.Equal` reports if the two error chains have the same errors no matter the order. This method assumes that there are no duplicates in either error chain and that the error chains are the same length.
