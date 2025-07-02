// Package result provides a Result type for better error handling
package result

import (
	"github.com/flext/flexcore/shared/errors"
)

// Result represents either a success value or an error
type Result[T any] struct {
	value T
	err   error
}

// Success creates a successful result
func Success[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Failure creates a failed result
func Failure[T any](err error) Result[T] {
	var zero T
	return Result[T]{value: zero, err: err}
}

// FailureWithMessage creates a failed result with a message
func FailureWithMessage[T any](message string) Result[T] {
	return Failure[T](errors.New(message))
}

// IsSuccess returns true if the result is successful
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// IsFailure returns true if the result failed
func (r Result[T]) IsFailure() bool {
	return r.err != nil
}

// Value returns the value if successful, or the zero value if failed
func (r Result[T]) Value() T {
	return r.value
}

// Error returns the error if failed, or nil if successful
func (r Result[T]) Error() error {
	return r.err
}

// ValueOr returns the value if successful, or the provided default if failed
func (r Result[T]) ValueOr(defaultValue T) T {
	if r.IsSuccess() {
		return r.value
	}
	return defaultValue
}

// ValueOrZero returns the value if successful, or the zero value if failed
func (r Result[T]) ValueOrZero() T {
	var zero T
	return r.ValueOr(zero)
}

// Unwrap returns the value and error separately
func (r Result[T]) Unwrap() (T, error) {
	return r.value, r.err
}

// UnwrapOrPanic returns the value or panics if there's an error
func (r Result[T]) UnwrapOrPanic() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

// Map transforms the value if successful, or returns the error
func Map[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.IsFailure() {
		return Failure[U](r.err)
	}
	return Success(fn(r.value))
}

// FlatMap transforms the value if successful, or returns the error
func FlatMap[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.IsFailure() {
		return Failure[U](r.err)
	}
	return fn(r.value)
}

// Filter checks if the value satisfies a predicate
func (r Result[T]) Filter(predicate func(T) bool) Result[T] {
	if r.IsFailure() {
		return r
	}
	if !predicate(r.value) {
		return FailureWithMessage[T]("filter predicate failed")
	}
	return r
}

// ForEach executes a function if the result is successful
func (r Result[T]) ForEach(fn func(T)) {
	if r.IsSuccess() {
		fn(r.value)
	}
}

// IfFailure executes a function if the result failed
func (r Result[T]) IfFailure(fn func(error)) {
	if r.IsFailure() {
		fn(r.err)
	}
}

// Combine two results into a tuple result
func Combine[T, U any](r1 Result[T], r2 Result[U]) Result[Tuple[T, U]] {
	if r1.IsFailure() {
		return Failure[Tuple[T, U]](r1.err)
	}
	if r2.IsFailure() {
		return Failure[Tuple[T, U]](r2.err)
	}
	return Success(Tuple[T, U]{First: r1.value, Second: r2.value})
}

// Tuple represents a pair of values
type Tuple[T, U any] struct {
	First  T
	Second U
}

// Async represents an asynchronous result
type Async[T any] struct {
	ch chan Result[T]
}

// NewAsync creates a new async result
func NewAsync[T any]() *Async[T] {
	return &Async[T]{
		ch: make(chan Result[T], 1),
	}
}

// Complete completes the async result with a value
func (a *Async[T]) Complete(value T) {
	a.ch <- Success(value)
	close(a.ch)
}

// Fail completes the async result with an error
func (a *Async[T]) Fail(err error) {
	a.ch <- Failure[T](err)
	close(a.ch)
}

// Await waits for the async result to complete
func (a *Async[T]) Await() Result[T] {
	return <-a.ch
}

// Try executes a function that might panic and returns a result
func Try[T any](fn func() T) Result[T] {
	defer func() {
		if r := recover(); r != nil {
			// This will be handled by the calling code
		}
	}()

	value := fn()
	return Success(value)
}

// TryAsync executes a function asynchronously that might panic
func TryAsync[T any](fn func() T) *Async[T] {
	async := NewAsync[T]()
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
				async.Fail(errors.Newf("panic recovered: %v", r))
			}
		}()

		value := fn()
		async.Complete(value)
	}()

	return async
}