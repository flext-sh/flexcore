// Package adapter provides base functionality for all FlexCore adapters
package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/flext/flexcore/shared/result"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// BaseAdapter provides common functionality for all adapters
type BaseAdapter struct {
	name      string
	version   string
	config    interface{}
	validator *validator.Validate
	tracer    trace.Tracer
	hooks     AdapterHooks
}

// AdapterHooks allows customization of adapter lifecycle
type AdapterHooks struct {
	OnBeforeExtract func(context.Context, ExtractRequest) error
	OnAfterExtract  func(context.Context, ExtractRequest, *ExtractResponse, error)
	OnBeforeLoad    func(context.Context, LoadRequest) error
	OnAfterLoad     func(context.Context, LoadRequest, *LoadResponse, error)
	OnError         func(context.Context, error)
}

// AdapterOption is a functional option for configuring adapters
type AdapterOption func(*BaseAdapter)

// WithHooks sets adapter hooks
func WithHooks(hooks AdapterHooks) AdapterOption {
	return func(a *BaseAdapter) {
		a.hooks = hooks
	}
}

// WithValidator sets a custom validator
func WithValidator(v *validator.Validate) AdapterOption {
	return func(a *BaseAdapter) {
		a.validator = v
	}
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(name, version string, opts ...AdapterOption) *BaseAdapter {
	adapter := &BaseAdapter{
		name:      name,
		version:   version,
		validator: validator.New(),
		tracer:    otel.Tracer(name),
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// Name returns the adapter name
func (a *BaseAdapter) Name() string {
	return a.name
}

// Version returns the adapter version
func (a *BaseAdapter) Version() string {
	return a.version
}

// Configure configures the adapter with the provided configuration
func (a *BaseAdapter) Configure(configType interface{}, rawConfig map[string]interface{}) error {
	v := viper.New()
	v.SetConfigType("yaml")

	for k, val := range rawConfig {
		v.Set(k, val)
	}

	if err := v.Unmarshal(configType); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := a.validator.Struct(configType); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	a.config = configType
	return nil
}

// ExtractWithHooks wraps extraction with hooks and tracing
func (a *BaseAdapter) ExtractWithHooks(
	ctx context.Context,
	req ExtractRequest,
	extractFn func(context.Context, ExtractRequest) result.Result[*ExtractResponse],
) (*ExtractResponse, error) {
	ctx, span := a.tracer.Start(ctx, "Extract")
	defer span.End()

	// Before hook
	if a.hooks.OnBeforeExtract != nil {
		if err := a.hooks.OnBeforeExtract(ctx, req); err != nil {
			span.RecordError(err)
			return nil, err
		}
	}

	// Execute extraction
	start := time.Now()
	result := extractFn(ctx, req)
	duration := time.Since(start)

	span.SetAttributes(
		attribute.String("adapter.name", a.name),
		attribute.String("adapter.version", a.version),
		attribute.Int64("duration.ms", duration.Milliseconds()),
	)

	var resp *ExtractResponse
	var err error

	if result.IsSuccess() {
		resp = result.Value()
		span.SetAttributes(
			attribute.Int64("records.count", int64(len(resp.Records))),
		)
	} else {
		err = result.Error()
		span.RecordError(err)
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

// LoadWithHooks wraps loading with hooks and tracing
func (a *BaseAdapter) LoadWithHooks(
	ctx context.Context,
	req LoadRequest,
	loadFn func(context.Context, LoadRequest) result.Result[*LoadResponse],
) (*LoadResponse, error) {
	ctx, span := a.tracer.Start(ctx, "Load")
	defer span.End()

	// Before hook
	if a.hooks.OnBeforeLoad != nil {
		if err := a.hooks.OnBeforeLoad(ctx, req); err != nil {
			span.RecordError(err)
			return nil, err
		}
	}

	// Execute load
	start := time.Now()
	result := loadFn(ctx, req)
	duration := time.Since(start)

	span.SetAttributes(
		attribute.String("adapter.name", a.name),
		attribute.String("adapter.version", a.version),
		attribute.Int64("duration.ms", duration.Milliseconds()),
	)

	var resp *LoadResponse
	var err error

	if result.IsSuccess() {
		resp = result.Value()
		span.SetAttributes(
			attribute.Int64("records.loaded", resp.RecordsLoaded),
			attribute.Int64("records.failed", resp.RecordsFailed),
		)
	} else {
		err = result.Error()
		span.RecordError(err)
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

// Config returns the typed configuration
func (a *BaseAdapter) Config() interface{} {
	return a.config
}