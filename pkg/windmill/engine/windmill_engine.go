// Package engine provides Windmill workflow engine integration for FlexCore
package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/flext-sh/flexcore/pkg/runtimes"
	"go.uber.org/zap"
)

// WindmillEngine integrates with Windmill workflow orchestration for FlexCore
type WindmillEngine struct {
	mu         sync.RWMutex
	baseURL    string
	authToken  string
	httpClient *http.Client
	logger     *zap.Logger
	runtimeMgr runtimes.RuntimeManager
	workflows  map[string]*WorkflowDefinition
	activeJobs map[string]*WorkflowExecution
	metrics    *EngineMetrics
}

// WorkflowDefinition represents a Windmill workflow definition
type WorkflowDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Path        string                 `json:"path"`
	RuntimeType string                 `json:"runtime_type"`
	Schema      map[string]interface{} `json:"schema"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// WorkflowExecution represents an active workflow execution
type WorkflowExecution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	JobID       string                 `json:"job_id"`
	Status      string                 `json:"status"`
	RuntimeType string                 `json:"runtime_type"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Logs        []string               `json:"logs,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EngineMetrics tracks Windmill engine performance metrics
type EngineMetrics struct {
	mu               sync.RWMutex
	TotalJobs        int64     `json:"total_jobs"`
	ActiveJobs       int64     `json:"active_jobs"`
	CompletedJobs    int64     `json:"completed_jobs"`
	FailedJobs       int64     `json:"failed_jobs"`
	AverageExecution float64   `json:"average_execution_ms"`
	LastJobExecution time.Time `json:"last_job_execution"`
	EngineStartTime  time.Time `json:"engine_start_time"`
}

// WorkflowRequest represents a workflow execution request
type WorkflowRequest struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	RuntimeType string                 `json:"runtime_type"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// WorkflowResponse represents the result of workflow execution
type WorkflowResponse struct {
	JobID     string      `json:"job_id"`
	Status    string      `json:"status"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	StartTime time.Time   `json:"start_time"`
	EndTime   *time.Time  `json:"end_time,omitempty"`
	Logs      []string    `json:"logs,omitempty"`
}

// WorkflowStatus represents the current status of a workflow
type WorkflowStatus struct {
	JobID       string                 `json:"job_id"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	Progress    float64                `json:"progress"`
	RuntimeType string                 `json:"runtime_type"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Logs        []string               `json:"logs,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewWindmillEngine creates a new Windmill engine instance with runtime manager integration
func NewWindmillEngine(baseURL, authToken string, runtimeMgr runtimes.RuntimeManager) *WindmillEngine {
	engine := &WindmillEngine{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: true,
			},
		},
		logger: func() *zap.Logger {
			logger, err := zap.NewProduction()
			if err != nil {
				return zap.NewNop()
			}
			return logger
		}(),
		runtimeMgr: runtimeMgr,
		workflows:  make(map[string]*WorkflowDefinition),
		activeJobs: make(map[string]*WorkflowExecution),
		metrics: &EngineMetrics{
			EngineStartTime: time.Now(),
		},
	}

	// Initialize built-in workflows
	engine.initializeBuiltinWorkflows()

	return engine
}

// ExecuteWorkflow executes a workflow in Windmill
func (w *WindmillEngine) ExecuteWorkflow(ctx context.Context, request WorkflowRequest) (*WorkflowResponse, error) {
	w.logger.Info("Executing workflow in Windmill",
		zap.String("workflow_id", request.WorkflowID),
		zap.String("runtime_type", request.RuntimeType),
		zap.String("command", request.Command))

	// Create Windmill job based on runtime type
	jobPayload := w.createJobPayload(request)

	// Submit job to Windmill
	jobID, err := w.submitJob(ctx, jobPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to submit job to Windmill: %w", err)
	}

	w.logger.Info("Workflow job submitted to Windmill",
		zap.String("job_id", jobID),
		zap.String("workflow_id", request.WorkflowID))

	return &WorkflowResponse{
		JobID:     jobID,
		Status:    "submitted",
		StartTime: time.Now(),
	}, nil
}

// GetWorkflowStatus gets the current status of a workflow
func (w *WindmillEngine) GetWorkflowStatus(ctx context.Context, jobID string) (*WorkflowStatus, error) {
	w.logger.Debug("Getting workflow status from Windmill", zap.String("job_id", jobID))

	// Query Windmill API for job status
	status, err := w.queryJobStatus(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to query job status: %w", err)
	}

	return status, nil
}

// ListWorkflows lists all workflows managed by this engine
func (w *WindmillEngine) ListWorkflows(ctx context.Context) ([]WorkflowStatus, error) {
	w.logger.Debug("Listing workflows from Windmill")

	// Query Windmill API for all jobs
	workflows, err := w.queryAllJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	return workflows, nil
}

// CancelWorkflow cancels a running workflow
func (w *WindmillEngine) CancelWorkflow(ctx context.Context, jobID string) error {
	w.logger.Info("Cancelling workflow in Windmill", zap.String("job_id", jobID))

	// Send cancellation request to Windmill
	if err := w.cancelJob(ctx, jobID); err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	w.logger.Info("Workflow cancelled successfully", zap.String("job_id", jobID))
	return nil
}

// Private helper methods

func (w *WindmillEngine) createJobPayload(request WorkflowRequest) map[string]interface{} {
	// Create Windmill-compatible job payload based on runtime type
	payload := map[string]interface{}{
		"path":       w.getWorkflowPath(request.RuntimeType),
		"args":       w.createWorkflowArgs(request),
		"tag":        fmt.Sprintf("flexcore-%s", request.RuntimeType),
		"parent_job": nil,
		"job_id":     request.ID,
	}

	return payload
}

func (w *WindmillEngine) getWorkflowPath(runtimeType string) string {
	// Map runtime types to Windmill workflow paths
	switch runtimeType {
	case "meltano":
		return "f/flexcore/meltano_execution"
	case "ray":
		return "f/flexcore/ray_execution"
	case "kubernetes":
		return "f/flexcore/k8s_execution"
	default:
		return "f/flexcore/generic_execution"
	}
}

func (w *WindmillEngine) createWorkflowArgs(request WorkflowRequest) map[string]interface{} {
	return map[string]interface{}{
		"runtime_type": request.RuntimeType,
		"command":      request.Command,
		"args":         request.Args,
		"config":       request.Config,
		"job_id":       request.ID,
		"timestamp":    time.Now().Unix(),
	}
}

func (w *WindmillEngine) submitJob(ctx context.Context, payload map[string]interface{}) (string, error) {
	// Prepare Windmill API request
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", w.baseURL+"/api/w/flexcore/jobs/run", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.authToken)

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Windmill API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		JobID string `json:"id"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	w.logger.Info("Job submitted to Windmill successfully",
		zap.String("job_id", response.JobID),
		zap.Int("payload_size", len(jsonPayload)))

	return response.JobID, nil
}

func (w *WindmillEngine) queryJobStatus(ctx context.Context, jobID string) (*WorkflowStatus, error) {
	// Check if job is in active jobs cache
	w.mu.RLock()
	execution, exists := w.activeJobs[jobID]
	w.mu.RUnlock()

	if exists {
		return w.convertExecutionToStatus(execution), nil
	}

	// Query Windmill API for job status
	req, err := http.NewRequestWithContext(ctx, "GET", w.baseURL+"/api/w/flexcore/jobs/"+jobID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+w.authToken)

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Windmill API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var windmillJob struct {
		ID          string      `json:"id"`
		Type        string      `json:"type"`
		Running     bool        `json:"running_state"`
		Success     bool        `json:"success"`
		Result      interface{} `json:"result"`
		Logs        string      `json:"logs"`
		StartedAt   time.Time   `json:"started_at"`
		CompletedAt *time.Time  `json:"completed_at"`
	}
	if err := json.Unmarshal(body, &windmillJob); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to WorkflowStatus
	status := &WorkflowStatus{
		JobID:     windmillJob.ID,
		Status:    w.convertWindmillStatus(windmillJob.Running, windmillJob.Success),
		Progress:  w.calculateProgress(windmillJob.Running, windmillJob.Success),
		Result:    windmillJob.Result,
		StartTime: windmillJob.StartedAt,
		EndTime:   windmillJob.CompletedAt,
		Logs:      w.parseLogs(windmillJob.Logs),
		Metadata: map[string]interface{}{
			"flexcore_instance": "flexcore-runtime",
			"workflow_engine":   "windmill",
			"job_type":          windmillJob.Type,
		},
	}

	// Calculate duration
	if status.EndTime != nil {
		status.Duration = status.EndTime.Sub(status.StartTime)
	} else {
		status.Duration = time.Since(status.StartTime)
	}

	return status, nil
}

func (w *WindmillEngine) queryAllJobs(ctx context.Context) ([]WorkflowStatus, error) {
	// TODO: Implement actual Windmill API call
	// For now, return mock list
	return []WorkflowStatus{
		{
			JobID:       "job_001",
			Status:      "completed",
			Progress:    1.0,
			RuntimeType: "meltano",
			StartTime:   time.Now().Add(-10 * time.Minute),
			EndTime:     &[]time.Time{time.Now().Add(-2 * time.Minute)}[0],
			Duration:    8 * time.Minute,
		},
		{
			JobID:       "job_002",
			Status:      "running",
			Progress:    0.7,
			RuntimeType: "meltano",
			StartTime:   time.Now().Add(-5 * time.Minute),
			Duration:    5 * time.Minute,
		},
	}, nil
}

func (w *WindmillEngine) cancelJob(ctx context.Context, jobID string) error {
	// Create HTTP request for cancellation
	req, err := http.NewRequestWithContext(ctx, "POST", w.baseURL+"/api/w/flexcore/jobs/"+jobID+"/cancel", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+w.authToken)

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("job %s not found or already completed", jobID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Windmill API error (status %d): failed to read response body", resp.StatusCode)
		}
		return fmt.Errorf("Windmill API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Remove from active jobs
	w.mu.Lock()
	delete(w.activeJobs, jobID)
	w.mu.Unlock()

	w.logger.Info("Job cancelled in Windmill", zap.String("job_id", jobID))
	return nil
}

// HealthCheck checks if Windmill is accessible and healthy
func (w *WindmillEngine) HealthCheck(ctx context.Context) error {
	w.logger.Debug("Performing Windmill health check")

	// TODO: Implement actual Windmill health check
	// For now, just check if we have connection parameters
	if w.baseURL == "" {
		return fmt.Errorf("Windmill base URL not configured")
	}

	return nil
}

// GetEngineInfo returns information about the Windmill engine
func (w *WindmillEngine) GetEngineInfo() map[string]interface{} {
	w.metrics.mu.RLock()
	metrics := EngineMetrics{
		TotalJobs:        w.metrics.TotalJobs,
		ActiveJobs:       w.metrics.ActiveJobs,
		CompletedJobs:    w.metrics.CompletedJobs,
		FailedJobs:       w.metrics.FailedJobs,
		AverageExecution: w.metrics.AverageExecution,
		LastJobExecution: w.metrics.LastJobExecution,
		EngineStartTime:  w.metrics.EngineStartTime,
	}
	w.metrics.mu.RUnlock()

	// Create a metrics copy without the mutex for safe serialization
	metricsMap := map[string]interface{}{
		"total_jobs":         metrics.TotalJobs,
		"active_jobs":        metrics.ActiveJobs,
		"completed_jobs":     metrics.CompletedJobs,
		"failed_jobs":        metrics.FailedJobs,
		"average_execution":  metrics.AverageExecution,
		"last_job_execution": metrics.LastJobExecution,
		"engine_start_time":  metrics.EngineStartTime,
	}

	return map[string]interface{}{
		"engine":   "windmill",
		"base_url": w.baseURL,
		"version":  "2.0.0",
		"status":   "operational",
		"uptime":   time.Since(metrics.EngineStartTime).String(),
		"metrics":  metricsMap,
		"features": []string{
			"workflow_orchestration",
			"job_scheduling",
			"runtime_coordination",
			"distributed_execution",
			"multi_runtime_support",
			"real_time_monitoring",
		},
		"supported_runtimes": w.runtimeMgr.GetRuntimeTypes(),
	}
}

// Helper methods for Windmill integration

// initializeBuiltinWorkflows sets up built-in workflow definitions
func (w *WindmillEngine) initializeBuiltinWorkflows() {
	workflows := []*WorkflowDefinition{
		{
			ID:          "meltano_execution",
			Name:        "Meltano Data Pipeline Execution",
			Path:        "f/flexcore/meltano_execution",
			RuntimeType: "meltano",
			Schema: map[string]interface{}{
				"command": "string",
				"args":    "array",
				"config":  "object",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "ray_execution",
			Name:        "Ray Distributed Computing Execution",
			Path:        "f/flexcore/ray_execution",
			RuntimeType: "ray",
			Schema: map[string]interface{}{
				"script":    "string",
				"resources": "object",
				"cluster":   "string",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "k8s_execution",
			Name:        "Kubernetes Job Execution",
			Path:        "f/flexcore/k8s_execution",
			RuntimeType: "kubernetes",
			Schema: map[string]interface{}{
				"jobSpec":   "object",
				"namespace": "string",
				"resources": "object",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	w.mu.Lock()
	for _, workflow := range workflows {
		w.workflows[workflow.ID] = workflow
	}
	w.mu.Unlock()

	w.logger.Info("Built-in workflows initialized",
		zap.Int("workflow_count", len(workflows)))
}

// convertExecutionToStatus converts WorkflowExecution to WorkflowStatus
func (w *WindmillEngine) convertExecutionToStatus(execution *WorkflowExecution) *WorkflowStatus {
	status := &WorkflowStatus{
		JobID:       execution.JobID,
		Status:      execution.Status,
		RuntimeType: execution.RuntimeType,
		Result:      execution.Result,
		Error:       execution.Error,
		StartTime:   execution.StartTime,
		EndTime:     execution.EndTime,
		Logs:        execution.Logs,
		Metadata:    execution.Metadata,
	}

	// Calculate progress and duration
	status.Progress = w.calculateProgressFromStatus(execution.Status)
	if status.EndTime != nil {
		status.Duration = status.EndTime.Sub(status.StartTime)
	} else {
		status.Duration = time.Since(status.StartTime)
	}

	return status
}

// convertWindmillStatus converts Windmill job state to standard status
func (w *WindmillEngine) convertWindmillStatus(running, success bool) string {
	if running {
		return "running"
	}
	if success {
		return "completed"
	}
	return "failed"
}

// calculateProgress calculates job progress based on Windmill state
func (w *WindmillEngine) calculateProgress(running, success bool) float64 {
	if success {
		return 1.0
	}
	if running {
		return 0.5 // Assume 50% for running jobs
	}
	return 0.0
}

// calculateProgressFromStatus calculates progress from status string
func (w *WindmillEngine) calculateProgressFromStatus(status string) float64 {
	switch status {
	case "completed":
		return 1.0
	case "running":
		return 0.5
	case "failed", "cancelled":
		return 0.0
	default:
		return 0.0
	}
}

// parseLogs parses Windmill log string into array
func (w *WindmillEngine) parseLogs(logString string) []string {
	if logString == "" {
		return []string{}
	}
	return strings.Split(logString, "\n")
}

// ExecuteWorkflowWithRuntime executes workflow with specific runtime coordination
func (w *WindmillEngine) ExecuteWorkflowWithRuntime(ctx context.Context, request WorkflowRequest) (*WorkflowResponse, error) {
	// Check if runtime is available
	if !w.runtimeMgr.IsRuntimeRegistered(request.RuntimeType) {
		return nil, fmt.Errorf("runtime %s is not registered", request.RuntimeType)
	}

	// Check runtime health
	runtime, err := w.runtimeMgr.GetRuntime(request.RuntimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime: %w", err)
	}

	if err := runtime.HealthCheck(ctx); err != nil {
		return nil, fmt.Errorf("runtime %s health check failed: %w", request.RuntimeType, err)
	}

	// Create workflow execution record
	execution := &WorkflowExecution{
		ID:          request.ID,
		WorkflowID:  request.WorkflowID,
		Status:      "pending",
		RuntimeType: request.RuntimeType,
		StartTime:   time.Now(),
		Logs:        []string{"Workflow execution started"},
		Metadata: map[string]interface{}{
			"flexcore_instance": "flexcore-runtime",
			"workflow_engine":   "windmill",
			"runtime_type":      request.RuntimeType,
		},
	}

	// Execute workflow
	response, err := w.ExecuteWorkflow(ctx, request)
	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
		endTime := time.Now()
		execution.EndTime = &endTime

		// Update metrics
		w.updateMetrics("failed")

		return nil, err
	}

	// Update execution record
	execution.JobID = response.JobID
	execution.Status = "submitted"

	// Store in active jobs
	w.mu.Lock()
	w.activeJobs[response.JobID] = execution
	w.mu.Unlock()

	// Update metrics
	w.updateMetrics("submitted")

	w.logger.Info("Workflow executed with runtime coordination",
		zap.String("job_id", response.JobID),
		zap.String("runtime_type", request.RuntimeType))

	return response, nil
}

// updateMetrics updates engine performance metrics
func (w *WindmillEngine) updateMetrics(status string) {
	w.metrics.mu.Lock()
	defer w.metrics.mu.Unlock()

	w.metrics.TotalJobs++
	w.metrics.LastJobExecution = time.Now()

	switch status {
	case "submitted":
		w.metrics.ActiveJobs++
	case "completed":
		w.metrics.CompletedJobs++
		if w.metrics.ActiveJobs > 0 {
			w.metrics.ActiveJobs--
		}
	case "failed":
		w.metrics.FailedJobs++
		if w.metrics.ActiveJobs > 0 {
			w.metrics.ActiveJobs--
		}
	}

	// Update average execution time (simplified calculation)
	if w.metrics.CompletedJobs > 0 {
		w.metrics.AverageExecution = float64(time.Since(w.metrics.EngineStartTime).Milliseconds()) / float64(w.metrics.CompletedJobs)
	}
}

// GetWorkflowDefinitions returns all available workflow definitions
func (w *WindmillEngine) GetWorkflowDefinitions() map[string]*WorkflowDefinition {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Create a copy to avoid race conditions
	definitions := make(map[string]*WorkflowDefinition)
	for id, workflow := range w.workflows {
		definitions[id] = workflow
	}

	return definitions
}

// GetActiveJobs returns all currently active jobs
func (w *WindmillEngine) GetActiveJobs() map[string]*WorkflowExecution {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Create a copy to avoid race conditions
	jobs := make(map[string]*WorkflowExecution)
	for id, job := range w.activeJobs {
		jobs[id] = job
	}

	return jobs
}

// GetEngineMetrics returns current engine metrics
func (w *WindmillEngine) GetEngineMetrics() *EngineMetrics {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()

	// Create a copy to avoid race conditions, excluding the mutex
	return &EngineMetrics{
		TotalJobs:        w.metrics.TotalJobs,
		ActiveJobs:       w.metrics.ActiveJobs,
		CompletedJobs:    w.metrics.CompletedJobs,
		FailedJobs:       w.metrics.FailedJobs,
		AverageExecution: w.metrics.AverageExecution,
		LastJobExecution: w.metrics.LastJobExecution,
		EngineStartTime:  w.metrics.EngineStartTime,
	}
}
