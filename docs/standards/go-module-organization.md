# Go Module Organization Standard

**Comprehensive guide to Go module organization in FlexCore**

**Version**: 0.9.0
**Last Updated**: 2025-08-01  
**Authority**: FLEXT Development Team  
**Scope**: FlexCore Go service and FLEXT ecosystem Go projects

---

## 🎯 Overview

FlexCore implements a **professional Go module organization** following **Clean Architecture**, **Domain-Driven Design**, and **enterprise-grade patterns** aligned with the broader FLEXT ecosystem standards and Go community best practices.

### **Module Organization Philosophy**

1. **Clean Architecture Compliance** - Clear separation of concerns with strict dependency rules
2. **Domain-Driven Design** - Business domain modeling with rich aggregates and bounded contexts
3. **Go Idioms** - Following Go community standards and effective Go patterns
4. **FLEXT Integration** - Consistent patterns across the 33-project FLEXT ecosystem
5. **Enterprise Scalability** - Maintainable, testable, and performant organization

---

## 📁 Current Module Structure

### **FlexCore Package Structure**

```
flexcore/
├── main.go                          # Application entry point
├── go.mod                           # Module definition and dependencies
├── go.sum                           # Dependency checksums
├── Makefile                         # Build and development automation
├── Dockerfile                       # Container definition
├── docker-compose.yml               # Development environment
├── .env.example                     # Environment configuration template
├── cmd/                             # Application commands
│   └── server/                      # HTTP server command
│       ├── main.go                  # Server entry point
│       └── config.go                # Server configuration
├── internal/                            # Private application packages
│   ├── adapters/                        # Adapter layer (Clean Architecture)
│   │   ├── primary/                     # Primary adapters (driving)
│   │   │   └── http/                    # HTTP controllers and handlers
│   │   │       ├── server.go            # HTTP server configuration
│   │   │       ├── server_test.go       # HTTP server tests
│   │   │       └── gateway_config.json  # Gateway configuration
│   │   └── secondary/                   # Secondary adapters (driven)
│   │       └── persistence/             # Persistence adapters
│   │           └── memory/              # In-memory implementations
│   │               └── repository.go    # Memory repository implementation
│   ├── app/                            # Application layer (Clean Architecture)
│   │   ├── application.go              # ⚠️ VIOLATION: Contains HTTP server
│   │   ├── application_test.go         # Application tests
│   │   ├── commands/                   # Command handling (CQRS)
│   │   │   ├── command_bus.go          # Command bus implementation
│   │   │   ├── command_bus_test.go     # Command bus tests
│   │   │   └── pipeline_commands.go    # Pipeline command handlers
│   │   └── queries/                    # Query handling (CQRS)
│   │       ├── pipeline_queries.go     # Pipeline query handlers
│   │       └── query_bus.go            # Query bus implementation
│   ├── application/                    # ⚠️ DUPLICATE: Application services
│   │   └── services/                   # Application services
│   │       ├── pipeline_execution_services.go # Pipeline execution
│   │       └── workflow_service.go     # Workflow orchestration
│   ├── domain/                         # Domain layer (Clean Architecture)
│   │   ├── base.go                     # Base domain types and patterns
│   │   ├── base_test.go                # Base domain tests
│   │   ├── core.go                     # Core domain services
│   │   └── entities/                   # Domain entities and aggregates
│   │       ├── pipeline.go             # Pipeline aggregate root
│   │       ├── pipeline_test.go        # Pipeline entity tests
│   │       └── plugin.go               # Plugin entity
│   └── infrastructure/                 # Infrastructure layer
│       ├── ⚠️ MULTIPLE CQRS IMPLEMENTATIONS (architectural violation)
│       ├── command_bus.go              # Function-based command bus
│       ├── event_store.go              # In-memory event store
│       ├── postgres_event_store.go     # PostgreSQL event store
│       ├── redis_coordinator.go        # Redis distributed coordination
│       ├── redis_coordinator_real.go   # Real Redis coordinator
│       ├── plugin_loader.go            # Plugin loading system
│       ├── plugin_execution_services.go # Plugin execution infrastructure
│       ├── real_endpoints.go           # Real HTTP endpoints
│       ├── real_endpoints_*.go         # Endpoint implementations
│       ├── server.go                   # Infrastructure server
│       ├── cqrs/                       # CQRS implementation (SQLite-based)
│       │   ├── cqrs_bus.go             # CQRS bus with SQLite
│       │   └── examples.go             # CQRS examples
│       ├── eventsourcing/              # Event sourcing infrastructure
│       │   ├── event_storage.go        # Event storage interface
│       │   ├── event_store.go          # Event store implementation
│       │   ├── event_types.go          # Event type definitions
│       │   └── snapshot_service.go     # Snapshot management
│       ├── middleware/                 # HTTP middleware
│       │   └── logging.go              # Logging middleware
│       └── observability/              # Observability infrastructure
│           ├── metrics.go              # Prometheus metrics
│           ├── monitor.go              # System monitoring
│           └── tracing.go              # Distributed tracing
├── pkg/                                # Public packages (reusable)
│   ├── adapter/                        # Adapter utilities
│   │   ├── base_adapter.go             # Base adapter patterns
│   │   ├── base_adapter_test.go        # Base adapter tests
│   │   ├── builder.go                  # Adapter builder pattern
│   │   └── types.go                    # Adapter type definitions
│   ├── config/                         # Configuration management
│   │   ├── config.go                   # Configuration structures
│   │   └── config_test.go              # Configuration tests
│   ├── cqrs/                           # CQRS utilities
│   │   ├── bus_validation_utilities.go # CQRS validation utilities
│   │   └── stats_utilities.go          # CQRS statistics utilities
│   ├── errors/                         # Error handling utilities
│   │   ├── errors.go                   # Custom error types
│   │   └── errors_test.go              # Error handling tests
│   ├── logging/                        # Logging utilities
│   │   ├── logger.go                   # Logger interface and implementation
│   │   ├── logger_test.go              # Logger tests
│   │   └── types.go                    # Logging type definitions
│   ├── patterns/                       # Common design patterns
│   │   ├── options.go                  # Options pattern implementation
│   │   ├── query_builder.go            # Query builder pattern
│   │   ├── railway.go                  # Railway-oriented programming
│   │   └── railway_test.go             # Railway pattern tests
│   ├── plugin/                         # Plugin framework
│   │   ├── data_processor.go           # Data processor plugin
│   │   ├── data_transformer.go         # Data transformer plugin
│   │   ├── data_transformers.go        # Data transformers collection
│   │   ├── enrichment_service.go       # Data enrichment plugin
│   │   ├── gob_registration.go         # Gob serialization registration
│   │   ├── json_processor.go           # JSON processor plugin
│   │   ├── main_utilities.go           # Main plugin utilities
│   │   ├── plugin_utils.go             # Plugin utilities
│   │   ├── postgres_processor.go       # PostgreSQL processor plugin
│   │   ├── processing_stats.go         # Processing statistics
│   │   ├── types.go                    # Plugin type definitions
│   │   ├── types_test.go               # Plugin type tests
│   │   └── validation_service.go       # Plugin validation service
│   └── result/                         # Result pattern implementation
│       ├── result.go                   # Result type with generics
│       └── result_test.go              # Result pattern tests
├── plugins/                            # Plugin implementations
│   ├── data-processor/                 # Data processor plugin
│   │   ├── main.go                     # Plugin main implementation
│   │   └── data-processor              # Built plugin binary
│   ├── data-transformer/               # Data transformer plugin
│   │   └── main.go                     # Plugin main implementation
│   ├── json-processor/                 # JSON processor plugin
│   │   └── main.go                     # Plugin main implementation
│   ├── postgres-processor/             # PostgreSQL processor plugin
│   │   ├── main.go                     # Plugin main implementation
│   │   └── postgres-processor          # Built plugin binary
│   └── simple-processor/               # Simple processor plugin
│       ├── main.go                     # Plugin main implementation
│       └── simple-processor            # Built plugin binary
├── infrastructure/                     # Infrastructure components
│   └── di/                            # Dependency injection
│       └── container.go               # DI container implementation
├── test/                              # Test utilities and integration tests
│   ├── e2e/                           # End-to-end tests
│   │   └── api_test.go                # API end-to-end tests
│   └── integration/                   # Integration tests
│       └── flexcore_integration_test.go # FlexCore integration tests
├── go.mod                             # Go module definition
├── go.sum                             # Go module checksums
├── Makefile                           # Build automation
├── README.md                          # Project documentation
├── CLAUDE.md                          # Claude development guidance
└── docs/                              # Documentation directory
    ├── README.md                      # Documentation index
    ├── TODO.md                        # ⚠️ CRITICAL: Architecture issues
    ├── api-reference.md               # API documentation
    ├── architecture/                  # Architecture documentation
    ├── integration/                   # Integration guides
    └── standards/                     # Standards documentation (this file)
```

---

## 🏗️ Architecture Patterns

### **Clean Architecture Implementation**

#### **Target Layer Organization** (Post-Refactoring)

```
📦 cmd/ (Application Entry Points)
├── 🚀 Main Applications
│   ├── flexcore/main.go           # Primary service entry point
│   ├── REDACTED_LDAP_BIND_PASSWORD/main.go              # Administrative tools (planned)
│   └── migrate/main.go            # Database migration tools (planned)

📦 internal/ (Private Application Code)
├── 🎯 adapters/ (Adapter Layer)
│   ├── primary/ (Driving Adapters)
│   │   ├── http/                  # HTTP/REST API adapters
│   │   ├── grpc/                  # gRPC API adapters (planned)
│   │   └── cli/                   # Command-line adapters (planned)
│   └── secondary/ (Driven Adapters)
│       ├── persistence/           # Database adapters
│       ├── messaging/             # Message queue adapters (planned)
│       └── external/              # External service adapters
│
├── 🏢 app/ (Application Layer)
│   ├── usecases/                  # Use case implementations
│   │   ├── pipeline/              # Pipeline management use cases
│   │   ├── plugin/                # Plugin management use cases
│   │   └── monitoring/            # System monitoring use cases
│   ├── commands/                  # Command handlers (CQRS)
│   ├── queries/                   # Query handlers (CQRS)
│   └── services/                  # Application services
│
├── 🏛️ domain/ (Domain Layer)
│   ├── entities/                  # Domain entities and aggregates
│   │   ├── pipeline/              # Pipeline aggregate
│   │   ├── plugin/                # Plugin aggregate
│   │   └── shared/                # Shared domain entities
│   ├── services/                  # Domain services
│   │   ├── pipeline_orchestrator.go # Pipeline orchestration logic
│   │   ├── plugin_security.go     # Plugin security service
│   │   └── event_coordinator.go   # Event coordination service
│   ├── events/                    # Domain events
│   ├── repositories/              # Repository interfaces
│   └── valueobjects/              # Value objects
│
└── 🔧 infrastructure/ (Infrastructure Layer)
    ├── persistence/               # Database implementations
    │   ├── postgres/              # PostgreSQL implementations
    │   └── redis/                 # Redis implementations
    ├── messaging/                 # Message queue implementations
    ├── external/                  # External service clients
    ├── monitoring/                # Observability implementations
    └── config/                    # Configuration management

📦 pkg/ (Public Packages)
├── 🛠️ Core Utilities
│   ├── result/                    # Result pattern for error handling
│   ├── errors/                    # Custom error types
│   ├── logging/                   # Logging interfaces and utilities
│   └── config/                    # Configuration utilities
├── 🔄 Patterns
│   ├── cqrs/                      # CQRS utilities and interfaces
│   ├── eventsourcing/             # Event sourcing patterns
│   ├── patterns/                  # Common design patterns
│   └── adapter/                   # Adapter pattern utilities
└── 🔌 Plugin Framework
    ├── plugin/                    # Plugin interfaces and utilities
    ├── security/                  # Plugin security framework
    └── runtime/                   # Plugin runtime management
```

### **Domain-Driven Design Patterns**

#### **Bounded Contexts**

1. **Pipeline Management** (`internal/domain/entities/pipeline/`) - Pipeline lifecycle and orchestration
2. **Plugin Management** (`internal/domain/entities/plugin/`) - Plugin lifecycle and execution
3. **Event Processing** (`internal/domain/events/`) - Event sourcing and CQRS coordination
4. **System Monitoring** (`internal/domain/entities/monitoring/`) - Health and metrics management

#### **Domain Services** (Target Implementation)

```go
// internal/domain/services/pipeline_orchestrator.go
package services

import (
    "context"
    "github.com/flext-sh/flexcore/internal/domain/entities/pipeline"
    "github.com/flext-sh/flexcore/internal/domain/entities/plugin"
    "github.com/flext-sh/flexcore/pkg/result"
)

// PipelineOrchestrator coordinates complex pipeline execution workflows
type PipelineOrchestrator struct {
    pipelineRepo pipeline.Repository
    pluginRepo   plugin.Repository
    eventBus     EventBus
    logger       LoggerInterface
}

// NewPipelineOrchestrator creates a new pipeline orchestrator
func NewPipelineOrchestrator(
    pipelineRepo pipeline.Repository,
    pluginRepo plugin.Repository,
    eventBus EventBus,
    logger LoggerInterface,
) *PipelineOrchestrator {
    return &PipelineOrchestrator{
        pipelineRepo: pipelineRepo,
        pluginRepo:   pluginRepo,
        eventBus:     eventBus,
        logger:       logger,
    }
}

// ExecutePipeline orchestrates complete pipeline execution with error handling
func (o *PipelineOrchestrator) ExecutePipeline(
    ctx context.Context,
    pipelineID pipeline.ID,
) result.Result[*pipeline.ExecutionResult] {
    // Domain service implements rich business logic
    pipeline, err := o.pipelineRepo.GetByID(ctx, pipelineID)
    if err != nil {
        return result.Failure[*pipeline.ExecutionResult](err)
    }

    // Validate pipeline can be executed
    validationResult := pipeline.ValidateForExecution()
    if validationResult.IsFailure() {
        return result.Failure[*pipeline.ExecutionResult](validationResult.Error())
    }

    // Execute pipeline steps with proper error handling and events
    executionResult := o.executeStepsSequentially(ctx, pipeline)

    // Publish domain events based on execution result
    if executionResult.IsSuccess() {
        event := pipeline.CreateCompletedEvent(executionResult.Value())
        o.eventBus.Publish(ctx, event)
    } else {
        event := pipeline.CreateFailedEvent(executionResult.Error())
        o.eventBus.Publish(ctx, event)
    }

    return executionResult
}
```

#### **Rich Domain Entities** (Target Implementation)

```go
// internal/domain/entities/pipeline/pipeline.go
package pipeline

import (
    "time"
    "github.com/flext-sh/flexcore/internal/domain/events"
    "github.com/flext-sh/flexcore/pkg/result"
    "github.com/google/uuid"
)

// ID represents a pipeline identifier as a value object
type ID struct {
    value string
}

// NewID creates a new pipeline ID
func NewID() ID {
    return ID{value: uuid.New().String()}
}

// String returns the string representation
func (id ID) String() string {
    return id.value
}

// Pipeline represents a data processing pipeline aggregate root
type Pipeline struct {
    // Aggregate root fields
    id           ID
    version      int64
    domainEvents []events.DomainEvent

    // Pipeline-specific fields
    name         string
    description  string
    status       Status
    steps        []Step
    owner        string
    schedule     *Schedule
    lastRunAt    *time.Time
    nextRunAt    *time.Time
    createdAt    time.Time
    updatedAt    time.Time
}

// NewPipeline creates a new pipeline with rich validation
func NewPipeline(name, description, owner string) result.Result[*Pipeline] {
    // Rich domain validation
    if err := validatePipelineName(name); err != nil {
        return result.Failure[*Pipeline](err)
    }

    if err := validateOwner(owner); err != nil {
        return result.Failure[*Pipeline](err)
    }

    id := NewID()
    now := time.Now()

    pipeline := &Pipeline{
        id:           id,
        version:      1,
        domainEvents: make([]events.DomainEvent, 0),
        name:         name,
        description:  description,
        status:       StatusDraft,
        steps:        make([]Step, 0),
        owner:        owner,
        createdAt:    now,
        updatedAt:    now,
    }

    // Generate domain event
    event := events.NewPipelineCreatedEvent(id, name, owner, now)
    pipeline.raiseEvent(event)

    return result.Success(pipeline)
}

// AddStep adds a step with rich business logic validation
func (p *Pipeline) AddStep(step Step) result.Result[void] {
    // Business rule: Cannot add steps to running pipelines
    if p.status == StatusRunning {
        return result.Failure[void](ErrCannotModifyRunningPipeline)
    }

    // Business rule: Validate step dependencies
    if err := p.validateStepDependencies(step); err != nil {
        return result.Failure[void](err)
    }

    // Business rule: Check for naming conflicts
    if p.hasStepWithName(step.Name()) {
        return result.Failure[void](ErrDuplicateStepName)
    }

    // Add step and update aggregate
    p.steps = append(p.steps, step)
    p.version++
    p.updatedAt = time.Now()

    // Generate domain event
    event := events.NewPipelineStepAddedEvent(p.id, step.ID(), step.Name(), p.updatedAt)
    p.raiseEvent(event)

    return result.Success(void{})
}

// Start initiates pipeline execution with comprehensive validation
func (p *Pipeline) Start() result.Result[*ExecutionContext] {
    // Business rule validation
    if p.status != StatusActive {
        return result.Failure[*ExecutionContext](ErrPipelineNotActive)
    }

    if len(p.steps) == 0 {
        return result.Failure[*ExecutionContext](ErrPipelineHasNoSteps)
    }

    // Validate all steps are ready for execution
    if validationResult := p.validateAllStepsReady(); validationResult.IsFailure() {
        return result.Failure[*ExecutionContext](validationResult.Error())
    }

    // Create execution context
    executionContext := NewExecutionContext(p.id, p.steps)

    // Update pipeline state
    p.status = StatusRunning
    now := time.Now()
    p.lastRunAt = &now
    p.version++
    p.updatedAt = now

    // Generate domain event
    event := events.NewPipelineStartedEvent(p.id, p.name, executionContext.ID(), now)
    p.raiseEvent(event)

    return result.Success(executionContext)
}

// ValidateForExecution performs comprehensive pre-execution validation
func (p *Pipeline) ValidateForExecution() result.Result[ValidationReport] {
    report := NewValidationReport(p.id)

    // Validate pipeline state
    if p.status != StatusActive {
        report.AddError("Pipeline must be in Active status for execution")
    }

    // Validate steps
    if len(p.steps) == 0 {
        report.AddError("Pipeline must have at least one step")
    }

    // Validate step dependencies
    for _, step := range p.steps {
        if err := p.validateStepDependencies(step); err != nil {
            report.AddError(fmt.Sprintf("Step %s has invalid dependencies: %v", step.Name(), err))
        }
    }

    // Validate resource requirements
    if err := p.validateResourceRequirements(); err != nil {
        report.AddError(fmt.Sprintf("Resource validation failed: %v", err))
    }

    if report.HasErrors() {
        return result.Failure[ValidationReport](ErrPipelineValidationFailed)
    }

    return result.Success(report)
}

// GetDomainEvents returns uncommitted domain events
func (p *Pipeline) GetDomainEvents() []events.DomainEvent {
    return p.domainEvents
}

// MarkEventsAsCommitted clears domain events after persistence
func (p *Pipeline) MarkEventsAsCommitted() {
    p.domainEvents = make([]events.DomainEvent, 0)
}

// Private methods for rich domain behavior

func (p *Pipeline) raiseEvent(event events.DomainEvent) {
    p.domainEvents = append(p.domainEvents, event)
}

func (p *Pipeline) validateStepDependencies(step Step) error {
    for _, depName := range step.Dependencies() {
        if !p.hasStepWithName(depName) {
            return ErrStepDependencyNotFound.WithDetails(map[string]interface{}{
                "step": step.Name(),
                "missing_dependency": depName,
            })
        }
    }
    return nil
}

func (p *Pipeline) hasStepWithName(name string) bool {
    for _, step := range p.steps {
        if step.Name() == name {
            return true
        }
    }
    return false
}

func (p *Pipeline) validateAllStepsReady() result.Result[void] {
    for _, step := range p.steps {
        if readyResult := step.ValidateReady(); readyResult.IsFailure() {
            return result.Failure[void](readyResult.Error())
        }
    }
    return result.Success(void{})
}

func (p *Pipeline) validateResourceRequirements() error {
    // Complex business logic for resource validation
    totalCPU := 0.0
    totalMemory := int64(0)

    for _, step := range p.steps {
        requirements := step.ResourceRequirements()
        totalCPU += requirements.CPU
        totalMemory += requirements.Memory
    }

    if totalCPU > MaxAllowedCPU {
        return ErrExceedsMaxCPU
    }

    if totalMemory > MaxAllowedMemory {
        return ErrExceedsMaxMemory
    }

    return nil
}
```

---

## 📋 Go Documentation Standards

### **Package-Level Documentation**

#### **Package Comment Standards**

```go
// Package pipeline provides enterprise-grade pipeline management capabilities
// for the FlexCore data integration platform, implementing Clean Architecture
// and Domain-Driven Design patterns for scalable pipeline orchestration.
//
// This package serves as the core pipeline domain within FlexCore's bounded context
// for data processing workflow management. It implements rich domain models with
// comprehensive business rule validation, event sourcing for complete audit trails,
// and CQRS patterns for optimal read/write separation.
//
// Key Features:
//   - Rich domain model with comprehensive business rule validation
//   - Event sourcing with immutable domain events for complete audit trails
//   - CQRS separation for optimized command and query operations
//   - Plugin orchestration with secure execution boundaries
//   - Resource management with configurable limits and monitoring
//
// Architecture:
//   This package follows Clean Architecture principles with strict dependency
//   inversion. Domain entities contain rich business logic while remaining
//   independent of infrastructure concerns. All cross-cutting concerns are
//   handled through dependency injection interfaces.
//
// Integration:
//   - Uses FlexCore result patterns for consistent error handling
//   - Integrates with FlexCore event bus for domain event publishing
//   - Coordinates with plugin domain for execution orchestration
//   - Connects with monitoring domain for observability
//
// Example:
//   Basic pipeline creation and execution:
//
//     package main
//
//     import (
//         "context"
//         "github.com/flext-sh/flexcore/internal/domain/entities/pipeline"
//         "github.com/flext-sh/flexcore/pkg/result"
//     )
//
//     func main() {
//         // Create new pipeline with validation
//         pipelineResult := pipeline.NewPipeline(
//             "data-processing-pipeline",
//             "Processes customer data from Oracle to warehouse",
//             "data-team",
//         )
//
//         if pipelineResult.IsFailure() {
//             log.Fatal("Pipeline creation failed:", pipelineResult.Error())
//         }
//
//         p := pipelineResult.Value()
//
//         // Add processing steps
//         extractStep := pipeline.NewStep("extract-customers", "oracle-extractor")
//         if addResult := p.AddStep(extractStep); addResult.IsFailure() {
//             log.Fatal("Failed to add step:", addResult.Error())
//         }
//
//         // Activate and start pipeline
//         if activateResult := p.Activate(); activateResult.IsFailure() {
//             log.Fatal("Failed to activate pipeline:", activateResult.Error())
//         }
//
//         executionResult := p.Start()
//         if executionResult.IsFailure() {
//             log.Fatal("Failed to start pipeline:", executionResult.Error())
//         }
//
//         log.Println("Pipeline started successfully:", executionResult.Value().ID())
//     }
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package pipeline
```

#### **Struct Documentation Standards**

```go
// Pipeline represents a data processing pipeline aggregate root with rich domain behavior.
//
// Pipeline implements comprehensive business logic for data processing workflow management
// within the FlexCore ecosystem. As an aggregate root in the DDD pattern, it maintains
// consistency boundaries for all pipeline-related operations and generates domain events
// for all state changes to support event sourcing and audit requirements.
//
// The Pipeline aggregate encapsulates complex business rules around pipeline execution,
// step dependencies, resource management, and scheduling while maintaining loose coupling
// with infrastructure concerns through dependency injection interfaces.
//
// Business Rules:
//   - Pipelines must have at least one step before activation
//   - Steps cannot be modified while pipeline is running
//   - Step dependencies must form a valid directed acyclic graph (DAG)
//   - Resource requirements cannot exceed configured system limits
//   - Only active pipelines can be started for execution
//   - Failed pipelines can be restarted only after error resolution
//
// Domain Events Generated:
//   - PipelineCreated: When new pipeline is instantiated
//   - PipelineActivated: When pipeline is marked as ready for execution
//   - PipelineStarted: When pipeline execution begins
//   - PipelineStepAdded: When new step is added to pipeline
//   - PipelineCompleted: When pipeline execution finishes successfully
//   - PipelineFailed: When pipeline execution encounters failure
//
// State Transitions:
//   Draft → Active → Running → {Completed|Failed} → Archive
//
//   Valid transitions:
//   - Draft → Active (via Activate())
//   - Active → Running (via Start())
//   - Running → Completed (via Complete())
//   - Running → Failed (via Fail())
//   - Any → Archive (via Archive())
//
// Example:
//   Creating and configuring a data processing pipeline:
//
//     // Create pipeline with business validation
//     pipeline, err := pipeline.NewPipeline(
//         "customer-sync-pipeline",
//         "Synchronizes customer data between Oracle and warehouse",
//         "data-engineering-team",
//     )
//     if err != nil {
//         return fmt.Errorf("pipeline creation failed: %w", err)
//     }
//
//     // Add extraction step
//     extractStep := pipeline.NewStep("extract", "oracle-customer-extractor")
//     extractStep.SetResourceRequirements(pipeline.ResourceRequirements{
//         CPU:    0.5,
//         Memory: 1024 * 1024 * 1024, // 1GB
//     })
//
//     if result := pipeline.AddStep(extractStep); result.IsFailure() {
//         return result.Error()
//     }
//
//     // Add transformation step with dependency
//     transformStep := pipeline.NewStep("transform", "customer-normalizer")
//     transformStep.AddDependency("extract")
//
//     if result := pipeline.AddStep(transformStep); result.IsFailure() {
//         return result.Error()
//     }
//
//     // Activate pipeline for execution
//     if result := pipeline.Activate(); result.IsFailure() {
//         return result.Error()
//     }
//
// Architecture:
//   Pipeline follows Clean Architecture with rich domain model pattern.
//   Infrastructure dependencies are injected through interfaces, and all
//   external communication occurs through domain events published to the
//   event bus. Business logic is completely encapsulated within the aggregate.
//
// Performance:
//   Pipeline operations are optimized for frequent reads with occasional writes.
//   Domain events are generated in-memory and batched for persistence efficiency.
//   Step validation uses cached dependency graphs to minimize computation overhead.
//
// Thread Safety:
//   Pipeline instances are not thread-safe and should be accessed from a single
//   goroutine or protected with external synchronization. Repository implementations
//   handle concurrent access through optimistic locking with version numbers.
//
// Integration:
//   - Event Bus: Publishes domain events for cross-aggregate coordination
//   - Plugin Domain: Coordinates plugin execution through domain services
//   - Monitoring Domain: Provides metrics and health status information
//   - Resource Manager: Validates and tracks resource utilization
type Pipeline struct {
    // ... field definitions
}
```

#### **Method Documentation Standards**

```go
// AddStep adds a new processing step to the pipeline with comprehensive validation.
//
// This method implements complex business logic for step addition including dependency
// validation, resource requirement verification, and state consistency checks. The
// operation is atomic - either the step is successfully added with all validations
// passing, or the pipeline remains unchanged with detailed error information.
//
// Business Rules Enforced:
//   - Pipeline must not be in running state for step modification
//   - Step names must be unique within the pipeline scope
//   - Step dependencies must reference existing steps (no forward references)
//   - Dependency graph must remain acyclic (no circular dependencies)
//   - Combined resource requirements cannot exceed system limits
//   - Step configuration must be valid for the specified step type
//
// Parameters:
//   step Step: The processing step to add to the pipeline. Must be a valid Step
//             instance with proper configuration. Step dependencies will be
//             validated against existing pipeline steps.
//
// Returns:
//   result.Result[void]: Success result if step is added successfully, or failure
//                       result with detailed error information. Possible errors:
//                       - ErrCannotModifyRunningPipeline: Pipeline is currently executing
//                       - ErrDuplicateStepName: Step name conflicts with existing step
//                       - ErrStepDependencyNotFound: Referenced dependency does not exist
//                       - ErrCircularDependency: Adding step would create dependency cycle
//                       - ErrResourceLimitExceeded: Step would exceed resource limits
//                       - ErrInvalidStepConfiguration: Step configuration is invalid
//
// Side Effects:
//   - Generates PipelineStepAddedEvent domain event for event sourcing
//   - Increments pipeline version number for optimistic locking
//   - Updates pipeline modification timestamp
//   - Rebuilds internal dependency graph cache
//
// Example:
//   Adding a data transformation step with dependencies:
//
//     // Create transformation step
//     transformStep := pipeline.NewStep("customer-transform", "customer-normalizer")
//     transformStep.AddDependency("customer-extract")
//     transformStep.SetResourceRequirements(pipeline.ResourceRequirements{
//         CPU:    0.3,
//         Memory: 512 * 1024 * 1024, // 512MB
//     })
//     transformStep.SetConfiguration(map[string]interface{}{
//         "normalization_rules": "customer-v2.yaml",
//         "error_handling":      "strict",
//     })
//
//     // Add step with validation
//     result := p.AddStep(transformStep)
//     if result.IsFailure() {
//         switch {
//         case errors.Is(result.Error(), pipeline.ErrDuplicateStepName):
//             log.Error("Step name already exists in pipeline")
//         case errors.Is(result.Error(), pipeline.ErrStepDependencyNotFound):
//             log.Error("Step dependency 'customer-extract' not found")
//         case errors.Is(result.Error(), pipeline.ErrResourceLimitExceeded):
//             log.Error("Step would exceed pipeline resource limits")
//         default:
//             log.Error("Unexpected error adding step:", result.Error())
//         }
//         return result.Error()
//     }
//
//     log.Info("Step added successfully", "step", transformStep.Name())
//
// Architecture:
//   Method follows Domain-Driven Design with rich business logic encapsulation.
//   Validation is performed using domain services and value objects. State changes
//   are atomic and generate appropriate domain events for external coordination.
//
// Performance:
//   Dependency validation has O(V + E) complexity where V is the number of steps
//   and E is the number of dependencies. For typical pipelines (< 100 steps),
//   performance impact is negligible. Large pipelines may benefit from dependency
//   graph caching optimizations.
//
// Thread Safety:
//   This method is not thread-safe. External synchronization or single-threaded
//   access is required. Repository implementations provide optimistic locking
//   for concurrent modification detection.
func (p *Pipeline) AddStep(step Step) result.Result[void] {
    // Implementation...
}
```

---

## 🔧 Import Organization

### **Import Standards**

#### **Import Order (Following Go Community Standards)**

```go
// Package pipeline provides enterprise pipeline management
package pipeline

// 1. Standard library imports (alphabetical)
import (
    "context"
    "fmt"
    "sort"
    "strings"
    "sync"
    "time"
)

// 2. Third-party imports (alphabetical by domain)
import (
    "github.com/google/uuid"
    "github.com/pkg/errors"
    "go.uber.org/zap"
)

// 3. FlexCore internal imports (by layer, then alphabetical)
import (
    // Domain layer imports (current layer)
    "github.com/flext-sh/flexcore/internal/domain/events"
    "github.com/flext-sh/flexcore/internal/domain/valueobjects"

    // Package layer imports (utilities)
    "github.com/flext-sh/flexcore/pkg/errors"
    "github.com/flext-sh/flexcore/pkg/logging"
    "github.com/flext-sh/flexcore/pkg/result"
)

// Note: Imports from application or infrastructure layers are NOT allowed
// in domain layer - this enforces Clean Architecture dependency rules
```

#### **Package Exports and Internal APIs**

```go
// Package pipeline exposes domain entities and interfaces for pipeline management
package pipeline

// Public API - exported types and functions
type (
    // Core domain types
    Pipeline struct { /* ... */ }
    Step     struct { /* ... */ }
    Status   int

    // Value objects
    ID               struct { /* ... */ }
    ResourceRequirements struct { /* ... */ }
    ExecutionContext struct { /* ... */ }

    // Interfaces for dependency injection
    Repository interface { /* ... */ }
    EventBus   interface { /* ... */ }
    Logger     interface { /* ... */ }
)

// Public constants
const (
    StatusDraft Status = iota
    StatusActive
    StatusRunning
    StatusCompleted
    StatusFailed
    StatusArchived
)

// Public constructor functions
func NewPipeline(name, description, owner string) result.Result[*Pipeline] { /* ... */ }
func NewStep(name, stepType string) Step { /* ... */ }
func NewID() ID { /* ... */ }

// Public utility functions
func ValidatePipelineName(name string) error { /* ... */ }
func ValidateStepDependencies(steps []Step) result.Result[void] { /* ... */ }

// Internal types (not exported)
type (
    dependencyGraph struct { /* ... */ }
    validationContext struct { /* ... */ }
    executionState struct { /* ... */ }
)

// Internal constants
const (
    maxStepNameLength = 100
    maxDependencyDepth = 50
    defaultExecutionTimeout = 30 * time.Minute
)

// Internal utility functions
func buildDependencyGraph(steps []Step) *dependencyGraph { /* ... */ }
func validateCircularDependencies(graph *dependencyGraph) error { /* ... */ }
func calculateResourceRequirements(steps []Step) ResourceRequirements { /* ... */ }
```

---

## 🧪 Testing Organization

### **Test Structure**

#### **Test File Organization**

```
internal/
├── domain/
│   ├── entities/
│   │   ├── pipeline/
│   │   │   ├── pipeline.go               # Production code
│   │   │   ├── pipeline_test.go          # Unit tests
│   │   │   ├── pipeline_integration_test.go # Integration tests
│   │   │   ├── step.go                   # Production code
│   │   │   ├── step_test.go              # Unit tests
│   │   │   └── testdata/                 # Test fixtures
│   │   │       ├── valid_pipelines.json
│   │   │       ├── invalid_pipelines.json
│   │   │       └── sample_steps.yaml
│   │   └── plugin/
│   │       ├── plugin.go
│   │       ├── plugin_test.go
│   │       └── plugin_integration_test.go
│   └── services/
│       ├── pipeline_orchestrator.go
│       ├── pipeline_orchestrator_test.go
│       └── pipeline_orchestrator_integration_test.go
├── app/
│   ├── commands/
│   │   ├── pipeline_commands.go
│   │   ├── pipeline_commands_test.go
│   │   └── pipeline_commands_integration_test.go
│   └── queries/
│       ├── pipeline_queries.go
│       ├── pipeline_queries_test.go
│       └── pipeline_queries_integration_test.go
└── infrastructure/
    ├── persistence/
    │   ├── postgres/
    │   │   ├── pipeline_repository.go
    │   │   ├── pipeline_repository_test.go
    │   │   └── pipeline_repository_integration_test.go
    │   └── redis/
    │       ├── coordinator.go
    │       ├── coordinator_test.go
    │       └── coordinator_integration_test.go
    └── messaging/
        ├── event_publisher.go
        ├── event_publisher_test.go
        └── event_publisher_integration_test.go

test/
├── fixtures/                            # Shared test fixtures
│   ├── pipelines/                       # Pipeline test data
│   │   ├── simple_pipeline.yaml
│   │   ├── complex_pipeline.yaml
│   │   └── invalid_pipeline.yaml
│   ├── plugins/                         # Plugin test implementations
│   │   ├── mock_processor.go
│   │   └── test_transformer.go
│   └── database/                        # Database test fixtures
│       ├── schema.sql
│       └── seed_data.sql
├── helpers/                             # Test utility functions
│   ├── assertions.go                    # Custom assertions
│   ├── builders.go                      # Test data builders
│   ├── database.go                      # Database test utilities
│   └── mocks.go                         # Mock generators
├── integration/                         # Cross-layer integration tests
│   ├── pipeline_lifecycle_test.go       # Full pipeline lifecycle
│   ├── event_sourcing_test.go          # Event sourcing integration
│   └── api_integration_test.go         # HTTP API integration
└── e2e/                                # End-to-end tests
    ├── complete_pipeline_test.go        # Complete pipeline execution
    ├── multi_service_test.go           # Multi-service coordination
    └── performance_test.go             # Performance benchmarks
```

#### **Test Documentation Standards**

```go
// Package pipeline_test provides comprehensive test coverage for pipeline domain
package pipeline_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/flext-sh/flexcore/internal/domain/entities/pipeline"
    "github.com/flext-sh/flexcore/test/fixtures"
    "github.com/flext-sh/flexcore/test/helpers"
)

// TestPipeline provides comprehensive test coverage for Pipeline aggregate
type TestPipeline struct {
    t *testing.T

    // Test dependencies
    mockEventBus    *helpers.MockEventBus
    mockRepository  *helpers.MockPipelineRepository
    testFixtures    *fixtures.PipelineFixtures

    // System under test
    pipeline *pipeline.Pipeline
}

// TestPipelineCreation tests pipeline creation with various scenarios
//
// This test suite validates pipeline creation including:
// - Successful creation with valid parameters
// - Validation failures with invalid parameters
// - Domain event generation verification
// - Business rule enforcement
//
// Test Scenarios:
//   - Valid pipeline creation with all required fields
//   - Invalid name handling (empty, too long, invalid characters)
//   - Invalid owner handling (empty, invalid format)
//   - Domain event generation and content validation
//   - Aggregate root initialization verification
func TestPipelineCreation(t *testing.T) {
    tests := []struct {
        name        string
        pipelineName string
        description string
        owner       string
        wantErr     bool
        expectedErr error
    }{
        {
            name:        "Valid pipeline creation",
            pipelineName: "customer-data-pipeline",
            description: "Processes customer data from Oracle to warehouse",
            owner:       "data-engineering-team",
            wantErr:     false,
        },
        {
            name:        "Empty pipeline name",
            pipelineName: "",
            description: "Valid description",
            owner:       "valid-team",
            wantErr:     true,
            expectedErr: pipeline.ErrEmptyPipelineName,
        },
        {
            name:        "Pipeline name too long",
            pipelineName: strings.Repeat("a", 101), // Over 100 character limit
            description: "Valid description",
            owner:       "valid-team",
            wantErr:     true,
            expectedErr: pipeline.ErrPipelineNameTooLong,
        },
        {
            name:        "Empty owner",
            pipelineName: "valid-pipeline-name",
            description: "Valid description",
            owner:       "",
            wantErr:     true,
            expectedErr: pipeline.ErrEmptyOwner,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Act
            result := pipeline.NewPipeline(tt.pipelineName, tt.description, tt.owner)

            // Assert
            if tt.wantErr {
                require.True(t, result.IsFailure(), "Expected pipeline creation to fail")
                assert.ErrorIs(t, result.Error(), tt.expectedErr,
                    "Expected specific error type")
            } else {
                require.True(t, result.IsSuccess(),
                    "Expected pipeline creation to succeed: %v", result.Error())

                p := result.Value()
                assert.NotNil(t, p, "Pipeline should not be nil")
                assert.Equal(t, tt.pipelineName, p.Name(), "Pipeline name mismatch")
                assert.Equal(t, tt.description, p.Description(), "Description mismatch")
                assert.Equal(t, tt.owner, p.Owner(), "Owner mismatch")
                assert.Equal(t, pipeline.StatusDraft, p.Status(), "Initial status should be Draft")

                // Verify domain events
                events := p.GetDomainEvents()
                require.Len(t, events, 1, "Should generate exactly one domain event")

                createdEvent, ok := events[0].(*pipeline.PipelineCreatedEvent)
                require.True(t, ok, "First event should be PipelineCreatedEvent")
                assert.Equal(t, tt.pipelineName, createdEvent.PipelineName)
                assert.Equal(t, tt.owner, createdEvent.Owner)
                assert.WithinDuration(t, time.Now(), createdEvent.OccurredAt(), time.Second,
                    "Event timestamp should be recent")
            }
        })
    }
}

// TestPipelineStepAddition tests step addition with comprehensive validation
//
// This test suite covers:
// - Successful step addition with valid configuration
// - Duplicate step name prevention
// - Dependency validation (existing dependencies, circular references)
// - Resource requirement validation
// - Business rule enforcement (cannot modify running pipelines)
// - Domain event generation for step addition
//
// The tests use table-driven approach to cover multiple scenarios efficiently
// while maintaining clear test case documentation and assertion specificity.
func TestPipelineStepAddition(t *testing.T) {
    // Setup
    pipeline := helpers.CreateValidPipeline(t)

    tests := []struct {
        name          string
        setupPipeline func(*pipeline.Pipeline)
        step          pipeline.Step
        wantErr       bool
        expectedErr   error
        validateState func(*testing.T, *pipeline.Pipeline)
    }{
        {
            name: "Add first step successfully",
            setupPipeline: func(p *pipeline.Pipeline) {
                // No additional setup needed
            },
            step:    helpers.CreateValidStep("extract-data", "oracle-extractor"),
            wantErr: false,
            validateState: func(t *testing.T, p *pipeline.Pipeline) {
                steps := p.GetSteps()
                assert.Len(t, steps, 1, "Should have exactly one step")
                assert.Equal(t, "extract-data", steps[0].Name())

                // Verify domain event
                events := p.GetDomainEvents()
                stepAddedEvents := helpers.FilterEventsByType(events, "pipeline.step.added")
                assert.Len(t, stepAddedEvents, 1, "Should generate step added event")
            },
        },
        {
            name: "Prevent duplicate step names",
            setupPipeline: func(p *pipeline.Pipeline) {
                existingStep := helpers.CreateValidStep("existing-step", "processor")
                result := p.AddStep(existingStep)
                require.True(t, result.IsSuccess(), "Setup step addition should succeed")
            },
            step:        helpers.CreateValidStep("existing-step", "different-processor"),
            wantErr:     true,
            expectedErr: pipeline.ErrDuplicateStepName,
        },
        {
            name: "Validate step dependencies exist",
            setupPipeline: func(p *pipeline.Pipeline) {
                // No setup - dependency will not exist
            },
            step: func() pipeline.Step {
                step := helpers.CreateValidStep("transform-data", "transformer")
                step.AddDependency("non-existent-step")
                return step
            }(),
            wantErr:     true,
            expectedErr: pipeline.ErrStepDependencyNotFound,
        },
        {
            name: "Prevent adding steps to running pipeline",
            setupPipeline: func(p *pipeline.Pipeline) {
                // Add step and start pipeline
                step := helpers.CreateValidStep("initial-step", "processor")
                require.True(t, p.AddStep(step).IsSuccess())
                require.True(t, p.Activate().IsSuccess())
                require.True(t, p.Start().IsSuccess())
            },
            step:        helpers.CreateValidStep("new-step", "processor"),
            wantErr:     true,
            expectedErr: pipeline.ErrCannotModifyRunningPipeline,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup fresh pipeline for each test
            p := helpers.CreateValidPipeline(t)
            tt.setupPipeline(p)

            // Act
            result := p.AddStep(tt.step)

            // Assert
            if tt.wantErr {
                require.True(t, result.IsFailure(), "Expected step addition to fail")
                if tt.expectedErr != nil {
                    assert.ErrorIs(t, result.Error(), tt.expectedErr,
                        "Expected specific error type")
                }
            } else {
                require.True(t, result.IsSuccess(),
                    "Expected step addition to succeed: %v", result.Error())

                if tt.validateState != nil {
                    tt.validateState(t, p)
                }
            }
        })
    }
}
```

---

## 📊 Quality Standards

### **Code Quality Requirements**

#### **Documentation Coverage**

- **100% coverage** for all exported functions, types, and methods
- **90% coverage** for internal functions and complex logic
- **Comprehensive examples** for all public APIs
- **Architecture notes** for complex implementations
- **Performance considerations** for critical paths
- **Thread safety documentation** for concurrent code

#### **Type Safety and Generics**

```go
// Proper use of Go generics for type safety
type Repository[T Entity, ID comparable] interface {
    Save(ctx context.Context, entity T) Result[T]
    GetByID(ctx context.Context, id ID) Result[T]
    Delete(ctx context.Context, id ID) Result[void]
    List(ctx context.Context, criteria ListCriteria) Result[[]T]
}

// Type-safe pipeline repository
type PipelineRepository = Repository[*Pipeline, pipeline.ID]

// Generic result pattern usage
func ProcessEntity[T Entity](ctx context.Context, entity T, processor Processor[T]) Result[ProcessResult] {
    // Type-safe entity processing
    if validationResult := entity.Validate(); validationResult.IsFailure() {
        return Result.Failure[ProcessResult](validationResult.Error())
    }

    return processor.Process(ctx, entity)
}
```

#### **Error Handling Standards**

```go
// Comprehensive error types with context
type PipelineError struct {
    Code    ErrorCode             `json:"code"`
    Message string               `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
    Cause   error                `json:"-"`
}

// Error constants with clear naming
const (
    ErrCodeInvalidPipelineState ErrorCode = "INVALID_PIPELINE_STATE"
    ErrCodeStepDependencyNotFound ErrorCode = "STEP_DEPENDENCY_NOT_FOUND"
    ErrCodeResourceLimitExceeded ErrorCode = "RESOURCE_LIMIT_EXCEEDED"
    ErrCodeCircularDependency ErrorCode = "CIRCULAR_DEPENDENCY"
)

// Error creation with context
func NewPipelineError(code ErrorCode, message string) *PipelineError {
    return &PipelineError{
        Code:    code,
        Message: message,
        Details: make(map[string]interface{}),
    }
}

// Error with contextual details
func (e *PipelineError) WithDetails(details map[string]interface{}) *PipelineError {
    for k, v := range details {
        e.Details[k] = v
    }
    return e
}

// Error wrapping
func (e *PipelineError) WithCause(cause error) *PipelineError {
    e.Cause = cause
    return e
}

// Standard error interface implementation
func (e *PipelineError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

### **Validation Tools**

#### **Automated Quality Checks**

```bash
# Documentation validation
make docs-validate                   # Validate all Go documentation
make docs-coverage                   # Check documentation coverage
make docs-examples-test              # Test all documentation examples

# Type and code validation
make lint                           # golangci-lint comprehensive checks
make vet                            # go vet static analysis
make security                       # gosec security scanning
make format                         # gofmt and goimports formatting

# Testing validation
make test                           # Run all tests with coverage
make test-race                      # Race condition detection
make test-integration               # Integration test suite
make test-benchmark                 # Performance benchmarks
```

#### **Linting Configuration**

```yaml
# .golangci.yml - Comprehensive linting configuration
run:
  timeout: 5m
  go: "1.24"

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

linters:
  enable:
    # Enabled by default
    - errcheck # Check unchecked errors
    - gosimple # Simplify code suggestions
    - govet # Go vet analysis
    - ineffassign # Detect ineffectual assignments
    - staticcheck # Advanced static analysis
    - typecheck # Type checking
    - unused # Unused constants, variables, functions

    # Additional quality linters
    - asciicheck # Non-ASCII character detection
    - bodyclose # HTTP response body closing
    - cyclop # Cyclomatic complexity
    - dupl # Code duplication detection
    - errorlint # Error wrapping analysis
    - exhaustive # Enum switch completeness
    - exportloopref # Loop variable capture issues
    - funlen # Function length limits
    - gocognit # Cognitive complexity
    - goconst # Repeated string constants
    - gocyclo # Cyclomatic complexity
    - godot # Comment punctuation
    - gofmt # Code formatting
    - goimports # Import organization
    - gomnd # Magic number detection
    - gomodguard # Module import restrictions
    - goprintffuncname # Printf function naming
    - gosec # Security issues
    - lll # Line length limits
    - misspell # Spelling mistakes
    - nakedret # Naked returns
    - nestif # Nested if statements
    - noctx # HTTP request context
    - nolintlint # Nolint directive usage
    - prealloc # Slice preallocation
    - predeclared # Predeclared identifier usage
    - revive # Golint replacement
    - rowserrcheck # SQL rows.Err() checking
    - sqlclosecheck # SQL resource closing
    - stylecheck # Style guidelines
    - tparallel # Parallel test detection
    - unconvert # Unnecessary type conversions
    - unparam # Unused function parameters
    - whitespace # Whitespace issues

linters-settings:
  cyclop:
    max-complexity: 15
  funlen:
    lines: 100
    statements: 50
  gocognit:
    min-complexity: 20
  gocyclo:
    min-complexity: 15
  lll:
    line-length: 120
  revive:
    rules:
      - name: exported
        arguments: ["disableStutteringCheck"]
```

---

## 🔄 Migration and Maintenance

### **Module Evolution Guidelines**

#### **Adding New Packages**

1. **Follow Go package naming conventions** (short, descriptive, lowercase)
2. **Create comprehensive package documentation** with examples
3. **Define clear interfaces** for dependency injection
4. **Implement proper error handling** with contextual information
5. **Add comprehensive tests** with appropriate coverage
6. **Document architectural decisions** in package comments

#### **Refactoring Existing Packages**

1. **Maintain backward compatibility** where possible using interface evolution
2. **Add deprecation notices** for removed functionality
3. **Use Go module versioning** for breaking changes
4. **Update all cross-references** in documentation and code
5. **Migrate tests** to new structure with proper fixtures

#### **Clean Architecture Compliance**

```go
// Example of enforcing architectural boundaries
package domain

// Domain layer can only import from:
import (
    // Standard library - allowed
    "context"
    "time"

    // Public utilities - allowed
    "github.com/flext-sh/flexcore/pkg/result"
    "github.com/flext-sh/flexcore/pkg/errors"

    // Other domain packages - allowed
    "github.com/flext-sh/flexcore/internal/domain/valueobjects"
)

// NOT ALLOWED - violates Clean Architecture:
// import "github.com/flext-sh/flexcore/internal/infrastructure/..."
// import "github.com/flext-sh/flexcore/internal/app/..."
// import "database/sql" (infrastructure concern)
// import "net/http" (infrastructure concern)
```

### **Dependency Management**

#### **Internal Dependencies**

- **Minimize coupling** between packages using interfaces
- **Use dependency injection** for cross-package communication
- **Document dependencies** in package documentation
- **Validate dependency graph** in CI/CD pipeline
- **Enforce architectural boundaries** with linting rules

#### **External Dependencies**

```go
// go.mod - Careful dependency management
module github.com/flext-sh/flexcore

go 1.24

require (
    // Core dependencies with pinned versions
    github.com/gin-gonic/gin v1.10.1       // HTTP framework
    github.com/google/uuid v1.6.0          // UUID generation
    github.com/lib/pq v1.10.9              // PostgreSQL driver
    github.com/go-redis/redis/v8 v8.11.5   // Redis client
    go.uber.org/zap v1.21.0               // Structured logging

    // Testing dependencies
    github.com/stretchr/testify v1.10.0    // Test assertions

    // Development dependencies (not in production builds)
    github.com/golangci/golangci-lint v1.50.0 // Linting
)

// Exclude vulnerable or problematic versions
exclude (
    github.com/lib/pq v1.10.8  // Known security issue
)

// Replace with maintained forks if needed
replace (
    // Example: replace abandoned dependency
    github.com/old/package => github.com/maintained/package v1.2.3
)
```

---

## 📞 Support and Guidelines

### **Development Guidelines**

#### **Go Module Development Checklist**

- [ ] **Comprehensive package documentation** with usage examples
- [ ] **Complete function/method documentation** with business context
- [ ] **Proper error handling** with contextual error types
- [ ] **Interface-based design** for dependency injection
- [ ] **Unit tests** with 90%+ coverage requirement
- [ ] **Integration tests** for external dependencies
- [ ] **Benchmark tests** for performance-critical code
- [ ] **Race condition testing** for concurrent code
- [ ] **Clean Architecture compliance** with proper layer boundaries
- [ ] **Result pattern usage** for error handling consistency

#### **Code Review Requirements**

- [ ] **Documentation quality** and completeness assessment
- [ ] **Error handling** patterns and context preservation
- [ ] **Interface design** for testability and maintainability
- [ ] **Clean Architecture** boundary compliance verification
- [ ] **Test coverage** and quality evaluation
- [ ] **Performance impact** analysis for critical paths
- [ ] **Security considerations** review for sensitive operations
- [ ] **Concurrency safety** analysis for shared state

### **Getting Help**

#### **Documentation Support**

- **Architecture Questions**: See [Architecture Overview](../architecture/overview.md)
- **Clean Architecture**: See [TODO.md](../TODO.md) for current compliance issues
- **FLEXT Integration**: See [Integration Guide](../integration/flext-ecosystem.md)
- **Result Pattern**: See `pkg/result/result.go` implementation
- **Error Handling**: See `pkg/errors/errors.go` patterns

#### **Quality Assurance**

- **Automated Validation**: `make validate` runs comprehensive quality checks
- **Documentation Validation**: `make docs-validate` checks documentation coverage
- **Security Scanning**: `make security` runs gosec vulnerability analysis
- **Performance Testing**: `make test-benchmark` validates performance requirements

---

## 🚧 Current State and Refactoring

### **Architectural Issues (See TODO.md)**

#### **Critical Violations Identified**

1. **Clean Architecture Violations**:

   - HTTP server in application layer (`internal/app/application.go`)
   - Infrastructure dependencies in application layer
   - Mixed concerns across architectural boundaries

2. **Multiple CQRS Implementations**:

   - Generic implementation in `internal/app/commands/`
   - SQLite implementation in `internal/infrastructure/cqrs/`
   - Function-based implementation in `internal/infrastructure/`

3. **Inadequate Event Sourcing**:
   - In-memory event store for production (`internal/infrastructure/event_store.go`)
   - Mutable domain events (violates Event Sourcing principles)
   - Missing event replay capabilities

### **Refactoring Roadmap**

#### **Phase 1: Foundation (2-3 weeks)**

1. **Extract HTTP Layer**: Move HTTP infrastructure to presentation layer
2. **Unify CQRS**: Choose single CQRS implementation with proper interfaces
3. **Implement PostgreSQL Event Store**: Replace in-memory with persistent store
4. **Add Integration Tests**: Comprehensive test coverage for refactored components

#### **Phase 2: Domain Enhancement (3-4 weeks)**

1. **Rich Domain Model**: Implement proper aggregates with business behavior
2. **Domain Services**: Extract complex business logic to domain services
3. **Plugin Security**: Implement process isolation and sandboxing
4. **Event Sourcing**: Complete immutable event streams with replay capability

#### **Phase 3: Production Readiness (4-6 weeks)**

1. **Performance Optimization**: Meet enterprise performance requirements
2. **Security Hardening**: Complete security audit and vulnerability fixes
3. **Observability**: Comprehensive monitoring, metrics, and tracing
4. **Documentation**: Update all documentation to reflect new architecture

---

**Module Organization Version**: 0.9.0  
**Last Updated**: 2025-08-01  
**Compliance**: FLEXT Ecosystem Standards  
**Maintained By**: FLEXT Development Team

This document serves as the **definitive guide** for Go module organization in FlexCore. All new packages and refactoring efforts must follow these standards for ecosystem consistency and architectural compliance.

> ⚠️ **Important**: Due to ongoing architectural refactoring, some examples represent target implementation rather than current state. Always refer to [TODO.md](../TODO.md) for current architectural issues and implementation status.
