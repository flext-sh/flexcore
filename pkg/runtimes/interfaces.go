// Package runtimes - Runtime Management Interfaces
package runtimes

import (
	"context"
	"time"
)

// RuntimeStatus represents the status of a runtime
type RuntimeStatus struct {
	RuntimeType  string            `json:"runtime_type"`
	Status       string            `json:"status"`
	LastSeen     time.Time         `json:"last_seen"`
	Config       map[string]string `json:"config"`
	Capabilities []string          `json:"capabilities"`
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
	RuntimeType string                 `json:"runtime_type"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	Config      map[string]interface{} `json:"config"`
}

// RuntimeExecutionResult represents the result of a runtime execution
type RuntimeExecutionResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
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
	StartRuntime(ctx context.Context, runtimeType string, config map[string]string, capabilities []string) (*RuntimeStatus, error)
	StopRuntime(ctx context.Context, runtimeType string, force bool, timeout time.Duration) error
	GetRuntimeStatus(ctx context.Context, runtimeType string) (*RuntimeStatus, error)
	ListRuntimes(ctx context.Context, includeInactive bool) ([]*RuntimeStatus, error)
	ExecuteOnRuntime(ctx context.Context, req RuntimeExecutionRequest) (*RuntimeExecutionResult, error)
	IsRuntimeRegistered(runtimeType string) bool
	GetRuntimeMetadata(runtimeType string) (RuntimeMetadata, error)
}
