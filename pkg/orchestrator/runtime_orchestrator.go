// Package orchestrator - FlexCore Runtime Orchestrator with Windmill Integration
package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"github.com/flext-sh/flexcore/pkg/runtimes"
	"github.com/flext-sh/flexcore/pkg/runtimes/meltano"
	"github.com/flext-sh/flexcore/pkg/windmill/engine"
)

// Status constants for orchestration
const (
	StatusFailed    = "failed"
	StatusSubmitted = "submitted"
	StatusRunning   = "running"
	StatusCompleted = "completed"
)

// RuntimeOrchestrator coordinates multiple runtimes through Windmill workflows
type RuntimeOrchestrator struct {
	mu             sync.RWMutex
	logger         *zap.Logger
	runtimeManager runtimes.RuntimeManager
	windmillEngine *engine.WindmillEngine
	config         *OrchestratorConfig
	orchestrations map[string]*Orchestration
	metrics        *OrchestratorMetrics
}

// OrchestratorConfig holds orchestrator configuration
type OrchestratorConfig struct {
	WindmillURL          string        `json:"windmill_url"`
	WindmillToken        string        `json:"windmill_token"`
	DefaultTimeout       time.Duration `json:"default_timeout"`
	MaxConcurrentJobs    int           `json:"max_concurrent_jobs"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
	MetricsUpdateInterval time.Duration `json:"metrics_update_interval"`
	EnabledRuntimes      []string      `json:"enabled_runtimes"`
}

// Orchestration represents a coordinated workflow execution across runtimes
type Orchestration struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	RuntimeType     string                 `json:"runtime_type"`
	WorkflowID      string                 `json:"workflow_id"`
	JobID           string                 `json:"job_id"`
	Status          string                 `json:"status"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Duration        time.Duration          `json:"duration"`
	Result          interface{}            `json:"result,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Progress        float64                `json:"progress"`
	Logs            []string               `json:"logs,omitempty"`
}

// OrchestratorMetrics tracks orchestrator performance
type OrchestratorMetrics struct {
	mu                     sync.RWMutex
	TotalOrchestrations    int64                  `json:"total_orchestrations"`
	ActiveOrchestrations   int64                  `json:"active_orchestrations"`
	CompletedOrchestrations int64                 `json:"completed_orchestrations"`
	FailedOrchestrations   int64                  `json:"failed_orchestrations"`
	AverageExecution       float64                `json:"average_execution_ms"`
	RuntimeDistribution    map[string]int64       `json:"runtime_distribution"`
	LastOrchestration      time.Time              `json:"last_orchestration"`
	OrchestratorStartTime  time.Time              `json:"orchestrator_start_time"`
}

// OrchestrationRequest represents a request for workflow orchestration
type OrchestrationRequest struct {
	Name        string                 `json:"name"`
	RuntimeType string                 `json:"runtime_type"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	Priority    string                 `json:"priority,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// OrchestrationResult represents the result of workflow orchestration
type OrchestrationResult struct {
	OrchestrationID string      `json:"orchestration_id"`
	JobID           string      `json:"job_id"`
	Status          string      `json:"status"`
	Result          interface{} `json:"result,omitempty"`
	Error           string      `json:"error,omitempty"`
	StartTime       time.Time   `json:"start_time"`
	Duration        time.Duration `json:"duration"`
}

// NewRuntimeOrchestrator creates a new runtime orchestrator
func NewRuntimeOrchestrator(config *OrchestratorConfig) (*RuntimeOrchestrator, error) {
	if config == nil {
		config = &OrchestratorConfig{
			WindmillURL:           "http://localhost:8000",
			DefaultTimeout:        300 * time.Second,
			MaxConcurrentJobs:     10,
			HealthCheckInterval:   30 * time.Second,
			MetricsUpdateInterval: 60 * time.Second,
			EnabledRuntimes:       []string{"meltano"},
		}
	}

	// Create runtime manager
	runtimeManager := runtimes.NewDefaultRuntimeManager()

	// Create Windmill engine
	windmillEngine := engine.NewWindmillEngine(
		config.WindmillURL,
		config.WindmillToken,
		runtimeManager,
	)

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	orchestrator := &RuntimeOrchestrator{
		logger:         logger,
		runtimeManager: runtimeManager,
		windmillEngine: windmillEngine,
		config:         config,
		orchestrations: make(map[string]*Orchestration),
		metrics: &OrchestratorMetrics{
			RuntimeDistribution:   make(map[string]int64),
			OrchestratorStartTime: time.Now(),
		},
	}

	return orchestrator, nil
}

// Initialize initializes the orchestrator and registers runtimes
func (ro *RuntimeOrchestrator) Initialize(ctx context.Context) error {
	ro.logger.Info("Initializing FlexCore Runtime Orchestrator",
		zap.Any("enabled_runtimes", ro.config.EnabledRuntimes),
		zap.String("windmill_url", ro.config.WindmillURL))

	// Register enabled runtimes
	if err := ro.registerRuntimes(ctx); err != nil {
		return fmt.Errorf("failed to register runtimes: %w", err)
	}

	// Start background health checking
	go ro.startHealthChecking(ctx)

	// Start metrics collection
	go ro.startMetricsCollection(ctx)

	ro.logger.Info("Runtime Orchestrator initialized successfully",
		zap.Any("registered_runtimes", ro.runtimeManager.GetRuntimeTypes()))

	return nil
}

// ExecuteOrchestration executes a workflow orchestration
func (ro *RuntimeOrchestrator) ExecuteOrchestration(ctx context.Context, req *OrchestrationRequest) (*OrchestrationResult, error) {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	// Check concurrency limits
	if ro.metrics.ActiveOrchestrations >= int64(ro.config.MaxConcurrentJobs) {
		return nil, fmt.Errorf("maximum concurrent orchestrations reached (%d/%d)",
			ro.metrics.ActiveOrchestrations, ro.config.MaxConcurrentJobs)
	}

	// Validate runtime
	if !ro.runtimeManager.IsRuntimeRegistered(req.RuntimeType) {
		return nil, fmt.Errorf("runtime %s is not registered", req.RuntimeType)
	}

	// Create orchestration
	orchestration := &Orchestration{
		ID:          fmt.Sprintf("orch_%d", time.Now().UnixNano()),
		Name:        req.Name,
		RuntimeType: req.RuntimeType,
		Status:      "pending",
		StartTime:   time.Now(),
		Progress:    0.0,
		Metadata:    req.Metadata,
		Logs:        []string{"Orchestration created"},
	}

	// Set timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = ro.config.DefaultTimeout
	}

	// Create Windmill workflow request
	workflowReq := engine.WorkflowRequest{
		ID:          orchestration.ID,
		WorkflowID:  ro.getWorkflowIDForRuntime(req.RuntimeType),
		RuntimeType: req.RuntimeType,
		Command:     req.Command,
		Args:        req.Args,
		Config:      req.Config,
	}

	// Execute workflow through Windmill
	response, err := ro.windmillEngine.ExecuteWorkflowWithRuntime(ctx, workflowReq)
	if err != nil {
		orchestration.Status = StatusFailed
		orchestration.Error = err.Error()
		endTime := time.Now()
		orchestration.EndTime = &endTime
		orchestration.Duration = time.Since(orchestration.StartTime)

		ro.updateMetrics("failed", req.RuntimeType)

		return &OrchestrationResult{
			OrchestrationID: orchestration.ID,
			Status:          "failed",
			Error:           err.Error(),
			StartTime:       orchestration.StartTime,
			Duration:        orchestration.Duration,
		}, err
	}

	// Update orchestration with job information
	orchestration.JobID = response.JobID
	orchestration.WorkflowID = workflowReq.WorkflowID
	orchestration.Status = StatusSubmitted
	orchestration.Progress = 0.1

	// Store orchestration
	ro.orchestrations[orchestration.ID] = orchestration

	// Update metrics
	ro.updateMetrics("submitted", req.RuntimeType)

	// Start monitoring in background
	go ro.monitorOrchestration(ctx, orchestration.ID, timeout)

	ro.logger.Info("Orchestration executed successfully",
		zap.String("orchestration_id", orchestration.ID),
		zap.String("job_id", orchestration.JobID),
		zap.String("runtime_type", req.RuntimeType))

	return &OrchestrationResult{
		OrchestrationID: orchestration.ID,
		JobID:           orchestration.JobID,
		Status:          orchestration.Status,
		StartTime:       orchestration.StartTime,
		Duration:        time.Since(orchestration.StartTime),
	}, nil
}

// GetOrchestration returns an orchestration by ID
func (ro *RuntimeOrchestrator) GetOrchestration(orchestrationID string) (*Orchestration, error) {
	ro.mu.RLock()
	defer ro.mu.RUnlock()

	orchestration, exists := ro.orchestrations[orchestrationID]
	if !exists {
		return nil, fmt.Errorf("orchestration %s not found", orchestrationID)
	}

	// Create a copy to avoid race conditions
	orchestrationCopy := *orchestration
	return &orchestrationCopy, nil
}

// ListOrchestrations returns all orchestrations with optional filtering
func (ro *RuntimeOrchestrator) ListOrchestrations(status, runtimeType string, limit int) ([]*Orchestration, error) {
	ro.mu.RLock()
	defer ro.mu.RUnlock()

	orchestrations := make([]*Orchestration, 0)
	count := 0

	for _, orchestration := range ro.orchestrations {
		// Apply filters
		if status != "" && orchestration.Status != status {
			continue
		}
		if runtimeType != "" && orchestration.RuntimeType != runtimeType {
			continue
		}

		// Create copy and add to results
		orchestrationCopy := *orchestration
		orchestrations = append(orchestrations, &orchestrationCopy)
		count++

		// Apply limit
		if limit > 0 && count >= limit {
			break
		}
	}

	return orchestrations, nil
}

// CancelOrchestration cancels a running orchestration
func (ro *RuntimeOrchestrator) CancelOrchestration(ctx context.Context, orchestrationID string) error {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	orchestration, exists := ro.orchestrations[orchestrationID]
	if !exists {
		return fmt.Errorf("orchestration %s not found", orchestrationID)
	}

	if orchestration.Status != StatusRunning && orchestration.Status != StatusSubmitted {
		return fmt.Errorf("orchestration %s cannot be canceled (status: %s)", 
			orchestrationID, orchestration.Status)
	}

	// Cancel job in Windmill
	if orchestration.JobID != "" {
		if err := ro.windmillEngine.CancelWorkflow(ctx, orchestration.JobID); err != nil {
			ro.logger.Warn("Failed to cancel workflow in Windmill",
				zap.String("job_id", orchestration.JobID),
				zap.Error(err))
		}
	}

	// Update orchestration status
	orchestration.Status = "canceled"
	endTime := time.Now()
	orchestration.EndTime = &endTime
	orchestration.Duration = time.Since(orchestration.StartTime)
	orchestration.Logs = append(orchestration.Logs, "Orchestration canceled")

	ro.updateMetrics("canceled", orchestration.RuntimeType)

	ro.logger.Info("Orchestration canceled",
		zap.String("orchestration_id", orchestrationID))

	return nil
}

// GetOrchestratorInfo returns orchestrator information and metrics
func (ro *RuntimeOrchestrator) GetOrchestratorInfo() map[string]interface{} {
	ro.metrics.mu.RLock()
	metrics := OrchestratorMetrics{
		TotalOrchestrations:     ro.metrics.TotalOrchestrations,
		ActiveOrchestrations:    ro.metrics.ActiveOrchestrations,
		CompletedOrchestrations: ro.metrics.CompletedOrchestrations,
		FailedOrchestrations:    ro.metrics.FailedOrchestrations,
		AverageExecution:        ro.metrics.AverageExecution,
		RuntimeDistribution:     make(map[string]int64),
		LastOrchestration:       ro.metrics.LastOrchestration,
		OrchestratorStartTime:   ro.metrics.OrchestratorStartTime,
	}
	// Copy the map to avoid mutex issues
	for k, v := range ro.metrics.RuntimeDistribution {
		metrics.RuntimeDistribution[k] = v
	}
	ro.metrics.mu.RUnlock()

	windmillInfo := ro.windmillEngine.GetEngineInfo()
	runtimeTypes := ro.runtimeManager.GetRuntimeTypes()

	// Create a metrics copy without the mutex for safe serialization
	metricsMap := map[string]interface{}{
		"total_orchestrations":     metrics.TotalOrchestrations,
		"active_orchestrations":    metrics.ActiveOrchestrations,
		"completed_orchestrations": metrics.CompletedOrchestrations,
		"failed_orchestrations":    metrics.FailedOrchestrations,
		"average_execution":        metrics.AverageExecution,
		"runtime_distribution":     metrics.RuntimeDistribution,
		"last_orchestration":       metrics.LastOrchestration,
		"orchestrator_start_time":  metrics.OrchestratorStartTime,
	}

	return map[string]interface{}{
		"orchestrator": "FlexCore Runtime Orchestrator",
		"version":      "2.0.0",
		"status":       "operational",
		"uptime":       time.Since(metrics.OrchestratorStartTime).String(),
		"metrics":      metricsMap,
		"windmill":     windmillInfo,
		"runtimes":     runtimeTypes,
		"config": map[string]interface{}{
			"max_concurrent_jobs":     ro.config.MaxConcurrentJobs,
			"default_timeout":         ro.config.DefaultTimeout.String(),
			"health_check_interval":   ro.config.HealthCheckInterval.String(),
			"metrics_update_interval": ro.config.MetricsUpdateInterval.String(),
		},
	}
}

// Private helper methods

// registerRuntimes registers all enabled runtimes
func (ro *RuntimeOrchestrator) registerRuntimes(ctx context.Context) error {
	for _, runtimeType := range ro.config.EnabledRuntimes {
		switch runtimeType {
		case "meltano":
			if err := ro.registerMeltanoRuntime(); err != nil {
				return fmt.Errorf("failed to register Meltano runtime: %w", err)
			}
		case "ray":
			ro.logger.Info("Ray runtime not yet implemented (stub)")
		case "kubernetes":
			ro.logger.Info("Kubernetes runtime not yet implemented (stub)")
		default:
			ro.logger.Warn("Unknown runtime type", zap.String("runtime_type", runtimeType))
		}
	}

	return nil
}

// registerMeltanoRuntime registers the Meltano runtime
func (ro *RuntimeOrchestrator) registerMeltanoRuntime() error {
	meltanoConfig := &meltano.MeltanoConfig{
		PythonPath:       "python3",
		MeltanoProject:   "./meltano_project",
		VirtualEnv:       "./venv",
		FlextCoreProject: "../flext-core",
		Timeout:          300 * time.Second,
		MaxConcurrency:   4,
		Environment: map[string]string{
			"MELTANO_ENVIRONMENT": "production",
			"MELTANO_SEND_ANONYMOUS_USAGE_STATS": "false",
		},
	}

	meltanoRuntime := meltano.NewMeltanoRuntime(meltanoConfig)
	
	if err := ro.runtimeManager.RegisterRuntime(meltanoRuntime); err != nil {
		return err
	}

	ro.logger.Info("Meltano runtime registered successfully")
	return nil
}

// getWorkflowIDForRuntime returns the appropriate workflow ID for a runtime type
func (ro *RuntimeOrchestrator) getWorkflowIDForRuntime(runtimeType string) string {
	switch runtimeType {
	case "meltano":
		return "meltano_execution"
	case "ray":
		return "ray_execution"
	case "kubernetes":
		return "k8s_execution"
	default:
		return "generic_execution"
	}
}

// monitorOrchestration monitors an orchestration in background
func (ro *RuntimeOrchestrator) monitorOrchestration(ctx context.Context, orchestrationID string, timeout time.Duration) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Now().After(deadline) {
				ro.timeoutOrchestration(orchestrationID)
				return
			}

			if ro.updateOrchestrationStatus(ctx, orchestrationID) {
				return // Orchestration completed
			}
		}
	}
}

// updateOrchestrationStatus updates orchestration status from Windmill
func (ro *RuntimeOrchestrator) updateOrchestrationStatus(ctx context.Context, orchestrationID string) bool {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	orchestration, exists := ro.orchestrations[orchestrationID]
	if !exists {
		return true // Orchestration removed
	}

	if orchestration.JobID == "" {
		return false // No job ID yet
	}

	// Get status from Windmill
	status, err := ro.windmillEngine.GetWorkflowStatus(ctx, orchestration.JobID)
	if err != nil {
		ro.logger.Warn("Failed to get workflow status",
			zap.String("job_id", orchestration.JobID),
			zap.Error(err))
		return false
	}

	// Update orchestration
	orchestration.Status = status.Status
	orchestration.Progress = status.Progress
	orchestration.Result = status.Result
	orchestration.Error = status.Error
	orchestration.Duration = status.Duration
	if len(status.Logs) > 0 {
		orchestration.Logs = status.Logs
	}

	// Check if completed
	if status.Status == StatusCompleted || status.Status == StatusFailed {
		endTime := time.Now()
		orchestration.EndTime = &endTime
		ro.updateMetrics(status.Status, orchestration.RuntimeType)
		return true
	}

	return false
}

// timeoutOrchestration handles orchestration timeout
func (ro *RuntimeOrchestrator) timeoutOrchestration(orchestrationID string) {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	orchestration, exists := ro.orchestrations[orchestrationID]
	if !exists {
		return
	}

	orchestration.Status = "timeout"
	orchestration.Error = "Orchestration timeout exceeded"
	endTime := time.Now()
	orchestration.EndTime = &endTime
	orchestration.Duration = time.Since(orchestration.StartTime)

	ro.updateMetrics("timeout", orchestration.RuntimeType)

	ro.logger.Warn("Orchestration timed out",
		zap.String("orchestration_id", orchestrationID),
		zap.Duration("duration", orchestration.Duration))
}

// updateMetrics updates orchestrator metrics
func (ro *RuntimeOrchestrator) updateMetrics(status string, runtimeType string) {
	ro.metrics.mu.Lock()
	defer ro.metrics.mu.Unlock()

	ro.metrics.LastOrchestration = time.Now()

	switch status {
	case "submitted":
		ro.metrics.TotalOrchestrations++
		ro.metrics.ActiveOrchestrations++
		ro.metrics.RuntimeDistribution[runtimeType]++
	case StatusCompleted:
		ro.metrics.CompletedOrchestrations++
		if ro.metrics.ActiveOrchestrations > 0 {
			ro.metrics.ActiveOrchestrations--
		}
	case "failed", "canceled", "timeout":
		ro.metrics.FailedOrchestrations++
		if ro.metrics.ActiveOrchestrations > 0 {
			ro.metrics.ActiveOrchestrations--
		}
	}

	// Update average execution time
	if ro.metrics.CompletedOrchestrations > 0 {
		ro.metrics.AverageExecution = float64(time.Since(ro.metrics.OrchestratorStartTime).Milliseconds()) / 
			float64(ro.metrics.CompletedOrchestrations)
	}
}

// startHealthChecking starts background health checking
func (ro *RuntimeOrchestrator) startHealthChecking(ctx context.Context) {
	ticker := time.NewTicker(ro.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ro.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks performs health checks on all runtimes
func (ro *RuntimeOrchestrator) performHealthChecks(ctx context.Context) {
	runtimeTypes := ro.runtimeManager.GetRuntimeTypes()
	
	for _, runtimeType := range runtimeTypes {
		runtime, err := ro.runtimeManager.GetRuntime(runtimeType)
		if err != nil {
			continue
		}

		if err := runtime.HealthCheck(ctx); err != nil {
			ro.logger.Warn("Runtime health check failed",
				zap.String("runtime_type", runtimeType),
				zap.Error(err))
		}
	}

	// Check Windmill health
	if err := ro.windmillEngine.HealthCheck(ctx); err != nil {
		ro.logger.Warn("Windmill health check failed",
			zap.Error(err))
	}
}

// startMetricsCollection starts background metrics collection
func (ro *RuntimeOrchestrator) startMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(ro.config.MetricsUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ro.collectMetrics()
		}
	}
}

// collectMetrics collects metrics from all components
func (ro *RuntimeOrchestrator) collectMetrics() {
	// Collect Windmill engine metrics
	engineMetrics := ro.windmillEngine.GetEngineMetrics()
	
	// Collect runtime metrics (can be extended)
	runtimeTypes := ro.runtimeManager.GetRuntimeTypes()
	
	ro.logger.Debug("Metrics collected",
		zap.Int64("total_orchestrations", ro.metrics.TotalOrchestrations),
		zap.Int64("active_orchestrations", ro.metrics.ActiveOrchestrations),
		zap.Int64("windmill_total_jobs", engineMetrics.TotalJobs),
		zap.Int("runtime_count", len(runtimeTypes)))
}