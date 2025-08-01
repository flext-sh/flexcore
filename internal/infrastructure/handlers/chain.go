// Package handlers provides advanced handler patterns for FlexCore
package handlers

import (
	"context"
	"time"

	"github.com/flext/flexcore/pkg/result"
	"github.com/flext/flexcore/shared/errors"
)

const (
	// Cache configuration constants
	defaultCacheTTLMinutes = 5
)

// Context key types to avoid collisions
type contextKey string

const (
	userContextKey contextKey = "user"
)

// Handler represents a request handler
type Handler interface {
	Handle(ctx context.Context, req Request) result.Result[Response]
}

// Request represents a generic request
type Request interface {
	Context() context.Context
	ID() string
	Method() string
	Path() string
	Headers() map[string][]string
	Body() interface{}
}

// Response represents a generic response
type Response interface {
	StatusCode() int
	Headers() map[string][]string
	Body() interface{}
}

// Middleware represents a handler middleware
type Middleware func(Handler) Handler

// Chain represents a middleware chain
type Chain struct {
	middlewares []Middleware
}

// NewChain creates a new middleware chain
func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{middlewares: middlewares}
}

// Then chains the middlewares and returns final handler
func (c *Chain) Then(handler Handler) Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

// Append appends middlewares to the chain
func (c *Chain) Append(middlewares ...Middleware) *Chain {
	newMiddlewares := make([]Middleware, 0, len(c.middlewares)+len(middlewares))
	newMiddlewares = append(newMiddlewares, c.middlewares...)
	newMiddlewares = append(newMiddlewares, middlewares...)
	return &Chain{middlewares: newMiddlewares}
}

// Extend extends the chain with another chain
func (c *Chain) Extend(chain *Chain) *Chain {
	return c.Append(chain.middlewares...)
}

// HandlerFunc is an adapter to allow functions as handlers
type HandlerFunc func(ctx context.Context, req Request) result.Result[Response]

// Handle implements Handler interface
func (f HandlerFunc) Handle(ctx context.Context, req Request) result.Result[Response] {
	return f(ctx, req)
}

// Common Middlewares

// DRY PRINCIPLE: LoggingMiddleware moved to observability_middleware.go to reduce file complexity

// DRY PRINCIPLE: RecoveryMiddleware moved to infrastructure_middleware.go

// DRY PRINCIPLE: TimeoutMiddleware moved to infrastructure_middleware.go

// RetryMiddleware adds retry logic using Strategy Pattern
// SOLID SRP: Reduced from 26 complexity to ~8 by delegating to RetryExecutor
// SOLID OCP: Open for extension through RetryStrategy interface
func RetryMiddleware(maxRetries int, backoff time.Duration) Middleware {
	return RetryMiddlewareWithStrategy(NewStandardRetryStrategy(maxRetries, backoff))
}

// RetryMiddlewareWithStrategy creates retry middleware with custom strategy
// SOLID DIP: Depends on abstraction (RetryStrategy) not concretion
func RetryMiddlewareWithStrategy(strategy RetryStrategy) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			executor := NewRetryExecutor(strategy)
			
			// Define the operation to retry
			var lastResult result.Result[Response]
			operation := func() error {
				lastResult = next.Handle(ctx, req)
				if lastResult.IsFailure() {
					return lastResult.Error()
				}
				return nil
			}
			
			// Execute with retry
			retryResult := executor.ExecuteWithRetry(ctx, operation)
			
			if !retryResult.Success {
				return result.Failure[Response](
					errors.Wrapf(retryResult.FinalError, 
						"failed after %d attempts in %v", 
						retryResult.AttemptsMade, retryResult.TotalDelay))
			}
			
			return lastResult
		})
	}
}

// DRY PRINCIPLE: AuthenticationMiddleware moved to security_middleware.go

// DRY PRINCIPLE: ValidationMiddleware moved to security_middleware.go

// DRY PRINCIPLE: RateLimitingMiddleware moved to security_middleware.go

// DRY PRINCIPLE: MetricsMiddleware moved to observability_middleware.go

// DRY PRINCIPLE: TracingMiddleware moved to observability_middleware.go

// DRY PRINCIPLE: CORSMiddleware moved to security_middleware.go

// DRY PRINCIPLE: CompressionMiddleware moved to infrastructure_middleware.go

// DRY PRINCIPLE: CacheMiddleware moved to infrastructure_middleware.go

// DRY PRINCIPLE: TransformMiddleware moved to infrastructure_middleware.go

// Basic implementations

// BasicRequest is a basic request implementation
type BasicRequest struct {
	ctx     context.Context
	id      string
	method  string
	path    string
	headers map[string][]string
	body    interface{}
}

func (r *BasicRequest) Context() context.Context     { return r.ctx }
func (r *BasicRequest) ID() string                   { return r.id }
func (r *BasicRequest) Method() string               { return r.method }
func (r *BasicRequest) Path() string                 { return r.path }
func (r *BasicRequest) Headers() map[string][]string { return r.headers }
func (r *BasicRequest) Body() interface{}            { return r.body }

// BasicResponse is a basic response implementation
type BasicResponse struct {
	status  int
	headers map[string][]string
	body    interface{}
}

func (r *BasicResponse) StatusCode() int              { return r.status }
func (r *BasicResponse) Headers() map[string][]string { return r.headers }
func (r *BasicResponse) Body() interface{}            { return r.body }

// ChainBuilder provides fluent interface for building chains
type ChainBuilder struct {
	chain *Chain
}

// NewChainBuilder creates a new chain builder
func NewChainBuilder() *ChainBuilder {
	return &ChainBuilder{
		chain: NewChain(),
	}
}

// Use adds middleware to the chain
func (b *ChainBuilder) Use(middleware Middleware) *ChainBuilder {
	b.chain = b.chain.Append(middleware)
	return b
}

// UseLogging adds logging middleware
func (b *ChainBuilder) UseLogging(logger interface{ Info(string, ...interface{}) }) *ChainBuilder {
	return b.Use(LoggingMiddleware(logger))
}

// UseRecovery adds recovery middleware
func (b *ChainBuilder) UseRecovery() *ChainBuilder {
	return b.Use(RecoveryMiddleware())
}

// UseTimeout adds timeout middleware
func (b *ChainBuilder) UseTimeout(timeout time.Duration) *ChainBuilder {
	return b.Use(TimeoutMiddleware(timeout))
}

// UseRetry adds retry middleware
func (b *ChainBuilder) UseRetry(maxRetries int, backoff time.Duration) *ChainBuilder {
	return b.Use(RetryMiddleware(maxRetries, backoff))
}

// UseAuthentication adds authentication middleware
func (b *ChainBuilder) UseAuthentication(auth Authenticator) *ChainBuilder {
	return b.Use(AuthenticationMiddleware(auth))
}

// UseValidation adds validation middleware
func (b *ChainBuilder) UseValidation(validator Validator) *ChainBuilder {
	return b.Use(ValidationMiddleware(validator))
}

// UseRateLimiting adds rate limiting middleware
func (b *ChainBuilder) UseRateLimiting(limiter RateLimiter) *ChainBuilder {
	return b.Use(RateLimitingMiddleware(limiter))
}

// UseMetrics adds metrics middleware
func (b *ChainBuilder) UseMetrics(collector MetricsCollector) *ChainBuilder {
	return b.Use(MetricsMiddleware(collector))
}

// UseTracing adds tracing middleware
func (b *ChainBuilder) UseTracing(tracer Tracer) *ChainBuilder {
	return b.Use(TracingMiddleware(tracer))
}

// Build returns the built chain
func (b *ChainBuilder) Build() *Chain {
	return b.chain
}
