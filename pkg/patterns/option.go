// Package patterns provides advanced Go patterns for FlexCore
package patterns

// Option represents an optional value
type Option[T any] struct {
	value   T
	present bool
}

// Some creates an Option with a value
func Some[T any](value T) Option[T] {
	return Option[T]{value: value, present: true}
}

// None creates an empty Option
func None[T any]() Option[T] {
	var zero T
	return Option[T]{value: zero, present: false}
}

// IsSome returns true if the Option contains a value
func (o Option[T]) IsSome() bool {
	return o.present
}

// IsNone returns true if the Option is empty
func (o Option[T]) IsNone() bool {
	return !o.present
}

// Unwrap returns the value, panics if None
func (o Option[T]) Unwrap() T {
	if !o.present {
		panic("called Unwrap on None value")
	}
	return o.value
}

// UnwrapOr returns the value or a default
func (o Option[T]) UnwrapOr(defaultValue T) T {
	if o.present {
		return o.value
	}
	return defaultValue
}

// UnwrapOrElse returns the value or calls a function
func (o Option[T]) UnwrapOrElse(fn func() T) T {
	if o.present {
		return o.value
	}
	return fn()
}

// Map transforms the value if present
func MapOption[T, U any](o Option[T], fn func(T) U) Option[U] {
	if o.IsNone() {
		return None[U]()
	}
	return Some(fn(o.value))
}

// FlatMap transforms the value if present
func FlatMapOption[T, U any](o Option[T], fn func(T) Option[U]) Option[U] {
	if o.IsNone() {
		return None[U]()
	}
	return fn(o.value)
}

// Filter returns None if the predicate fails
func (o Option[T]) Filter(predicate func(T) bool) Option[T] {
	if o.IsNone() || !predicate(o.value) {
		return None[T]()
	}
	return o
}

// ForEach executes a function if the Option has a value
func (o Option[T]) ForEach(fn func(T)) {
	if o.IsSome() {
		fn(o.value)
	}
}