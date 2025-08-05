// Package plugin provides processing statistics functionality
// SOLID SRP: Dedicated module for statistics and metrics handling
package plugin

import (
	"encoding/json"
	"os"
	"time"
)

// StatisticsManager handles all statistics operations
// SOLID SRP: Single responsibility for statistics management
type StatisticsManager struct {
	stats *ProcessingStats
}

// NewStatisticsManager creates a new statistics manager
func NewStatisticsManager() *StatisticsManager {
	return &StatisticsManager{
		stats: &ProcessingStats{
			StartTime: time.Now(),
		},
	}
}

// UpdateStatistics updates processing statistics with timing information
func (sm *StatisticsManager) UpdateStatistics(startTime time.Time) {
	sm.stats.DurationMs = time.Since(startTime).Milliseconds()
	sm.stats.EndTime = time.Now()
	sm.stats.TotalRecords++
	sm.stats.ProcessedOK++
}

// GetStats returns a copy of current processing statistics
func (sm *StatisticsManager) GetStats() ProcessingStats {
	if sm.stats == nil {
		return ProcessingStats{}
	}
	return *sm.stats
}

// GetStatistics returns pointer to current statistics for testing
func (sm *StatisticsManager) GetStatistics() *ProcessingStats {
	return sm.stats
}

// SaveStatsToFile saves processing statistics to specified file
func (sm *StatisticsManager) SaveStatsToFile(filename string) error {
	if sm.stats == nil {
		return &HealthCheckError{Message: "no statistics to save"}
	}

	// Create stats data for JSON marshaling
	statsData, err := json.Marshal(sm.stats)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, statsData, 0o644)
}

// CalculateFinalStatistics computes final metrics like rates and percentages
func (sm *StatisticsManager) CalculateFinalStatistics() {
	if sm.stats != nil && !sm.stats.EndTime.IsZero() && sm.stats.TotalRecords > 0 {
		duration := sm.stats.EndTime.Sub(sm.stats.StartTime)
		if duration > 0 {
			sm.stats.RecordsPerSec = float64(sm.stats.TotalRecords) / duration.Seconds()
		}
		if sm.stats.TotalRecords > 0 {
			sm.stats.ErrorRate = float64(sm.stats.ProcessedError) / float64(sm.stats.TotalRecords)
		}
	}
}
