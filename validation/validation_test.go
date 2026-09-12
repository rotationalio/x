package validation_test

import (
	"errors"
	"testing"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/validation"
)

func TestError(t *testing.T) {
	t.Run("NilNoErrors", func(t *testing.T) {
		// No prior error and no field errors means nothing is wrong.
		assert.Nil(t, validation.Error(nil))
	})

	t.Run("OnlyNilFieldErrors", func(t *testing.T) {
		// nil field errors are skipped, so the result is still nil.
		assert.Nil(t, validation.Error(nil, nil, nil))
	})

	t.Run("NilWithFieldErrors", func(t *testing.T) {
		err := validation.Error(nil, validation.Missing("email"), validation.ReadOnly("id"))

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 2)
		assert.True(t, verrs[0].Equal(validation.Missing("email")))
		assert.True(t, verrs[1].Equal(validation.ReadOnly("id")))
	})

	t.Run("SkipsNilAmongFieldErrors", func(t *testing.T) {
		err := validation.Error(nil, nil, validation.Missing("email"), nil)

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 1)
		assert.True(t, verrs[0].Equal(validation.Missing("email")))
	})

	t.Run("AppendsToExistingErrors", func(t *testing.T) {
		existing := validation.Error(nil, validation.Missing("email"))
		err := validation.Error(existing, validation.ReadOnly("id"))

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 2)
		assert.True(t, verrs[0].Equal(validation.Missing("email")))
		assert.True(t, verrs[1].Equal(validation.ReadOnly("id")))
	})

	t.Run("WrapsExistingFieldError", func(t *testing.T) {
		existing := validation.Missing("email")
		err := validation.Error(existing, validation.ReadOnly("id"))

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 2)
		assert.True(t, verrs[0].Equal(validation.Missing("email")))
		assert.True(t, verrs[1].Equal(validation.ReadOnly("id")))
	})

	t.Run("WrapsGenericError", func(t *testing.T) {
		existing := errors.New("something broke")
		err := validation.Error(existing, validation.Missing("email"))

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 2)
		assert.True(t, verrs[0].Equal(validation.Incorrect("input", "something broke")))
		assert.True(t, verrs[1].Equal(validation.Missing("email")))
	})
}

func TestSubfieldError(t *testing.T) {
	t.Run("NilSuberr", func(t *testing.T) {
		existing := validation.Error(nil, validation.Missing("total"))
		err := validation.SubfieldError(existing, nil, "customer")

		// A nil suberror leaves the parent error untouched.
		assert.Equal(t, existing, err)
	})

	t.Run("FieldError", func(t *testing.T) {
		suberr := validation.Missing("email")
		err := validation.SubfieldError(nil, suberr, "customer")

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 1)
		assert.True(t, verrs[0].Equal(validation.Missing("customer.email")))
	})

	t.Run("Errors", func(t *testing.T) {
		suberr := validation.Error(nil, validation.Missing("email"), validation.ReadOnly("id"))
		err := validation.SubfieldError(nil, suberr, "customer")

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 2)
		assert.True(t, verrs[0].Equal(validation.Missing("customer.email")))
		assert.True(t, verrs[1].Equal(validation.ReadOnly("customer.id")))
	})

	t.Run("GenericError", func(t *testing.T) {
		suberr := errors.New("bad customer")
		err := validation.SubfieldError(nil, suberr, "customer")

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 1)
		assert.True(t, verrs[0].Equal(validation.Incorrect("customer", "bad customer")))
	})

	t.Run("MergesIntoParent", func(t *testing.T) {
		parent := validation.Error(nil, validation.Incorrect("total", "must not be negative"))
		err := validation.SubfieldError(parent, validation.Missing("email"), "customer")

		verrs, ok := err.(validation.Errors)
		assert.True(t, ok, "expected an Errors value")
		assert.Len(t, verrs, 2)
		assert.True(t, verrs[0].Equal(validation.Incorrect("total", "must not be negative")))
		assert.True(t, verrs[1].Equal(validation.Missing("customer.email")))
	})
}

func TestErrors_Error(t *testing.T) {
	t.Run("Single", func(t *testing.T) {
		verrs := validation.Errors{validation.Missing("email")}
		assert.Equal(t, "missing email: this field is required", verrs.Error())
	})

	t.Run("Multiple", func(t *testing.T) {
		verrs := validation.Errors{
			validation.Missing("email"),
			validation.Incorrect("age", "must not be negative"),
		}

		expected := "2 validation errors occurred:\n" +
			"  missing email: this field is required\n" +
			"  invalid age: must not be negative"
		assert.Equal(t, expected, verrs.Error())
	})
}

func TestErrors_Prefix(t *testing.T) {
	t.Run("EmptyParent", func(t *testing.T) {
		verrs := validation.Errors{validation.Missing("email")}
		got := verrs.Prefix("")

		// An empty parent is a no-op.
		assert.True(t, got[0].Equal(validation.Missing("email")))
	})

	t.Run("PrefixesAll", func(t *testing.T) {
		verrs := validation.Errors{
			validation.Missing("email"),
			validation.ReadOnly("id"),
		}
		got := verrs.Prefix("customer")

		assert.True(t, got[0].Equal(validation.Missing("customer.email")))
		assert.True(t, got[1].Equal(validation.ReadOnly("customer.id")))
	})
}

func TestErrors_Map(t *testing.T) {
	verrs := validation.Errors{
		validation.Missing("email"),
		validation.Incorrect("age", "must not be negative"),
	}

	expected := map[string]string{
		"email": "missing email: this field is required",
		"age":   "invalid age: must not be negative",
	}
	assert.Equal(t, expected, verrs.Map())
}

func TestErrors_Equal(t *testing.T) {
	t.Run("DifferentLength", func(t *testing.T) {
		a := validation.Errors{validation.Missing("email")}
		b := validation.Errors{validation.Missing("email"), validation.ReadOnly("id")}
		assert.False(t, a.Equal(b), "different lengths should not be equal")
	})

	t.Run("SameOrder", func(t *testing.T) {
		a := validation.Errors{validation.Missing("email"), validation.ReadOnly("id")}
		b := validation.Errors{validation.Missing("email"), validation.ReadOnly("id")}
		assert.True(t, a.Equal(b), "identical error chains should be equal")
	})

	t.Run("DifferentOrder", func(t *testing.T) {
		// Order does not matter for equality.
		a := validation.Errors{validation.Missing("email"), validation.ReadOnly("id")}
		b := validation.Errors{validation.ReadOnly("id"), validation.Missing("email")}
		assert.True(t, a.Equal(b), "order should not affect equality")
	})

	t.Run("DifferentContent", func(t *testing.T) {
		a := validation.Errors{validation.Missing("email"), validation.ReadOnly("id")}
		b := validation.Errors{validation.Missing("email"), validation.ReadOnly("name")}
		assert.False(t, a.Equal(b), "different content should not be equal")
	})
}
