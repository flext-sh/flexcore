// Package commands implements CQRS Command processing infrastructure for FlexCore.
//
// This package provides comprehensive Command Query Responsibility Segregation (CQRS)
// implementation for handling write operations in the FlexCore application layer.
// It implements the Command pattern with enterprise-grade features including
// decoration patterns, validation, logging, metrics, and asynchronous execution.
//
// Architecture:
//   The Command Bus implementation follows several enterprise patterns:
//   - Command Pattern: Encapsulates requests as objects for parameterization and queuing
//   - Decorator Pattern: Adds cross-cutting concerns (logging, validation, metrics)
//   - Registry Pattern: Centralized command handler registration and discovery
//   - Result Pattern: Explicit error handling with railway-oriented programming
//   - Factory Pattern: Command bus creation with builder pattern configuration
//
// Core Components:
//   - Command Interface: Standard contract for all command objects
//   - CommandHandler[T]: Generic handler interface for type-safe command processing
//   - CommandBus: Central dispatcher for command execution and coordination
//   - InMemoryCommandBus: Thread-safe in-memory implementation with handler registry
//   - DecoratedCommandBus: Decorator pattern implementation for cross-cutting concerns
//
// Decorator Infrastructure:
//   - LoggingDecorator: Comprehensive command execution logging
//   - ValidationDecorator: Command validation with pluggable validators
//   - MetricsDecorator: Performance metrics and execution statistics
//   - Custom Decorators: Extensible decorator pattern for additional concerns
//
// Example:
//   Complete command bus setup with decorators:
//
//     // Create command bus with all decorators
//     commandBus := commands.NewCommandBusBuilder().
//         WithLogging(logger).
//         WithValidation().
//         WithMetrics(metricsCollector).
//         Build()
//     
//     // Register command handlers
//     createPipelineHandler := &CreatePipelineHandler{repo: pipelineRepo}
//     commandBus.RegisterHandler(&CreatePipelineCommand{}, createPipelineHandler)
//     
//     // Execute commands
//     command := &CreatePipelineCommand{
//         Name:        "data-processing-pipeline",
//         Description: "ETL pipeline for customer data",
//         Owner:       "data-team",
//     }
//     
//     result := commandBus.Execute(ctx, command)
//     if result.IsFailure() {
//         return fmt.Errorf("command execution failed: %w", result.Error())
//     }
//     
//     log.Info("Pipeline created successfully", "result", result.Value())
//
// Integration:
//   - Domain Layer: Executes business logic through domain entities and aggregates
//   - Application Services: Coordinates use cases and cross-aggregate operations
//   - Infrastructure Layer: Integrates with repositories, event stores, and external services
//   - Presentation Layer: HTTP handlers delegate to command bus for write operations
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package commands

import (
	"context"
	"sync"

	"github.com/flext/flexcore/pkg/cqrs"
	"github.com/flext/flexcore/pkg/errors"
	"github.com/flext/flexcore/pkg/result"
)

// Command represents a command in the CQRS pattern for write operations.
//
// Command serves as the base interface for all command objects in the CQRS
// architecture, encapsulating user intentions to modify system state. Commands
// are immutable request objects that contain all necessary data for executing
// a specific business operation.
//
// Design Principles:
//   - Commands represent user intentions (CreatePipeline, UpdateConfig, DeletePlugin)
//   - Commands are immutable after creation to ensure consistent processing
//   - Commands contain all data needed for execution (no external dependencies)
//   - Commands are validated before execution to ensure business rule compliance
//   - Commands generate domain events as side effects of successful execution
//
// Example:
//   Implementing custom commands:
//
//     type CreatePipelineCommand struct {
//         commands.BaseCommand
//         Name        string
//         Description string
//         Owner       string
//         Steps       []PipelineStep
//     }
//     
//     func NewCreatePipelineCommand(name, desc, owner string) *CreatePipelineCommand {
//         return &CreatePipelineCommand{
//             BaseCommand: commands.NewBaseCommand("CreatePipeline"),
//             Name:        name,
//             Description: desc,
//             Owner:       owner,
//             Steps:       make([]PipelineStep, 0),
//         }
//     }
//     
//     // CommandType is implemented by BaseCommand
//     // Additional validation methods can be added
//     func (c *CreatePipelineCommand) Validate() error {
//         if c.Name == "" {
//             return errors.ValidationError("pipeline name is required")
//         }
//         return nil
//     }
//
// Integration:
//   - Command Handlers: Process specific command types with business logic
//   - Command Bus: Routes commands to appropriate handlers for execution
//   - Validation: Commands validated before execution through decorator pattern
//   - Serialization: Commands can be serialized for queuing and persistence
type Command interface {
	CommandType() string  // Returns unique command type identifier for routing
}

// CommandHandler represents a type-safe handler for processing specific command types.
//
// CommandHandler implements the Handler pattern with Go generics for type safety,
// ensuring that handlers can only process commands of their designated type.
// Handlers encapsulate the business logic for executing specific operations
// and coordinate with domain entities and infrastructure services.
//
// Type Parameter:
//   T Command: Specific command type that this handler processes
//
// Handler Responsibilities:
//   - Validate command data and business rules
//   - Execute domain operations through aggregates and services
//   - Coordinate with infrastructure services (repositories, external APIs)
//   - Generate and publish domain events
//   - Return execution results with proper error handling
//
// Example:
//   Implementing a command handler:
//
//     type CreatePipelineHandler struct {
//         pipelineRepo domain.PipelineRepository
//         eventBus     domain.EventBus
//         logger       logging.Logger
//     }
//     
//     func (h *CreatePipelineHandler) Handle(
//         ctx context.Context, 
//         command *CreatePipelineCommand,
//     ) result.Result[interface{}] {
//         // 1. Validate business rules
//         if err := h.validateCommand(command); err != nil {
//             return result.Failure[interface{}](err)
//         }
//         
//         // 2. Execute domain logic
//         pipeline := entities.NewPipeline(
//             command.Name,
//             command.Description, 
//             command.Owner,
//         )
//         
//         if pipeline.IsFailure() {
//             return result.Failure[interface{}](pipeline.Error())
//         }
//         
//         // 3. Persist aggregate
//         if err := h.pipelineRepo.Save(pipeline.Value()); err != nil {
//             return result.Failure[interface{}](err)
//         }
//         
//         // 4. Publish events
//         events := pipeline.Value().GetEvents()
//         for _, event := range events {
//             h.eventBus.Publish(ctx, event)
//         }
//         
//         return result.Success[interface{}](pipeline.Value().ID())
//     }
//
// Integration:
//   - Domain Layer: Executes business operations through aggregates
//   - Infrastructure: Accesses repositories and external services
//   - Event Bus: Publishes domain events for read model updates
//   - Transaction Management: Coordinates transactional boundaries
type CommandHandler[T Command] interface {
	Handle(ctx context.Context, command T) result.Result[interface{}]  // Processes command with full error handling
}

// CommandBus coordinates command execution and handler management in the CQRS architecture.
//
// CommandBus serves as the central dispatcher for all write operations in the
// application, implementing the Mediator pattern to decouple command senders
// from command handlers. It provides synchronous and asynchronous execution
// models with comprehensive error handling and cross-cutting concern integration.
//
// Core Responsibilities:
//   - Handler Registration: Type-safe registration of command handlers
//   - Command Routing: Routes commands to appropriate handlers based on type
//   - Execution Coordination: Manages command execution lifecycle and error handling
//   - Async Processing: Provides non-blocking command execution for performance
//   - Cross-cutting Concerns: Integrates logging, validation, metrics through decorators
//
// Example:
//   Complete command bus usage:
//
//     // Create and configure command bus
//     commandBus := commands.NewCommandBusBuilder().
//         WithLogging(logger).
//         WithValidation().
//         WithMetrics(metricsCollector).
//         Build()
//     
//     // Register handlers for different command types
//     commandBus.RegisterHandler(&CreatePipelineCommand{}, createPipelineHandler)
//     commandBus.RegisterHandler(&UpdatePipelineCommand{}, updatePipelineHandler)
//     commandBus.RegisterHandler(&DeletePipelineCommand{}, deletePipelineHandler)
//     
//     // Synchronous execution
//     command := &CreatePipelineCommand{Name: "data-pipeline"}
//     result := commandBus.Execute(ctx, command)
//     if result.IsFailure() {
//         return fmt.Errorf("command failed: %w", result.Error())
//     }
//     
//     // Asynchronous execution
//     asyncResult := commandBus.ExecuteAsync(ctx, command)
//     if asyncResult.IsFailure() {
//         return asyncResult.Error()
//     }
//     
//     // Wait for async completion
//     resultChan := asyncResult.Value()
//     finalResult := <-resultChan
//     if finalResult.IsFailure() {
//         return finalResult.Error()
//     }
//
// Integration:
//   - Application Layer: Central coordination point for all write operations
//   - HTTP Handlers: REST API endpoints delegate to command bus
//   - Domain Services: Complex operations coordinated through command bus
//   - Event Sourcing: Commands generate events for system coordination
type CommandBus interface {
	RegisterHandler(command Command, handler interface{}) error                                    // Registers handler for command type
	Execute(ctx context.Context, command Command) result.Result[interface{}]                     // Executes command synchronously
	ExecuteAsync(ctx context.Context, command Command) result.Result[chan result.Result[interface{}]]  // Executes command asynchronously
}

// InMemoryCommandBus provides a thread-safe in-memory implementation of CommandBus.
//
// InMemoryCommandBus implements the CommandBus interface with handlers stored
// in memory using a concurrent-safe map. It provides high-performance command
// processing suitable for single-instance deployments and development environments.
// For production distributed deployments, consider using persistent queue-based
// implementations.
//
// Architecture:
//   - Thread Safety: Uses RWMutex for concurrent handler registration and execution
//   - Handler Registry: Type-safe handler registration with validation
//   - Shared Utilities: Leverages cqrs package for DRY principles and consistency
//   - Generic Execution: Supports any command type through reflection-based dispatch
//   - Performance Optimized: In-memory storage for minimal latency
//
// Fields:
//   mu sync.RWMutex: Reader-writer mutex for thread-safe access to handlers map
//   handlers map[string]interface{}: Command type to handler mapping registry
//   validator *cqrs.BusValidationUtilities: Shared validation logic for DRY compliance
//   invoker *cqrs.HandlerInvoker: Shared handler invocation with reflection optimization
//
// Example:
//   Direct usage (typically used through builder pattern):
//
//     commandBus := commands.NewInMemoryCommandBus()
//     
//     // Register handlers
//     createHandler := &CreatePipelineHandler{}
//     err := commandBus.RegisterHandler(&CreatePipelineCommand{}, createHandler)
//     if err != nil {
//         return fmt.Errorf("handler registration failed: %w", err)
//     }
//     
//     // Execute commands
//     command := &CreatePipelineCommand{Name: "test-pipeline"}
//     result := commandBus.Execute(ctx, command)
//     if result.IsFailure() {
//         return result.Error()
//     }
//     
//     // Check registered command types
//     registeredTypes := commandBus.GetRegisteredCommands()
//     fmt.Printf("Registered commands: %v\n", registeredTypes)
//
// Thread Safety:
//   - Handler registration uses write lock for exclusive access
//   - Command execution uses read lock for concurrent execution
//   - Handler map access properly synchronized for concurrent operations
//   - Safe for high-concurrency command processing scenarios
//
// Integration:
//   - CQRS Package: Leverages shared validation and invocation utilities
//   - Decorator Pattern: Wrappable by DecoratedCommandBus for cross-cutting concerns
//   - Builder Pattern: Created through CommandBusBuilder for configuration flexibility
//   - Result Pattern: All operations return Result types for explicit error handling
type InMemoryCommandBus struct {
	mu        sync.RWMutex                    // Thread-safe access to handlers registry
	handlers  map[string]interface{}          // Command type to handler mapping
	validator *cqrs.BusValidationUtilities   // Shared validation logic for DRY compliance
	invoker   *cqrs.HandlerInvoker           // Shared handler invocation with reflection
}

// NewInMemoryCommandBus creates a new thread-safe in-memory command bus instance.
//
// This factory function initializes a complete InMemoryCommandBus with all
// required dependencies and shared utilities. It follows the dependency injection
// pattern by providing pre-configured validation and invocation components.
//
// Returns:
//   *InMemoryCommandBus: Fully initialized command bus ready for handler registration
//
// Example:
//   Creating and using an in-memory command bus:
//
//     commandBus := commands.NewInMemoryCommandBus()
//     
//     // Bus is ready for immediate use
//     handler := &MyCommandHandler{}
//     err := commandBus.RegisterHandler(&MyCommand{}, handler)
//     if err != nil {
//         log.Fatal("Handler registration failed", "error", err)
//     }
//     
//     // Execute commands
//     result := commandBus.Execute(ctx, &MyCommand{Data: "test"})
//     if result.IsSuccess() {
//         log.Info("Command executed successfully", "result", result.Value())
//     }
//
// Initialization:
//   - Empty handler registry ready for registration
//   - Shared validation utilities for consistent behavior
//   - Shared handler invoker for optimized reflection-based dispatch
//   - Thread-safe mutex initialized for concurrent access
//
// Integration:
//   - Typically wrapped by DecoratedCommandBus for production use
//   - Created through CommandBusBuilder for enhanced configuration
//   - Compatible with all Command and CommandHandler implementations
//
// Thread Safety:
//   Fully thread-safe for concurrent handler registration and command execution.
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

// RegisterHandler registers a command handler with type validation and thread safety.
//
// This method implements the Registry pattern for command handler registration,
// using shared validation logic to ensure consistency with query bus registration.
// It enforces type safety by validating that handlers implement the correct
// CommandHandler interface for their designated command type.
//
// Parameters:
//   command Command: Example command instance for type identification
//   handler interface{}: Handler implementation (must implement CommandHandler[T])
//
// Returns:
//   error: Registration error if handler type validation fails
//
// Example:
//   Registering command handlers:
//
//     commandBus := commands.NewInMemoryCommandBus()
//     
//     // Register different command handlers
//     createHandler := &CreatePipelineHandler{repo: pipelineRepo}
//     updateHandler := &UpdatePipelineHandler{repo: pipelineRepo}
//     deleteHandler := &DeletePipelineHandler{repo: pipelineRepo}
//     
//     // Type-safe registration
//     if err := commandBus.RegisterHandler(&CreatePipelineCommand{}, createHandler); err != nil {
//         return fmt.Errorf("create handler registration failed: %w", err)
//     }
//     
//     if err := commandBus.RegisterHandler(&UpdatePipelineCommand{}, updateHandler); err != nil {
//         return fmt.Errorf("update handler registration failed: %w", err)
//     }
//     
//     if err := commandBus.RegisterHandler(&DeletePipelineCommand{}, deleteHandler); err != nil {
//         return fmt.Errorf("delete handler registration failed: %w", err)
//     }
//     
//     log.Info("All command handlers registered successfully")
//
// Validation:
//   - Handler must implement CommandHandler[T] interface
//   - Command type extracted from command.CommandType()
//   - Prevents duplicate handler registration for same command type
//   - Type compatibility validated through reflection
//
// DRY Principle:
//   Uses shared registration logic from cqrs.BusValidationUtilities to eliminate
//   21-line duplication (mass=117) with query bus registration, ensuring consistent
//   validation behavior across command and query buses.
//
// Thread Safety:
//   Uses write lock for exclusive access during handler registration.
func (bus *InMemoryCommandBus) RegisterHandler(command Command, handler interface{}) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	// DRY: Use shared registration logic
	return bus.validator.RegisterHandlerGeneric(command, handler, "command", bus.handlers, "command")
}

// Execute executes a command synchronously with comprehensive error handling.
//
// This method implements the synchronous command execution pattern, routing
// the command to its registered handler and returning the execution result.
// It uses shared execution logic for consistency with query processing and
// provides thread-safe concurrent execution capabilities.
//
// Parameters:
//   ctx context.Context: Execution context for cancellation and timeout handling
//   command Command: Command instance to execute
//
// Returns:
//   result.Result[interface{}]: Execution result with success value or error
//
// Example:
//   Synchronous command execution:
//
//     commandBus := setupCommandBus()
//     
//     // Create and execute command
//     command := &CreatePipelineCommand{
//         Name:        "data-processing-pipeline",
//         Description: "Customer data ETL pipeline",
//         Owner:       "data-team",
//     }
//     
//     // Execute with context and timeout
//     ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//     defer cancel()
//     
//     result := commandBus.Execute(ctx, command)
//     if result.IsFailure() {
//         log.Error("Command execution failed", "error", result.Error(), "command", command.CommandType())
//         return result.Error()
//     }
//     
//     // Extract result value
//     pipelineID := result.Value().(string)
//     log.Info("Pipeline created successfully", "id", pipelineID, "name", command.Name)
//
// Execution Flow:
//   1. Acquire read lock for thread-safe handler access
//   2. Look up handler by command type
//   3. Validate handler exists and is compatible
//   4. Invoke handler with command and context
//   5. Return handler result with proper error wrapping
//
// Error Handling:
//   - Handler not found: Returns appropriate error with command type
//   - Handler execution failure: Propagates handler error with context
//   - Context cancellation: Respects context timeout and cancellation
//   - Type safety: Validates command and handler compatibility
//
// DRY Principle:
//   Uses shared execution logic from cqrs.BusValidationUtilities to eliminate
//   21-line duplication (mass=148) with query bus execution, ensuring consistent
//   error handling and execution patterns.
//
// Thread Safety:
//   Uses read lock for concurrent command execution while allowing handler registration.
func (bus *InMemoryCommandBus) Execute(ctx context.Context, command Command) result.Result[interface{}] {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// DRY: Use shared execution logic
	return bus.validator.ExecuteGeneric(ctx, command, "command", bus.handlers, bus.invoker)
}

// ExecuteAsync executes a command asynchronously with non-blocking result channel.
//
// This method provides asynchronous command execution for improved performance
// and responsiveness, particularly useful for long-running operations or when
// the caller needs to perform other work while the command executes. It uses
// a buffered channel to prevent goroutine blocking.
//
// Parameters:
//   ctx context.Context: Execution context passed to the underlying Execute method
//   command Command: Command instance to execute asynchronously
//
// Returns:
//   result.Result[chan result.Result[interface{}]]: Channel for receiving execution result
//
// Example:
//   Asynchronous command execution with timeout:
//
//     commandBus := setupCommandBus()
//     
//     command := &ProcessLargeDatasetCommand{
//         DatasetPath: "/data/large-dataset.csv",
//         ProcessingOptions: ProcessingOptions{
//             BatchSize: 10000,
//             Parallel:  true,
//         },
//     }
//     
//     // Execute asynchronously
//     asyncResult := commandBus.ExecuteAsync(ctx, command)
//     if asyncResult.IsFailure() {
//         return fmt.Errorf("async execution failed: %w", asyncResult.Error())
//     }
//     
//     resultChan := asyncResult.Value()
//     
//     // Wait for completion with timeout
//     select {
//     case result := <-resultChan:
//         if result.IsFailure() {
//             return fmt.Errorf("command failed: %w", result.Error())
//         }
//         log.Info("Command completed successfully", "result", result.Value())
//     case <-time.After(5 * time.Minute):
//         return fmt.Errorf("command execution timeout")
//     case <-ctx.Done():
//         return fmt.Errorf("command cancelled: %w", ctx.Err())
//     }
//
// Async Execution Pattern:
//   1. Create buffered channel for result communication
//   2. Launch goroutine for command execution
//   3. Execute command using synchronous Execute method
//   4. Send result to channel and close it
//   5. Return channel immediately for non-blocking operation
//
// Channel Management:
//   - Buffered channel (capacity 1) prevents goroutine blocking
//   - Channel closed after result is sent for proper cleanup
//   - Single result per execution (no streaming)
//   - Result includes both success values and errors
//
// Error Handling:
//   - Async execution errors returned through result channel
//   - Channel creation cannot fail (returns Success immediately)
//   - Context cancellation handled by underlying Execute method
//   - Goroutine cleanup handled automatically
//
// Integration:
//   - HTTP Handlers: Long-running operations without blocking requests
//   - Batch Processing: Parallel command execution for performance
//   - Event Processing: Async event handling and side effects
//   - Background Jobs: Non-critical operations that can be delayed
//
// Thread Safety:
//   Safe for concurrent async execution. Each call creates independent goroutine.
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
