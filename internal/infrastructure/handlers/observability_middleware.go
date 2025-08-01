// Package handlers - Observability-related middleware implementations  
// SOLID SRP: Separated observability concerns from general handler chain
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/flext/flexcore/pkg/result"
)

// LoggingMiddleware logs requests and responses
// SOLID SRP: Single responsibility for request/response logging
func LoggingMiddleware(logger Logger) Middleware {
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

// Logger interface for structured logging
type Logger interface {
	Info(msg string, fields ...interface{})
}

// MetricsMiddleware collects metrics
// SOLID SRP: Single responsibility for metrics collection
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
// SOLID SRP: Single responsibility for distributed tracing
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