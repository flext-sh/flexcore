// Package errors contains comprehensive tests for error handling
package errors

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFlexError_Creation(t *testing.T) {
	tests := []struct {
		name      string
		createErr func() *FlexError
		wantMsg   string
		wantCode  string
	}{
		{
			name: "new error",
			createErr: func() *FlexError {
				return New("test error")
			},
			wantMsg: "test error",
		},
		{
			name: "formatted error",
			createErr: func() *FlexError {
				return Newf("test error %d", 42)
			},
			wantMsg: "test error 42",
		},
		{
			name: "validation error",
			createErr: func() *FlexError {
				return ValidationError("invalid input")
			},
			wantMsg:  "invalid input",
			wantCode: CodeValidation,
		},
		{
			name: "not found error",
			createErr: func() *FlexError {
				return NotFoundError("user")
			},
			wantMsg:  "user not found",
			wantCode: CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createErr()
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), tt.wantMsg)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, err.Code())
			}
			assert.NotEmpty(t, err.Location())
		})
	}
}

func TestFlexError_Wrapping(t *testing.T) {
	original := errors.New("original error")

	tests := []struct {
		name      string
		wrapFunc  func() *FlexError
		wantMsg   string
		wantCause bool
	}{
		{
			name: "wrap error",
			wrapFunc: func() *FlexError {
				return Wrap(original, "wrapped")
			},
			wantMsg:   "wrapped: original error",
			wantCause: true,
		},
		{
			name: "wrap formatted",
			wrapFunc: func() *FlexError {
				return Wrapf(original, "wrapped %d", 42)
			},
			wantMsg:   "wrapped 42: original error",
			wantCause: true,
		},
		{
			name: "wrap nil",
			wrapFunc: func() *FlexError {
				return Wrap(nil, "no error")
			},
			wantCause: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wrapFunc()
			if tt.wantCause {
				assert.NotNil(t, err)
				assert.Equal(t, tt.wantMsg, err.Error())
				assert.Equal(t, original, err.Unwrap())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestFlexError_WithMethods(t *testing.T) {
	err := New("test error")

	// Test WithCode
	errWithCode := err.WithCode("TEST_CODE")
	assert.Equal(t, "TEST_CODE", errWithCode.Code())
	assert.Same(t, err, errWithCode) // Should return same instance

	// Test WithOperation
	errWithOp := err.WithOperation("test_operation")
	assert.Equal(t, "test_operation", errWithOp.Operation())
	assert.Same(t, err, errWithOp) // Should return same instance

	// Test chaining
	errorChained := New("chained").WithCode("CHAIN").WithOperation("chain_op")
	assert.Equal(t, "CHAIN", errorChained.Code())
	assert.Equal(t, "chain_op", errorChained.Operation())
}

func TestFlexError_AllErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		createErr func() *FlexError
		wantCode  string
	}{
		{"validation", func() *FlexError { return ValidationError("invalid") }, CodeValidation},
		{"not found", func() *FlexError { return NotFoundError("item") }, CodeNotFound},
		{"already exists", func() *FlexError { return AlreadyExistsError("item") }, CodeAlreadyExists},
		{"unauthorized", func() *FlexError { return UnauthorizedError("access denied") }, CodeUnauthorized},
		{"forbidden", func() *FlexError { return ForbiddenError("forbidden") }, CodeForbidden},
		{"internal", func() *FlexError { return InternalError("internal") }, CodeInternal},
		{"network", func() *FlexError { return NetworkError("network") }, CodeNetwork},
		{"timeout", func() *FlexError { return TimeoutError("timeout") }, CodeTimeout},
		{"system", func() *FlexError { return SystemError("system") }, CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createErr()
			assert.NotNil(t, err)
			assert.Equal(t, tt.wantCode, err.Code())
			assert.NotEmpty(t, err.Error())
			assert.NotEmpty(t, err.Location())
		})
	}
}

func TestFlexError_ErrorInterface(t *testing.T) {
	original := errors.New("original")
	wrapped := Wrap(original, "wrapped")

	// Test error interface
	var err error = wrapped
	assert.Equal(t, "wrapped: original", err.Error())

	// Test unwrapping
	unwrapped := errors.Unwrap(err)
	assert.Equal(t, original, unwrapped)

	// Test errors.Is
	assert.True(t, errors.Is(wrapped, original))
}

func TestFlexError_Location(t *testing.T) {
	err := New("test")
	location := err.Location()

	// Should contain filename and line number
	assert.Contains(t, location, "errors_test.go")
	assert.Contains(t, location, ":")
}

func TestFlexError_Performance(t *testing.T) {
	// Performance test: creating 10000 errors should be fast
	start := time.Now()
	for i := 0; i < 10000; i++ {
		err := Newf("error %d", i)
		assert.NotNil(t, err)
	}
	duration := time.Since(start)

	// Should create 10000 errors in less than 100ms
	assert.Less(t, duration, 100*time.Millisecond, "Error creation too slow: %v", duration)
}

func TestFlexError_Concurrency(t *testing.T) {
	// Test concurrent error creation
	const numGoroutines = 100
	var wg sync.WaitGroup
	errorChan := make(chan *FlexError, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			err := Newf("concurrent error %d", index)
			errorChan <- err
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Verify all errors were created
	errorCount := 0
	for err := range errorChan {
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "concurrent error")
		errorCount++
	}

	assert.Equal(t, numGoroutines, errorCount)
}

func BenchmarkFlexError_New(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := New("benchmark error")
			if err == nil {
				b.Error("Error creation failed")
			}
		}
	})
}

func BenchmarkFlexError_Newf(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := Newf("benchmark error %d", 42)
			if err == nil {
				b.Error("Error creation failed")
			}
		}
	})
}

func BenchmarkFlexError_Wrap(b *testing.B) {
	original := errors.New("original")
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := Wrap(original, "wrapped")
			if err == nil {
				b.Error("Error wrapping failed")
			}
		}
	})
}
