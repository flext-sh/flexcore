// Package commands provides pipeline-specific commands
package commands

import (
	"context"
	"time"

	"github.com/flext/flexcore/internal/domain/entities"
	"github.com/flext/flexcore/pkg/result"
	"github.com/flext/flexcore/shared/errors"
)

// CreatePipelineCommand represents a command to create a new pipeline
type CreatePipelineCommand struct {
	BaseCommand
	Name        string
	Description string
	Owner       string
	Tags        []string
}

// NewCreatePipelineCommand creates a new create pipeline command
func NewCreatePipelineCommand(name, description, owner string, tags []string) CreatePipelineCommand {
	return CreatePipelineCommand{
		BaseCommand: NewBaseCommand("CreatePipeline"),
		Name:        name,
		Description: description,
		Owner:       owner,
		Tags:        tags,
	}
}

// CreatePipelineCommandHandler handles pipeline creation commands
type CreatePipelineCommandHandler struct {
	repository PipelineRepository
	eventBus   EventBus
}

// PipelineRepository represents a repository for pipelines
type PipelineRepository interface {
	Save(ctx context.Context, pipeline *entities.Pipeline) error
	FindByID(ctx context.Context, id entities.PipelineID) (*entities.Pipeline, error)
	FindByName(ctx context.Context, name string) (*entities.Pipeline, error)
	Delete(ctx context.Context, id entities.PipelineID) error
	List(ctx context.Context, limit, offset int) ([]*entities.Pipeline, error)
}

// EventBus represents an event bus for publishing domain events
type EventBus interface {
	Publish(ctx context.Context, event interface{}) error
}

// NewCreatePipelineCommandHandler creates a new create pipeline command handler
func NewCreatePipelineCommandHandler(repository PipelineRepository, eventBus EventBus) *CreatePipelineCommandHandler {
	return &CreatePipelineCommandHandler{
		repository: repository,
		eventBus:   eventBus,
	}
}

// Handle handles the create pipeline command
func (h *CreatePipelineCommandHandler) Handle(ctx context.Context, command CreatePipelineCommand) result.Result[interface{}] {
	// Check if pipeline with same name already exists
	if existingPipeline, _ := h.repository.FindByName(ctx, command.Name); existingPipeline != nil {
		return result.Failure[interface{}](errors.AlreadyExistsError("pipeline with name " + command.Name))
	}

	// Create new pipeline
	pipelineResult := entities.NewPipeline(command.Name, command.Description, command.Owner)
	if pipelineResult.IsFailure() {
		return result.Failure[interface{}](pipelineResult.Error())
	}

	pipeline := pipelineResult.Value()

	// Add tags
	for _, tag := range command.Tags {
		pipeline.AddTag(tag)
	}

	// Save pipeline
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to save pipeline"))
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	return result.Success[interface{}](pipeline)
}

// AddPipelineStepCommand represents a command to add a step to a pipeline
type AddPipelineStepCommand struct {
	BaseCommand
	PipelineID entities.PipelineID
	StepName   string
	StepType   string
	Config     map[string]interface{}
	DependsOn  []string
	MaxRetries int
	Timeout    time.Duration
}

// NewAddPipelineStepCommand creates a new add pipeline step command
func NewAddPipelineStepCommand(pipelineID entities.PipelineID, stepName, stepType string) AddPipelineStepCommand {
	return AddPipelineStepCommand{
		BaseCommand: NewBaseCommand("AddPipelineStep"),
		PipelineID:  pipelineID,
		StepName:    stepName,
		StepType:    stepType,
		Config:      make(map[string]interface{}),
		DependsOn:   make([]string, 0),
		MaxRetries:  entities.DefaultMaxRetries,
		Timeout:     time.Minute * entities.DefaultTimeoutMinutes,
	}
}

// AddPipelineStepCommandHandler handles add pipeline step commands
type AddPipelineStepCommandHandler struct {
	repository PipelineRepository
	eventBus   EventBus
}

// NewAddPipelineStepCommandHandler creates a new add pipeline step command handler
func NewAddPipelineStepCommandHandler(repository PipelineRepository, eventBus EventBus) *AddPipelineStepCommandHandler {
	return &AddPipelineStepCommandHandler{
		repository: repository,
		eventBus:   eventBus,
	}
}

// Handle handles the add pipeline step command
func (h *AddPipelineStepCommandHandler) Handle(ctx context.Context, command AddPipelineStepCommand) result.Result[interface{}] {
	// Find pipeline
	pipeline, err := h.repository.FindByID(ctx, command.PipelineID)
	if err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to find pipeline"))
	}

	// Create step
	step := entities.NewPipelineStep(command.StepName, command.StepType)
	step.Config = command.Config
	step.DependsOn = command.DependsOn
	step.MaxRetries = command.MaxRetries
	step.Timeout = command.Timeout

	// Add step to pipeline
	addResult := pipeline.AddStep(step)
	if addResult.IsFailure() {
		return result.Failure[interface{}](addResult.Error())
	}

	// Save pipeline
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to save pipeline"))
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	return result.Success[interface{}](step)
}

// ExecutePipelineCommand represents a command to execute a pipeline
type ExecutePipelineCommand struct {
	BaseCommand
	PipelineID entities.PipelineID
	Parameters map[string]interface{}
}

// NewExecutePipelineCommand creates a new execute pipeline command
func NewExecutePipelineCommand(pipelineID entities.PipelineID, parameters map[string]interface{}) ExecutePipelineCommand {
	return ExecutePipelineCommand{
		BaseCommand: NewBaseCommand("ExecutePipeline"),
		PipelineID:  pipelineID,
		Parameters:  parameters,
	}
}

// ExecutePipelineCommandHandler handles pipeline execution commands
type ExecutePipelineCommandHandler struct {
	repository     PipelineRepository
	eventBus       EventBus
	workflowEngine WorkflowEngine
}

// WorkflowEngine represents a workflow execution engine
type WorkflowEngine interface {
	StartWorkflow(ctx context.Context, workflowName string, input interface{}) (string, error)
}

// NewExecutePipelineCommandHandler creates a new execute pipeline command handler
func NewExecutePipelineCommandHandler(repository PipelineRepository, eventBus EventBus, workflowEngine WorkflowEngine) *ExecutePipelineCommandHandler {
	return &ExecutePipelineCommandHandler{
		repository:     repository,
		eventBus:       eventBus,
		workflowEngine: workflowEngine,
	}
}

// Handle handles the execute pipeline command
func (h *ExecutePipelineCommandHandler) Handle(ctx context.Context, command ExecutePipelineCommand) result.Result[interface{}] {
	// Find pipeline
	pipeline, err := h.repository.FindByID(ctx, command.PipelineID)
	if err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to find pipeline"))
	}

	// Check if pipeline can be executed
	if !pipeline.CanExecute() {
		return result.Failure[interface{}](errors.ValidationError("pipeline cannot be executed"))
	}

	// Start pipeline execution
	startResult := pipeline.Start()
	if startResult.IsFailure() {
		return result.Failure[interface{}](startResult.Error())
	}

	// Save pipeline state
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to save pipeline"))
	}

	// Start workflow execution
	workflowInput := map[string]interface{}{
		"pipelineID": pipeline.ID.String(),
		"parameters": command.Parameters,
	}

	workflowID, err := h.workflowEngine.StartWorkflow(ctx, "pipeline-execution", workflowInput)
	if err != nil {
		// Mark pipeline as failed
		pipeline.Fail("failed to start workflow: " + err.Error())
		h.repository.Save(ctx, pipeline)
		return result.Failure[interface{}](errors.Wrap(err, "failed to start workflow"))
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	executionResult := map[string]interface{}{
		"pipelineID": pipeline.ID.String(),
		"workflowID": workflowID,
		"status":     pipeline.Status.String(),
	}

	return result.Success[interface{}](executionResult)
}

// ActivatePipelineCommand represents a command to activate a pipeline
type ActivatePipelineCommand struct {
	BaseCommand
	PipelineID entities.PipelineID
}

// NewActivatePipelineCommand creates a new activate pipeline command
func NewActivatePipelineCommand(pipelineID entities.PipelineID) ActivatePipelineCommand {
	return ActivatePipelineCommand{
		BaseCommand: NewBaseCommand("ActivatePipeline"),
		PipelineID:  pipelineID,
	}
}

// ActivatePipelineCommandHandler handles pipeline activation commands
type ActivatePipelineCommandHandler struct {
	repository PipelineRepository
	eventBus   EventBus
}

// NewActivatePipelineCommandHandler creates a new activate pipeline command handler
func NewActivatePipelineCommandHandler(repository PipelineRepository, eventBus EventBus) *ActivatePipelineCommandHandler {
	return &ActivatePipelineCommandHandler{
		repository: repository,
		eventBus:   eventBus,
	}
}

// Handle handles the activate pipeline command
func (h *ActivatePipelineCommandHandler) Handle(ctx context.Context, command ActivatePipelineCommand) result.Result[interface{}] {
	// Find pipeline
	pipeline, err := h.repository.FindByID(ctx, command.PipelineID)
	if err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to find pipeline"))
	}

	// Activate pipeline
	activateResult := pipeline.Activate()
	if activateResult.IsFailure() {
		return result.Failure[interface{}](activateResult.Error())
	}

	// Save pipeline
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to save pipeline"))
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	return result.Success[interface{}](pipeline)
}

// SetPipelineScheduleCommand represents a command to set a pipeline schedule
type SetPipelineScheduleCommand struct {
	BaseCommand
	PipelineID     entities.PipelineID
	CronExpression string
	Timezone       string
}

// NewSetPipelineScheduleCommand creates a new set pipeline schedule command
func NewSetPipelineScheduleCommand(pipelineID entities.PipelineID, cronExpression, timezone string) SetPipelineScheduleCommand {
	return SetPipelineScheduleCommand{
		BaseCommand:    NewBaseCommand("SetPipelineSchedule"),
		PipelineID:     pipelineID,
		CronExpression: cronExpression,
		Timezone:       timezone,
	}
}

// SetPipelineScheduleCommandHandler handles set pipeline schedule commands
type SetPipelineScheduleCommandHandler struct {
	repository PipelineRepository
	eventBus   EventBus
}

// NewSetPipelineScheduleCommandHandler creates a new set pipeline schedule command handler
func NewSetPipelineScheduleCommandHandler(repository PipelineRepository, eventBus EventBus) *SetPipelineScheduleCommandHandler {
	return &SetPipelineScheduleCommandHandler{
		repository: repository,
		eventBus:   eventBus,
	}
}

// Handle handles the set pipeline schedule command
func (h *SetPipelineScheduleCommandHandler) Handle(ctx context.Context, command SetPipelineScheduleCommand) result.Result[interface{}] {
	// Find pipeline
	pipeline, err := h.repository.FindByID(ctx, command.PipelineID)
	if err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to find pipeline"))
	}

	// Set schedule
	scheduleResult := pipeline.SetSchedule(command.CronExpression, command.Timezone)
	if scheduleResult.IsFailure() {
		return result.Failure[interface{}](scheduleResult.Error())
	}

	// Save pipeline
	if err := h.repository.Save(ctx, pipeline); err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to save pipeline"))
	}

	// Publish domain events
	for _, event := range pipeline.DomainEvents() {
		if err := h.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the command
		}
	}

	pipeline.ClearEvents()

	return result.Success[interface{}](pipeline.Schedule)
}
