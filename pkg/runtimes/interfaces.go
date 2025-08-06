// Package runtimes - Runtime Management Interfaces
package runtimes

import (
	"context"
	"time"
)

// RuntimeStatus represents the status of a runtime
type RuntimeStatus struct {
	// Large fields first (time.Time, maps, slices)
	StartTime     time.Time         `json:"start_time"`
	LastCheck     time.Time         `json:"last_check"`
	LastSeen      time.Time         `json:"last_seen"`
	Config        map[string]string `json:"config"`
	Metadata      map[string]string `json:"metadata"`
	Capabilities  []string          `json:"capabilities"`
	ResourceUsage ResourceUsage     `json:"resource_usage"`
	Uptime        time.Duration     `json:"uptime"`
	// String fields (pointer-sized)
	RuntimeType string `json:"runtime_type"`
	Status      string `json:"status"`
	Health      string `json:"health"`
	Version     string `json:"version"`
}

// ResourceUsage represents runtime resource usage statistics
type ResourceUsage struct {
	// 64-bit fields first
	MemoryBytes int64   `json:"memory_bytes"`
	DiskBytes   int64   `json:"disk_bytes"`
	CPUPercent  float64 `json:"cpu_percent"`
	// 32-bit fields after
	ActiveJobs int `json:"active_jobs"`
	QueuedJobs int `json:"queued_jobs"`
}

// RuntimeMetadata contains metadata about a runtime
type RuntimeMetadata struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

// RuntimeExecutionRequest represents a runtime execution request
type RuntimeExecutionRequest struct {
	// Large fields first (map, slice)
	Config map[string]interface{} `json:"config"`
	Args   []string               `json:"args"`
	// String fields after
	RuntimeType string `json:"runtime_type"`
	Command     string `json:"command"`
}

// RuntimeExecutionResult represents the result of a runtime execution
type RuntimeExecutionResult struct {
	// Interface field first (pointer-sized)
	Data interface{} `json:"data"`
	// String field after (pointer-sized)
	Error string `json:"error,omitempty"`
	// Bool field last (1 byte + padding)
	Success bool `json:"success"`
}

// Runtime interface for all runtime implementations
type Runtime interface {
	GetType() string
	Start(ctx context.Context, config map[string]string, capabilities []string) error
	Stop(ctx context.Context, force bool, timeout time.Duration) error
	Execute(ctx context.Context, command string, args []string, config map[string]interface{}) (interface{}, error)
	GetStatus(ctx context.Context) (*RuntimeStatus, error)
	HealthCheck(ctx context.Context) error
	GetMetadata() RuntimeMetadata
}

// RuntimeManager manages multiple runtime instances
type RuntimeManager interface {
	RegisterRuntime(runtime Runtime) error
	GetRuntime(runtimeType string) (Runtime, error)
	GetRuntimeTypes() []string
	StartRuntime(
		ctx context.Context,
		runtimeType string,
		config map[string]string,
		capabilities []string,
	) (*RuntimeStatus, error)
	StopRuntime(ctx context.Context, runtimeType string, force bool, timeout time.Duration) error
	GetRuntimeStatus(ctx context.Context, runtimeType string) (*RuntimeStatus, error)
	ListRuntimes(ctx context.Context, includeInactive bool) ([]*RuntimeStatus, error)
	ExecuteOnRuntime(ctx context.Context, req RuntimeExecutionRequest) (*RuntimeExecutionResult, error)
	IsRuntimeRegistered(runtimeType string) bool
	GetRuntimeMetadata(runtimeType string) (RuntimeMetadata, error)
}
