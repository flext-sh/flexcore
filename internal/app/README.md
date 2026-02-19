# Application Layer


<!-- TOC START -->
- [Overview](#overview)
- [Architecture](#architecture)
- [Current Status: Refactoring Required](#current-status-refactoring-required)
  - [Known Issues](#known-issues)
- [Core Components](#core-components)
  - [Command Processing (`commands/`)](#command-processing-commands)
  - [Query Processing (`queries/`)](#query-processing-queries)
  - [Application Services (`application.go`)](#application-services-applicationgo)
- [CQRS Implementation](#cqrs-implementation)
  - [Command Side (Write Operations)](#command-side-write-operations)
  - [Query Side (Read Operations)](#query-side-read-operations)
- [Use Case Patterns](#use-case-patterns)
  - [Transaction Management](#transaction-management)
  - [Event-Driven Workflows](#event-driven-workflows)
- [Integration Patterns](#integration-patterns)
  - [Domain Layer Integration](#domain-layer-integration)
  - [Infrastructure Layer Integration](#infrastructure-layer-integration)
  - [Presentation Layer Integration](#presentation-layer-integration)
- [Planned Improvements (Post-Refactoring)](#planned-improvements-post-refactoring)
  - [Enhanced CQRS Implementation](#enhanced-cqrs-implementation)
  - [Advanced Transaction Management](#advanced-transaction-management)
  - [Performance Optimizations](#performance-optimizations)
  - [Monitoring and Observability](#monitoring-and-observability)
- [Best Practices](#best-practices)
- [Testing Strategy](#testing-strategy)
  - [Unit Testing](#unit-testing)
  - [Integration Testing](#integration-testing)
  - [Performance Testing](#performance-testing)
<!-- TOC END -->

**Package**: `github.com/flext-sh/flexcore/internal/app`  
**Version**: 0.9.9 RC  
**Status**: In Development - Architecture Refactoring Required · 1.0.0 Release Preparation

## Overview

The Application Layer orchestrates business workflows by coordinating domain entities, implementing use cases, and managing cross-cutting concerns. It serves as the bridge between the presentation layer and domain layer, implementing CQRS (Command Query Responsibility Segregation) patterns for clear separation of read and write operations.

## Architecture

This layer implements Clean Architecture principles with clear separation of concerns:

- **Command Processing**: Write operations that modify system state
- **Query Processing**: Read operations for data retrieval and reporting
- **Use Case Orchestration**: Coordination of domain operations and workflows
- **Transaction Management**: Ensuring data consistency across operations
- **Event Coordination**: Publishing domain events to interested handlers

## Current Status: Refactoring Required

⚠️ **Important**: This layer requires significant refactoring to properly implement Clean Architecture and CQRS patterns. See [TODO.md](../../docs/TODO.md) for architectural issues.

### Known Issues

- Mixed responsibilities between application and infrastructure concerns
- Incomplete CQRS implementation with multiple conflicting patterns
- Insufficient separation between command and query processing
- Limited transaction boundary management
- Inconsistent error handling patterns

## Core Components

### Command Processing (`commands/`)

#### Command Bus

Central dispatcher for command processing with middleware support.

```go
// Located in: commands/command_bus.go
type CommandBus interface {
    Execute(ctx context.Context, command Command) error
    Register(commandType string, handler CommandHandler)
}

type Command interface {
    CommandType() string
    CommandID() string
    Validate() error
}

type CommandHandler interface {
    Handle(ctx context.Context, command Command) error
}
```

#### Pipeline Commands

Business operations for pipeline management.

```go
// Located in: commands/pipeline_commands.go
type CreatePipelineCommand struct {
    ID          string
    Name        string
    Description string
    Owner       string
}

type StartPipelineCommand struct {
    PipelineID string
    Parameters map[string]interface{}
}

type AddPipelineStepCommand struct {
    PipelineID string
    Step       PipelineStepData
}
```

### Query Processing (`queries/`)

#### Query Bus

Central dispatcher for query processing with caching and optimization support.

```go
// Located in: queries/query_bus.go
type QueryBus interface {
    Execute(ctx context.Context, query Query) (interface{}, error)
    Register(queryType string, handler QueryHandler)
}

type Query interface {
    QueryType() string
    QueryID() string
    Validate() error
}

type QueryHandler interface {
    Handle(ctx context.Context, query Query) (interface{}, error)
}
```

#### Pipeline Queries

Read operations for pipeline data retrieval.

```go
// Located in: queries/pipeline_queries.go
type GetPipelineQuery struct {
    PipelineID string
}

type ListPipelinesQuery struct {
    Owner    string
    Status   *PipelineStatus
    Tags     []string
    PageSize int
    Offset   int
}

type GetPipelineStatusQuery struct {
    PipelineID string
}
```

### Application Services (`application.go`)

Main application service coordinating use cases and workflows.

```go
// Located in: application.go
type ApplicationService interface {
    // Pipeline Management
    CreatePipeline(ctx context.Context, cmd CreatePipelineCommand) error
    UpdatePipeline(ctx context.Context, cmd UpdatePipelineCommand) error
    DeletePipeline(ctx context.Context, cmd DeletePipelineCommand) error

    // Pipeline Execution
    StartPipeline(ctx context.Context, cmd StartPipelineCommand) error
    StopPipeline(ctx context.Context, cmd StopPipelineCommand) error

    // Pipeline Queries
    GetPipeline(ctx context.Context, query GetPipelineQuery) (*PipelineView, error)
    ListPipelines(ctx context.Context, query ListPipelinesQuery) (*PipelineListView, error)
}
```

## CQRS Implementation

### Command Side (Write Operations)

Commands represent user intentions to change system state:

1. **Command Validation**: Ensure command structure and business rules
2. **Domain Operation**: Execute business logic through aggregates
3. **Event Generation**: Domain events are raised for state changes
4. **Persistence**: Save aggregate state and events to storage
5. **Event Publication**: Publish events to update read models

```go
// Example command flow
func (s *ApplicationService) CreatePipeline(ctx context.Context, cmd CreatePipelineCommand) error {
    // 1. Validate command
    if err := cmd.Validate(); err != nil {
        return err
    }

    // 2. Execute domain logic
    pipeline := entities.NewPipeline(cmd.Name, cmd.Description, cmd.Owner)
    if pipeline.IsFailure() {
        return pipeline.Error()
    }

    // 3. Save aggregate (generates events)
    if err := s.pipelineRepo.Save(pipeline.Value()); err != nil {
        return err
    }

    // 4. Publish events
    events := pipeline.Value().DomainEvents()
    for _, event := range events {
        s.eventBus.Publish(ctx, event)
    }

    return nil
}
```

### Query Side (Read Operations)

Queries retrieve data without modifying system state:

1. **Query Validation**: Ensure query parameters are valid
2. **Data Retrieval**: Access optimized read models or projections
3. **Result Composition**: Build view models for presentation layer
4. **Caching**: Leverage caching for frequently accessed data

```go
// Example query flow
func (s *ApplicationService) GetPipeline(ctx context.Context, query GetPipelineQuery) (*PipelineView, error) {
    // 1. Validate query
    if err := query.Validate(); err != nil {
        return nil, err
    }

    // 2. Retrieve from read model
    pipeline, err := s.pipelineReadModel.GetByID(query.PipelineID)
    if err != nil {
        return nil, err
    }

    // 3. Build view model
    view := &PipelineView{
        ID:          pipeline.ID,
        Name:        pipeline.Name,
        Status:      pipeline.Status.String(),
        StepsCount:  len(pipeline.Steps),
        LastRunAt:   pipeline.LastRunAt,
        NextRunAt:   pipeline.NextRunAt,
    }

    return view, nil
}
```

## Use Case Patterns

### Transaction Management

Ensuring consistency across multiple operations:

```go
func (s *ApplicationService) ExecutePipelineWorkflow(ctx context.Context, cmd ExecuteWorkflowCommand) error {
    return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
        // 1. Load pipeline aggregate
        pipeline, err := s.pipelineRepo.FindByID(cmd.PipelineID)
        if err != nil {
            return err
        }

        // 2. Execute business operations
        if err := pipeline.Start(); err.IsFailure() {
            return err.Error()
        }

        // 3. Process each step
        for _, step := range pipeline.Steps {
            if err := s.executeStep(txCtx, step); err != nil {
                pipeline.Fail(err.Error())
                break
            }
        }

        // 4. Complete or fail pipeline
        if pipeline.Status == PipelineStatusRunning {
            pipeline.Complete()
        }

        // 5. Save final state
        return s.pipelineRepo.Save(pipeline)
    })
}
```

### Event-Driven Workflows

Coordinating operations through domain events:

```go
// Event handler for pipeline completion
func (h *PipelineCompletedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    completedEvent := event.(*PipelineCompletedEvent)

    // Update read models
    if err := h.updatePipelineView(completedEvent); err != nil {
        return err
    }

    // Trigger downstream processes
    if err := h.triggerNotifications(completedEvent); err != nil {
        return err
    }

    // Update metrics and analytics
    return h.updatePipelineMetrics(completedEvent)
}
```

## Integration Patterns

### Domain Layer Integration

- **Aggregate Operations**: Execute business logic through domain entities
- **Event Processing**: Handle domain events for cross-aggregate coordination
- **Business Rules**: Enforce domain invariants and validation rules
- **Result Patterns**: Use Result types for explicit error handling

### Infrastructure Layer Integration

- **Repository Access**: Persist and retrieve aggregates
- **Event Store**: Save and replay domain events
- **External Services**: Integrate with plugins and external systems
- **Caching**: Optimize read operations with cached data

### Presentation Layer Integration

- **API Controllers**: Receive HTTP requests and delegate to application services
- **Command/Query DTOs**: Data transfer objects for request/response mapping
- **View Models**: Optimized data structures for presentation needs
- **Error Handling**: Translate domain errors to appropriate HTTP responses

## Planned Improvements (Post-Refactoring)

### Enhanced CQRS Implementation

- Clear separation of command and query models
- Optimized read projections for complex queries
- Event-driven read model updates
- Command validation and authorization middleware

### Advanced Transaction Management

- Saga pattern for long-running workflows
- Compensation logic for distributed transactions
- Event-driven process coordination
- Retry and error recovery mechanisms

### Performance Optimizations

- Async command processing for non-critical operations
- Query result caching with intelligent invalidation
- Batch processing for bulk operations
- Connection pooling and resource management

### Monitoring and Observability

- Command and query execution metrics
- Performance tracking and alerting
- Distributed tracing for workflow visibility
- Business process monitoring and analytics

## Best Practices

1. **Clear Separation**: Keep commands and queries strictly separated
2. **Thin Application Layer**: Delegate business logic to domain entities
3. **Event-Driven Design**: Use domain events for loose coupling
4. **Explicit Error Handling**: Use Result patterns throughout
5. **Transaction Boundaries**: Manage consistency at aggregate boundaries
6. **Idempotent Operations**: Ensure operations can be safely retried

## Testing Strategy

### Unit Testing

- Test command and query handlers in isolation
- Mock domain services and repositories
- Verify proper error handling and validation
- Test event generation and publication

### Integration Testing

- Test complete use case workflows
- Verify transaction boundary management
- Test event-driven process coordination
- Validate read model consistency

### Performance Testing

- Load test command and query processing
- Benchmark critical business workflows
- Test system behavior under concurrent load
- Validate caching effectiveness

---

**See Also**: [Domain Layer](../domain/README.md) | [Infrastructure Layer](../infrastructure/README.md) | [Architecture Overview](../../docs/architecture/overview.md)
