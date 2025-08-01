// Package plugin provides shared PostgreSQL processor implementation
// DRY PRINCIPLE: Eliminates 39-line duplication (mass=209) between infrastructure and plugins postgres-processor
package plugin

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// BasePostgresProcessor provides shared implementation for PostgreSQL processing plugins
// SOLID SRP: Single responsibility for PostgreSQL operations
type BasePostgresProcessor struct {
	config map[string]interface{}
	db     *sql.DB
	stats  *ProcessingStats
}

// NewBasePostgresProcessor creates a new base PostgreSQL processor instance
// DRY PRINCIPLE: Eliminates duplicate constructor logic
func NewBasePostgresProcessor() *BasePostgresProcessor {
	return &BasePostgresProcessor{
		stats: &ProcessingStats{},
	}
}

// Initialize initializes the PostgreSQL processor with configuration
func (bpp *BasePostgresProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	bpp.config = config
	bpp.stats.StartTime = time.Now()
	return nil
}

// InitializeConfig initializes with just config (backward compatibility)
func (bpp *BasePostgresProcessor) InitializeConfig(config map[string]interface{}) error {
	bpp.config = config
	return nil
}

// SetDB sets the database connection
func (bpp *BasePostgresProcessor) SetDB(db *sql.DB) {
	bpp.db = db
}

// GetConfig returns the current configuration for testing purposes
func (bpp *BasePostgresProcessor) GetConfig() map[string]interface{} {
	return bpp.config
}

// GetStatistics returns the current statistics pointer
func (bpp *BasePostgresProcessor) GetStatistics() *ProcessingStats {
	return bpp.stats
}

// GetDB returns the database connection for testing purposes
func (bpp *BasePostgresProcessor) GetDB() *sql.DB {
	return bpp.db
}

// ExecuteQuery executes a PostgreSQL query - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 39-line duplication (mass=209) between two main.go files
func (bpp *BasePostgresProcessor) ExecuteQuery(ctx context.Context, query string) ([]map[string]interface{}, error) {
	rows, err := bpp.db.QueryContext(ctx, query)
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

// ProcessArray processes array data - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates array processing duplication
func (bpp *BasePostgresProcessor) ProcessArray(data []interface{}) []interface{} {
	processed := make([]interface{}, 0, len(data))

	for _, item := range data {
		// Add metadata
		if itemMap, ok := item.(map[string]interface{}); ok {
			itemMap["_postgres_processed"] = true
			itemMap["_processed_by"] = "postgres-processor"
			processed = append(processed, itemMap)
		} else {
			// Wrap non-map items
			wrapped := map[string]interface{}{
				"value":              item,
				"_postgres_processed": true,
				"_processed_by":      "postgres-processor",
			}
			processed = append(processed, wrapped)
		}
	}

	return processed
}

// ValidateConnection validates the database connection
func (bpp *BasePostgresProcessor) ValidateConnection(ctx context.Context) error {
	if bpp.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	return bpp.db.PingContext(ctx)
}

// Close closes the database connection if owned
func (bpp *BasePostgresProcessor) Close() error {
	if bpp.db != nil {
		return bpp.db.Close()
	}
	return nil
}

// ExecuteWithStats processes data with PostgreSQL - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates Execute function duplication between infrastructure and plugins
func (bpp *BasePostgresProcessor) ExecuteWithStats(ctx context.Context, input map[string]interface{}, processorType string) (map[string]interface{}, error) {
	startTime := time.Now()
	defer func() {
		bpp.stats.DurationMs = time.Since(startTime).Milliseconds()
		bpp.stats.EndTime = time.Now()
	}()

	result := make(map[string]interface{})
	result["processor"] = processorType
	result["timestamp"] = time.Now().Unix()
	result["processed_by"] = "FlexCore PostgresProcessor v1.0"

	// Process the input data
	if query, ok := input["sql_query"].(string); ok && bpp.db != nil {
		// Execute SQL query
		queryResult, err := bpp.ExecuteQuery(ctx, query)
		if err != nil {
			result["error"] = err.Error()
			result["data"] = input
		} else {
			result["data"] = queryResult
			bpp.stats.ProcessedOK = int64(len(queryResult))
		}
	} else if data, ok := input["data"]; ok {
		// Process data without database
		switch v := data.(type) {
		case []interface{}:
			processed := bpp.ProcessArray(v)
			result["data"] = processed
			result["records_count"] = len(processed)
		case map[string]interface{}:
			processed := bpp.processMap(v)
			result["data"] = processed
			result["records_count"] = 1
		case string:
			processed := bpp.processString(v)
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
		"total_records":      bpp.stats.TotalRecords,
		"processed_ok":       bpp.stats.ProcessedOK,
		"processed_error":    bpp.stats.ProcessedError,
		"duration_ms":        bpp.stats.DurationMs,
		"records_per_sec":    bpp.stats.RecordsPerSec,
		"database_connected": bpp.db != nil,
	}

	bpp.stats.TotalRecords++
	bpp.stats.ProcessedOK++
	return result, nil
}

// processMap processes map data - SHARED IMPLEMENTATION
func (bpp *BasePostgresProcessor) processMap(data map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for key, value := range data {
		processed[key] = value
	}

	// Add metadata
	processed["_postgres_processed"] = true
	processed["_processed_by"] = "postgres-processor"

	return processed
}

// processString processes string data - SHARED IMPLEMENTATION
func (bpp *BasePostgresProcessor) processString(data string) string {
	return "[POSTGRES_PROCESSED] " + data
}