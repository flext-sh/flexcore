// Package meltano - Meltano Runtime Implementation for FlexCore
package meltano

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"github.com/flext-sh/flexcore/pkg/runtimes"
)

// MeltanoRuntime implements the Runtime interface for Meltano integration
type MeltanoRuntime struct {
	mu               sync.RWMutex
	config           *MeltanoConfig
	logger           *zap.Logger
	status           string
	lastSeen         time.Time
	pythonPath       string
	meltanoProject   string
	virtualEnv       string
	activeJobs       map[string]*MeltanoJob
	capabilities     []string
	metrics          *MeltanoMetrics
}

// MeltanoConfig holds Meltano runtime configuration
type MeltanoConfig struct {
	PythonPath       string            `json:"python_path"`
	MeltanoProject   string            `json:"meltano_project"`
	VirtualEnv       string            `json:"virtual_env"`
	FlextCoreProject string            `json:"flext_core_project"`
	Environment      map[string]string `json:"environment"`
	Timeout          time.Duration     `json:"timeout"`
	MaxConcurrency   int               `json:"max_concurrency"`
}

// MeltanoJob represents a running Meltano job
type MeltanoJob struct {
	ID          string                 `json:"id"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	Config      map[string]interface{} `json:"config"`
	Status      string                 `json:"status"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Output      string                 `json:"output"`
	Error       string                 `json:"error,omitempty"`
	PID         int                    `json:"pid,omitempty"`
	ExitCode    int                    `json:"exit_code,omitempty"`
}

// MeltanoMetrics tracks Meltano runtime performance
type MeltanoMetrics struct {
	mu                sync.RWMutex
	TotalJobs         int64     `json:"total_jobs"`
	ActiveJobs        int64     `json:"active_jobs"`
	CompletedJobs     int64     `json:"completed_jobs"`
	FailedJobs        int64     `json:"failed_jobs"`
	AverageExecution  float64   `json:"average_execution_ms"`
	LastJobExecution  time.Time `json:"last_job_execution"`
	RuntimeStartTime  time.Time `json:"runtime_start_time"`
}

// NewMeltanoRuntime creates a new Meltano runtime instance
func NewMeltanoRuntime(config *MeltanoConfig) *MeltanoRuntime {
	if config == nil {
		config = &MeltanoConfig{
			PythonPath:       "python3",
			MeltanoProject:   "./meltano_project",
			VirtualEnv:       "./venv",
			FlextCoreProject: "../flext-core",
			Timeout:          300 * time.Second,
			MaxConcurrency:   4,
			Environment:      make(map[string]string),
		}
	}

	runtime := &MeltanoRuntime{
		config:     config,
		logger:     func() *zap.Logger { logger, _ := zap.NewProduction(); return logger }(),
		status:     "stopped",
		lastSeen:   time.Now(),
		activeJobs: make(map[string]*MeltanoJob),
		capabilities: []string{
			"singer_taps",
			"singer_targets", 
			"dbt_models",
			"meltano_pipelines",
			"data_extraction",
			"data_transformation",
			"data_loading",
		},
		metrics: &MeltanoMetrics{
			RuntimeStartTime: time.Now(),
		},
	}

	runtime.pythonPath = config.PythonPath
	runtime.meltanoProject = config.MeltanoProject
	runtime.virtualEnv = config.VirtualEnv

	return runtime
}

// GetType returns the runtime type
func (mr *MeltanoRuntime) GetType() string {
	return "meltano"
}

// Start starts the Meltano runtime
func (mr *MeltanoRuntime) Start(ctx context.Context, config map[string]string, capabilities []string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.logger.Info("Starting Meltano runtime",
		zap.String("meltano_project", mr.meltanoProject),
		zap.String("python_path", mr.pythonPath))

	// Validate Meltano project directory
	if err := mr.validateMeltanoProject(); err != nil {
		mr.status = "failed"
		return fmt.Errorf("Meltano project validation failed: %w", err)
	}

	// Check Python and Meltano installation
	if err := mr.validatePythonEnvironment(); err != nil {
		mr.status = "failed"
		return fmt.Errorf("Python environment validation failed: %w", err)
	}

	// Initialize flext-core integration
	if err := mr.initializeFlextCoreIntegration(); err != nil {
		mr.logger.Warn("FlextCore integration initialization failed", 
			zap.Error(err))
		// Continue without flext-core integration
	}

	// Apply configuration overrides
	mr.applyConfigurationOverrides(config)

	// Filter capabilities if provided
	if len(capabilities) > 0 {
		mr.filterCapabilities(capabilities)
	}

	mr.status = "running"
	mr.lastSeen = time.Now()

	mr.logger.Info("Meltano runtime started successfully",
		zap.Any("capabilities", mr.capabilities),
		zap.Int("max_concurrency", mr.config.MaxConcurrency))

	return nil
}

// Stop stops the Meltano runtime
func (mr *MeltanoRuntime) Stop(ctx context.Context, force bool, timeout time.Duration) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.logger.Info("Stopping Meltano runtime",
		zap.Bool("force", force),
		zap.Int("active_jobs", len(mr.activeJobs)))

	// Stop all active jobs
	for jobID, job := range mr.activeJobs {
		if job.Status == "running" {
			if force {
				mr.logger.Info("Force stopping job", zap.String("job_id", jobID))
				if job.PID > 0 {
					if proc, err := os.FindProcess(job.PID); err == nil {
						_ = proc.Kill()
					}
				}
			} else {
				mr.logger.Info("Gracefully stopping job", zap.String("job_id", jobID))
				// Allow jobs to finish naturally within timeout
			}
		}
	}

	// Wait for jobs to complete if not forcing
	if !force && len(mr.activeJobs) > 0 {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) && len(mr.activeJobs) > 0 {
			time.Sleep(1 * time.Second)
			mr.cleanupCompletedJobs()
		}
	}

	mr.status = "stopped"
	mr.lastSeen = time.Now()

	mr.logger.Info("Meltano runtime stopped successfully")
	return nil
}

// Execute executes a command on the Meltano runtime
func (mr *MeltanoRuntime) Execute(ctx context.Context, command string, args []string, config map[string]interface{}) (interface{}, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if mr.status != "running" {
		return nil, fmt.Errorf("Meltano runtime is not running (status: %s)", mr.status)
	}

	// Check concurrency limits
	activeCount := mr.countActiveJobs()
	if activeCount >= mr.config.MaxConcurrency {
		return nil, fmt.Errorf("maximum concurrency reached (%d/%d)", activeCount, mr.config.MaxConcurrency)
	}

	// Create job
	job := &MeltanoJob{
		ID:        fmt.Sprintf("meltano_%d", time.Now().UnixNano()),
		Command:   command,
		Args:      args,
		Config:    config,
		Status:    "pending",
		StartTime: time.Now(),
	}

	mr.activeJobs[job.ID] = job

	// Execute in background
	go mr.executeJob(ctx, job)

	mr.logger.Info("Meltano job started",
		zap.String("job_id", job.ID),
		zap.String("command", command),
		zap.Any("args", args))

	// Update metrics
	mr.updateMetrics("started")

	return map[string]interface{}{
		"job_id":    job.ID,
		"status":    job.Status,
		"command":   job.Command,
		"args":      job.Args,
		"timestamp": job.StartTime,
	}, nil
}

// GetStatus returns the current runtime status
func (mr *MeltanoRuntime) GetStatus(ctx context.Context) (*runtimes.RuntimeStatus, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	return &runtimes.RuntimeStatus{
		RuntimeType:  "meltano",
		Status:       mr.status,
		LastSeen:     mr.lastSeen,
		Capabilities: mr.capabilities,
		Config: map[string]string{
			"python_path":       mr.pythonPath,
			"meltano_project":   mr.meltanoProject,
			"virtual_env":       mr.virtualEnv,
			"active_jobs":       fmt.Sprintf("%d", mr.countActiveJobs()),
			"max_concurrency":   fmt.Sprintf("%d", mr.config.MaxConcurrency),
		},
	}, nil
}

// HealthCheck performs a health check on the Meltano runtime
func (mr *MeltanoRuntime) HealthCheck(ctx context.Context) error {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	if mr.status != "running" {
		return fmt.Errorf("Meltano runtime is not running")
	}

	// Test Meltano installation
	cmd := exec.CommandContext(ctx, mr.pythonPath, "-m", "meltano", "--version")
	cmd.Dir = mr.meltanoProject
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Meltano health check failed: %w", err)
	}

	mr.lastSeen = time.Now()
	return nil
}

// GetMetadata returns runtime metadata
func (mr *MeltanoRuntime) GetMetadata() runtimes.RuntimeMetadata {
	return runtimes.RuntimeMetadata{
		Name:        "Meltano Data Platform Runtime",
		Version:     "3.8.0+flext",
		Description: "Singer/DBT orchestration runtime with FlextCore integration",
		Capabilities: mr.capabilities,
	}
}

// Private helper methods

// validateMeltanoProject validates the Meltano project structure
func (mr *MeltanoRuntime) validateMeltanoProject() error {
	projectPath := mr.meltanoProject
	
	// Check if directory exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("Meltano project directory does not exist: %s", projectPath)
	}

	// Check for meltano.yml
	meltanoYml := filepath.Join(projectPath, "meltano.yml")
	if _, err := os.Stat(meltanoYml); os.IsNotExist(err) {
		return fmt.Errorf("meltano.yml not found in project directory: %s", projectPath)
	}

	mr.logger.Debug("Meltano project validated", 
		zap.String("project_path", projectPath))

	return nil
}

// validatePythonEnvironment validates Python and Meltano installation
func (mr *MeltanoRuntime) validatePythonEnvironment() error {
	// Check Python installation
	cmd := exec.Command(mr.pythonPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Python not found at %s: %w", mr.pythonPath, err)
	}

	// Check Meltano installation
	cmd = exec.Command(mr.pythonPath, "-m", "meltano", "--version")
	cmd.Dir = mr.meltanoProject
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Meltano not installed or not accessible: %w", err)
	}

	mr.logger.Debug("Python environment validated",
		zap.String("python_path", mr.pythonPath))

	return nil
}

// initializeFlextCoreIntegration sets up flext-core integration
func (mr *MeltanoRuntime) initializeFlextCoreIntegration() error {
	flextCorePath := mr.config.FlextCoreProject
	
	// Check if flext-core project exists
	if _, err := os.Stat(flextCorePath); os.IsNotExist(err) {
		return fmt.Errorf("flext-core project not found: %s", flextCorePath)
	}

	// Add flext-core to Python path
	if pythonPath := os.Getenv("PYTHONPATH"); pythonPath != "" {
		os.Setenv("PYTHONPATH", pythonPath+":"+filepath.Join(flextCorePath, "src"))
	} else {
		os.Setenv("PYTHONPATH", filepath.Join(flextCorePath, "src"))
	}

	mr.logger.Info("FlextCore integration initialized",
		zap.String("flext_core_path", flextCorePath))

	return nil
}

// applyConfigurationOverrides applies runtime configuration overrides
func (mr *MeltanoRuntime) applyConfigurationOverrides(config map[string]string) {
	for key, value := range config {
		switch key {
		case "python_path":
			mr.pythonPath = value
		case "meltano_project":
			mr.meltanoProject = value
		case "virtual_env":
			mr.virtualEnv = value
		case "max_concurrency":
			if concurrency, err := fmt.Sscanf(value, "%d", &mr.config.MaxConcurrency); err == nil && concurrency == 1 {
				mr.logger.Info("Max concurrency updated", 
					zap.Int("max_concurrency", mr.config.MaxConcurrency))
			}
		}
	}
}

// filterCapabilities filters capabilities based on provided list
func (mr *MeltanoRuntime) filterCapabilities(requested []string) {
	filteredCapabilities := []string{}
	
	for _, requested := range requested {
		for _, available := range mr.capabilities {
			if requested == available {
				filteredCapabilities = append(filteredCapabilities, requested)
				break
			}
		}
	}
	
	mr.capabilities = filteredCapabilities
}

// executeJob executes a Meltano job in background
func (mr *MeltanoRuntime) executeJob(ctx context.Context, job *MeltanoJob) {
	defer func() {
		mr.mu.Lock()
		endTime := time.Now()
		job.EndTime = &endTime
		mr.mu.Unlock()
		
		if job.Status == "running" {
			job.Status = "completed"
			mr.updateMetrics("completed")
		}
	}()

	job.Status = "running"

	// Build Meltano command
	meltanoCmd := mr.buildMeltanoCommand(job)
	
	// Execute command
	cmd := exec.CommandContext(ctx, mr.pythonPath, meltanoCmd...)
	cmd.Dir = mr.meltanoProject
	
	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range mr.config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture output
	output, err := cmd.CombinedOutput()
	job.Output = string(output)
	
	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			job.ExitCode = exitError.ExitCode()
		}
		mr.updateMetrics("failed")
		
		mr.logger.Error("Meltano job failed",
			zap.String("job_id", job.ID),
			zap.Error(err),
			zap.Int("exit_code", job.ExitCode))
	} else {
		job.Status = "completed"
		job.ExitCode = 0
		
		mr.logger.Info("Meltano job completed",
			zap.String("job_id", job.ID),
			zap.Duration("duration", time.Since(job.StartTime)))
	}
}

// buildMeltanoCommand builds the Meltano command from job specification
func (mr *MeltanoRuntime) buildMeltanoCommand(job *MeltanoJob) []string {
	cmd := []string{"-m", "meltano"}
	
	// Add command
	cmd = append(cmd, job.Command)
	
	// Add arguments
	cmd = append(cmd, job.Args...)
	
	// Add configuration flags if provided
	if job.Config != nil {
		for key, value := range job.Config {
			switch key {
			case "environment":
				if env, ok := value.(string); ok {
					cmd = append(cmd, "--environment", env)
				}
			case "state_id":
				if stateID, ok := value.(string); ok {
					cmd = append(cmd, "--state-id", stateID)
				}
			case "force":
				if force, ok := value.(bool); ok && force {
					cmd = append(cmd, "--force")
				}
			}
		}
	}
	
	return cmd
}

// countActiveJobs counts currently active jobs
func (mr *MeltanoRuntime) countActiveJobs() int {
	count := 0
	for _, job := range mr.activeJobs {
		if job.Status == "running" || job.Status == "pending" {
			count++
		}
	}
	return count
}

// cleanupCompletedJobs removes completed jobs from active jobs map
func (mr *MeltanoRuntime) cleanupCompletedJobs() {
	for jobID, job := range mr.activeJobs {
		if job.Status == "completed" || job.Status == "failed" {
			delete(mr.activeJobs, jobID)
		}
	}
}

// updateMetrics updates runtime performance metrics
func (mr *MeltanoRuntime) updateMetrics(status string) {
	mr.metrics.mu.Lock()
	defer mr.metrics.mu.Unlock()

	mr.metrics.LastJobExecution = time.Now()

	switch status {
	case "started":
		mr.metrics.TotalJobs++
		mr.metrics.ActiveJobs++
	case "completed":
		mr.metrics.CompletedJobs++
		if mr.metrics.ActiveJobs > 0 {
			mr.metrics.ActiveJobs--
		}
	case "failed":
		mr.metrics.FailedJobs++
		if mr.metrics.ActiveJobs > 0 {
			mr.metrics.ActiveJobs--
		}
	}

	// Update average execution time (simplified)
	if mr.metrics.CompletedJobs > 0 {
		mr.metrics.AverageExecution = float64(time.Since(mr.metrics.RuntimeStartTime).Milliseconds()) / float64(mr.metrics.CompletedJobs)
	}
}

// GetActiveJobs returns all active jobs (for monitoring)
func (mr *MeltanoRuntime) GetActiveJobs() map[string]*MeltanoJob {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	// Create a copy to avoid race conditions
	jobs := make(map[string]*MeltanoJob)
	for id, job := range mr.activeJobs {
		jobCopy := *job
		jobs[id] = &jobCopy
	}

	return jobs
}

// GetMetrics returns current runtime metrics
func (mr *MeltanoRuntime) GetMetrics() *MeltanoMetrics {
	mr.metrics.mu.RLock()
	defer mr.metrics.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *mr.metrics
	return &metrics
}