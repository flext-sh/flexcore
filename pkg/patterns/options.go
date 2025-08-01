// Package patterns provides advanced functional patterns for FlexCore
package patterns

import (
)

// Option represents a functional option pattern
type Option[T any] func(*T) error

// Apply applies all options to a target
func Apply[T any](target *T, opts ...Option[T]) error {
	for _, opt := range opts {
		if err := opt(target); err != nil {
			return err
		}
	}
	return nil
}

// Maybe represents an optional value
type Maybe[T any] struct {
	value   T
	present bool
}

// Some creates a Maybe with a value
func Some[T any](value T) Maybe[T] {
	return Maybe[T]{value: value, present: true}
}

// None creates an empty Maybe
func None[T any]() Maybe[T] {
	var zero T
	return Maybe[T]{value: zero, present: false}
}

// IsPresent returns true if the Maybe contains a value
func (m Maybe[T]) IsPresent() bool {
	return m.present
}

// IsEmpty returns true if the Maybe is empty
func (m Maybe[T]) IsEmpty() bool {
	return !m.present
}

// Get returns the value and a boolean indicating if it exists
func (m Maybe[T]) Get() (T, bool) {
	return m.value, m.present
}

// OrElse returns the value if present, otherwise returns the default
func (m Maybe[T]) OrElse(defaultValue T) T {
	if m.present {
		return m.value
	}
	return defaultValue
}

// OrElseGet returns the value if present, otherwise calls the supplier
func (m Maybe[T]) OrElseGet(supplier func() T) T {
	if m.present {
		return m.value
	}
	return supplier()
}

// Map transforms the value if present
func Map[T, U any](m Maybe[T], mapper func(T) U) Maybe[U] {
	if m.present {
		return Some(mapper(m.value))
	}
	return None[U]()
}

// FlatMap transforms the value if present, flattening the result
func FlatMap[T, U any](m Maybe[T], mapper func(T) Maybe[U]) Maybe[U] {
	if m.present {
		return mapper(m.value)
	}
	return None[U]()
}

// Filter returns the Maybe if the predicate is true, otherwise returns None
func (m Maybe[T]) Filter(predicate func(T) bool) Maybe[T] {
	if m.present && predicate(m.value) {
		return m
	}
	return None[T]()
}

// IfPresent executes the consumer if a value is present
func (m Maybe[T]) IfPresent(consumer func(T)) {
	if m.present {
		consumer(m.value)
	}
}

// IfPresentOrElse executes the consumer if present, otherwise executes the empty action
func (m Maybe[T]) IfPresentOrElse(consumer func(T), emptyAction func()) {
	if m.present {
		consumer(m.value)
	} else {
		emptyAction()
	}
}


// Lazy represents a lazy-evaluated value
type Lazy[T any] struct {
	fn       func() T
	value    T
	computed bool
}

// NewLazy creates a new lazy value
func NewLazy[T any](fn func() T) *Lazy[T] {
	return &Lazy[T]{fn: fn}
}

// Get returns the value, computing it if necessary
func (l *Lazy[T]) Get() T {
	if !l.computed {
		l.value = l.fn()
		l.computed = true
	}
	return l.value
}

// Reset clears the cached value
func (l *Lazy[T]) Reset() {
	l.computed = false
	var zero T
	l.value = zero
}

// Either represents a value that can be one of two types
type Either[L, R any] struct {
	left   L
	right  R
	isLeft bool
}

// Left creates a left Either
func Left[L, R any](value L) Either[L, R] {
	return Either[L, R]{left: value, isLeft: true}
}

// Right creates a right Either
func Right[L, R any](value R) Either[L, R] {
	return Either[L, R]{right: value, isLeft: false}
}

// IsLeft returns true if this is a left value
func (e Either[L, R]) IsLeft() bool {
	return e.isLeft
}

// IsRight returns true if this is a right value
func (e Either[L, R]) IsRight() bool {
	return !e.isLeft
}

// Left returns the left value
func (e Either[L, R]) Left() (L, bool) {
	return e.left, e.isLeft
}

// Right returns the right value
func (e Either[L, R]) Right() (R, bool) {
	return e.right, !e.isLeft
}

// Fold applies one of two functions based on the Either type
func Fold[L, R, T any](e Either[L, R], leftFn func(L) T, rightFn func(R) T) T {
	if e.isLeft {
		return leftFn(e.left)
	}
	return rightFn(e.right)
}
