// Package services defines domain service interfaces for FlexCore
// Following Clean Architecture and DDD patterns
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventBus handles domain event publishing and subscription
// CQRS Event Side - Decouples event producers from consumers
type EventBus interface {
	// Publish sends a domain event to all registered subscribers
	Publish(ctx context.Context, event DomainEvent) error

	// Subscribe registers a handler for specific event types
	Subscribe(eventType string, handler EventHandler) error

	// Unsubscribe removes a handler for specific event types
	Unsubscribe(eventType string, handler EventHandler) error

	// Close shuts down the event bus and cleans up resources
	Close() error
}

// CommandBus handles command processing and routing
// CQRS Command Side - Ensures single responsibility for write operations
type CommandBus interface {
	// Send dispatches a command to its registered handler
	Send(ctx context.Context, command Command) error

	// Register associates a command type with its handler
	Register(commandType string, handler CommandHandler) error

	// Unregister removes a command handler
	Unregister(commandType string) error
}

// PluginLoader manages dynamic plugin loading and lifecycle
// Plugin Architecture - Dynamic .so file loading with sandbox execution
type PluginLoader interface {
	// LoadPlugin dynamically loads a plugin from file path
	LoadPlugin(ctx context.Context, pluginPath string) (Plugin, error)

	// UnloadPlugin removes a plugin from memory and cleans resources
	UnloadPlugin(ctx context.Context, pluginID string) error

	// GetPlugin retrieves a loaded plugin by ID
	GetPlugin(pluginID string) (Plugin, bool)

	// ListPlugins returns all currently loaded plugins
	ListPlugins() []Plugin

	// ValidatePlugin checks plugin compatibility and dependencies
	ValidatePlugin(pluginPath string) error
}

// CoordinationLayer provides distributed coordination and clustering
// Event Sourcing + Distributed Systems - Ensures consistency across nodes
type CoordinationLayer interface {
	// JoinCluster adds this node to the cluster
	JoinCluster(ctx context.Context, nodeID string) error

	// LeaveCluster removes this node from the cluster
	LeaveCluster(ctx context.Context) error

	// GetClusterNodes returns all active cluster nodes
	GetClusterNodes(ctx context.Context) ([]ClusterNode, error)

	// DistributeLock acquires a distributed lock for coordination
	DistributeLock(ctx context.Context, resource string, ttl time.Duration) (Lock, error)

	// PublishEvent sends events across the cluster
	PublishEvent(ctx context.Context, event ClusterEvent) error
}

// EventStore provides event sourcing persistence
// Event Sourcing - Immutable event storage with snapshotting
type EventStore interface {
	// SaveEvents persists domain events for an aggregate
	SaveEvents(ctx context.Context, aggregateID string, events []DomainEvent, expectedVersion int) error

	// GetEvents retrieves events for an aggregate from a specific version
	GetEvents(ctx context.Context, aggregateID string, fromVersion int) ([]DomainEvent, error)

	// SaveSnapshot stores an aggregate snapshot for performance
	SaveSnapshot(ctx context.Context, snapshot AggregateSnapshot) error

	// GetSnapshot retrieves the latest snapshot for an aggregate
	GetSnapshot(ctx context.Context, aggregateID string) (AggregateSnapshot, error)

	// GetEventsByType retrieves all events of a specific type
	GetEventsByType(ctx context.Context, eventType string, from time.Time) ([]DomainEvent, error)
}

// Supporting types for the interfaces

// DomainEvent represents something that happened in the domain
type DomainEvent interface {
	GetEventID() uuid.UUID
	GetAggregateID() string
	GetEventType() string
	GetEventData() map[string]interface{}
	GetOccurredAt() time.Time
	GetVersion() int
}

// EventHandler processes domain events
type EventHandler interface {
	Handle(ctx context.Context, event DomainEvent) error
	EventType() string
}

// Command represents an intention to change system state
type Command interface {
	GetCommandID() uuid.UUID
	GetCommandType() string
	GetPayload() map[string]interface{}
	GetTimestamp() time.Time
}

// CommandHandler processes commands
type CommandHandler interface {
	Handle(ctx context.Context, command Command) error
	CommandType() string
}

// Plugin represents a dynamically loaded processing plugin
type Plugin interface {
	GetID() string
	GetName() string
	GetVersion() string
	Initialize(ctx context.Context, config map[string]interface{}) error
	Process(ctx context.Context, data []byte) ([]byte, error)
	Cleanup(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}

// ClusterNode represents a node in the distributed cluster
type ClusterNode struct {
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata"`
	JoinedAt time.Time         `json:"joined_at"`
}

// Lock represents a distributed lock
type Lock interface {
	Release(ctx context.Context) error
	Extend(ctx context.Context, ttl time.Duration) error
	IsValid() bool
}

// ClusterEvent represents events distributed across cluster nodes
type ClusterEvent struct {
	ID        uuid.UUID              `json:"id"`
	Type      string                 `json:"type"`
	NodeID    string                 `json:"node_id"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// AggregateSnapshot represents a point-in-time snapshot of an aggregate
type AggregateSnapshot struct {
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int                    `json:"version"`
	Data          map[string]interface{} `json:"data"`
	CreatedAt     time.Time              `json:"created_at"`
}