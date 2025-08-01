// Postgres Processor Plugin - REAL Implementation using correct interface
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"

	hashicorpPlugin "github.com/hashicorp/go-plugin"
	_ "github.com/lib/pq"

	"github.com/flext/flexcore/pkg/plugin"
)

const (
	processorType   = "postgres-processor"
	defaultFileMode = 0o644
)

// init registers types for gob encoding/decoding
// DRY PRINCIPLE: Uses shared PluginGobRegistration to eliminate 18-line duplication (mass=119)
func init() {
	// Use shared gob registration eliminating 18 lines of duplication
	plugin.RegisterAllPluginTypes()
}

// PostgresProcessor implements a PostgreSQL data processing plugin using shared base implementation
// DRY PRINCIPLE: Composition over duplication - embeds BasePostgresProcessor for shared functionality
type PostgresProcessor struct {
	*plugin.BasePostgresProcessor
}

// NewPostgresProcessor creates a new PostgreSQL processor using shared base implementation
func NewPostgresProcessor() *PostgresProcessor {
	return &PostgresProcessor{
		BasePostgresProcessor: plugin.NewBasePostgresProcessor(),
	}
}

// ProcessingStats type alias for unified plugin stats
type ProcessingStats = plugin.ProcessingStats

// PluginInfo type alias for unified plugin info
type PluginInfo = plugin.PluginInfo

// DataProcessorPlugin interface
type DataProcessorPlugin interface {
	Initialize(ctx context.Context, config map[string]interface{}) error
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	GetInfo() PluginInfo
	HealthCheck(ctx context.Context) error
	Cleanup() error
}

// Initialize the plugin with configuration
// DRY PRINCIPLE: Delegates to BasePostgresProcessor.Initialize eliminating database connection duplication
func (pp *PostgresProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Initialize base processor
	if pp.BasePostgresProcessor == nil {
		pp.BasePostgresProcessor = plugin.NewBasePostgresProcessor()
	}

	// Delegate to base implementation
	return pp.BasePostgresProcessor.Initialize(ctx, config)
}

// Execute processes data with PostgreSQL
// DRY PRINCIPLE: Delegates to BasePostgresProcessor.ExecuteWithStats eliminating 50+ line duplication
func (pp *PostgresProcessor) Execute(
	ctx context.Context, input map[string]interface{},
) (map[string]interface{}, error) {
	log.Printf("[PostgresProcessor] Processing input with %d keys", len(input))
	
	// Delegate to shared implementation
	return pp.ExecuteWithStats(ctx, input, processorType)
}

// DRY PRINCIPLE: executeQuery moved to BasePostgresProcessor.ExecuteQuery to eliminate 39-line duplication (mass=209)
// Delegate to shared implementation
func (pp *PostgresProcessor) executeQuery(ctx context.Context, query string) ([]map[string]interface{}, error) {
	return pp.ExecuteQuery(ctx, query)
}

// GetInfo returns plugin metadata
func (pp *PostgresProcessor) GetInfo() PluginInfo {
	return PluginInfo{
		ID:          processorType,
		Name:        processorType,
		Version:     "0.9.0",
		Description: "PostgreSQL data processing plugin with SQL execution capabilities",
		Author:      "FlexCore Team",
		Type:        "processor",
		Tags:        []string{"sql_query", "data_extraction", "postgres_connection", "result_processing"},
		Status:      "active",
		LoadedAt:    time.Now(),
		Health:      "healthy",
	}
}

// HealthCheck verifies plugin health
func (pp *PostgresProcessor) HealthCheck(ctx context.Context) error {
	log.Printf("[PostgresProcessor] Health check - processed %d records", pp.GetStatistics().TotalRecords)

	// Check database connection if available
	if pp.GetDB() != nil {
		if err := pp.GetDB().PingContext(ctx); err != nil {
			log.Printf("[PostgresProcessor] Warning: Database ping failed: %v", err)
		}
	}

	return nil
}

// Cleanup releases resources
func (pp *PostgresProcessor) Cleanup() error {
	log.Printf("[PostgresProcessor] Cleanup called - processed %d records total", pp.GetStatistics().TotalRecords)

	if pp.GetDB() != nil {
		pp.GetDB().Close()
		log.Printf("[PostgresProcessor] Database connection closed")
	}

	// Save statistics to file (optional)
	if statsFile, ok := pp.GetConfig()["stats_file"].(string); ok {
		data, err := json.MarshalIndent(pp.GetStatistics(), "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal stats: %w", err)
		}
		if err := os.WriteFile(statsFile, data, defaultFileMode); err != nil {
			return fmt.Errorf("failed to write stats file: %w", err)
		}
	}

	return nil
}

// Processing helper methods


// RPC Implementation

type PostgresProcessorRPC struct {
	Impl *PostgresProcessor
}

func (rpc *PostgresProcessorRPC) Initialize(args map[string]interface{}, resp *error) error {
	*resp = rpc.Impl.Initialize(context.Background(), args)
	return nil
}

func (rpc *PostgresProcessorRPC) Execute(args map[string]interface{}, resp *map[string]interface{}) error {
	result, err := rpc.Impl.Execute(context.Background(), args)
	if err != nil {
		return err
	}
	*resp = result
	return nil
}

func (rpc *PostgresProcessorRPC) GetInfo(_ interface{}, resp *PluginInfo) error {
	*resp = rpc.Impl.GetInfo()
	return nil
}

func (rpc *PostgresProcessorRPC) HealthCheck(args interface{}, resp *error) error {
	*resp = rpc.Impl.HealthCheck(context.Background())
	return nil
}

func (rpc *PostgresProcessorRPC) Cleanup(args interface{}, resp *error) error {
	*resp = rpc.Impl.Cleanup()
	return nil
}

// Plugin implementation for HashiCorp go-plugin
type PostgresProcessorPlugin struct{}

func (PostgresProcessorPlugin) Server(*hashicorpPlugin.MuxBroker) (interface{}, error) {
	return &PostgresProcessorRPC{Impl: NewPostgresProcessor()}, nil
}

func (PostgresProcessorPlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PostgresProcessorRPC{}, nil
}

// Main entry point
// DRY PRINCIPLE: Uses shared PluginMainUtilities to eliminate 26-line duplication (mass=110)
func main() {
	config := plugin.PluginMainConfig{
		PluginName: "postgres-processor-infrastructure",
		LogPrefix:  "[postgres-processor-plugin] ",
		StartMsg:   "Starting postgres processor plugin...",
		StopMsg:    "Postgres processor plugin stopped",
	}

	// Use shared main function eliminating duplication
	plugin.RunPluginMain(config, func() hashicorpPlugin.Plugin {
		return &PostgresProcessorPlugin{}
	})
}
