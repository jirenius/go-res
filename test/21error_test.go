package test

import (
	"errors"
	"testing"

	res "github.com/jirenius/go-res"
)

// Test InternalError method coverts an unknown error to a system.internalError *Error.
func TestInternalError(t *testing.T) {
	e := res.InternalError(errors.New("foo"))
	AssertEqual(t, "error code", e.Code, res.CodeInternalError)
}

// Test ToError method coverts an unknown error to a system.internalError *Error.
func TestToErrorWithConversion(t *testing.T) {
	e := res.ToError(errors.New("foo"))
	AssertEqual(t, "error code", e.Code, res.CodeInternalError)
}

// Test ToError method does not alter an error of type *Error.
func TestToErrorWithNoConversion(t *testing.T) {
	e := res.ToError(res.ErrMethodNotFound)
	AssertEqual(t, "Error", e, res.ErrMethodNotFound)
}

// Test Error method to return the error message string
func TestErrorMethod(t *testing.T) {
	e := &res.Error{
		Code:    mock.CustomErrorCode,
		Message: mock.ErrorMessage,
	}
	AssertEqual(t, "Error", e.Error(), mock.ErrorMessage)
}
