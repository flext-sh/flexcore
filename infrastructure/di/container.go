// Package di provides dependency injection capabilities for FlexCore
package di

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
)

// Container represents a dependency injection container
type Container struct {
	mu        sync.RWMutex
	services  map[string]*serviceDescriptor
	instances map[string]interface{}
}

// serviceDescriptor describes how to create a service
type serviceDescriptor struct {
	serviceType reflect.Type
	factory     interface{}
	lifetime    Lifetime
	singleton   bool
}

// Lifetime represents the lifetime of a service
type Lifetime int

const (
	// Transient creates a new instance every time
	Transient Lifetime = iota
	// SingletonLifetime creates only one instance
	SingletonLifetime
	// Scoped creates one instance per scope (not implemented yet)
	Scoped
)

// NewContainer creates a new DI container
func NewContainer() *Container {
	return &Container{
		services:  make(map[string]*serviceDescriptor),
		instances: make(map[string]interface{}),
	}
}

// Register registers a service with transient lifetime
func (c *Container) Register(factory interface{}) {
	c.RegisterWithLifetime(factory, Transient)
}

// RegisterSingleton registers a service with singleton lifetime
func (c *Container) RegisterSingleton(factory interface{}) {
	c.RegisterWithLifetime(factory, SingletonLifetime)
}

// RegisterWithLifetime registers a service with the specified lifetime
func (c *Container) RegisterWithLifetime(factory interface{}, lifetime Lifetime) {
	factoryType := reflect.TypeOf(factory)
	if factoryType.Kind() != reflect.Func {
		panic("factory must be a function")
	}

	if factoryType.NumOut() == 0 {
		panic("factory must return at least one value")
	}

	serviceType := factoryType.Out(0)
	serviceName := getServiceName(serviceType)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[serviceName] = &serviceDescriptor{
		serviceType: serviceType,
		factory:     factory,
		lifetime:    lifetime,
		singleton:   lifetime == SingletonLifetime,
	}
}

// RegisterInstance registers a specific instance as a singleton
func (c *Container) RegisterInstance(instance interface{}) {
	instanceType := reflect.TypeOf(instance)
	serviceName := getServiceName(instanceType)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[serviceName] = &serviceDescriptor{
		serviceType: instanceType,
		factory:     nil,
		lifetime:    SingletonLifetime,
		singleton:   true,
	}
	c.instances[serviceName] = instance
}

// Resolve resolves a service by type
func Resolve[T any](c *Container) result.Result[T] {
	var zero T
	serviceType := reflect.TypeOf((*T)(nil)).Elem()
	serviceName := getServiceName(serviceType)

	instance, err := c.resolve(serviceName)
	if err != nil {
		return result.Failure[T](err)
	}

	typedInstance, ok := instance.(T)
	if !ok {
		return result.Failure[T](errors.Newf("service %s cannot be cast to type %T", serviceName, zero))
	}

	return result.Success(typedInstance)
}

// ResolveByName resolves a service by name
func (c *Container) ResolveByName(serviceName string) result.Result[interface{}] {
	instance, err := c.resolve(serviceName)
	if err != nil {
		return result.Failure[interface{}](err)
	}
	return result.Success(instance)
}

// MustResolve resolves a service and panics if it fails
func MustResolve[T any](c *Container) T {
	return Resolve[T](c).UnwrapOrPanic()
}

// resolve internal resolution logic
func (c *Container) resolve(serviceName string) (interface{}, error) {
	c.mu.RLock()
	descriptor, exists := c.services[serviceName]
	if !exists {
		c.mu.RUnlock()
		return nil, errors.NotFoundError(fmt.Sprintf("service %s", serviceName))
	}

	// Check if singleton instance already exists
	if descriptor.singleton {
		if instance, exists := c.instances[serviceName]; exists {
			c.mu.RUnlock()
			return instance, nil
		}
	}
	c.mu.RUnlock()

	// Create new instance
	instance, err := c.createInstance(descriptor)
	if err != nil {
		return nil, err
	}

	// Store singleton instance
	if descriptor.singleton {
		c.mu.Lock()
		c.instances[serviceName] = instance
		c.mu.Unlock()
	}

	return instance, nil
}

// createInstance creates a new service instance
func (c *Container) createInstance(descriptor *serviceDescriptor) (interface{}, error) {
	if descriptor.factory == nil {
		return nil, errors.InternalError("factory is nil")
	}

	factoryValue := reflect.ValueOf(descriptor.factory)
	factoryType := factoryValue.Type()

	// Resolve dependencies
	args := make([]reflect.Value, factoryType.NumIn())
	for i := 0; i < factoryType.NumIn(); i++ {
		paramType := factoryType.In(i)
		paramName := getServiceName(paramType)

		dependency, err := c.resolve(paramName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve dependency %s", paramName)
		}

		args[i] = reflect.ValueOf(dependency)
	}

	// Call factory function
	results := factoryValue.Call(args)
	if len(results) == 0 {
		return nil, errors.InternalError("factory returned no values")
	}

	instance := results[0].Interface()

	// Check for error return value
	if len(results) > 1 {
		if err, ok := results[1].Interface().(error); ok && err != nil {
			return nil, errors.Wrap(err, "factory returned error")
		}
	}

	return instance, nil
}

// getServiceName generates a service name from a type
func getServiceName(serviceType reflect.Type) string {
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
	}

	if serviceType.Kind() == reflect.Interface {
		return serviceType.Name()
	}

	return fmt.Sprintf("%s.%s", serviceType.PkgPath(), serviceType.Name())
}

// IsRegistered checks if a service is registered
func (c *Container) IsRegistered(serviceName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.services[serviceName]
	return exists
}

// GetRegisteredServices returns all registered service names
func (c *Container) GetRegisteredServices() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	services := make([]string, 0, len(c.services))
	for name := range c.services {
		services = append(services, name)
	}
	return services
}

// Clear removes all services and instances
func (c *Container) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services = make(map[string]*serviceDescriptor)
	c.instances = make(map[string]interface{})
}

// Scope creates a new scoped container (child container)
func (c *Container) Scope() *Container {
	c.mu.RLock()
	defer c.mu.RUnlock()

	child := NewContainer()

	// Copy service descriptors (but not instances)
	for name, descriptor := range c.services {
		child.services[name] = descriptor
	}

	return child
}
