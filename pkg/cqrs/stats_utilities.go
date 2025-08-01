// Package cqrs provides CQRS statistics utilities
// SOLID SRP: Eliminates many returns in GetCQRSStats by creating specialized statistics collectors
package cqrs

import (
	"database/sql"
)

const (
	// StatusSuccess represents successful command status
	StatusSuccess = "success"
	// StatusFailed represents failed command status
	StatusFailed = "failed"
	// PercentageMultiplier for percentage calculations
	PercentageMultiplier = 100.0
)

// CommandStatistics represents command execution statistics
type CommandStatistics struct {
	Total       int     `json:"total"`
	Successful  int     `json:"successful"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// QueryStatistics represents query execution statistics
type QueryStatistics struct {
	Total int `json:"total"`
}

// HandlerStatistics represents handler registration statistics
type HandlerStatistics struct {
	CommandHandlers int `json:"command_handlers"`
	QueryHandlers   int `json:"query_handlers"`
}

// StatsCollector provides specialized statistics collection methods
// SOLID SRP: Single responsibility for collecting specific statistics types
type StatsCollector struct {
	writeDB *sql.DB
	readDB  *sql.DB
}

// NewStatsCollector creates a new statistics collector
func NewStatsCollector(writeDB, readDB *sql.DB) *StatsCollector {
	return &StatsCollector{
		writeDB: writeDB,
		readDB:  readDB,
	}
}

// CollectCommandStatistics collects command execution statistics
// SOLID SRP: Eliminates 3 error returns from main GetCQRSStats function
func (sc *StatsCollector) CollectCommandStatistics() (*CommandStatistics, error) {
	stats := &CommandStatistics{}

	// Collect all command statistics in a single transaction-like approach
	rows, err := sc.writeDB.Query(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = ? THEN 1 END) as successful,
			COUNT(CASE WHEN status = ? THEN 1 END) as failed
		FROM commands
	`, StatusSuccess, StatusFailed)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&stats.Total, &stats.Successful, &stats.Failed)
		if err != nil {
			return nil, err
		}
	}

	// Calculate success rate
	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Successful) / float64(stats.Total) * PercentageMultiplier
	}

	return stats, nil
}

// CollectQueryStatistics collects query execution statistics
// SOLID SRP: Eliminates 1 error return from main GetCQRSStats function
func (sc *StatsCollector) CollectQueryStatistics() (*QueryStatistics, error) {
	stats := &QueryStatistics{}

	err := sc.readDB.QueryRow("SELECT COUNT(*) FROM query_results").Scan(&stats.Total)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// CollectHandlerStatistics collects handler registration statistics
// SOLID SRP: Thread-safe handler counting with no error returns
func (sc *StatsCollector) CollectHandlerStatistics(commandHandlersCount, queryHandlersCount int) *HandlerStatistics {
	return &HandlerStatistics{
		CommandHandlers: commandHandlersCount,
		QueryHandlers:   queryHandlersCount,
	}
}

// CQRSStatsResult represents the complete CQRS statistics
type CQRSStatsResult struct {
	Commands map[string]interface{} `json:"commands"`
	Queries  map[string]interface{} `json:"queries"`
	Handlers map[string]interface{} `json:"handlers"`
}

// CollectAllStatistics collects all CQRS statistics with reduced error returns
// SOLID SRP: Coordinates all statistics collection with centralized error handling
func (sc *StatsCollector) CollectAllStatistics(commandHandlersCount, queryHandlersCount int) (*CQRSStatsResult, error) {
	// Collect command statistics
	commandStats, err := sc.CollectCommandStatistics()
	if err != nil {
		return nil, err
	}

	// Collect query statistics
	queryStats, err := sc.CollectQueryStatistics()
	if err != nil {
		return nil, err
	}

	// Collect handler statistics (no error possible)
	handlerStats := sc.CollectHandlerStatistics(commandHandlersCount, queryHandlersCount)

	return &CQRSStatsResult{
		Commands: map[string]interface{}{
			"total":        commandStats.Total,
			"successful":   commandStats.Successful,
			"failed":       commandStats.Failed,
			"success_rate": commandStats.SuccessRate,
		},
		Queries: map[string]interface{}{
			"total": queryStats.Total,
		},
		Handlers: map[string]interface{}{
			"command_handlers": handlerStats.CommandHandlers,
			"query_handlers":   handlerStats.QueryHandlers,
		},
	}, nil
}