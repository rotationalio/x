package validation_test

import (
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/validation"
)

func TestFieldError_Error(t *testing.T) {
	testCases := []struct {
		err      *validation.FieldError
		expected string
	}{
		{validation.Missing("email"), "missing email: this field is required"},
		{validation.Incorrect("age", "must not be negative"), "invalid age: must not be negative"},
		{validation.ReadOnly("id"), "read-only id: this field cannot be modified by the user"},
		{validation.Duplicate("email", "jon@example.com"), `duplicate email: the value "jon@example.com" already exists`},
		{validation.Duplicate("email", ""), "duplicate email: this value for this field must be unique"},
	}

	for i, tc := range testCases {
		assert.Equal(t, tc.expected, tc.err.Error(), "test case %d failed", i)
	}
}

func TestFieldError_Field(t *testing.T) {
	assert.Equal(t, "email", validation.Missing("email").Field())
	assert.Equal(t, "", validation.Incorrect("", "issue").Field())
}

func TestFieldError_Subfield(t *testing.T) {
	err := validation.Missing("email")
	got := err.Subfield("contact")

	// Subfield prefixes the field and returns the same error for chaining.
	assert.Equal(t, "contact.email", err.Field())
	assert.Equal(t, "contact.email", got.Field())
	assert.Equal(t, "missing contact.email: this field is required", err.Error())

	// Subfield can be applied repeatedly to build a nested path.
	err.Subfield("user")
	assert.Equal(t, "user.contact.email", err.Field())
}

func TestFieldError_SubfieldArray(t *testing.T) {
	err := validation.Missing("street")
	got := err.SubfieldArray("addresses", 2)

	assert.Equal(t, "addresses[2].street", err.Field())
	assert.Equal(t, "addresses[2].street", got.Field())
	assert.Equal(t, "missing addresses[2].street: this field is required", err.Error())
}

func TestFieldError_Equal(t *testing.T) {
	t.Run("Equal", func(t *testing.T) {
		a := validation.Missing("email")
		b := validation.Missing("email")
		assert.True(t, a.Equal(b), "identical field errors should be equal")
	})

	t.Run("DifferentVerb", func(t *testing.T) {
		a := validation.Missing("email")
		b := validation.ReadOnly("email")
		assert.False(t, a.Equal(b), "different verbs should not be equal")
	})

	t.Run("DifferentField", func(t *testing.T) {
		a := validation.Missing("email")
		b := validation.Missing("name")
		assert.False(t, a.Equal(b), "different fields should not be equal")
	})

	t.Run("DifferentIssue", func(t *testing.T) {
		a := validation.Incorrect("age", "too small")
		b := validation.Incorrect("age", "too big")
		assert.False(t, a.Equal(b), "different issues should not be equal")
	})
}

func TestMissing(t *testing.T) {
	expected := validation.Missing("email")

	// Every alias should produce an equivalent field error.
	assert.True(t, expected.Equal(validation.MissingField("email")))
	assert.True(t, expected.Equal(validation.Required("email")))
	assert.True(t, expected.Equal(validation.RequiredField("email")))
}

func TestIncorrect(t *testing.T) {
	expected := validation.Incorrect("age", "must not be negative")

	assert.True(t, expected.Equal(validation.IncorrectField("age", "must not be negative")))
	assert.True(t, expected.Equal(validation.InvalidField("age", "must not be negative")))
	assert.True(t, expected.Equal(validation.Invalid("age", "must not be negative")))
}

func TestReadOnly(t *testing.T) {
	expected := validation.ReadOnly("id")

	assert.True(t, expected.Equal(validation.ReadOnlyField("id")))
	assert.True(t, expected.Equal(validation.Immutable("id")))
	assert.True(t, expected.Equal(validation.ImmutableField("id")))
}

func TestDuplicate(t *testing.T) {
	t.Run("WithValue", func(t *testing.T) {
		err := validation.Duplicate("email", "jon@example.com")
		assert.Equal(t, `duplicate email: the value "jon@example.com" already exists`, err.Error())
	})

	t.Run("WithoutValue", func(t *testing.T) {
		err := validation.Duplicate("email", "")
		assert.Equal(t, "duplicate email: this value for this field must be unique", err.Error())
	})

	t.Run("Aliases", func(t *testing.T) {
		expected := validation.Duplicate("email", "jon@example.com")
		assert.True(t, expected.Equal(validation.DuplicateField("email", "jon@example.com")))
		assert.True(t, expected.Equal(validation.AlreadyExists("email", "jon@example.com")))
		assert.True(t, expected.Equal(validation.AlreadyExistsField("email", "jon@example.com")))
	})
}

func TestOneOfMissing(t *testing.T) {
	t.Run("None", func(t *testing.T) {
		assert.Nil(t, validation.OneOfMissing())
	})

	t.Run("One", func(t *testing.T) {
		// A single field collapses to a standard missing error.
		assert.True(t, validation.Missing("email").Equal(validation.OneOfMissing("email")))
	})

	t.Run("Two", func(t *testing.T) {
		err := validation.OneOfMissing("email", "phone")
		assert.Equal(t, "email or phone", err.Field())
		assert.Equal(t, "missing one of email or phone: at least one of these fields is required", err.Error())
	})

	t.Run("Many", func(t *testing.T) {
		err := validation.OneOfMissing("email", "phone", "fax")
		assert.Equal(t, "email, phone, or fax", err.Field())
	})

	t.Run("Aliases", func(t *testing.T) {
		expected := validation.OneOfMissing("email", "phone")
		assert.True(t, expected.Equal(validation.OneOfMissingField("email", "phone")))
		assert.True(t, expected.Equal(validation.OneOfRequired("email", "phone")))
		assert.True(t, expected.Equal(validation.OneOfRequiredField("email", "phone")))
	})
}

func TestOneOfTooMany(t *testing.T) {
	t.Run("None", func(t *testing.T) {
		assert.Nil(t, validation.OneOfTooMany())
	})

	t.Run("One", func(t *testing.T) {
		assert.Nil(t, validation.OneOfTooMany("email"))
	})

	t.Run("Two", func(t *testing.T) {
		err := validation.OneOfTooMany("email", "phone")
		assert.Equal(t, "email or phone", err.Field())
		assert.Equal(t, "specify only one of email or phone: at most only one of these fields can be set", err.Error())
	})

	t.Run("Many", func(t *testing.T) {
		err := validation.OneOfTooMany("email", "phone", "fax")
		assert.Equal(t, "email, phone, or fax", err.Field())
	})

	t.Run("Aliases", func(t *testing.T) {
		expected := validation.OneOfTooMany("email", "phone")
		assert.True(t, expected.Equal(validation.OneOfTooManyField("email", "phone")))
		assert.True(t, expected.Equal(validation.OneOfOnlyOne("email", "phone")))
		assert.True(t, expected.Equal(validation.OneOfOnlyOneField("email", "phone")))
	})
}
