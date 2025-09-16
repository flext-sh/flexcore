// Package domain provides the foundational Domain-Driven Design patterns and types for FlexCore.
//
// This package implements the core building blocks of FlexCore's domain layer,
// following Domain-Driven Design (DDD) principles and patterns. It provides
// type-safe, generic implementations of essential DDD concepts including
// entities, value objects, aggregate roots, domain events, and specifications.
//
// Key Components:
//   - Entity[T]: Base type for all domain entities with identity and lifecycle
//   - AggregateRoot[T]: Aggregate root with domain event management
//   - DomainEvent: Event sourcing support with immutable event patterns
//   - Specification[T]: Business rule encapsulation with composable logic
//   - Repository[T, ID]: Persistence abstraction with generic type safety
//   - DomainService: Domain service pattern for complex business operations
//
// Architecture:
//
//	This package forms the foundation for all FlexCore domain logic, implementing:
//	- Clean Architecture domain layer patterns
//	- Event Sourcing with domain event generation
//	- CQRS support through event-driven architecture
//	- Generic type safety with Go 1.24+ generics
//	- Immutable value objects and event streams
//
// Design Patterns:
//   - Entity Pattern: Identity-based entities with lifecycle tracking
//   - Aggregate Pattern: Consistency boundaries with transaction scope
//   - Event Sourcing: Immutable event streams for state reconstruction
//   - Specification Pattern: Composable business rules and validation
//   - Repository Pattern: Persistence abstraction with clean interfaces
//   - Domain Service Pattern: Complex business operations spanning aggregates
//
// Integration:
//   - Used by all FlexCore domain entities (Pipeline, Plugin, etc.)
//   - Supports CQRS command/query handlers in application layer
//   - Integrates with event store implementations in infrastructure layer
//   - Compatible with dependency injection containers
//   - Enables comprehensive testing through specification patterns
//
// Example:
//
//	Creating domain entities with event generation:
//
//	  type UserID string
//
//	  type User struct {
//	      domain.AggregateRoot[UserID]
//	      Name  string
//	      Email string
//	  }
//
//	  func NewUser(id UserID, name, email string) *User {
//	      user := &User{
//	          AggregateRoot: domain.NewAggregateRoot(id),
//	          Name:          name,
//	          Email:         email,
//	      }
//
//	      event := NewUserCreatedEvent(id, name, email)
//	      user.RaiseEvent(event)
//
//	      return user
//	  }
//
// Thread Safety:
//
//	Domain entities are designed for single-threaded access within aggregate boundaries.
//	Concurrent access should be managed at the application layer through proper
//	synchronization mechanisms or message-based coordination.
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entity represents a domain entity with identity, lifecycle tracking, and optimistic concurrency control.
//
// Entity provides the base implementation for all domain entities following
// Domain-Driven Design principles. It ensures entity identity, tracks creation
// and modification timestamps, and supports optimistic concurrency control
// through version management.
//
// Type Parameters:
//
//	T comparable: The type of the entity's unique identifier (must be comparable for equality)
//
// Fields:
//
//	ID T: Unique identifier for the entity (immutable after creation)
//	CreatedAt time.Time: Timestamp when the entity was first created
//	UpdatedAt time.Time: Timestamp of the most recent modification
//	Version int64: Optimistic locking version for concurrency control
//
// Design Patterns:
//   - Identity Pattern: Entities are distinguished by unique identifiers
//   - Optimistic Locking: Version field prevents lost update problems
//   - Immutable Identity: ID cannot be changed after entity creation
//   - Lifecycle Tracking: Comprehensive audit trail with timestamps
//
// Example:
//
//	Using Entity as a base for domain objects:
//
//	  type ProductID string
//
//	  type Product struct {
//	      domain.Entity[ProductID]
//	      Name        string
//	      Price       decimal.Decimal
//	      Category    string
//	  }
//
//	  func NewProduct(id ProductID, name string, price decimal.Decimal) *Product {
//	      return &Product{
//	          Entity:   domain.NewEntity(id),
//	          Name:     name,
//	          Price:    price,
//	          Category: "uncategorized",
//	      }
//	  }
//
// Concurrency Control:
//
//	The Version field enables optimistic locking patterns:
//	- Version increments on each modification via Touch()
//	- Application layer can detect concurrent modifications
//	- Prevents lost update problems in distributed systems
//
// Thread Safety:
//
//	Entity instances should be accessed by single threads within aggregate boundaries.
//	Version-based optimistic locking handles cross-transaction concurrency.
type Entity[T comparable] struct {
	ID        T         // Unique entity identifier (immutable)
	CreatedAt time.Time // Entity creation timestamp
	UpdatedAt time.Time // Last modification timestamp
	Version   int64     // Optimistic locking version
}

// NewEntity creates a new entity with the specified ID and initializes lifecycle tracking.
//
// This constructor establishes the entity's identity and sets up proper lifecycle
// tracking with synchronized timestamps and initial version. The entity starts
// at version 1, indicating it's a newly created instance.
//
// Type Parameters:
//
//	T comparable: The type of the unique identifier
//
// Parameters:
//
//	id T: Unique identifier for the entity (must be non-zero value for most types)
//
// Returns:
//
//	Entity[T]: New entity instance with initialized metadata
//
// Example:
//
//	Creating entities with different ID types:
//
//	  // String-based ID
//	  userEntity := domain.NewEntity("user-123")
//
//	  // UUID-based ID
//	  orderID := uuid.New()
//	  orderEntity := domain.NewEntity(orderID)
//
//	  // Integer-based ID
//	  productEntity := domain.NewEntity(int64(42))
//
// Initial State:
//   - CreatedAt and UpdatedAt are set to the same timestamp
//   - Version is set to 1 (initial version)
//   - ID is set to the provided value (immutable thereafter)
//
// Thread Safety:
//
//	Safe for concurrent use as each call creates an independent entity instance.
func NewEntity[T comparable](id T) Entity[T] {
	now := time.Now()
	return Entity[T]{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
}

// Touch updates the entity's modification timestamp and increments the version for optimistic locking.
//
// This method should be called whenever the entity's state is modified,
// ensuring proper audit trails and enabling optimistic concurrency control.
// It automatically updates the UpdatedAt timestamp and increments the version
// to detect concurrent modifications.
//
// Example:
//
//	Updating entity state with proper versioning:
//
//	  product := domain.NewEntity("product-123")
//	  fmt.Printf("Initial version: %d\n", product.Version) // 1
//
//	  // Modify product state
//	  product.Name = "Updated Product Name"
//	  product.Touch() // Update timestamp and version
//
//	  fmt.Printf("New version: %d\n", product.Version) // 2
//	  fmt.Printf("Updated at: %v\n", product.UpdatedAt)
//
// Concurrency Control:
//
//	The version increment enables optimistic locking patterns:
//	- Application layer can compare versions before persistence
//	- Concurrent modifications can be detected and handled
//	- Prevents lost update problems in multi-user scenarios
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread accessing the entity.
//	Concurrent access should be managed at higher architectural layers.
func (e *Entity[T]) Touch() {
	e.UpdatedAt = time.Now()
	e.Version++
}

// Equals compares two entities for equality based on their unique identifiers.
//
// This method implements entity equality semantics following Domain-Driven Design
// principles, where entities are considered equal if they have the same identity,
// regardless of their other attribute values or state.
//
// Parameters:
//
//	other Entity[T]: The entity to compare against
//
// Returns:
//
//	bool: true if both entities have the same ID, false otherwise
//
// Example:
//
//	Entity equality comparison:
//
//	  user1 := domain.NewEntity("user-123")
//	  user2 := domain.NewEntity("user-456")
//	  user3 := domain.NewEntity("user-123")
//
//	  fmt.Println(user1.Equals(user2)) // false - different IDs
//	  fmt.Println(user1.Equals(user3)) // true - same ID
//
//	  // Even if other fields differ, entities with same ID are equal
//	  user3.Touch() // Different version and timestamp
//	  fmt.Println(user1.Equals(user3)) // still true - same ID
//
// DDD Principle:
//
//	Entities are distinguished by identity, not attributes. Two entities
//	with the same ID are considered the same entity, even if their
//	state or version differs.
//
// Thread Safety:
//
//	Safe for concurrent use as it only reads immutable ID fields.
func (e *Entity[T]) Equals(other Entity[T]) bool {
	return e.ID == other.ID
}

// ValueObject defines the contract for immutable value objects in Domain-Driven Design.
//
// ValueObject represents concepts that are defined by their attributes rather
// than identity. Value objects are immutable, meaning their state cannot change
// after creation. They are compared by value equality rather than identity.
//
// Methods:
//
//	Equals(other ValueObject) bool: Compares value objects for equality by attributes
//	String() string: Provides string representation for debugging and logging
//
// Design Principles:
//   - Immutability: Value objects cannot be modified after creation
//   - Value Equality: Equality is based on attributes, not identity
//   - Side-Effect Free: Methods should not cause observable state changes
//   - Conceptual Whole: Represents a cohesive concept in the domain
//
// Example:
//
//	Implementing a Money value object:
//
//	  type Money struct {
//	      amount   decimal.Decimal
//	      currency string
//	  }
//
//	  func (m Money) Equals(other domain.ValueObject) bool {
//	      if otherMoney, ok := other.(Money); ok {
//	          return m.amount.Equal(otherMoney.amount) && m.currency == otherMoney.currency
//	      }
//	      return false
//	  }
//
//	  func (m Money) String() string {
//	      return fmt.Sprintf("%s %s", m.amount.String(), m.currency)
//	  }
//
// Common Value Objects:
//   - Email addresses, phone numbers, postal addresses
//   - Money amounts with currency
//   - Date ranges, time periods
//   - Coordinates, measurements
//   - Configuration settings, validation rules
//
// Thread Safety:
//
//	Value objects are inherently thread-safe due to their immutable nature.
type ValueObject interface {
	Equals(other ValueObject) bool // Compare by value, not identity
	String() string                // String representation for display/debugging
}

// AggregateRoot represents a domain aggregate root with event sourcing capabilities.
//
// AggregateRoot extends Entity to provide the foundation for aggregate boundaries
// in Domain-Driven Design. It manages domain events for event sourcing and CQRS
// patterns, ensuring consistency within the aggregate boundary while enabling
// eventual consistency across aggregates through event-driven communication.
//
// Type Parameters:
//
//	T comparable: The type of the aggregate's unique identifier
//
// Fields:
//
//	Entity[T]: Base entity with identity, timestamps, and versioning
//	domainEvents []DomainEvent: Collection of uncommitted domain events
//
// Design Patterns:
//   - Aggregate Pattern: Defines consistency boundary and transaction scope
//   - Event Sourcing: Generates events for all state changes
//   - Command-Query Responsibility Segregation (CQRS): Events enable read model updates
//   - Domain Event Pattern: Decouples aggregates through event-driven communication
//
// Responsibilities:
//   - Enforce business invariants within aggregate boundary
//   - Generate domain events for state changes
//   - Provide transactional consistency for aggregate operations
//   - Enable eventual consistency through event publication
//
// Example:
//
//	Implementing an Order aggregate:
//
//	  type OrderID string
//
//	  type Order struct {
//	      domain.AggregateRoot[OrderID]
//	      CustomerID   string
//	      Items        []OrderItem
//	      Status       OrderStatus
//	      TotalAmount  Money
//	  }
//
//	  func (o *Order) AddItem(item OrderItem) error {
//	      // Business logic validation
//	      if o.Status != OrderStatusDraft {
//	          return errors.New("cannot modify confirmed order")
//	      }
//
//	      // State change
//	      o.Items = append(o.Items, item)
//	      o.Touch() // Update entity metadata
//
//	      // Generate domain event
//	      event := NewOrderItemAddedEvent(o.ID, item)
//	      o.RaiseEvent(event)
//
//	      return nil
//	  }
//
// Event Lifecycle:
//  1. Business operations call RaiseEvent() for state changes
//  2. Events accumulate in domainEvents collection
//  3. Application layer persists aggregate and publishes events
//  4. Events are cleared after successful persistence
//
// Thread Safety:
//
//	Aggregate roots should be accessed by single threads within transaction boundaries.
//	Cross-aggregate communication should use domain events for eventual consistency.
type AggregateRoot[T comparable] struct {
	Entity[T]                  // Base entity with identity and lifecycle
	domainEvents []DomainEvent // Uncommitted domain events for event sourcing
}

// NewAggregateRoot creates a new aggregate root with proper initialization for event sourcing.
//
// This constructor initializes both the base entity with lifecycle tracking
// and the domain event collection for event sourcing patterns. The aggregate
// starts with an empty event collection, ready to accumulate events as
// business operations are performed.
//
// Type Parameters:
//
//	T comparable: The type of the aggregate's unique identifier
//
// Parameters:
//
//	id T: Unique identifier for the aggregate root
//
// Returns:
//
//	AggregateRoot[T]: New aggregate root with initialized entity and empty event collection
//
// Example:
//
//	Creating different types of aggregate roots:
//
//	  // User aggregate with string ID
//	  userRoot := domain.NewAggregateRoot("user-123")
//
//	  // Order aggregate with UUID
//	  orderID := uuid.New()
//	  orderRoot := domain.NewAggregateRoot(orderID)
//
//	  // Product aggregate with integer ID
//	  productRoot := domain.NewAggregateRoot(int64(12345))
//
// Initial State:
//   - Entity is initialized with ID, timestamps, and version 1
//   - Domain events collection is empty but ready for event accumulation
//   - Aggregate is ready for business operations and event generation
//
// Integration:
//   - Command handlers use this to create new aggregates
//   - Event store implementations persist the aggregate and its events
//   - CQRS read models are updated based on generated events
//
// Thread Safety:
//
//	Safe for concurrent use as each call creates an independent aggregate instance.
func NewAggregateRoot[T comparable](id T) AggregateRoot[T] {
	return AggregateRoot[T]{
		Entity:       NewEntity(id),
		domainEvents: make([]DomainEvent, 0),
	}
}

// RaiseEvent adds a domain event to the aggregate's uncommitted event collection.
//
// This method accumulates domain events generated by business operations,
// supporting event sourcing and CQRS patterns. Events are stored locally
// until the aggregate is persisted, at which point they are published to
// event handlers and then cleared.
//
// Parameters:
//
//	event DomainEvent: The domain event to add to the collection
//
// Example:
//
//	Raising events from business operations:
//
//	  func (u *User) ChangeEmail(newEmail string) error {
//	      // Business logic validation
//	      if !isValidEmail(newEmail) {
//	          return errors.New("invalid email format")
//	      }
//
//	      oldEmail := u.Email
//	      u.Email = newEmail
//	      u.Touch() // Update entity metadata
//
//	      // Generate and raise domain event
//	      event := NewUserEmailChangedEvent(u.ID, oldEmail, newEmail)
//	      u.RaiseEvent(event)
//
//	      return nil
//	  }
//
// Event Sourcing Pattern:
//  1. Business operations modify aggregate state
//  2. Operations call RaiseEvent() to record what happened
//  3. Events accumulate in the domainEvents collection
//  4. Application layer persists aggregate and publishes events
//  5. Event handlers update read models and trigger side effects
//
// Thread Safety:
//
//	Not thread-safe - should be called from the single thread managing the aggregate.
//	Cross-aggregate communication should use domain events for decoupling.
func (ar *AggregateRoot[T]) RaiseEvent(event DomainEvent) {
	ar.domainEvents = append(ar.domainEvents, event)
}

// DomainEvents returns all uncommitted domain events generated by the aggregate.
//
// This method provides access to the domain events that have been raised
// but not yet published. It's typically used by the application layer to
// retrieve events for persistence and publication to event handlers.
//
// Returns:
//
//	[]DomainEvent: Slice of all uncommitted domain events in chronological order
//
// Example:
//
//	Processing events in the application layer:
//
//	  // After executing business operations
//	  user.ChangeEmail("new@example.com")
//	  user.UpdateProfile(newProfile)
//
//	  // Retrieve all generated events
//	  events := user.DomainEvents()
//
//	  // Persist aggregate and publish events
//	  if err := repository.Save(user); err != nil {
//	      return err
//	  }
//
//	  for _, event := range events {
//	      if err := eventBus.Publish(event); err != nil {
//	          log.Error("Failed to publish event", "event", event)
//	      }
//	  }
//
//	  // Clear events after successful publication
//	  user.ClearEvents()
//
// Event Ordering:
//
//	Events are returned in the order they were raised, preserving
//	the chronological sequence of business operations.
//
// Thread Safety:
//
//	Returns the internal slice directly - callers should not modify the returned slice.
//	Consider returning a copy if mutation by callers is a concern.
func (ar *AggregateRoot[T]) DomainEvents() []DomainEvent {
	return ar.domainEvents
}

// ClearEvents removes all domain events from the aggregate's event collection.
//
// This method is called after domain events have been successfully persisted
// and published to event handlers. It resets the event collection to prevent
// duplicate event processing and prepares the aggregate for additional operations.
//
// Example:
//
//	Event processing lifecycle:
//
//	  // Execute business operations (generates events)
//	  order.AddItem(item1)
//	  order.AddItem(item2)
//	  order.ApplyDiscount(discount)
//
//	  // Process accumulated events
//	  events := order.DomainEvents()
//
//	  // Persist aggregate state
//	  if err := repository.Save(order); err != nil {
//	      return err // Don't clear events if persistence fails
//	  }
//
//	  // Publish events to handlers
//	  for _, event := range events {
//	      eventBus.Publish(event)
//	  }
//
//	  // Clear events after successful processing
//	  order.ClearEvents()
//
// Error Handling:
//
//	Only clear events after successful persistence and publication.
//	If any step fails, keep events for retry or compensation logic.
//
// Performance:
//
//	Creates a new empty slice rather than clearing existing slice,
//	allowing garbage collection of event objects.
//
// Thread Safety:
//
//	Not thread-safe - should be called from the same thread managing the aggregate.
func (ar *AggregateRoot[T]) ClearEvents() {
	ar.domainEvents = make([]DomainEvent, 0)
}

// DomainEvent represents a domain event
type DomainEvent interface {
	EventID() string
	EventType() string
	OccurredAt() time.Time
	AggregateID() string
}

// BaseDomainEvent provides a base implementation for domain events
type BaseDomainEvent struct {
	ID          string
	Type        string
	Timestamp   time.Time
	AggregateId string
}

// NewBaseDomainEvent creates a new base domain event
func NewBaseDomainEvent(eventType, aggregateID string) BaseDomainEvent {
	return BaseDomainEvent{
		ID:          uuid.New().String(),
		Type:        eventType,
		Timestamp:   time.Now(),
		AggregateId: aggregateID,
	}
}

// EventID returns the event ID
func (e BaseDomainEvent) EventID() string {
	return e.ID
}

// EventType returns the event type
func (e BaseDomainEvent) EventType() string {
	return e.Type
}

// OccurredAt returns when the event occurred
func (e BaseDomainEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the aggregate ID
func (e BaseDomainEvent) AggregateID() string {
	return e.AggregateId
}

// DomainEventFactory provides centralized domain event creation following the unified struct pattern.
//
// This factory encapsulates all domain event creation logic, ensuring consistent event
// structure and eliminating loose helper functions. It follows SOLID principles by
// providing a single responsibility for event creation with type-safe operations.
//
// Design Patterns:
//   - Factory Pattern: Centralized creation of domain events
//   - Single Responsibility: Only handles domain event creation
//   - Unified Struct: All event creation logic in one struct
//
// Example:
//
//	Creating events with the factory:
//
//	  factory := domain.NewDomainEventFactory()
//
//	  // Create plugin events
//	  pluginEvent := factory.CreatePluginEvent("plugin.loaded", "plugin-123", eventData)
//
//	  // Create pipeline events
//	  pipelineEvent := factory.CreatePipelineEvent("pipeline.started", "pipeline-456", eventData)
//
//	  // Create system events
//	  systemEvent := factory.CreateSystemEvent("system.startup", eventData)
//
// Thread Safety:
//
//	Safe for concurrent use as event creation is stateless.
type DomainEventFactory struct{}

// NewDomainEventFactory creates a new domain event factory instance.
//
// Returns:
//
//	*DomainEventFactory: A new factory instance ready for event creation
//
// Example:
//
//	factory := domain.NewDomainEventFactory()
//	event := factory.CreatePluginEvent("plugin.loaded", "plugin-123", nil)
func NewDomainEventFactory() *DomainEventFactory {
	return &DomainEventFactory{}
}

// CreatePluginEvent creates a plugin-related domain event with consistent structure.
//
// This method generates domain events for plugin lifecycle operations, ensuring
// consistent event format and proper aggregate identification for event sourcing.
//
// Parameters:
//
//	eventType string: The type of plugin event (e.g., "plugin.loaded", "plugin.failed")
//	pluginID string: The unique identifier of the plugin aggregate
//	data map[string]interface{}: Additional event data (currently unused but reserved for future use)
//
// Returns:
//
//	DomainEvent: A properly structured domain event for plugin operations
//
// Example:
//
//	factory := domain.NewDomainEventFactory()
//	event := factory.CreatePluginEvent("plugin.loaded", "plugin-123", map[string]interface{}{
//	    "version": "1.0.0",
//	    "author":  "FLEXT Team",
//	})
func (factory *DomainEventFactory) CreatePluginEvent(eventType, pluginID string, data map[string]interface{}) DomainEvent {
	return NewBaseDomainEvent(eventType, pluginID)
}

// CreatePipelineEvent creates a pipeline-related domain event with consistent structure.
//
// This method generates domain events for pipeline lifecycle operations, ensuring
// consistent event format and proper aggregate identification for event sourcing.
//
// Parameters:
//
//	eventType string: The type of pipeline event (e.g., "pipeline.started", "pipeline.completed")
//	pipelineID string: The unique identifier of the pipeline aggregate
//	data map[string]interface{}: Additional event data (currently unused but reserved for future use)
//
// Returns:
//
//	DomainEvent: A properly structured domain event for pipeline operations
//
// Example:
//
//	factory := domain.NewDomainEventFactory()
//	event := factory.CreatePipelineEvent("pipeline.started", "pipeline-456", map[string]interface{}{
//	    "owner":     "data-team",
//	    "step_count": 3,
//	})
func (factory *DomainEventFactory) CreatePipelineEvent(eventType, pipelineID string, data map[string]interface{}) DomainEvent {
	return NewBaseDomainEvent(eventType, pipelineID)
}

// CreateSystemEvent creates a system-related domain event with consistent structure.
//
// This method generates domain events for system-wide operations that don't belong
// to a specific aggregate, ensuring consistent event format for monitoring and
// observability systems.
//
// Parameters:
//
//	eventType string: The type of system event (e.g., "system.startup", "system.shutdown")
//	data map[string]interface{}: Additional event data (currently unused but reserved for future use)
//
// Returns:
//
//	DomainEvent: A properly structured domain event for system operations
//
// Example:
//
//	factory := domain.NewDomainEventFactory()
//	event := factory.CreateSystemEvent("system.startup", map[string]interface{}{
//	    "version":   "0.9.0",
//	    "node_id":   "node-789",
//	    "timestamp": time.Now(),
//	})
func (factory *DomainEventFactory) CreateSystemEvent(eventType string, data map[string]interface{}) DomainEvent {
	return NewBaseDomainEvent(eventType, "")
}

// Repository represents a domain repository
type Repository[T any, ID comparable] interface {
	Save(entity T) error
	FindByID(id ID) (T, error)
	Delete(id ID) error
	Exists(id ID) bool
}

// Specification represents a domain specification pattern
type Specification[T any] interface {
	IsSatisfiedBy(entity T) bool
	And(other Specification[T]) Specification[T]
	Or(other Specification[T]) Specification[T]
	Not() Specification[T]
}

// BaseSpecification provides a base implementation for specifications
type BaseSpecification[T any] struct {
	predicate func(T) bool
}

// NewSpecification creates a new specification with the given predicate
func NewSpecification[T any](predicate func(T) bool) Specification[T] {
	return &BaseSpecification[T]{predicate: predicate}
}

// IsSatisfiedBy checks if the entity satisfies the specification
func (s *BaseSpecification[T]) IsSatisfiedBy(entity T) bool {
	return s.predicate(entity)
}

// And combines two specifications with AND logic
func (s *BaseSpecification[T]) And(other Specification[T]) Specification[T] {
	return NewSpecification(func(entity T) bool {
		return s.IsSatisfiedBy(entity) && other.IsSatisfiedBy(entity)
	})
}

// Or combines two specifications with OR logic
func (s *BaseSpecification[T]) Or(other Specification[T]) Specification[T] {
	return NewSpecification(func(entity T) bool {
		return s.IsSatisfiedBy(entity) || other.IsSatisfiedBy(entity)
	})
}

// Not negates the specification
func (s *BaseSpecification[T]) Not() Specification[T] {
	return NewSpecification(func(entity T) bool {
		return !s.IsSatisfiedBy(entity)
	})
}

// DomainService represents a domain service
type DomainService interface {
	Name() string
}

// BaseDomainService provides a base implementation for domain services
type BaseDomainService struct {
	name string
}

// NewDomainService creates a new domain service
func NewDomainService(name string) BaseDomainService {
	return BaseDomainService{name: name}
}

// Name returns the service name
func (s BaseDomainService) Name() string {
	return s.name
}
