// Package events provides event handling capabilities using windmill
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/flext/flexcore/domain"
	"github.com/flext/flexcore/infrastructure/scheduler"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
)

// EventHandler represents a function that handles domain events
type EventHandler func(ctx context.Context, event domain.DomainEvent) error

// EventBus represents an event bus for publishing and subscribing to domain events
type EventBus interface {
	Publish(ctx context.Context, event domain.DomainEvent) error
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
	Start(ctx context.Context) error
	Stop() error
}

// InMemoryEventBus provides an in-memory implementation of EventBus
type InMemoryEventBus struct {
	mu           sync.RWMutex
	subscribers  map[string][]EventHandler
	eventQueue   chan EventMessage
	isRunning    bool
	stopChan     chan struct{}
	maxWorkers   int
	bufferSize   int
	retryPolicy  *RetryPolicy
}

// EventMessage represents an event message in the queue
type EventMessage struct {
	Event     domain.DomainEvent
	Timestamp time.Time
	Retries   int
}

// RetryPolicy defines retry behavior for failed event handling
type RetryPolicy struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  time.Millisecond * 100,
		MaxDelay:      time.Second * 10,
		BackoffFactor: 2.0,
	}
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus(maxWorkers, bufferSize int) *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers:  make(map[string][]EventHandler),
		eventQueue:   make(chan EventMessage, bufferSize),
		maxWorkers:   maxWorkers,
		bufferSize:   bufferSize,
		retryPolicy:  DefaultRetryPolicy(),
		stopChan:     make(chan struct{}),
	}
}

// Publish publishes a domain event to the bus
func (bus *InMemoryEventBus) Publish(ctx context.Context, event domain.DomainEvent) error {
	if !bus.isRunning {
		return errors.ValidationError("event bus is not running")
	}

	message := EventMessage{
		Event:     event,
		Timestamp: time.Now(),
		Retries:   0,
	}

	select {
	case bus.eventQueue <- message:
		return nil
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "context cancelled while publishing event")
	default:
		return errors.ValidationError("event queue is full")
	}
}

// Subscribe subscribes a handler to events of a specific type
func (bus *InMemoryEventBus) Subscribe(eventType string, handler EventHandler) error {
	if handler == nil {
		return errors.ValidationError("handler cannot be nil")
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if _, exists := bus.subscribers[eventType]; !exists {
		bus.subscribers[eventType] = make([]EventHandler, 0)
	}

	bus.subscribers[eventType] = append(bus.subscribers[eventType], handler)
	return nil
}

// Unsubscribe removes a handler from event subscriptions
func (bus *InMemoryEventBus) Unsubscribe(eventType string, handler EventHandler) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	handlers, exists := bus.subscribers[eventType]
	if !exists {
		return errors.NotFoundError("event type " + eventType)
	}

	// Find and remove the handler
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			bus.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// Start starts the event bus processing
func (bus *InMemoryEventBus) Start(ctx context.Context) error {
	if bus.isRunning {
		return errors.ValidationError("event bus is already running")
	}

	bus.isRunning = true

	// Start worker goroutines
	for i := 0; i < bus.maxWorkers; i++ {
		go bus.worker(ctx, i)
	}

	return nil
}

// Stop stops the event bus processing
func (bus *InMemoryEventBus) Stop() error {
	if !bus.isRunning {
		return nil
	}

	bus.isRunning = false
	close(bus.stopChan)
	close(bus.eventQueue)

	return nil
}

// worker processes events from the queue
func (bus *InMemoryEventBus) worker(ctx context.Context, workerID int) {
	for {
		select {
		case message, ok := <-bus.eventQueue:
			if !ok {
				return // Channel closed
			}
			bus.processEvent(ctx, message, workerID)

		case <-bus.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// processEvent processes a single event message
func (bus *InMemoryEventBus) processEvent(ctx context.Context, message EventMessage, workerID int) {
	bus.mu.RLock()
	handlers, exists := bus.subscribers[message.Event.EventType()]
	bus.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return // No handlers for this event type
	}

	// Process each handler
	for _, handler := range handlers {
		if err := bus.executeHandler(ctx, handler, message.Event); err != nil {
			// Handle retry logic
			if message.Retries < bus.retryPolicy.MaxRetries {
				delay := bus.calculateRetryDelay(message.Retries)
				time.AfterFunc(delay, func() {
					message.Retries++
					select {
					case bus.eventQueue <- message:
					default:
						// Queue full, drop the retry
					}
				})
			}
		}
	}
}

// executeHandler executes a single event handler
func (bus *InMemoryEventBus) executeHandler(ctx context.Context, handler EventHandler, event domain.DomainEvent) error {
	defer func() {
		if r := recover(); r != nil {
			// Log panic recovery
		}
	}()

	return handler(ctx, event)
}

// calculateRetryDelay calculates the delay for retry attempts
func (bus *InMemoryEventBus) calculateRetryDelay(retryCount int) time.Duration {
	delay := float64(bus.retryPolicy.InitialDelay)
	for i := 0; i < retryCount; i++ {
		delay *= bus.retryPolicy.BackoffFactor
	}

	maxDelay := float64(bus.retryPolicy.MaxDelay)
	if delay > maxDelay {
		delay = maxDelay
	}

	return time.Duration(delay)
}

// GetSubscriberCount returns the number of subscribers for an event type
func (bus *InMemoryEventBus) GetSubscriberCount(eventType string) int {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	if handlers, exists := bus.subscribers[eventType]; exists {
		return len(handlers)
	}
	return 0
}

// GetQueueSize returns the current size of the event queue
func (bus *InMemoryEventBus) GetQueueSize() int {
	return len(bus.eventQueue)
}

// EventStore represents persistent event storage
type EventStore interface {
	Save(ctx context.Context, event domain.DomainEvent) error
	GetEvents(ctx context.Context, aggregateID string) ([]domain.DomainEvent, error)
	GetEventsByType(ctx context.Context, eventType string, limit int) ([]domain.DomainEvent, error)
}

// InMemoryEventStore provides an in-memory implementation of EventStore
type InMemoryEventStore struct {
	mu     sync.RWMutex
	events map[string][]StoredEvent
}

// StoredEvent represents a stored domain event
type StoredEvent struct {
	ID          string
	AggregateID string
	EventType   string
	EventData   json.RawMessage
	Timestamp   time.Time
	Version     int64
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string][]StoredEvent),
	}
}

// Save saves a domain event to the store
func (store *InMemoryEventStore) Save(ctx context.Context, event domain.DomainEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal event")
	}

	storedEvent := StoredEvent{
		ID:          event.EventID(),
		AggregateID: event.AggregateID(),
		EventType:   event.EventType(),
		EventData:   data,
		Timestamp:   event.OccurredAt(),
		Version:     1,
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	aggregateID := event.AggregateID()
	if _, exists := store.events[aggregateID]; !exists {
		store.events[aggregateID] = make([]StoredEvent, 0)
	}

	store.events[aggregateID] = append(store.events[aggregateID], storedEvent)
	return nil
}

// GetEvents retrieves all events for an aggregate
func (store *InMemoryEventStore) GetEvents(ctx context.Context, aggregateID string) ([]domain.DomainEvent, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	_, exists := store.events[aggregateID]
	if !exists {
		return make([]domain.DomainEvent, 0), nil
	}

	// For this example, we return empty slice since we'd need specific event types
	// In a real implementation, you'd deserialize based on event type
	return make([]domain.DomainEvent, 0), nil
}

// GetEventsByType retrieves events by type with a limit
func (store *InMemoryEventStore) GetEventsByType(ctx context.Context, eventType string, limit int) ([]domain.DomainEvent, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	var foundEvents []domain.DomainEvent
	count := 0

	for _, events := range store.events {
		for _, event := range events {
			if event.EventType == eventType {
				// In a real implementation, you'd deserialize the event
				count++
				if limit > 0 && count >= limit {
					break
				}
			}
		}
		if limit > 0 && count >= limit {
			break
		}
	}

	return foundEvents, nil
}

// ClusterAwareEventBus extends EventBus with cluster coordination
type ClusterAwareEventBus interface {
	EventBus
	BroadcastToCluster(ctx context.Context, event domain.DomainEvent) error
	SubscribeToClusterEvents(ctx context.Context) error
	GetClusterStats() ClusterEventStats
}

// ClusterEventStats provides statistics about cluster event processing
type ClusterEventStats struct {
	LocalEvents    int64 `json:"local_events"`
	ClusterEvents  int64 `json:"cluster_events"`
	NodesReached   int   `json:"nodes_reached"`
	LastBroadcast  time.Time `json:"last_broadcast"`
	BroadcastErrors int64 `json:"broadcast_errors"`
}

// Use scheduler types for cluster coordination
type ClusterCoordinator = scheduler.ClusterCoordinator
type Lock = scheduler.Lock
type NodeInfo = scheduler.NodeInfo
type ClusterMessage = scheduler.ClusterMessage
type ClusterMessageHandler = scheduler.ClusterMessageHandler

// DistributedEventBus provides distributed event bus functionality
type DistributedEventBus struct {
	*InMemoryEventBus
	coordinator     ClusterCoordinator
	mu              sync.RWMutex
	clusterStats    ClusterEventStats
	nodeID          string
	isClusterActive bool
}

// NewDistributedEventBus creates a new distributed event bus
func NewDistributedEventBus(maxWorkers, bufferSize int, coordinator ClusterCoordinator) *DistributedEventBus {
	localBus := NewInMemoryEventBus(maxWorkers, bufferSize)
	
	return &DistributedEventBus{
		InMemoryEventBus: localBus,
		coordinator:      coordinator,
		nodeID:          coordinator.GetNodeID(),
		clusterStats:    ClusterEventStats{},
		isClusterActive: false,
	}
}

// Start starts the distributed event bus
func (bus *DistributedEventBus) Start(ctx context.Context) error {
	// Start the local event bus first
	if err := bus.InMemoryEventBus.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start local event bus")
	}

	// Subscribe to cluster events
	if err := bus.SubscribeToClusterEvents(ctx); err != nil {
		return errors.Wrap(err, "failed to subscribe to cluster events")
	}

	bus.mu.Lock()
	bus.isClusterActive = true
	bus.mu.Unlock()

	return nil
}

// Stop stops the distributed event bus
func (bus *DistributedEventBus) Stop() error {
	bus.mu.Lock()
	bus.isClusterActive = false
	bus.mu.Unlock()

	// Stop local event bus
	if err := bus.InMemoryEventBus.Stop(); err != nil {
		return errors.Wrap(err, "failed to stop local event bus")
	}

	return nil
}

// Publish publishes an event locally and optionally to the cluster
func (bus *DistributedEventBus) Publish(ctx context.Context, event domain.DomainEvent) error {
	// Publish locally first
	if err := bus.InMemoryEventBus.Publish(ctx, event); err != nil {
		return err
	}

	bus.mu.Lock()
	bus.clusterStats.LocalEvents++
	bus.mu.Unlock()

	// Determine if this event should be broadcast to cluster
	if bus.shouldBroadcastToCluster(event) {
		if err := bus.BroadcastToCluster(ctx, event); err != nil {
			// Log error but don't fail local publish
			bus.mu.Lock()
			bus.clusterStats.BroadcastErrors++
			bus.mu.Unlock()
		}
	}

	return nil
}

// BroadcastToCluster broadcasts an event to all cluster nodes
func (bus *DistributedEventBus) BroadcastToCluster(ctx context.Context, event domain.DomainEvent) error {
	if !bus.isClusterActive || bus.coordinator == nil {
		return errors.ValidationError("cluster coordination not active")
	}

	// Create cluster message with event data
	eventData := map[string]interface{}{
		"event_id":     event.EventID(),
		"event_type":   event.EventType(),
		"aggregate_id": event.AggregateID(),
		"occurred_at":  event.OccurredAt(),
		"source_node":  bus.nodeID,
	}

	// Add event payload if available
	if payloadEvent, ok := event.(interface{ GetPayload() map[string]interface{} }); ok {
		eventData["payload"] = payloadEvent.GetPayload()
	}

	message := ClusterMessage{
		Type:      "domain_event",
		SourceID:  bus.nodeID,
		Payload:   eventData,
		Timestamp: time.Now(),
	}

	// Broadcast to cluster
	if err := bus.coordinator.BroadcastMessage(ctx, message); err != nil {
		return errors.Wrap(err, "failed to broadcast event to cluster")
	}

	bus.mu.Lock()
	bus.clusterStats.ClusterEvents++
	bus.clusterStats.LastBroadcast = time.Now()
	bus.mu.Unlock()

	return nil
}

// SubscribeToClusterEvents subscribes to cluster-wide events
func (bus *DistributedEventBus) SubscribeToClusterEvents(ctx context.Context) error {
	if bus.coordinator == nil {
		return errors.ValidationError("cluster coordinator not available")
	}

	// Subscribe to cluster messages
	handler := func(ctx context.Context, message ClusterMessage) error {
		return bus.handleClusterMessage(ctx, message)
	}

	return bus.coordinator.SubscribeToMessages(ctx, handler)
}

// handleClusterMessage handles incoming cluster messages
func (bus *DistributedEventBus) handleClusterMessage(ctx context.Context, message ClusterMessage) error {
	// Skip messages from this node
	if message.SourceID == bus.nodeID {
		return nil
	}

	// Handle domain events
	if message.Type == "domain_event" {
		return bus.processClusterEvent(ctx, message)
	}

	return nil
}

// processClusterEvent processes a domain event received from cluster
func (bus *DistributedEventBus) processClusterEvent(ctx context.Context, message ClusterMessage) error {
	eventData := message.Payload

	// Extract event information
	eventID, _ := eventData["event_id"].(string)
	eventType, _ := eventData["event_type"].(string)
	aggregateID, _ := eventData["aggregate_id"].(string)
	sourceNode, _ := eventData["source_node"].(string)

	if eventID == "" || eventType == "" {
		return errors.ValidationError("invalid cluster event data")
	}

	// Create a cluster event representation
	clusterEvent := &ClusterDomainEvent{
		id:          eventID,
		eventType:   eventType,
		aggregateID: aggregateID,
		occurredAt:  message.Timestamp,
		sourceNode:  sourceNode,
		payload:     eventData,
	}

	// Process the event locally (without re-broadcasting)
	if err := bus.InMemoryEventBus.Publish(ctx, clusterEvent); err != nil {
		return errors.Wrap(err, "failed to process cluster event locally")
	}

	bus.mu.Lock()
	bus.clusterStats.ClusterEvents++
	bus.mu.Unlock()

	return nil
}

// shouldBroadcastToCluster determines if an event should be broadcast to cluster
func (bus *DistributedEventBus) shouldBroadcastToCluster(event domain.DomainEvent) bool {
	// Skip cluster events to avoid infinite loops
	if _, isClusterEvent := event.(*ClusterDomainEvent); isClusterEvent {
		return false
	}

	// Broadcast system events and domain events that are marked for clustering
	eventType := event.EventType()
	return eventType == "system.event" || 
		   eventType == "pipeline.created" ||
		   eventType == "pipeline.executed" ||
		   eventType == "plugin.registered" ||
		   eventType == "plugin.activated"
}

// GetClusterStats returns cluster event statistics
func (bus *DistributedEventBus) GetClusterStats() ClusterEventStats {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// Update nodes reached
	if bus.coordinator != nil {
		nodes := bus.coordinator.GetActiveNodes(context.Background())
		bus.clusterStats.NodesReached = len(nodes)
	}

	return bus.clusterStats
}

// ClusterDomainEvent represents a domain event received from cluster
type ClusterDomainEvent struct {
	id          string
	eventType   string
	aggregateID string
	occurredAt  time.Time
	sourceNode  string
	payload     map[string]interface{}
}

func (e *ClusterDomainEvent) EventID() string     { return e.id }
func (e *ClusterDomainEvent) EventType() string  { return e.eventType }
func (e *ClusterDomainEvent) AggregateID() string { return e.aggregateID }
func (e *ClusterDomainEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *ClusterDomainEvent) GetSourceNode() string { return e.sourceNode }
func (e *ClusterDomainEvent) GetPayload() map[string]interface{} { return e.payload }

// EventBusBuilder helps build event bus configurations
type EventBusBuilder struct {
	maxWorkers      int
	bufferSize      int
	retryPolicy     *RetryPolicy
	eventStore      EventStore
	coordinator     ClusterCoordinator
	enableClustering bool
}

// NewEventBusBuilder creates a new event bus builder
func NewEventBusBuilder() *EventBusBuilder {
	return &EventBusBuilder{
		maxWorkers:       4,
		bufferSize:       1000,
		retryPolicy:      DefaultRetryPolicy(),
		enableClustering: false,
	}
}

// WithMaxWorkers sets the maximum number of worker goroutines
func (b *EventBusBuilder) WithMaxWorkers(workers int) *EventBusBuilder {
	b.maxWorkers = workers
	return b
}

// WithBufferSize sets the event queue buffer size
func (b *EventBusBuilder) WithBufferSize(size int) *EventBusBuilder {
	b.bufferSize = size
	return b
}

// WithRetryPolicy sets the retry policy
func (b *EventBusBuilder) WithRetryPolicy(policy *RetryPolicy) *EventBusBuilder {
	b.retryPolicy = policy
	return b
}

// WithEventStore sets the event store
func (b *EventBusBuilder) WithEventStore(store EventStore) *EventBusBuilder {
	b.eventStore = store
	return b
}

// WithClusterCoordinator enables clustering with the provided coordinator
func (b *EventBusBuilder) WithClusterCoordinator(coordinator ClusterCoordinator) *EventBusBuilder {
	b.coordinator = coordinator
	b.enableClustering = true
	return b
}

// WithClustering enables or disables clustering
func (b *EventBusBuilder) WithClustering(enabled bool) *EventBusBuilder {
	b.enableClustering = enabled
	return b
}

// Build creates the event bus
func (b *EventBusBuilder) Build() result.Result[EventBus] {
	if b.enableClustering && b.coordinator != nil {
		// Build distributed event bus
		bus := NewDistributedEventBus(b.maxWorkers, b.bufferSize, b.coordinator)
		bus.InMemoryEventBus.retryPolicy = b.retryPolicy
		return result.Success[EventBus](bus)
	} else {
		// Build local event bus
		bus := NewInMemoryEventBus(b.maxWorkers, b.bufferSize)
		bus.retryPolicy = b.retryPolicy
		return result.Success[EventBus](bus)
	}
}