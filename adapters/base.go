// Package adapters provides base adapter implementation for FlexCore
package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flext/flexcore/infrastructure/config"
	"github.com/flext/flexcore/infrastructure/di"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/patterns"
)

// Adapter represents a FlexCore adapter interface
type Adapter interface {
	// Lifecycle
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Shutdown(ctx context.Context) error
	
	// Health
	HealthCheck(ctx context.Context) HealthStatus
	
	// Info
	Name() string
	Version() string
	Type() AdapterType
}

// AdapterType represents the type of adapter
type AdapterType string

const (
	AdapterTypeSource      AdapterType = "source"
	AdapterTypeTarget      AdapterType = "target"
	AdapterTypeTransformer AdapterType = "transformer"
	AdapterTypeConnector   AdapterType = "connector"
)

// HealthStatus represents adapter health status
type HealthStatus struct {
	Healthy   bool
	Message   string
	Details   map[string]interface{}
	Timestamp time.Time
}

// BaseAdapter provides common functionality for all adapters
type BaseAdapter struct {
	name        string
	version     string
	adapterType AdapterType
	config      *config.Manager
	container   *di.AdvancedContainer
	logger      Logger
	metrics     MetricsCollector
	state       AdapterState
	mu          sync.RWMutex
	
	// Lifecycle hooks
	onInitialize func(context.Context) error
	onStart      func(context.Context) error
	onStop       func(context.Context) error
	onShutdown   func(context.Context) error
	onHealthCheck func(context.Context) HealthStatus
}

// AdapterState represents the current state of an adapter
type AdapterState int

const (
	AdapterStateNew AdapterState = iota
	AdapterStateInitialized
	AdapterStateStarting
	AdapterStateRunning
	AdapterStateStopping
	AdapterStateStopped
	AdapterStateShutdown
	AdapterStateError
)

// String returns string representation of adapter state
func (s AdapterState) String() string {
	switch s {
	case AdapterStateNew:
		return "new"
	case AdapterStateInitialized:
		return "initialized"
	case AdapterStateStarting:
		return "starting"
	case AdapterStateRunning:
		return "running"
	case AdapterStateStopping:
		return "stopping"
	case AdapterStateStopped:
		return "stopped"
	case AdapterStateShutdown:
		return "shutdown"
	case AdapterStateError:
		return "error"
	default:
		return "unknown"
	}
}

// Logger interface for adapter logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, err error, fields ...interface{})
}

// MetricsCollector interface for adapter metrics
type MetricsCollector interface {
	RecordAdapterOperation(adapter, operation string, duration int64, success bool)
	RecordAdapterHealth(adapter string, healthy bool)
	RecordAdapterState(adapter string, state string)
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(name, version string, adapterType AdapterType, options ...patterns.Option[BaseAdapter]) *BaseAdapter {
	adapter := &BaseAdapter{
		name:        name,
		version:     version,
		adapterType: adapterType,
		state:       AdapterStateNew,
	}

	// Apply options
	for _, opt := range options {
		opt(adapter)
	}

	return adapter
}

// WithConfig sets the configuration manager
func WithConfig(config *config.Manager) patterns.Option[BaseAdapter] {
	return func(a *BaseAdapter) error {
		a.config = config
		return nil
	}
}

// WithContainer sets the DI container
func WithContainer(container *di.AdvancedContainer) patterns.Option[BaseAdapter] {
	return func(a *BaseAdapter) error {
		a.container = container
		return nil
	}
}

// WithLogger sets the logger
func WithLogger(logger Logger) patterns.Option[BaseAdapter] {
	return func(a *BaseAdapter) error {
		a.logger = logger
		return nil
	}
}

// WithMetrics sets the metrics collector
func WithMetrics(metrics MetricsCollector) patterns.Option[BaseAdapter] {
	return func(a *BaseAdapter) error {
		a.metrics = metrics
		return nil
	}
}

// WithInitializeHook sets the initialize hook
func WithInitializeHook(hook func(context.Context) error) patterns.Option[BaseAdapter] {
	return func(a *BaseAdapter) error {
		a.onInitialize = hook
		return nil
	}
}

// WithStartHook sets the start hook
func WithStartHook(hook func(context.Context) error) patterns.Option[BaseAdapter] {
	return func(a *BaseAdapter) error {
		a.onStart = hook
		return nil
	}
}

// WithStopHook sets the stop hook
func WithStopHook(hook func(context.Context) error) patterns.Option[BaseAdapter] {
	return func(a *BaseAdapter) error {
		a.onStop = hook
		return nil
	}
}

// WithHealthCheckHook sets the health check hook
func WithHealthCheckHook(hook func(context.Context) HealthStatus) patterns.Option[BaseAdapter] {
	return func(a *BaseAdapter) error {
		a.onHealthCheck = hook
		return nil
	}
}

// Initialize initializes the adapter
func (a *BaseAdapter) Initialize(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != AdapterStateNew {
		return errors.ValidationError(fmt.Sprintf("adapter is in state %s, cannot initialize", a.state))
	}

	start := time.Now()
	a.logInfo("Initializing adapter")
	a.setState(AdapterStateInitialized)

	if a.onInitialize != nil {
		if err := a.onInitialize(ctx); err != nil {
			a.setState(AdapterStateError)
			a.recordMetric("initialize", time.Since(start), false)
			return errors.Wrap(err, "adapter initialization failed")
		}
	}

	a.recordMetric("initialize", time.Since(start), true)
	a.logInfo("Adapter initialized successfully")
	return nil
}

// Start starts the adapter
func (a *BaseAdapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != AdapterStateInitialized && a.state != AdapterStateStopped {
		return errors.ValidationError(fmt.Sprintf("adapter is in state %s, cannot start", a.state))
	}

	start := time.Now()
	a.logInfo("Starting adapter")
	a.setState(AdapterStateStarting)

	if a.onStart != nil {
		if err := a.onStart(ctx); err != nil {
			a.setState(AdapterStateError)
			a.recordMetric("start", time.Since(start), false)
			return errors.Wrap(err, "adapter start failed")
		}
	}

	a.setState(AdapterStateRunning)
	a.recordMetric("start", time.Since(start), true)
	a.logInfo("Adapter started successfully")
	return nil
}

// Stop stops the adapter
func (a *BaseAdapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != AdapterStateRunning {
		return errors.ValidationError(fmt.Sprintf("adapter is in state %s, cannot stop", a.state))
	}

	start := time.Now()
	a.logInfo("Stopping adapter")
	a.setState(AdapterStateStopping)

	if a.onStop != nil {
		if err := a.onStop(ctx); err != nil {
			a.setState(AdapterStateError)
			a.recordMetric("stop", time.Since(start), false)
			return errors.Wrap(err, "adapter stop failed")
		}
	}

	a.setState(AdapterStateStopped)
	a.recordMetric("stop", time.Since(start), true)
	a.logInfo("Adapter stopped successfully")
	return nil
}

// Shutdown shuts down the adapter
func (a *BaseAdapter) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	start := time.Now()
	a.logInfo("Shutting down adapter")

	// Stop if running
	if a.state == AdapterStateRunning {
		a.setState(AdapterStateStopping)
		if a.onStop != nil {
			a.onStop(ctx)
		}
	}

	if a.onShutdown != nil {
		if err := a.onShutdown(ctx); err != nil {
			a.recordMetric("shutdown", time.Since(start), false)
			return errors.Wrap(err, "adapter shutdown failed")
		}
	}

	a.setState(AdapterStateShutdown)
	a.recordMetric("shutdown", time.Since(start), true)
	a.logInfo("Adapter shut down successfully")
	return nil
}

// HealthCheck performs health check
func (a *BaseAdapter) HealthCheck(ctx context.Context) HealthStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.onHealthCheck != nil {
		status := a.onHealthCheck(ctx)
		a.recordHealth(status.Healthy)
		return status
	}

	// Default health check based on state
	healthy := a.state == AdapterStateRunning
	status := HealthStatus{
		Healthy:   healthy,
		Message:   fmt.Sprintf("Adapter is %s", a.state),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"state":   a.state.String(),
			"version": a.version,
		},
	}

	a.recordHealth(healthy)
	return status
}

// Name returns adapter name
func (a *BaseAdapter) Name() string {
	return a.name
}

// Version returns adapter version
func (a *BaseAdapter) Version() string {
	return a.version
}

// Type returns adapter type
func (a *BaseAdapter) Type() AdapterType {
	return a.adapterType
}

// GetState returns current adapter state
func (a *BaseAdapter) GetState() AdapterState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// IsRunning returns true if adapter is running
func (a *BaseAdapter) IsRunning() bool {
	return a.GetState() == AdapterStateRunning
}

// Config returns configuration manager
func (a *BaseAdapter) Config() *config.Manager {
	return a.config
}

// Container returns DI container
func (a *BaseAdapter) Container() *di.AdvancedContainer {
	return a.container
}

// setState sets adapter state
func (a *BaseAdapter) setState(state AdapterState) {
	a.state = state
	if a.metrics != nil {
		a.metrics.RecordAdapterState(a.name, state.String())
	}
}

// logInfo logs info message
func (a *BaseAdapter) logInfo(msg string, fields ...interface{}) {
	if a.logger != nil {
		fields = append(fields, "adapter", a.name, "type", a.adapterType)
		a.logger.Info(msg, fields...)
	}
}

// recordMetric records operation metric
func (a *BaseAdapter) recordMetric(operation string, duration time.Duration, success bool) {
	if a.metrics != nil {
		a.metrics.RecordAdapterOperation(a.name, operation, duration.Milliseconds(), success)
	}
}

// recordHealth records health metric
func (a *BaseAdapter) recordHealth(healthy bool) {
	if a.metrics != nil {
		a.metrics.RecordAdapterHealth(a.name, healthy)
	}
}

// SourceAdapter represents a source adapter (data producer)
type SourceAdapter interface {
	Adapter
	Discover(ctx context.Context) (Schema, error)
	Read(ctx context.Context, state State) (<-chan Record, error)
}

// TargetAdapter represents a target adapter (data consumer)
type TargetAdapter interface {
	Adapter
	Write(ctx context.Context, records <-chan Record) error
}

// TransformerAdapter represents a transformer adapter
type TransformerAdapter interface {
	Adapter
	Transform(ctx context.Context, record Record) (Record, error)
}

// Schema represents data schema
type Schema struct {
	Streams []Stream `json:"streams"`
}

// Stream represents a data stream
type Stream struct {
	Name       string                 `json:"name"`
	JSONSchema map[string]interface{} `json:"json_schema"`
}

// Record represents a data record
type Record struct {
	Type   RecordType             `json:"type"`
	Stream string                 `json:"stream"`
	Data   map[string]interface{} `json:"record,omitempty"`
	State  map[string]interface{} `json:"state,omitempty"`
}

// RecordType represents the type of record
type RecordType string

const (
	RecordTypeRecord RecordType = "RECORD"
	RecordTypeState  RecordType = "STATE"
	RecordTypeSchema RecordType = "SCHEMA"
)

// State represents adapter state for resumption
type State map[string]interface{}

// AdapterBuilder provides fluent interface for building adapters
type AdapterBuilder struct {
	adapter *BaseAdapter
}

// NewAdapterBuilder creates a new adapter builder
func NewAdapterBuilder(name, version string, adapterType AdapterType) *AdapterBuilder {
	return &AdapterBuilder{
		adapter: NewBaseAdapter(name, version, adapterType),
	}
}

// WithConfiguration adds configuration
func (b *AdapterBuilder) WithConfiguration(config *config.Manager) *AdapterBuilder {
	b.adapter.config = config
	return b
}

// WithDependencyInjection adds DI container
func (b *AdapterBuilder) WithDependencyInjection(container *di.AdvancedContainer) *AdapterBuilder {
	b.adapter.container = container
	return b
}

// WithLogging adds logger
func (b *AdapterBuilder) WithLogging(logger Logger) *AdapterBuilder {
	b.adapter.logger = logger
	return b
}

// WithMetricsCollection adds metrics collector
func (b *AdapterBuilder) WithMetricsCollection(metrics MetricsCollector) *AdapterBuilder {
	b.adapter.metrics = metrics
	return b
}

// OnInitialize sets initialize handler
func (b *AdapterBuilder) OnInitialize(handler func(context.Context) error) *AdapterBuilder {
	b.adapter.onInitialize = handler
	return b
}

// OnStart sets start handler
func (b *AdapterBuilder) OnStart(handler func(context.Context) error) *AdapterBuilder {
	b.adapter.onStart = handler
	return b
}

// OnStop sets stop handler
func (b *AdapterBuilder) OnStop(handler func(context.Context) error) *AdapterBuilder {
	b.adapter.onStop = handler
	return b
}

// OnHealthCheck sets health check handler
func (b *AdapterBuilder) OnHealthCheck(handler func(context.Context) HealthStatus) *AdapterBuilder {
	b.adapter.onHealthCheck = handler
	return b
}

// Build builds the adapter
func (b *AdapterBuilder) Build() Adapter {
	return b.adapter
}

// AdapterRegistry manages adapter registration and discovery
type AdapterRegistry struct {
	adapters map[string]Adapter
	mu       sync.RWMutex
}

// NewAdapterRegistry creates a new adapter registry
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]Adapter),
	}
}

// Register registers an adapter
func (r *AdapterRegistry) Register(adapter Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := adapter.Name()
	if _, exists := r.adapters[name]; exists {
		return errors.AlreadyExistsError("adapter " + name)
	}

	r.adapters[name] = adapter
	return nil
}

// Get retrieves an adapter by name
func (r *AdapterRegistry) Get(name string) patterns.Maybe[Adapter] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if adapter, exists := r.adapters[name]; exists {
		return patterns.Some(adapter)
	}
	return patterns.None[Adapter]()
}

// List returns all registered adapters
func (r *AdapterRegistry) List() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapters := make([]Adapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}
	return adapters
}

// ListByType returns adapters of specific type
func (r *AdapterRegistry) ListByType(adapterType AdapterType) []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapters := make([]Adapter, 0)
	for _, adapter := range r.adapters {
		if adapter.Type() == adapterType {
			adapters = append(adapters, adapter)
		}
	}
	return adapters
}