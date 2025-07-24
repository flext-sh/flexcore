// Package entities contains domain entities for FlexCore
package entities

import (
	"time"

	"github.com/flext/flexcore/internal/domain"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
	"github.com/google/uuid"
)

// PipelineID represents a unique pipeline identifier
type PipelineID string

// NewPipelineID creates a new pipeline ID
func NewPipelineID() PipelineID {
	return PipelineID(uuid.New().String())
}

// String returns the string representation of the pipeline ID
func (id PipelineID) String() string {
	return string(id)
}

// PipelineStatus represents the status of a pipeline
type PipelineStatus int

const (
	// Pipeline status constants
	unknownStatusString = "unknown"

	// Default configuration constants
	DefaultMaxRetries     = 3
	DefaultTimeoutMinutes = 30

	PipelineStatusDraft PipelineStatus = iota
	PipelineStatusActive
	PipelineStatusRunning
	PipelineStatusCompleted
	PipelineStatusFailed
	PipelineStatusPaused
	PipelineStatusArchived
)

// String returns the string representation of the pipeline status
func (s PipelineStatus) String() string {
	switch s {
	case PipelineStatusDraft:
		return "draft"
	case PipelineStatusActive:
		return "active"
	case PipelineStatusRunning:
		return "running"
	case PipelineStatusCompleted:
		return "completed"
	case PipelineStatusFailed:
		return "failed"
	case PipelineStatusPaused:
		return "paused"
	case PipelineStatusArchived:
		return "archived"
	default:
		return unknownStatusString
	}
}

// PipelineStep represents a step in a pipeline
type PipelineStep struct {
	ID         string
	Name       string
	Type       string
	Config     map[string]interface{}
	DependsOn  []string
	RetryCount int
	MaxRetries int
	Timeout    time.Duration
	IsEnabled  bool
	CreatedAt  time.Time
}

// NewPipelineStep creates a new pipeline step
func NewPipelineStep(name, stepType string) PipelineStep {
	return PipelineStep{
		ID:         uuid.New().String(),
		Name:       name,
		Type:       stepType,
		Config:     make(map[string]interface{}),
		DependsOn:  make([]string, 0),
		RetryCount: 0,
		MaxRetries: DefaultMaxRetries,
		Timeout:    time.Minute * DefaultTimeoutMinutes,
		IsEnabled:  true,
		CreatedAt:  time.Now(),
	}
}

// Pipeline represents a data processing pipeline
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

// PipelineSchedule represents a pipeline schedule
type PipelineSchedule struct {
	CronExpression string
	Timezone       string
	IsEnabled      bool
	CreatedAt      time.Time
}

// NewPipeline creates a new pipeline
func NewPipeline(name, description, owner string) result.Result[*Pipeline] {
	if name == "" {
		return result.Failure[*Pipeline](errors.ValidationError("pipeline name cannot be empty"))
	}

	if owner == "" {
		return result.Failure[*Pipeline](errors.ValidationError("pipeline owner cannot be empty"))
	}

	id := NewPipelineID()
	pipeline := &Pipeline{
		AggregateRoot: domain.NewAggregateRoot(id),
		Name:          name,
		Description:   description,
		Status:        PipelineStatusDraft,
		Steps:         make([]PipelineStep, 0),
		Tags:          make([]string, 0),
		Owner:         owner,
	}

	// Raise domain event
	event := NewPipelineCreatedEvent(id, name, owner)
	pipeline.RaiseEvent(event)

	return result.Success(pipeline)
}

// AddStep adds a step to the pipeline
func (p *Pipeline) AddStep(step PipelineStep) result.Result[bool] {
	if step.Name == "" {
		return result.Failure[bool](errors.ValidationError("step name cannot be empty"))
	}

	// Check for duplicate step names
	for _, existingStep := range p.Steps {
		if existingStep.Name == step.Name {
			return result.Failure[bool](errors.AlreadyExistsError("step with name " + step.Name))
		}
	}

	// Validate dependencies
	for _, dependency := range step.DependsOn {
		if !p.hasStep(dependency) {
			return result.Failure[bool](errors.ValidationError("dependency step not found: " + dependency))
		}
	}

	p.Steps = append(p.Steps, step)
	p.Touch()

	// Raise domain event
	event := NewPipelineStepAddedEvent(p.ID, step.ID, step.Name)
	p.RaiseEvent(event)

	return result.Success(true)
}

// RemoveStep removes a step from the pipeline
func (p *Pipeline) RemoveStep(stepName string) result.Result[bool] {
	stepIndex := -1
	for i, step := range p.Steps {
		if step.Name == stepName {
			stepIndex = i
			break
		}
	}

	if stepIndex == -1 {
		return result.Failure[bool](errors.NotFoundError("step " + stepName))
	}

	// Check if other steps depend on this step
	for _, step := range p.Steps {
		for _, dependency := range step.DependsOn {
			if dependency == stepName {
				return result.Failure[bool](errors.ValidationError("cannot remove step: other steps depend on it"))
			}
		}
	}

	// Remove the step
	p.Steps = append(p.Steps[:stepIndex], p.Steps[stepIndex+1:]...)
	p.Touch()

	// Raise domain event
	event := NewPipelineStepRemovedEvent(p.ID, stepName)
	p.RaiseEvent(event)

	return result.Success(true)
}

// Activate activates the pipeline
func (p *Pipeline) Activate() result.Result[bool] {
	if p.Status == PipelineStatusActive {
		return result.Failure[bool](errors.ValidationError("pipeline is already active"))
	}

	if len(p.Steps) == 0 {
		return result.Failure[bool](errors.ValidationError("cannot activate pipeline without steps"))
	}

	p.Status = PipelineStatusActive
	p.Touch()

	// Raise domain event
	event := NewPipelineActivatedEvent(p.ID, p.Name)
	p.RaiseEvent(event)

	return result.Success(true)
}

// Deactivate deactivates the pipeline
func (p *Pipeline) Deactivate() result.Result[bool] {
	if p.Status == PipelineStatusRunning {
		return result.Failure[bool](errors.ValidationError("cannot deactivate running pipeline"))
	}

	p.Status = PipelineStatusDraft
	p.Touch()

	// Raise domain event
	event := NewPipelineDeactivatedEvent(p.ID, p.Name)
	p.RaiseEvent(event)

	return result.Success(true)
}

// Start starts the pipeline execution
func (p *Pipeline) Start() result.Result[bool] {
	if p.Status != PipelineStatusActive {
		return result.Failure[bool](errors.ValidationError("can only start active pipelines"))
	}

	if p.Status == PipelineStatusRunning {
		return result.Failure[bool](errors.ValidationError("pipeline is already running"))
	}

	p.Status = PipelineStatusRunning
	now := time.Now()
	p.LastRunAt = &now
	p.Touch()

	// Raise domain event
	event := NewPipelineStartedEvent(p.ID, p.Name, now)
	p.RaiseEvent(event)

	return result.Success(true)
}

// Complete marks the pipeline as completed
func (p *Pipeline) Complete() result.Result[bool] {
	if p.Status != PipelineStatusRunning {
		return result.Failure[bool](errors.ValidationError("can only complete running pipelines"))
	}

	p.Status = PipelineStatusCompleted
	p.Touch()

	// Raise domain event
	event := NewPipelineCompletedEvent(p.ID, p.Name, time.Now())
	p.RaiseEvent(event)

	return result.Success(true)
}

// Fail marks the pipeline as failed
func (p *Pipeline) Fail(reason string) result.Result[bool] {
	if p.Status != PipelineStatusRunning {
		return result.Failure[bool](errors.ValidationError("can only fail running pipelines"))
	}

	p.Status = PipelineStatusFailed
	p.Touch()

	// Raise domain event
	event := NewPipelineFailedEvent(p.ID, p.Name, reason, time.Now())
	p.RaiseEvent(event)

	return result.Success(true)
}

// SetSchedule sets the pipeline schedule
func (p *Pipeline) SetSchedule(cronExpression, timezone string) result.Result[bool] {
	if cronExpression == "" {
		return result.Failure[bool](errors.ValidationError("cron expression cannot be empty"))
	}

	p.Schedule = &PipelineSchedule{
		CronExpression: cronExpression,
		Timezone:       timezone,
		IsEnabled:      true,
		CreatedAt:      time.Now(),
	}
	p.Touch()

	// Raise domain event
	event := NewPipelineScheduleSetEvent(p.ID, cronExpression, timezone)
	p.RaiseEvent(event)

	return result.Success(true)
}

// ClearSchedule removes the pipeline schedule
func (p *Pipeline) ClearSchedule() {
	p.Schedule = nil
	p.Touch()

	// Raise domain event
	event := NewPipelineScheduleClearedEvent(p.ID)
	p.RaiseEvent(event)
}

// AddTag adds a tag to the pipeline
func (p *Pipeline) AddTag(tag string) {
	for _, existingTag := range p.Tags {
		if existingTag == tag {
			return // Tag already exists
		}
	}
	p.Tags = append(p.Tags, tag)
	p.Touch()
}

// RemoveTag removes a tag from the pipeline
func (p *Pipeline) RemoveTag(tag string) {
	for i, existingTag := range p.Tags {
		if existingTag == tag {
			p.Tags = append(p.Tags[:i], p.Tags[i+1:]...)
			p.Touch()
			break
		}
	}
}

// hasStep checks if a step with the given name exists
func (p *Pipeline) hasStep(stepName string) bool {
	for _, step := range p.Steps {
		if step.Name == stepName {
			return true
		}
	}
	return false
}

// GetStep returns a step by name
func (p *Pipeline) GetStep(stepName string) (PipelineStep, bool) {
	for _, step := range p.Steps {
		if step.Name == stepName {
			return step, true
		}
	}
	return PipelineStep{}, false
}

// IsScheduled returns true if the pipeline has a schedule
func (p *Pipeline) IsScheduled() bool {
	return p.Schedule != nil && p.Schedule.IsEnabled
}

// CanExecute returns true if the pipeline can be executed
func (p *Pipeline) CanExecute() bool {
	return p.Status == PipelineStatusActive && len(p.Steps) > 0
}

// Validate validates the pipeline
func (p *Pipeline) Validate() result.Result[bool] {
	if p.Name == "" {
		return result.Failure[bool](errors.ValidationError("name is required"))
	}

	if p.Owner == "" {
		return result.Failure[bool](errors.ValidationError("owner is required"))
	}

	if len(p.Steps) == 0 {
		return result.Failure[bool](errors.ValidationError("at least one step is required"))
	}

	return result.Success(true)
}

// HasTag checks if the pipeline has a specific tag
func (p *Pipeline) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Domain Events

// PipelineCreatedEvent represents a pipeline created event
type PipelineCreatedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
	Owner        string
}

// NewPipelineCreatedEvent creates a new pipeline created event
func NewPipelineCreatedEvent(pipelineID PipelineID, name, owner string) *PipelineCreatedEvent {
	return &PipelineCreatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.created", pipelineID.String()),
		PipelineName:    name,
		Owner:           owner,
	}
}

// PipelineStepAddedEvent represents a pipeline step added event
type PipelineStepAddedEvent struct {
	domain.BaseDomainEvent
	StepID   string
	StepName string
}

// NewPipelineStepAddedEvent creates a new pipeline step added event
func NewPipelineStepAddedEvent(pipelineID PipelineID, stepID, stepName string) *PipelineStepAddedEvent {
	return &PipelineStepAddedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.step.added", pipelineID.String()),
		StepID:          stepID,
		StepName:        stepName,
	}
}

// PipelineStepRemovedEvent represents a pipeline step removed event
type PipelineStepRemovedEvent struct {
	domain.BaseDomainEvent
	StepName string
}

// NewPipelineStepRemovedEvent creates a new pipeline step removed event
func NewPipelineStepRemovedEvent(pipelineID PipelineID, stepName string) *PipelineStepRemovedEvent {
	return &PipelineStepRemovedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.step.removed", pipelineID.String()),
		StepName:        stepName,
	}
}

// PipelineActivatedEvent represents a pipeline activated event
type PipelineActivatedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
}

// NewPipelineActivatedEvent creates a new pipeline activated event
func NewPipelineActivatedEvent(pipelineID PipelineID, name string) *PipelineActivatedEvent {
	return &PipelineActivatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.activated", pipelineID.String()),
		PipelineName:    name,
	}
}

// PipelineDeactivatedEvent represents a pipeline deactivated event
type PipelineDeactivatedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
}

// NewPipelineDeactivatedEvent creates a new pipeline deactivated event
func NewPipelineDeactivatedEvent(pipelineID PipelineID, name string) *PipelineDeactivatedEvent {
	return &PipelineDeactivatedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.deactivated", pipelineID.String()),
		PipelineName:    name,
	}
}

// PipelineStartedEvent represents a pipeline started event
type PipelineStartedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
	StartTime    time.Time
}

// NewPipelineStartedEvent creates a new pipeline started event
func NewPipelineStartedEvent(pipelineID PipelineID, name string, startTime time.Time) *PipelineStartedEvent {
	return &PipelineStartedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.started", pipelineID.String()),
		PipelineName:    name,
		StartTime:       startTime,
	}
}

// PipelineCompletedEvent represents a pipeline completed event
type PipelineCompletedEvent struct {
	domain.BaseDomainEvent
	PipelineName  string
	CompletedTime time.Time
}

// NewPipelineCompletedEvent creates a new pipeline completed event
func NewPipelineCompletedEvent(pipelineID PipelineID, name string, completedTime time.Time) *PipelineCompletedEvent {
	return &PipelineCompletedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.completed", pipelineID.String()),
		PipelineName:    name,
		CompletedTime:   completedTime,
	}
}

// PipelineFailedEvent represents a pipeline failed event
type PipelineFailedEvent struct {
	domain.BaseDomainEvent
	PipelineName string
	Reason       string
	FailedTime   time.Time
}

// NewPipelineFailedEvent creates a new pipeline failed event
func NewPipelineFailedEvent(pipelineID PipelineID, name, reason string, failedTime time.Time) *PipelineFailedEvent {
	return &PipelineFailedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.failed", pipelineID.String()),
		PipelineName:    name,
		Reason:          reason,
		FailedTime:      failedTime,
	}
}

// PipelineScheduleSetEvent represents a pipeline schedule set event
type PipelineScheduleSetEvent struct {
	domain.BaseDomainEvent
	CronExpression string
	Timezone       string
}

// NewPipelineScheduleSetEvent creates a new pipeline schedule set event
func NewPipelineScheduleSetEvent(pipelineID PipelineID, cronExpression, timezone string) *PipelineScheduleSetEvent {
	return &PipelineScheduleSetEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.schedule.set", pipelineID.String()),
		CronExpression:  cronExpression,
		Timezone:        timezone,
	}
}

// PipelineScheduleClearedEvent represents a pipeline schedule cleared event
type PipelineScheduleClearedEvent struct {
	domain.BaseDomainEvent
}

// NewPipelineScheduleClearedEvent creates a new pipeline schedule cleared event
func NewPipelineScheduleClearedEvent(pipelineID PipelineID) *PipelineScheduleClearedEvent {
	return &PipelineScheduleClearedEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("pipeline.schedule.cleared", pipelineID.String()),
	}
}
