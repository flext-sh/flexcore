// Package entities contains domain events for pipeline operations
package entities

import (
	"time"

	"github.com/flext/flexcore/domain"
)

// PipelineCreatedEvent is raised when a pipeline is created
type PipelineCreatedEvent struct {
	domain.BaseDomainEvent
	PipelineID   PipelineID
	PipelineName string
	Owner        string
}

// NewPipelineCreatedEvent creates a new pipeline created event
func NewPipelineCreatedEvent(pipelineID PipelineID, name, owner string) PipelineCreatedEvent {
	return PipelineCreatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineCreated", pipelineID.String()),
		PipelineID:      pipelineID,
		PipelineName:    name,
		Owner:           owner,
	}
}

// PipelineActivatedEvent is raised when a pipeline is activated
type PipelineActivatedEvent struct {
	domain.BaseDomainEvent
	PipelineID   PipelineID
	PipelineName string
}

// NewPipelineActivatedEvent creates a new pipeline activated event
func NewPipelineActivatedEvent(pipelineID PipelineID, name string) PipelineActivatedEvent {
	return PipelineActivatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineActivated", pipelineID.String()),
		PipelineID:      pipelineID,
		PipelineName:    name,
	}
}

// PipelineDeactivatedEvent is raised when a pipeline is deactivated
type PipelineDeactivatedEvent struct {
	domain.BaseDomainEvent
	PipelineID   PipelineID
	PipelineName string
}

// NewPipelineDeactivatedEvent creates a new pipeline deactivated event
func NewPipelineDeactivatedEvent(pipelineID PipelineID, name string) PipelineDeactivatedEvent {
	return PipelineDeactivatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineDeactivated", pipelineID.String()),
		PipelineID:      pipelineID,
		PipelineName:    name,
	}
}

// PipelineStartedEvent is raised when a pipeline starts execution
type PipelineStartedEvent struct {
	domain.BaseDomainEvent
	PipelineID   PipelineID
	PipelineName string
	StartedAt    time.Time
}

// NewPipelineStartedEvent creates a new pipeline started event
func NewPipelineStartedEvent(pipelineID PipelineID, name string, startedAt time.Time) PipelineStartedEvent {
	return PipelineStartedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineStarted", pipelineID.String()),
		PipelineID:      pipelineID,
		PipelineName:    name,
		StartedAt:       startedAt,
	}
}

// PipelineCompletedEvent is raised when a pipeline completes successfully
type PipelineCompletedEvent struct {
	domain.BaseDomainEvent
	PipelineID   PipelineID
	PipelineName string
	CompletedAt  time.Time
}

// NewPipelineCompletedEvent creates a new pipeline completed event
func NewPipelineCompletedEvent(pipelineID PipelineID, name string, completedAt time.Time) PipelineCompletedEvent {
	return PipelineCompletedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineCompleted", pipelineID.String()),
		PipelineID:      pipelineID,
		PipelineName:    name,
		CompletedAt:     completedAt,
	}
}

// PipelineFailedEvent is raised when a pipeline fails
type PipelineFailedEvent struct {
	domain.BaseDomainEvent
	PipelineID   PipelineID
	PipelineName string
	Reason       string
	FailedAt     time.Time
}

// NewPipelineFailedEvent creates a new pipeline failed event
func NewPipelineFailedEvent(pipelineID PipelineID, name, reason string, failedAt time.Time) PipelineFailedEvent {
	return PipelineFailedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineFailed", pipelineID.String()),
		PipelineID:      pipelineID,
		PipelineName:    name,
		Reason:          reason,
		FailedAt:        failedAt,
	}
}

// PipelineStepAddedEvent is raised when a step is added to a pipeline
type PipelineStepAddedEvent struct {
	domain.BaseDomainEvent
	PipelineID PipelineID
	StepID     string
	StepName   string
}

// NewPipelineStepAddedEvent creates a new pipeline step added event
func NewPipelineStepAddedEvent(pipelineID PipelineID, stepID, stepName string) PipelineStepAddedEvent {
	return PipelineStepAddedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineStepAdded", pipelineID.String()),
		PipelineID:      pipelineID,
		StepID:          stepID,
		StepName:        stepName,
	}
}

// PipelineStepRemovedEvent is raised when a step is removed from a pipeline
type PipelineStepRemovedEvent struct {
	domain.BaseDomainEvent
	PipelineID PipelineID
	StepName   string
}

// NewPipelineStepRemovedEvent creates a new pipeline step removed event
func NewPipelineStepRemovedEvent(pipelineID PipelineID, stepName string) PipelineStepRemovedEvent {
	return PipelineStepRemovedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineStepRemoved", pipelineID.String()),
		PipelineID:      pipelineID,
		StepName:        stepName,
	}
}

// PipelineScheduleSetEvent is raised when a pipeline schedule is set
type PipelineScheduleSetEvent struct {
	domain.BaseDomainEvent
	PipelineID     PipelineID
	CronExpression string
	Timezone       string
}

// NewPipelineScheduleSetEvent creates a new pipeline schedule set event
func NewPipelineScheduleSetEvent(pipelineID PipelineID, cronExpression, timezone string) PipelineScheduleSetEvent {
	return PipelineScheduleSetEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineScheduleSet", pipelineID.String()),
		PipelineID:      pipelineID,
		CronExpression:  cronExpression,
		Timezone:        timezone,
	}
}

// PipelineScheduleClearedEvent is raised when a pipeline schedule is cleared
type PipelineScheduleClearedEvent struct {
	domain.BaseDomainEvent
	PipelineID PipelineID
}

// NewPipelineScheduleClearedEvent creates a new pipeline schedule cleared event
func NewPipelineScheduleClearedEvent(pipelineID PipelineID) PipelineScheduleClearedEvent {
	return PipelineScheduleClearedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("PipelineScheduleCleared", pipelineID.String()),
		PipelineID:      pipelineID,
	}
}