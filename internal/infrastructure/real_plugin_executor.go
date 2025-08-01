package infrastructure

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/flext/flexcore/pkg/logging"
	"github.com/flext/flexcore/pkg/result"
	"go.uber.org/zap"
)

// RealFlextServicePlugin implements actual FLEXT service execution as subprocess
type RealFlextServicePlugin struct {
	servicePath string
	configPath  string
	workingDir  string
	logger      logging.LoggerInterface
	process     *exec.Cmd
	cancel      context.CancelFunc
	running     bool
	mu          sync.RWMutex
}

// NewRealFlextServicePlugin creates a new real FLEXT service plugin
func NewRealFlextServicePlugin(servicePath, configPath, workingDir string) *RealFlextServicePlugin {
	// Make paths absolute to avoid issues
	absServicePath, _ := filepath.Abs(servicePath)
	absConfigPath, _ := filepath.Abs(configPath)
	absWorkingDir, _ := filepath.Abs(workingDir)

	return &RealFlextServicePlugin{
		servicePath: absServicePath,
		configPath:  absConfigPath,
		workingDir:  absWorkingDir,
		logger:      logging.NewLogger("real-flext-service-plugin"),
		running:     false,
	}
}

// Name returns the plugin name
func (rfsp *RealFlextServicePlugin) Name() string {
	return "real-flext-service"
}

// Version returns the plugin version
func (rfsp *RealFlextServicePlugin) Version() string {
	return "2.0.0"
}

// Description returns the plugin description
func (rfsp *RealFlextServicePlugin) Description() string {
	return "Real FLEXT service execution as subprocess with proper process management"
}

// IsHealthy checks if the plugin is healthy
func (rfsp *RealFlextServicePlugin) IsHealthy() bool {
	rfsp.mu.RLock()
	defer rfsp.mu.RUnlock()
	return !rfsp.running || (rfsp.process != nil && rfsp.process.ProcessState == nil)
}

// Execute executes the FLEXT service as a real subprocess with proper management
// SOLID SRP: Reduced from 6 returns to 1 return using Result pattern and ExecutionOrchestrator
func (rfsp *RealFlextServicePlugin) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	rfsp.mu.Lock()
	defer rfsp.mu.Unlock()

	// Use orchestrator for centralized error handling
	orchestrator := rfsp.createExecutionOrchestrator(ctx, params)
	executionResult := orchestrator.ExecuteService()
	
	if executionResult.IsFailure() {
		return nil, executionResult.Error()
	}

	return executionResult.Value(), nil
}

// RealPluginExecutionOrchestrator handles complete plugin execution with Result pattern
// SOLID SRP: Single responsibility for plugin execution orchestration
type RealPluginExecutionOrchestrator struct {
	plugin *RealFlextServicePlugin
	ctx    context.Context
	params map[string]interface{}
}

// createExecutionOrchestrator creates a specialized execution orchestrator
// SOLID SRP: Factory method for creating specialized orchestrators
func (rfsp *RealFlextServicePlugin) createExecutionOrchestrator(ctx context.Context, params map[string]interface{}) *RealPluginExecutionOrchestrator {
	return &RealPluginExecutionOrchestrator{
		plugin: rfsp,
		ctx:    ctx,
		params: params,
	}
}

// ExecuteService executes the complete service with centralized error handling
// SOLID SRP: Single responsibility for complete service execution
func (orchestrator *RealPluginExecutionOrchestrator) ExecuteService() result.Result[interface{}] {
	// Check if already running
	if orchestrator.plugin.running {
		return result.Failure[interface{}](fmt.Errorf("FLEXT service plugin is already running"))
	}

	// Setup execution environment
	setupResult := orchestrator.setupExecution()
	if setupResult.IsFailure() {
		return result.Failure[interface{}](setupResult.Error())
	}

	cmd := setupResult.Value()

	// Execute process with monitoring
	executionResult := orchestrator.executeProcessWithMonitoring(cmd)
	if executionResult.IsFailure() {
		return result.Failure[interface{}](executionResult.Error())
	}

	return result.Success(executionResult.Value())
}

// setupExecution sets up the execution environment and command
// SOLID SRP: Single responsibility for execution setup
func (orchestrator *RealPluginExecutionOrchestrator) setupExecution() result.Result[*exec.Cmd] {
	orchestrator.plugin.logger.Info("Executing real FLEXT service subprocess",
		zap.String("service_path", orchestrator.plugin.servicePath),
		zap.String("config_path", orchestrator.plugin.configPath),
		zap.String("working_dir", orchestrator.plugin.workingDir),
		zap.Any("params", orchestrator.params))

	// Create context with cancellation
	execCtx, cancel := context.WithCancel(orchestrator.ctx)
	orchestrator.plugin.cancel = cancel

	// Create command
	cmdResult := orchestrator.createCommand(execCtx)
	if cmdResult.IsFailure() {
		return result.Failure[*exec.Cmd](cmdResult.Error())
	}

	cmd := cmdResult.Value()

	// Setup pipes
	pipeResult := orchestrator.setupPipes(cmd)
	if pipeResult.IsFailure() {
		return result.Failure[*exec.Cmd](pipeResult.Error())
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return result.Failure[*exec.Cmd](fmt.Errorf("failed to start FLEXT service: %w", err))
	}

	orchestrator.plugin.process = cmd
	orchestrator.plugin.running = true

	orchestrator.plugin.logger.Info("FLEXT service subprocess started",
		zap.Int("pid", cmd.Process.Pid),
		zap.String("command", cmd.String()))

	return result.Success(cmd)
}

// createCommand creates the appropriate command based on service path
// SOLID SRP: Single responsibility for command creation
func (orchestrator *RealPluginExecutionOrchestrator) createCommand(execCtx context.Context) result.Result[*exec.Cmd] {
	var cmd *exec.Cmd
	if strings.HasSuffix(orchestrator.plugin.servicePath, ".go") {
		cmd = exec.CommandContext(execCtx, "go", "run", orchestrator.plugin.servicePath)
	} else if strings.HasSuffix(orchestrator.plugin.servicePath, ".py") {
		cmd = exec.CommandContext(execCtx, "python", orchestrator.plugin.servicePath)
	} else {
		cmd = exec.CommandContext(execCtx, orchestrator.plugin.servicePath)
	}

	cmd.Dir = orchestrator.plugin.workingDir

	// Setup environment
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("FLEXT_CONFIG_PATH=%s", orchestrator.plugin.configPath),
		"FLEXT_ENV=production",
		"FLEXT_DEBUG=false",
		"PYTHONPATH=/home/marlonsc/flext/flext-core/src:/home/marlonsc/flext/flext-meltano/src",
	)

	// Add parameters as environment variables
	for key, value := range orchestrator.params {
		envKey := fmt.Sprintf("FLEXT_%s", strings.ToUpper(key))
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", envKey, value))
	}

	return result.Success(cmd)
}

// setupPipes creates stdout and stderr pipes
// SOLID SRP: Single responsibility for pipe setup
func (orchestrator *RealPluginExecutionOrchestrator) setupPipes(cmd *exec.Cmd) result.Result[bool] {
	_, err := cmd.StdoutPipe()
	if err != nil {
		return result.Failure[bool](fmt.Errorf("failed to create stdout pipe: %w", err))
	}

	_, err = cmd.StderrPipe()
	if err != nil {
		return result.Failure[bool](fmt.Errorf("failed to create stderr pipe: %w", err))
	}

	return result.Success(true)
}

// executeProcessWithMonitoring executes process with complete monitoring
// SOLID SRP: Single responsibility for process execution and monitoring
func (orchestrator *RealPluginExecutionOrchestrator) executeProcessWithMonitoring(cmd *exec.Cmd) result.Result[interface{}] {
	startTime := time.Now()

	// Wait for process completion or timeout
	processErr := make(chan error, 1)
	go func() {
		processErr <- cmd.Wait()
	}()

	var execErr error
	select {
	case <-orchestrator.ctx.Done():
		orchestrator.plugin.logger.Warn("FLEXT service execution cancelled by context")
		if err := cmd.Process.Kill(); err != nil {
			orchestrator.plugin.logger.Error("Failed to kill FLEXT service process", zap.Error(err))
		}
		execErr = orchestrator.ctx.Err()
	case err := <-processErr:
		execErr = err
	case <-time.After(5 * time.Minute):
		orchestrator.plugin.logger.Warn("FLEXT service execution timed out")
		if err := cmd.Process.Kill(); err != nil {
			orchestrator.plugin.logger.Error("Failed to kill FLEXT service process", zap.Error(err))
		}
		execErr = fmt.Errorf("execution timeout")
	}

	orchestrator.plugin.running = false
	execDuration := time.Since(startTime)

	if execErr != nil {
		orchestrator.plugin.logger.Error("FLEXT service subprocess execution failed",
			zap.Error(execErr),
			zap.Duration("duration", execDuration))

		failureResult := map[string]interface{}{
			"status":         "failed",
			"error":          execErr.Error(),
			"duration":       execDuration.String(),
			"exit_code":      cmd.ProcessState.ExitCode(),
			"plugin_name":    orchestrator.plugin.Name(),
			"plugin_version": orchestrator.plugin.Version(),
		}
		return result.Success[interface{}](failureResult)
	}

	orchestrator.plugin.logger.Info("FLEXT service subprocess executed successfully",
		zap.Duration("duration", execDuration),
		zap.Int("exit_code", cmd.ProcessState.ExitCode()))

	successResult := map[string]interface{}{
		"status":         "completed",
		"duration":       execDuration.String(),
		"exit_code":      cmd.ProcessState.ExitCode(),
		"execution_time": time.Now().UTC(),
		"plugin_name":    orchestrator.plugin.Name(),
		"plugin_version": orchestrator.plugin.Version(),
		"process_id":     cmd.Process.Pid,
	}
	return result.Success[interface{}](successResult)
}

// captureOutput captures output from a pipe and logs it in real-time
func (rfsp *RealFlextServicePlugin) captureOutput(pipe io.ReadCloser, builder *strings.Builder, streamType string) {
	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {
		line := scanner.Text()
		builder.WriteString(line + "\n")

		// Log output in real-time with appropriate level
		if streamType == "stderr" {
			rfsp.logger.Warn("FLEXT service stderr", zap.String("line", line))
		} else {
			rfsp.logger.Info("FLEXT service stdout", zap.String("line", line))
		}
	}

	if err := scanner.Err(); err != nil {
		rfsp.logger.Error("Error reading FLEXT service output", zap.Error(err), zap.String("stream", streamType))
	}
}

// Stop stops the running FLEXT service
func (rfsp *RealFlextServicePlugin) Stop() error {
	rfsp.mu.Lock()
	defer rfsp.mu.Unlock()

	if !rfsp.running || rfsp.process == nil {
		return nil
	}

	rfsp.logger.Info("Stopping FLEXT service subprocess",
		zap.Int("pid", rfsp.process.Process.Pid))

	// Cancel context to signal shutdown
	if rfsp.cancel != nil {
		rfsp.cancel()
	}

	// Send SIGTERM first
	if err := rfsp.process.Process.Signal(os.Interrupt); err != nil {
		rfsp.logger.Warn("Failed to send interrupt signal", zap.Error(err))
	}

	// Wait a bit for graceful shutdown
	time.Sleep(2 * time.Second)

	// Force kill if still running
	if rfsp.process.ProcessState == nil {
		if err := rfsp.process.Process.Kill(); err != nil {
			rfsp.logger.Error("Failed to kill FLEXT service process", zap.Error(err))
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for process to finish
	_ = rfsp.process.Wait()

	rfsp.running = false
	rfsp.process = nil

	rfsp.logger.Info("FLEXT service subprocess stopped successfully")
	return nil
}

// GetStatus returns the current status of the plugin
func (rfsp *RealFlextServicePlugin) GetStatus() map[string]interface{} {
	rfsp.mu.RLock()
	defer rfsp.mu.RUnlock()

	status := map[string]interface{}{
		"plugin_name":    rfsp.Name(),
		"plugin_version": rfsp.Version(),
		"running":        rfsp.running,
		"healthy":        rfsp.IsHealthy(),
		"service_path":   rfsp.servicePath,
		"config_path":    rfsp.configPath,
		"working_dir":    rfsp.workingDir,
	}

	if rfsp.process != nil {
		status["process_id"] = rfsp.process.Process.Pid
		if rfsp.process.ProcessState != nil {
			status["exit_code"] = rfsp.process.ProcessState.ExitCode()
		}
	}

	return status
}