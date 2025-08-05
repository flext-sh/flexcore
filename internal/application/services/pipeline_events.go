// Package services provides pipeline event constructors and commands
package services

import (
	"time"

	"github.com/google/uuid"
	"github.com/flext-sh/flexcore/internal/domain/services"
)

// PipelineExecutionStartedEvent represents a pipeline execution started event
type PipelineExecutionStartedEvent struct {
	eventID     uuid.UUID
	aggregateID string
	eventType   string
	eventData   map[string]interface{}
	occurredAt  time.Time
	version     int
}

// NewPipelineExecutionStartedEvent creates a new pipeline execution started event
func NewPipelineExecutionStartedEvent(pipelineID string, timestamp time.Time) services.DomainEvent {
	return &PipelineExecutionStartedEvent{
		eventID:     uuid.New(),
		aggregateID: pipelineID,
		eventType:   "pipeline.execution.started",
		eventData: map[string]interface{}{
			"pipeline_id": pipelineID,
			"started_at":  timestamp,
		},
		occurredAt: timestamp,
		version:    1,
	}
}

// Implement DomainEvent interface
func (e *PipelineExecutionStartedEvent) GetEventID() uuid.UUID {
	return e.eventID
}

func (e *PipelineExecutionStartedEvent) GetAggregateID() string {
	return e.aggregateID
}

func (e *PipelineExecutionStartedEvent) GetEventType() string {
	return e.eventType
}

func (e *PipelineExecutionStartedEvent) GetEventData() map[string]interface{} {
	return e.eventData
}

func (e *PipelineExecutionStartedEvent) GetOccurredAt() time.Time {
	return e.occurredAt
}

func (e *PipelineExecutionStartedEvent) GetVersion() int {
	return e.version
}

// PipelineExecutionCompletedEvent represents a pipeline execution completed event
type PipelineExecutionCompletedEvent struct {
	eventID     uuid.UUID
	aggregateID string
	eventType   string
	eventData   map[string]interface{}
	occurredAt  time.Time
	version     int
}

// NewPipelineExecutionCompletedEvent creates a new pipeline execution completed event
func NewPipelineExecutionCompletedEvent(pipelineID string, result interface{}) services.DomainEvent {
	return &PipelineExecutionCompletedEvent{
		eventID:     uuid.New(),
		aggregateID: pipelineID,
		eventType:   "pipeline.execution.completed",
		eventData: map[string]interface{}{
			"pipeline_id": pipelineID,
			"result":      result,
			"completed_at": time.Now(),
		},
		occurredAt: time.Now(),
		version:    1,
	}
}

// Implement DomainEvent interface
func (e *PipelineExecutionCompletedEvent) GetEventID() uuid.UUID {
	return e.eventID
}

func (e *PipelineExecutionCompletedEvent) GetAggregateID() string {
	return e.aggregateID
}

func (e *PipelineExecutionCompletedEvent) GetEventType() string {
	return e.eventType
}

func (e *PipelineExecutionCompletedEvent) GetEventData() map[string]interface{} {
	return e.eventData
}

func (e *PipelineExecutionCompletedEvent) GetOccurredAt() time.Time {
	return e.occurredAt
}

func (e *PipelineExecutionCompletedEvent) GetVersion() int {
	return e.version
}

// ExecutePipelineCommand represents a command to execute a pipeline
type ExecutePipelineCommand struct {
	commandID   uuid.UUID
	commandType string
	payload     map[string]interface{}
	timestamp   time.Time
}

// NewExecutePipelineCommand creates a new execute pipeline command
func NewExecutePipelineCommand(pipelineID string) services.Command {
	return &ExecutePipelineCommand{
		commandID:   uuid.New(),
		commandType: "pipeline.execute",
		payload: map[string]interface{}{
			"pipeline_id": pipelineID,
		},
		timestamp: time.Now(),
	}
}

// Implement Command interface
func (c *ExecutePipelineCommand) GetCommandID() uuid.UUID {
	return c.commandID
}

func (c *ExecutePipelineCommand) GetCommandType() string {
	return c.commandType
}

func (c *ExecutePipelineCommand) GetPayload() map[string]interface{} {
	return c.payload
}

func (c *ExecutePipelineCommand) GetTimestamp() time.Time {
	return c.timestamp
}