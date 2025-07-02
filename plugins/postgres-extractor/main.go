// Postgres Extractor Plugin - Real executable plugin using HashiCorp go-plugin
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/flext/flexcore/infrastructure/plugins"
	"github.com/flext/flexcore/shared/result"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	_ "github.com/lib/pq"
)

// PostgresExtractor implements the ExtractorPlugin interface
type PostgresExtractor struct {
	logger hclog.Logger
	db     *sql.DB
}

// Init initializes the plugin with configuration
func (p *PostgresExtractor) Init(config map[string]interface{}) error {
	p.logger.Info("Initializing PostgreSQL extractor", "config", config)

	// Get connection parameters from config
	host, _ := config["host"].(string)
	port, _ := config["port"].(string)
	database, _ := config["database"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host, port, database, username, password)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.db = db
	p.logger.Info("PostgreSQL extractor initialized successfully")
	return nil
}

// Execute executes the plugin with given input (generic interface method)
func (p *PostgresExtractor) Execute(ctx context.Context, input interface{}) result.Result[interface{}] {
	p.logger.Debug("Execute called", "input", input)
	
	// Convert input to string (SQL query)
	query, ok := input.(string)
	if !ok {
		return result.Failure[interface{}](fmt.Errorf("input must be a SQL query string"))
	}

	// Use Extract method
	extractResult := p.Extract(ctx, query)
	if extractResult.IsFailure() {
		return result.Failure[interface{}](extractResult.Error())
	}

	// Convert []interface{} to interface{}
	data := extractResult.Value()
	return result.Success[interface{}](data)
}

// GetMetadata returns plugin metadata
func (p *PostgresExtractor) GetMetadata() plugins.PluginMetadata {
	return plugins.PluginMetadata{
		Name:        "postgres-extractor",
		Version:     "1.0.0",
		Description: "PostgreSQL data extractor plugin",
		Author:      "FlexCore",
		Type:        plugins.ExtractorType,
		Capabilities: []string{
			"sql-queries",
			"batch-extraction", 
			"schema-detection",
			"incremental-sync",
		},
	}
}

// Shutdown gracefully shuts down the plugin
func (p *PostgresExtractor) Shutdown() error {
	p.logger.Info("Shutting down PostgreSQL extractor")

	if p.db != nil {
		if err := p.db.Close(); err != nil {
			p.logger.Error("Error closing database connection", "error", err)
			return err
		}
	}

	return nil
}

// GetInfo returns plugin information
func (p *PostgresExtractor) GetInfo() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "postgres-extractor",
		Version:     "1.0.0",
		Description: "PostgreSQL data extractor plugin",
		Type:        plugins.ExtractorType,
		Capabilities: []string{
			"sql-queries",
			"batch-extraction",
			"schema-detection",
			"incremental-sync",
		},
	}
}

// Configure configures the plugin with connection parameters
func (p *PostgresExtractor) Configure(config map[string]interface{}) result.Result[bool] {
	p.logger.Info("Configuring PostgreSQL extractor", "config", config)

	// Get connection parameters from config
	host, _ := config["host"].(string)
	port, _ := config["port"].(string)
	database, _ := config["database"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)
	sslmode, _ := config["sslmode"].(string)

	// Set defaults
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, username, password, database, sslmode)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		p.logger.Error("Failed to open database connection", "error", err)
		return result.Failure[bool](fmt.Errorf("failed to connect to PostgreSQL: %w", err))
	}

	// Test connection
	if err := db.Ping(); err != nil {
		p.logger.Error("Failed to ping database", "error", err)
		return result.Failure[bool](fmt.Errorf("failed to ping PostgreSQL: %w", err))
	}

	p.db = db
	p.logger.Info("Successfully connected to PostgreSQL", "host", host, "database", database)
	return result.Success(true)
}

// Extract extracts data from PostgreSQL source
func (p *PostgresExtractor) Extract(ctx context.Context, source string) result.Result[[]interface{}] {
	p.logger.Info("Starting data extraction", "source", source)

	if p.db == nil {
		return result.Failure[[]interface{}](fmt.Errorf("database not configured"))
	}

	// Parse source as SQL query or table name
	var query string
	if isValidSQL(source) {
		query = source
	} else {
		// Treat as table name
		query = fmt.Sprintf("SELECT * FROM %s", source)
	}

	p.logger.Debug("Executing query", "query", query)

	// Execute query
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		p.logger.Error("Query execution failed", "error", err, "query", query)
		return result.Failure[[]interface{}](fmt.Errorf("query failed: %w", err))
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return result.Failure[[]interface{}](fmt.Errorf("failed to get columns: %w", err))
	}

	var results []interface{}
	recordCount := 0

	// Process rows
	for rows.Next() {
		// Create slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row
		if err := rows.Scan(valuePtrs...); err != nil {
			p.logger.Error("Failed to scan row", "error", err)
			continue
		}

		// Convert to map
		record := make(map[string]interface{})
		for i, col := range columns {
			record[col] = convertValue(values[i])
		}

		results = append(results, record)
		recordCount++

		// Limit for safety
		if recordCount >= 10000 {
			p.logger.Warn("Reached record limit", "limit", 10000)
			break
		}
	}

	if err := rows.Err(); err != nil {
		return result.Failure[[]interface{}](fmt.Errorf("row iteration error: %w", err))
	}

	p.logger.Info("Extraction completed", "records", recordCount)
	return result.Success(results)
}

// GetSchema returns the schema of the specified source
func (p *PostgresExtractor) GetSchema() map[string]interface{} {
	if p.db == nil {
		return map[string]interface{}{
			"error": "database not configured",
		}
	}

	// Get table schemas
	query := `
		SELECT table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		ORDER BY table_name, ordinal_position
	`

	rows, err := p.db.Query(query)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to get schema: %v", err),
		}
	}
	defer rows.Close()

	tables := make(map[string]interface{})

	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable); err != nil {
			continue
		}

		if tables[tableName] == nil {
			tables[tableName] = map[string]interface{}{
				"columns": make(map[string]interface{}),
			}
		}

		tableInfo := tables[tableName].(map[string]interface{})
		columns := tableInfo["columns"].(map[string]interface{})
		columns[columnName] = map[string]interface{}{
			"type":     dataType,
			"nullable": isNullable == "YES",
		}
	}

	return map[string]interface{}{
		"tables": tables,
		"driver": "postgres",
	}
}

// Health checks plugin health
func (p *PostgresExtractor) Health() result.Result[bool] {
	if p.db == nil {
		return result.Failure[bool](fmt.Errorf("database not configured"))
	}

	if err := p.db.Ping(); err != nil {
		return result.Failure[bool](fmt.Errorf("database ping failed: %w", err))
	}

	return result.Success(true)
}


// Helper functions

func isValidSQL(query string) bool {
	query = strings.ToUpper(strings.TrimSpace(query))
	return strings.HasPrefix(query, "SELECT") ||
		strings.HasPrefix(query, "WITH") ||
		strings.HasPrefix(query, "SHOW")
}

func convertValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case []byte:
		// Try to parse as JSON first
		var jsonVal interface{}
		if err := json.Unmarshal(v, &jsonVal); err == nil {
			return jsonVal
		}
		// Otherwise return as string
		return string(v)
	case int64:
		return v
	case float64:
		return v
	case bool:
		return v
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Plugin handshake and serve
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "FLEXCORE_PLUGIN",
	MagicCookieValue: "postgres-extractor",
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	extractor := &PostgresExtractor{
		logger: logger,
	}

	var pluginMap = map[string]plugin.Plugin{
		"extractor": &plugins.ExtractorGRPCPlugin{Impl: extractor},
	}

	logger.Debug("Starting postgres-extractor plugin")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}