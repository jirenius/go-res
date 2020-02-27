package test

import (
	"errors"
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test InternalError method coverts an unknown error to a system.internalError *Error.
func TestInternalError(t *testing.T) {
	e := res.InternalError(errors.New("foo"))
	restest.AssertEqualJSON(t, "error code", e.Code, res.CodeInternalError)
}

// Test ToError method coverts an unknown error to a system.internalError *Error.
func TestToErrorWithConversion(t *testing.T) {
	e := res.ToError(errors.New("foo"))
	restest.AssertEqualJSON(t, "error code", e.Code, res.CodeInternalError)
}

// Test ToError method does not alter an error of type *Error.
func TestToErrorWithNoConversion(t *testing.T) {
	e := res.ToError(res.ErrMethodNotFound)
	restest.AssertEqualJSON(t, "Error", e, res.ErrMethodNotFound)
}

// Test Error method to return the error message string
func TestErrorMethod(t *testing.T) {
	e := &res.Error{
		Code:    mock.CustomErrorCode,
		Message: mock.ErrorMessage,
	}
	restest.AssertEqualJSON(t, "Error", e.Error(), mock.ErrorMessage)
}
