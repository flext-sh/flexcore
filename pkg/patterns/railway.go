// Package patterns provides Railway-Oriented Programming for FlexCore
package patterns

import (
	"github.com/flext/flexcore/pkg/result"
	"github.com/flext/flexcore/pkg/errors"
)

// Railway represents a computation that can succeed or fail
type Railway[T any] struct {
	result result.Result[T]
}

// Track creates a new Railway from a Result
func Track[T any](r result.Result[T]) Railway[T] {
	return Railway[T]{result: r}
}

// Success creates a successful Railway
func Success[T any](value T) Railway[T] {
	return Railway[T]{result: result.Success(value)}
}

// Failure creates a failed Railway
func Failure[T any](err error) Railway[T] {
	return Railway[T]{result: result.Failure[T](err)}
}

// Then chains operations on the success track
func (r Railway[T]) Then(fn func(T) error) Railway[T] {
	if r.result.IsFailure() {
		return r
	}

	if err := fn(r.result.Value()); err != nil {
		return Failure[T](err)
	}

	return r
}

// Map transforms the value on the success track
func (r Railway[T]) Map(fn func(T) T) Railway[T] {
	if r.result.IsFailure() {
		return r
	}

	return Success(fn(r.result.Value()))
}

// FlatMap chains Railway operations
func (r Railway[T]) FlatMap(fn func(T) Railway[T]) Railway[T] {
	if r.result.IsFailure() {
		return r
	}

	return fn(r.result.Value())
}

// Recover handles errors and returns to success track
func (r Railway[T]) Recover(fn func(error) T) Railway[T] {
	if r.result.IsSuccess() {
		return r
	}

	return Success(fn(r.result.Error()))
}

// Result returns the underlying Result
func (r Railway[T]) Result() result.Result[T] {
	return r.result
}

// Validate adds validation to the railway
func (r Railway[T]) Validate(predicate func(T) bool, errorMsg string) Railway[T] {
	if r.result.IsFailure() {
		return r
	}

	if !predicate(r.result.Value()) {
		return Failure[T](errors.New(errorMsg))
	}

	return r
}

// Tap allows side effects without changing the value
func (r Railway[T]) Tap(fn func(T)) Railway[T] {
	if r.result.IsSuccess() {
		fn(r.result.Value())
	}
	return r
}

// TapError allows side effects on errors
func (r Railway[T]) TapError(fn func(error)) Railway[T] {
	if r.result.IsFailure() {
		fn(r.result.Error())
	}
	return r
}
