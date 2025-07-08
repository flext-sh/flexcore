// Package adapter provides builder pattern for creating adapters
package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/flext/flexcore/pkg/result"
)

// AdapterBuilder provides a fluent interface for building adapters
type AdapterBuilder struct {
	name    string
	version string

	// Functions
	extractFn   ExtractFunc
	loadFn      LoadFunc
	transformFn TransformFunc
	healthFn    HealthCheckFunc
	configureFn ConfigureFunc

	// Options
	hooks      AdapterHooks
	middleware []Middleware

	// Validation
	validator func(interface{}) error

	// Error tracking
	err error
}

// Function types
type ExtractFunc func(context.Context, ExtractRequest) result.Result[*ExtractResponse]
type LoadFunc func(context.Context, LoadRequest) result.Result[*LoadResponse]
type TransformFunc func(context.Context, TransformRequest) result.Result[*TransformResponse]
type HealthCheckFunc func(context.Context) error
type ConfigureFunc func(map[string]interface{}) error

// Middleware for adapter operations
type Middleware func(Operation) Operation
type Operation func(context.Context, interface{}) (interface{}, error)

// NewAdapterBuilder creates a new adapter builder
func NewAdapterBuilder(name, version string) *AdapterBuilder {
	return &AdapterBuilder{
		name:       name,
		version:    version,
		middleware: make([]Middleware, 0),
	}
}

// WithExtract sets the extract function
func (b *AdapterBuilder) WithExtract(fn ExtractFunc) *AdapterBuilder {
	if b.err != nil {
		return b
	}
	b.extractFn = fn
	return b
}

// WithLoad sets the load function
func (b *AdapterBuilder) WithLoad(fn LoadFunc) *AdapterBuilder {
	if b.err != nil {
		return b
	}
	b.loadFn = fn
	return b
}

// WithTransform sets the transform function
func (b *AdapterBuilder) WithTransform(fn TransformFunc) *AdapterBuilder {
	if b.err != nil {
		return b
	}
	b.transformFn = fn
	return b
}

// WithHealthCheck sets the health check function
func (b *AdapterBuilder) WithHealthCheck(fn HealthCheckFunc) *AdapterBuilder {
	if b.err != nil {
		return b
	}
	b.healthFn = fn
	return b
}

// WithConfigure sets the configure function
func (b *AdapterBuilder) WithConfigure(fn ConfigureFunc) *AdapterBuilder {
	if b.err != nil {
		return b
	}
	b.configureFn = fn
	return b
}

// WithHooks sets adapter hooks
func (b *AdapterBuilder) WithHooks(hooks AdapterHooks) *AdapterBuilder {
	if b.err != nil {
		return b
	}
	b.hooks = hooks
	return b
}

// WithMiddleware adds middleware to the adapter
func (b *AdapterBuilder) WithMiddleware(mw ...Middleware) *AdapterBuilder {
	if b.err != nil {
		return b
	}
	b.middleware = append(b.middleware, mw...)
	return b
}

// WithValidator sets a validator function
func (b *AdapterBuilder) WithValidator(fn func(interface{}) error) *AdapterBuilder {
	if b.err != nil {
		return b
	}
	b.validator = fn
	return b
}

// Build creates the adapter
func (b *AdapterBuilder) Build() (Adapter, error) {
	if b.err != nil {
		return nil, b.err
	}

	// Validate required functions
	if b.extractFn == nil && b.loadFn == nil && b.transformFn == nil {
		return nil, fmt.Errorf("adapter must implement at least one operation (extract, load, or transform)")
	}

	return &builtAdapter{
		name:        b.name,
		version:     b.version,
		extractFn:   b.extractFn,
		loadFn:      b.loadFn,
		transformFn: b.transformFn,
		healthFn:    b.healthFn,
		configureFn: b.configureFn,
		hooks:       b.hooks,
		middleware:  b.middleware,
		validator:   b.validator,
	}, nil
}

// builtAdapter is the adapter created by the builder
type builtAdapter struct {
	name    string
	version string

	extractFn   ExtractFunc
	loadFn      LoadFunc
	transformFn TransformFunc
	healthFn    HealthCheckFunc
	configureFn ConfigureFunc

	hooks      AdapterHooks
	middleware []Middleware
	validator  func(interface{}) error

	config interface{}
}

func (a *builtAdapter) Name() string {
	return a.name
}

func (a *builtAdapter) Version() string {
	return a.version
}

func (a *builtAdapter) Configure(config map[string]interface{}) error {
	if a.configureFn != nil {
		return a.configureFn(config)
	}
	a.config = config
	return nil
}

func (a *builtAdapter) Extract(ctx context.Context, req ExtractRequest) (*ExtractResponse, error) {
	if a.extractFn == nil {
		return nil, fmt.Errorf("extract not implemented")
	}

	// Apply hooks
	if a.hooks.OnBeforeExtract != nil {
		if err := a.hooks.OnBeforeExtract(ctx, req); err != nil {
			return nil, err
		}
	}

	// Execute with middleware
	start := time.Now()
	result := a.extractFn(ctx, req)
	_ = time.Since(start)

	var resp *ExtractResponse
	var err error

	if result.IsSuccess() {
		resp = result.Value()
	} else {
		err = result.Error()
		if a.hooks.OnError != nil {
			a.hooks.OnError(ctx, err)
		}
	}

	// After hook
	if a.hooks.OnAfterExtract != nil {
		a.hooks.OnAfterExtract(ctx, req, resp, err)
	}

	return resp, err
}

func (a *builtAdapter) Load(ctx context.Context, req LoadRequest) (*LoadResponse, error) {
	if a.loadFn == nil {
		return nil, fmt.Errorf("load not implemented")
	}

	// Apply hooks
	if a.hooks.OnBeforeLoad != nil {
		if err := a.hooks.OnBeforeLoad(ctx, req); err != nil {
			return nil, err
		}
	}

	// Execute
	result := a.loadFn(ctx, req)

	var resp *LoadResponse
	var err error

	if result.IsSuccess() {
		resp = result.Value()
	} else {
		err = result.Error()
		if a.hooks.OnError != nil {
			a.hooks.OnError(ctx, err)
		}
	}

	// After hook
	if a.hooks.OnAfterLoad != nil {
		a.hooks.OnAfterLoad(ctx, req, resp, err)
	}

	return resp, err
}

func (a *builtAdapter) Transform(ctx context.Context, req TransformRequest) (*TransformResponse, error) {
	if a.transformFn == nil {
		return nil, fmt.Errorf("transform not implemented")
	}

	result := a.transformFn(ctx, req)
	if result.IsFailure() {
		return nil, result.Error()
	}

	return result.Value(), nil
}

func (a *builtAdapter) HealthCheck(ctx context.Context) error {
	if a.healthFn == nil {
		return nil // Assume healthy if no health check provided
	}
	return a.healthFn(ctx)
}

// Common middleware implementations

// LoggingMiddleware logs all operations
func LoggingMiddleware(logger func(string, ...interface{})) Middleware {
	return func(next Operation) Operation {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			start := time.Now()
			logger("Starting operation: %T", req)

			resp, err := next(ctx, req)

			duration := time.Since(start)
			if err != nil {
				logger("Operation failed after %v: %v", duration, err)
			} else {
				logger("Operation completed in %v", duration)
			}

			return resp, err
		}
	}
}

// RetryMiddleware adds retry logic
func RetryMiddleware(maxRetries int, backoff time.Duration) Middleware {
	return func(next Operation) Operation {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var lastErr error

			for i := 0; i <= maxRetries; i++ {
				if i > 0 {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(backoff * time.Duration(i)):
						// Continue with retry
					}
				}

				resp, err := next(ctx, req)
				if err == nil {
					return resp, nil
				}

				lastErr = err
			}

			return nil, fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
		}
	}
}

// TimeoutMiddleware adds timeout to operations
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Operation) Operation {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			return next(ctx, req)
		}
	}
}
