package res

// Error represents an RES error
type Error struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// ToError converts an error to an *Error. If it isn't of type *Error already, it will become a system.internalError.
func ToError(err error) *Error {
	rerr, ok := err.(*Error)
	if !ok {
		rerr = InternalError(err)
	}
	return rerr
}

// InternalError converts an error to an *Error with the code system.internalError.
func InternalError(err error) *Error {
	return &Error{Code: CodeInternalError, Message: "Internal error: " + err.Error()}
}

// Predefined error codes
const (
	CodeAccessDenied   = "system.accessDenied"
	CodeInternalError  = "system.internalError"
	CodeInvalidParams  = "system.invalidParams"
	CodeInvalidQuery   = "system.invalidQuery"
	CodeMethodNotFound = "system.methodNotFound"
	CodeNotFound       = "system.notFound"
	CodeTimeout        = "system.timeout"
)

// Predefined errors
var (
	ErrAccessDenied   = &Error{Code: CodeAccessDenied, Message: "Access denied"}
	ErrInternalError  = &Error{Code: CodeInternalError, Message: "Internal error"}
	ErrInvalidParams  = &Error{Code: CodeInvalidParams, Message: "Invalid parameters"}
	ErrInvalidQuery   = &Error{Code: CodeInvalidQuery, Message: "Invalid query"}
	ErrMethodNotFound = &Error{Code: CodeMethodNotFound, Message: "Method not found"}
	ErrNotFound       = &Error{Code: CodeNotFound, Message: "Not found"}
	ErrTimeout        = &Error{Code: CodeTimeout, Message: "Request timeout"}
)
