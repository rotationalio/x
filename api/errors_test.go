package api_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"go.rtnl.ai/x/api"
	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/validation"
)

func TestSentinelReplies(t *testing.T) {
	assert.Equal(t, api.Reply{Success: false}, api.Unsuccessful)
	assert.Equal(t, api.Reply{Success: false, Error: "resource not found"}, api.NotFound)
	assert.Equal(t, api.Reply{Success: false, Error: "method not allowed"}, api.NotAllowed)
}

func TestError(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		assert.Equal(t, api.Unsuccessful, api.Error(nil))
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		verrs := validation.Errors{
			validation.Missing("email"),
			validation.Incorrect("age", "must not be negative"),
		}

		rep := api.Error(verrs)
		assert.False(t, rep.Success)
		assert.Equal(t, "2 validation errors occurred", rep.Error)
		assert.Equal(t, api.ErrorDetail{
			{Field: "email", Error: "missing email: this field is required"},
			{Field: "age", Error: "invalid age: must not be negative"},
		}, rep.ErrorDetail)
	})

	t.Run("EmptyValidationErrors", func(t *testing.T) {
		rep := api.Error(validation.Errors{})
		assert.False(t, rep.Success)
		assert.Equal(t, "0 validation errors occurred", rep.Error)
		assert.Equal(t, api.ErrorDetail{}, rep.ErrorDetail)
	})

	t.Run("FieldError", func(t *testing.T) {
		ferr := validation.Missing("email")
		rep := api.Error(ferr)
		assert.False(t, rep.Success)
		assert.Equal(t, "missing email: this field is required", rep.Error)
		assert.Equal(t, api.ErrorDetail{
			{Field: "email", Error: "missing email: this field is required"},
		}, rep.ErrorDetail)
	})

	t.Run("Error", func(t *testing.T) {
		rep := api.Error(errors.New("something broke"))
		assert.False(t, rep.Success)
		assert.Equal(t, "something broke", rep.Error)
		assert.Nil(t, rep.ErrorDetail)
	})

	t.Run("String", func(t *testing.T) {
		rep := api.Error("plain string error")
		assert.False(t, rep.Success)
		assert.Equal(t, "plain string error", rep.Error)
		assert.Nil(t, rep.ErrorDetail)
	})

	t.Run("Stringer", func(t *testing.T) {
		rep := api.Error(testStringer("from stringer"))
		assert.False(t, rep.Success)
		assert.Equal(t, "from stringer", rep.Error)
		assert.Nil(t, rep.ErrorDetail)
	})

	t.Run("JSONMarshaler", func(t *testing.T) {
		rep := api.Error(testMarshaler(`{"code":"bad_request"}`))
		assert.False(t, rep.Success)
		assert.Equal(t, `{"code":"bad_request"}`, rep.Error)
		assert.Nil(t, rep.ErrorDetail)
	})

	t.Run("JSONMarshalerPanic", func(t *testing.T) {
		m := failingMarshaler{}
		assert.PanicsWithValue(t, m, func() {
			api.Error(m)
		})
	})

	t.Run("Unhandled", func(t *testing.T) {
		rep := api.Error(42)
		assert.False(t, rep.Success)
		assert.Equal(t, "unhandled error response", rep.Error)
		assert.Nil(t, rep.ErrorDetail)
	})

	t.Run("ErrorTakesPrecedenceOverStringer", func(t *testing.T) {
		rep := api.Error(errorStringer{})
		assert.Equal(t, "from error", rep.Error)
	})
}

func TestStatusError(t *testing.T) {
	inner := errors.New("resource missing")
	err := &api.StatusError{Code: http.StatusNotFound, Err: inner}

	t.Run("Error", func(t *testing.T) {
		assert.EqualError(t, err, "[404] resource missing")
	})

	t.Run("Unwrap", func(t *testing.T) {
		assert.Equal(t, inner, err.Unwrap())
		assert.ErrorIs(t, err, inner)
	})

	t.Run("Reply", func(t *testing.T) {
		rep := err.Reply()
		assert.False(t, rep.Success)
		assert.Equal(t, "resource missing", rep.Error)
		assert.Nil(t, rep.ErrorDetail)
	})

	t.Run("ReplyValidationErrors", func(t *testing.T) {
		verrs := validation.Errors{validation.Missing("query")}
		serr := &api.StatusError{Code: http.StatusBadRequest, Err: verrs}

		rep := serr.Reply()
		assert.False(t, rep.Success)
		assert.Equal(t, "1 validation errors occurred", rep.Error)
		assert.Equal(t, api.ErrorDetail{
			{Field: "query", Error: "missing query: this field is required"},
		}, rep.ErrorDetail)
	})
}

func TestStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
	}{
		{"Nil", nil, http.StatusOK},
		{"PlainError", errors.New("boom"), http.StatusInternalServerError},
		{"NotFound", &api.StatusError{Code: http.StatusNotFound, Err: errors.New("missing")}, http.StatusNotFound},
		{"BadRequest", &api.StatusError{Code: http.StatusBadRequest, Err: errors.New("invalid")}, http.StatusBadRequest},
		{"MinValid", &api.StatusError{Code: 100, Err: errors.New("continue")}, 100},
		{"MaxValid", &api.StatusError{Code: 599, Err: errors.New("timeout")}, 599},
		{"BelowMin", &api.StatusError{Code: 99, Err: errors.New("too low")}, http.StatusInternalServerError},
		{"AboveMax", &api.StatusError{Code: 600, Err: errors.New("too high")}, http.StatusInternalServerError},
		{"ZeroCode", &api.StatusError{Code: 0, Err: errors.New("unset")}, http.StatusInternalServerError},
		{"Wrapped", fmt.Errorf("wrap: %w", &api.StatusError{Code: http.StatusNotFound, Err: errors.New("missing")}), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.code, api.StatusCode(tc.err))
		})
	}
}

func TestDetailErrorJSON(t *testing.T) {
	detail := api.ErrorDetail{
		{Field: "email", Error: "this field is required"},
	}

	data, err := json.Marshal(detail)
	assert.Ok(t, err)
	assert.Equal(t, `[{"field":"email","error":"this field is required"}]`, string(data))
}

type testStringer string

func (s testStringer) String() string { return string(s) }

type testMarshaler string

func (m testMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(m), nil
}

type failingMarshaler struct{}

func (failingMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failed")
}

type errorStringer struct{}

func (errorStringer) Error() string  { return "from error" }
func (errorStringer) String() string { return "from stringer" }
