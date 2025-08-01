// Package plugin provides shared data processor implementation
// DRY PRINCIPLE: Eliminates 60-line duplication (mass=340) between infrastructure and plugins data-processor
// REFACTORED: Reduced complexity by extracting specialized modules for statistics, transformations, and utilities
package plugin

import (
	"context"
	"log"
	"time"
)

// BaseDataProcessor provides shared implementation for data processing plugins
// SOLID SRP: Single responsibility for data processing coordination using composed services
type BaseDataProcessor struct {
	config            map[string]interface{}
	statistics        *ProcessingStats
	statisticsManager *StatisticsManager
	transformationSvc *DataTransformationService
	utilities         *PluginUtilities
}

// NewBaseDataProcessor creates a new base data processor instance
// DRY PRINCIPLE: Eliminates duplicate constructor logic
func NewBaseDataProcessor() *BaseDataProcessor {
	return &BaseDataProcessor{
		statistics: &ProcessingStats{
			StartTime: time.Now(),
		},
		statisticsManager: NewStatisticsManager(),
		transformationSvc: NewDataTransformationService(make(map[string]interface{})),
		utilities:         NewPluginUtilities(),
	}
}

// Initialize the plugin with configuration - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 21-line duplication (mass=96) between two main.go files
func (dp *BaseDataProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Printf("[DataProcessor] Initializing with config: %+v", config)

	dp.config = config
	dp.statistics = &ProcessingStats{
		StartTime: time.Now(),
	}
	// Update transformation service with new config
	dp.transformationSvc = NewDataTransformationService(config)

	// Validate required configuration
	if mode, ok := config["mode"].(string); ok {
		log.Printf("[DataProcessor] Running in %s mode", mode)
	}

	return nil
}

// createBaseResult creates the base result structure - SRP principle
func (dp *BaseDataProcessor) createBaseResult() map[string]interface{} {
	return dp.utilities.CreateBaseResult("data-processor")
}

// processInputData processes input data based on type - Strategy pattern
func (dp *BaseDataProcessor) processInputData(input map[string]interface{}, result map[string]interface{}) {
	data, ok := input["data"]
	if !ok {
		return
	}

	switch v := data.(type) {
	case []interface{}:
		processed, stats := dp.transformationSvc.ProcessArray(v)
		result["data"] = processed
		result["processing_stats"] = stats
	case map[string]interface{}:
		result["data"] = dp.transformationSvc.ProcessMap(v)
	case string:
		result["data"] = dp.transformationSvc.ProcessString(v)
	default:
		result["data"] = data
	}
}

// updateStatistics updates processing statistics - SRP principle
func (dp *BaseDataProcessor) updateStatistics(startTime time.Time) {
	dp.statisticsManager.UpdateStatistics(startTime)
}

// Execute processes data according to configuration - SHARED IMPLEMENTATION with reduced complexity
// DRY PRINCIPLE: Eliminates 60-line duplication + separated concerns reduce complexity
func (dp *BaseDataProcessor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	startTime := time.Now()
	defer func() {
		dp.updateStatistics(startTime)
	}()

	log.Printf("[DataProcessor] Processing input with %d keys", len(input))

	// Create base result structure
	result := dp.createBaseResult()

	// Process input data based on type
	dp.processInputData(input, result)

	// Apply transformations based on config
	if transform, ok := dp.config["transform"].(string); ok {
		result = dp.transformationSvc.ApplyTransformation(result, transform)
	}

	return result, nil
}

// Process is an alias for Execute to maintain interface compatibility
// DRY PRINCIPLE: Single entry point for processing operations
func (dp *BaseDataProcessor) Process(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	return dp.Execute(ctx, input)
}

// HealthCheck performs health check - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 18-line duplication (mass=80) between two main.go files
func (dp *BaseDataProcessor) HealthCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Printf("[DataProcessor] Health check OK")

	// Check if processor is configured
	if dp.config == nil {
		return &HealthCheckError{Message: "processor not initialized"}
	}

	// Check processing statistics
	if dp.statistics == nil {
		return &HealthCheckError{Message: "statistics not available"}
	}
	
	return nil
}

// GetInfo returns plugin information - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 20-line duplication (mass=111) between two main.go files
func (dp *BaseDataProcessor) GetInfo() PluginInfo {
	return dp.utilities.CreatePluginInfo(
		"data-processor",
		"data-processor", 
		"0.9.0",
		"Real data processing plugin with filtering, enrichment, and transformation",
		dp.statistics.StartTime,
	)
}

// GetStats returns processing statistics
func (dp *BaseDataProcessor) GetStats() ProcessingStats {
	return dp.statisticsManager.GetStats()
}

// GetConfig returns the current configuration for testing purposes
func (dp *BaseDataProcessor) GetConfig() map[string]interface{} {
	return dp.config
}

// GetStatistics returns the current statistics pointer for testing purposes
func (dp *BaseDataProcessor) GetStatistics() *ProcessingStats {
	return dp.statisticsManager.GetStatistics()
}

// Cleanup saves processing statistics to file if configured and performs cleanup
// DRY PRINCIPLE: Eliminates 35-line duplication (mass=178) between all processor implementations
func (dp *BaseDataProcessor) Cleanup() error {
	log.Printf("[DataProcessor] Starting cleanup...")
	
	// Calculate final statistics
	dp.statisticsManager.CalculateFinalStatistics()
	
	// Save statistics to file if configured
	if dp.config != nil {
		if statsFile, ok := dp.config["stats_file"].(string); ok && statsFile != "" {
			if err := dp.statisticsManager.SaveStatsToFile(statsFile); err != nil {
				log.Printf("[DataProcessor] Warning: Failed to save stats to file: %v", err)
				// Don't return error - cleanup should continue
			}
		}
	}
	
	log.Printf("[DataProcessor] Cleanup completed")
	return nil
}

// Shutdown gracefully shuts down the processor
func (dp *BaseDataProcessor) Shutdown() error {
	log.Printf("[DataProcessor] Shutting down")
	return nil
}




