// Package handlers provides advanced handler patterns for FlexCore
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flext/flexcore/pkg/errors"
	"github.com/flext/flexcore/pkg/result"
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

// LoggingMiddleware logs requests and responses
func LoggingMiddleware(logger interface{ Info(string, ...interface{}) }) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			start := time.Now()
			
			logger.Info("Request started",
				"id", req.ID(),
				"method", req.Method(),
				"path", req.Path(),
			)
			
			result := next.Handle(ctx, req)
			
			duration := time.Since(start)
			
			if result.IsSuccess() {
				resp := result.Value()
				logger.Info("Request completed",
					"id", req.ID(),
					"status", resp.StatusCode(),
					"duration", duration,
				)
			} else {
				logger.Info("Request failed",
					"id", req.ID(),
					"error", result.Error(),
					"duration", duration,
				)
			}
			
			return result
		})
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (resp result.Result[Response]) {
			defer func() {
				if r := recover(); r != nil {
					err := errors.InternalError(fmt.Sprintf("panic recovered: %v", r))
					resp = result.Failure[Response](err)
				}
			}()
			
			return next.Handle(ctx, req)
		})
	}
}

// TimeoutMiddleware adds timeout to requests
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			
			done := make(chan result.Result[Response], 1)
			
			go func() {
				done <- next.Handle(ctx, req)
			}()
			
			select {
			case result := <-done:
				return result
			case <-ctx.Done():
				return result.Failure[Response](errors.TimeoutError("request timeout"))
			}
		})
	}
}

// RetryMiddleware adds retry logic
func RetryMiddleware(maxRetries int, backoff time.Duration) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			var lastErr error
			
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					select {
					case <-time.After(backoff * time.Duration(attempt)):
					case <-ctx.Done():
						return result.Failure[Response](ctx.Err())
					}
				}
				
				result := next.Handle(ctx, req)
				if result.IsSuccess() {
					return result
				}
				
				lastErr = result.Error()
				
				// Don't retry on client errors
				if httpErr, ok := lastErr.(*errors.FlexError); ok {
					if httpErr.Code() == errors.CodeValidation ||
					   httpErr.Code() == errors.CodeUnauthorized ||
					   httpErr.Code() == errors.CodeForbidden {
						return result
					}
				}
			}
			
			return result.Failure[Response](errors.Wrapf(lastErr, "failed after %d retries", maxRetries))
		})
	}
}

// AuthenticationMiddleware adds authentication
func AuthenticationMiddleware(authenticator Authenticator) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			authResult := authenticator.Authenticate(ctx, req)
			if authResult.IsFailure() {
				return result.Failure[Response](authResult.Error())
			}
			
			// Add authenticated user to context
			user := authResult.Value()
			ctx = context.WithValue(ctx, "user", user)
			
			return next.Handle(ctx, req)
		})
	}
}

// Authenticator interface
type Authenticator interface {
	Authenticate(ctx context.Context, req Request) result.Result[interface{}]
}

// ValidationMiddleware validates requests
func ValidationMiddleware(validator Validator) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			if err := validator.Validate(req); err != nil {
				return result.Failure[Response](errors.ValidationError(err.Error()))
			}
			
			return next.Handle(ctx, req)
		})
	}
}

// Validator interface
type Validator interface {
	Validate(req Request) error
}

// RateLimitingMiddleware adds rate limiting
func RateLimitingMiddleware(limiter RateLimiter) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			if !limiter.Allow(req.ID()) {
				return result.Failure[Response](errors.ForbiddenError("rate limit exceeded"))
			}
			
			return next.Handle(ctx, req)
		})
	}
}

// RateLimiter interface
type RateLimiter interface {
	Allow(key string) bool
}

// MetricsMiddleware collects metrics
func MetricsMiddleware(collector MetricsCollector) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			start := time.Now()
			
			result := next.Handle(ctx, req)
			
			duration := time.Since(start)
			status := 0
			
			if result.IsSuccess() {
				status = result.Value().StatusCode()
			} else {
				status = http.StatusInternalServerError
			}
			
			collector.RecordRequest(req.Method(), req.Path(), status, duration)
			
			return result
		})
	}
}

// MetricsCollector interface
type MetricsCollector interface {
	RecordRequest(method, path string, status int, duration time.Duration)
}

// TracingMiddleware adds distributed tracing
func TracingMiddleware(tracer Tracer) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			span := tracer.StartSpan(ctx, "handler.request")
			defer span.End()
			
			span.SetAttribute("request.id", req.ID())
			span.SetAttribute("request.method", req.Method())
			span.SetAttribute("request.path", req.Path())
			
			ctx = span.Context()
			result := next.Handle(ctx, req)
			
			if result.IsSuccess() {
				span.SetAttribute("response.status", result.Value().StatusCode())
			} else {
				span.SetError(result.Error())
			}
			
			return result
		})
	}
}

// Tracer interface for distributed tracing
type Tracer interface {
	StartSpan(ctx context.Context, name string) Span
}

// Span interface for tracing spans
type Span interface {
	Context() context.Context
	SetAttribute(key string, value interface{})
	SetError(err error)
	End()
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(config CORSConfig) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			// Handle preflight
			if req.Method() == "OPTIONS" {
				return result.Success[Response](&BasicResponse{
					status: http.StatusNoContent,
					headers: map[string][]string{
						"Access-Control-Allow-Origin":  config.AllowedOrigins,
						"Access-Control-Allow-Methods": config.AllowedMethods,
						"Access-Control-Allow-Headers": config.AllowedHeaders,
					},
				})
			}
			
			result := next.Handle(ctx, req)
			
			// Add CORS headers to response
			if result.IsSuccess() {
				resp := result.Value()
				headers := resp.Headers()
				headers["Access-Control-Allow-Origin"] = config.AllowedOrigins
			}
			
			return result
		})
	}
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CompressionMiddleware adds response compression
func CompressionMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			result := next.Handle(ctx, req)
			
			if result.IsSuccess() {
				// In real implementation, compress response body
				// based on Accept-Encoding header
			}
			
			return result
		})
	}
}

// CacheMiddleware adds caching
func CacheMiddleware(cache Cache) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			// Only cache GET requests
			if req.Method() != "GET" {
				return next.Handle(ctx, req)
			}
			
			cacheKey := fmt.Sprintf("%s:%s", req.Method(), req.Path())
			
			// Check cache
			if cached := cache.Get(cacheKey); cached != nil {
				if resp, ok := cached.(Response); ok {
					return result.Success(resp)
				}
			}
			
			// Execute handler
			result := next.Handle(ctx, req)
			
			// Cache successful responses
			if result.IsSuccess() {
				cache.Set(cacheKey, result.Value(), time.Minute*5)
			}
			
			return result
		})
	}
}

// Cache interface
type Cache interface {
	Get(key string) interface{}
	Set(key string, value interface{}, ttl time.Duration)
}

// TransformMiddleware transforms requests and responses
func TransformMiddleware(reqTransformer RequestTransformer, respTransformer ResponseTransformer) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			// Transform request
			if reqTransformer != nil {
				transformedReq, err := reqTransformer.Transform(req)
				if err != nil {
					return result.Failure[Response](err)
				}
				req = transformedReq
			}
			
			// Execute handler
			res := next.Handle(ctx, req)
			
			// Transform response
			if respTransformer != nil && res.IsSuccess() {
				transformedResp, err := respTransformer.Transform(res.Value())
				if err != nil {
					return result.Failure[Response](err)
				}
				return result.Success(transformedResp)
			}
			
			return res
		})
	}
}

// RequestTransformer transforms requests
type RequestTransformer interface {
	Transform(req Request) (Request, error)
}

// ResponseTransformer transforms responses
type ResponseTransformer interface {
	Transform(resp Response) (Response, error)
}

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

func (r *BasicRequest) Context() context.Context         { return r.ctx }
func (r *BasicRequest) ID() string                       { return r.id }
func (r *BasicRequest) Method() string                   { return r.method }
func (r *BasicRequest) Path() string                     { return r.path }
func (r *BasicRequest) Headers() map[string][]string     { return r.headers }
func (r *BasicRequest) Body() interface{}                { return r.body }

// BasicResponse is a basic response implementation
type BasicResponse struct {
	status  int
	headers map[string][]string
	body    interface{}
}

func (r *BasicResponse) StatusCode() int              { return r.status }
func (r *BasicResponse) Headers() map[string][]string  { return r.headers }
func (r *BasicResponse) Body() interface{}             { return r.body }

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