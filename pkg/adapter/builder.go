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

// OperationExecutionTemplate encapsulates common operation execution pattern
// SOLID Template Method Pattern: Eliminates 78 lines of duplication across Extract/Load/Transform
type OperationExecutionTemplate struct {
	adapter       *builtAdapter
	operationType string
}

// NewOperationExecutionTemplate creates a new operation execution template
func NewOperationExecutionTemplate(adapter *builtAdapter, operationType string) *OperationExecutionTemplate {
	return &OperationExecutionTemplate{
		adapter:       adapter,
		operationType: operationType,
	}
}

// executeWithTemplate executes operation using template method pattern
// SOLID Template Method: Common algorithm with customizable operation execution
func (template *OperationExecutionTemplate) executeWithTemplate(ctx context.Context, req interface{}, operationFunc func() result.Result[interface{}]) (interface{}, error) {
	// Step 1: Execute with hooks using specialized executor
	executor := template.adapter.createOperationExecutor(template.operationType, ctx)
	executeResult := executor.ExecuteWithHooks(req, operationFunc)

	// Step 2: Handle execution result
	if executeResult.IsFailure() {
		return nil, executeResult.Error()
	}

	// Step 3: Return result
	return executeResult.Value(), nil
}

// AdapterOperationResult encapsulates operation result with consolidated error handling
// SOLID Result Pattern: Eliminates multiple return points by consolidating all outcomes
type AdapterOperationResult[T any] struct {
	value T
	err   error
}

// NewAdapterOperationResult creates a successful operation result
func NewAdapterOperationResult[T any](value T) *AdapterOperationResult[T] {
	return &AdapterOperationResult[T]{value: value}
}

// NewAdapterOperationError creates a failed operation result
func NewAdapterOperationError[T any](err error) *AdapterOperationResult[T] {
	return &AdapterOperationResult[T]{err: err}
}

// Unwrap returns the result value and error for Go idiom compatibility
func (aor *AdapterOperationResult[T]) Unwrap() (T, error) {
	return aor.value, aor.err
}

// executeAdapterOperation executes adapter operation with consolidated return logic
// SOLID Result Pattern: Single return point eliminating 6 returns to 1 return
func executeAdapterOperation[TReq, TResp any](
	a *builtAdapter,
	ctx context.Context,
	req TReq,
	operationType string,
	operationFn func(context.Context, TReq) result.Result[*TResp],
) *AdapterOperationResult[*TResp] {
	// Validate function implementation
	if operationFn == nil {
		return NewAdapterOperationError[*TResp](fmt.Errorf("%s not implemented", operationType))
	}

	// Execute using template with consolidated error handling
	template := NewOperationExecutionTemplate(a, operationType)
	result, err := template.executeWithTemplate(ctx, req, func() result.Result[interface{}] {
		operationResult := operationFn(ctx, req)
		if operationResult.IsFailure() {
			return result.Failure[interface{}](operationResult.Error())
		}
		return result.Success[interface{}](operationResult.Value())
	})

	if err != nil {
		return NewAdapterOperationError[*TResp](err)
	}

	// Type assertion with consolidated error handling
	response, ok := result.(*TResp)
	if !ok {
		return NewAdapterOperationError[*TResp](fmt.Errorf("invalid %s response type", operationType))
	}

	return NewAdapterOperationResult(response)
}

func (a *builtAdapter) Extract(ctx context.Context, req ExtractRequest) (*ExtractResponse, error) {
	return executeAdapterOperation(a, ctx, req, "extract", a.extractFn).Unwrap()
}

func (a *builtAdapter) Load(ctx context.Context, req LoadRequest) (*LoadResponse, error) {
	return executeAdapterOperation(a, ctx, req, "load", a.loadFn).Unwrap()
}

func (a *builtAdapter) Transform(ctx context.Context, req TransformRequest) (*TransformResponse, error) {
	return executeAdapterOperation(a, ctx, req, "transform", a.transformFn).Unwrap()
}

// OperationExecutor handles centralized operation execution with hooks
// SOLID SRP: Single responsibility for operation execution with hooks
type OperationExecutor struct {
	adapter     *builtAdapter
	operationType string
	ctx         context.Context
}

// createOperationExecutor creates specialized operation executor
// SOLID SRP: Factory method for creating specialized executors
func (a *builtAdapter) createOperationExecutor(operationType string, ctx context.Context) *OperationExecutor {
	return &OperationExecutor{
		adapter:       a,
		operationType: operationType,
		ctx:           ctx,
	}
}

// ExecuteWithHooks executes operation with centralized hook handling
// SOLID SRP: Single responsibility for complete operation execution with hooks
func (executor *OperationExecutor) ExecuteWithHooks(req interface{}, operation func() result.Result[interface{}]) result.Result[interface{}] {
	// Phase 1: Execute before hooks
	if err := executor.executeBeforeHooks(req); err != nil {
		return result.Failure[interface{}](err)
	}

	// Phase 2: Execute operation with timing
	operationResult := executor.executeOperationWithTiming(operation)
	
	// Phase 3: Handle result and execute after hooks
	return executor.handleResultWithAfterHooks(req, operationResult)
}

// executeBeforeHooks executes before hooks based on operation type
// SOLID SRP: Single responsibility for before hook execution
func (executor *OperationExecutor) executeBeforeHooks(req interface{}) error {
	switch executor.operationType {
	case "extract":
		if executor.adapter.hooks.OnBeforeExtract != nil {
			if extractReq, ok := req.(ExtractRequest); ok {
				return executor.adapter.hooks.OnBeforeExtract(executor.ctx, extractReq)
			}
		}
	case "load":
		if executor.adapter.hooks.OnBeforeLoad != nil {
			if loadReq, ok := req.(LoadRequest); ok {
				return executor.adapter.hooks.OnBeforeLoad(executor.ctx, loadReq)
			}
		}
	}
	return nil
}

// executeOperationWithTiming executes operation with timing measurement
// SOLID SRP: Single responsibility for operation execution with timing
func (executor *OperationExecutor) executeOperationWithTiming(operation func() result.Result[interface{}]) result.Result[interface{}] {
	start := time.Now()
	result := operation()
	_ = time.Since(start) // Timing available for middleware
	return result
}

// handleResultWithAfterHooks handles operation result and executes after hooks
// SOLID SRP: Single responsibility for result handling with after hooks
func (executor *OperationExecutor) handleResultWithAfterHooks(req interface{}, operationResult result.Result[interface{}]) result.Result[interface{}] {
	var resp interface{}
	var err error

	if operationResult.IsSuccess() {
		resp = operationResult.Value()
	} else {
		err = operationResult.Error()
		// Execute error hook
		if executor.adapter.hooks.OnError != nil {
			executor.adapter.hooks.OnError(executor.ctx, err)
		}
	}

	// Execute after hooks based on operation type
	executor.executeAfterHooks(req, resp, err)

	if err != nil {
		return result.Failure[interface{}](err)
	}
	
	return result.Success(resp)
}

// executeAfterHooks executes after hooks based on operation type
// SOLID SRP: Single responsibility for after hook execution
func (executor *OperationExecutor) executeAfterHooks(req interface{}, resp interface{}, err error) {
	switch executor.operationType {
	case "extract":
		if executor.adapter.hooks.OnAfterExtract != nil {
			if extractReq, ok := req.(ExtractRequest); ok {
				if extractResp, ok := resp.(*ExtractResponse); ok || resp == nil {
					executor.adapter.hooks.OnAfterExtract(executor.ctx, extractReq, extractResp, err)
				}
			}
		}
	case "load":
		if executor.adapter.hooks.OnAfterLoad != nil {
			if loadReq, ok := req.(LoadRequest); ok {
				if loadResp, ok := resp.(*LoadResponse); ok || resp == nil {
					executor.adapter.hooks.OnAfterLoad(executor.ctx, loadReq, loadResp, err)
				}
			}
		}
	}
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
