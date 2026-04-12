package errors

import (
	"fmt"
)

// ErrorCode represents application-specific error codes
type ErrorCode string

const (
	ErrCodeUnknown            ErrorCode = "UNKNOWN"
	ErrCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrCodeInvalidInput       ErrorCode = "INVALID_INPUT"
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrCodeConflict           ErrorCode = "CONFLICT"
	ErrCodeTimeout            ErrorCode = "TIMEOUT"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// AppError represents an application error with code and details
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Cause   error     `json:"cause,omitempty"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with a new code and message
func Wrap(code ErrorCode, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NotFound creates a not found error
func NotFound(resource string, id string) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found: %s", resource, id))
}

// InvalidInput creates an invalid input error
func InvalidInput(field string, reason string) *AppError {
	return New(ErrCodeInvalidInput, fmt.Sprintf("invalid %s: %s", field, reason))
}

// InternalError creates an internal error
func InternalError(message string, cause error) *AppError {
	return Wrap(ErrCodeInternalError, message, cause)
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) *AppError {
	return New(ErrCodeUnauthorized, message)
}

// Forbidden creates a forbidden error
func Forbidden(message string) *AppError {
	return New(ErrCodeForbidden, message)
}

// Conflict creates a conflict error
func Conflict(resource string) *AppError {
	return New(ErrCodeConflict, fmt.Sprintf("resource conflict: %s", resource))
}

// Timeout creates a timeout error
func Timeout(operation string) *AppError {
	return New(ErrCodeTimeout, fmt.Sprintf("operation timed out: %s", operation))
}

// ServiceUnavailable creates a service unavailable error
func ServiceUnavailable(service string) *AppError {
	return New(ErrCodeServiceUnavailable, fmt.Sprintf("service unavailable: %s", service))
}
