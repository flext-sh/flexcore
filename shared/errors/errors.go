package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ValidationError ErrorType = "VALIDATION"
	NotFoundError   ErrorType = "NOT_FOUND"
	ConflictError   ErrorType = "CONFLICT"
	InternalError   ErrorType = "INTERNAL"
	ExternalError   ErrorType = "EXTERNAL"
)

// DomainError represents a structured error with context
type DomainError struct {
	Type       ErrorType
	Message    string
	Cause      error
	Context    map[string]interface{}
	StackTrace []string
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *DomainError) WithContext(key string, value interface{}) *DomainError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New creates a new DomainError
func New(errType ErrorType, message string) *DomainError {
	return &DomainError{
		Type:       errType,
		Message:    message,
		Context:    make(map[string]interface{}),
		StackTrace: captureStackTrace(),
	}
}

// Wrap creates a new DomainError wrapping an existing error
func Wrap(errType ErrorType, message string, cause error) *DomainError {
	return &DomainError{
		Type:       errType,
		Message:    message,
		Cause:      cause,
		Context:    make(map[string]interface{}),
		StackTrace: captureStackTrace(),
	}
}

// NewValidation creates a validation error
func NewValidation(message string) *DomainError {
	return New(ValidationError, message)
}

// NewNotFound creates a not found error
func NewNotFound(resource string) *DomainError {
	return New(NotFoundError, fmt.Sprintf("%s not found", resource))
}

// NewConflict creates a conflict error
func NewConflict(message string) *DomainError {
	return New(ConflictError, message)
}

// NewInternal creates an internal error
func NewInternal(message string) *DomainError {
	return New(InternalError, message)
}

// NewExternal creates an external error
func NewExternal(message string, cause error) *DomainError {
	return Wrap(ExternalError, message, cause)
}

// captureStackTrace captures the current stack trace
func captureStackTrace() []string {
	var traces []string
	for i := 2; i < 10; i++ { // Skip current function and New/Wrap
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Shorten file path for readability
		if idx := strings.LastIndex(file, "/"); idx != -1 {
			file = file[idx+1:]
		}
		traces = append(traces, fmt.Sprintf("%s:%d", file, line))
	}
	return traces
}
