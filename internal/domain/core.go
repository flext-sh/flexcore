// Package domain contains the core business logic and domain models
package domain

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DEPRECATED: Legacy imports will be removed in v2.0.0
// Use the new internal/domain structure instead
func init() {
	log.Println("WARNING: Using legacy FlexCore. Migrate to internal/domain structure.")
}

// Event represents a domain event in the system
type Event struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	AggregateID  string                 `json:"aggregate_id"`
	Data         map[string]interface{} `json:"data"`
	Timestamp    time.Time              `json:"timestamp"`
	Version      int                    `json:"version"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// NewEvent creates a new domain event
func NewEvent(eventType, aggregateID string, data map[string]interface{}) *Event {
	return &Event{
		ID:          uuid.New().String(),
		Type:        eventType,
		AggregateID: aggregateID,
		Data:        data,
		Timestamp:   time.Now(),
		Version:     1,
		Metadata:    make(map[string]interface{}),
	}
}

// Message represents a message in the distributed queue
type Message struct {
	ID        string                 `json:"id"`
	Queue     string                 `json:"queue"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	Attempts  int                    `json:"attempts"`
	Status    string                 `json:"status"`
}

// NewMessage creates a new message
func NewMessage(queue string, data map[string]interface{}) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Queue:     queue,
		Data:      data,
		CreatedAt: time.Now(),
		Attempts:  0,
		Status:    "pending",
	}
}

// FlexCoreConfig represents the core configuration
type FlexCoreConfig struct {
	WindmillURL       string `json:"windmill_url"`
	WindmillToken     string `json:"windmill_token"`
	WindmillWorkspace string `json:"windmill_workspace"`
	ClusterName       string `json:"cluster_name"`
	NodeID            string `json:"node_id"`
	ClusterNodes      []string `json:"cluster_nodes"`
	PluginDirectory   string `json:"plugin_directory"`
	MaxConcurrentJobs int    `json:"max_concurrent_jobs"`
	EventBufferSize   int    `json:"event_buffer_size"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *FlexCoreConfig {
	return &FlexCoreConfig{
		WindmillURL:       "http://localhost:8000",
		WindmillWorkspace: "default",
		ClusterName:       "flexcore-cluster",
		NodeID:            "node-" + uuid.New().String()[:8],
		ClusterNodes:      []string{},
		PluginDirectory:   "./plugins",
		MaxConcurrentJobs: 100,
		EventBufferSize:   10000,
	}
}

// Result represents a result that can either be a success or failure
type Result[T any] struct {
	value T
	err   error
}

// NewSuccess creates a successful result
func NewSuccess[T any](value T) *Result[T] {
	return &Result[T]{value: value, err: nil}
}

// NewFailure creates a failed result
func NewFailure[T any](err error) *Result[T] {
	var zero T
	return &Result[T]{value: zero, err: err}
}

// IsSuccess returns true if the result is successful
func (r *Result[T]) IsSuccess() bool {
	return r.err == nil
}

// IsFailure returns true if the result is a failure
func (r *Result[T]) IsFailure() bool {
	return r.err != nil
}

// Value returns the value if successful, panics if failure
func (r *Result[T]) Value() T {
	if r.err != nil {
		panic(fmt.Sprintf("Result is failure: %v", r.err))
	}
	return r.value
}

// Error returns the error if failure, nil if success
func (r *Result[T]) Error() error {
	return r.err
}

// FlexCore represents the main distributed event processing engine
type FlexCore struct {
	config         *FlexCoreConfig
	eventQueue     chan *Event
	messageQueues  map[string]chan *Message
	plugins        map[string]interface{}
	workflows      map[string]interface{}
	customParams   map[string]interface{}
	
	mu             sync.RWMutex
	running        bool
	stopChan       chan struct{}
	waitGroup      sync.WaitGroup
}

// NewFlexCore creates a new FlexCore instance
func NewFlexCore(config *FlexCoreConfig) *Result[*FlexCore] {
	if config == nil {
		return NewFailure[*FlexCore](fmt.Errorf("config cannot be nil"))
	}

	core := &FlexCore{
		config:        config,
		eventQueue:    make(chan *Event, config.EventBufferSize),
		messageQueues: make(map[string]chan *Message),
		plugins:       make(map[string]interface{}),
		workflows:     make(map[string]interface{}),
		customParams:  make(map[string]interface{}),
		stopChan:      make(chan struct{}),
	}

	// Initialize custom parameters
	core.customParams["config"] = config
	core.customParams["node_id"] = config.NodeID

	return NewSuccess(core)
}

// Start starts the FlexCore engine
func (fc *FlexCore) Start(ctx context.Context) *Result[bool] {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.running {
		return NewFailure[bool](fmt.Errorf("FlexCore is already running"))
	}

	fc.running = true
	
	// Start event processing
	fc.waitGroup.Add(1)
	go fc.processEvents(ctx)

	return NewSuccess(true)
}

// Stop stops the FlexCore engine
func (fc *FlexCore) Stop(ctx context.Context) *Result[bool] {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if !fc.running {
		return NewSuccess(true)
	}

	fc.running = false
	close(fc.stopChan)
	fc.waitGroup.Wait()

	return NewSuccess(true)
}

// SendEvent sends an event to the system
func (fc *FlexCore) SendEvent(ctx context.Context, event *Event) *Result[bool] {
	if event == nil {
		return NewFailure[bool](fmt.Errorf("event cannot be nil"))
	}

	select {
	case fc.eventQueue <- event:
		return NewSuccess(true)
	case <-ctx.Done():
		return NewFailure[bool](ctx.Err())
	default:
		return NewFailure[bool](fmt.Errorf("event queue is full"))
	}
}

// SendMessage sends a message to a specific queue
func (fc *FlexCore) SendMessage(ctx context.Context, queue string, message *Message) *Result[bool] {
	if message == nil {
		return NewFailure[bool](fmt.Errorf("message cannot be nil"))
	}

	fc.mu.Lock()
	queueChan, exists := fc.messageQueues[queue]
	if !exists {
		queueChan = make(chan *Message, 1000)
		fc.messageQueues[queue] = queueChan
	}
	fc.mu.Unlock()

	select {
	case queueChan <- message:
		return NewSuccess(true)
	case <-ctx.Done():
		return NewFailure[bool](ctx.Err())
	default:
		return NewFailure[bool](fmt.Errorf("queue %s is full", queue))
	}
}

// ReceiveMessages receives messages from a queue
func (fc *FlexCore) ReceiveMessages(ctx context.Context, queue string, maxMessages int) *Result[[]*Message] {
	fc.mu.RLock()
	queueChan, exists := fc.messageQueues[queue]
	fc.mu.RUnlock()

	if !exists {
		return NewSuccess([]*Message{})
	}

	messages := make([]*Message, 0, maxMessages)
	for i := 0; i < maxMessages; i++ {
		select {
		case msg := <-queueChan:
			messages = append(messages, msg)
		default:
			// No more messages available
			break
		}
	}

	return NewSuccess(messages)
}

// ExecuteWorkflow executes a workflow
func (fc *FlexCore) ExecuteWorkflow(ctx context.Context, path string, input map[string]interface{}) *Result[string] {
	jobID := uuid.New().String()
	
	// Simulate workflow execution
	fc.workflows[jobID] = map[string]interface{}{
		"path":       path,
		"input":      input,
		"status":     "running",
		"started_at": time.Now(),
	}

	return NewSuccess(jobID)
}

// GetWorkflowStatus gets the status of a workflow
func (fc *FlexCore) GetWorkflowStatus(ctx context.Context, jobID string) *Result[map[string]interface{}] {
	fc.mu.RLock()
	workflow, exists := fc.workflows[jobID]
	fc.mu.RUnlock()

	if !exists {
		return NewFailure[map[string]interface{}](fmt.Errorf("workflow %s not found", jobID))
	}

	return NewSuccess(workflow.(map[string]interface{}))
}

// GetClusterStatus returns the cluster status
func (fc *FlexCore) GetClusterStatus(ctx context.Context) *Result[map[string]interface{}] {
	status := map[string]interface{}{
		"cluster_name": fc.config.ClusterName,
		"node_id":      fc.config.NodeID,
		"cluster_size": len(fc.config.ClusterNodes) + 1, // +1 for this node
		"status":       "healthy",
		"uptime":       time.Since(time.Now()).String(), // Would track actual uptime
	}

	return NewSuccess(status)
}

// GetCustomParameter gets a custom parameter
func (fc *FlexCore) GetCustomParameter(key string) interface{} {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.customParams[key]
}

// SetCustomParameter sets a custom parameter
func (fc *FlexCore) SetCustomParameter(key string, value interface{}) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.customParams[key] = value
}

// processEvents processes events from the event queue
func (fc *FlexCore) processEvents(ctx context.Context) {
	defer fc.waitGroup.Done()

	for {
		select {
		case event := <-fc.eventQueue:
			// Process event
			log.Printf("Processing event: %s (Type: %s)", event.ID, event.Type)
			
		case <-fc.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}