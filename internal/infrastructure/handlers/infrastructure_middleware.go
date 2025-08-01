// Package handlers - Infrastructure-related middleware implementations
// SOLID SRP: Separated infrastructure concerns from general handler chain  
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/flext/flexcore/pkg/result"
	"github.com/flext/flexcore/shared/errors"
)

// RecoveryMiddleware recovers from panics
// SOLID SRP: Single responsibility for panic recovery
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
// SOLID SRP: Single responsibility for request timeout
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

// CompressionMiddleware adds response compression
// SOLID SRP: Single responsibility for response compression
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
// SOLID SRP: Single responsibility for response caching
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
				cache.Set(cacheKey, result.Value(), time.Minute*defaultCacheTTLMinutes)
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
// SOLID SRP: Single responsibility for request/response transformation
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