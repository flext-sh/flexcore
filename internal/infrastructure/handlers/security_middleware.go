// Package handlers - Security-related middleware implementations
// SOLID SRP: Separated security concerns from general handler chain
package handlers

import (
	"context"
	"net/http"

	"github.com/flext/flexcore/pkg/result"
	"github.com/flext/flexcore/shared/errors"
)

// AuthenticationMiddleware adds authentication
// SOLID SRP: Single responsibility for authentication
func AuthenticationMiddleware(authenticator Authenticator) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) result.Result[Response] {
			authResult := authenticator.Authenticate(ctx, req)
			if authResult.IsFailure() {
				return result.Failure[Response](authResult.Error())
			}

			// Add authenticated user to context
			user := authResult.Value()
			ctx = context.WithValue(ctx, userContextKey, user)

			return next.Handle(ctx, req)
		})
	}
}

// Authenticator interface
type Authenticator interface {
	Authenticate(ctx context.Context, req Request) result.Result[interface{}]
}

// ValidationMiddleware validates requests
// SOLID SRP: Single responsibility for request validation
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
// SOLID SRP: Single responsibility for rate limiting
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

// CORSMiddleware adds CORS headers
// SOLID SRP: Single responsibility for CORS handling
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