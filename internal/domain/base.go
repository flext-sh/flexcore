// Package domain provides the core domain layer types and patterns
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entity represents a domain entity with identity
type Entity[T comparable] struct {
	ID        T
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int64
}

// NewEntity creates a new entity with the given ID
func NewEntity[T comparable](id T) Entity[T] {
	now := time.Now()
	return Entity[T]{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
}

// Touch updates the entity's timestamp and version
func (e *Entity[T]) Touch() {
	e.UpdatedAt = time.Now()
	e.Version++
}

// Equals compares two entities by ID
func (e *Entity[T]) Equals(other Entity[T]) bool {
	return e.ID == other.ID
}

// ValueObject represents an immutable value object
type ValueObject interface {
	Equals(other ValueObject) bool
	String() string
}

// AggregateRoot represents a domain aggregate root
type AggregateRoot[T comparable] struct {
	Entity[T]
	domainEvents []DomainEvent
}

// NewAggregateRoot creates a new aggregate root
func NewAggregateRoot[T comparable](id T) AggregateRoot[T] {
	return AggregateRoot[T]{
		Entity:       NewEntity(id),
		domainEvents: make([]DomainEvent, 0),
	}
}

// RaiseEvent adds a domain event to the aggregate
func (ar *AggregateRoot[T]) RaiseEvent(event DomainEvent) {
	ar.domainEvents = append(ar.domainEvents, event)
}

// DomainEvents returns all raised domain events
func (ar *AggregateRoot[T]) DomainEvents() []DomainEvent {
	return ar.domainEvents
}

// ClearEvents clears all domain events
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

// CreatePluginEvent creates a plugin-related domain event
func CreatePluginEvent(eventType, pluginID string, data map[string]interface{}) DomainEvent {
	return NewBaseDomainEvent(eventType, pluginID)
}

// CreatePipelineEvent creates a pipeline-related domain event
func CreatePipelineEvent(eventType, pipelineID string, data map[string]interface{}) DomainEvent {
	return NewBaseDomainEvent(eventType, pipelineID)
}

// CreateSystemEvent creates a system-related domain event
func CreateSystemEvent(eventType string, data map[string]interface{}) DomainEvent {
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
