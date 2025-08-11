# Domain Layer

**Package**: `github.com/flext-sh/flexcore/internal/domain`  
**Version**: 0.9.0  
**Status**: Active Development

## Overview

The Domain Layer implements FlexCore's core business logic using Domain-Driven Design (DDD) patterns. It provides the foundational types and patterns for building rich domain models with proper encapsulation, event sourcing, and business rule enforcement.

## Architecture

This package forms the heart of FlexCore's Clean Architecture implementation:

- **Domain Entities**: Business objects with identity and lifecycle
- **Aggregate Roots**: Consistency boundaries with transaction scope
- **Value Objects**: Immutable concepts defined by attributes
- **Domain Events**: Event sourcing for state change tracking
- **Domain Services**: Complex business operations spanning aggregates
- **Specifications**: Composable business rules and validation

## Core Components

### Entity[T]

Base type for all domain entities with identity, lifecycle tracking, and optimistic concurrency control.

```go
type ProductID string

type Product struct {
    domain.Entity[ProductID]
    Name        string
    Price       decimal.Decimal
    Category    string
}

func NewProduct(id ProductID, name string, price decimal.Decimal) *Product {
    return &Product{
        Entity:   domain.NewEntity(id),
        Name:     name,
        Price:    price,
        Category: "uncategorized",
    }
}
```

### AggregateRoot[T]

Foundation for aggregates with domain event management and consistency enforcement.

```go
type OrderID string

type Order struct {
    domain.AggregateRoot[OrderID]
    CustomerID   string
    Items        []OrderItem
    Status       OrderStatus
    TotalAmount  Money
}

func (o *Order) AddItem(item OrderItem) error {
    // Business logic validation
    if o.Status != OrderStatusDraft {
        return errors.New("cannot modify confirmed order")
    }

    // State change
    o.Items = append(o.Items, item)
    o.Touch() // Update entity metadata

    // Generate domain event
    event := NewOrderItemAddedEvent(o.ID, item)
    o.RaiseEvent(event)

    return nil
}
```

### Domain Events

Event sourcing support with immutable event patterns for CQRS integration.

```go
type UserCreatedEvent struct {
    domain.BaseDomainEvent
    UserName string
    Email    string
}

func NewUserCreatedEvent(userID UserID, name, email string) *UserCreatedEvent {
    return &UserCreatedEvent{
        BaseDomainEvent: domain.NewBaseDomainEvent("user.created", userID.String()),
        UserName:        name,
        Email:           email,
    }
}
```

### Value Objects

Immutable objects representing domain concepts without identity.

```go
type Money struct {
    amount   decimal.Decimal
    currency string
}

func (m Money) Equals(other domain.ValueObject) bool {
    if otherMoney, ok := other.(Money); ok {
        return m.amount.Equal(otherMoney.amount) && m.currency == otherMoney.currency
    }
    return false
}

func (m Money) String() string {
    return fmt.Sprintf("%s %s", m.amount.String(), m.currency)
}
```

### Specifications

Composable business rules with logical operations for complex validation.

```go
// Age validation specification
ageSpec := domain.NewSpecification(func(user User) bool {
    return user.Age >= 18
})

// Email validation specification
emailSpec := domain.NewSpecification(func(user User) bool {
    return strings.Contains(user.Email, "@")
})

// Combined validation
validUserSpec := ageSpec.And(emailSpec)

if validUserSpec.IsSatisfiedBy(user) {
    // User meets all criteria
}
```

## Domain Entities

### Pipeline

Main aggregate root for data processing pipeline management.

```go
// Located in: internal/domain/entities/pipeline.go
type Pipeline struct {
    domain.AggregateRoot[PipelineID]
    Name        string
    Description string
    Status      PipelineStatus
    Steps       []PipelineStep
    Tags        []string
    Owner       string
    Schedule    *PipelineSchedule
    LastRunAt   *time.Time
    NextRunAt   *time.Time
}
```

**Key Operations**:

- `NewPipeline()` - Create with validation and event generation
- `AddStep()` / `RemoveStep()` - Manage processing steps
- `Activate()` / `Deactivate()` - Control pipeline availability
- `Start()` / `Complete()` / `Fail()` - Execution lifecycle
- `SetSchedule()` / `ClearSchedule()` - Automated scheduling

### Plugin

Entity representing executable data processing components.

```go
// Located in: internal/domain/entities/plugin.go
type Plugin struct {
    domain.AggregateRoot[PluginID]
    Name         string
    Version      string
    Type         PluginType
    Configuration map[string]interface{}
    Status       PluginStatus
}
```

## Event Sourcing Integration

All aggregate operations generate domain events for:

- **State Change Tracking**: Complete audit trail of business operations
- **CQRS Read Models**: Event handlers update query projections
- **Cross-Aggregate Communication**: Eventual consistency through events
- **Integration Events**: External system notifications and webhooks

### Event Lifecycle

1. **Business Operation**: Aggregate method modifies state
2. **Event Generation**: `RaiseEvent()` adds event to collection
3. **Event Accumulation**: Events collect until aggregate persistence
4. **Event Publication**: Application layer publishes to event bus
5. **Event Processing**: Handlers update read models and trigger side effects
6. **Event Clearing**: `ClearEvents()` resets collection after success

## Business Rules & Validation

### Pipeline Business Rules

- Pipeline names must be unique within owner context
- Steps must have unique names within the pipeline
- Status transitions follow defined state machine rules
- Dependencies between steps must form valid DAG (no cycles)
- Only active pipelines can be started
- Running pipelines cannot be modified

### Validation Patterns

```go
// Input validation in constructors
func NewPipeline(name, description, owner string) result.Result[*Pipeline] {
    if name == "" {
        return result.Failure[*Pipeline](errors.ValidationError("pipeline name cannot be empty"))
    }

    if owner == "" {
        return result.Failure[*Pipeline](errors.ValidationError("pipeline owner cannot be empty"))
    }

    // Create pipeline and raise creation event
    // ...
}

// Business rule validation in methods
func (p *Pipeline) Start() result.Result[bool] {
    if p.Status != PipelineStatusActive {
        return result.Failure[bool](errors.ValidationError("can only start active pipelines"))
    }

    // State transition and event generation
    // ...
}
```

## Integration with Application Layer

The domain layer integrates with FlexCore's application layer through:

- **Command Handlers**: Execute domain operations from commands
- **Event Handlers**: Process domain events for read model updates
- **Query Handlers**: Access aggregates for query processing
- **Domain Services**: Coordinate complex operations across aggregates

## Repository Pattern

Generic repository interfaces for aggregate persistence:

```go
type Repository[T any, ID comparable] interface {
    Save(entity T) error
    FindByID(id ID) (T, error)
    Delete(id ID) error
    Exists(id ID) bool
}

// Usage with specific aggregates
type PipelineRepository interface {
    domain.Repository[*Pipeline, PipelineID]
    FindByOwner(owner string) ([]*Pipeline, error)
    FindByStatus(status PipelineStatus) ([]*Pipeline, error)
}
```

## Best Practices

1. **Rich Domain Model**: Encapsulate business logic in entities and aggregates
2. **Explicit Error Handling**: All operations return Results for clear error handling
3. **Event-Driven Architecture**: Generate events for all state changes
4. **Immutable Value Objects**: Represent domain concepts without identity
5. **Aggregate Boundaries**: Maintain consistency within aggregate scope
6. **Domain Services**: Use for complex operations spanning multiple aggregates

## Performance Considerations

- **Optimistic Locking**: Version-based concurrency control prevents lost updates
- **Event Batching**: Accumulate events for efficient publication
- **Lazy Loading**: Load related entities only when needed
- **Aggregate Size**: Keep aggregates focused and reasonably sized

## Thread Safety

- **Single-threaded Access**: Aggregates designed for single-thread access within boundaries
- **Immutable Events**: Domain events are immutable after creation
- **Stateless Services**: Domain services should be stateless and thread-safe
- **Optimistic Concurrency**: Version-based conflict detection for concurrent access

## Testing Strategy

- **Unit Tests**: Test domain logic in isolation with comprehensive coverage
- **Specification Tests**: Validate business rules with specification patterns
- **Event Testing**: Verify correct event generation for state changes
- **Integration Tests**: Test aggregate interactions and repository patterns

---

**See Also**: [FlexCore Architecture](../../docs/architecture/overview.md) | [Result Package](../../pkg/result/README.md) | [Application Layer](../app/README.md)
