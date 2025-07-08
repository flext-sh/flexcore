// Package commands provides CQRS command handling infrastructure
package commands

import (
	"context"
	"reflect"
	"sync"

	"github.com/flext/flexcore/pkg/result"
	"github.com/flext/flexcore/shared/errors"
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
	mu       sync.RWMutex
	handlers map[string]interface{}
}

// NewInMemoryCommandBus creates a new in-memory command bus
func NewInMemoryCommandBus() *InMemoryCommandBus {
	return &InMemoryCommandBus{
		handlers: make(map[string]interface{}),
	}
}

// NewCommandBus creates a new command bus (returns interface)
func NewCommandBus() CommandBus {
	return NewInMemoryCommandBus()
}

// RegisterHandler registers a command handler
func (bus *InMemoryCommandBus) RegisterHandler(command Command, handler interface{}) error {
	if command == nil {
		return errors.ValidationError("command cannot be nil")
	}

	if handler == nil {
		return errors.ValidationError("handler cannot be nil")
	}

	// Validate handler implements CommandHandler interface
	handlerType := reflect.TypeOf(handler)
	if !bus.isValidHandler(handlerType) {
		return errors.ValidationError("handler must implement CommandHandler interface")
	}

	commandType := command.CommandType()

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if _, exists := bus.handlers[commandType]; exists {
		return errors.AlreadyExistsError("handler for command type " + commandType)
	}

	bus.handlers[commandType] = handler
	return nil
}

// Execute executes a command synchronously
func (bus *InMemoryCommandBus) Execute(ctx context.Context, command Command) result.Result[interface{}] {
	if command == nil {
		return result.Failure[interface{}](errors.ValidationError("command cannot be nil"))
	}

	bus.mu.RLock()
	handler, exists := bus.handlers[command.CommandType()]
	bus.mu.RUnlock()

	if !exists {
		return result.Failure[interface{}](errors.NotFoundError("handler for command type " + command.CommandType()))
	}

	return bus.invokeHandler(ctx, handler, command)
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

// isValidHandler checks if a type implements the CommandHandler interface
func (bus *InMemoryCommandBus) isValidHandler(handlerType reflect.Type) bool {
	// Check if it has a Handle method with correct signature
	method, exists := handlerType.MethodByName("Handle")
	if !exists {
		return false
	}

	// Check method signature: Handle(ctx context.Context, command T) result.Result[interface{}]
	methodType := method.Type
	if methodType.NumIn() != 3 || methodType.NumOut() != 1 {
		return false
	}

	// Check context parameter
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !methodType.In(1).Implements(contextType) {
		return false
	}

	// Check command parameter implements Command interface
	commandType := reflect.TypeOf((*Command)(nil)).Elem()
	if !methodType.In(2).Implements(commandType) {
		return false
	}

	return true
}

// invokeHandler invokes a command handler using reflection
func (bus *InMemoryCommandBus) invokeHandler(ctx context.Context, handler interface{}, command Command) result.Result[interface{}] {
	handlerValue := reflect.ValueOf(handler)
	handlerType := reflect.TypeOf(handler)

	method, exists := handlerType.MethodByName("Handle")
	if !exists {
		return result.Failure[interface{}](errors.InternalError("handler does not have Handle method"))
	}

	// Prepare arguments
	args := []reflect.Value{
		handlerValue,
		reflect.ValueOf(ctx),
		reflect.ValueOf(command),
	}

	// Call the handler method
	results := method.Func.Call(args)
	if len(results) != 1 {
		return result.Failure[interface{}](errors.InternalError("handler returned unexpected number of values"))
	}

	// Extract the result
	resultValue := results[0].Interface()
	if cmdResult, ok := resultValue.(result.Result[interface{}]); ok {
		return cmdResult
	}

	return result.Failure[interface{}](errors.InternalError("handler returned unexpected result type"))
}

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
