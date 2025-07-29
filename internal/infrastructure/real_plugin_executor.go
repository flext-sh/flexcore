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
	"go.uber.org/zap"
)

// RealFlextServicePlugin implements actual FLEXT service execution as subprocess
type RealFlextServicePlugin struct {
	servicePath string
	configPath  string
	workingDir  string
	logger      logging.LoggerInterface
	mu          sync.RWMutex
	running     bool
	process     *exec.Cmd
	cancel      context.CancelFunc
}

// NewRealFlextServicePlugin creates a new real FLEXT service plugin
func NewRealFlextServicePlugin(servicePath, configPath string, logger logging.LoggerInterface) *RealFlextServicePlugin {
	// Determine working directory from service path
	workingDir := filepath.Dir(servicePath)
	if workingDir == "." {
		workingDir = "/home/marlonsc/flext"
	}
	// Ensure working directory exists for FLEXT service
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		workingDir = "/home/marlonsc/flext"
	}

	return &RealFlextServicePlugin{
		servicePath: servicePath,
		configPath:  configPath,
		workingDir:  workingDir,
		logger:      logger,
	}
}

// Name returns the plugin name
func (rfsp *RealFlextServicePlugin) Name() string {
	return "flext-service-real"
}

// Version returns the plugin version
func (rfsp *RealFlextServicePlugin) Version() string {
	return "2.0.0"
}

// Execute executes the FLEXT service as a real subprocess with proper management
func (rfsp *RealFlextServicePlugin) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	rfsp.mu.Lock()
	defer rfsp.mu.Unlock()

	if rfsp.running {
		return nil, fmt.Errorf("FLEXT service plugin is already running")
	}

	rfsp.logger.Info("Executing real FLEXT service subprocess",
		zap.String("service_path", rfsp.servicePath),
		zap.String("config_path", rfsp.configPath),
		zap.String("working_dir", rfsp.workingDir),
		zap.Any("params", params))

	// Create context with cancellation
	execCtx, cancel := context.WithCancel(ctx)
	rfsp.cancel = cancel

	// Determine if this is a Go service or Python script
	var cmd *exec.Cmd
	if strings.HasSuffix(rfsp.servicePath, ".go") {
		// Go service - run with go run
		cmd = exec.CommandContext(execCtx, "go", "run", rfsp.servicePath)
	} else if strings.HasSuffix(rfsp.servicePath, ".py") {
		// Python script - run with python
		cmd = exec.CommandContext(execCtx, "python", rfsp.servicePath)
	} else {
		// Binary executable
		cmd = exec.CommandContext(execCtx, rfsp.servicePath)
	}

	// Set working directory
	cmd.Dir = rfsp.workingDir

	// Set environment variables for FLEXT service
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("FLEXT_CONFIG_PATH=%s", rfsp.configPath),
		"FLEXT_ENV=production",
		"FLEXT_DEBUG=false",
		"PYTHONPATH=/home/marlonsc/flext/flext-core/src:/home/marlonsc/flext/flext-meltano/src",
	)

	// Add parameters as environment variables
	for key, value := range params {
		envKey := fmt.Sprintf("FLEXT_%s", strings.ToUpper(key))
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", envKey, value))
	}

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start FLEXT service: %w", err)
	}

	rfsp.process = cmd
	rfsp.running = true

	rfsp.logger.Info("FLEXT service subprocess started",
		zap.Int("pid", cmd.Process.Pid),
		zap.String("command", cmd.String()))

	// Capture output in separate goroutines
	var stdoutBuilder, stderrBuilder strings.Builder
	var wg sync.WaitGroup

	wg.Add(2)

	// Capture stdout
	go func() {
		defer wg.Done()
		rfsp.captureOutput(stdout, &stdoutBuilder, "stdout")
	}()

	// Capture stderr
	go func() {
		defer wg.Done()
		rfsp.captureOutput(stderr, &stderrBuilder, "stderr")
	}()

	// Wait for process completion or timeout
	processErr := make(chan error, 1)
	go func() {
		processErr <- cmd.Wait()
	}()

	var execErr error
	select {
	case <-ctx.Done():
		// Context cancelled - kill the process
		rfsp.logger.Warn("FLEXT service execution cancelled by context")
		if err := cmd.Process.Kill(); err != nil {
			rfsp.logger.Error("Failed to kill FLEXT service process", zap.Error(err))
		}
		execErr = ctx.Err()
	case err := <-processErr:
		// Process completed
		execErr = err
	case <-time.After(5 * time.Minute):
		// Timeout - kill the process
		rfsp.logger.Warn("FLEXT service execution timed out")
		if err := cmd.Process.Kill(); err != nil {
			rfsp.logger.Error("Failed to kill FLEXT service process", zap.Error(err))
		}
		execErr = fmt.Errorf("execution timeout")
	}

	// Wait for output capture to complete
	wg.Wait()

	rfsp.running = false
	execDuration := time.Since(startTime)

	stdoutOutput := stdoutBuilder.String()
	stderrOutput := stderrBuilder.String()

	if execErr != nil {
		rfsp.logger.Error("FLEXT service subprocess execution failed",
			zap.Error(execErr),
			zap.String("stdout", stdoutOutput),
			zap.String("stderr", stderrOutput),
			zap.Duration("duration", execDuration))

		return map[string]interface{}{
			"status":         "failed",
			"error":          execErr.Error(),
			"stdout":         stdoutOutput,
			"stderr":         stderrOutput,
			"duration":       execDuration.String(),
			"exit_code":      cmd.ProcessState.ExitCode(),
			"plugin_name":    rfsp.Name(),
			"plugin_version": rfsp.Version(),
		}, fmt.Errorf("FLEXT service execution failed: %w", execErr)
	}

	rfsp.logger.Info("FLEXT service subprocess executed successfully",
		zap.String("stdout", stdoutOutput),
		zap.String("stderr", stderrOutput),
		zap.Duration("duration", execDuration),
		zap.Int("exit_code", cmd.ProcessState.ExitCode()))

	// Return detailed execution result
	return map[string]interface{}{
		"status":         "completed",
		"stdout":         stdoutOutput,
		"stderr":         stderrOutput,
		"duration":       execDuration.String(),
		"exit_code":      cmd.ProcessState.ExitCode(),
		"execution_time": time.Now().UTC(),
		"plugin_name":    rfsp.Name(),
		"plugin_version": rfsp.Version(),
		"process_id":     cmd.Process.Pid,
	}, nil
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
			rfsp.logger.Debug("FLEXT service stdout", zap.String("line", line))
		}
	}

	if err := scanner.Err(); err != nil {
		rfsp.logger.Error("Error reading FLEXT service output",
			zap.String("stream", streamType),
			zap.Error(err))
	}
}

// Shutdown gracefully shuts down the plugin
func (rfsp *RealFlextServicePlugin) Shutdown() error {
	rfsp.mu.Lock()
	defer rfsp.mu.Unlock()

	if !rfsp.running {
		return nil
	}

	rfsp.logger.Info("Shutting down FLEXT service plugin")

	// Cancel the context first
	if rfsp.cancel != nil {
		rfsp.cancel()
	}

	// If process is still running, try graceful shutdown first
	if rfsp.process != nil && rfsp.process.Process != nil {
		// Send SIGTERM for graceful shutdown
		if err := rfsp.process.Process.Signal(os.Interrupt); err != nil {
			rfsp.logger.Warn("Failed to send interrupt signal", zap.Error(err))
		}

		// Wait up to 10 seconds for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- rfsp.process.Wait()
		}()

		select {
		case <-done:
			rfsp.logger.Info("FLEXT service shutdown gracefully")
		case <-time.After(10 * time.Second):
			// Force kill if graceful shutdown fails
			rfsp.logger.Warn("Forcing FLEXT service shutdown")
			if err := rfsp.process.Process.Kill(); err != nil {
				rfsp.logger.Error("Failed to force kill FLEXT service", zap.Error(err))
			}
		}
	}

	rfsp.running = false
	rfsp.process = nil
	rfsp.cancel = nil

	rfsp.logger.Info("FLEXT service plugin shutdown complete")
	return nil
}

// IsRunning returns whether the plugin is currently running
func (rfsp *RealFlextServicePlugin) IsRunning() bool {
	rfsp.mu.RLock()
	defer rfsp.mu.RUnlock()
	return rfsp.running
}

// GetProcessID returns the process ID if running
func (rfsp *RealFlextServicePlugin) GetProcessID() int {
	rfsp.mu.RLock()
	defer rfsp.mu.RUnlock()

	if rfsp.process != nil && rfsp.process.Process != nil {
		return rfsp.process.Process.Pid
	}
	return 0
}
