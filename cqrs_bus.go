// REAL CQRS Implementation - Complete Command/Query Separation
package cqrs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Command represents a command in the CQRS pattern
type Command interface {
	CommandID() string
	CommandType() string
	AggregateID() string
	Payload() interface{}
	Metadata() map[string]interface{}
}

// Query represents a query in the CQRS pattern
type Query interface {
	QueryID() string
	QueryType() string
	Parameters() map[string]interface{}
}

// CommandHandler handles commands
type CommandHandler interface {
	Handle(ctx context.Context, command Command) error
	CanHandle(commandType string) bool
}

// QueryHandler handles queries
type QueryHandler interface {
	Handle(ctx context.Context, query Query) (interface{}, error)
	CanHandle(queryType string) bool
}

// CommandResult represents the result of command execution
type CommandResult struct {
	CommandID   string                 `json:"command_id"`
	CommandType string                 `json:"command_type"`
	Status      string                 `json:"status"` // "success", "failed"
	Error       string                 `json:"error,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	ExecutedAt  time.Time              `json:"executed_at"`
	Duration    time.Duration          `json:"duration"`
}

// QueryResult represents the result of query execution
type QueryResult struct {
	QueryID    string        `json:"query_id"`
	QueryType  string        `json:"query_type"`
	Data       interface{}   `json:"data"`
	Count      int           `json:"count"`
	ExecutedAt time.Time     `json:"executed_at"`
	Duration   time.Duration `json:"duration"`
}

// CQRSBus implements the CQRS pattern with command and query buses
type CQRSBus struct {
	// Write-side database (commands)
	writeDB *sql.DB

	// Read-side database (queries)
	readDB *sql.DB

	// Handlers
	commandHandlers map[string]CommandHandler
	queryHandlers   map[string]QueryHandler

	// Synchronization
	mu sync.RWMutex

	// Metrics
	commandMetrics map[string]*CommandMetrics
	queryMetrics   map[string]*QueryMetrics
}

type CommandMetrics struct {
	TotalExecuted   int64         `json:"total_executed"`
	TotalSucceeded  int64         `json:"total_succeeded"`
	TotalFailed     int64         `json:"total_failed"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastExecuted    time.Time     `json:"last_executed"`
}

type QueryMetrics struct {
	TotalExecuted  int64         `json:"total_executed"`
	AverageLatency time.Duration `json:"average_latency"`
	LastExecuted   time.Time     `json:"last_executed"`
}

// NewCQRSBus creates a new CQRS bus with separate read/write databases
func NewCQRSBus(writeDBPath, readDBPath string) (*CQRSBus, error) {
	// Open write database
	writeDB, err := sql.Open("sqlite3", writeDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open write database: %w", err)
	}

	// Open read database
	readDB, err := sql.Open("sqlite3", readDBPath)
	if err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("failed to open read database: %w", err)
	}

	bus := &CQRSBus{
		writeDB:         writeDB,
		readDB:          readDB,
		commandHandlers: make(map[string]CommandHandler),
		queryHandlers:   make(map[string]QueryHandler),
		commandMetrics:  make(map[string]*CommandMetrics),
		queryMetrics:    make(map[string]*QueryMetrics),
	}

	if err := bus.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Printf("CQRS Bus initialized with write DB: %s, read DB: %s", writeDBPath, readDBPath)
	return bus, nil
}

func (bus *CQRSBus) createTables() error {
	// Create write-side tables (commands)
	writeSchema := `
	CREATE TABLE IF NOT EXISTS commands (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		aggregate_id TEXT NOT NULL,
		payload TEXT NOT NULL,
		metadata TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		error TEXT,
		result TEXT,
		created_at DATETIME NOT NULL,
		executed_at DATETIME,
		duration_ms INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_commands_type ON commands(type);
	CREATE INDEX IF NOT EXISTS idx_commands_aggregate ON commands(aggregate_id);
	CREATE INDEX IF NOT EXISTS idx_commands_status ON commands(status);
	CREATE INDEX IF NOT EXISTS idx_commands_created_at ON commands(created_at);
	`

	if _, err := bus.writeDB.Exec(writeSchema); err != nil {
		return fmt.Errorf("failed to create write schema: %w", err)
	}

	// Create read-side tables (projections)
	readSchema := `
	CREATE TABLE IF NOT EXISTS projections (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		data TEXT NOT NULL,
		version INTEGER NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS query_results (
		id TEXT PRIMARY KEY,
		query_type TEXT NOT NULL,
		parameters TEXT NOT NULL,
		result TEXT NOT NULL,
		count INTEGER NOT NULL,
		executed_at DATETIME NOT NULL,
		duration_ms INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_projections_type ON projections(type);
	CREATE INDEX IF NOT EXISTS idx_query_results_type ON query_results(query_type);
	CREATE INDEX IF NOT EXISTS idx_query_results_executed_at ON query_results(executed_at);
	`

	if _, err := bus.readDB.Exec(readSchema); err != nil {
		return fmt.Errorf("failed to create read schema: %w", err)
	}

	log.Println("CQRS tables created successfully")
	return nil
}

// RegisterCommandHandler registers a command handler
func (bus *CQRSBus) RegisterCommandHandler(commandType string, handler CommandHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.commandHandlers[commandType] = handler
	bus.commandMetrics[commandType] = &CommandMetrics{}

	log.Printf("Command handler registered: %s", commandType)
}

// RegisterQueryHandler registers a query handler
func (bus *CQRSBus) RegisterQueryHandler(queryType string, handler QueryHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.queryHandlers[queryType] = handler
	bus.queryMetrics[queryType] = &QueryMetrics{}

	log.Printf("Query handler registered: %s", queryType)
}

// SendCommand sends a command through the command bus
func (bus *CQRSBus) SendCommand(ctx context.Context, command Command) (*CommandResult, error) {
	startTime := time.Now()

	result := &CommandResult{
		CommandID:   command.CommandID(),
		CommandType: command.CommandType(),
		ExecutedAt:  startTime,
	}

	// Find handler
	bus.mu.RLock()
	handler, exists := bus.commandHandlers[command.CommandType()]
	bus.mu.RUnlock()

	if !exists {
		result.Status = "failed"
		result.Error = fmt.Sprintf("no handler found for command type: %s", command.CommandType())
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf(result.Error)
	}

	// Store command in write database
	if err := bus.storeCommand(command); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to store command: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Execute command
	err := handler.Handle(ctx, command)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		bus.updateCommandStatus(command.CommandID(), "failed", err.Error(), "", result.Duration)
	} else {
		result.Status = "success"
		bus.updateCommandStatus(command.CommandID(), "success", "", "", result.Duration)
	}

	// Update metrics
	bus.updateCommandMetrics(command.CommandType(), result.Status, result.Duration)

	log.Printf("Command executed: %s (%s) - %s in %v", command.CommandType(), command.CommandID(), result.Status, result.Duration)
	return result, err
}

// SendQuery sends a query through the query bus
func (bus *CQRSBus) SendQuery(ctx context.Context, query Query) (*QueryResult, error) {
	startTime := time.Now()

	result := &QueryResult{
		QueryID:    query.QueryID(),
		QueryType:  query.QueryType(),
		ExecutedAt: startTime,
	}

	// Find handler
	bus.mu.RLock()
	handler, exists := bus.queryHandlers[query.QueryType()]
	bus.mu.RUnlock()

	if !exists {
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("no handler found for query type: %s", query.QueryType())
	}

	// Execute query
	data, err := handler.Handle(ctx, query)
	result.Duration = time.Since(startTime)

	if err != nil {
		return result, err
	}

	result.Data = data

	// Determine count
	if dataSlice, ok := data.([]interface{}); ok {
		result.Count = len(dataSlice)
	} else if dataMap, ok := data.(map[string]interface{}); ok {
		if items, ok := dataMap["items"]; ok {
			if itemsSlice, ok := items.([]interface{}); ok {
				result.Count = len(itemsSlice)
			}
		} else {
			result.Count = 1
		}
	} else {
		result.Count = 1
	}

	// Store query result
	bus.storeQueryResult(query, result)

	// Update metrics
	bus.updateQueryMetrics(query.QueryType(), result.Duration)

	log.Printf("Query executed: %s (%s) - %d results in %v", query.QueryType(), query.QueryID(), result.Count, result.Duration)
	return result, nil
}

func (bus *CQRSBus) storeCommand(command Command) error {
	payloadJSON, err := json.Marshal(command.Payload())
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(command.Metadata())
	if err != nil {
		return err
	}

	query := `
	INSERT INTO commands (id, type, aggregate_id, payload, metadata, created_at)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = bus.writeDB.Exec(query,
		command.CommandID(),
		command.CommandType(),
		command.AggregateID(),
		string(payloadJSON),
		string(metadataJSON),
		time.Now(),
	)

	return err
}

func (bus *CQRSBus) updateCommandStatus(commandID, status, error, result string, duration time.Duration) error {
	query := `
	UPDATE commands
	SET status = ?, error = ?, result = ?, executed_at = ?, duration_ms = ?
	WHERE id = ?
	`

	_, err := bus.writeDB.Exec(query, status, error, result, time.Now(), duration.Milliseconds(), commandID)
	return err
}

func (bus *CQRSBus) storeQueryResult(query Query, result *QueryResult) error {
	parametersJSON, err := json.Marshal(query.Parameters())
	if err != nil {
		return err
	}

	resultJSON, err := json.Marshal(result.Data)
	if err != nil {
		return err
	}

	querySQL := `
	INSERT INTO query_results (id, query_type, parameters, result, count, executed_at, duration_ms)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = bus.readDB.Exec(querySQL,
		query.QueryID(),
		query.QueryType(),
		string(parametersJSON),
		string(resultJSON),
		result.Count,
		result.ExecutedAt,
		result.Duration.Milliseconds(),
	)

	return err
}

func (bus *CQRSBus) updateCommandMetrics(commandType, status string, duration time.Duration) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	metrics, exists := bus.commandMetrics[commandType]
	if !exists {
		return
	}

	metrics.TotalExecuted++
	if status == "success" {
		metrics.TotalSucceeded++
	} else {
		metrics.TotalFailed++
	}

	// Update average latency (simple moving average)
	if metrics.TotalExecuted == 1 {
		metrics.AverageLatency = duration
	} else {
		metrics.AverageLatency = (metrics.AverageLatency*time.Duration(metrics.TotalExecuted-1) + duration) / time.Duration(metrics.TotalExecuted)
	}

	metrics.LastExecuted = time.Now()
}

func (bus *CQRSBus) updateQueryMetrics(queryType string, duration time.Duration) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	metrics, exists := bus.queryMetrics[queryType]
	if !exists {
		return
	}

	metrics.TotalExecuted++

	// Update average latency
	if metrics.TotalExecuted == 1 {
		metrics.AverageLatency = duration
	} else {
		metrics.AverageLatency = (metrics.AverageLatency*time.Duration(metrics.TotalExecuted-1) + duration) / time.Duration(metrics.TotalExecuted)
	}

	metrics.LastExecuted = time.Now()
}

// GetCommandMetrics returns metrics for all command types
func (bus *CQRSBus) GetCommandMetrics() map[string]*CommandMetrics {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	result := make(map[string]*CommandMetrics)
	for k, v := range bus.commandMetrics {
		result[k] = v
	}

	return result
}

// GetQueryMetrics returns metrics for all query types
func (bus *CQRSBus) GetQueryMetrics() map[string]*QueryMetrics {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	result := make(map[string]*QueryMetrics)
	for k, v := range bus.queryMetrics {
		result[k] = v
	}

	return result
}

// GetCQRSStats returns overall CQRS statistics
func (bus *CQRSBus) GetCQRSStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Command statistics from write database
	var totalCommands, successfulCommands, failedCommands int
	err := bus.writeDB.QueryRow("SELECT COUNT(*) FROM commands").Scan(&totalCommands)
	if err != nil {
		return nil, err
	}

	err = bus.writeDB.QueryRow("SELECT COUNT(*) FROM commands WHERE status = 'success'").Scan(&successfulCommands)
	if err != nil {
		return nil, err
	}

	err = bus.writeDB.QueryRow("SELECT COUNT(*) FROM commands WHERE status = 'failed'").Scan(&failedCommands)
	if err != nil {
		return nil, err
	}

	stats["commands"] = map[string]interface{}{
		"total":      totalCommands,
		"successful": successfulCommands,
		"failed":     failedCommands,
		"success_rate": func() float64 {
			if totalCommands == 0 {
				return 0
			}
			return float64(successfulCommands) / float64(totalCommands) * 100
		}(),
	}

	// Query statistics from read database
	var totalQueries int
	err = bus.readDB.QueryRow("SELECT COUNT(*) FROM query_results").Scan(&totalQueries)
	if err != nil {
		return nil, err
	}

	stats["queries"] = map[string]interface{}{
		"total": totalQueries,
	}

	// Handler counts
	bus.mu.RLock()
	stats["handlers"] = map[string]interface{}{
		"command_handlers": len(bus.commandHandlers),
		"query_handlers":   len(bus.queryHandlers),
	}
	bus.mu.RUnlock()

	return stats, nil
}

// UpdateProjection updates a read-side projection
func (bus *CQRSBus) UpdateProjection(projectionType, projectionID string, data interface{}, version int) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	query := `
	INSERT OR REPLACE INTO projections (id, type, data, version, created_at, updated_at)
	VALUES (?, ?, ?, ?, COALESCE((SELECT created_at FROM projections WHERE id = ?), ?), ?)
	`

	now := time.Now()
	_, err = bus.readDB.Exec(query, projectionID, projectionType, string(dataJSON), version, projectionID, now, now)

	if err != nil {
		return err
	}

	log.Printf("Projection updated: %s/%s (version %d)", projectionType, projectionID, version)
	return nil
}

// GetProjection retrieves a read-side projection
func (bus *CQRSBus) GetProjection(projectionID string) (map[string]interface{}, error) {
	query := `SELECT type, data, version, created_at, updated_at FROM projections WHERE id = ?`

	var projectionType, dataJSON string
	var version int
	var createdAt, updatedAt time.Time

	err := bus.readDB.QueryRow(query, projectionID).Scan(&projectionType, &dataJSON, &version, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":         projectionID,
		"type":       projectionType,
		"data":       data,
		"version":    version,
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	return result, nil
}

// Close closes the CQRS bus database connections
func (bus *CQRSBus) Close() error {
	var err1, err2 error

	if bus.writeDB != nil {
		err1 = bus.writeDB.Close()
	}

	if bus.readDB != nil {
		err2 = bus.readDB.Close()
	}

	if err1 != nil {
		return err1
	}

	return err2
}
