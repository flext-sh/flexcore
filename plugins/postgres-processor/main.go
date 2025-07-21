// Postgres Processor Plugin - REAL Implementation using correct interface
package main

import (
	"context"
	"database/sql"
	"encoding/gob"
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
func init() {
	// Register types for RPC serialization
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register([]map[string]interface{}{})

	// Register primitive types
	gob.Register(string(""))
	gob.Register(int(0))
	gob.Register(int64(0))
	gob.Register(float64(0))
	gob.Register(bool(false))
	gob.Register(time.Time{})

	// Register plugin types
	gob.Register(plugin.PluginInfo{})
	gob.Register(plugin.ProcessingStats{})
}

// PostgresProcessor implements a PostgreSQL data processing plugin
type PostgresProcessor struct {
	config map[string]interface{}
	db     *sql.DB
	stats  plugin.ProcessingStats
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
func (pp *PostgresProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Printf("[PostgresProcessor] Initializing with config: %+v", config)

	pp.config = config
	pp.stats = plugin.ProcessingStats{}

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

	pp.db = db
	log.Printf("[PostgresProcessor] PostgreSQL connected successfully")
	return nil
}

// Execute processes data with PostgreSQL
func (pp *PostgresProcessor) Execute(ctx context.Context, input map[string]interface{}) (
	map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	startTime := time.Now()
	defer func() {
		pp.stats.DurationMs = time.Since(startTime).Milliseconds()
		pp.stats.EndTime = time.Now()
	}()

	log.Printf("[PostgresProcessor] Processing input with %d keys", len(input))

	result := make(map[string]interface{})
	result["processor"] = processorType
	result["timestamp"] = time.Now().Unix()
	result["processed_by"] = "FlexCore PostgresProcessor v1.0"

	// Process the input data
	if query, ok := input["sql_query"].(string); ok && pp.db != nil {
		// Execute SQL query
		queryResult, err := pp.executeQuery(ctx, query)
		if err != nil {
			result["error"] = err.Error()
			result["data"] = input
		} else {
			result["data"] = queryResult
			pp.stats.ProcessedOK = int64(len(queryResult))
		}
	} else if data, ok := input["data"]; ok {
		// Process data without database
		switch v := data.(type) {
		case []interface{}:
			processed := pp.processArray(v)
			result["data"] = processed
			result["records_count"] = len(processed)
		case map[string]interface{}:
			processed := pp.processMap(v)
			result["data"] = processed
			result["records_count"] = 1
		case string:
			processed := pp.processString(v)
			result["data"] = processed
			result["records_count"] = 1
		default:
			result["data"] = data
			result["records_count"] = 1
		}
	} else {
		result["data"] = input
		result["records_count"] = 1
	}

	// Add processing stats
	result["stats"] = map[string]interface{}{
		"total_records":      pp.stats.TotalRecords,
		"processed_ok":       pp.stats.ProcessedOK,
		"processed_error":    pp.stats.ProcessedError,
		"duration_ms":        pp.stats.DurationMs,
		"records_per_sec":    pp.stats.RecordsPerSec,
		"database_connected": pp.db != nil,
	}

	pp.stats.TotalRecords++
	pp.stats.ProcessedOK++
	return result, nil
}

// executeQuery runs a SQL query and returns results
func (pp *PostgresProcessor) executeQuery(ctx context.Context, query string) ([]map[string]interface{}, error) {
	rows, err := pp.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}

	for rows.Next() {
		// Create a slice to hold values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}

		results = append(results, rowMap)
	}

	return results, nil
}

// GetInfo returns plugin metadata
func (pp *PostgresProcessor) GetInfo() plugin.PluginInfo {
	return plugin.PluginInfo{
		ID:          processorType,
		Name:        processorType,
		Version:     "1.0.0",
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
	log.Printf("[PostgresProcessor] Health check - processed %d records", pp.stats.TotalRecords)

	// Check database connection if available
	if pp.db != nil {
		if err := pp.db.PingContext(ctx); err != nil {
			return fmt.Errorf("database health check failed: %w", err)
		}
	}

	return nil
}

// Cleanup releases resources
func (pp *PostgresProcessor) Cleanup() error {
	log.Printf("[PostgresProcessor] Cleanup called - processed %d records total", pp.stats.TotalRecords)

	if pp.db != nil {
		if err := pp.db.Close(); err != nil {
			log.Printf("[PostgresProcessor] Warning: Failed to close database: %v", err)
		}
		log.Printf("[PostgresProcessor] Database connection closed")
	}

	// Save statistics to file (optional)
	if statsFile, ok := pp.config["stats_file"].(string); ok {
		data, err := json.MarshalIndent(pp.stats, "", "  ")
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

func (pp *PostgresProcessor) processArray(data []interface{}) []interface{} {
	processed := make([]interface{}, 0, len(data))

	for _, item := range data {
		// Add metadata
		if itemMap, ok := item.(map[string]interface{}); ok {
			itemMap["_processed_at"] = time.Now().Unix()
			itemMap["_processor"] = processorType
			processed = append(processed, itemMap)
		} else {
			processed = append(processed, item)
		}
	}

	return processed
}

func (pp *PostgresProcessor) processMap(data map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for key, value := range data {
		processed[key] = value
	}

	// Add metadata
	processed["_processed_at"] = time.Now().Unix()
	processed["_processor"] = processorType

	return processed
}

func (pp *PostgresProcessor) processString(data string) string {
	return "[POSTGRES_PROCESSED] " + data
}

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

func (rpc *PostgresProcessorRPC) GetInfo(_ interface{}, resp *plugin.PluginInfo) error {
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
	return &PostgresProcessorRPC{Impl: &PostgresProcessor{}}, nil
}

func (PostgresProcessorPlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PostgresProcessorRPC{}, nil
}

// Main entry point
func main() {
	log.SetPrefix("[postgres-processor-plugin] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Println("Starting postgres processor plugin...")

	// Handshake configuration
	handshakeConfig := hashicorpPlugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: "flexcore-plugin-magic-cookie",
	}

	// Plugin map
	pluginMap := map[string]hashicorpPlugin.Plugin{
		"flexcore": &PostgresProcessorPlugin{},
	}

	// Serve the plugin
	hashicorpPlugin.Serve(&hashicorpPlugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
