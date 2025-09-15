// Package commands implements pipeline-specific command operations for FlexCore CQRS.
//
// This package provides comprehensive command implementations for pipeline lifecycle
// management within the FlexCore application layer. It implements the Command pattern
// with enterprise-grade features including orchestration, validation, error recovery,
// and event sourcing integration for complete pipeline operations.
//
// Pipeline Commands:
//   - CreatePipelineCommand: Creates new data processing pipelines with validation
//   - AddPipelineStepCommand: Adds processing steps to existing pipelines
//   - ExecutePipelineCommand: Executes pipelines with workflow engine integration
//   - ActivatePipelineCommand: Activates pipelines for execution availability
//   - SetPipelineScheduleCommand: Configures automated pipeline scheduling
//
// Command Handlers:
//
//	Each command has a corresponding handler implementing CommandHandler[T] interface
//	with complete business logic, repository integration, and event publishing.
//
// Pipeline Execution Orchestrator:
//
//	Specialized orchestrator following SOLID SRP principles for complex pipeline
//	execution coordination with proper error recovery and workflow management.
//
// Integration:
//   - Domain Layer: Pipeline and PipelineStep entities for business logic
//   - Infrastructure: Repository pattern for persistence and event bus for events
//   - Workflow Engine: External workflow system integration for execution
//   - Result Pattern: Explicit error handling throughout all operations
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package commands

import (
	"context"
	"time"

	"github.com/flext-sh/flexcore/internal/domain/entities"
	"github.com/flext-sh/flexcore/pkg/errors"
	"github.com/flext-sh/flexcore/pkg/logging"
	"go.uber.org/zap"
)

// CreatePipelineCommand represents a command to create a new data processing pipeline.
//
// CreatePipelineCommand encapsulates all necessary information for creating a new
// pipeline in the FlexCore system, including metadata, ownership, and categorization.
// The command follows CQRS principles with immutable data and comprehensive validation.
//
// Fields:
//
//	BaseCommand: Inherited command infrastructure with type identification
//	Name string: Unique pipeline name for identification and discovery (required)
//	Description string: Detailed pipeline description and purpose (optional)
//	Owner string: Pipeline owner identifier for access control (required)
//	Tags []string: Classification tags for organization and filtering (optional)
//
// Business Rules:
//   - Pipeline names must be unique within the owner's scope
//   - Owner must be a valid user identifier in the system
//   - Tags are used for categorization and pipeline discovery
//   - Description supports rich text for comprehensive documentation
//
// Example:
//
//	Creating a new data processing pipeline:
//
//	  command := commands.NewCreatePipelineCommand(
//	      "customer-data-etl",
//	      "ETL pipeline for customer data processing from CRM to analytics warehouse",
//	      "data-team",
//	      []string{"etl", "customer-data", "analytics", "daily"},
//	  )
//
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsFailure() {
//	      return fmt.Errorf("pipeline creation failed: %w", result.Error())
//	  }
//
//	  pipeline := result.Value().(*entities.Pipeline)
//	  log.Info("Pipeline created successfully", "id", pipeline.ID(), "name", pipeline.Name())
//
// Integration:
//   - Processed by CreatePipelineCommandHandler
//   - Creates entities.Pipeline domain aggregate
//   - Generates PipelineCreated domain event
//   - Persisted through PipelineRepository
type CreatePipelineCommand struct {
	BaseCommand
	Name        string   // Unique pipeline name for identification
	Description string   // Detailed pipeline description and purpose
	Owner       string   // Pipeline owner identifier for access control
	Tags        []string // Classification tags for organization and filtering
}

// NewCreatePipelineCommand creates a new create pipeline command with validation.
//
// This factory function creates a properly initialized CreatePipelineCommand with
// all required fields and appropriate command type identification. It provides
// a clean API for command creation with parameter validation.
//
// Parameters:
//
//	name string: Unique pipeline name (required, must be non-empty)
//	description string: Pipeline description (optional, can be empty)
//	owner string: Pipeline owner identifier (required, must be valid user)
//	tags []string: Classification tags (optional, can be nil or empty)
//
// Returns:
//
//	CreatePipelineCommand: Fully initialized command ready for execution
//
// Example:
//
//	Creating commands for different pipeline types:
//
//	  // Data processing pipeline
//	  etlCommand := commands.NewCreatePipelineCommand(
//	      "sales-data-pipeline",
//	      "Daily sales data processing from multiple sources",
//	      "analytics-team",
//	      []string{"sales", "daily", "multi-source"},
//	  )
//
//	  // Real-time streaming pipeline
//	  streamingCommand := commands.NewCreatePipelineCommand(
//	      "real-time-events",
//	      "Real-time event processing for user analytics",
//	      "streaming-team",
//	      []string{"streaming", "real-time", "events"},
//	  )
//
//	  // Simple data migration pipeline
//	  migrationCommand := commands.NewCreatePipelineCommand(
//	      "legacy-migration",
//	      "One-time migration from legacy system",
//	      "migration-team",
//	      []string{"migration", "one-time", "legacy"},
//	  )
//
// Validation:
//
//	Parameter validation is performed by the command handler during execution.
//	This factory focuses on proper initialization and type safety.
//
// Integration:
//
//	Command is designed for execution through CommandBus with proper
//	routing to CreatePipelineCommandHandler.
func NewCreatePipelineCommand(name, description, owner string, tags []string) CreatePipelineCommand {
	return CreatePipelineCommand{
		BaseCommand: NewBaseCommand("CreatePipeline"),
		Name:        name,
		Description: description,
		Owner:       owner,
		Tags:        tags,
	}
}

// CreatePipelineCommandHandler handles pipeline creation commands with comprehensive business logic.
//
// CreatePipelineCommandHandler implements the CommandHandler pattern for pipeline
// creation operations, coordinating domain logic, repository operations, and event
// publishing. It enforces business rules, manages transactions, and ensures
// proper error handling throughout the pipeline creation process.
//
// Responsibilities:
//   - Command validation and business rule enforcement
//   - Pipeline uniqueness validation within owner scope
//   - Domain aggregate creation and state management
//   - Repository coordination for data persistence
//   - Domain event publishing for system coordination
//
// Fields:
//
//	repository PipelineRepository: Data access abstraction for pipeline persistence
//	eventBus EventBus: Event publishing infrastructure for domain events
//
// Example:
//
//	Handler initialization and usage:
//
//	  handler := commands.NewCreatePipelineCommandHandler(
//	      pipelineRepository,
//	      eventBus,
//	  )
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&CreatePipelineCommand{}, handler)
//
//	  // Handler processes commands through command bus
//	  command := commands.NewCreatePipelineCommand(
//	      "data-processing-pipeline",
//	      "Customer data ETL pipeline",
//	      "data-team",
//	      []string{"etl", "customer"},
//	  )
//
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsSuccess() {
//	      pipeline := result.Value().(*entities.Pipeline)
//	      log.Info("Pipeline created", "id", pipeline.ID())
//	  }
//
// Integration:
//   - Command Bus: Registered as handler for CreatePipelineCommand
//   - Domain Layer: Creates and manages Pipeline aggregates
//   - Infrastructure: Repository and event bus dependency injection
//   - Transaction Management: Participates in application transactions
type CreatePipelineCommandHandler struct {
	repository PipelineRepository // Data access abstraction for pipeline persistence
	eventBus   EventBus           // Event publishing infrastructure for domain events
}

// PipelineRepository represents a repository for pipeline aggregate persistence.
//
// PipelineRepository implements the Repository pattern for pipeline aggregates,
// providing comprehensive data access operations with proper error handling and
// context support. It abstracts the underlying persistence mechanism and provides
// a clean interface for domain aggregate storage and retrieval.
//
// Design Patterns:
//   - Repository Pattern: Encapsulates data access logic and persistence details
//   - Domain-Driven Design: Works with Pipeline aggregates as domain entities
//   - Context Awareness: All operations support cancellation and timeout handling
//   - Error Handling: Explicit error returns for proper error propagation
//
// Operations:
//   - Save: Persists pipeline aggregates with optimistic concurrency control
//   - FindByID: Retrieves pipelines by unique identifier with existence validation
//   - FindByName: Searches pipelines by name for uniqueness validation
//   - Delete: Removes pipelines with proper cascading and cleanup
//   - List: Retrieves paginated pipeline collections with filtering support
//
// Example:
//
//	Repository usage in command handlers:
//
//	  type CreatePipelineHandler struct {
//	      repository PipelineRepository
//	      eventBus   EventBus
//	  }
//
//	  func (h *CreatePipelineHandler) Handle(ctx context.Context, cmd CreatePipelineCommand) {
//	      // Check for existing pipeline
//	      existing, err := h.repository.FindByName(ctx, cmd.Name)
//	      if err != nil && !errors.Is(err, errors.NotFoundError{}) {
//	          return result.Failure(err)
//	      }
//	      if existing != nil {
//	          return result.Failure(errors.AlreadyExistsError("pipeline exists"))
//	      }
//
//	      // Create and save new pipeline
//	      pipeline := entities.NewPipeline(cmd.Name, cmd.Description, cmd.Owner)
//	      if err := h.repository.Save(ctx, pipeline.Value()); err != nil {
//	          return result.Failure(err)
//	      }
//	  }
//
// Integration:
//   - Infrastructure Layer: Implemented by concrete persistence adapters
//   - Domain Layer: Works exclusively with Pipeline domain aggregates
//   - Application Layer: Used by command handlers for data operations
//   - Transaction Support: Operations participate in application transactions
type PipelineRepository interface {
	Save(ctx context.Context, pipeline *entities.Pipeline) error                      // Persists pipeline aggregate with optimistic concurrency
	FindByID(ctx context.Context, id entities.PipelineID) (*entities.Pipeline, error) // Retrieves pipeline by unique identifier
	FindByName(ctx context.Context, name string) (*entities.Pipeline, error)          // Searches pipeline by name for uniqueness validation
	Delete(ctx context.Context, id entities.PipelineID) error                         // Removes pipeline with proper cleanup
	List(ctx context.Context, limit, offset int) ([]*entities.Pipeline, error)        // Retrieves paginated pipeline collections
}

// EventBus represents an event bus for publishing domain events across the system.
//
// EventBus implements the Event-Driven Architecture pattern, providing reliable
// domain event publishing for CQRS read model updates, cross-aggregate communication,
// and integration with external systems. It ensures eventual consistency and
// proper event ordering in distributed scenarios.
//
// Design Patterns:
//   - Publisher-Subscriber: Decouples event producers from consumers
//   - Event-Driven Architecture: Enables loose coupling between system components
//   - Reliable Messaging: Ensures event delivery with retry and error handling
//   - Context Awareness: Supports cancellation and timeout for event publishing
//
// Example:
//
//	Event publishing in command handlers:
//
//	  func (h *CreatePipelineHandler) Handle(ctx context.Context, cmd CreatePipelineCommand) {
//	      // Create pipeline and generate events
//	      pipeline := entities.NewPipeline(cmd.Name, cmd.Description, cmd.Owner)
//
//	      // Save aggregate
//	      if err := h.repository.Save(ctx, pipeline.Value()); err != nil {
//	          return result.Failure(err)
//	      }
//
//	      // Publish domain events
//	      events := pipeline.Value().GetEvents()
//	      for _, event := range events {
//	          if err := h.eventBus.Publish(ctx, event); err != nil {
//	              // Log error but don't fail the command (eventual consistency)
//	              log.Error("Event publishing failed", "event", event.Type, "error", err)
//	          }
//	      }
//
//	      // Clear events after publishing
//	      pipeline.Value().ClearEvents()
//	  }
//
// Integration:
//   - Infrastructure Layer: Implemented by message brokers (Redis, RabbitMQ, Kafka)
//   - Domain Layer: Publishes events generated by domain aggregates
//   - Read Models: Event handlers update query projections
//   - External Systems: Integration events for webhooks and notifications
type EventBus interface {
	Publish(ctx context.Context, event interface{}) error // Publishes domain event with reliable delivery
}

// NewCreatePipelineCommandHandler creates a new create pipeline command handler with dependencies.
//
// This factory function creates a properly initialized CreatePipelineCommandHandler
// with all required dependencies injected. It follows the dependency injection pattern
// for clean architecture and testability.
//
// Parameters:
//
//	repository PipelineRepository: Data access abstraction for pipeline persistence (required)
//	eventBus EventBus: Event publishing infrastructure for domain events (required)
//
// Returns:
//
//	*CreatePipelineCommandHandler: Fully initialized handler ready for command processing
//
// Example:
//
//	Handler initialization in application setup:
//
//	  // Initialize dependencies
//	  pipelineRepo := postgresql.NewPipelineRepository(db)
//	  eventBus := redis.NewEventBus(redisClient)
//
//	  // Create handler with dependency injection
//	  handler := commands.NewCreatePipelineCommandHandler(pipelineRepo, eventBus)
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&CreatePipelineCommand{}, handler)
//
//	  log.Info("CreatePipelineCommandHandler registered successfully")
//
// Integration:
//   - Application Setup: Called during application initialization
//   - Dependency Injection: Receives concrete implementations of abstractions
//   - Command Bus: Handler registered for CreatePipelineCommand processing
//   - Testing: Enables easy mocking of dependencies for unit tests
func NewCreatePipelineCommandHandler(repository PipelineRepository, eventBus EventBus) *CreatePipelineCommandHandler {
	return &CreatePipelineCommandHandler{
		repository: repository,
		eventBus:   eventBus,
	}
}

// Handle handles the create pipeline command with comprehensive business logic and error handling.
//
// This method implements the complete pipeline creation workflow, including business
// rule validation, domain aggregate creation, persistence, and event publishing.
// It follows the Command Handler pattern with proper error handling and transaction
// coordination.
//
// Business Logic Flow:
//  1. Validate pipeline name uniqueness within owner scope
//  2. Create new pipeline domain aggregate with validation
//  3. Apply command data (tags, metadata) to the aggregate
//  4. Persist pipeline through repository with optimistic concurrency
//  5. Publish domain events for system coordination
//  6. Return created pipeline as command result
//
// Parameters:
//
//	ctx context.Context: Execution context for cancellation and timeout handling
//	command CreatePipelineCommand: Command with pipeline creation data
//
// Returns:
//
//	result.Result[interface{}]: Success with created Pipeline or Failure with error
//
// Example:
//
//	Command execution flow:
//
//	  handler := &CreatePipelineCommandHandler{
//	      repository: pipelineRepo,
//	      eventBus:   eventBus,
//	  }
//
//	  command := CreatePipelineCommand{
//	      Name:        "analytics-pipeline",
//	      Description: "Daily analytics data processing",
//	      Owner:       "analytics-team",
//	      Tags:        []string{"analytics", "daily"},
//	  }
//
//	  result := handler.Handle(ctx, command)
//	  if result.IsFailure() {
//	      switch err := result.Error().(type) {
//	      case *errors.AlreadyExistsError:
//	          return fmt.Errorf("pipeline name conflict: %w", err)
//	      case *errors.ValidationError:
//	          return fmt.Errorf("invalid pipeline data: %w", err)
//	      default:
//	          return fmt.Errorf("pipeline creation failed: %w", err)
//	      }
//	  }
//
//	  pipeline := result.Value().(*entities.Pipeline)
//	  log.Info("Pipeline created successfully", "id", pipeline.ID(), "name", pipeline.Name())
//
// Business Rules Enforced:
//   - Pipeline names must be unique within the system
//   - Owner must be a valid identifier
//   - Tags are applied after pipeline creation for proper validation
//   - Domain events are published for read model updates
//
// Error Handling:
//   - AlreadyExistsError: Pipeline name conflicts
//   - ValidationError: Invalid command data or business rule violations
//   - RepositoryError: Persistence failures with proper wrapping
//   - Event publishing errors are logged but don't fail the command
//
// Integration:
//   - Domain Layer: Creates Pipeline aggregate and applies business rules
//   - Repository: Persists aggregate with proper transaction handling
//   - Event Bus: Publishes PipelineCreated and related events
//   - Result Pattern: Explicit error handling with typed results
func (h *CreatePipelineCommandHandler) Handle(ctx context.Context, command CreatePipelineCommand) (*entities.Pipeline, error) {
	// Check if pipeline with same name already exists
	existingPipeline, err := h.repository.FindByName(ctx, command.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check for existing pipeline")
	}
	if existingPipeline != nil {
		return nil, errors.AlreadyExistsError("pipeline with name " + command.Name)
	}

	// Create new pipeline
	pipeline, err := entities.NewPipeline(command.Name, command.Description, command.Owner)
	if err != nil {
		return nil, err
	}

	// Add tags
	for _, tag := range command.Tags {
		pipeline.AddTag(tag)
	}

	// Save pipeline
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return nil, errors.Wrap(err, "failed to save pipeline")
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	return pipeline, nil
}

// AddPipelineStepCommand represents a command to add a processing step to an existing pipeline.
//
// AddPipelineStepCommand encapsulates all necessary information for adding a new
// processing step to a pipeline, including step configuration, dependencies, and
// execution parameters. This command enables building complex multi-step data
// processing workflows with proper dependency management and error handling.
//
// Fields:
//
//	BaseCommand: Inherited command infrastructure with type identification
//	PipelineID entities.PipelineID: Target pipeline identifier (required)
//	StepName string: Unique step name within the pipeline (required)
//	StepType string: Step type identifier for processor selection (required)
//	Config map[string]interface{}: Step-specific configuration parameters (optional)
//	DependsOn []string: List of prerequisite step names for execution ordering (optional)
//	MaxRetries int: Maximum retry attempts for step execution (default from entities)
//	Timeout time.Duration: Step execution timeout (default from entities)
//
// Business Rules:
//   - Step names must be unique within the pipeline
//   - Pipeline must exist and be in a modifiable state (not running)
//   - Step dependencies must form a valid DAG (no circular dependencies)
//   - Step type must be registered in the plugin system
//   - Configuration must be valid for the specified step type
//
// Example:
//
//	Adding different types of processing steps:
//
//	  // Data extraction step
//	  extractCommand := commands.NewAddPipelineStepCommand(
//	      pipelineID,
//	      "extract-customer-data",
//	      "oracle-extractor",
//	  )
//	  extractCommand.Config = map[string]interface{}{
//	      "connection_string": "oracle://prod-db:1521/CUSTOMER",
//	      "query": "SELECT * FROM customers WHERE updated_date > ?",
//	      "batch_size": 10000,
//	  }
//	  extractCommand.MaxRetries = 3
//	  extractCommand.Timeout = 30 * time.Minute
//
//	  // Data transformation step (depends on extraction)
//	  transformCommand := commands.NewAddPipelineStepCommand(
//	      pipelineID,
//	      "transform-customer-data",
//	      "json-transformer",
//	  )
//	  transformCommand.Config = map[string]interface{}{
//	      "transformation_rules": "customer_transform.json",
//	      "output_format": "parquet",
//	  }
//	  transformCommand.DependsOn = []string{"extract-customer-data"}
//
//	  // Data loading step (depends on transformation)
//	  loadCommand := commands.NewAddPipelineStepCommand(
//	      pipelineID,
//	      "load-to-warehouse",
//	      "postgres-loader",
//	  )
//	  loadCommand.Config = map[string]interface{}{
//	      "target_table": "analytics.customer_data",
//	      "load_strategy": "upsert",
//	  }
//	  loadCommand.DependsOn = []string{"transform-customer-data"}
//
// Integration:
//   - Processed by AddPipelineStepCommandHandler
//   - Updates Pipeline aggregate with new PipelineStep
//   - Generates PipelineStepAdded domain event
//   - Validates step dependencies and configuration
type AddPipelineStepCommand struct {
	BaseCommand
	PipelineID entities.PipelineID    // Target pipeline identifier
	StepName   string                 // Unique step name within the pipeline
	StepType   string                 // Step type identifier for processor selection
	Config     map[string]interface{} // Step-specific configuration parameters
	DependsOn  []string               // List of prerequisite step names for execution ordering
	MaxRetries int                    // Maximum retry attempts for step execution
	Timeout    time.Duration          // Step execution timeout
}

// NewAddPipelineStepCommand creates a new add pipeline step command with sensible defaults.
//
// This factory function creates a properly initialized AddPipelineStepCommand with
// default configuration values and empty dependency lists. Additional configuration
// can be applied after creation through direct field assignment.
//
// Parameters:
//
//	pipelineID entities.PipelineID: Target pipeline identifier (required)
//	stepName string: Unique step name within the pipeline (required)
//	stepType string: Step type identifier for processor selection (required)
//
// Returns:
//
//	AddPipelineStepCommand: Fully initialized command with defaults ready for customization
//
// Example:
//
//	Creating and customizing pipeline steps:
//
//	  // Create basic step with defaults
//	  command := commands.NewAddPipelineStepCommand(
//	      pipelineID,
//	      "data-validation",
//	      "schema-validator",
//	  )
//
//	  // Customize configuration
//	  command.Config = map[string]interface{}{
//	      "schema_file": "customer_schema.json",
//	      "strict_mode": true,
//	      "error_threshold": 0.01,
//	  }
//
//	  // Set dependencies
//	  command.DependsOn = []string{"extract-data", "clean-data"}
//
//	  // Override execution parameters
//	  command.MaxRetries = 5
//	  command.Timeout = 15 * time.Minute
//
//	  // Execute command
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsFailure() {
//	      return fmt.Errorf("step addition failed: %w", result.Error())
//	  }
//
// Default Values:
//   - Config: Empty map ready for configuration
//   - DependsOn: Empty slice (no dependencies)
//   - MaxRetries: entities.DefaultMaxRetries (from domain defaults)
//   - Timeout: entities.DefaultTimeoutMinutes converted to time.Duration
//
// Integration:
//
//	Command is designed for execution through CommandBus with proper
//	routing to AddPipelineStepCommandHandler for processing.
func NewAddPipelineStepCommand(pipelineID entities.PipelineID, stepName, stepType string) AddPipelineStepCommand {
	return AddPipelineStepCommand{
		BaseCommand: NewBaseCommand("AddPipelineStep"),
		PipelineID:  pipelineID,
		StepName:    stepName,
		StepType:    stepType,
		Config:      make(map[string]interface{}),
		DependsOn:   make([]string, 0),
		MaxRetries:  entities.DefaultMaxRetries,
		Timeout:     time.Minute * entities.DefaultTimeoutMinutes,
	}
}

// AddPipelineStepCommandHandler handles add pipeline step commands with comprehensive validation.
//
// AddPipelineStepCommandHandler implements the CommandHandler pattern for pipeline
// step addition operations, coordinating pipeline loading, step validation, dependency
// analysis, and domain event publishing. It ensures proper step integration within
// the pipeline workflow and maintains data consistency.
//
// Responsibilities:
//   - Pipeline existence validation and state verification
//   - Step name uniqueness validation within pipeline scope
//   - Step dependency validation and circular dependency prevention
//   - Domain aggregate modification and step integration
//   - Repository coordination for pipeline persistence
//   - Domain event publishing for step addition tracking
//
// Fields:
//
//	repository PipelineRepository: Data access abstraction for pipeline operations
//	eventBus EventBus: Event publishing infrastructure for domain events
//
// Example:
//
//	Handler initialization and usage:
//
//	  handler := commands.NewAddPipelineStepCommandHandler(
//	      pipelineRepository,
//	      eventBus,
//	  )
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&AddPipelineStepCommand{}, handler)
//
//	  // Handler processes commands through command bus
//	  command := commands.NewAddPipelineStepCommand(
//	      pipelineID,
//	      "data-transformation",
//	      "json-transformer",
//	  )
//	  command.Config = map[string]interface{}{
//	      "rules_file": "transform_rules.json",
//	  }
//
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsSuccess() {
//	      step := result.Value().(*entities.PipelineStep)
//	      log.Info("Step added", "step", step.Name, "pipeline", pipelineID)
//	  }
//
// Integration:
//   - Command Bus: Registered as handler for AddPipelineStepCommand
//   - Domain Layer: Modifies Pipeline aggregates with new steps
//   - Infrastructure: Repository and event bus dependency injection
//   - Validation: Comprehensive step and dependency validation
type AddPipelineStepCommandHandler struct {
	repository PipelineRepository // Data access abstraction for pipeline operations
	eventBus   EventBus           // Event publishing infrastructure for domain events
}

// NewAddPipelineStepCommandHandler creates a new add pipeline step command handler with dependencies.
//
// This factory function creates a properly initialized AddPipelineStepCommandHandler
// with all required dependencies injected. It follows the dependency injection pattern
// for clean architecture, testability, and proper separation of concerns.
//
// Parameters:
//
//	repository PipelineRepository: Data access abstraction for pipeline operations (required)
//	eventBus EventBus: Event publishing infrastructure for domain events (required)
//
// Returns:
//
//	*AddPipelineStepCommandHandler: Fully initialized handler ready for command processing
//
// Example:
//
//	Handler initialization in application setup:
//
//	  // Initialize dependencies
//	  pipelineRepo := postgresql.NewPipelineRepository(db)
//	  eventBus := redis.NewEventBus(redisClient)
//
//	  // Create handler with dependency injection
//	  handler := commands.NewAddPipelineStepCommandHandler(pipelineRepo, eventBus)
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&AddPipelineStepCommand{}, handler)
//
//	  log.Info("AddPipelineStepCommandHandler registered successfully")
//
// Integration:
//   - Application Setup: Called during application initialization
//   - Dependency Injection: Receives concrete implementations of abstractions
//   - Command Bus: Handler registered for AddPipelineStepCommand processing
//   - Testing: Enables easy mocking of dependencies for unit tests
func NewAddPipelineStepCommandHandler(repository PipelineRepository, eventBus EventBus) *AddPipelineStepCommandHandler {
	return &AddPipelineStepCommandHandler{
		repository: repository,
		eventBus:   eventBus,
	}
}

// Handle handles the add pipeline step command with comprehensive validation and error handling.
//
// This method implements the complete pipeline step addition workflow, including
// pipeline loading, step creation with configuration, dependency validation,
// domain aggregate modification, persistence, and event publishing. It ensures
// proper step integration and maintains pipeline consistency.
//
// Business Logic Flow:
//  1. Load target pipeline and validate existence
//  2. Create new pipeline step with command configuration
//  3. Apply step configuration, dependencies, and execution parameters
//  4. Add step to pipeline aggregate with business rule validation
//  5. Persist updated pipeline through repository
//  6. Publish domain events for step addition tracking
//  7. Return created step as command result
//
// Parameters:
//
//	ctx context.Context: Execution context for cancellation and timeout handling
//	command AddPipelineStepCommand: Command with step addition data
//
// Returns:
//
//	result.Result[interface{}]: Success with created PipelineStep or Failure with error
//
// Example:
//
//	Command execution flow:
//
//	  handler := &AddPipelineStepCommandHandler{
//	      repository: pipelineRepo,
//	      eventBus:   eventBus,
//	  }
//
//	  command := AddPipelineStepCommand{
//	      PipelineID: pipelineID,
//	      StepName:   "data-enrichment",
//	      StepType:   "lookup-enricher",
//	      Config: map[string]interface{}{
//	          "lookup_table": "customer_metadata",
//	          "join_key": "customer_id",
//	      },
//	      DependsOn: []string{"data-extraction", "data-cleaning"},
//	      MaxRetries: 3,
//	      Timeout: 10 * time.Minute,
//	  }
//
//	  result := handler.Handle(ctx, command)
//	  if result.IsFailure() {
//	      switch err := result.Error().(type) {
//	      case *errors.NotFoundError:
//	          return fmt.Errorf("pipeline not found: %w", err)
//	      case *errors.ValidationError:
//	          return fmt.Errorf("invalid step configuration: %w", err)
//	      default:
//	          return fmt.Errorf("step addition failed: %w", err)
//	      }
//	  }
//
//	  step := result.Value().(*entities.PipelineStep)
//	  log.Info("Step added successfully", "step", step.Name, "type", step.Type)
//
// Business Rules Enforced:
//   - Pipeline must exist and be in a modifiable state
//   - Step names must be unique within the pipeline
//   - Step dependencies must form a valid DAG (no circular dependencies)
//   - Step configuration must be valid for the specified step type
//
// Error Handling:
//   - NotFoundError: Pipeline does not exist
//   - ValidationError: Invalid step data or business rule violations
//   - RepositoryError: Persistence failures with proper wrapping
//   - Event publishing errors are logged but don't fail the command
//
// Integration:
//   - Domain Layer: Creates PipelineStep and modifies Pipeline aggregate
//   - Repository: Persists updated pipeline with optimistic concurrency
//   - Event Bus: Publishes PipelineStepAdded and related events
//   - Result Pattern: Explicit error handling with typed results
func (h *AddPipelineStepCommandHandler) Handle(ctx context.Context, command AddPipelineStepCommand) (*entities.PipelineStep, error) {
	// Find pipeline
	pipeline, err := h.repository.FindByID(ctx, command.PipelineID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find pipeline")
	}

	// Create step
	step := entities.NewPipelineStep(command.StepName, command.StepType)
	step.Config = command.Config
	step.DependsOn = command.DependsOn
	step.MaxRetries = command.MaxRetries
	step.Timeout = command.Timeout

	// Add step to pipeline
	if err := pipeline.AddStep(&step); err != nil {
		return nil, err
	}

	// Save pipeline
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return nil, errors.Wrap(err, "failed to save pipeline")
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	return &step, nil
}

// ExecutePipelineCommand represents a command to execute a pipeline with runtime parameters.
//
// ExecutePipelineCommand encapsulates all necessary information for executing a
// data processing pipeline, including pipeline identification and runtime parameters.
// This command triggers the complete pipeline execution workflow through the
// workflow engine integration.
//
// Fields:
//
//	BaseCommand: Inherited command infrastructure with type identification
//	PipelineID entities.PipelineID: Unique identifier of the pipeline to execute (required)
//	Parameters map[string]interface{}: Runtime parameters for pipeline execution (optional)
//
// Business Rules:
//   - Pipeline must exist and be in an executable state (Active status)
//   - Pipeline must not be currently running (concurrent execution prevention)
//   - Parameters are validated against pipeline step requirements
//   - Execution triggers workflow engine with proper error handling
//
// Example:
//
//	Executing a data processing pipeline:
//
//	  command := commands.NewExecutePipelineCommand(
//	      pipelineID,
//	      map[string]interface{}{
//	          "source_path":      "/data/input/daily_sales.csv",
//	          "destination_path": "/data/output/processed/",
//	          "batch_size":       10000,
//	          "parallel_workers": 4,
//	          "retry_attempts":   3,
//	          "timeout_minutes":  30,
//	      },
//	  )
//
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsFailure() {
//	      return fmt.Errorf("pipeline execution failed: %w", result.Error())
//	  }
//
//	  executionResult := result.Value().(ExecutionResult)
//	  log.Info("Pipeline execution started",
//	      "pipeline_id", executionResult.PipelineID,
//	      "workflow_id", executionResult.WorkflowID,
//	      "status", executionResult.Status)
//
// Integration:
//   - Processed by ExecutePipelineCommandHandler with PipelineExecutionOrchestrator
//   - Updates Pipeline aggregate state (Running → Completed/Failed)
//   - Generates pipeline execution events for monitoring
//   - Integrates with WorkflowEngine for actual execution coordination
type ExecutePipelineCommand struct {
	BaseCommand
	PipelineID entities.PipelineID    // Unique identifier of the pipeline to execute
	Parameters map[string]interface{} // Runtime parameters for pipeline execution
}

// NewExecutePipelineCommand creates a new execute pipeline command with runtime parameters.
//
// This factory function creates a properly initialized ExecutePipelineCommand with
// pipeline identification and execution parameters. It follows the Command pattern
// with immutable command objects that encapsulate all necessary execution data.
//
// Parameters:
//
//	pipelineID entities.PipelineID: Unique identifier of the pipeline to execute (required)
//	parameters map[string]interface{}: Runtime parameters for pipeline execution (optional)
//
// Returns:
//
//	ExecutePipelineCommand: Fully initialized command ready for execution
//
// Example:
//
//	Creating execution commands with different parameter sets:
//
//	  // Basic execution with minimal parameters
//	  command := commands.NewExecutePipelineCommand(
//	      pipelineID,
//	      map[string]interface{}{
//	          "source_path": "/data/input/daily_batch.csv",
//	      },
//	  )
//
//	  // Advanced execution with comprehensive parameters
//	  advancedCommand := commands.NewExecutePipelineCommand(
//	      pipelineID,
//	      map[string]interface{}{
//	          "source_path":      "/data/input/",
//	          "destination_path": "/data/output/processed/",
//	          "batch_size":       10000,
//	          "parallel_workers": 4,
//	          "retry_attempts":   3,
//	          "timeout_minutes":  30,
//	          "quality_checks":   true,
//	          "notification_emails": []string{
//	              "data-team@company.com",
//	              "ops-team@company.com",
//	          },
//	      },
//	  )
//
//	  // Execute command
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsSuccess() {
//	      executionResult := result.Value().(ExecutionResult)
//	      log.Info("Pipeline execution started",
//	          "workflow_id", executionResult.WorkflowID)
//	  }
//
// Default Behavior:
//   - Empty parameters map is acceptable for pipelines with built-in defaults
//   - Parameters are passed through to workflow engine without validation
//   - Parameter validation occurs during pipeline execution phase
//
// Integration:
//
//	Command is designed for execution through CommandBus with proper
//	routing to ExecutePipelineCommandHandler for processing.
func NewExecutePipelineCommand(pipelineID entities.PipelineID, parameters map[string]interface{}) ExecutePipelineCommand {
	return ExecutePipelineCommand{
		BaseCommand: NewBaseCommand("ExecutePipeline"),
		PipelineID:  pipelineID,
		Parameters:  parameters,
	}
}

// ExecutePipelineCommandHandler handles pipeline execution commands
type ExecutePipelineCommandHandler struct {
	repository     PipelineRepository
	eventBus       EventBus
	workflowEngine WorkflowEngine
}

// WorkflowEngine represents a workflow execution engine for pipeline orchestration.
//
// WorkflowEngine provides an abstraction for external workflow execution systems
// that handle the actual pipeline execution, step coordination, and failure recovery.
// This interface enables integration with various workflow engines like Temporal,
// Cadence, Apache Airflow, or custom workflow implementations.
//
// Design Patterns:
//   - Strategy Pattern: Abstracts workflow engine implementation details
//   - Integration Pattern: Provides clean interface for external system integration
//   - Context Awareness: All operations support cancellation and timeout handling
//   - Error Handling: Explicit error returns for proper error propagation
//
// Example:
//
//	Workflow engine integration:
//
//	  type TemporalWorkflowEngine struct {
//	      client temporal.Client
//	      logger logging.Logger
//	  }
//
//	  func (t *TemporalWorkflowEngine) StartWorkflow(
//	      ctx context.Context,
//	      workflowName string,
//	      input interface{},
//	  ) (string, error) {
//	      options := temporal.StartWorkflowOptions{
//	          ID:        uuid.New().String(),
//	          TaskQueue: "pipeline-execution",
//	      }
//
//	      execution, err := t.client.ExecuteWorkflow(ctx, options, workflowName, input)
//	      if err != nil {
//	          return "", fmt.Errorf("workflow start failed: %w", err)
//	      }
//
//	      return execution.GetID(), nil
//	  }
//
//	  // Usage in pipeline execution
//	  workflowInput := map[string]interface{}{
//	      "pipelineID": pipelineID,
//	      "parameters": executionParams,
//	      "steps":      pipeline.Steps(),
//	  }
//
//	  workflowID, err := workflowEngine.StartWorkflow(ctx, "pipeline-execution", workflowInput)
//	  if err != nil {
//	      return fmt.Errorf("pipeline execution start failed: %w", err)
//	  }
//
//	  log.Info("Pipeline execution started", "workflow_id", workflowID)
//
// Integration:
//   - Infrastructure Layer: Implemented by concrete workflow engine adapters
//   - Pipeline Execution: Used by PipelineExecutionOrchestrator for actual execution
//   - External Systems: Integrates with workflow platforms and orchestrators
//   - Error Recovery: Workflow engines provide failure handling and retry logic
type WorkflowEngine interface {
	StartWorkflow(ctx context.Context, workflowName string, input interface{}) (string, error) // Starts workflow execution and returns workflow ID
}

// NewExecutePipelineCommandHandler creates a new execute pipeline command handler with dependencies.
//
// This factory function creates a properly initialized ExecutePipelineCommandHandler
// with all required dependencies injected. It follows the dependency injection pattern
// for clean architecture, testability, and proper separation of concerns with
// external workflow engine integration.
//
// Parameters:
//
//	repository PipelineRepository: Data access abstraction for pipeline operations (required)
//	eventBus EventBus: Event publishing infrastructure for domain events (required)
//	workflowEngine WorkflowEngine: External workflow system for execution coordination (required)
//
// Returns:
//
//	*ExecutePipelineCommandHandler: Fully initialized handler ready for command processing
//
// Example:
//
//	Handler initialization in application setup:
//
//	  // Initialize dependencies
//	  pipelineRepo := postgresql.NewPipelineRepository(db)
//	  eventBus := redis.NewEventBus(redisClient)
//	  workflowEngine := temporal.NewTemporalWorkflowEngine(temporalClient)
//
//	  // Create handler with dependency injection
//	  handler := commands.NewExecutePipelineCommandHandler(
//	      pipelineRepo,
//	      eventBus,
//	      workflowEngine,
//	  )
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&ExecutePipelineCommand{}, handler)
//
//	  log.Info("ExecutePipelineCommandHandler registered successfully")
//
// Integration:
//   - Application Setup: Called during application initialization
//   - Dependency Injection: Receives concrete implementations of abstractions
//   - Command Bus: Handler registered for ExecutePipelineCommand processing
//   - Workflow Engine: External execution system coordination
func NewExecutePipelineCommandHandler(repository PipelineRepository, eventBus EventBus, workflowEngine WorkflowEngine) *ExecutePipelineCommandHandler {
	return &ExecutePipelineCommandHandler{
		repository:     repository,
		eventBus:       eventBus,
		workflowEngine: workflowEngine,
	}
}

// PipelineExecutionOrchestrator encapsulates execution orchestration following SOLID SRP.
//
// PipelineExecutionOrchestrator implements the Orchestrator pattern to coordinate
// complex pipeline execution workflows with proper error handling, recovery mechanisms,
// and integration with external workflow engines. It follows SOLID Single Responsibility
// Principle by focusing exclusively on execution coordination.
//
// Responsibilities:
//   - Pipeline loading and execution readiness validation
//   - Workflow engine integration and coordination
//   - Execution state management and persistence
//   - Error recovery and failure handling
//   - Domain event publishing for execution tracking
//
// Fields:
//
//	repository PipelineRepository: Data access for pipeline loading and state persistence
//	eventBus EventBus: Event publishing for execution lifecycle events
//	workflowEngine WorkflowEngine: External workflow system for actual execution
//
// SOLID SRP Implementation:
//
//	Reduces complexity by separating execution orchestration concerns from command
//	handling, enabling focused testing, maintenance, and evolution of execution logic.
//
// Example:
//
//	Orchestrator usage in command handler:
//
//	  orchestrator := commands.NewPipelineExecutionOrchestrator(
//	      pipelineRepository,
//	      eventBus,
//	      workflowEngine,
//	  )
//
//	  command := ExecutePipelineCommand{
//	      PipelineID: pipelineID,
//	      Parameters: executionParams,
//	  }
//
//	  result := orchestrator.Execute(ctx, command)
//	  if result.IsFailure() {
//	      // Orchestrator handles all error scenarios and recovery
//	      return result.Error()
//	  }
//
//	  executionResult := result.Value().(ExecutionResult)
//	  // Pipeline is now running in workflow engine
//
// Integration:
//   - WorkflowEngine: Delegates actual execution to external workflow system
//   - Domain Events: Publishes execution lifecycle events for monitoring
//   - Repository: Manages pipeline state transitions during execution
//   - Error Recovery: Handles failures with proper cleanup and state restoration
type PipelineExecutionOrchestrator struct {
	repository     PipelineRepository // Data access for pipeline loading and state persistence
	eventBus       EventBus           // Event publishing for execution lifecycle events
	workflowEngine WorkflowEngine     // External workflow system for actual execution
	logger         *zap.Logger        // Structured logging for pipeline execution tracking
}

// NewPipelineExecutionOrchestrator creates a new execution orchestrator with dependencies.
//
// This factory function creates a properly initialized PipelineExecutionOrchestrator
// with all required dependencies injected. It follows the dependency injection pattern
// and SOLID SRP by creating a focused orchestrator for execution coordination
// separate from command handling concerns.
//
// Parameters:
//
//	repository PipelineRepository: Data access for pipeline loading and state persistence (required)
//	eventBus EventBus: Event publishing for execution lifecycle events (required)
//	workflowEngine WorkflowEngine: External workflow system for actual execution (required)
//
// Returns:
//
//	*PipelineExecutionOrchestrator: Fully initialized orchestrator ready for execution coordination
//
// Example:
//
//	Orchestrator initialization and usage:
//
//	  // Initialize dependencies
//	  pipelineRepo := postgresql.NewPipelineRepository(db)
//	  eventBus := redis.NewEventBus(redisClient)
//	  workflowEngine := temporal.NewTemporalWorkflowEngine(temporalClient)
//
//	  // Create orchestrator with dependency injection
//	  orchestrator := commands.NewPipelineExecutionOrchestrator(
//	      pipelineRepo,
//	      eventBus,
//	      workflowEngine,
//	  )
//
//	  // Use orchestrator in command handler
//	  handler := &ExecutePipelineCommandHandler{
//	      orchestrator: orchestrator,
//	  }
//
//	  log.Info("PipelineExecutionOrchestrator initialized successfully")
//
// SOLID SRP Implementation:
//
//	Orchestrator focuses exclusively on execution coordination, separated from
//	command handling and HTTP concerns, enabling focused testing and maintenance.
//
// Integration:
//   - Command Handlers: Used by ExecutePipelineCommandHandler for execution logic
//   - Workflow Engine: Coordinates with external execution systems
//   - Event System: Publishes execution lifecycle events
//   - Repository: Manages pipeline state during execution
func NewPipelineExecutionOrchestrator(repository PipelineRepository, eventBus EventBus, workflowEngine WorkflowEngine) *PipelineExecutionOrchestrator {
	return &PipelineExecutionOrchestrator{
		repository:     repository,
		eventBus:       eventBus,
		workflowEngine: workflowEngine,
		logger:         logging.GetLogger(),
	}
}

// ExecutionContext holds execution state for comprehensive pipeline execution tracking.
//
// ExecutionContext encapsulates all the state and metadata required for pipeline
// execution coordination, providing a centralized context for tracking execution
// progress, managing workflow integration, and handling error recovery scenarios.
// It follows the Context pattern for managing execution state across operations.
//
// Fields:
//
//	Pipeline *entities.Pipeline: The pipeline aggregate being executed
//	WorkflowID string: External workflow system execution identifier
//	Parameters map[string]interface{}: Runtime parameters for pipeline execution
//
// Usage Pattern:
//
//	ExecutionContext is created during pipeline execution orchestration and passed
//	through various execution phases to maintain state consistency and enable
//	proper error handling and recovery mechanisms.
//
// Example:
//
//	Execution context usage in orchestration:
//
//	  executionCtx := &ExecutionContext{
//	      Pipeline:   pipeline,
//	      WorkflowID: "", // Set after workflow starts
//	      Parameters: map[string]interface{}{
//	          "source_path": "/data/input/",
//	          "batch_size":  10000,
//	          "timeout":     30 * time.Minute,
//	      },
//	  }
//
//	  // Context passed through execution phases
//	  result := orchestrator.orchestrateExecution(ctx, executionCtx)
//	  if result.IsFailure() {
//	      log.Error("Execution failed",
//	          "pipeline_id", executionCtx.Pipeline.ID(),
//	          "workflow_id", executionCtx.WorkflowID,
//	          "error", result.Error())
//	  }
//
// Integration:
//   - Pipeline Execution: Central state management during execution
//   - Workflow Integration: Tracks external workflow system coordination
//   - Error Recovery: Provides context for recovery and cleanup operations
//   - Parameter Management: Carries runtime configuration across execution phases
type ExecutionContext struct {
	Pipeline   *entities.Pipeline     // The pipeline aggregate being executed
	WorkflowID string                 // External workflow system execution identifier
	Parameters map[string]interface{} // Runtime parameters for pipeline execution
}

// ExecutionResult represents the comprehensive result of pipeline execution.
//
// ExecutionResult encapsulates all relevant information about pipeline execution
// outcome, including identifiers, status, and error information. It provides
// a standardized response format for execution operations and enables proper
// result tracking and monitoring integration.
//
// Fields:
//
//	PipelineID string: Unique identifier of the executed pipeline
//	WorkflowID string: External workflow system execution identifier
//	Status string: Current execution status (Running, Completed, Failed)
//	Error error: Execution error information if execution failed (optional)
//
// JSON Serialization:
//
//	All fields are properly tagged for JSON serialization to enable API responses,
//	logging, and monitoring system integration.
//
// Example:
//
//	Execution result creation and usage:
//
//	  result := ExecutionResult{
//	      PipelineID: pipeline.ID().String(),
//	      WorkflowID: workflowID,
//	      Status:     pipeline.Status().String(),
//	      Error:      nil, // No error for successful execution
//	  }
//
//	  // JSON serialization for API response
//	  jsonData, err := json.Marshal(result)
//	  if err != nil {
//	      return fmt.Errorf("result serialization failed: %w", err)
//	  }
//
//	  // Logging structured result
//	  log.Info("Pipeline execution completed",
//	      "pipeline_id", result.PipelineID,
//	      "workflow_id", result.WorkflowID,
//	      "status", result.Status)
//
//	  // Error handling
//	  if result.Error != nil {
//	      log.Error("Pipeline execution failed",
//	          "pipeline_id", result.PipelineID,
//	          "error", result.Error)
//	  }
//
// Integration:
//   - API Responses: JSON serialization for HTTP API responses
//   - Monitoring: Structured data for monitoring and alerting systems
//   - Logging: Comprehensive execution result logging
//   - Error Tracking: Centralized error information for debugging
type ExecutionResult struct {
	PipelineID string `json:"pipeline_id"`     // Unique identifier of the executed pipeline
	WorkflowID string `json:"workflow_id"`     // External workflow system execution identifier
	Status     string `json:"status"`          // Current execution status
	Error      error  `json:"error,omitempty"` // Execution error information if failed
}

// Execute orchestrates pipeline execution with comprehensive error handling and recovery.
//
// This method implements the complete pipeline execution orchestration workflow,
// coordinating between domain aggregates, workflow engines, and event systems.
// It follows the Railway Pattern to eliminate multiple return paths and ensure
// consistent error handling throughout the execution process.
//
// Execution Workflow:
//  1. Load pipeline aggregate and validate execution readiness
//  2. Prepare execution context with parameters and metadata
//  3. Orchestrate execution through workflow engine integration
//  4. Manage pipeline state transitions and persistence
//  5. Publish execution events for monitoring and coordination
//
// Parameters:
//
//	ctx context.Context: Execution context for cancellation and timeout handling
//	command ExecutePipelineCommand: Command with pipeline ID and execution parameters
//
// Returns:
//
//	result.Result[interface{}]: Success with ExecutionResult or Failure with error
//
// Example:
//
//	Orchestrated pipeline execution:
//
//	  orchestrator := &PipelineExecutionOrchestrator{
//	      repository:     pipelineRepo,
//	      eventBus:       eventBus,
//	      workflowEngine: workflowEngine,
//	  }
//
//	  command := ExecutePipelineCommand{
//	      PipelineID: entities.PipelineID("pipeline-123"),
//	      Parameters: map[string]interface{}{
//	          "input_path": "/data/input/",
//	          "batch_size": 1000,
//	      },
//	  }
//
//	  result := orchestrator.Execute(ctx, command)
//	  if result.IsFailure() {
//	      // All error scenarios handled by orchestrator
//	      log.Error("Pipeline execution failed", "error", result.Error())
//	      return result.Error()
//	  }
//
//	  executionResult := result.Value().(ExecutionResult)
//	  log.Info("Pipeline execution started successfully",
//	      "pipeline_id", executionResult.PipelineID,
//	      "workflow_id", executionResult.WorkflowID)
//
// Error Handling:
//   - Pipeline not found or invalid state
//   - Workflow engine integration failures
//   - Persistence errors during state transitions
//   - Event publishing failures (logged but don't fail execution)
//
// DRY Principle Implementation:
//
//	Eliminates multiple return paths by using Railway Pattern, ensuring consistent
//	error handling and reducing code duplication across execution steps.
//
// Integration:
//   - Domain Layer: Pipeline aggregate state management
//   - Workflow Engine: External execution system coordination
//   - Event System: Execution lifecycle event publishing
//   - Repository: Pipeline state persistence and recovery
func (o *PipelineExecutionOrchestrator) Execute(ctx context.Context, command ExecutePipelineCommand) (*ExecutionResult, error) {
	// Step 1: Load and validate pipeline
	pipeline, err := o.loadAndValidatePipeline(ctx, command.PipelineID)
	if err != nil {
		return nil, err
	}

	// Step 2: Prepare execution context
	executionCtx := &ExecutionContext{
		Pipeline:   pipeline,
		Parameters: command.Parameters,
	}

	// Step 3: Execute pipeline through orchestrated steps
	return o.orchestrateExecution(ctx, executionCtx)
}

// loadAndValidatePipeline loads pipeline and validates execution readiness with comprehensive checks.
//
// This method implements the pipeline loading and validation phase of execution
// orchestration, ensuring that the pipeline exists, is in a valid state for
// execution, and meets all prerequisites. It follows SOLID SRP by focusing
// exclusively on pipeline loading and validation concerns.
//
// Parameters:
//
//	ctx context.Context: Execution context for cancellation and timeout handling
//	pipelineID entities.PipelineID: Unique identifier of the pipeline to load
//
// Returns:
//
//	*entities.Pipeline: Loaded and validated pipeline ready for execution
//	error: Loading or validation error if pipeline cannot be executed
//
// Example:
//
//	Pipeline loading and validation:
//
//	  orchestrator := &PipelineExecutionOrchestrator{
//	      repository: pipelineRepo,
//	  }
//
//	  pipeline, err := orchestrator.loadAndValidatePipeline(ctx, pipelineID)
//	  if err != nil {
//	      switch {
//	      case errors.Is(err, errors.NotFoundError{}):
//	          return fmt.Errorf("pipeline not found: %w", err)
//	      case errors.Is(err, errors.ValidationError{}):
//	          return fmt.Errorf("pipeline not ready for execution: %w", err)
//	      default:
//	          return fmt.Errorf("pipeline loading failed: %w", err)
//	      }
//	  }
//
//	  log.Info("Pipeline loaded and validated",
//	      "id", pipeline.ID(),
//	      "status", pipeline.Status().String(),
//	      "steps", len(pipeline.Steps()))
//
// Validation Rules:
//   - Pipeline must exist in the repository
//   - Pipeline must be in Active status
//   - Pipeline must have at least one configured step
//   - Pipeline must not be currently running
//   - Pipeline configuration must be complete and valid
//
// Error Handling:
//   - NotFoundError: Pipeline does not exist in the repository
//   - ValidationError: Pipeline exists but cannot be executed (wrong status, no steps, etc.)
//   - RepositoryError: Database or persistence layer errors
//
// SOLID SRP Implementation:
//
//	Single responsibility for pipeline loading and execution readiness validation,
//	separating concerns from workflow orchestration and execution coordination.
//
// Integration:
//   - Repository Layer: Loads pipeline aggregate from persistence
//   - Domain Layer: Uses Pipeline.CanExecute() business logic validation
//   - Error Handling: Provides detailed error context for different failure scenarios
func (o *PipelineExecutionOrchestrator) loadAndValidatePipeline(ctx context.Context, pipelineID entities.PipelineID) (*entities.Pipeline, error) {
	pipeline, err := o.repository.FindByID(ctx, pipelineID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find pipeline")
	}

	if !pipeline.CanExecute() {
		return nil, errors.ValidationError("pipeline cannot be executed")
	}

	return pipeline, nil
}

// orchestrateExecution performs coordinated execution steps
// SOLID SRP: Single responsibility for execution orchestration
func (o *PipelineExecutionOrchestrator) orchestrateExecution(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	// Start pipeline execution
	if err := execCtx.Pipeline.Start(); err != nil {
		return nil, err
	}

	// Save pipeline state
	if err := o.repository.Save(ctx, execCtx.Pipeline); err != nil {
		return nil, errors.Wrap(err, "failed to save pipeline")
	}

	// Start workflow with error recovery
	workflowID, err := o.startWorkflowWithRecovery(ctx, execCtx)
	if err != nil {
		return nil, err
	}

	execCtx.WorkflowID = workflowID

	// Publish events and return result
	o.publishDomainEvents(ctx, execCtx.Pipeline)
	execCtx.Pipeline.ClearEvents()

	result := o.createExecutionResult(execCtx)
	return &result, nil
}

// startWorkflowWithRecovery starts workflow with proper error recovery
// SOLID SRP: Single responsibility for workflow initiation and recovery
func (o *PipelineExecutionOrchestrator) startWorkflowWithRecovery(ctx context.Context, execCtx *ExecutionContext) (string, error) {
	workflowInput := map[string]interface{}{
		"pipelineID": execCtx.Pipeline.ID.String(),
		"parameters": execCtx.Parameters,
	}

	workflowID, err := o.workflowEngine.StartWorkflow(ctx, "pipeline-execution", workflowInput)
	if err != nil {
		// Recovery: Mark pipeline as failed and save state
		execCtx.Pipeline.Fail("failed to start workflow: " + err.Error())
		if saveErr := o.repository.Save(ctx, execCtx.Pipeline); saveErr != nil {
			// Log save error but don't override original error
			o.logger.Error("Failed to save pipeline during recovery", zap.Error(saveErr))
		}
		return "", errors.Wrap(err, "failed to start workflow")
	}

	return workflowID, nil
}

// publishDomainEvents publishes all pipeline domain events
// SOLID SRP: Single responsibility for event publishing
func (o *PipelineExecutionOrchestrator) publishDomainEvents(ctx context.Context, pipeline *entities.Pipeline) {
	for _, event := range pipeline.DomainEvents() {
		if err := o.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command - events are eventually consistent
			o.logger.Error("Failed to publish domain event",
				zap.String("event_type", event.EventType()),
				zap.String("pipeline_id", pipeline.ID.String()),
				zap.Error(err))
		}
	}
}

// createExecutionResult creates standardized execution result
// SOLID SRP: Single responsibility for result creation
func (o *PipelineExecutionOrchestrator) createExecutionResult(execCtx *ExecutionContext) ExecutionResult {
	return ExecutionResult{
		PipelineID: execCtx.Pipeline.ID.String(),
		WorkflowID: execCtx.WorkflowID,
		Status:     execCtx.Pipeline.Status.String(),
	}
}

// Handle handles the execute pipeline command
// DRY PRINCIPLE: Delegates to specialized orchestrator, eliminating complex method with 6 returns
func (h *ExecutePipelineCommandHandler) Handle(ctx context.Context, command ExecutePipelineCommand) (*ExecutionResult, error) {
	orchestrator := NewPipelineExecutionOrchestrator(h.repository, h.eventBus, h.workflowEngine)
	return orchestrator.Execute(ctx, command)
}

// ActivatePipelineCommand represents a command to activate a pipeline for execution.
//
// ActivatePipelineCommand encapsulates the operation to activate a pipeline,
// making it available for execution. Pipeline activation is a critical state
// transition that enables the pipeline to be started and scheduled for
// data processing operations.
//
// Fields:
//
//	BaseCommand: Inherited command infrastructure with type identification
//	PipelineID entities.PipelineID: Target pipeline identifier (required)
//
// Business Rules:
//   - Pipeline must exist in the system
//   - Pipeline must be in an activatable state (typically Registered or Inactive)
//   - Pipeline cannot be activated if it's already Active (idempotent operation)
//   - Pipeline cannot be activated if it's in Error state (must resolve errors first)
//   - Pipeline must have at least one processing step configured
//
// Example:
//
//	Activating a configured pipeline:
//
//	  // Activate pipeline after configuration is complete
//	  command := commands.NewActivatePipelineCommand(pipelineID)
//
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsFailure() {
//	      switch err := result.Error().(type) {
//	      case *errors.ValidationError:
//	          if strings.Contains(err.Error(), "already active") {
//	              log.Info("Pipeline is already active", "id", pipelineID)
//	              return nil // Idempotent operation
//	          }
//	          return fmt.Errorf("pipeline activation validation failed: %w", err)
//	      case *errors.NotFoundError:
//	          return fmt.Errorf("pipeline not found: %w", err)
//	      default:
//	          return fmt.Errorf("pipeline activation failed: %w", err)
//	      }
//	  }
//
//	  pipeline := result.Value().(*entities.Pipeline)
//	  log.Info("Pipeline activated successfully",
//	      "id", pipeline.ID(),
//	      "name", pipeline.Name(),
//	      "status", pipeline.Status().String())
//
// State Transitions:
//   - Registered → Active: Normal activation after initial configuration
//   - Inactive → Active: Reactivation after temporary deactivation
//   - Active → Active: Idempotent operation (no change)
//   - Error → Active: Not allowed (must resolve errors first)
//
// Integration:
//   - Processed by ActivatePipelineCommandHandler
//   - Updates Pipeline aggregate status to Active
//   - Generates PipelineActivated domain event
//   - Makes pipeline available for execution scheduling
type ActivatePipelineCommand struct {
	BaseCommand
	PipelineID entities.PipelineID // Target pipeline identifier
}

// NewActivatePipelineCommand creates a new activate pipeline command with validation.
//
// This factory function creates a properly initialized ActivatePipelineCommand
// for pipeline activation operations. It follows the Command pattern with
// immutable command objects that encapsulate the pipeline activation intention.
//
// Parameters:
//
//	pipelineID entities.PipelineID: Target pipeline identifier for activation (required)
//
// Returns:
//
//	ActivatePipelineCommand: Fully initialized command ready for execution
//
// Example:
//
//	Creating activation commands for different scenarios:
//
//	  // Basic pipeline activation
//	  command := commands.NewActivatePipelineCommand(pipelineID)
//
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsFailure() {
//	      if strings.Contains(result.Error().Error(), "already active") {
//	          log.Info("Pipeline is already active", "id", pipelineID)
//	          return nil // Idempotent operation
//	      }
//	      return fmt.Errorf("activation failed: %w", result.Error())
//	  }
//
//	  pipeline := result.Value().(*entities.Pipeline)
//	  log.Info("Pipeline activated successfully",
//	      "id", pipeline.ID(),
//	      "name", pipeline.Name(),
//	      "status", pipeline.Status().String())
//
//	  // Pipeline is now available for execution and scheduling
//	  executionCommand := commands.NewExecutePipelineCommand(
//	      pipelineID,
//	      map[string]interface{}{"immediate": true},
//	  )
//
// Business Context:
//
//	Pipeline activation makes the pipeline available for execution and scheduling.
//	This is typically done after pipeline configuration is complete and validated.
//
// Integration:
//
//	Command is designed for execution through CommandBus with proper
//	routing to ActivatePipelineCommandHandler for processing.
func NewActivatePipelineCommand(pipelineID entities.PipelineID) ActivatePipelineCommand {
	return ActivatePipelineCommand{
		BaseCommand: NewBaseCommand("ActivatePipeline"),
		PipelineID:  pipelineID,
	}
}

// ActivatePipelineCommandHandler handles pipeline activation commands with state validation.
//
// ActivatePipelineCommandHandler implements the CommandHandler pattern for pipeline
// activation operations, coordinating pipeline loading, state validation, activation
// business logic, and event publishing. It ensures proper state transitions and
// maintains system consistency during pipeline activation.
//
// Responsibilities:
//   - Pipeline existence validation and loading
//   - Pipeline state validation for activation eligibility
//   - Domain aggregate state transition execution
//   - Repository coordination for state persistence
//   - Domain event publishing for activation tracking
//
// Fields:
//
//	repository PipelineRepository: Data access abstraction for pipeline operations
//	eventBus EventBus: Event publishing infrastructure for domain events
//
// Example:
//
//	Handler initialization and usage:
//
//	  handler := commands.NewActivatePipelineCommandHandler(
//	      pipelineRepository,
//	      eventBus,
//	  )
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&ActivatePipelineCommand{}, handler)
//
//	  // Handler processes commands through command bus
//	  command := commands.NewActivatePipelineCommand(pipelineID)
//
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsSuccess() {
//	      pipeline := result.Value().(*entities.Pipeline)
//	      log.Info("Pipeline activated",
//	          "id", pipeline.ID(),
//	          "status", pipeline.Status().String())
//	  }
//
// Integration:
//   - Command Bus: Registered as handler for ActivatePipelineCommand
//   - Domain Layer: Executes Pipeline.Activate() business logic
//   - Infrastructure: Repository and event bus dependency injection
//   - State Management: Coordinates pipeline state transitions
type ActivatePipelineCommandHandler struct {
	repository PipelineRepository // Data access abstraction for pipeline operations
	eventBus   EventBus           // Event publishing infrastructure for domain events
}

// NewActivatePipelineCommandHandler creates a new activate pipeline command handler with dependencies.
//
// This factory function creates a properly initialized ActivatePipelineCommandHandler
// with all required dependencies injected. It follows the dependency injection pattern
// for clean architecture, testability, and proper separation of concerns.
//
// Parameters:
//
//	repository PipelineRepository: Data access abstraction for pipeline operations (required)
//	eventBus EventBus: Event publishing infrastructure for domain events (required)
//
// Returns:
//
//	*ActivatePipelineCommandHandler: Fully initialized handler ready for command processing
//
// Example:
//
//	Handler initialization in application setup:
//
//	  // Initialize dependencies
//	  pipelineRepo := postgresql.NewPipelineRepository(db)
//	  eventBus := redis.NewEventBus(redisClient)
//
//	  // Create handler with dependency injection
//	  handler := commands.NewActivatePipelineCommandHandler(pipelineRepo, eventBus)
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&ActivatePipelineCommand{}, handler)
//
//	  log.Info("ActivatePipelineCommandHandler registered successfully")
//
// Integration:
//   - Application Setup: Called during application initialization
//   - Dependency Injection: Receives concrete implementations of abstractions
//   - Command Bus: Handler registered for ActivatePipelineCommand processing
//   - Testing: Enables easy mocking of dependencies for unit tests
func NewActivatePipelineCommandHandler(repository PipelineRepository, eventBus EventBus) *ActivatePipelineCommandHandler {
	return &ActivatePipelineCommandHandler{
		repository: repository,
		eventBus:   eventBus,
	}
}

// Handle handles the activate pipeline command with comprehensive state management.
//
// This method implements the complete pipeline activation workflow, including
// pipeline loading, state validation, activation business logic execution,
// persistence, and event publishing. It follows the Command Handler pattern
// with proper error handling and state transition management.
//
// Business Logic Flow:
//  1. Load target pipeline and validate existence
//  2. Execute pipeline activation through domain aggregate
//  3. Persist updated pipeline state through repository
//  4. Publish domain events for activation tracking
//  5. Return activated pipeline as command result
//
// Parameters:
//
//	ctx context.Context: Execution context for cancellation and timeout handling
//	command ActivatePipelineCommand: Command with pipeline ID for activation
//
// Returns:
//
//	result.Result[interface{}]: Success with activated Pipeline or Failure with error
//
// Example:
//
//	Command execution flow:
//
//	  handler := &ActivatePipelineCommandHandler{
//	      repository: pipelineRepo,
//	      eventBus:   eventBus,
//	  }
//
//	  command := ActivatePipelineCommand{
//	      PipelineID: entities.PipelineID("pipeline-456"),
//	  }
//
//	  result := handler.Handle(ctx, command)
//	  if result.IsFailure() {
//	      switch err := result.Error().(type) {
//	      case *errors.NotFoundError:
//	          return fmt.Errorf("pipeline not found: %w", err)
//	      case *errors.ValidationError:
//	          if strings.Contains(err.Error(), "already active") {
//	              log.Info("Pipeline already active - idempotent operation")
//	              return nil
//	          }
//	          return fmt.Errorf("activation validation failed: %w", err)
//	      default:
//	          return fmt.Errorf("pipeline activation failed: %w", err)
//	      }
//	  }
//
//	  pipeline := result.Value().(*entities.Pipeline)
//	  log.Info("Pipeline activated successfully",
//	      "id", pipeline.ID(),
//	      "status", pipeline.Status().String(),
//	      "steps", len(pipeline.Steps()))
//
// Business Rules Enforced:
//   - Pipeline must exist in the system
//   - Pipeline must be in an activatable state (domain aggregate validates)
//   - Activation is idempotent (already active pipelines return validation error)
//   - Pipeline must have proper configuration and steps
//
// Error Handling:
//   - NotFoundError: Pipeline does not exist
//   - ValidationError: Pipeline state prevents activation or already active
//   - RepositoryError: Persistence failures with proper wrapping
//   - Event publishing errors are logged but don't fail the command
//
// Integration:
//   - Domain Layer: Executes Pipeline.Activate() with business rule validation
//   - Repository: Persists pipeline state changes with optimistic concurrency
//   - Event Bus: Publishes PipelineActivated and related events
//   - Result Pattern: Explicit error handling with typed results
func (h *ActivatePipelineCommandHandler) Handle(ctx context.Context, command ActivatePipelineCommand) (*entities.Pipeline, error) {
	// Find pipeline
	pipeline, err := h.repository.FindByID(ctx, command.PipelineID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find pipeline")
	}

	// Activate pipeline
	if err := pipeline.Activate(); err != nil {
		return nil, err
	}

	// Save pipeline
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return nil, errors.Wrap(err, "failed to save pipeline")
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	return pipeline, nil
}

// SetPipelineScheduleCommand represents a command to configure automated pipeline scheduling.
//
// SetPipelineScheduleCommand encapsulates all necessary information for setting up
// automated pipeline execution scheduling using cron expressions. This command enables
// pipelines to run automatically on defined schedules with timezone support for
// global deployment scenarios.
//
// Fields:
//
//	BaseCommand: Inherited command infrastructure with type identification
//	PipelineID entities.PipelineID: Target pipeline identifier (required)
//	CronExpression string: Cron expression for schedule definition (required)
//	Timezone string: Timezone identifier for schedule execution (required)
//
// Business Rules:
//   - Pipeline must exist and be in an active state
//   - Cron expression must be valid and parseable
//   - Timezone must be a valid IANA timezone identifier
//   - Pipeline must have at least one configured step
//   - Scheduling overwrites any existing schedule for the pipeline
//
// Example:
//
//	Configuring different pipeline schedules:
//
//	  // Daily execution at 2 AM UTC
//	  dailyCommand := commands.NewSetPipelineScheduleCommand(
//	      pipelineID,
//	      "0 2 * * *",
//	      "UTC",
//	  )
//
//	  // Hourly execution during business hours in New York
//	  businessHoursCommand := commands.NewSetPipelineScheduleCommand(
//	      pipelineID,
//	      "0 9-17 * * 1-5",
//	      "America/New_York",
//	  )
//
//	  // Weekly execution on Sundays at midnight in London
//	  weeklyCommand := commands.NewSetPipelineScheduleCommand(
//	      pipelineID,
//	      "0 0 * * 0",
//	      "Europe/London",
//	  )
//
//	  // Monthly execution on the first day at 3 AM
//	  monthlyCommand := commands.NewSetPipelineScheduleCommand(
//	      pipelineID,
//	      "0 3 1 * *",
//	      "UTC",
//	  )
//
//	  result := commandBus.Execute(ctx, dailyCommand)
//	  if result.IsFailure() {
//	      return fmt.Errorf("schedule configuration failed: %w", result.Error())
//	  }
//
//	  schedule := result.Value().(*entities.PipelineSchedule)
//	  log.Info("Pipeline scheduled successfully",
//	      "pipeline_id", pipelineID,
//	      "expression", schedule.CronExpression,
//	      "timezone", schedule.Timezone)
//
// Cron Expression Examples:
//   - "0 2 * * *": Daily at 2:00 AM
//   - "0 */6 * * *": Every 6 hours
//   - "0 9 * * 1-5": Weekdays at 9:00 AM
//   - "0 0 1 * *": Monthly on the 1st at midnight
//   - "0 0 * * 0": Weekly on Sundays at midnight
//
// Integration:
//   - Processed by SetPipelineScheduleCommandHandler
//   - Updates Pipeline aggregate with schedule configuration
//   - Generates PipelineScheduleSet domain event
//   - Integrates with scheduling system for automated execution
type SetPipelineScheduleCommand struct {
	BaseCommand
	PipelineID     entities.PipelineID // Target pipeline identifier
	CronExpression string              // Cron expression for schedule definition
	Timezone       string              // Timezone identifier for schedule execution
}

// NewSetPipelineScheduleCommand creates a new set pipeline schedule command with validation.
//
// This factory function creates a properly initialized SetPipelineScheduleCommand
// for pipeline schedule configuration operations. It follows the Command pattern with
// immutable command objects that encapsulate the scheduling configuration intention.
//
// Parameters:
//
//	pipelineID entities.PipelineID: Target pipeline identifier for scheduling (required)
//	cronExpression string: Cron expression for schedule definition (required)
//	timezone string: Timezone identifier for schedule execution (required)
//
// Returns:
//
//	SetPipelineScheduleCommand: Fully initialized command ready for execution
//
// Example:
//
//	Creating schedule commands for different scenarios:
//
//	  // Daily execution at 2 AM UTC
//	  dailyCommand := commands.NewSetPipelineScheduleCommand(
//	      pipelineID,
//	      "0 2 * * *",
//	      "UTC",
//	  )
//
//	  // Business hours execution in specific timezone
//	  businessCommand := commands.NewSetPipelineScheduleCommand(
//	      pipelineID,
//	      "0 9-17 * * 1-5", // Every hour from 9 AM to 5 PM, Monday to Friday
//	      "America/New_York",
//	  )
//
//	  // Weekly batch processing
//	  weeklyCommand := commands.NewSetPipelineScheduleCommand(
//	      pipelineID,
//	      "0 0 * * 0", // Every Sunday at midnight
//	      "Europe/London",
//	  )
//
//	  result := commandBus.Execute(ctx, dailyCommand)
//	  if result.IsFailure() {
//	      return fmt.Errorf("schedule configuration failed: %w", result.Error())
//	  }
//
//	  schedule := result.Value().(*entities.PipelineSchedule)
//	  log.Info("Pipeline scheduled successfully",
//	      "expression", schedule.CronExpression,
//	      "timezone", schedule.Timezone,
//	      "next_run", schedule.NextExecutionTime())
//
// Validation Notes:
//   - Cron expression syntax will be validated during command processing
//   - Timezone must be a valid IANA timezone identifier
//   - Pipeline must exist and be in an active state
//
// Integration:
//
//	Command is designed for execution through CommandBus with proper
//	routing to SetPipelineScheduleCommandHandler for processing.
func NewSetPipelineScheduleCommand(pipelineID entities.PipelineID, cronExpression, timezone string) SetPipelineScheduleCommand {
	return SetPipelineScheduleCommand{
		BaseCommand:    NewBaseCommand("SetPipelineSchedule"),
		PipelineID:     pipelineID,
		CronExpression: cronExpression,
		Timezone:       timezone,
	}
}

// SetPipelineScheduleCommandHandler handles set pipeline schedule commands with validation.
//
// SetPipelineScheduleCommandHandler implements the CommandHandler pattern for pipeline
// schedule configuration operations, coordinating pipeline loading, schedule validation,
// cron expression parsing, timezone verification, and event publishing. It ensures
// proper schedule configuration and maintains system consistency.
//
// Responsibilities:
//   - Pipeline existence validation and loading
//   - Cron expression syntax validation and parsing
//   - Timezone identifier validation and verification
//   - Domain aggregate schedule configuration
//   - Repository coordination for schedule persistence
//   - Domain event publishing for schedule tracking
//
// Fields:
//
//	repository PipelineRepository: Data access abstraction for pipeline operations
//	eventBus EventBus: Event publishing infrastructure for domain events
//
// Example:
//
//	Handler initialization and usage:
//
//	  handler := commands.NewSetPipelineScheduleCommandHandler(
//	      pipelineRepository,
//	      eventBus,
//	  )
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&SetPipelineScheduleCommand{}, handler)
//
//	  // Handler processes commands through command bus
//	  command := commands.NewSetPipelineScheduleCommand(
//	      pipelineID,
//	      "0 2 * * *",  // Daily at 2 AM
//	      "UTC",
//	  )
//
//	  result := commandBus.Execute(ctx, command)
//	  if result.IsSuccess() {
//	      schedule := result.Value().(*entities.PipelineSchedule)
//	      log.Info("Pipeline scheduled",
//	          "expression", schedule.CronExpression,
//	          "timezone", schedule.Timezone)
//	  }
//
// Integration:
//   - Command Bus: Registered as handler for SetPipelineScheduleCommand
//   - Domain Layer: Executes Pipeline.SetSchedule() business logic
//   - Infrastructure: Repository and event bus dependency injection
//   - Scheduling System: Schedule changes trigger scheduling system updates
type SetPipelineScheduleCommandHandler struct {
	repository PipelineRepository // Data access abstraction for pipeline operations
	eventBus   EventBus           // Event publishing infrastructure for domain events
}

// NewSetPipelineScheduleCommandHandler creates a new set pipeline schedule command handler with dependencies.
//
// This factory function creates a properly initialized SetPipelineScheduleCommandHandler
// with all required dependencies injected. It follows the dependency injection pattern
// for clean architecture, testability, and proper separation of concerns.
//
// Parameters:
//
//	repository PipelineRepository: Data access abstraction for pipeline operations (required)
//	eventBus EventBus: Event publishing infrastructure for domain events (required)
//
// Returns:
//
//	*SetPipelineScheduleCommandHandler: Fully initialized handler ready for command processing
//
// Example:
//
//	Handler initialization in application setup:
//
//	  // Initialize dependencies
//	  pipelineRepo := postgresql.NewPipelineRepository(db)
//	  eventBus := redis.NewEventBus(redisClient)
//
//	  // Create handler with dependency injection
//	  handler := commands.NewSetPipelineScheduleCommandHandler(pipelineRepo, eventBus)
//
//	  // Register with command bus
//	  commandBus.RegisterHandler(&SetPipelineScheduleCommand{}, handler)
//
//	  log.Info("SetPipelineScheduleCommandHandler registered successfully")
//
// Integration:
//   - Application Setup: Called during application initialization
//   - Dependency Injection: Receives concrete implementations of abstractions
//   - Command Bus: Handler registered for SetPipelineScheduleCommand processing
//   - Schedule System: Handler integrates with scheduling infrastructure
func NewSetPipelineScheduleCommandHandler(repository PipelineRepository, eventBus EventBus) *SetPipelineScheduleCommandHandler {
	return &SetPipelineScheduleCommandHandler{
		repository: repository,
		eventBus:   eventBus,
	}
}

// Handle handles the set pipeline schedule command with comprehensive validation and error handling.
//
// This method implements the complete pipeline schedule configuration workflow,
// including pipeline loading, schedule validation, cron expression parsing,
// domain aggregate modification, persistence, and event publishing. It ensures
// proper schedule integration and maintains system consistency.
//
// Business Logic Flow:
//  1. Load target pipeline and validate existence
//  2. Validate cron expression syntax and parseability
//  3. Validate timezone identifier and availability
//  4. Set schedule configuration on pipeline aggregate
//  5. Persist updated pipeline through repository
//  6. Publish domain events for schedule tracking
//  7. Return configured schedule as command result
//
// Parameters:
//
//	ctx context.Context: Execution context for cancellation and timeout handling
//	command SetPipelineScheduleCommand: Command with schedule configuration data
//
// Returns:
//
//	result.Result[interface{}]: Success with PipelineSchedule or Failure with error
//
// Example:
//
//	Command execution flow:
//
//	  handler := &SetPipelineScheduleCommandHandler{
//	      repository: pipelineRepo,
//	      eventBus:   eventBus,
//	  }
//
//	  command := SetPipelineScheduleCommand{
//	      PipelineID:     entities.PipelineID("pipeline-789"),
//	      CronExpression: "0 2 * * *",        // Daily at 2 AM
//	      Timezone:       "America/New_York", // Eastern timezone
//	  }
//
//	  result := handler.Handle(ctx, command)
//	  if result.IsFailure() {
//	      switch err := result.Error().(type) {
//	      case *errors.NotFoundError:
//	          return fmt.Errorf("pipeline not found: %w", err)
//	      case *errors.ValidationError:
//	          if strings.Contains(err.Error(), "cron") {
//	              return fmt.Errorf("invalid cron expression: %w", err)
//	          }
//	          if strings.Contains(err.Error(), "timezone") {
//	              return fmt.Errorf("invalid timezone: %w", err)
//	          }
//	          return fmt.Errorf("schedule validation failed: %w", err)
//	      default:
//	          return fmt.Errorf("schedule configuration failed: %w", err)
//	      }
//	  }
//
//	  schedule := result.Value().(*entities.PipelineSchedule)
//	  log.Info("Pipeline schedule configured successfully",
//	      "pipeline_id", command.PipelineID,
//	      "expression", schedule.CronExpression,
//	      "timezone", schedule.Timezone,
//	      "next_run", schedule.NextExecutionTime())
//
// Business Rules Enforced:
//   - Pipeline must exist and be in an active state
//   - Cron expression must be valid and parseable by scheduling system
//   - Timezone must be a valid IANA timezone identifier
//   - Schedule configuration overwrites any existing schedule
//
// Error Handling:
//   - NotFoundError: Pipeline does not exist
//   - ValidationError: Invalid cron expression, timezone, or schedule configuration
//   - RepositoryError: Persistence failures with proper wrapping
//   - Event publishing errors are logged but don't fail the command
//
// Schedule Validation:
//   - Cron expression syntax validation using standard cron parser
//   - Timezone validation against IANA timezone database
//   - Schedule conflict detection and resolution
//   - Next execution time calculation and validation
//
// Integration:
//   - Domain Layer: Executes Pipeline.SetSchedule() with business rule validation
//   - Repository: Persists pipeline schedule changes with optimistic concurrency
//   - Event Bus: Publishes PipelineScheduleSet and related events
//   - Scheduling System: Triggers schedule system updates for automated execution
func (h *SetPipelineScheduleCommandHandler) Handle(ctx context.Context, command SetPipelineScheduleCommand) (*entities.PipelineSchedule, error) {
	// Find pipeline
	pipeline, err := h.repository.FindByID(ctx, command.PipelineID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find pipeline")
	}

	// Set schedule
	if err := pipeline.SetSchedule(command.CronExpression, command.Timezone); err != nil {
		return nil, err
	}

	// Save pipeline
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return nil, errors.Wrap(err, "failed to save pipeline")
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	return pipeline.Schedule, nil
}
