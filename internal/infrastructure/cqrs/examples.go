// CQRS Examples - Real Command and Query Handlers
package cqrs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Example Commands
type CreatePipelineCommand struct {
	CommandIDField   string                   `json:"command_id"`
	AggregateIDField string                   `json:"aggregate_id"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	Steps            []map[string]interface{} `json:"steps"`
	MetadataField    map[string]interface{}   `json:"metadata"`
}

func (c *CreatePipelineCommand) CommandID() string {
	return c.CommandIDField
}

func (c *CreatePipelineCommand) CommandType() string {
	return "create_pipeline"
}

func (c *CreatePipelineCommand) AggregateID() string {
	return c.AggregateIDField
}

func (c *CreatePipelineCommand) Payload() interface{} {
	return map[string]interface{}{
		"name":        c.Name,
		"description": c.Description,
		"steps":       c.Steps,
	}
}

func (c *CreatePipelineCommand) Metadata() map[string]interface{} {
	if c.MetadataField == nil {
		c.MetadataField = make(map[string]interface{})
	}
	c.MetadataField["created_at"] = time.Now()
	c.MetadataField["source"] = "cqrs_example"
	return c.MetadataField
}

type UpdatePipelineStatusCommand struct {
	CommandIDField   string                 `json:"command_id"`
	AggregateIDField string                 `json:"aggregate_id"`
	Status           string                 `json:"status"`
	Message          string                 `json:"message"`
	MetadataField    map[string]interface{} `json:"metadata"`
}

func (c *UpdatePipelineStatusCommand) CommandID() string {
	return c.CommandIDField
}

func (c *UpdatePipelineStatusCommand) CommandType() string {
	return "update_pipeline_status"
}

func (c *UpdatePipelineStatusCommand) AggregateID() string {
	return c.AggregateIDField
}

func (c *UpdatePipelineStatusCommand) Payload() interface{} {
	return map[string]interface{}{
		"status":  c.Status,
		"message": c.Message,
	}
}

func (c *UpdatePipelineStatusCommand) Metadata() map[string]interface{} {
	if c.MetadataField == nil {
		c.MetadataField = make(map[string]interface{})
	}
	c.MetadataField["updated_at"] = time.Now()
	return c.MetadataField
}

// Example Queries
type GetPipelineQuery struct {
	QueryIDField    string                 `json:"query_id"`
	PipelineID      string                 `json:"pipeline_id"`
	ParametersField map[string]interface{} `json:"parameters"`
}

func (q *GetPipelineQuery) QueryID() string {
	return q.QueryIDField
}

func (q *GetPipelineQuery) QueryType() string {
	return "get_pipeline"
}

func (q *GetPipelineQuery) Parameters() map[string]interface{} {
	if q.ParametersField == nil {
		q.ParametersField = make(map[string]interface{})
	}
	q.ParametersField["pipeline_id"] = q.PipelineID
	return q.ParametersField
}

type ListPipelinesQuery struct {
	QueryIDField    string                 `json:"query_id"`
	Status          string                 `json:"status,omitempty"`
	Limit           int                    `json:"limit,omitempty"`
	Offset          int                    `json:"offset,omitempty"`
	ParametersField map[string]interface{} `json:"parameters"`
}

func (q *ListPipelinesQuery) QueryID() string {
	return q.QueryIDField
}

func (q *ListPipelinesQuery) QueryType() string {
	return "list_pipelines"
}

func (q *ListPipelinesQuery) Parameters() map[string]interface{} {
	if q.ParametersField == nil {
		q.ParametersField = make(map[string]interface{})
	}
	if q.Status != "" {
		q.ParametersField["status"] = q.Status
	}
	if q.Limit > 0 {
		q.ParametersField["limit"] = q.Limit
	}
	if q.Offset > 0 {
		q.ParametersField["offset"] = q.Offset
	}
	return q.ParametersField
}

// Example Command Handlers
type PipelineCommandHandler struct {
	cqrsBus *CQRSBus
}

func NewPipelineCommandHandler(bus *CQRSBus) *PipelineCommandHandler {
	return &PipelineCommandHandler{cqrsBus: bus}
}

func (h *PipelineCommandHandler) Handle(ctx context.Context, command Command) error {
	switch cmd := command.(type) {
	case *CreatePipelineCommand:
		return h.handleCreatePipeline(ctx, cmd)
	case *UpdatePipelineStatusCommand:
		return h.handleUpdatePipelineStatus(ctx, cmd)
	default:
		return fmt.Errorf("unknown command type: %s", command.CommandType())
	}
}

func (h *PipelineCommandHandler) CanHandle(commandType string) bool {
	return commandType == "create_pipeline" || commandType == "update_pipeline_status"
}

func (h *PipelineCommandHandler) handleCreatePipeline(ctx context.Context, cmd *CreatePipelineCommand) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Simulate pipeline creation
	pipelineData := map[string]interface{}{
		"id":          cmd.AggregateID(),
		"name":        cmd.Name,
		"description": cmd.Description,
		"steps":       cmd.Steps,
		"status":      "created",
		"created_at":  time.Now(),
	}

	// Update read-side projection
	err := h.cqrsBus.UpdateProjection("pipeline", cmd.AggregateID(), pipelineData, 1)
	if err != nil {
		return fmt.Errorf("failed to update pipeline projection: %w", err)
	}

	return nil
}

// PipelineUpdateContext encapsulates data for pipeline status updates - reduces returns
type PipelineUpdateContext struct {
	command        *UpdatePipelineStatusCommand
	projection     map[string]interface{}
	pipelineData   map[string]interface{}
	currentVersion int
}

// validateProjectionData validates projection data format - DRY error handling
func (h *PipelineCommandHandler) validateProjectionData(ctx *PipelineUpdateContext) error {
	if ctx.projection == nil {
		return fmt.Errorf("pipeline not found: %s", ctx.command.AggregateID())
	}

	// Validate pipeline data format
	pipelineData, ok := ctx.projection["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid pipeline data format")
	}
	ctx.pipelineData = pipelineData

	// Validate version format
	currentVersion, ok := ctx.projection["version"].(int)
	if !ok {
		return fmt.Errorf("invalid pipeline version format")
	}
	ctx.currentVersion = currentVersion

	return nil
}

// updatePipelineData updates the pipeline data with new status - SRP
func (h *PipelineCommandHandler) updatePipelineData(ctx *PipelineUpdateContext) {
	ctx.pipelineData["status"] = ctx.command.Status
	ctx.pipelineData["status_message"] = ctx.command.Message
	ctx.pipelineData["updated_at"] = time.Now()
}

// persistUpdatedPipeline saves the updated pipeline data - SRP
func (h *PipelineCommandHandler) persistUpdatedPipeline(ctx *PipelineUpdateContext) error {
	newVersion := ctx.currentVersion + 1
	return h.cqrsBus.UpdateProjection(
		"pipeline",
		ctx.command.AggregateID(),
		ctx.pipelineData,
		newVersion,
	)
}

func (h *PipelineCommandHandler) handleUpdatePipelineStatus(
	ctx context.Context, cmd *UpdatePipelineStatusCommand,
) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create update context for consolidated error handling
	updateCtx := &PipelineUpdateContext{command: cmd}

	// Get existing projection
	var err error
	updateCtx.projection, err = h.cqrsBus.GetProjection(cmd.AggregateID())
	if err != nil {
		return fmt.Errorf("failed to get pipeline projection: %w", err)
	}

	// Validate and extract projection data - consolidated validation
	if err := h.validateProjectionData(updateCtx); err != nil {
		return err
	}

	// Update pipeline data
	h.updatePipelineData(updateCtx)

	// Persist updated pipeline - single return point for persistence errors
	if err := h.persistUpdatedPipeline(updateCtx); err != nil {
		return fmt.Errorf("failed to update pipeline projection: %w", err)
	}

	return nil
}

// Example Query Handlers
type PipelineQueryHandler struct {
	cqrsBus *CQRSBus
}

func NewPipelineQueryHandler(bus *CQRSBus) *PipelineQueryHandler {
	return &PipelineQueryHandler{cqrsBus: bus}
}

func (h *PipelineQueryHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
	switch q := query.(type) {
	case *GetPipelineQuery:
		return h.handleGetPipeline(ctx, q)
	case *ListPipelinesQuery:
		return h.handleListPipelines(ctx, q)
	default:
		return nil, fmt.Errorf("unknown query type: %s", query.QueryType())
	}
}

func (h *PipelineQueryHandler) CanHandle(queryType string) bool {
	return queryType == "get_pipeline" || queryType == "list_pipelines"
}

func (h *PipelineQueryHandler) handleGetPipeline(ctx context.Context, query *GetPipelineQuery) (interface{}, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	projection, err := h.cqrsBus.GetProjection(query.PipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	if projection == nil {
		return nil, fmt.Errorf("pipeline not found: %s", query.PipelineID)
	}

	return projection, nil
}

func (h *PipelineQueryHandler) handleListPipelines(
	ctx context.Context, query *ListPipelinesQuery,
) (interface{}, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// In a real implementation, this would query the read database
	// For now, simulate a list of pipelines
	pipelines := []map[string]interface{}{
		{
			"id":          "pipeline-1",
			"name":        "Sample Pipeline 1",
			"description": "A sample pipeline for demonstration",
			"status":      "running",
			"created_at":  time.Now().Add(-2 * time.Hour),
		},
		{
			"id":          "pipeline-2",
			"name":        "Sample Pipeline 2",
			"description": "Another sample pipeline",
			"status":      "completed",
			"created_at":  time.Now().Add(-1 * time.Hour),
		},
	}

	// Apply status filter if specified
	if query.Status != "" {
		filtered := []map[string]interface{}{}
		for _, pipeline := range pipelines {
			if pipeline["status"] == query.Status {
				filtered = append(filtered, pipeline)
			}
		}
		pipelines = filtered
	}

	// Apply pagination
	if query.Limit > 0 && query.Offset >= 0 {
		start := query.Offset
		end := start + query.Limit
		if start >= len(pipelines) {
			pipelines = []map[string]interface{}{}
		} else {
			if end > len(pipelines) {
				end = len(pipelines)
			}
			pipelines = pipelines[start:end]
		}
	}

	return map[string]interface{}{
		"items": pipelines,
		"total": len(pipelines),
	}, nil
}

// Helper functions for creating example commands and queries
func NewCreatePipelineCommand(name, description string, steps []map[string]interface{}) *CreatePipelineCommand {
	return &CreatePipelineCommand{
		CommandIDField:   uuid.New().String(),
		AggregateIDField: uuid.New().String(),
		Name:             name,
		Description:      description,
		Steps:            steps,
		MetadataField:    make(map[string]interface{}),
	}
}

func NewUpdatePipelineStatusCommand(pipelineID, status, message string) *UpdatePipelineStatusCommand {
	return &UpdatePipelineStatusCommand{
		CommandIDField:   uuid.New().String(),
		AggregateIDField: pipelineID,
		Status:           status,
		Message:          message,
		MetadataField:    make(map[string]interface{}),
	}
}

func NewGetPipelineQuery(pipelineID string) *GetPipelineQuery {
	return &GetPipelineQuery{
		QueryIDField:    uuid.New().String(),
		PipelineID:      pipelineID,
		ParametersField: make(map[string]interface{}),
	}
}

func NewListPipelinesQuery(status string, limit, offset int) *ListPipelinesQuery {
	return &ListPipelinesQuery{
		QueryIDField:    uuid.New().String(),
		Status:          status,
		Limit:           limit,
		Offset:          offset,
		ParametersField: make(map[string]interface{}),
	}
}
