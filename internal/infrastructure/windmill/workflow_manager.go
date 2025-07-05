// Package windmill provides workflow management using Windmill platform
package windmill

import (
	"context"
	"fmt"
	"time"

	"github.com/flext/flexcore/pkg/errors"
	"github.com/flext/flexcore/pkg/result"
)

// WorkflowManager manages workflow lifecycle using Windmill
type WorkflowManager struct {
	client      *Client
	workflows   map[string]*WorkflowDefinition
	defaultTags map[string]string
}

// WorkflowDefinition represents a FlexCore workflow definition
type WorkflowDefinition struct {
	Path        string
	Name        string
	Description string
	Steps       []WorkflowStep
	Timeout     time.Duration
	RetryPolicy *RetryPolicy
	Tags        map[string]string
}

// WorkflowStep represents a step in a FlexCore workflow
type WorkflowStep struct {
	ID          string
	Name        string
	Type        StepType
	Script      string
	Language    string
	Input       map[string]interface{}
	Timeout     *int
	RetryCount  *int
	Dependencies []string
}

// StepType represents the type of workflow step
type StepType string

const (
	StepTypeScript     StepType = "script"
	StepTypeFlow       StepType = "flow"
	StepTypeForLoop    StepType = "forloop"
	StepTypeBranchOne  StepType = "branchone"
	StepTypeBranchAll  StepType = "branchall"
	StepTypeIdentity   StepType = "identity"
)

// RetryPolicy defines retry behavior for workflows
type RetryPolicy struct {
	MaxRetries    int
	InitialDelay  time.Duration
	BackoffFactor float64
}

// ExecutionResult represents the result of a workflow execution
type ExecutionResult struct {
	JobID       string
	Status      ExecutionStatus
	Result      interface{}
	StartedAt   time.Time
	CompletedAt *time.Time
	Logs        string
	Error       error
}

// ExecutionStatus represents the status of workflow execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCanceled  ExecutionStatus = "canceled"
)

// NewWorkflowManager creates a new workflow manager
func NewWorkflowManager(client *Client) *WorkflowManager {
	return &WorkflowManager{
		client:      client,
		workflows:   make(map[string]*WorkflowDefinition),
		defaultTags: make(map[string]string),
	}
}

// RegisterWorkflow registers a workflow definition
func (wm *WorkflowManager) RegisterWorkflow(ctx context.Context, definition *WorkflowDefinition) result.Result[bool] {
	// Convert FlexCore workflow to Windmill workflow
	windmillWorkflow := wm.convertToWindmillWorkflow(definition)

	// Create workflow in Windmill
	createReq := CreateWorkflowRequest{
		Path:        definition.Path,
		Summary:     definition.Name,
		Description: definition.Description,
		Value:       windmillWorkflow,
		Schema:      wm.generateWorkflowSchema(definition),
	}

	createResult := wm.client.CreateWorkflow(ctx, createReq)
	if createResult.IsFailure() {
		return result.Failure[bool](createResult.Error())
	}

	// Store definition locally
	wm.workflows[definition.Path] = definition

	return result.Success(true)
}

// ExecuteWorkflow executes a workflow with the given input
func (wm *WorkflowManager) ExecuteWorkflow(ctx context.Context, path string, input map[string]interface{}) result.Result[*ExecutionResult] {
	definition, exists := wm.workflows[path]
	if !exists {
		return result.Failure[*ExecutionResult](errors.NotFoundError("workflow not found: " + path))
	}

	// Prepare run request
	runReq := RunWorkflowRequest{
		Args: input,
		Tag:  &definition.Name,
	}

	// Add default tags
	if len(wm.defaultTags) > 0 {
		tag := ""
		for k, v := range wm.defaultTags {
			tag += fmt.Sprintf("%s:%s,", k, v)
		}
		if tag != "" {
			tag = tag[:len(tag)-1] // Remove trailing comma
			runReq.Tag = &tag
		}
	}

	// Execute workflow
	jobResult := wm.client.RunWorkflow(ctx, path, runReq)
	if jobResult.IsFailure() {
		return result.Failure[*ExecutionResult](jobResult.Error())
	}

	job := jobResult.Value()

	// Convert to execution result
	execResult := &ExecutionResult{
		JobID:     job.ID,
		Status:    wm.convertJobStatusToExecutionStatus(job),
		StartedAt: job.CreatedAt,
		Result:    job.Result,
		Logs:      job.Logs,
	}

	if job.StartedAt != nil {
		execResult.StartedAt = *job.StartedAt
	}

	if job.CompletedAt != nil {
		execResult.CompletedAt = job.CompletedAt
		execResult.Status = ExecutionStatusCompleted
	}

	if job.Running {
		execResult.Status = ExecutionStatusRunning
	}

	return result.Success(execResult)
}

// ExecuteWorkflowAsync executes a workflow asynchronously
func (wm *WorkflowManager) ExecuteWorkflowAsync(ctx context.Context, path string, input map[string]interface{}) result.Result[string] {
	execResult := wm.ExecuteWorkflow(ctx, path, input)
	if execResult.IsFailure() {
		return result.Failure[string](execResult.Error())
	}

	return result.Success(execResult.Value().JobID)
}

// ExecuteWorkflowSync executes a workflow synchronously and waits for completion
func (wm *WorkflowManager) ExecuteWorkflowSync(ctx context.Context, path string, input map[string]interface{}, timeout time.Duration) result.Result[*ExecutionResult] {
	// Start execution
	execResult := wm.ExecuteWorkflow(ctx, path, input)
	if execResult.IsFailure() {
		return execResult
	}

	jobID := execResult.Value().JobID

	// Wait for completion
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	finalJobResult := wm.client.WaitForJob(waitCtx, jobID, 2*time.Second)
	if finalJobResult.IsFailure() {
		return result.Failure[*ExecutionResult](finalJobResult.Error())
	}

	finalJob := finalJobResult.Value()

	// Convert final job to execution result
	finalExecResult := &ExecutionResult{
		JobID:     finalJob.ID,
		Status:    wm.convertJobStatusToExecutionStatus(finalJob),
		Result:    finalJob.Result,
		StartedAt: finalJob.CreatedAt,
		Logs:      finalJob.Logs,
	}

	if finalJob.StartedAt != nil {
		finalExecResult.StartedAt = *finalJob.StartedAt
	}

	if finalJob.CompletedAt != nil {
		finalExecResult.CompletedAt = finalJob.CompletedAt
	}

	return result.Success(finalExecResult)
}

// GetExecutionStatus gets the status of a workflow execution
func (wm *WorkflowManager) GetExecutionStatus(ctx context.Context, jobID string) result.Result[*ExecutionResult] {
	jobResult := wm.client.GetJob(ctx, jobID)
	if jobResult.IsFailure() {
		return result.Failure[*ExecutionResult](jobResult.Error())
	}

	job := jobResult.Value()

	execResult := &ExecutionResult{
		JobID:     job.ID,
		Status:    wm.convertJobStatusToExecutionStatus(job),
		Result:    job.Result,
		StartedAt: job.CreatedAt,
		Logs:      job.Logs,
	}

	if job.StartedAt != nil {
		execResult.StartedAt = *job.StartedAt
	}

	if job.CompletedAt != nil {
		execResult.CompletedAt = job.CompletedAt
	}

	return result.Success(execResult)
}

// CancelExecution cancels a workflow execution
func (wm *WorkflowManager) CancelExecution(ctx context.Context, jobID string, reason string) result.Result[bool] {
	return wm.client.CancelJob(ctx, jobID, reason)
}

// GetExecutionLogs gets the logs for a workflow execution
func (wm *WorkflowManager) GetExecutionLogs(ctx context.Context, jobID string) result.Result[string] {
	return wm.client.GetJobLogs(ctx, jobID)
}

// ListWorkflows lists all registered workflows
func (wm *WorkflowManager) ListWorkflows() []*WorkflowDefinition {
	workflows := make([]*WorkflowDefinition, 0, len(wm.workflows))
	for _, workflow := range wm.workflows {
		workflows = append(workflows, workflow)
	}
	return workflows
}

// GetWorkflow gets a workflow definition by path
func (wm *WorkflowManager) GetWorkflow(path string) result.Result[*WorkflowDefinition] {
	workflow, exists := wm.workflows[path]
	if !exists {
		return result.Failure[*WorkflowDefinition](errors.NotFoundError("workflow not found: " + path))
	}
	return result.Success(workflow)
}

// DeleteWorkflow deletes a workflow
func (wm *WorkflowManager) DeleteWorkflow(ctx context.Context, path string) result.Result[bool] {
	// Delete from Windmill
	deleteResult := wm.client.DeleteWorkflow(ctx, path)
	if deleteResult.IsFailure() {
		return deleteResult
	}

	// Remove from local storage
	delete(wm.workflows, path)

	return result.Success(true)
}

// SetDefaultTags sets default tags for all workflow executions
func (wm *WorkflowManager) SetDefaultTags(tags map[string]string) {
	wm.defaultTags = tags
}

// convertToWindmillWorkflow converts FlexCore workflow to Windmill format
func (wm *WorkflowManager) convertToWindmillWorkflow(definition *WorkflowDefinition) WorkflowValue {
	modules := make([]WorkflowModule, 0, len(definition.Steps))

	for _, step := range definition.Steps {
		module := WorkflowModule{
			ID:      step.ID,
			Summary: step.Name,
			Value: ModuleValue{
				Type:     string(step.Type),
				Content:  step.Script,
				Language: step.Language,
				Input:    step.Input,
			},
		}

		if step.Timeout != nil {
			module.Timeout = step.Timeout
		}

		modules = append(modules, module)
	}

	return WorkflowValue{
		Modules:     modules,
		Same_worker: false, // Allow distributed execution
	}
}

// generateWorkflowSchema generates JSON schema for workflow input
func (wm *WorkflowManager) generateWorkflowSchema(definition *WorkflowDefinition) map[string]interface{} {
	schema := map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "object",
				"description": "Workflow input parameters",
			},
		},
	}

	return schema
}

// convertJobStatusToExecutionStatus converts Windmill job status to execution status
func (wm *WorkflowManager) convertJobStatusToExecutionStatus(job *Job) ExecutionStatus {
	if job.Running {
		return ExecutionStatusRunning
	}

	if job.CompletedAt != nil {
		if job.Result != nil {
			return ExecutionStatusCompleted
		}
		return ExecutionStatusFailed
	}

	if job.StartedAt != nil {
		return ExecutionStatusRunning
	}

	return ExecutionStatusPending
}

// PipelineWorkflowBuilder helps build pipeline workflows for Windmill
type PipelineWorkflowBuilder struct {
	definition *WorkflowDefinition
}

// NewPipelineWorkflowBuilder creates a new pipeline workflow builder
func NewPipelineWorkflowBuilder(name, description string) *PipelineWorkflowBuilder {
	return &PipelineWorkflowBuilder{
		definition: &WorkflowDefinition{
			Path:        fmt.Sprintf("pipeline/%s", name),
			Name:        name,
			Description: description,
			Steps:       make([]WorkflowStep, 0),
			Timeout:     time.Hour,
			RetryPolicy: &RetryPolicy{
				MaxRetries:    3,
				InitialDelay:  time.Second * 5,
				BackoffFactor: 2.0,
			},
			Tags: make(map[string]string),
		},
	}
}

// AddExtractorStep adds an extractor step to the pipeline
func (b *PipelineWorkflowBuilder) AddExtractorStep(name, source string, config map[string]interface{}) *PipelineWorkflowBuilder {
	script := fmt.Sprintf(`
import subprocess
import json

def main(source: str, config: dict):
    """Extract data from source using configured extractor"""
    
    # Execute plugin-based extraction
    cmd = ["./plugins/extractors/%s", source]
    
    # Add config as environment variables
    env = {}
    for key, value in config.items():
        env[f"EXTRACTOR_{key.upper()}"] = str(value)
    
    result = subprocess.run(cmd, capture_output=True, text=True, env=env)
    
    if result.returncode != 0:
        raise Exception(f"Extraction failed: {result.stderr}")
    
    return json.loads(result.stdout)
`, config["plugin_name"])

	step := WorkflowStep{
		ID:       fmt.Sprintf("extract_%s", name),
		Name:     fmt.Sprintf("Extract: %s", name),
		Type:     StepTypeScript,
		Script:   script,
		Language: "python3",
		Input: map[string]interface{}{
			"source": source,
			"config": config,
		},
	}

	b.definition.Steps = append(b.definition.Steps, step)
	return b
}

// AddTransformerStep adds a transformer step to the pipeline
func (b *PipelineWorkflowBuilder) AddTransformerStep(name string, transformations []string, config map[string]interface{}) *PipelineWorkflowBuilder {
	script := fmt.Sprintf(`
def main(data: list, transformations: list, config: dict):
    """Transform data using configured transformations"""
    
    for transformation in transformations:
        if transformation == "clean":
            # Clean data - remove nulls, trim strings
            data = [
                {k: str(v).strip() if isinstance(v, str) else v 
                 for k, v in row.items() if v is not None}
                for row in data
            ]
        elif transformation == "normalize":
            # Normalize data - convert to standard format
            for row in data:
                for key, value in row.items():
                    if isinstance(value, str):
                        row[key] = value.lower().strip()
        elif transformation == "validate":
            # Validate data - check required fields
            required_fields = config.get("required_fields", [])
            data = [
                row for row in data 
                if all(field in row and row[field] for field in required_fields)
            ]
    
    return data
`)

	step := WorkflowStep{
		ID:       fmt.Sprintf("transform_%s", name),
		Name:     fmt.Sprintf("Transform: %s", name),
		Type:     StepTypeScript,
		Script:   script,
		Language: "python3",
		Input: map[string]interface{}{
			"data":            "{{ previous_step.result }}",
			"transformations": transformations,
			"config":          config,
		},
		Dependencies: []string{b.getPreviousStepID()},
	}

	b.definition.Steps = append(b.definition.Steps, step)
	return b
}

// AddLoaderStep adds a loader step to the pipeline
func (b *PipelineWorkflowBuilder) AddLoaderStep(name, destination string, config map[string]interface{}) *PipelineWorkflowBuilder {
	script := fmt.Sprintf(`
import subprocess
import json

def main(data: list, destination: str, config: dict):
    """Load data to destination using configured loader"""
    
    # Prepare data for loading
    input_data = {
        "data": data,
        "destination": destination,
        "config": config
    }
    
    # Execute plugin-based loading
    cmd = ["./plugins/loaders/%s"]
    
    # Add config as environment variables
    env = {}
    for key, value in config.items():
        env[f"LOADER_{key.upper()}"] = str(value)
    
    result = subprocess.run(
        cmd, 
        input=json.dumps(input_data), 
        capture_output=True, 
        text=True, 
        env=env
    )
    
    if result.returncode != 0:
        raise Exception(f"Loading failed: {result.stderr}")
    
    return {"status": "loaded", "count": len(data), "destination": destination}
`, config["plugin_name"])

	step := WorkflowStep{
		ID:       fmt.Sprintf("load_%s", name),
		Name:     fmt.Sprintf("Load: %s", name),
		Type:     StepTypeScript,
		Script:   script,
		Language: "python3",
		Input: map[string]interface{}{
			"data":        "{{ previous_step.result }}",
			"destination": destination,
			"config":      config,
		},
		Dependencies: []string{b.getPreviousStepID()},
	}

	b.definition.Steps = append(b.definition.Steps, step)
	return b
}

// AddCustomStep adds a custom step to the pipeline
func (b *PipelineWorkflowBuilder) AddCustomStep(id, name, script, language string, input map[string]interface{}) *PipelineWorkflowBuilder {
	step := WorkflowStep{
		ID:       id,
		Name:     name,
		Type:     StepTypeScript,
		Script:   script,
		Language: language,
		Input:    input,
	}

	if len(b.definition.Steps) > 0 {
		step.Dependencies = []string{b.getPreviousStepID()}
	}

	b.definition.Steps = append(b.definition.Steps, step)
	return b
}

// SetRetryPolicy sets the retry policy for the workflow
func (b *PipelineWorkflowBuilder) SetRetryPolicy(maxRetries int, initialDelay time.Duration, backoffFactor float64) *PipelineWorkflowBuilder {
	b.definition.RetryPolicy = &RetryPolicy{
		MaxRetries:    maxRetries,
		InitialDelay:  initialDelay,
		BackoffFactor: backoffFactor,
	}
	return b
}

// SetTimeout sets the workflow timeout
func (b *PipelineWorkflowBuilder) SetTimeout(timeout time.Duration) *PipelineWorkflowBuilder {
	b.definition.Timeout = timeout
	return b
}

// AddTag adds a tag to the workflow
func (b *PipelineWorkflowBuilder) AddTag(key, value string) *PipelineWorkflowBuilder {
	b.definition.Tags[key] = value
	return b
}

// Build builds the workflow definition
func (b *PipelineWorkflowBuilder) Build() *WorkflowDefinition {
	return b.definition
}

// getPreviousStepID gets the ID of the previous step
func (b *PipelineWorkflowBuilder) getPreviousStepID() string {
	if len(b.definition.Steps) == 0 {
		return ""
	}
	return b.definition.Steps[len(b.definition.Steps)-1].ID
}