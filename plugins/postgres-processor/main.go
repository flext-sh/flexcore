// Postgres Processor Plugin - REAL Implementation using correct interface
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"

	hashicorpPlugin "github.com/hashicorp/go-plugin"
	_ "github.com/lib/pq"

	"github.com/flext-sh/flexcore/pkg/plugin"
)

const (
	processorType   = "postgres-processor"
	defaultFileMode = 0o644
)

// init registers types for gob encoding/decoding
// DRY PRINCIPLE: Uses shared plugin.RegisterPluginTypesForRPC() eliminating 18-line duplication
func init() {
	plugin.RegisterPluginTypesForRPC()
}

// PostgresProcessor implements a PostgreSQL data processing plugin using shared base implementation
// DRY PRINCIPLE: Composition over duplication - embeds BasePostgresProcessor for shared functionality
type PostgresProcessor struct {
	*plugin.BasePostgresProcessor
}

// DataProcessorPlugin interface
type DataProcessorPlugin interface {
	Initialize(ctx context.Context, config map[string]interface{}) error
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	GetInfo() plugin.PluginInfo
	HealthCheck(ctx context.Context) error
	Cleanup() error
}

// Initialize the plugin with configuration
// DRY PRINCIPLE: Delegates to BasePostgresProcessor.Initialize eliminating database connection duplication
func (pp *PostgresProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Printf("[PostgresProcessor] Initializing with config: %+v", config)

	// Initialize base processor
	if pp.BasePostgresProcessor == nil {
		pp.BasePostgresProcessor = plugin.NewBasePostgresProcessor()
	}

	// Delegate to base implementation
	if err := pp.BasePostgresProcessor.Initialize(ctx, config); err != nil {
		return err
	}

	// Try to connect to PostgreSQL if config provided
	host, hasHost := config["postgres_host"].(string)
	if !hasHost {
		return nil
	}

	connConfig := pp.buildConnectionConfig(config, host)
	if err := pp.connectToDatabase(connConfig); err != nil {
		log.Printf("[PostgresProcessor] Warning: PostgreSQL connection failed: %v", err)
		// Don't fail - continue in test mode
	}

	return nil
}

// buildConnectionConfig creates connection configuration from config map
func (pp *PostgresProcessor) buildConnectionConfig(config map[string]interface{}, host string) string {
	port := "5432"
	if p, ok := config["postgres_port"].(string); ok {
		port = p
	}

	dbname := "postgres"
	if db, ok := config["postgres_db"].(string); ok {
		dbname = db
	}

	user := "postgres"
	if u, ok := config["postgres_user"].(string); ok {
		user = u
	}

	password := ""
	if pw, ok := config["postgres_password"].(string); ok {
		password = pw
	}

	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host, port, dbname, user, password)
}

// connectToDatabase establishes database connection
func (pp *PostgresProcessor) connectToDatabase(connStr string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	pp.SetDB(db)
	log.Printf("[PostgresProcessor] PostgreSQL connected successfully")
	return nil
}

// Execute processes data with PostgreSQL
// DRY PRINCIPLE: Delegates to BasePostgresProcessor.ExecuteWithStats eliminating 50+ line duplication
func (pp *PostgresProcessor) Execute(ctx context.Context, input map[string]interface{}) (
	map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	log.Printf("[PostgresProcessor] Processing input with %d keys", len(input))

	// Delegate to shared implementation
	return pp.ExecuteWithStats(ctx, input, processorType)
}

// Removed executeQuery - redundant wrapper eliminated (SOLID SRP + DRY principle)
// All calls now use BasePostgresProcessor.ExecuteQuery directly

// GetInfo returns plugin metadata
func (pp *PostgresProcessor) GetInfo() plugin.PluginInfo {
	return plugin.PluginInfo{
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
			return fmt.Errorf("database health check failed: %w", err)
		}
	}

	return nil
}

// Cleanup releases resources
func (pp *PostgresProcessor) Cleanup() error {
	log.Printf("[PostgresProcessor] Cleanup called - processed %d records total", pp.GetStatistics().TotalRecords)

	if pp.GetDB() != nil {
		if err := pp.GetDB().Close(); err != nil {
			log.Printf("[PostgresProcessor] Warning: Failed to close database: %v", err)
		}
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

// Processing helper methods - delegated to BasePostgresProcessor

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

func (rpc *PostgresProcessorRPC) GetInfo(_ interface{}, resp *plugin.PluginInfo) {
	*resp = rpc.Impl.GetInfo()
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
	return &PostgresProcessorRPC{Impl: &PostgresProcessor{
		BasePostgresProcessor: plugin.NewBasePostgresProcessor(),
	}}, nil
}

func (PostgresProcessorPlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PostgresProcessorRPC{}, nil
}

// Main entry point
// DRY PRINCIPLE: Uses shared PluginMainUtilities to eliminate 26-line duplication (mass=110)
func main() {
	config := plugin.PluginMainConfig{
		PluginName: "postgres-processor",
		LogPrefix:  "[postgres-processor-plugin] ",
		StartMsg:   "Starting postgres processor plugin...",
		StopMsg:    "Postgres processor plugin stopped",
	}

	// Use shared main function eliminating duplication
	plugin.RunPluginMain(&config, func() hashicorpPlugin.Plugin {
		return &PostgresProcessorPlugin{}
	})
}
