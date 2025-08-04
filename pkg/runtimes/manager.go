// Package runtimes - Default Runtime Manager Implementation
package runtimes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DefaultRuntimeManager is the default implementation of RuntimeManager
type DefaultRuntimeManager struct {
	mu       sync.RWMutex
	runtimes map[string]Runtime
	logger   *zap.Logger
}

// NewDefaultRuntimeManager creates a new default runtime manager
func NewDefaultRuntimeManager() *DefaultRuntimeManager {
	logger, _ := zap.NewProduction()
	return &DefaultRuntimeManager{
		runtimes: make(map[string]Runtime),
		logger:   logger,
	}
}

// RegisterRuntime registers a runtime
func (rm *DefaultRuntimeManager) RegisterRuntime(runtime Runtime) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	runtimeType := runtime.GetType()
	if _, exists := rm.runtimes[runtimeType]; exists {
		return fmt.Errorf("runtime %s already registered", runtimeType)
	}

	rm.runtimes[runtimeType] = runtime
	rm.logger.Info("Runtime registered", zap.String("runtime_type", runtimeType))
	return nil
}

// GetRuntime gets a runtime by type
func (rm *DefaultRuntimeManager) GetRuntime(runtimeType string) (Runtime, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	runtime, exists := rm.runtimes[runtimeType]
	if !exists {
		return nil, fmt.Errorf("runtime %s not found", runtimeType)
	}

	return runtime, nil
}

// GetRuntimeTypes returns all registered runtime types
func (rm *DefaultRuntimeManager) GetRuntimeTypes() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	types := make([]string, 0, len(rm.runtimes))
	for runtimeType := range rm.runtimes {
		types = append(types, runtimeType)
	}

	return types
}

// StartRuntime starts a runtime
func (rm *DefaultRuntimeManager) StartRuntime(ctx context.Context, runtimeType string, config map[string]string, capabilities []string) (*RuntimeStatus, error) {
	runtime, err := rm.GetRuntime(runtimeType)
	if err != nil {
		return nil, err
	}

	if err := runtime.Start(ctx, config, capabilities); err != nil {
		return nil, fmt.Errorf("failed to start runtime %s: %w", runtimeType, err)
	}

	return runtime.GetStatus(ctx)
}

// StopRuntime stops a runtime
func (rm *DefaultRuntimeManager) StopRuntime(ctx context.Context, runtimeType string, force bool, timeout time.Duration) error {
	runtime, err := rm.GetRuntime(runtimeType)
	if err != nil {
		return err
	}

	return runtime.Stop(ctx, force, timeout)
}

// GetRuntimeStatus gets the status of a runtime
func (rm *DefaultRuntimeManager) GetRuntimeStatus(ctx context.Context, runtimeType string) (*RuntimeStatus, error) {
	runtime, err := rm.GetRuntime(runtimeType)
	if err != nil {
		return nil, err
	}

	return runtime.GetStatus(ctx)
}

// ListRuntimes lists all runtimes
func (rm *DefaultRuntimeManager) ListRuntimes(ctx context.Context, includeInactive bool) ([]*RuntimeStatus, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	statuses := make([]*RuntimeStatus, 0, len(rm.runtimes))
	for _, runtime := range rm.runtimes {
		status, err := runtime.GetStatus(ctx)
		if err != nil {
			continue
		}

		if !includeInactive && status.Status != "running" {
			continue
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// ExecuteOnRuntime executes a command on a runtime
func (rm *DefaultRuntimeManager) ExecuteOnRuntime(ctx context.Context, req RuntimeExecutionRequest) (*RuntimeExecutionResult, error) {
	runtime, err := rm.GetRuntime(req.RuntimeType)
	if err != nil {
		return &RuntimeExecutionResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	data, err := runtime.Execute(ctx, req.Command, req.Args, req.Config)
	if err != nil {
		return &RuntimeExecutionResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &RuntimeExecutionResult{
		Success: true,
		Data:    data,
	}, nil
}

// IsRuntimeRegistered checks if a runtime is registered
func (rm *DefaultRuntimeManager) IsRuntimeRegistered(runtimeType string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	_, exists := rm.runtimes[runtimeType]
	return exists
}

// GetRuntimeMetadata gets metadata for a runtime
func (rm *DefaultRuntimeManager) GetRuntimeMetadata(runtimeType string) (RuntimeMetadata, error) {
	runtime, err := rm.GetRuntime(runtimeType)
	if err != nil {
		return RuntimeMetadata{}, err
	}

	return runtime.GetMetadata(), nil
}
