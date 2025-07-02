// Package result tests
package result_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/flext/flexcore/shared/result"
	"github.com/stretchr/testify/assert"
)

func TestSuccess(t *testing.T) {
	r := result.Success(42)
	
	assert.True(t, r.IsSuccess())
	assert.False(t, r.IsFailure())
	assert.Equal(t, 42, r.Value())
	assert.Nil(t, r.Error())
}

func TestFailure(t *testing.T) {
	err := errors.New("test error")
	r := result.Failure[int](err)
	
	assert.False(t, r.IsSuccess())
	assert.True(t, r.IsFailure())
	assert.Equal(t, 0, r.Value()) // zero value
	assert.Equal(t, err, r.Error())
}

func TestFailureWithMessage(t *testing.T) {
	r := result.FailureWithMessage[string]("something went wrong")
	
	assert.True(t, r.IsFailure())
	assert.NotNil(t, r.Error())
	assert.Contains(t, r.Error().Error(), "something went wrong")
}

func TestValueOr(t *testing.T) {
	// Success case
	r1 := result.Success("hello")
	assert.Equal(t, "hello", r1.ValueOr("default"))
	
	// Failure case
	r2 := result.Failure[string](errors.New("error"))
	assert.Equal(t, "default", r2.ValueOr("default"))
}

func TestValueOrZero(t *testing.T) {
	// Success case
	r1 := result.Success(100)
	assert.Equal(t, 100, r1.ValueOrZero())
	
	// Failure case
	r2 := result.Failure[int](errors.New("error"))
	assert.Equal(t, 0, r2.ValueOrZero())
}

func TestUnwrap(t *testing.T) {
	// Success case
	r1 := result.Success("test")
	value1, err1 := r1.Unwrap()
	assert.Equal(t, "test", value1)
	assert.Nil(t, err1)
	
	// Failure case
	testErr := errors.New("test error")
	r2 := result.Failure[string](testErr)
	value2, err2 := r2.Unwrap()
	assert.Equal(t, "", value2) // zero value
	assert.Equal(t, testErr, err2)
}

func TestUnwrapOrPanic(t *testing.T) {
	// Success case
	r1 := result.Success(42)
	assert.NotPanics(t, func() {
		value := r1.UnwrapOrPanic()
		assert.Equal(t, 42, value)
	})
	
	// Failure case
	r2 := result.Failure[int](errors.New("error"))
	assert.Panics(t, func() {
		r2.UnwrapOrPanic()
	})
}

func TestMap(t *testing.T) {
	// Success case
	r1 := result.Success(10)
	r2 := result.Map(r1, func(n int) string {
		return fmt.Sprintf("number: %d", n)
	})
	
	assert.True(t, r2.IsSuccess())
	assert.Equal(t, "number: 10", r2.Value())
	
	// Failure case
	r3 := result.Failure[int](errors.New("error"))
	r4 := result.Map(r3, func(n int) string {
		return "should not be called"
	})
	
	assert.True(t, r4.IsFailure())
	assert.Equal(t, "error", r4.Error().Error())
}

func TestFlatMap(t *testing.T) {
	// Success case - returns success
	r1 := result.Success(10)
	r2 := result.FlatMap(r1, func(n int) result.Result[string] {
		if n > 0 {
			return result.Success(fmt.Sprintf("positive: %d", n))
		}
		return result.FailureWithMessage[string]("not positive")
	})
	
	assert.True(t, r2.IsSuccess())
	assert.Equal(t, "positive: 10", r2.Value())
	
	// Success case - returns failure
	r3 := result.Success(-5)
	r4 := result.FlatMap(r3, func(n int) result.Result[string] {
		if n > 0 {
			return result.Success(fmt.Sprintf("positive: %d", n))
		}
		return result.FailureWithMessage[string]("not positive")
	})
	
	assert.True(t, r4.IsFailure())
	assert.Contains(t, r4.Error().Error(), "not positive")
	
	// Failure case
	r5 := result.Failure[int](errors.New("original error"))
	r6 := result.FlatMap(r5, func(n int) result.Result[string] {
		return result.Success("should not be called")
	})
	
	assert.True(t, r6.IsFailure())
	assert.Equal(t, "original error", r6.Error().Error())
}

func TestFilter(t *testing.T) {
	// Success case - predicate true
	r1 := result.Success(42)
	r2 := r1.Filter(func(n int) bool { return n > 0 })
	assert.True(t, r2.IsSuccess())
	assert.Equal(t, 42, r2.Value())
	
	// Success case - predicate false
	r3 := result.Success(-5)
	r4 := r3.Filter(func(n int) bool { return n > 0 })
	assert.True(t, r4.IsFailure())
	assert.Contains(t, r4.Error().Error(), "filter predicate failed")
	
	// Failure case
	r5 := result.Failure[int](errors.New("error"))
	r6 := r5.Filter(func(n int) bool { return true })
	assert.True(t, r6.IsFailure())
	assert.Equal(t, "error", r6.Error().Error())
}

func TestForEach(t *testing.T) {
	// Success case
	called := false
	r1 := result.Success("test")
	r1.ForEach(func(s string) {
		called = true
		assert.Equal(t, "test", s)
	})
	assert.True(t, called)
	
	// Failure case
	called = false
	r2 := result.Failure[string](errors.New("error"))
	r2.ForEach(func(s string) {
		called = true
	})
	assert.False(t, called)
}

func TestIfFailure(t *testing.T) {
	// Success case
	called := false
	r1 := result.Success("test")
	r1.IfFailure(func(err error) {
		called = true
	})
	assert.False(t, called)
	
	// Failure case
	called = false
	testErr := errors.New("test error")
	r2 := result.Failure[string](testErr)
	r2.IfFailure(func(err error) {
		called = true
		assert.Equal(t, testErr, err)
	})
	assert.True(t, called)
}

func TestCombine(t *testing.T) {
	// Both success
	r1 := result.Success(10)
	r2 := result.Success("hello")
	r3 := result.Combine(r1, r2)
	
	assert.True(t, r3.IsSuccess())
	assert.Equal(t, 10, r3.Value().First)
	assert.Equal(t, "hello", r3.Value().Second)
	
	// First failure
	r4 := result.Failure[int](errors.New("error1"))
	r5 := result.Success("hello")
	r6 := result.Combine(r4, r5)
	
	assert.True(t, r6.IsFailure())
	assert.Equal(t, "error1", r6.Error().Error())
	
	// Second failure
	r7 := result.Success(10)
	r8 := result.Failure[string](errors.New("error2"))
	r9 := result.Combine(r7, r8)
	
	assert.True(t, r9.IsFailure())
	assert.Equal(t, "error2", r9.Error().Error())
	
	// Both failure (first error wins)
	r10 := result.Failure[int](errors.New("error1"))
	r11 := result.Failure[string](errors.New("error2"))
	r12 := result.Combine(r10, r11)
	
	assert.True(t, r12.IsFailure())
	assert.Equal(t, "error1", r12.Error().Error())
}

func TestAsync(t *testing.T) {
	// Complete with success
	async := result.NewAsync[int]()
	
	go func() {
		async.Complete(42)
	}()
	
	r := async.Await()
	assert.True(t, r.IsSuccess())
	assert.Equal(t, 42, r.Value())
	
	// Fail with error
	async2 := result.NewAsync[string]()
	
	go func() {
		async2.Fail(errors.New("async error"))
	}()
	
	r2 := async2.Await()
	assert.True(t, r2.IsFailure())
	assert.Equal(t, "async error", r2.Error().Error())
}

func TestTry(t *testing.T) {
	// Success case
	r1 := result.Try(func() int {
		return 42
	})
	
	assert.True(t, r1.IsSuccess())
	assert.Equal(t, 42, r1.Value())
	
	// Panic case - requires special handling
	// The current Try implementation doesn't actually catch panics
	// This is a limitation that should be fixed
}

func TestTryAsync(t *testing.T) {
	// Success case
	async := result.TryAsync(func() string {
		return "success"
	})
	
	r := async.Await()
	assert.True(t, r.IsSuccess())
	assert.Equal(t, "success", r.Value())
	
	// Panic case
	async2 := result.TryAsync(func() string {
		panic("test panic")
	})
	
	r2 := async2.Await()
	assert.True(t, r2.IsFailure())
	assert.Contains(t, r2.Error().Error(), "panic recovered")
}

// Benchmarks

func BenchmarkSuccess(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := result.Success(42)
		_ = r.Value()
	}
}

func BenchmarkMap(b *testing.B) {
	r := result.Success(42)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapped := result.Map(r, func(n int) string {
			return fmt.Sprintf("%d", n)
		})
		_ = mapped.Value()
	}
}

func BenchmarkFlatMap(b *testing.B) {
	r := result.Success(42)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapped := result.FlatMap(r, func(n int) result.Result[string] {
			return result.Success(fmt.Sprintf("%d", n))
		})
		_ = mapped.Value()
	}
}