package result

import "fmt"

// Result represents a computation result that can succeed or fail
type Result[T any] struct {
	value T
	err   error
}

// Ok creates a successful Result with the given value
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value, err: nil}
}

// Error creates a failed Result with the given error
func Error[T any](err error) Result[T] {
	var zero T
	return Result[T]{value: zero, err: err}
}

// Errorf creates a failed Result with a formatted error message
func Errorf[T any](format string, args ...interface{}) Result[T] {
	return Error[T](fmt.Errorf(format, args...))
}

// IsSuccess returns true if the Result represents a successful operation
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// IsFailure returns true if the Result represents a failed operation
func (r Result[T]) IsFailure() bool {
	return r.err != nil
}

// Value returns the value if the Result is successful
// Panics if called on a failed Result
func (r Result[T]) Value() T {
	if r.IsFailure() {
		panic("called Value() on a failed Result")
	}
	return r.value
}

// Error returns the error if the Result is failed
// Returns nil if the Result is successful
func (r Result[T]) Error() error {
	return r.err
}

// ValueOr returns the value if successful, otherwise returns the default value
func (r Result[T]) ValueOr(defaultValue T) T {
	if r.IsSuccess() {
		return r.value
	}
	return defaultValue
}

// Map applies a function to the value if the Result is successful
func (r Result[T]) Map(fn func(T) T) Result[T] {
	if r.IsFailure() {
		return Error[T](r.err)
	}
	return Ok(fn(r.value))
}

// FlatMap applies a function that returns a Result to the value if successful
func (r Result[T]) FlatMap(fn func(T) Result[T]) Result[T] {
	if r.IsFailure() {
		return Error[T](r.err)
	}
	return fn(r.value)
}
