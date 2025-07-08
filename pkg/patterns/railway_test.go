// Package patterns provides Railway-Oriented Programming for FlexCore
package patterns

import (
	"errors"
	"testing"

	"github.com/flext/flexcore/pkg/result"
	"github.com/stretchr/testify/assert"
)

func TestTrack(t *testing.T) {
	tests := []struct {
		name   string
		result result.Result[string]
		want   string
		isErr  bool
	}{
		{
			name:   "success result",
			result: result.Success("test-value"),
			want:   "test-value",
			isErr:  false,
		},
		{
			name:   "failure result",
			result: result.Failure[string](errors.New("test error")),
			isErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			railway := Track(tt.result)
			if tt.isErr {
				assert.True(t, railway.Result().IsFailure())
				assert.Equal(t, "test error", railway.Result().Error().Error())
			} else {
				assert.True(t, railway.Result().IsSuccess())
				assert.Equal(t, tt.want, railway.Result().Value())
			}
		})
	}
}

func TestSuccess(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{"string value", "test"},
		{"integer value", 42},
		{"boolean value", true},
		{"nil value", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			railway := Success(tt.value)
			assert.True(t, railway.Result().IsSuccess())
			assert.Equal(t, tt.value, railway.Result().Value())
		})
	}
}

func TestFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"simple error", errors.New("simple error")},
		{"custom error", &customError{msg: "custom error"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			railway := Failure[string](tt.err)
			assert.True(t, railway.Result().IsFailure())
			assert.Equal(t, tt.err, railway.Result().Error())
		})
	}
}

func TestRailway_Then(t *testing.T) {
	tests := []struct {
		name     string
		initial  Railway[string]
		thenFunc func(string) error
		wantErr  bool
		wantVal  string
	}{
		{
			name:    "success then success",
			initial: Success("test"),
			thenFunc: func(s string) error {
				if s != "test" {
					return errors.New("unexpected value")
				}
				return nil
			},
			wantErr: false,
			wantVal: "test",
		},
		{
			name:    "success then failure",
			initial: Success("test"),
			thenFunc: func(s string) error {
				return errors.New("then failed")
			},
			wantErr: true,
		},
		{
			name:    "failure then not called",
			initial: Failure[string](errors.New("initial error")),
			thenFunc: func(s string) error {
				// Should not be called
				t.Error("Then function should not be called on failure")
				return nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.Then(tt.thenFunc)
			if tt.wantErr {
				assert.True(t, result.Result().IsFailure())
			} else {
				assert.True(t, result.Result().IsSuccess())
				assert.Equal(t, tt.wantVal, result.Result().Value())
			}
		})
	}
}

func TestRailway_Map(t *testing.T) {
	tests := []struct {
		name    string
		initial Railway[string]
		mapFunc func(string) string
		wantErr bool
		wantVal string
	}{
		{
			name:    "success map",
			initial: Success("hello"),
			mapFunc: func(s string) string {
				return s + " world"
			},
			wantErr: false,
			wantVal: "hello world",
		},
		{
			name:    "failure map not called",
			initial: Failure[string](errors.New("error")),
			mapFunc: func(s string) string {
				// Should not be called
				t.Error("Map function should not be called on failure")
				return s
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.Map(tt.mapFunc)
			if tt.wantErr {
				assert.True(t, result.Result().IsFailure())
			} else {
				assert.True(t, result.Result().IsSuccess())
				assert.Equal(t, tt.wantVal, result.Result().Value())
			}
		})
	}
}

func TestRailway_FlatMap(t *testing.T) {
	tests := []struct {
		name        string
		initial     Railway[string]
		flatMapFunc func(string) Railway[string]
		wantErr     bool
		wantVal     string
	}{
		{
			name:    "success flatmap to success",
			initial: Success("hello"),
			flatMapFunc: func(s string) Railway[string] {
				return Success(s + " world")
			},
			wantErr: false,
			wantVal: "hello world",
		},
		{
			name:    "success flatmap to failure",
			initial: Success("hello"),
			flatMapFunc: func(s string) Railway[string] {
				return Failure[string](errors.New("flatmap failed"))
			},
			wantErr: true,
		},
		{
			name:    "failure flatmap not called",
			initial: Failure[string](errors.New("initial error")),
			flatMapFunc: func(s string) Railway[string] {
				// Should not be called
				t.Error("FlatMap function should not be called on failure")
				return Success(s)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.FlatMap(tt.flatMapFunc)
			if tt.wantErr {
				assert.True(t, result.Result().IsFailure())
			} else {
				assert.True(t, result.Result().IsSuccess())
				assert.Equal(t, tt.wantVal, result.Result().Value())
			}
		})
	}
}

func TestRailway_Recover(t *testing.T) {
	tests := []struct {
		name        string
		initial     Railway[string]
		recoverFunc func(error) string
		wantErr     bool
		wantVal     string
	}{
		{
			name:    "failure recover to success",
			initial: Failure[string](errors.New("original error")),
			recoverFunc: func(err error) string {
				return "recovered: " + err.Error()
			},
			wantErr: false,
			wantVal: "recovered: original error",
		},
		{
			name:    "success recover not called",
			initial: Success("original value"),
			recoverFunc: func(err error) string {
				// Should not be called
				t.Error("Recover function should not be called on success")
				return "should not reach"
			},
			wantErr: false,
			wantVal: "original value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.Recover(tt.recoverFunc)
			if tt.wantErr {
				assert.True(t, result.Result().IsFailure())
			} else {
				assert.True(t, result.Result().IsSuccess())
				assert.Equal(t, tt.wantVal, result.Result().Value())
			}
		})
	}
}

func TestRailway_Validate(t *testing.T) {
	tests := []struct {
		name      string
		initial   Railway[string]
		predicate func(string) bool
		errorMsg  string
		wantErr   bool
		wantVal   string
	}{
		{
			name:    "success validate pass",
			initial: Success("valid"),
			predicate: func(s string) bool {
				return len(s) > 0
			},
			errorMsg: "string should not be empty",
			wantErr:  false,
			wantVal:  "valid",
		},
		{
			name:    "success validate fail",
			initial: Success(""),
			predicate: func(s string) bool {
				return len(s) > 0
			},
			errorMsg: "string should not be empty",
			wantErr:  true,
		},
		{
			name:    "failure validate not called",
			initial: Failure[string](errors.New("original error")),
			predicate: func(s string) bool {
				// Should not be called
				t.Error("Validate predicate should not be called on failure")
				return true
			},
			errorMsg: "should not reach",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.Validate(tt.predicate, tt.errorMsg)
			if tt.wantErr {
				assert.True(t, result.Result().IsFailure())
				if tt.initial.Result().IsSuccess() {
					assert.Contains(t, result.Result().Error().Error(), tt.errorMsg)
				}
			} else {
				assert.True(t, result.Result().IsSuccess())
				assert.Equal(t, tt.wantVal, result.Result().Value())
			}
		})
	}
}

func TestRailway_Tap(t *testing.T) {
	called := false
	var tappedValue string

	t.Run("success tap called", func(t *testing.T) {
		called = false
		railway := Success("test-value")
		result := railway.Tap(func(s string) {
			called = true
			tappedValue = s
		})

		assert.True(t, called)
		assert.Equal(t, "test-value", tappedValue)
		assert.True(t, result.Result().IsSuccess())
		assert.Equal(t, "test-value", result.Result().Value())
	})

	t.Run("failure tap not called", func(t *testing.T) {
		called = false
		railway := Failure[string](errors.New("error"))
		result := railway.Tap(func(s string) {
			called = true
		})

		assert.False(t, called)
		assert.True(t, result.Result().IsFailure())
	})
}

func TestRailway_TapError(t *testing.T) {
	called := false
	var tappedError error

	t.Run("failure tap error called", func(t *testing.T) {
		called = false
		originalErr := errors.New("test-error")
		railway := Failure[string](originalErr)
		result := railway.TapError(func(err error) {
			called = true
			tappedError = err
		})

		assert.True(t, called)
		assert.Equal(t, originalErr, tappedError)
		assert.True(t, result.Result().IsFailure())
		assert.Equal(t, originalErr, result.Result().Error())
	})

	t.Run("success tap error not called", func(t *testing.T) {
		called = false
		railway := Success("test-value")
		result := railway.TapError(func(err error) {
			called = true
		})

		assert.False(t, called)
		assert.True(t, result.Result().IsSuccess())
	})
}

// Test chaining multiple railway operations
func TestRailway_Chaining(t *testing.T) {
	t.Run("success chain", func(t *testing.T) {
		result := Success("hello")
			.Map(func(s string) string { return s + " world" })
			.Then(func(s string) error {
				if len(s) < 5 {
					return errors.New("too short")
				}
				return nil
			})
			.Validate(func(s string) bool { return len(s) > 10 }, "must be longer than 10")
			.Tap(func(s string) {
				// Side effect
			})

		assert.True(t, result.Result().IsSuccess())
		assert.Equal(t, "hello world", result.Result().Value())
	})

	t.Run("failure chain stops early", func(t *testing.T) {
		result := Success("hi")
			.Validate(func(s string) bool { return len(s) > 5 }, "too short")
			.Map(func(s string) string {
				// Should not be called
				t.Error("Map should not be called after validation failure")
				return s + " world"
			})

		assert.True(t, result.Result().IsFailure())
		assert.Contains(t, result.Result().Error().Error(), "too short")
	})

	t.Run("recover and continue", func(t *testing.T) {
		result := Failure[string](errors.New("initial error"))
			.Recover(func(err error) string { return "recovered" })
			.Map(func(s string) string { return s + " value" })

		assert.True(t, result.Result().IsSuccess())
		assert.Equal(t, "recovered value", result.Result().Value())
	})
}

// Custom error type for testing
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

// Benchmark tests for performance validation
func BenchmarkRailway_SuccessChain(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := Success("test")
		result = result.Map(func(s string) string { return s + "1" })
		result = result.Map(func(s string) string { return s + "2" })
		result = result.Map(func(s string) string { return s + "3" })
		_ = result
	}
}

func BenchmarkRailway_FailureChain(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := Failure[string](errors.New("error"))
		result = result.Map(func(s string) string { return s + "1" })
		result = result.Map(func(s string) string { return s + "2" })
		result = result.Map(func(s string) string { return s + "3" })
		_ = result
	}
}
