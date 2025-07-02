// Package main shows how to implement a PostgreSQL adapter using FlexCore patterns
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/flext/flexcore/pkg/adapter"
	"github.com/flext/flexcore/pkg/patterns"
	"github.com/flext/flexcore/shared/result"
	_ "github.com/lib/pq"
)

// PostgresConfig represents the adapter configuration
type PostgresConfig struct {
	ConnectionString string `mapstructure:"connection_string" validate:"required"`
	MaxConnections   int    `mapstructure:"max_connections" validate:"min=1,max=100"`
	QueryTimeout     int    `mapstructure:"query_timeout" validate:"min=1"`
}

// PostgresAdapter implements a PostgreSQL adapter
type PostgresAdapter struct {
	*adapter.BaseAdapter
	config *PostgresConfig
	db     *sql.DB
}

// NewPostgresAdapter creates a new PostgreSQL adapter using the builder pattern
func NewPostgresAdapter() (adapter.Adapter, error) {
	// Create adapter using builder pattern
	return adapter.NewAdapterBuilder("postgres-adapter", "1.0.0").
		WithExtract(extractData).
		WithLoad(loadData).
		WithHealthCheck(healthCheck).
		WithConfigure(configure).
		WithHooks(adapter.AdapterHooks{
			OnBeforeExtract: func(ctx context.Context, req adapter.ExtractRequest) error {
				log.Printf("Starting extraction from: %s", req.Source)
				return nil
			},
			OnAfterExtract: func(ctx context.Context, req adapter.ExtractRequest, resp *adapter.ExtractResponse, err error) {
				if err != nil {
					log.Printf("Extraction failed: %v", err)
				} else {
					log.Printf("Extracted %d records", len(resp.Records))
				}
			},
			OnError: func(ctx context.Context, err error) {
				log.Printf("Adapter error: %v", err)
			},
		}).
		WithMiddleware(
			adapter.LoggingMiddleware(log.Printf),
			adapter.TimeoutMiddleware(30*time.Second),
			adapter.RetryMiddleware(3, 1*time.Second),
		).
		Build()
}

// Global adapter instance (configured during Configure)
var globalAdapter *PostgresAdapter

// configure sets up the adapter configuration
func configure(rawConfig map[string]interface{}) error {
	globalAdapter = &PostgresAdapter{
		BaseAdapter: adapter.NewBaseAdapter("postgres-adapter", "1.0.0"),
	}

	// Parse configuration
	var config PostgresConfig
	if err := globalAdapter.Configure(&config, rawConfig); err != nil {
		return err
	}

	globalAdapter.config = &config

	// Connect to database
	db, err := sql.Open("postgres", config.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxConnections)
	db.SetMaxIdleConns(config.MaxConnections / 2)
	
	globalAdapter.db = db

	return nil
}

// extractData implements data extraction using Railway pattern
func extractData(ctx context.Context, req adapter.ExtractRequest) result.Result[*adapter.ExtractResponse] {
	// Use Railway pattern for clean error handling
	return patterns.Track(validateExtractRequest(req)).
		FlatMap(func(req adapter.ExtractRequest) patterns.Railway[*sql.Rows] {
			return executeQuery(ctx, req)
		}).
		FlatMap(func(rows *sql.Rows) patterns.Railway[[]adapter.Record] {
			return scanRows(rows)
		}).
		Map(func(records []adapter.Record) *adapter.ExtractResponse {
			return &adapter.ExtractResponse{
				Records:     records,
				HasMore:     len(records) == req.Limit,
				ExtractedAt: time.Now(),
			}
		}).
		Result()
}

// validateExtractRequest validates the extraction request
func validateExtractRequest(req adapter.ExtractRequest) result.Result[adapter.ExtractRequest] {
	if req.Source == "" {
		return result.FailureWithMessage[adapter.ExtractRequest]("source is required")
	}

	if req.Limit <= 0 {
		req.Limit = 1000 // Default limit
	}

	return result.Success(req)
}

// executeQuery executes the SQL query
func executeQuery(ctx context.Context, req adapter.ExtractRequest) patterns.Railway[*sql.Rows] {
	// Build query using builder pattern
	query := patterns.NewQueryBuilder(req.Source).
		Limit(req.Limit).
		Offset(req.Offset)

	// Add filters if present
	if req.Filter != nil {
		for key, value := range req.Filter {
			query = query.Where(key, "=", value)
		}
	}

	// Add time range if present
	if req.StartTime != nil {
		query = query.Where("created_at", ">=", req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("created_at", "<=", req.EndTime)
	}

	sqlQuery, args := query.Build()
	
	// Execute query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(globalAdapter.config.QueryTimeout)*time.Second)
	defer cancel()

	rows, err := globalAdapter.db.QueryContext(queryCtx, sqlQuery, args...)
	if err != nil {
		return patterns.Failure[*sql.Rows](fmt.Errorf("query failed: %w", err))
	}

	return patterns.Success(rows)
}

// scanRows scans SQL rows into records
func scanRows(rows *sql.Rows) patterns.Railway[[]adapter.Record] {
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return patterns.Failure[[]adapter.Record](fmt.Errorf("failed to get columns: %w", err))
	}

	var records []adapter.Record

	for rows.Next() {
		// Create a slice of interface{} to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return patterns.Failure[[]adapter.Record](fmt.Errorf("failed to scan row: %w", err))
		}

		// Build record
		data := make(map[string]interface{})
		for i, col := range columns {
			data[col] = values[i]
		}

		record := adapter.Record{
			ID:        fmt.Sprintf("%v", data["id"]), // Assume 'id' column exists
			Data:      data,
			Timestamp: time.Now(),
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return patterns.Failure[[]adapter.Record](fmt.Errorf("row iteration error: %w", err))
	}

	return patterns.Success(records)
}

// loadData implements data loading
func loadData(ctx context.Context, req adapter.LoadRequest) result.Result[*adapter.LoadResponse] {
	// Start transaction
	tx, err := globalAdapter.db.BeginTx(ctx, nil)
	if err != nil {
		return result.Failure[*adapter.LoadResponse](fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer tx.Rollback()

	var loaded, failed int64
	var failedRecords []adapter.FailedRecord

	// Process records in batches
	batchSize := 100
	for i := 0; i < len(req.Records); i += batchSize {
		end := i + batchSize
		if end > len(req.Records) {
			end = len(req.Records)
		}

		batch := req.Records[i:end]
		
		// Use Option type for batch processing
		batchResult := processBatch(ctx, tx, req.Target, batch, req.Mode)
		
		batchResult.ForEach(func(stats BatchStats) {
			loaded += stats.Loaded
			failed += stats.Failed
			failedRecords = append(failedRecords, stats.FailedRecords...)
		})

		if batchResult.IsNone() {
			// Batch failed completely
			failed += int64(len(batch))
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return result.Failure[*adapter.LoadResponse](fmt.Errorf("failed to commit transaction: %w", err))
	}

	return result.Success(&adapter.LoadResponse{
		RecordsLoaded: loaded,
		RecordsFailed: failed,
		FailedRecords: failedRecords,
		LoadedAt:      time.Now(),
	})
}

// BatchStats holds batch processing statistics
type BatchStats struct {
	Loaded        int64
	Failed        int64
	FailedRecords []adapter.FailedRecord
}

// processBatch processes a batch of records
func processBatch(ctx context.Context, tx *sql.Tx, table string, records []adapter.Record, mode adapter.LoadMode) patterns.Option[BatchStats] {
	// Implementation would depend on the load mode
	// This is a simplified example
	
	stats := BatchStats{}

	for _, record := range records {
		if err := insertRecord(ctx, tx, table, record); err != nil {
			stats.Failed++
			stats.FailedRecords = append(stats.FailedRecords, adapter.FailedRecord{
				Record: record,
				Error:  err.Error(),
			})
		} else {
			stats.Loaded++
		}
	}

	return patterns.Some(stats)
}

// insertRecord inserts a single record
func insertRecord(ctx context.Context, tx *sql.Tx, table string, record adapter.Record) error {
	// Build INSERT statement dynamically
	// This is simplified - real implementation would handle SQL injection prevention
	
	columns := make([]string, 0, len(record.Data))
	values := make([]interface{}, 0, len(record.Data))
	placeholders := make([]string, 0, len(record.Data))
	
	i := 1
	for col, val := range record.Data {
		columns = append(columns, col)
		values = append(values, val)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}
	
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
	
	_, err := tx.ExecContext(ctx, query, values...)
	return err
}

// healthCheck verifies the adapter is healthy
func healthCheck(ctx context.Context) error {
	if globalAdapter == nil || globalAdapter.db == nil {
		return fmt.Errorf("adapter not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return globalAdapter.db.PingContext(ctx)
}


// main function for testing
func main() {
	// Create adapter
	adapter, err := NewPostgresAdapter()
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	// Configure adapter
	config := map[string]interface{}{
		"connection_string": "postgres://user:pass@localhost/db",
		"max_connections":   10,
		"query_timeout":     30,
	}

	if err := adapter.Configure(config); err != nil {
		log.Fatalf("Failed to configure adapter: %v", err)
	}

	// Example extraction
	ctx := context.Background()
	
	extractReq := adapter.ExtractRequest{
		Source: "users",
		Filter: map[string]interface{}{
			"active": true,
		},
		Limit: 100,
	}

	resp, err := adapter.Extract(ctx, extractReq)
	if err != nil {
		log.Fatalf("Failed to extract: %v", err)
	}

	log.Printf("Extracted %d records", len(resp.Records))

	// Example load
	if len(resp.Records) > 0 {
		loadReq := adapter.LoadRequest{
			Target:  "users_backup",
			Records: resp.Records,
			Mode:    adapter.LoadModeAppend,
		}

		loadResp, err := adapter.Load(ctx, loadReq)
		if err != nil {
			log.Fatalf("Failed to load: %v", err)
		}

		log.Printf("Loaded %d records, failed %d", loadResp.RecordsLoaded, loadResp.RecordsFailed)
	}

	// Health check
	if err := adapter.HealthCheck(ctx); err != nil {
		log.Fatalf("Health check failed: %v", err)
	}

	log.Println("Adapter is healthy!")
}