package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.rtnl.ai/x/validation"
)

//===========================================================================
// Standard Error Handling
//===========================================================================

var (
	Unsuccessful = Reply{Success: false}
	NotFound     = Reply{Success: false, Error: "resource not found"}
	NotAllowed   = Reply{Success: false, Error: "method not allowed"}
)

// Construct a new response for an error or simply return unsuccessful.
func Error(err interface{}) Reply {
	if err == nil {
		return Unsuccessful
	}

	rep := Reply{Success: false}
	switch err := err.(type) {
	case validation.Errors:
		rep.Error = fmt.Sprintf("%d validation errors occurred", len(err))
		rep.ErrorDetail = make(ErrorDetail, 0, len(err))
		for _, verr := range err {
			rep.ErrorDetail = append(rep.ErrorDetail, &DetailError{
				Field: verr.Field(),
				Error: verr.Error(),
			})
		}
	case *validation.FieldError:
		rep.Error = err.Error()
		rep.ErrorDetail = ErrorDetail{
			&DetailError{
				Field: err.Field(),
				Error: err.Error(),
			},
		}
	case error:
		rep.Error = err.Error()
	case string:
		rep.Error = err
	case fmt.Stringer:
		rep.Error = err.String()
	case json.Marshaler:
		data, e := err.MarshalJSON()
		if e != nil {
			panic(err)
		}
		rep.Error = string(data)
	default:
		rep.Error = "unhandled error response"
	}

	return rep
}

//===========================================================================
// Status Errors
//===========================================================================

// Wraps an error with an HTTP status code.
type StatusError struct {
	Code int
	Err  error
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Err.Error())
}

func (e *StatusError) Unwrap() error {
	return e.Err
}

func (e *StatusError) Reply() Reply {
	return Error(e.Err)
}

// StatusCode returns the HTTP status code from an error or 500 if the error is not a StatusError.
func StatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if e, ok := err.(*StatusError); !ok || e.Code < 100 || e.Code >= 600 {
		return http.StatusInternalServerError
	} else {
		return e.Code
	}
}

//===========================================================================
// Detail Error
//===========================================================================

type ErrorDetail []*DetailError

type DetailError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}
