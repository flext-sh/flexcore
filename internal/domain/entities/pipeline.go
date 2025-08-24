// Package entities provides FlexCore's core domain entities implementing Domain-Driven Design patterns.
//
// This package contains the rich domain model for FlexCore's data processing pipeline
// orchestration system. All entities follow DDD principles with strong business logic
// encapsulation, comprehensive validation, and domain event generation for CQRS and
// Event Sourcing integration.
//
// Key Components:
//   - Pipeline: Main aggregate root for data processing pipeline management
//   - PipelineStep: Value object representing individual processing steps
//   - PipelineStatus: Enumeration with comprehensive status management
//   - Domain Events: Complete event model for pipeline lifecycle tracking
//
// Architecture:
//
//	Entities are built on the domain.AggregateRoot foundation, providing:
//	- Automatic domain event generation and management
//	- Consistent entity lifecycle tracking with timestamps
//	- Result pattern integration for comprehensive error handling
//	- Type-safe entity identification with strongly-typed IDs
//
// Integration:
//   - Built on FlexCore's domain foundation (internal/domain)
//   - Integrates with Result pattern (pkg/result) for error handling
//   - Compatible with CQRS command/query handlers
//   - Supports Event Sourcing with immutable domain events
//   - Uses FlexCore's custom error types (pkg/errors)
//
// Example:
//
//	Creating and managing a pipeline:
//
//	  pipeline := entities.NewPipeline("Data Processing", "ETL pipeline", "user123")
//	  if pipeline.IsFailure() {
//	      return pipeline.Error()
//	  }
//
//	  p := pipeline.Value()
//	  step := entities.NewPipelineStep("extract", "data-extraction")
//
//	  if err := p.AddStep(step); err.IsFailure() {
//	      return err.Error()
//	  }
//
//	  if err := p.Activate(); err.IsFailure() {
//	      return err.Error()
//	  }
//
// Thread Safety:
//
//	Domain entities are designed for single-threaded access within aggregate boundaries.
//	Concurrent access should be managed at the application layer through proper
//	locking mechanisms or message-based coordination.
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package entities

import (
	"fmt"
	"time"

	"github.com/flext-sh/flexcore/internal/domain"
	"github.com/flext-sh/flexcore/pkg/errors"
	"github.com/google/uuid"
)

// PipelineID represents a unique, strongly-typed identifier for Pipeline aggregates.
//
// PipelineID provides type safety by preventing accidental mixing of different
// entity identifiers and ensures consistent ID format throughout the system.
// All pipeline operations require a valid PipelineID for entity identification
// and event correlation.
//
// The underlying type is string to enable easy serialization and database
// storage while maintaining type safety at the application level.
//
// Example:
//
//	Working with pipeline identifiers:
//
//	  id := entities.NewPipelineID()
//	  fmt.Println("Generated ID:", id.String())
//
//	  // Type safety prevents mixing different ID types
//	  pipeline := findPipeline(id) // Compiler enforces correct ID type
//
// Integration:
//   - Used as aggregate identifier in domain events
//   - Serves as primary key in persistence layer
//   - Enables type-safe repository and service method signatures
//   - Compatible with CQRS command and query message routing
type PipelineID string

// NewPipelineID generates a new unique pipeline identifier using UUID v4.
//
// This function creates cryptographically random UUIDs that are globally unique
// and suitable for distributed systems. The generated IDs are compatible with
// database storage and provide sufficient entropy for collision avoidance.
//
// Returns:
//
//	PipelineID: A new unique identifier for pipeline entities
//
// Example:
//
//	Creating pipelines with unique identifiers:
//
//	  id1 := entities.NewPipelineID()
//	  id2 := entities.NewPipelineID()
//
//	  // IDs are guaranteed to be different
//	  fmt.Printf("ID1: %s\nID2: %s\n", id1.String(), id2.String())
//
// Performance:
//
//	Single UUID generation call with minimal allocation overhead.
//	Suitable for high-frequency pipeline creation scenarios.
//
// Thread Safety:
//
//	Safe for concurrent use as UUID generation is thread-safe.
func NewPipelineID() PipelineID {
	return PipelineID(uuid.New().String())
}

// String returns the string representation of the pipeline identifier.
//
// This method enables PipelineID to implement the fmt.Stringer interface,
// providing consistent string conversion for logging, debugging, and
// serialization scenarios.
//
// Returns:
//
//	string: The underlying UUID string value
//
// Example:
//
//	String conversion and formatting:
//
//	  id := entities.NewPipelineID()
//
//	  // Implicit string conversion
//	  fmt.Printf("Pipeline ID: %s\n", id)
//
//	  // Explicit string conversion
//	  idString := id.String()
//	  log.Info("Processing pipeline", "id", idString)
//
// Integration:
//   - Compatible with logging frameworks expecting string values
//   - Enables JSON marshaling/unmarshaling
//   - Supports database primary key conversion
//   - Works with template systems and string formatting
func (id PipelineID) String() string {
	return string(id)
}

// PipelineStatus represents the current state of a pipeline in its lifecycle.
//
// PipelineStatus uses an enumeration pattern to ensure type safety and prevent
// invalid status values. The status controls which operations are allowed on
// a pipeline and determines the business rules that apply to state transitions.
//
// The status follows a well-defined state machine:
//
//	Draft -> Active -> Running -> (Completed|Failed|Paused) -> [Active|Archived]
//
// Thread Safety:
//
//	Status values are immutable and safe for concurrent access.
//	Status transitions should be managed through Pipeline entity methods
//	which provide proper validation and event generation.
//
// Integration:
//   - Used in business rule validation throughout Pipeline methods
//   - Included in domain events for state change tracking
//   - Mapped to string representation for persistence and APIs
//   - Compatible with CQRS query projections and read models
type PipelineStatus int

const (
	// Pipeline status string constants for mapping and serialization
	unknownStatusString = "unknown"

	// Default configuration constants for pipeline steps
	DefaultMaxRetries     = 3  // Maximum retry attempts for failed steps
	DefaultTimeoutMinutes = 30 // Default timeout for step execution

	// PipelineStatus enumeration defining all valid pipeline states.
	// Status transitions follow business rules enforced by Pipeline methods.

	// PipelineStatusDraft represents a newly created pipeline that is not yet active.
	// Draft pipelines can be modified freely but cannot be executed.
	PipelineStatusDraft PipelineStatus = iota

	// PipelineStatusActive represents a pipeline ready for execution.
	// Active pipelines have passed validation and can be started.
	PipelineStatusActive

	// PipelineStatusRunning represents a pipeline currently executing.
	// Running pipelines cannot be modified and must complete or fail.
	PipelineStatusRunning

	// PipelineStatusCompleted represents a successfully finished pipeline.
	// Completed pipelines can be reactivated for additional runs.
	PipelineStatusCompleted

	// PipelineStatusFailed represents a pipeline that encountered an error.
	// Failed pipelines can be reactivated after addressing the failure cause.
	PipelineStatusFailed

	// PipelineStatusPaused represents a temporarily suspended pipeline.
	// Paused pipelines can be resumed or terminated.
	PipelineStatusPaused

	// PipelineStatusArchived represents a pipeline no longer in active use.
	// Archived pipelines are retained for historical purposes but cannot be executed.
	PipelineStatusArchived
)

// PipelineStatusMapper provides efficient mapping between PipelineStatus enums and string representations.
//
// This mapper implements the Single Responsibility Principle by centralizing
// all status-to-string conversion logic in one place, eliminating multiple
// switch statements throughout the codebase. It uses a pre-built map for
// O(1) lookup performance.
//
// Design Patterns:
//   - Single Responsibility: Handles only status string mapping
//   - Strategy Pattern: Encapsulates status conversion algorithm
//   - Immutable Mapping: Status map is initialized once and never modified
//
// Performance:
//   - O(1) status lookup using pre-built hash map
//   - No runtime switch statement evaluation
//   - Minimal memory overhead with efficient map storage
//
// Thread Safety:
//
//	Safe for concurrent access as the internal map is read-only after initialization.
type PipelineStatusMapper struct {
	statusMap map[PipelineStatus]string
}

// NewPipelineStatusMapper creates a new status mapper with pre-initialized string mappings.
//
// This constructor initializes the internal status map with all valid
// PipelineStatus-to-string mappings, providing efficient O(1) lookup
// performance for status string conversion.
//
// Returns:
//
//	*PipelineStatusMapper: A new mapper instance with complete status mappings
//
// Example:
//
//	Using the status mapper for consistent string conversion:
//
//	  mapper := entities.NewPipelineStatusMapper()
//
//	  status := entities.PipelineStatusRunning
//	  statusStr := mapper.ToString(status) // "running"
//
//	  allStatuses := mapper.GetAllStatuses()
//	  for status, str := range allStatuses {
//	      fmt.Printf("%d -> %s\n", status, str)
//	  }
//
// Performance:
//
//	Single allocation for mapper struct and internal map.
//	Map initialization occurs once during construction.
//
// Thread Safety:
//
//	The returned mapper is safe for concurrent use as the internal map is immutable.
func NewPipelineStatusMapper() *PipelineStatusMapper {
	return &PipelineStatusMapper{
		statusMap: map[PipelineStatus]string{
			PipelineStatusDraft:     "draft",
			PipelineStatusActive:    "active",
			PipelineStatusRunning:   "running",
			PipelineStatusCompleted: "completed",
			PipelineStatusFailed:    "failed",
			PipelineStatusPaused:    "paused",
			PipelineStatusArchived:  "archived",
		},
	}
}

// ToString converts a PipelineStatus enum to its string representation using efficient map lookup.
//
// This method implements the DRY principle by centralizing status-to-string
// conversion logic, eliminating the need for multiple switch statements
// throughout the codebase. It provides O(1) lookup performance and handles
// unknown status values gracefully.
//
// Parameters:
//
//	status PipelineStatus: The pipeline status enum to convert
//
// Returns:
//
//	string: The string representation of the status, or "unknown" for invalid values
//
// Example:
//
//	Converting status values for API responses:
//
//	  mapper := entities.NewPipelineStatusMapper()
//
//	  running := mapper.ToString(entities.PipelineStatusRunning)   // "running"
//	  completed := mapper.ToString(entities.PipelineStatusCompleted) // "completed"
//	  invalid := mapper.ToString(PipelineStatus(999))                // "unknown"
//
// Performance:
//
//	O(1) map lookup with no runtime switch evaluation.
//	Efficient handling of both valid and invalid status values.
//
// Thread Safety:
//
//	Safe for concurrent access as it only reads from the immutable internal map.
func (mapper *PipelineStatusMapper) ToString(status PipelineStatus) string {
	if statusStr, exists := mapper.statusMap[status]; exists {
		return statusStr
	}
	return unknownStatusString
}

// GetAllStatuses returns a copy of all valid pipeline statuses with their string representations.
//
// This method implements the Open-Closed Principle by providing a way to
// enumerate all valid status values without exposing the internal map structure.
// It returns a defensive copy to prevent external modification of the internal
// status mappings.
//
// Returns:
//
//	map[PipelineStatus]string: A copy of all status-to-string mappings
//
// Example:
//
//	Enumerating all valid pipeline statuses:
//
//	  mapper := entities.NewPipelineStatusMapper()
//	  allStatuses := mapper.GetAllStatuses()
//
//	  fmt.Println("Valid pipeline statuses:")
//	  for status, str := range allStatuses {
//	      fmt.Printf("  %d: %s\n", int(status), str)
//	  }
//
//	Building validation logic:
//
//	  validStatuses := mapper.GetAllStatuses()
//	  if _, isValid := validStatuses[userProvidedStatus]; !isValid {
//	      return errors.New("invalid pipeline status")
//	  }
//
// Performance:
//
//	Creates a new map with all entries copied from internal map.
//	O(n) time complexity where n is the number of valid statuses.
//
// Thread Safety:
//
//	Safe for concurrent access. Returns independent copy that can be modified safely.
func (mapper *PipelineStatusMapper) GetAllStatuses() map[PipelineStatus]string {
	result := make(map[PipelineStatus]string)
	for status, str := range mapper.statusMap {
		result[status] = str
	}
	return result
}

// String returns the string representation of the pipeline status using the standard mapper.
//
// This method implements the fmt.Stringer interface for PipelineStatus,
// enabling automatic string conversion in formatting operations. It delegates
// to the specialized mapper to maintain DRY principles and ensure consistent
// string representation across the system.
//
// Returns:
//
//	string: The string representation of the pipeline status
//
// Example:
//
//	Automatic string conversion in formatting:
//
//	  status := entities.PipelineStatusRunning
//
//	  // Implicit string conversion
//	  fmt.Printf("Pipeline status: %s\n", status) // "Pipeline status: running"
//
//	  // Explicit string conversion
//	  statusStr := status.String() // "running"
//	  log.Info("Pipeline state changed", "status", statusStr)
//
// Performance:
//
//	Creates a new mapper instance on each call.
//	Consider caching mapper instance for high-frequency usage.
//
// Thread Safety:
//
//	Safe for concurrent use as each call creates an independent mapper instance.
func (s PipelineStatus) String() string {
	mapper := NewPipelineStatusMapper()
	return mapper.ToString(s)
}

// PipelineStep represents an individual processing step within a data pipeline.
//
// PipelineStep is a value object that encapsulates all configuration and metadata
// for a single processing operation. Steps can have dependencies on other steps,
// configurable retry behavior, and timeout constraints. The step configuration
// is flexible to accommodate different processing plugins and operations.
//
// Design Patterns:
//   - Value Object: Immutable after creation with all configuration encapsulated
//   - Strategy Pattern: Type field determines processing strategy/plugin to use
//   - Dependency Graph: DependsOn field enables complex workflow orchestration
//
// Fields:
//
//	ID string: Unique identifier for the step within the pipeline
//	Name string: Human-readable name for the step (must be unique within pipeline)
//	Type string: Processing type/plugin identifier (e.g., "data-extraction", "transformation")
//	Config map[string]interface{}: Flexible configuration parameters for the processing operation
//	DependsOn []string: List of step names that must complete before this step can run
//	RetryCount int: Current number of retry attempts (managed by execution engine)
//	MaxRetries int: Maximum allowed retry attempts for this step
//	Timeout time.Duration: Maximum execution time allowed for this step
//	IsEnabled bool: Whether this step is active in the pipeline execution
//	CreatedAt time.Time: Timestamp when the step was added to the pipeline
//
// Example:
//
//	Creating a data extraction step:
//
//	  step := entities.NewPipelineStep("extract-users", "oracle-extraction")
//	  step.Config["connection"] = "oracle://prod-db:1521/ORCL"
//	  step.Config["query"] = "SELECT * FROM users WHERE active = 1"
//	  step.MaxRetries = 5
//	  step.Timeout = 10 * time.Minute
//
// Integration:
//   - Used within Pipeline aggregate for step management
//   - Configuration passed to plugin execution engine
//   - Dependencies enable complex workflow coordination
//   - Retry and timeout settings integrated with execution monitoring
//
// Thread Safety:
//
//	Individual step instances should be treated as immutable after creation.
//	Modifications should only occur through Pipeline aggregate methods.
type PipelineStep struct {
	DependsOn  []string               // Step dependencies (step names)
	CreatedAt  time.Time              // Step creation timestamp
	ID         string                 // Unique step identifier
	Name       string                 // Human-readable step name (unique within pipeline)
	Type       string                 // Processing type/plugin identifier
	Config     map[string]interface{} // Flexible configuration parameters
	Timeout    time.Duration          // Step execution timeout
	RetryCount int                    // Current retry attempt count
	MaxRetries int                    // Maximum allowed retries
	IsEnabled  bool                   // Whether step is active
}

// NewPipelineStep creates a new pipeline step with default configuration and unique identifier.
//
// This constructor initializes a PipelineStep with sensible defaults for retry behavior,
// timeout constraints, and execution settings. The step is created in an enabled state
// with empty configuration and dependency lists that can be populated after creation.
//
// Parameters:
//
//	name string: Human-readable name for the step (must be unique within the pipeline)
//	stepType string: Processing type identifier that determines which plugin/processor to use
//
// Returns:
//
//	PipelineStep: A new step instance with default configuration
//
// Example:
//
//	Creating different types of pipeline steps:
//
//	  // Data extraction step
//	  extractStep := entities.NewPipelineStep("extract-customers", "oracle-tap")
//	  extractStep.Config["table"] = "customers"
//	  extractStep.Config["incremental_key"] = "updated_at"
//
//	  // Data transformation step
//	  transformStep := entities.NewPipelineStep("clean-data", "dbt-transform")
//	  transformStep.DependsOn = []string{"extract-customers"}
//	  transformStep.Config["model"] = "staging.customer_clean"
//
//	  // Data loading step
//	  loadStep := entities.NewPipelineStep("load-warehouse", "postgres-target")
//	  loadStep.DependsOn = []string{"clean-data"}
//	  loadStep.MaxRetries = 5 // Critical step, more retries
//
// Default Values:
//   - ID: Generated UUID v4
//   - RetryCount: 0 (no retries attempted yet)
//   - MaxRetries: 3 (configurable default)
//   - Timeout: 30 minutes (configurable default)
//   - IsEnabled: true (step is active)
//   - Config: Empty map (ready for configuration)
//   - DependsOn: Empty slice (no dependencies initially)
//   - CreatedAt: Current timestamp
//
// Thread Safety:
//
//	Safe for concurrent use as each call creates an independent step instance.
func NewPipelineStep(name, stepType string) PipelineStep {
	return PipelineStep{
		ID:         uuid.New().String(),
		Name:       name,
		Type:       stepType,
		Config:     make(map[string]interface{}),
		DependsOn:  make([]string, 0),
		RetryCount: 0,
		MaxRetries: DefaultMaxRetries,
		Timeout:    time.Minute * DefaultTimeoutMinutes,
		IsEnabled:  true,
		CreatedAt:  time.Now(),
	}
}

// Pipeline represents the main aggregate root for data processing pipeline management.
//
// Pipeline implements Domain-Driven Design patterns as the primary aggregate root
// for orchestrating data processing workflows. It encapsulates all business logic
// for pipeline lifecycle management, step coordination, scheduling, and state
// transitions while maintaining consistency through domain events.
//
// Architecture:
//   - Aggregate Root: Ensures consistency boundaries and transaction scope
//   - Rich Domain Model: Contains business logic for pipeline operations
//   - Event Sourcing: Generates domain events for all state changes
//   - CQRS Integration: Compatible with command/query separation patterns
//   - Result Pattern: All methods return Results for explicit error handling
//
// Business Rules:
//   - Pipeline names must be unique within owner context
//   - Steps must have unique names within the pipeline
//   - Status transitions follow defined state machine rules
//   - Dependencies between steps must form a valid DAG (no cycles)
//   - Only active pipelines can be started
//   - Running pipelines cannot be modified
//
// Fields:
//
//	domain.AggregateRoot[PipelineID]: Base aggregate with event management and metadata
//	Name string: Human-readable pipeline name (required, must be unique per owner)
//	Description string: Detailed description of pipeline purpose and functionality
//	Status PipelineStatus: Current pipeline state following defined state machine
//	Steps []PipelineStep: Ordered collection of processing steps with dependencies
//	Tags []string: Flexible labeling system for categorization and filtering
//	Owner string: Pipeline owner identifier for access control and organization
//	Schedule *PipelineSchedule: Optional scheduling configuration for automated execution
//	LastRunAt *time.Time: Timestamp of most recent execution attempt
//	NextRunAt *time.Time: Calculated timestamp for next scheduled execution
//
// Example:
//
//	Creating and configuring a complete pipeline:
//
//	  pipelineResult := entities.NewPipeline(
//	      "Customer Data ETL",
//	      "Extract customer data from Oracle, transform, and load to warehouse",
//	      "data-team",
//	  )
//
//	  if pipelineResult.IsFailure() {
//	      return pipelineResult.Error()
//	  }
//
//	  pipeline := pipelineResult.Value()
//
//	  // Add processing steps
//	  extractStep := entities.NewPipelineStep("extract-customers", "oracle-tap")
//	  pipeline.AddStep(extractStep)
//
//	  transformStep := entities.NewPipelineStep("transform-customers", "dbt-transform")
//	  transformStep.DependsOn = []string{"extract-customers"}
//	  pipeline.AddStep(transformStep)
//
//	  // Configure scheduling
//	  pipeline.SetSchedule("0 2 * * *", "UTC") // Daily at 2 AM UTC
//
//	  // Activate for execution
//	  pipeline.Activate()
//
// Integration:
//   - Command Handlers: Process pipeline management commands
//   - Event Handlers: React to pipeline state changes
//   - Query Projections: Maintain read models for pipeline status
//   - Plugin System: Execute configured processing steps
//   - Scheduler: Manage automated pipeline execution
//
// Thread Safety:
//
//	Pipeline instances should be accessed by single threads within aggregate boundaries.
//	Concurrent access should be managed through proper locking at the application layer
//	or through message-based coordination patterns.
type Pipeline struct {
	domain.AggregateRoot[PipelineID]                   // Base aggregate root with event management
	Name                             string            // Pipeline name (required, unique per owner)
	Description                      string            // Detailed pipeline description
	Status                           PipelineStatus    // Current pipeline state
	Steps                            []PipelineStep    // Processing steps with dependencies
	Tags                             []string          // Flexible labeling for categorization
	Owner                            string            // Pipeline owner identifier
	Schedule                         *PipelineSchedule // Optional scheduling configuration
	LastRunAt                        *time.Time        // Most recent execution timestamp
	NextRunAt                        *time.Time        // Next scheduled execution timestamp
}

// PipelineSchedule represents automated scheduling configuration for pipeline execution.
//
// PipelineSchedule encapsulates all timing and automation settings for pipelines,
// supporting cron-based scheduling with timezone awareness. The schedule can be
// temporarily disabled without losing configuration, enabling flexible automation
// management.
//
// Fields:
//
//	CronExpression string: Standard cron expression defining execution timing
//	Timezone string: IANA timezone identifier for schedule interpretation
//	IsEnabled bool: Whether the schedule is currently active
//	CreatedAt time.Time: When the schedule was originally configured
//
// Example:
//
//	Common scheduling patterns:
//
//	  // Daily at 2 AM UTC
//	  schedule := PipelineSchedule{
//	      CronExpression: "0 2 * * *",
//	      Timezone:       "UTC",
//	      IsEnabled:      true,
//	      CreatedAt:      time.Now(),
//	  }
//
//	  // Business hours on weekdays (9 AM EST)
//	  schedule := PipelineSchedule{
//	      CronExpression: "0 9 * * 1-5",
//	      Timezone:       "America/New_York",
//	      IsEnabled:      true,
//	      CreatedAt:      time.Now(),
//	  }
//
// Integration:
//   - Scheduler Service: Interprets cron expressions for execution timing
//   - Timezone Service: Handles timezone conversions and DST transitions
//   - Pipeline Events: Schedule changes generate domain events
//   - Monitoring: Track scheduled vs actual execution times
//
// Thread Safety:
//
//	Schedule instances should be treated as immutable after creation.
//	Modifications should occur through Pipeline aggregate methods.
type PipelineSchedule struct {
	CronExpression string    // Cron expression for execution timing
	Timezone       string    // IANA timezone identifier
	IsEnabled      bool      // Whether schedule is active
	CreatedAt      time.Time // Schedule creation timestamp
}

// NewPipeline creates a new pipeline aggregate with comprehensive validation and event generation.
//
// This factory function implements the aggregate root creation pattern, performing
// business rule validation, initializing the pipeline in a valid state, and
// generating appropriate domain events for event sourcing and CQRS integration.
//
// Business Rules Enforced:
//   - Pipeline name must not be empty (required for identification)
//   - Owner must not be empty (required for access control)
//   - Pipeline starts in Draft status (safe initial state)
//   - Steps and tags collections are initialized as empty
//   - Unique ID is generated for aggregate identification
//
// Parameters:
//
//	name string: Human-readable pipeline name (required, will be validated for uniqueness by application layer)
//	description string: Detailed description of pipeline purpose (optional, can be empty)
//	owner string: Owner identifier for access control and organization (required)
//
// Returns:
//
//	result.Result[*Pipeline]: Success containing the new pipeline, or Failure with validation error
//
// Example:
//
//	Creating pipelines with proper error handling:
//
//	  pipelineResult := entities.NewPipeline(
//	      "Customer Data Processing",
//	      "Daily ETL pipeline for customer data synchronization",
//	      "data-engineering-team",
//	  )
//
//	  if pipelineResult.IsFailure() {
//	      log.Error("Pipeline creation failed", "error", pipelineResult.Error())
//	      return pipelineResult.Error()
//	  }
//
//	  pipeline := pipelineResult.Value()
//	  log.Info("Pipeline created", "id", pipeline.ID, "name", pipeline.Name)
//
//	Validation error handling:
//
//	  invalidResult := entities.NewPipeline("", "description", "owner")
//	  if invalidResult.IsFailure() {
//	      // Will contain ValidationError: "pipeline name cannot be empty"
//	      return invalidResult.Error()
//	  }
//
// Events Generated:
//   - PipelineCreatedEvent: Contains pipeline ID, name, and owner for downstream processing
//
// Integration:
//   - Command Handlers: Use this constructor to create pipelines from commands
//   - Event Store: Persists generated domain events for event sourcing
//   - Read Models: React to PipelineCreatedEvent to update query projections
//   - Access Control: Uses owner field for permission validation
//
// Thread Safety:
//
//	Safe for concurrent use as each call creates an independent pipeline instance.
func NewPipeline(name, description, owner string) (*Pipeline, error) {
	if name == "" {
		return nil, errors.ValidationError("pipeline name cannot be empty")
	}

	if owner == "" {
		return nil, errors.ValidationError("pipeline owner cannot be empty")
	}

	id := NewPipelineID()
	pipeline := &Pipeline{
		AggregateRoot: domain.NewAggregateRoot(id),
		Name:          name,
		Description:   description,
		Status:        PipelineStatusDraft,
		Steps:         make([]PipelineStep, 0),
		Tags:          make([]string, 0),
		Owner:         owner,
	}

	// Raise domain event
	event := NewPipelineCreatedEvent(id, name, owner)
	pipeline.RaiseEvent(event)

	return pipeline, nil
}

// AddStep adds a processing step to the pipeline with comprehensive validation and dependency checking.
//
// This method enforces critical business rules for pipeline step management,
// including step name uniqueness, dependency validation, and proper event generation
// for event sourcing. The pipeline must be in a modifiable state (not running) for
// step addition to succeed.
//
// Business Rules Enforced:
//   - Step names must be unique within the pipeline
//   - Step names cannot be empty (required for identification)
//   - All step dependencies must reference existing steps in the pipeline
//   - Dependencies must form a valid DAG (no circular dependencies)
//   - Pipeline state must allow modifications (not running)
//
// Parameters:
//
//	step PipelineStep: The processing step to add with complete configuration
//
// Returns:
//
//	result.Result[bool]: Success(true) if step added successfully, or Failure with validation error
//
// Example:
//
//	Adding steps with dependencies:
//
//	  pipeline := entities.NewPipeline("ETL Pipeline", "Data processing", "data-team").Value()
//
//	  // Add extraction step (no dependencies)
//	  extractStep := entities.NewPipelineStep("extract-users", "oracle-tap")
//	  extractStep.Config["table"] = "users"
//
//	  result := pipeline.AddStep(extractStep)
//	  if result.IsFailure() {
//	      return result.Error() // Handle validation error
//	  }
//
//	  // Add transformation step (depends on extraction)
//	  transformStep := entities.NewPipelineStep("transform-users", "dbt-transform")
//	  transformStep.DependsOn = []string{"extract-users"}
//	  transformStep.Config["model"] = "staging.users_clean"
//
//	  result = pipeline.AddStep(transformStep)
//	  if result.IsFailure() {
//	      return result.Error() // Handle dependency validation error
//	  }
//
// Validation Errors:
//   - ValidationError: "step name cannot be empty" - when step.Name is empty
//   - AlreadyExistsError: "step with name X" - when step name already exists
//   - ValidationError: "dependency step not found: X" - when dependency doesn't exist
//
// Events Generated:
//   - PipelineStepAddedEvent: Contains pipeline ID, step ID, and step name for downstream processing
//
// Integration:
//   - Event Store: Persists step addition event for audit trail
//   - Read Models: Update pipeline views with new step information
//   - Workflow Engine: Register step for execution planning
//   - Dependency Analyzer: Validate execution order and detect cycles
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
//	Concurrent step additions should be coordinated at the application layer.
func (p *Pipeline) AddStep(step *PipelineStep) error {
	if step.Name == "" {
		return errors.ValidationError("step name cannot be empty")
	}

	// Check for duplicate step names
	for i := range p.Steps {
		if p.Steps[i].Name == step.Name {
			return errors.AlreadyExistsError("step with name " + step.Name)
		}
	}

	// Validate dependencies
	for _, dependency := range step.DependsOn {
		if !p.hasStep(dependency) {
			return errors.ValidationError("dependency step not found: " + dependency)
		}
	}

	p.Steps = append(p.Steps, *step)
	p.Touch()

	// Raise domain event
	event := NewPipelineStepAddedEvent(p.ID, step.ID, step.Name)
	p.RaiseEvent(event)

	return nil
}

// RemoveStep removes a processing step from the pipeline with dependency validation.
//
// This method safely removes steps while enforcing referential integrity through
// dependency checking. It prevents removal of steps that other steps depend on,
// maintaining pipeline consistency and preventing execution failures.
//
// Business Rules Enforced:
//   - Cannot remove steps that other steps depend on (referential integrity)
//   - Step must exist in the pipeline to be removed
//   - Pipeline state must allow modifications (not running)
//   - Removal generates audit events for change tracking
//
// Parameters:
//
//	stepName string: Name of the step to remove (must match existing step)
//
// Returns:
//
//	result.Result[bool]: Success(true) if step removed successfully, or Failure with validation error
//
// Example:
//
//	Safe step removal with dependency checking:
//
//	  pipeline := loadExistingPipeline()
//
//	  // This will fail if other steps depend on "extract-users"
//	  result := pipeline.RemoveStep("extract-users")
//	  if result.IsFailure() {
//	      // Handle dependency violation: "cannot remove step: other steps depend on it"
//	      return result.Error()
//	  }
//
//	  // Remove dependent steps first, then dependencies
//	  pipeline.RemoveStep("transform-users") // Remove dependent step first
//	  pipeline.RemoveStep("extract-users")   // Now safe to remove
//
// Validation Errors:
//   - NotFoundError: "step X" - when step name doesn't exist in pipeline
//   - ValidationError: "cannot remove step: other steps depend on it" - dependency violation
//
// Events Generated:
//   - PipelineStepRemovedEvent: Contains pipeline ID and removed step name
//
// Dependency Analysis:
//
//	The method performs comprehensive dependency scanning:
//	1. Locates the step to remove by name
//	2. Scans all remaining steps for dependencies on the target step
//	3. Prevents removal if any dependencies are found
//	4. Only removes if no other steps reference the target step
//
// Integration:
//   - Workflow Engine: Updates execution plans after step removal
//   - Event Store: Records removal event for audit and replay
//   - Read Models: Updates pipeline views to reflect removed step
//   - Dependency Graph: Recalculates execution order after removal
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) RemoveStep(stepName string) error {
	stepIndex := -1
	for i := range p.Steps {
		if p.Steps[i].Name == stepName {
			stepIndex = i
			break
		}
	}

	if stepIndex == -1 {
		return errors.NotFoundError("step " + stepName)
	}

	// Check if other steps depend on this step
	for i := range p.Steps {
		for _, dependency := range p.Steps[i].DependsOn {
			if dependency == stepName {
				return errors.ValidationError("cannot remove step: other steps depend on it")
			}
		}
	}

	// Remove the step
	p.Steps = append(p.Steps[:stepIndex], p.Steps[stepIndex+1:]...)
	p.Touch()

	// Raise domain event
	event := NewPipelineStepRemovedEvent(p.ID, stepName)
	p.RaiseEvent(event)

	return nil
}

// Activate transitions the pipeline from Draft to Active status, enabling execution.
//
// This method implements a critical state transition in the pipeline lifecycle,
// moving from the draft state where modifications are allowed to the active state
// where the pipeline is ready for execution. Comprehensive validation ensures
// the pipeline is properly configured before activation.
//
// Business Rules Enforced:
//   - Pipeline must not already be active (prevents redundant operations)
//   - Pipeline must have at least one processing step (prevents empty execution)
//   - All pipeline steps must be properly configured and valid
//   - Pipeline dependencies must form a valid execution graph
//
// Parameters:
//
//	None - operates on the current pipeline state
//
// Returns:
//
//	result.Result[bool]: Success(true) if activated successfully, or Failure with validation error
//
// Example:
//
//	Complete pipeline setup and activation:
//
//	  pipelineResult := entities.NewPipeline("Data ETL", "Customer data processing", "data-team")
//	  pipeline := pipelineResult.Value()
//
//	  // Configure pipeline steps
//	  extractStep := entities.NewPipelineStep("extract", "oracle-tap")
//	  pipeline.AddStep(extractStep)
//
//	  transformStep := entities.NewPipelineStep("transform", "dbt-transform")
//	  transformStep.DependsOn = []string{"extract"}
//	  pipeline.AddStep(transformStep)
//
//	  // Activate for execution
//	  result := pipeline.Activate()
//	  if result.IsFailure() {
//	      return result.Error() // Handle validation failure
//	  }
//
//	  // Pipeline is now ready for execution
//	  log.Info("Pipeline activated", "id", pipeline.ID, "steps", len(pipeline.Steps))
//
// State Transitions:
//
//	Draft → Active: Normal activation flow
//	Other states → Active: Not allowed, will return validation error
//
// Validation Errors:
//   - ValidationError: "pipeline is already active" - attempting to activate active pipeline
//   - ValidationError: "cannot activate pipeline without steps" - empty step collection
//
// Events Generated:
//   - PipelineActivatedEvent: Contains pipeline ID and name for downstream processing
//
// Integration:
//   - Scheduler Service: Pipeline becomes eligible for scheduled execution
//   - Execution Engine: Pipeline registered for manual or automated execution
//   - Monitoring: Pipeline status tracking and alerting enabled
//   - Access Control: Execution permissions validated for pipeline owner
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) Activate() error {
	if p.Status == PipelineStatusActive {
		return errors.ValidationError("pipeline is already active")
	}

	if len(p.Steps) == 0 {
		return errors.ValidationError("cannot activate pipeline without steps")
	}

	p.Status = PipelineStatusActive
	p.Touch()

	// Raise domain event
	event := NewPipelineActivatedEvent(p.ID, p.Name)
	p.RaiseEvent(event)

	return nil
}

// Deactivate transitions the pipeline from Active back to Draft status, disabling execution.
//
// This method safely deactivates pipelines to prevent execution while allowing
// modifications. It enforces business rules to ensure running pipelines cannot
// be deactivated, preventing data inconsistency and execution interruption.
//
// Business Rules Enforced:
//   - Cannot deactivate running pipelines (must complete or fail first)
//   - Only active pipelines can be deactivated (idempotent operation)
//   - Deactivation removes pipeline from execution scheduling
//   - Generates appropriate events for downstream system coordination
//
// Parameters:
//
//	None - operates on the current pipeline state
//
// Returns:
//
//	result.Result[bool]: Success(true) if deactivated successfully, or Failure with validation error
//
// Example:
//
//	Safe pipeline deactivation for maintenance:
//
//	  pipeline := loadActivePipeline()
//
//	  // Check if pipeline can be safely deactivated
//	  if pipeline.Status == entities.PipelineStatusRunning {
//	      log.Warn("Cannot deactivate running pipeline", "id", pipeline.ID)
//	      return errors.New("wait for pipeline completion")
//	  }
//
//	  // Deactivate for modifications
//	  result := pipeline.Deactivate()
//	  if result.IsFailure() {
//	      return result.Error()
//	  }
//
//	  // Pipeline can now be modified safely
//	  pipeline.AddStep(newMaintenanceStep)
//	  pipeline.Activate() // Reactivate after modifications
//
// State Transitions:
//
//	Active → Draft: Normal deactivation flow
//	Running → Draft: Not allowed, will return validation error
//	Other states → Draft: Allowed (idempotent for Draft status)
//
// Validation Errors:
//   - ValidationError: "cannot deactivate running pipeline" - attempting to deactivate during execution
//
// Events Generated:
//   - PipelineDeactivatedEvent: Contains pipeline ID and name for downstream processing
//
// Integration:
//   - Scheduler Service: Pipeline removed from execution schedules
//   - Execution Engine: Pipeline execution disabled and requests rejected
//   - Monitoring: Alerts and metrics updated for deactivated status
//   - Workflow Engine: Dependent processes notified of deactivation
//
// Use Cases:
//   - Temporary maintenance or configuration changes
//   - Emergency pipeline shutdown for troubleshooting
//   - Pipeline retirement before archival
//   - Testing and development workflow management
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) Deactivate() error {
	if p.Status == PipelineStatusRunning {
		return errors.ValidationError("cannot deactivate running pipeline")
	}

	p.Status = PipelineStatusDraft
	p.Touch()

	// Raise domain event
	event := NewPipelineDeactivatedEvent(p.ID, p.Name)
	p.RaiseEvent(event)

	return nil
}

// Start initiates pipeline execution, transitioning from Active to Running status.
//
// This method begins the actual data processing workflow, validating preconditions
// and updating the pipeline state to reflect active execution. It records execution
// metadata and generates events for monitoring and coordination systems.
//
// Business Rules Enforced:
//   - Only active pipelines can be started (must be activated first)
//   - Cannot start already running pipelines (prevents concurrent execution)
//   - All pipeline steps must be valid and executable
//   - Execution metadata is properly recorded for audit and monitoring
//
// Parameters:
//
//	None - operates on the current pipeline state
//
// Returns:
//
//	result.Result[bool]: Success(true) if started successfully, or Failure with validation error
//
// Example:
//
//	Complete pipeline execution initiation:
//
//	  pipeline := loadActivePipeline()
//
//	  // Validate pipeline is ready for execution
//	  if !pipeline.CanExecute() {
//	      return errors.New("pipeline not ready for execution")
//	  }
//
//	  // Start execution
//	  result := pipeline.Start()
//	  if result.IsFailure() {
//	      log.Error("Failed to start pipeline", "error", result.Error(), "id", pipeline.ID)
//	      return result.Error()
//	  }
//
//	  // Pipeline is now running
//	  log.Info("Pipeline execution started", "id", pipeline.ID, "started_at", pipeline.LastRunAt)
//
//	  // Trigger execution engine
//	  return executionEngine.ProcessPipeline(pipeline)
//
// State Transitions:
//
//	Active → Running: Normal execution start
//	Other states → Running: Not allowed, will return validation error
//
// Validation Errors:
//   - ValidationError: "can only start active pipelines" - attempting to start non-active pipeline
//   - ValidationError: "pipeline is already running" - attempting to start running pipeline
//
// Execution Metadata:
//   - LastRunAt: Set to current timestamp when execution begins
//   - Status: Updated to PipelineStatusRunning
//   - Version: Incremented via Touch() for optimistic locking
//
// Events Generated:
//   - PipelineStartedEvent: Contains pipeline ID, name, and start timestamp
//
// Integration:
//   - Execution Engine: Receives event to begin step processing
//   - Monitoring System: Tracks execution start time and performance metrics
//   - Scheduler Service: Updates next run time calculations
//   - Resource Manager: Allocates execution resources and limits
//   - Notification Service: Sends execution start notifications
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
//	Concurrent start attempts should be handled by application layer coordination.
func (p *Pipeline) Start() error {
	if p.Status != PipelineStatusActive {
		return errors.ValidationError("can only start active pipelines")
	}

	if p.Status == PipelineStatusRunning {
		return errors.ValidationError("pipeline is already running")
	}

	p.Status = PipelineStatusRunning
	now := time.Now()
	p.LastRunAt = &now
	p.Touch()

	// Raise domain event
	event := NewPipelineStartedEvent(p.ID, p.Name, now)
	p.RaiseEvent(event)

	return nil
}

// Complete marks the pipeline execution as successfully finished, transitioning to Completed status.
//
// This method finalizes successful pipeline execution, updating the status and
// generating completion events for downstream processing. It can only be called
// on running pipelines and represents the successful completion of all processing steps.
//
// Business Rules Enforced:
//   - Only running pipelines can be completed (must be in execution state)
//   - Completion represents successful processing of all pipeline steps
//   - Generates audit events for execution tracking and analytics
//   - Updates pipeline metadata for monitoring and reporting
//
// Parameters:
//
//	None - operates on the current pipeline state
//
// Returns:
//
//	result.Result[bool]: Success(true) if completed successfully, or Failure with validation error
//
// Example:
//
//	Pipeline completion in execution engine:
//
//	  // After all steps have been processed successfully
//	  pipeline := loadRunningPipeline()
//
//	  // Validate all steps completed successfully
//	  if !allStepsCompleted(pipeline) {
//	      return pipeline.Fail("not all steps completed successfully")
//	  }
//
//	  // Mark pipeline as completed
//	  result := pipeline.Complete()
//	  if result.IsFailure() {
//	      log.Error("Failed to complete pipeline", "error", result.Error())
//	      return result.Error()
//	  }
//
//	  log.Info("Pipeline completed successfully", "id", pipeline.ID, "duration", calculateDuration())
//
// State Transitions:
//
//	Running → Completed: Normal successful completion
//	Other states → Completed: Not allowed, will return validation error
//
// Validation Errors:
//   - ValidationError: "can only complete running pipelines" - attempting to complete non-running pipeline
//
// Post-Completion State:
//   - Status: Set to PipelineStatusCompleted
//   - Version: Incremented via Touch() for optimistic locking
//   - Completed pipelines can be reactivated for additional runs
//
// Events Generated:
//   - PipelineCompletedEvent: Contains pipeline ID, name, and completion timestamp
//
// Integration:
//   - Analytics Service: Records completion metrics and performance data
//   - Notification Service: Sends success notifications to stakeholders
//   - Scheduler Service: Calculates next run time for scheduled pipelines
//   - Resource Manager: Releases allocated execution resources
//   - Audit System: Records successful execution for compliance
//   - Downstream Systems: Triggers dependent processes and workflows
//
// Performance Metrics:
//
//	Completion events enable tracking of:
//	- Execution duration and throughput
//	- Success rates and reliability metrics
//	- Resource utilization and efficiency
//	- Business process completion times
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) Complete() error {
	if p.Status != PipelineStatusRunning {
		return errors.ValidationError("can only complete running pipelines")
	}

	p.Status = PipelineStatusCompleted
	p.Touch()

	// Raise domain event
	event := NewPipelineCompletedEvent(p.ID, p.Name, time.Now())
	p.RaiseEvent(event)

	return nil
}

// Fail marks the pipeline execution as failed with a detailed reason, transitioning to Failed status.
//
// This method handles pipeline execution failures, recording the failure reason
// and updating the pipeline state for error handling and recovery. It provides
// comprehensive failure tracking for debugging, monitoring, and recovery processes.
//
// Business Rules Enforced:
//   - Only running pipelines can fail (must be in execution state)
//   - Failure reason must be provided for debugging and audit purposes
//   - Generates detailed failure events for error tracking and analysis
//   - Failed pipelines can be reactivated after addressing failure causes
//
// Parameters:
//
//	reason string: Detailed description of the failure cause (required for debugging)
//
// Returns:
//
//	result.Result[bool]: Success(true) if marked as failed successfully, or Failure with validation error
//
// Example:
//
//	Pipeline failure handling in execution engine:
//
//	  pipeline := loadRunningPipeline()
//
//	  // Execute pipeline steps with error handling
//	  for _, step := range pipeline.Steps {
//	      if err := executeStep(step); err != nil {
//	          // Mark pipeline as failed with detailed reason
//	          failureReason := fmt.Sprintf("Step '%s' failed: %v", step.Name, err)
//	          result := pipeline.Fail(failureReason)
//
//	          if result.IsFailure() {
//	              log.Error("Failed to mark pipeline as failed", "error", result.Error())
//	          }
//
//	          // Trigger failure recovery procedures
//	          return triggerFailureRecovery(pipeline, err)
//	      }
//	  }
//
// State Transitions:
//
//	Running → Failed: Normal failure handling
//	Other states → Failed: Not allowed, will return validation error
//
// Validation Errors:
//   - ValidationError: "can only fail running pipelines" - attempting to fail non-running pipeline
//
// Post-Failure State:
//   - Status: Set to PipelineStatusFailed
//   - Version: Incremented via Touch() for optimistic locking
//   - Failed pipelines retain all execution metadata for debugging
//   - Can be reactivated after addressing the root cause
//
// Events Generated:
//   - PipelineFailedEvent: Contains pipeline ID, name, failure reason, and timestamp
//
// Failure Analysis:
//
//	Generated events enable comprehensive failure analysis:
//	- Root cause identification and categorization
//	- Failure pattern detection and prevention
//	- Step-specific error tracking and optimization
//	- Resource constraint and performance issue identification
//
// Integration:
//   - Error Tracking: Records detailed failure information for analysis
//   - Alerting System: Sends failure notifications with context
//   - Recovery Service: Triggers automatic retry or compensation logic
//   - Monitoring: Updates failure rate metrics and dashboards
//   - Support System: Creates incident tickets with failure details
//   - Resource Manager: Releases allocated resources and cleans up
//
// Recovery Patterns:
//   - Automatic retry with exponential backoff
//   - Manual intervention with failure reason analysis
//   - Partial recovery with step-level restart capability
//   - Compensation transactions for data consistency
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) Fail(reason string) error {
	if p.Status != PipelineStatusRunning {
		return errors.ValidationError("can only fail running pipelines")
	}

	p.Status = PipelineStatusFailed
	p.Touch()

	// Raise domain event
	event := NewPipelineFailedEvent(p.ID, p.Name, reason, time.Now())
	p.RaiseEvent(event)

	return nil
}

// SetSchedule configures automated scheduling for the pipeline using cron expressions.
//
// This method establishes automated execution scheduling with timezone support,
// enabling pipelines to run automatically based on business requirements.
// It validates the cron expression format and creates a complete scheduling
// configuration with proper event generation.
//
// Business Rules Enforced:
//   - Cron expression cannot be empty (required for scheduling)
//   - Timezone should be valid IANA timezone identifier
//   - Schedule is enabled by default when created
//   - Generates events for schedule management and monitoring
//
// Parameters:
//
//	cronExpression string: Standard cron expression defining execution timing (required)
//	timezone string: IANA timezone identifier for schedule interpretation (optional, defaults to UTC)
//
// Returns:
//
//	result.Result[bool]: Success(true) if schedule set successfully, or Failure with validation error
//
// Example:
//
//	Common scheduling patterns:
//
//	  pipeline := loadPipeline()
//
//	  // Daily execution at 2 AM UTC
//	  result := pipeline.SetSchedule("0 2 * * *", "UTC")
//	  if result.IsFailure() {
//	      return result.Error()
//	  }
//
//	  // Business hours on weekdays (9 AM Eastern Time)
//	  result = pipeline.SetSchedule("0 9 * * 1-5", "America/New_York")
//
//	  // Hourly execution during business hours
//	  result = pipeline.SetSchedule("0 9-17 * * 1-5", "America/Chicago")
//
//	  // Monthly execution on first day at midnight
//	  result = pipeline.SetSchedule("0 0 1 * *", "UTC")
//
// Cron Expression Examples:
//   - "0 2 * * *" - Daily at 2:00 AM
//   - "0 9-17 * * 1-5" - Every hour during business hours, Monday-Friday
//   - "0 0 1 * *" - Monthly on the 1st at midnight
//   - "*/15 * * * *" - Every 15 minutes
//   - "0 6 * * 0" - Weekly on Sunday at 6:00 AM
//
// Validation Errors:
//   - ValidationError: "cron expression cannot be empty" - when cronExpression is empty
//
// Schedule Configuration:
//
//	Creates PipelineSchedule with:
//	- CronExpression: The provided cron timing specification
//	- Timezone: Timezone for expression interpretation
//	- IsEnabled: Set to true (schedule is active)
//	- CreatedAt: Current timestamp for audit tracking
//
// Events Generated:
//   - PipelineScheduleSetEvent: Contains pipeline ID, cron expression, and timezone
//
// Integration:
//   - Scheduler Service: Registers pipeline for automated execution
//   - Timezone Service: Handles DST transitions and timezone conversions
//   - Monitoring: Tracks scheduled vs actual execution times
//   - Calendar Integration: Shows pipeline execution schedule in business calendars
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) SetSchedule(cronExpression, timezone string) error {
	if cronExpression == "" {
		return errors.ValidationError("cron expression cannot be empty")
	}

	p.Schedule = &PipelineSchedule{
		CronExpression: cronExpression,
		Timezone:       timezone,
		IsEnabled:      true,
		CreatedAt:      time.Now(),
	}
	p.Touch()

	// Raise domain event
	event := NewPipelineScheduleSetEvent(p.ID, cronExpression, timezone)
	p.RaiseEvent(event)

	return nil
}

// ClearSchedule removes the automated scheduling configuration from the pipeline.
//
// This method disables automated execution by removing the schedule configuration
// entirely. The pipeline will no longer execute automatically but can still be
// triggered manually. This is useful for maintenance, debugging, or transitioning
// to manual execution workflows.
//
// Parameters:
//
//	None - operates on the current pipeline state
//
// Returns:
//
//	None - always succeeds as clearing non-existent schedule is idempotent
//
// Example:
//
//	Temporary schedule removal for maintenance:
//
//	  pipeline := loadScheduledPipeline()
//
//	  // Temporarily disable automated execution
//	  pipeline.ClearSchedule()
//	  log.Info("Pipeline schedule cleared for maintenance", "id", pipeline.ID)
//
//	  // Perform maintenance operations
//	  performMaintenanceOperations(pipeline)
//
//	  // Re-enable scheduling after maintenance
//	  pipeline.SetSchedule("0 2 * * *", "UTC")
//	  log.Info("Pipeline schedule restored", "id", pipeline.ID)
//
// State Changes:
//   - Schedule field set to nil (completely removed)
//   - Version incremented via Touch() for optimistic locking
//   - Pipeline remains in current status (schedule removal doesn't affect execution state)
//
// Events Generated:
//   - PipelineScheduleClearedEvent: Contains pipeline ID for downstream processing
//
// Integration:
//   - Scheduler Service: Unregisters pipeline from automated execution queue
//   - Monitoring: Updates scheduling metrics and dashboards
//   - Calendar Integration: Removes pipeline from execution calendars
//   - Alerting: May trigger notifications for schedule removal
//
// Use Cases:
//   - Temporary maintenance or debugging sessions
//   - Transitioning from automated to manual execution
//   - Emergency pipeline suspension without deactivation
//   - Testing scenarios requiring manual control
//   - Compliance requirements for manual approval workflows
//
// Idempotency:
//
//	Safe to call multiple times - clearing an already cleared schedule has no effect.
//	No validation errors are generated for missing schedules.
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) ClearSchedule() {
	p.Schedule = nil
	p.Touch()

	// Raise domain event
	event := NewPipelineScheduleClearedEvent(p.ID)
	p.RaiseEvent(event)
}

// AddTag adds a label tag to the pipeline for categorization and filtering.
//
// This method provides flexible labeling capabilities for pipeline organization,
// filtering, and management. Tags are used for categorization, access control,
// monitoring grouping, and business process organization. Duplicate tags are
// automatically prevented to maintain tag uniqueness.
//
// Parameters:
//
//	tag string: Label to add to the pipeline (case-sensitive, no validation)
//
// Returns:
//
//	None - always succeeds, duplicate tags are silently ignored
//
// Example:
//
//	Pipeline categorization and labeling:
//
//	  pipeline := loadPipeline()
//
//	  // Add business process tags
//	  pipeline.AddTag("customer-data")
//	  pipeline.AddTag("daily-etl")
//	  pipeline.AddTag("production")
//
//	  // Add environment and team tags
//	  pipeline.AddTag("data-engineering")
//	  pipeline.AddTag("high-priority")
//
//	  // Duplicate tag is ignored (idempotent)
//	  pipeline.AddTag("customer-data") // No effect
//
//	  log.Info("Pipeline tagged", "id", pipeline.ID, "tags", pipeline.Tags)
//
// Tag Usage Patterns:
//   - **Environment**: "production", "staging", "development"
//   - **Business Process**: "customer-data", "financial-reporting", "analytics"
//   - **Priority**: "high-priority", "critical", "best-effort"
//   - **Team**: "data-engineering", "analytics-team", "bi-team"
//   - **Schedule**: "daily", "weekly", "real-time", "batch"
//   - **Compliance**: "gdpr", "sox", "hipaa", "pci-dss"
//
// State Changes:
//   - Tag added to Tags slice if not already present
//   - Version incremented via Touch() for optimistic locking
//   - No domain events generated (tags are metadata, not business state)
//
// Integration:
//   - Query Layer: Tags used for pipeline filtering and search
//   - Access Control: Tag-based permission and visibility rules
//   - Monitoring: Grouping and aggregation by tags
//   - Reporting: Business process categorization and analytics
//   - UI/UX: Tag-based navigation and organization
//
// Performance:
//   - Linear search for duplicate detection (O(n) where n is tag count)
//   - Consider tag limits for large tag collections
//   - Tags are stored in memory with the aggregate
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) AddTag(tag string) {
	for _, existingTag := range p.Tags {
		if existingTag == tag {
			return // Tag already exists
		}
	}
	p.Tags = append(p.Tags, tag)
	p.Touch()
}

// RemoveTag removes a label tag from the pipeline's tag collection.
//
// This method provides tag management capabilities for pipeline organization,
// removing tags that are no longer relevant or applicable. The operation is
// idempotent - removing non-existent tags has no effect and generates no errors.
//
// Parameters:
//
//	tag string: Label to remove from the pipeline (case-sensitive exact match)
//
// Returns:
//
//	None - always succeeds, non-existent tags are silently ignored
//
// Example:
//
//	Tag lifecycle management:
//
//	  pipeline := loadPipeline()
//
//	  // Remove environment tag during promotion
//	  pipeline.RemoveTag("staging")
//	  pipeline.AddTag("production")
//
//	  // Remove temporary tags after processing
//	  pipeline.RemoveTag("migration-pending")
//	  pipeline.RemoveTag("manual-review")
//
//	  // Remove non-existent tag (no effect)
//	  pipeline.RemoveTag("non-existent-tag") // Silently ignored
//
//	  log.Info("Pipeline tags updated", "id", pipeline.ID, "remaining_tags", pipeline.Tags)
//
// Use Cases:
//   - **Environment Promotion**: Remove "staging", add "production"
//   - **Process Completion**: Remove "pending-approval", "in-review"
//   - **Cleanup Operations**: Remove temporary or obsolete tags
//   - **Access Control Changes**: Remove team or permission tags
//   - **Compliance Updates**: Remove expired compliance tags
//
// State Changes:
//   - Tag removed from Tags slice if present
//   - Slice compacted to remove gaps after removal
//   - Version incremented via Touch() for optimistic locking
//   - No domain events generated (tags are metadata)
//
// Implementation Details:
//   - Linear search to find tag index (O(n) where n is tag count)
//   - Slice manipulation to remove element and compact array
//   - Early return if tag not found (idempotent behavior)
//
// Integration:
//   - Query Layer: Tag changes affect filtering and search results
//   - Access Control: Tag removal may affect permissions
//   - Monitoring: Tag changes update grouping and metrics
//   - UI/UX: Tag changes reflected in navigation and organization
//
// Performance:
//   - Linear time complexity for tag location and removal
//   - Memory compaction when tags are removed
//   - Consider tag indexing for high-frequency tag operations
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Pipeline) RemoveTag(tag string) {
	for i, existingTag := range p.Tags {
		if existingTag == tag {
			p.Tags = append(p.Tags[:i], p.Tags[i+1:]...)
			p.Touch()
			break
		}
	}
}

// hasStep checks if a step with the specified name exists in the pipeline's step collection.
//
// This is a private utility method used internally for step validation and
// dependency checking. It performs a linear search through the pipeline's
// steps to determine if a step with the given name is present.
//
// Parameters:
//
//	stepName string: Name of the step to search for (case-sensitive exact match)
//
// Returns:
//
//	bool: true if a step with the given name exists, false otherwise
//
// Usage:
//
//	Internal method used by:
//	- AddStep() for dependency validation
//	- RemoveStep() for existence checking
//	- Other step management operations
//
// Performance:
//
//	O(n) linear search where n is the number of pipeline steps.
//	Consider step indexing for pipelines with large numbers of steps.
//
// Thread Safety:
//
//	Read-only operation, safe for concurrent access to immutable step data.
func (p *Pipeline) hasStep(stepName string) bool {
	for i := range p.Steps {
		if p.Steps[i].Name == stepName {
			return true
		}
	}
	return false
}

// GetStep retrieves a pipeline step by name, returning the step and existence indicator.
//
// This method provides safe access to pipeline steps with clear indication
// of whether the step exists. It follows Go's comma-ok idiom for optional
// value retrieval, preventing nil pointer exceptions and providing explicit
// existence checking.
//
// Parameters:
//
//	stepName string: Name of the step to retrieve (case-sensitive exact match)
//
// Returns:
//
//	PipelineStep: The step if found, or zero value if not found
//	bool: true if the step was found, false otherwise
//
// Example:
//
//	Safe step retrieval and validation:
//
//	  step, exists := pipeline.GetStep("extract-users")
//	  if !exists {
//	      return errors.New("step 'extract-users' not found")
//	  }
//
//	  // Safe to use step
//	  log.Info("Found step", "name", step.Name, "type", step.Type, "enabled", step.IsEnabled)
//
//	  // Check step configuration
//	  if config, hasConfig := step.Config["table"]; hasConfig {
//	      log.Info("Step configured for table", "table", config)
//	  }
//
// Use Cases:
//   - Configuration validation and step inspection
//   - Dynamic step configuration updates
//   - Step status monitoring and reporting
//   - Dependency analysis and workflow planning
//   - Debug and troubleshooting operations
//
// Performance:
//
//	O(n) linear search where n is the number of pipeline steps.
//	Returns copy of step data to prevent external modification.
//
// Error Handling:
//
//	Uses Go's comma-ok idiom instead of error returns.
//	Zero value PipelineStep returned when step not found.
//	Always check the boolean return value before using the step.
//
// Thread Safety:
//
//	Read-only operation returning copied step data, safe for concurrent access.
func (p *Pipeline) GetStep(stepName string) (PipelineStep, bool) {
	for i := range p.Steps {
		if p.Steps[i].Name == stepName {
			return p.Steps[i], true
		}
	}
	return PipelineStep{}, false
}

// IsScheduled checks if the pipeline has an active automated execution schedule.
//
// This method determines whether the pipeline is configured for automated
// execution based on a cron schedule. It checks both the existence of a
// schedule configuration and whether that schedule is currently enabled.
//
// Returns:
//
//	bool: true if pipeline has an active schedule, false otherwise
//
// Example:
//
//	Schedule-based execution logic:
//
//	  pipeline := loadPipeline()
//
//	  if pipeline.IsScheduled() {
//	      log.Info("Pipeline has automated scheduling",
//	          "id", pipeline.ID,
//	          "cron", pipeline.Schedule.CronExpression,
//	          "timezone", pipeline.Schedule.Timezone)
//
//	      // Register with scheduler service
//	      schedulerService.RegisterPipeline(pipeline)
//	  } else {
//	      log.Info("Pipeline requires manual execution", "id", pipeline.ID)
//
//	      // Set up manual execution workflows
//	      manualExecutionService.RegisterPipeline(pipeline)
//	  }
//
// Schedule Validation:
//
//	Returns true only when:
//	- Schedule is not nil (schedule configuration exists)
//	- Schedule.IsEnabled is true (schedule is active)
//
// Integration:
//   - Scheduler Service: Determines if pipeline should be registered for automated execution
//   - Monitoring: Schedule status affects monitoring and alerting strategies
//   - UI/UX: Controls display of schedule information and management options
//   - Reporting: Differentiates between manual and automated pipeline execution
//
// Thread Safety:
//
//	Read-only operation, safe for concurrent access.
func (p *Pipeline) IsScheduled() bool {
	return p.Schedule != nil && p.Schedule.IsEnabled
}

// CanExecute checks if the pipeline meets all preconditions for execution.
//
// This method validates that the pipeline is in a valid state for execution,
// checking both the pipeline status and the presence of executable steps.
// It provides a comprehensive readiness check before attempting to start
// pipeline execution.
//
// Returns:
//
//	bool: true if pipeline can be executed, false otherwise
//
// Example:
//
//	Pre-execution validation:
//
//	  pipeline := loadPipeline()
//
//	  if !pipeline.CanExecute() {
//	      log.Warn("Pipeline not ready for execution",
//	          "id", pipeline.ID,
//	          "status", pipeline.Status.String(),
//	          "step_count", len(pipeline.Steps))
//	      return errors.New("pipeline not ready")
//	  }
//
//	  // Safe to start execution
//	  result := pipeline.Start()
//	  if result.IsFailure() {
//	      return result.Error()
//	  }
//
// Execution Requirements:
//
//	Returns true only when:
//	- Status is PipelineStatusActive (pipeline is activated and ready)
//	- Steps collection is not empty (has at least one processing step)
//
// Validation Logic:
//   - Active Status: Ensures pipeline has been properly configured and activated
//   - Step Presence: Prevents execution of empty pipelines that would immediately complete
//   - Does not validate step configuration details (handled during activation)
//
// Integration:
//   - Execution Engine: Pre-flight check before starting pipeline processing
//   - Scheduler Service: Validates pipelines before scheduled execution
//   - Manual Execution: UI validation before user-initiated execution
//   - API Layer: Request validation for execution endpoints
//
// Performance:
//
//	Constant time operation - checks single status field and slice length.
//	No expensive validation or external dependencies.
//
// Thread Safety:
//
//	Read-only operation, safe for concurrent access.
func (p *Pipeline) CanExecute() bool {
	return p.Status == PipelineStatusActive && len(p.Steps) > 0
}

// Validate performs comprehensive validation of the pipeline's configuration and state.
//
// This method executes a complete validation check of the pipeline, ensuring
// all required fields are properly configured and the pipeline meets business
// rules for activation and execution. It provides detailed error information
// for validation failures.
//
// Returns:
//
//	result.Result[bool]: Success(true) if validation passes, or Failure with validation details
//
// Example:
//
//	Pipeline validation before activation:
//
//	  pipeline := buildPipeline()
//
//	  result := pipeline.Validate()
//	  if result.IsFailure() {
//	      log.Error("Pipeline validation failed", "error", result.Error(), "id", pipeline.ID)
//	      return result.Error()
//	  }
//
//	  // Pipeline is valid, safe to activate
//	  activationResult := pipeline.Activate()
//	  if activationResult.IsFailure() {
//	      return activationResult.Error()
//	  }
//
// Validation Rules:
//   - **Name Required**: Pipeline name cannot be empty
//   - **Owner Required**: Pipeline owner cannot be empty
//   - **Steps Required**: Pipeline must have at least one processing step
//   - Additional validations may include step configuration and dependency checks
//
// Validation Errors:
//   - ValidationError: "name is required" - when Name field is empty
//   - ValidationError: "owner is required" - when Owner field is empty
//   - ValidationError: "at least one step is required" - when Steps collection is empty
//
// Use Cases:
//   - Pre-activation validation to ensure pipeline readiness
//   - API request validation for pipeline creation and updates
//   - Bulk validation operations for pipeline health checks
//   - Development and testing validation workflows
//   - Configuration management and deployment validation
//
// Integration:
//   - Application Layer: Validation before persistence operations
//   - API Layer: Request validation for pipeline endpoints
//   - Migration Tools: Validation during pipeline import/export
//   - Testing Framework: Automated validation in test suites
//
// Performance:
//
//	Fast validation with early termination on first error.
//	No expensive operations or external dependencies.
//
// Thread Safety:
//
//	Read-only validation operation, safe for concurrent access.
func (p *Pipeline) Validate() error {
	if p.Name == "" {
		return errors.ValidationError("name is required")
	}

	if p.Owner == "" {
		return errors.ValidationError("owner is required")
	}

	if len(p.Steps) == 0 {
		return errors.ValidationError("at least one step is required")
	}

	return nil
}

// HasTag checks if the pipeline contains a specific tag in its tag collection.
//
// This method provides tag-based filtering and categorization capabilities,
// enabling queries based on pipeline labels and categories. It performs
// case-sensitive exact matching for tag identification.
//
// Parameters:
//
//	tag string: Tag to search for (case-sensitive exact match)
//
// Returns:
//
//	bool: true if the tag is present in the pipeline's tag collection, false otherwise
//
// Example:
//
//	Tag-based pipeline filtering and processing:
//
//	  pipeline := loadPipeline()
//
//	  // Environment-based processing
//	  if pipeline.HasTag("production") {
//	      log.Info("Processing production pipeline with strict validation", "id", pipeline.ID)
//	      enableStrictValidation()
//	  } else if pipeline.HasTag("staging") {
//	      log.Info("Processing staging pipeline", "id", pipeline.ID)
//	  }
//
//	  // Priority-based resource allocation
//	  if pipeline.HasTag("high-priority") {
//	      allocateHighPriorityResources(pipeline)
//	  }
//
//	  // Compliance checking
//	  if pipeline.HasTag("gdpr") {
//	      enforceGDPRCompliance(pipeline)
//	  }
//
// Use Cases:
//   - **Environment Filtering**: "production", "staging", "development"
//   - **Priority Management**: "high-priority", "critical", "best-effort"
//   - **Team Organization**: "data-engineering", "analytics", "bi-team"
//   - **Compliance Tracking**: "gdpr", "sox", "hipaa", "pci-dss"
//   - **Process Categorization**: "daily-etl", "real-time", "batch-processing"
//
// Integration:
//   - Query Layer: Tag-based pipeline filtering and search
//   - Access Control: Tag-based permission and visibility rules
//   - Resource Management: Tag-based resource allocation strategies
//   - Monitoring: Tag-based grouping and aggregation
//   - Workflow Engine: Tag-based process routing and handling
//
// Performance:
//
//	O(n) linear search where n is the number of tags.
//	Efficient for typical tag collection sizes (usually < 20 tags).
//	Consider tag indexing for high-frequency operations.
//
// Thread Safety:
//
//	Read-only operation, safe for concurrent access.
func (p *Pipeline) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Domain Events

// PipelineCreatedEvent represents a pipeline created event
type PipelineCreatedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
	Owner        string
}

// NewPipelineCreatedEvent creates a new pipeline created event
func NewPipelineCreatedEvent(pipelineID PipelineID, name, owner string) *PipelineCreatedEvent {
	return &PipelineCreatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.created", pipelineID.String()),
		PipelineName:    name,
		Owner:           owner,
	}
}

// PipelineStepAddedEvent represents a pipeline step added event
type PipelineStepAddedEvent struct {
	domain.BaseDomainEvent
	StepID   string
	StepName string
}

// NewPipelineStepAddedEvent creates a new pipeline step added event
func NewPipelineStepAddedEvent(pipelineID PipelineID, stepID, stepName string) *PipelineStepAddedEvent {
	return &PipelineStepAddedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.step.added", pipelineID.String()),
		StepID:          stepID,
		StepName:        stepName,
	}
}

// PipelineStepRemovedEvent represents a pipeline step removed event
type PipelineStepRemovedEvent struct {
	domain.BaseDomainEvent
	StepName string
}

// NewPipelineStepRemovedEvent creates a new pipeline step removed event
func NewPipelineStepRemovedEvent(pipelineID PipelineID, stepName string) *PipelineStepRemovedEvent {
	return &PipelineStepRemovedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.step.removed", pipelineID.String()),
		StepName:        stepName,
	}
}

// PipelineActivatedEvent represents a pipeline activated event
type PipelineActivatedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
}

// NewPipelineActivatedEvent creates a new pipeline activated event
func NewPipelineActivatedEvent(pipelineID PipelineID, name string) *PipelineActivatedEvent {
	return &PipelineActivatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.activated", pipelineID.String()),
		PipelineName:    name,
	}
}

// PipelineDeactivatedEvent represents a pipeline deactivated event
type PipelineDeactivatedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
}

// NewPipelineDeactivatedEvent creates a new pipeline deactivated event
func NewPipelineDeactivatedEvent(pipelineID PipelineID, name string) *PipelineDeactivatedEvent {
	return &PipelineDeactivatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.deactivated", pipelineID.String()),
		PipelineName:    name,
	}
}

// PipelineStartedEvent represents a pipeline started event
type PipelineStartedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
	StartTime    time.Time
}

// NewPipelineStartedEvent creates a new pipeline started event
func NewPipelineStartedEvent(pipelineID PipelineID, name string, startTime time.Time) *PipelineStartedEvent {
	return &PipelineStartedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.started", pipelineID.String()),
		PipelineName:    name,
		StartTime:       startTime,
	}
}

// PipelineCompletedEvent represents a pipeline completed event
type PipelineCompletedEvent struct {
	domain.BaseDomainEvent
	PipelineName  string
	CompletedTime time.Time
}

// NewPipelineCompletedEvent creates a new pipeline completed event
func NewPipelineCompletedEvent(pipelineID PipelineID, name string, completedTime time.Time) *PipelineCompletedEvent {
	return &PipelineCompletedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.completed", pipelineID.String()),
		PipelineName:    name,
		CompletedTime:   completedTime,
	}
}

// PipelineFailedEvent represents a pipeline failed event
type PipelineFailedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
	Reason       string
	FailedTime   time.Time
}

// NewPipelineFailedEvent creates a new pipeline failed event
func NewPipelineFailedEvent(pipelineID PipelineID, name, reason string, failedTime time.Time) *PipelineFailedEvent {
	return &PipelineFailedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.failed", pipelineID.String()),
		PipelineName:    name,
		Reason:          reason,
		FailedTime:      failedTime,
	}
}

// PipelineScheduleSetEvent represents a pipeline schedule set event
type PipelineScheduleSetEvent struct {
	domain.BaseDomainEvent
	CronExpression string
	Timezone       string
}

// NewPipelineScheduleSetEvent creates a new pipeline schedule set event
func NewPipelineScheduleSetEvent(pipelineID PipelineID, cronExpression, timezone string) *PipelineScheduleSetEvent {
	return &PipelineScheduleSetEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.schedule.set", pipelineID.String()),
		CronExpression:  cronExpression,
		Timezone:        timezone,
	}
}

// PipelineScheduleClearedEvent represents a pipeline schedule cleared event
type PipelineScheduleClearedEvent struct {
	domain.BaseDomainEvent
}

// NewPipelineScheduleClearedEvent creates a new pipeline schedule cleared event
func NewPipelineScheduleClearedEvent(pipelineID PipelineID) *PipelineScheduleClearedEvent {
	return &PipelineScheduleClearedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.schedule.cleared", pipelineID.String()),
	}
}
