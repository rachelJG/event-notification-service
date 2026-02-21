package apperror

import "fmt"

type Code string

const (
	CodeInvalidArgument  Code = "invalid_argument"
	CodeUnauthenticated  Code = "unauthenticated"
	CodePermissionDenied Code = "permission_denied"
	CodeNotFound         Code = "not_found"
	CodeConflict         Code = "conflict"
	CodeTimeout          Code = "timeout"
	CodeRateLimited      Code = "rate_limited"
	CodeInternal         Code = "internal"
)

type AppError struct {
	Code    Code
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code Code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func InvalidArgument(message string, err error) *AppError {
	return New(CodeInvalidArgument, message, err)
}

func Unauthenticated(message string, err error) *AppError {
	return New(CodeUnauthenticated, message, err)
}

func PermissionDenied(message string, err error) *AppError {
	return New(CodePermissionDenied, message, err)
}

func NotFound(message string, err error) *AppError {
	return New(CodeNotFound, message, err)
}

func Conflict(message string, err error) *AppError {
	return New(CodeConflict, message, err)
}

func Timeout(message string, err error) *AppError {
	return New(CodeTimeout, message, err)
}

func Internal(message string, err error) *AppError {
	return New(CodeInternal, message, err)
}
