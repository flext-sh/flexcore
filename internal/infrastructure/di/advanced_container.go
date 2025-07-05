// Package di provides advanced dependency injection features
package di

import (
	"context"
	"reflect"
	"sync"

	"github.com/flext/flexcore/pkg/errors"
	"github.com/flext/flexcore/pkg/patterns"
	"github.com/flext/flexcore/pkg/result"
)

// AdvancedContainer provides advanced DI features similar to Python's lato and dependency-injector
type AdvancedContainer struct {
	*Container
	providers   map[string]Provider[any]
	decorators  map[string][]Decorator[any]
	interceptors []Interceptor
	scopes      map[string]*Scope
	mu          sync.RWMutex
}

// Provider represents a service provider
type Provider[T any] interface {
	Get(ctx context.Context) result.Result[T]
	GetAsync(ctx context.Context) <-chan result.Result[T]
	Lifecycle() Lifetime
}

// Decorator decorates a service
type Decorator[T any] func(T) T

// Interceptor intercepts service resolution
type Interceptor func(ctx context.Context, serviceName string, next func() result.Result[any]) result.Result[any]

// Scope represents a dependency scope
type Scope struct {
	name      string
	parent    *Scope
	instances map[string]any
	mu        sync.RWMutex
}

// NewAdvancedContainer creates a new advanced DI container
func NewAdvancedContainer() *AdvancedContainer {
	return &AdvancedContainer{
		Container:    NewContainer(),
		providers:    make(map[string]Provider[any]),
		decorators:   make(map[string][]Decorator[any]),
		interceptors: make([]Interceptor, 0),
		scopes:       make(map[string]*Scope),
	}
}

// Provide registers a service using automatic resolution (lato-style)
func (c *AdvancedContainer) Provide(name string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	provider := NewValueProvider(value)
	c.providers[name] = provider
	return nil
}

// Call executes a function with automatic dependency injection (lato-style)
func (c *AdvancedContainer) Call(ctx context.Context, fn any, args ...any) result.Result[any] {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return result.Failure[any](errors.ValidationError("argument must be a function"))
	}

	// Prepare arguments
	callArgs := make([]reflect.Value, fnType.NumIn())
	providedArgIndex := 0

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		paramTypeName := paramType.String()

		// Check if it's context.Context
		if paramType.String() == "context.Context" {
			callArgs[i] = reflect.ValueOf(ctx)
			continue
		}

		// Try to resolve from container first (except for the last N params where N = len(args))
		argsStartIndex := fnType.NumIn() - len(args)
		if i >= argsStartIndex && providedArgIndex < len(args) {
			// This parameter should come from explicit args
			callArgs[i] = reflect.ValueOf(args[providedArgIndex])
			providedArgIndex++
			continue
		}

		// Try to resolve from container by type name with interceptors
		if resolved := c.resolveWithInterceptors(ctx, paramTypeName); resolved != nil {
			callArgs[i] = reflect.ValueOf(resolved)
		} else if resolved := c.resolveByType(ctx, paramType); resolved != nil {
			callArgs[i] = reflect.ValueOf(resolved)
		} else {
			// Use zero value
			callArgs[i] = reflect.Zero(paramType)
		}
	}

	// Call the function
	results := fnValue.Call(callArgs)

	// Handle results
	if len(results) == 0 {
		return result.Success[any](nil)
	}

	// Check for error in last result
	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if err, ok := lastResult.Interface().(error); ok && err != nil {
			return result.Failure[any](err)
		}
	}

	return result.Success[any](results[0].Interface())
}

// Execute executes a command/handler with DI (lato-style)
func (c *AdvancedContainer) Execute(ctx context.Context, handler any) result.Result[any] {
	return c.Call(ctx, handler)
}

// Factory creates a factory provider
func Factory[T any](factory func(context.Context) (T, error)) Provider[T] {
	return &FactoryProvider[T]{
		factory:  factory,
		lifetime: Transient,
	}
}

// Singleton creates a singleton provider
func Singleton[T any](factory func(context.Context) (T, error)) Provider[T] {
	return &SingletonProvider[T]{
		FactoryProvider: FactoryProvider[T]{
			factory:  factory,
			lifetime: SingletonLifetime,
		},
	}
}

// Value creates a value provider
func Value[T any](value T) Provider[T] {
	return &ValueProvider[T]{
		value:    value,
		lifetime: SingletonLifetime,
	}
}

// Resource creates a resource provider with lifecycle
func Resource[T any](
	factory func(context.Context) (T, error),
	cleanup func(T) error,
) Provider[T] {
	return &ResourceProvider[T]{
		factory:  factory,
		cleanup:  cleanup,
		lifetime: SingletonLifetime,
	}
}

// FactoryProvider provides transient instances
type FactoryProvider[T any] struct {
	factory  func(context.Context) (T, error)
	lifetime Lifetime
}

func (p *FactoryProvider[T]) Get(ctx context.Context) result.Result[T] {
	value, err := p.factory(ctx)
	if err != nil {
		return result.Failure[T](err)
	}
	return result.Success(value)
}

func (p *FactoryProvider[T]) GetAsync(ctx context.Context) <-chan result.Result[T] {
	ch := make(chan result.Result[T], 1)
	go func() {
		defer close(ch)
		ch <- p.Get(ctx)
	}()
	return ch
}

func (p *FactoryProvider[T]) Lifecycle() Lifetime {
	return p.lifetime
}

// SingletonProvider provides singleton instances
type SingletonProvider[T any] struct {
	FactoryProvider[T]
	instance patterns.Maybe[T]
	once     sync.Once
	mu       sync.RWMutex
}

func (p *SingletonProvider[T]) Get(ctx context.Context) result.Result[T] {
	p.mu.RLock()
	if p.instance.IsPresent() {
		p.mu.RUnlock()
		return result.Success(p.instance.OrElse(*new(T)))
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if p.instance.IsPresent() {
		return result.Success(p.instance.OrElse(*new(T)))
	}

	value, err := p.factory(ctx)
	if err != nil {
		return result.Failure[T](err)
	}

	p.instance = patterns.Some(value)
	return result.Success(value)
}

// ValueProvider provides a constant value
type ValueProvider[T any] struct {
	value    T
	lifetime Lifetime
}

func NewValueProvider[T any](value T) Provider[T] {
	return &ValueProvider[T]{
		value:    value,
		lifetime: SingletonLifetime,
	}
}

func (p *ValueProvider[T]) Get(ctx context.Context) result.Result[T] {
	return result.Success(p.value)
}

func (p *ValueProvider[T]) GetAsync(ctx context.Context) <-chan result.Result[T] {
	ch := make(chan result.Result[T], 1)
	ch <- result.Success(p.value)
	close(ch)
	return ch
}

func (p *ValueProvider[T]) Lifecycle() Lifetime {
	return p.lifetime
}

// ResourceProvider provides resources with lifecycle management
type ResourceProvider[T any] struct {
	factory  func(context.Context) (T, error)
	cleanup  func(T) error
	instance patterns.Maybe[T]
	lifetime Lifetime
	mu       sync.RWMutex
}

func (p *ResourceProvider[T]) Get(ctx context.Context) result.Result[T] {
	p.mu.RLock()
	if p.instance.IsPresent() {
		p.mu.RUnlock()
		return result.Success(p.instance.OrElse(*new(T)))
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	value, err := p.factory(ctx)
	if err != nil {
		return result.Failure[T](err)
	}

	p.instance = patterns.Some(value)
	return result.Success(value)
}

func (p *ResourceProvider[T]) GetAsync(ctx context.Context) <-chan result.Result[T] {
	ch := make(chan result.Result[T], 1)
	go func() {
		defer close(ch)
		ch <- p.Get(ctx)
	}()
	return ch
}

func (p *ResourceProvider[T]) Lifecycle() Lifetime {
	return SingletonLifetime
}

func (p *ResourceProvider[T]) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.instance.IsPresent() && p.cleanup != nil {
		value, _ := p.instance.Get()
		if err := p.cleanup(value); err != nil {
			return err
		}
		p.instance = patterns.None[T]()
	}
	return nil
}

// RegisterProvider registers a provider
func (c *AdvancedContainer) RegisterProvider(name string, provider Provider[any]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providers[name] = provider
}

// RegisterDecorator registers a decorator for a service
func (c *AdvancedContainer) RegisterDecorator(serviceName string, decorator Decorator[any]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.decorators[serviceName]; !exists {
		c.decorators[serviceName] = make([]Decorator[any], 0)
	}
	c.decorators[serviceName] = append(c.decorators[serviceName], decorator)
}

// AddInterceptor adds a global interceptor
func (c *AdvancedContainer) AddInterceptor(interceptor Interceptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, interceptor)
}

// ResolveProvider resolves a service using providers
func ResolveProvider[T any](c *AdvancedContainer, ctx context.Context, name string) result.Result[T] {
	c.mu.RLock()
	provider, exists := c.providers[name]
	c.mu.RUnlock()

	if !exists {
		return result.Failure[T](errors.NotFoundError("provider " + name))
	}

	// Apply interceptors
	var serviceResult result.Result[any]
	next := func() result.Result[any] {
		return provider.Get(ctx)
	}

	// Apply interceptors in reverse order
	for i := len(c.interceptors) - 1; i >= 0; i-- {
		interceptor := c.interceptors[i]
		currentNext := next
		next = func() result.Result[any] {
			return interceptor(ctx, name, currentNext)
		}
	}

	serviceResult = next()

	if serviceResult.IsFailure() {
		return result.Failure[T](serviceResult.Error())
	}

	value := serviceResult.Value()
	
	// Apply decorators
	c.mu.RLock()
	decorators, hasDecorators := c.decorators[name]
	c.mu.RUnlock()

	if hasDecorators {
		for _, decorator := range decorators {
			value = decorator(value)
		}
	}

	typed, ok := value.(T)
	if !ok {
		return result.Failure[T](errors.ValidationError("type mismatch"))
	}

	return result.Success(typed)
}

// CreateScope creates a new scope
func (c *AdvancedContainer) CreateScope(name string) *Scope {
	c.mu.Lock()
	defer c.mu.Unlock()

	scope := &Scope{
		name:      name,
		instances: make(map[string]any),
	}
	c.scopes[name] = scope
	return scope
}

// GetScope retrieves a scope
func (c *AdvancedContainer) GetScope(name string) patterns.Maybe[*Scope] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if scope, exists := c.scopes[name]; exists {
		return patterns.Some(scope)
	}
	return patterns.None[*Scope]()
}

// resolveWithInterceptors resolves a dependency by name with interceptor support
func (c *AdvancedContainer) resolveWithInterceptors(ctx context.Context, name string) any {
	c.mu.RLock()
	interceptors := make([]Interceptor, len(c.interceptors))
	copy(interceptors, c.interceptors)
	c.mu.RUnlock()

	// Get both the resolved value and the actual service name for interceptors
	resolved, serviceName := c.resolveByNameWithServiceName(ctx, name)
	if resolved == nil {
		return nil
	}

	// If no interceptors, return the resolved value directly
	if len(interceptors) == 0 {
		return resolved
	}

	// Create the next function that returns the already resolved value
	next := func() result.Result[any] {
		return result.Success[any](resolved)
	}

	// Apply interceptors in reverse order (like middleware)
	for i := len(interceptors) - 1; i >= 0; i-- {
		currentNext := next
		interceptor := interceptors[i]
		currentServiceName := serviceName // capture for closure
		next = func() result.Result[any] {
			return interceptor(ctx, currentServiceName, currentNext)
		}
	}

	// Execute the interceptor chain
	result := next()
	if result.IsSuccess() {
		return result.Value()
	}
	return nil
}

// resolveByType attempts to resolve a dependency by type
func (c *AdvancedContainer) resolveByType(ctx context.Context, paramType reflect.Type) any {
	// This is a simplified version - in production, implement type matching
	return nil
}

// resolveByName attempts to resolve a dependency by name
func (c *AdvancedContainer) resolveByName(ctx context.Context, name string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// First try exact name match
	if provider, exists := c.providers[name]; exists {
		result := provider.Get(ctx)
		if result.IsSuccess() {
			return result.Value()
		}
	}

	// Then try to resolve by type name for common types
	switch name {
	case "string":
		// Look for any string provider - include all registered string services
		// Return the first matching string provider we find
		for _, provider := range c.providers {
			if r := provider.Get(ctx); r.IsSuccess() {
				if _, ok := r.Value().(string); ok {
					return r.Value()
				}
			}
		}
	case "int":
		// Look for any int provider
		for _, provider := range c.providers {
			if r := provider.Get(ctx); r.IsSuccess() {
				if _, ok := r.Value().(int); ok {
					return r.Value()
				}
			}
		}
	}
	
	return nil
}

// resolveByNameWithServiceName attempts to resolve with proper service name for interceptors
func (c *AdvancedContainer) resolveByNameWithServiceName(ctx context.Context, typeName string) (any, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// First try exact name match
	if provider, exists := c.providers[typeName]; exists {
		result := provider.Get(ctx)
		if result.IsSuccess() {
			return result.Value(), typeName
		}
	}

	// Then try to resolve by type name and return the actual service name
	switch typeName {
	case "string":
		// Look for any string provider and return both value and service name
		for serviceName, provider := range c.providers {
			if r := provider.Get(ctx); r.IsSuccess() {
				if _, ok := r.Value().(string); ok {
					return r.Value(), serviceName
				}
			}
		}
	case "int":
		// Look for any int provider and return both value and service name
		for serviceName, provider := range c.providers {
			if r := provider.Get(ctx); r.IsSuccess() {
				if _, ok := r.Value().(int); ok {
					return r.Value(), serviceName
				}
			}
		}
	}
	
	return nil, ""
}

// Configuration provider
type ConfigurationProvider struct {
	sources map[string]any
	mu      sync.RWMutex
}

func NewConfigurationProvider() *ConfigurationProvider {
	return &ConfigurationProvider{
		sources: make(map[string]any),
	}
}

func (p *ConfigurationProvider) FromEnv(key string) Provider[string] {
	return Factory[string](func(ctx context.Context) (string, error) {
		// In real implementation, read from environment
		return "", nil
	})
}

func (p *ConfigurationProvider) FromFile(path string) Provider[map[string]any] {
	return Factory[map[string]any](func(ctx context.Context) (map[string]any, error) {
		// In real implementation, read from file
		return make(map[string]any), nil
	})
}

// Builder for fluent container configuration
type ContainerBuilder struct {
	container *AdvancedContainer
}

func NewContainerBuilder() *ContainerBuilder {
	return &ContainerBuilder{
		container: NewAdvancedContainer(),
	}
}

func (b *ContainerBuilder) WithProvider(name string, provider Provider[any]) *ContainerBuilder {
	b.container.RegisterProvider(name, provider)
	return b
}

func (b *ContainerBuilder) WithSingleton(name string, factory func(context.Context) (any, error)) *ContainerBuilder {
	b.container.RegisterProvider(name, Singleton[any](factory))
	return b
}

func (b *ContainerBuilder) WithFactory(name string, factory func(context.Context) (any, error)) *ContainerBuilder {
	b.container.RegisterProvider(name, Factory[any](factory))
	return b
}

func (b *ContainerBuilder) WithValue(name string, value any) *ContainerBuilder {
	b.container.RegisterProvider(name, Value[any](value))
	return b
}

func (b *ContainerBuilder) WithInterceptor(interceptor Interceptor) *ContainerBuilder {
	b.container.AddInterceptor(interceptor)
	return b
}

func (b *ContainerBuilder) Build() *AdvancedContainer {
	return b.container
}

// Common interceptors

// LoggingInterceptor logs service resolution
func LoggingInterceptor(logger interface{ Info(string, ...any) }) Interceptor {
	return func(ctx context.Context, serviceName string, next func() result.Result[any]) result.Result[any] {
		logger.Info("Resolving service", "name", serviceName)
		result := next()
		if result.IsSuccess() {
			logger.Info("Service resolved successfully", "name", serviceName)
		} else {
			logger.Info("Service resolution failed", "name", serviceName, "error", result.Error())
		}
		return result
	}
}

// MetricsInterceptor collects metrics
func MetricsInterceptor(collector interface{ RecordServiceResolution(string, bool) }) Interceptor {
	return func(ctx context.Context, serviceName string, next func() result.Result[any]) result.Result[any] {
		result := next()
		collector.RecordServiceResolution(serviceName, result.IsSuccess())
		return result
	}
}

// CachingInterceptor caches service instances
func CachingInterceptor(cache map[string]any) Interceptor {
	var mu sync.RWMutex
	return func(ctx context.Context, serviceName string, next func() result.Result[any]) result.Result[any] {
		mu.RLock()
		if cached, exists := cache[serviceName]; exists {
			mu.RUnlock()
			return result.Success[any](cached)
		}
		mu.RUnlock()

		result := next()
		if result.IsSuccess() {
			mu.Lock()
			cache[serviceName] = result.Value()
			mu.Unlock()
		}
		return result
	}
}