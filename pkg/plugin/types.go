// Package plugin provides unified plugin types and interfaces for FlexCore
package plugin

import (
	"context"
	"time"

	"github.com/flext-sh/flexcore/pkg/result"
)

// ProcessingStats represents plugin processing statistics
type ProcessingStats struct {
	TotalRecords    int64     `json:"total_records"`
	ProcessedOK     int64     `json:"processed_ok"`
	ProcessedError  int64     `json:"processed_error"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	DurationMs      int64     `json:"duration_ms"`
	RecordsPerSec   float64   `json:"records_per_sec"`
	ErrorRate       float64   `json:"error_rate"`
	MemoryUsedBytes int64     `json:"memory_used_bytes"`
}

// PluginInfo represents plugin information
type PluginInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Author      string            `json:"author"`
	Tags        []string          `json:"tags"`
	Config      map[string]string `json:"config"`
	Status      string            `json:"status"`
	LoadedAt    time.Time         `json:"loaded_at"`
	Health      string            `json:"health"`
}

// DataProcessorPlugin interface for data processing plugins
type DataProcessorPlugin interface {
	GetInfo() PluginInfo
	Process(ctx context.Context, data []byte) result.Result[[]byte]
	GetStats() ProcessingStats
	HealthCheck() result.Result[bool]
	Shutdown() error
}

// ProcessingResult represents the result of data processing
type ProcessingResult struct {
	Data     []byte                 `json:"data"`
	Stats    ProcessingStats        `json:"stats"`
	Metadata map[string]interface{} `json:"metadata"`
	Errors   []string               `json:"errors,omitempty"`
}

// ProcessingRequest represents a processing request
type ProcessingRequest struct {
	Data     []byte                 `json:"data"`
	Config   map[string]string      `json:"config"`
	Metadata map[string]interface{} `json:"metadata"`
	Timeout  time.Duration          `json:"timeout"`
}

// HealthCheckError represents a health check error
type HealthCheckError struct {
	Message string
}

func (e *HealthCheckError) Error() string {
	return e.Message
}
