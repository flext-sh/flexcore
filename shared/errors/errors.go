// Package errors provides error handling utilities for FlexCore
package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// Constants for caller depth
const (
	callerDepth = 2         // Number of stack frames to skip to get the actual caller
	unknownFile = "unknown" // Default file name when caller cannot be determined
)

// FlexError represents a FlexCore error with context
type FlexError struct {
	message   string
	cause     error
	code      string
	operation string
	file      string
	line      int
}

// Error implements the error interface
func (e *FlexError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// Unwrap returns the underlying error
func (e *FlexError) Unwrap() error {
	return e.cause
}

// Code returns the error code
func (e *FlexError) Code() string {
	return e.code
}

// Operation returns the operation that caused the error
func (e *FlexError) Operation() string {
	return e.operation
}

// Location returns the file and line where the error occurred
func (e *FlexError) Location() string {
	return fmt.Sprintf("%s:%d", e.file, e.line)
}

// New creates a new FlexError
func New(message string) *FlexError {
	file, line := getCaller(callerDepth)
	return &FlexError{
		message: message,
		file:    file,
		line:    line,
	}
}

// Newf creates a new FlexError with formatted message
func Newf(format string, args ...interface{}) *FlexError {
	file, line := getCaller(callerDepth)
	return &FlexError{
		message: fmt.Sprintf(format, args...),
		file:    file,
		line:    line,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, message string) *FlexError {
	if err == nil {
		return nil
	}

	file, line := getCaller(callerDepth)
	return &FlexError{
		message: message,
		cause:   err,
		file:    file,
		line:    line,
	}
}

// Wrapf wraps an existing error with formatted message
func Wrapf(err error, format string, args ...interface{}) *FlexError {
	if err == nil {
		return nil
	}

	file, line := getCaller(callerDepth)
	return &FlexError{
		message: fmt.Sprintf(format, args...),
		cause:   err,
		file:    file,
		line:    line,
	}
}

// WithCode adds an error code
func (e *FlexError) WithCode(code string) *FlexError {
	e.code = code
	return e
}

// WithOperation adds the operation context
func (e *FlexError) WithOperation(operation string) *FlexError {
	e.operation = operation
	return e
}

// getCaller returns the file and line of the caller
func getCaller(skip int) (file string, line int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return unknownFile, 0
	}

	// Extract just the filename
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return file, line
}

// Common error codes
const (
	CodeValidation    = "VALIDATION_ERROR"
	CodeNotFound      = "NOT_FOUND"
	CodeAlreadyExists = "ALREADY_EXISTS"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeForbidden     = "FORBIDDEN"
	CodeInternal      = "INTERNAL_ERROR"
	CodeNetwork       = "NETWORK_ERROR"
	CodeTimeout       = "TIMEOUT_ERROR"
)

// Predefined error constructors
func ValidationError(message string) *FlexError {
	return New(message).WithCode(CodeValidation)
}

func NotFoundError(resource string) *FlexError {
	return Newf("%s not found", resource).WithCode(CodeNotFound)
}

func AlreadyExistsError(resource string) *FlexError {
	return Newf("%s already exists", resource).WithCode(CodeAlreadyExists)
}

func UnauthorizedError(message string) *FlexError {
	return New(message).WithCode(CodeUnauthorized)
}

func ForbiddenError(message string) *FlexError {
	return New(message).WithCode(CodeForbidden)
}

func InternalError(message string) *FlexError {
	return New(message).WithCode(CodeInternal)
}

func NetworkError(message string) *FlexError {
	return New(message).WithCode(CodeNetwork)
}

func TimeoutError(message string) *FlexError {
	return New(message).WithCode(CodeTimeout)
}

func SystemError(message string) *FlexError {
	return New(message).WithCode(CodeInternal)
}
