// Package commands provides CQRS command handling infrastructure
package commands

import (
	"context"
	"sync"

	"github.com/flext/flexcore/pkg/cqrs"
	"github.com/flext/flexcore/pkg/errors"
	"github.com/flext/flexcore/pkg/result"
)

// Command represents a command in the CQRS pattern
type Command interface {
	CommandType() string
}

// CommandHandler represents a handler for a specific command type
type CommandHandler[T Command] interface {
	Handle(ctx context.Context, command T) result.Result[interface{}]
}

// CommandBus coordinates command execution
type CommandBus interface {
	RegisterHandler(command Command, handler interface{}) error
	Execute(ctx context.Context, command Command) result.Result[interface{}]
	ExecuteAsync(ctx context.Context, command Command) result.Result[chan result.Result[interface{}]]
}

// InMemoryCommandBus provides an in-memory implementation of CommandBus
type InMemoryCommandBus struct {
	mu        sync.RWMutex
	handlers  map[string]interface{}
	validator *cqrs.BusValidationUtilities
	invoker   *cqrs.HandlerInvoker
}

// NewInMemoryCommandBus creates a new in-memory command bus
func NewInMemoryCommandBus() *InMemoryCommandBus {
	return &InMemoryCommandBus{
		handlers:  make(map[string]interface{}),
		validator: cqrs.NewBusValidationUtilities(),
		invoker:   cqrs.NewHandlerInvoker(),
	}
}

// NewCommandBus creates a new command bus (returns interface)
func NewCommandBus() CommandBus {
	return NewInMemoryCommandBus()
}

// RegisterHandler registers a command handler
// DRY PRINCIPLE: Uses shared registration logic eliminating 21-line duplication (mass=117)
func (bus *InMemoryCommandBus) RegisterHandler(command Command, handler interface{}) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	// DRY: Use shared registration logic
	return bus.validator.RegisterHandlerGeneric(command, handler, "command", bus.handlers, "command")
}

// Execute executes a command synchronously
// DRY PRINCIPLE: Uses shared execution logic eliminating 21-line duplication (mass=148)
func (bus *InMemoryCommandBus) Execute(ctx context.Context, command Command) result.Result[interface{}] {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// DRY: Use shared execution logic
	return bus.validator.ExecuteGeneric(ctx, command, "command", bus.handlers, bus.invoker)
}

// ExecuteAsync executes a command asynchronously
func (bus *InMemoryCommandBus) ExecuteAsync(ctx context.Context, command Command) result.Result[chan result.Result[interface{}]] {
	resultChan := make(chan result.Result[interface{}], 1)

	go func() {
		defer close(resultChan)
		result := bus.Execute(ctx, command)
		resultChan <- result
	}()

	return result.Success(resultChan)
}

// isValidHandler has been replaced by shared BusValidationUtilities.IsValidCommandHandler()
// DRY PRINCIPLE: Eliminates 27-line duplication by using shared validation utilities

// invokeHandler invokes a command handler using shared reflection logic
// DRY PRINCIPLE: Uses shared HandlerInvoker to eliminate duplication with query_bus.go
// invokeHandler has been replaced by shared ExecuteGeneric method
// DRY PRINCIPLE: Eliminates duplicate handler invocation by using shared execution logic

// GetRegisteredCommands returns all registered command types
func (bus *InMemoryCommandBus) GetRegisteredCommands() []string {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	commands := make([]string, 0, len(bus.handlers))
	for commandType := range bus.handlers {
		commands = append(commands, commandType)
	}
	return commands
}

// BaseCommand provides a base implementation for commands
type BaseCommand struct {
	commandType string
}

// NewBaseCommand creates a new base command
func NewBaseCommand(commandType string) BaseCommand {
	return BaseCommand{commandType: commandType}
}

// CommandType returns the command type
func (c BaseCommand) CommandType() string {
	return c.commandType
}

// Decorator represents a command decorator
type Decorator interface {
	Decorate(ctx context.Context, command Command, next func(ctx context.Context, command Command) result.Result[interface{}]) result.Result[interface{}]
}

// DecoratedCommandBus wraps a command bus with decorators
type DecoratedCommandBus struct {
	inner      CommandBus
	decorators []Decorator
}

// NewDecoratedCommandBus creates a new decorated command bus
func NewDecoratedCommandBus(inner CommandBus, decorators ...Decorator) *DecoratedCommandBus {
	return &DecoratedCommandBus{
		inner:      inner,
		decorators: decorators,
	}
}

// RegisterHandler registers a command handler
func (bus *DecoratedCommandBus) RegisterHandler(command Command, handler interface{}) error {
	return bus.inner.RegisterHandler(command, handler)
}

// Execute executes a command with decorators applied
func (bus *DecoratedCommandBus) Execute(ctx context.Context, command Command) result.Result[interface{}] {
	next := func(ctx context.Context, cmd Command) result.Result[interface{}] {
		return bus.inner.Execute(ctx, cmd)
	}

	// Apply decorators in reverse order
	for i := len(bus.decorators) - 1; i >= 0; i-- {
		decorator := bus.decorators[i]
		currentNext := next
		next = func(ctx context.Context, cmd Command) result.Result[interface{}] {
			return decorator.Decorate(ctx, cmd, currentNext)
		}
	}

	return next(ctx, command)
}

// ExecuteAsync executes a command asynchronously with decorators applied
func (bus *DecoratedCommandBus) ExecuteAsync(ctx context.Context, command Command) result.Result[chan result.Result[interface{}]] {
	resultChan := make(chan result.Result[interface{}], 1)

	go func() {
		defer close(resultChan)
		result := bus.Execute(ctx, command)
		resultChan <- result
	}()

	return result.Success(resultChan)
}

// LoggingDecorator logs command execution
type LoggingDecorator struct {
	logger Logger
}

// Logger interface for logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, err error, fields ...interface{})
}

// NewLoggingDecorator creates a new logging decorator
func NewLoggingDecorator(logger Logger) *LoggingDecorator {
	return &LoggingDecorator{logger: logger}
}

// Decorate decorates command execution with logging
func (d *LoggingDecorator) Decorate(ctx context.Context, command Command, next func(ctx context.Context, command Command) result.Result[interface{}]) result.Result[interface{}] {
	d.logger.Info("Executing command", "type", command.CommandType())

	result := next(ctx, command)

	if result.IsSuccess() {
		d.logger.Info("Command executed successfully", "type", command.CommandType())
	} else {
		d.logger.Error("Command execution failed", result.Error(), "type", command.CommandType())
	}

	return result
}

// ValidationDecorator validates commands before execution
type ValidationDecorator struct {
	validators map[string]CommandValidator
}

// CommandValidator validates commands
type CommandValidator interface {
	Validate(command Command) error
}

// NewValidationDecorator creates a new validation decorator
func NewValidationDecorator() *ValidationDecorator {
	return &ValidationDecorator{
		validators: make(map[string]CommandValidator),
	}
}

// RegisterValidator registers a validator for a command type
func (d *ValidationDecorator) RegisterValidator(commandType string, validator CommandValidator) {
	d.validators[commandType] = validator
}

// Decorate decorates command execution with validation
func (d *ValidationDecorator) Decorate(ctx context.Context, command Command, next func(ctx context.Context, command Command) result.Result[interface{}]) result.Result[interface{}] {
	if validator, exists := d.validators[command.CommandType()]; exists {
		if err := validator.Validate(command); err != nil {
			return result.Failure[interface{}](errors.Wrap(err, "command validation failed"))
		}
	}

	return next(ctx, command)
}

// MetricsDecorator collects metrics for command execution
type MetricsDecorator struct {
	metricsCollector MetricsCollector
}

// MetricsCollector collects execution metrics
type MetricsCollector interface {
	RecordCommandExecution(commandType string, duration int64, success bool)
}

// NewMetricsDecorator creates a new metrics decorator
func NewMetricsDecorator(collector MetricsCollector) *MetricsDecorator {
	return &MetricsDecorator{metricsCollector: collector}
}

// Decorate decorates command execution with metrics collection
func (d *MetricsDecorator) Decorate(ctx context.Context, command Command, next func(ctx context.Context, command Command) result.Result[interface{}]) result.Result[interface{}] {
	// start := result.Try(func() int64 { return 0 }).Value() // Simplified for example

	result := next(ctx, command)

	duration := int64(0) // Calculate actual duration
	d.metricsCollector.RecordCommandExecution(command.CommandType(), duration, result.IsSuccess())

	return result
}

// CommandBusBuilder helps build command bus configurations
type CommandBusBuilder struct {
	decorators []Decorator
}

// NewCommandBusBuilder creates a new command bus builder
func NewCommandBusBuilder() *CommandBusBuilder {
	return &CommandBusBuilder{
		decorators: make([]Decorator, 0),
	}
}

// WithLogging adds logging decorator
func (b *CommandBusBuilder) WithLogging(logger Logger) *CommandBusBuilder {
	b.decorators = append(b.decorators, NewLoggingDecorator(logger))
	return b
}

// WithValidation adds validation decorator
func (b *CommandBusBuilder) WithValidation() *CommandBusBuilder {
	b.decorators = append(b.decorators, NewValidationDecorator())
	return b
}

// WithMetrics adds metrics decorator
func (b *CommandBusBuilder) WithMetrics(collector MetricsCollector) *CommandBusBuilder {
	b.decorators = append(b.decorators, NewMetricsDecorator(collector))
	return b
}

// WithDecorator adds a custom decorator
func (b *CommandBusBuilder) WithDecorator(decorator Decorator) *CommandBusBuilder {
	b.decorators = append(b.decorators, decorator)
	return b
}

// Build creates the command bus
func (b *CommandBusBuilder) Build() CommandBus {
	inner := NewInMemoryCommandBus()

	if len(b.decorators) == 0 {
		return inner
	}

	return NewDecoratedCommandBus(inner, b.decorators...)
}
