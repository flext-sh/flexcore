// Package plugin provides shared data processor implementation
// DRY PRINCIPLE: Eliminates 60-line duplication (mass=340) between infrastructure and plugins data-processor
package plugin

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings" 
	"time"
)

// BaseDataProcessor provides shared implementation for data processing plugins
// SOLID SRP: Single responsibility for data processing operations
type BaseDataProcessor struct {
	config     map[string]interface{}
	statistics *ProcessingStats
}

// NewBaseDataProcessor creates a new base data processor instance
// DRY PRINCIPLE: Eliminates duplicate constructor logic
func NewBaseDataProcessor() *BaseDataProcessor {
	return &BaseDataProcessor{
		statistics: &ProcessingStats{
			StartTime: time.Now(),
		},
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

	// Validate required configuration
	if mode, ok := config["mode"].(string); ok {
		log.Printf("[DataProcessor] Running in %s mode", mode)
	}

	return nil
}

// Execute processes data according to configuration - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 60-line duplication (mass=340) between two main.go files
func (dp *BaseDataProcessor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	startTime := time.Now()
	defer func() {
		dp.statistics.DurationMs = time.Since(startTime).Milliseconds()
		dp.statistics.EndTime = time.Now()
	}()

	log.Printf("[DataProcessor] Processing input with %d keys", len(input))

	result := make(map[string]interface{})
	result["processor"] = "data-processor"
	result["timestamp"] = time.Now().Unix()
	result["node_info"] = map[string]interface{}{
		"hostname": getHostname(),
		"pid":      os.Getpid(),
	}

	// Process different data types
	if data, ok := input["data"]; ok {
		switch v := data.(type) {
		case []interface{}:
			processed, stats := dp.processArray(v)
			result["data"] = processed
			result["processing_stats"] = stats

		case map[string]interface{}:
			processed := dp.processMap(v)
			result["data"] = processed

		case string:
			processed := dp.processString(v)
			result["data"] = processed

		default:
			result["data"] = data
		}
	}

	// Apply transformations based on config
	if transform, ok := dp.config["transform"].(string); ok {
		result = dp.applyTransformation(result, transform)
	}

	// Update statistics
	dp.statistics.TotalRecords++
	dp.statistics.ProcessedOK++

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
	return PluginInfo{
		ID:          "data-processor",
		Name:        "data-processor",
		Version:     "0.9.0",
		Description: "Real data processing plugin with filtering, enrichment, and transformation",
		Type:        "processor",
		Author:      "FLEXT Team",
		Tags:        []string{"data", "processing", "transformation", "enrichment"},
		Config: map[string]string{
			"mode":           "production",
			"batch_size":     "1000",
			"timeout":        "30s",
			"enable_cache":   "true",
			"max_memory":     "512MB",
		},
		Status:   "active",
		LoadedAt: dp.statistics.StartTime,
		Health:   "healthy",
	}
}

// GetStats returns processing statistics
func (dp *BaseDataProcessor) GetStats() ProcessingStats {
	if dp.statistics == nil {
		return ProcessingStats{}
	}
	return *dp.statistics
}

// GetConfig returns the current configuration for testing purposes
func (dp *BaseDataProcessor) GetConfig() map[string]interface{} {
	return dp.config
}

// GetStatistics returns the current statistics pointer for testing purposes
func (dp *BaseDataProcessor) GetStatistics() *ProcessingStats {
	return dp.statistics
}

// Cleanup saves processing statistics to file if configured and performs cleanup
// DRY PRINCIPLE: Eliminates 35-line duplication (mass=178) between all processor implementations
func (dp *BaseDataProcessor) Cleanup() error {
	log.Printf("[DataProcessor] Starting cleanup...")
	
	// Calculate final statistics
	if dp.statistics != nil && !dp.statistics.EndTime.IsZero() && dp.statistics.TotalRecords > 0 {
		duration := dp.statistics.EndTime.Sub(dp.statistics.StartTime)
		if duration > 0 {
			dp.statistics.RecordsPerSec = float64(dp.statistics.TotalRecords) / duration.Seconds()
		}
		if dp.statistics.TotalRecords > 0 {
			dp.statistics.ErrorRate = float64(dp.statistics.ProcessedError) / float64(dp.statistics.TotalRecords)
		}
	}
	
	// Save statistics to file if configured
	if dp.config != nil {
		if statsFile, ok := dp.config["stats_file"].(string); ok && statsFile != "" {
			if err := dp.saveStatsToFile(statsFile); err != nil {
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

// saveStatsToFile saves processing statistics to file
func (dp *BaseDataProcessor) saveStatsToFile(filename string) error {
	if dp.statistics == nil {
		return &HealthCheckError{Message: "no statistics to save"}
	}
	
	// Create stats data for JSON marshaling
	statsData, err := json.Marshal(dp.statistics)
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, statsData, 0644)
}

// ProcessArray processes array data - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 49-line duplication (mass=207) between two main.go files
func (dp *BaseDataProcessor) processArray(data []interface{}) ([]interface{}, map[string]interface{}) {
	processed := make([]interface{}, 0, len(data))
	filtered := 0
	enriched := 0

	for _, item := range data {
		// Validate and filter item
		if mapItem, ok := item.(map[string]interface{}); ok {
			if dp.validateRecord(mapItem) {
				// Enrich the item
				enrichedItem := dp.enrichRecord(mapItem)
				processed = append(processed, enrichedItem)
				enriched++
			} else {
				filtered++
			}
		} else {
			// Non-map items pass through
			processed = append(processed, item)
		}
	}

	stats := map[string]interface{}{
		"input_count":    len(data),
		"output_count":   len(processed),
		"filtered_count": filtered,
		"enriched_count": enriched,
		"success_rate":   float64(len(processed)) / float64(len(data)),
	}

	return processed, stats
}

// ProcessMap processes map data - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 25-line duplication (mass=108) between two main.go files
func (dp *BaseDataProcessor) processMap(data map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for key, value := range data {
		// Skip private fields
		if strings.HasPrefix(key, "_") {
			continue
		}

		// Apply field transformations
		if stringValue, ok := value.(string); ok {
			processed[key] = dp.processString(stringValue)
		} else if arrayValue, ok := value.([]interface{}); ok {
			processedArray, _ := dp.processArray(arrayValue)
			processed[key] = processedArray
		} else {
			processed[key] = value
		}
	}

	return processed
}

// ProcessString processes string data - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 25-line duplication (mass=159) between two main.go files
func (dp *BaseDataProcessor) processString(data string) string {
	result := data

	// Apply string transformations
	if transforms, ok := dp.config["transform"].(string); ok {
		for _, transform := range strings.Split(transforms, ",") {
			switch strings.TrimSpace(transform) {
			case "uppercase":
				result = strings.ToUpper(result)
			case "lowercase":
				result = strings.ToLower(result)
			case "trim":
				result = strings.TrimSpace(result)
			case "reverse":
				runes := []rune(result)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				result = string(runes)
			}
		}
	}

	return result
}

// ValidateRecord validates a record - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 23-line duplication (mass=104) between two main.go files
func (dp *BaseDataProcessor) validateRecord(record map[string]interface{}) bool {
	// Check required fields based on configuration
	if requiredFields, ok := dp.config["required_fields"].([]string); ok {
		for _, field := range requiredFields {
			if _, exists := record[field]; !exists {
				return false
			}
		}
	}

	// Check field constraints
	if constraints, ok := dp.config["field_constraints"].(map[string]interface{}); ok {
		for field, constraint := range constraints {
			if value, exists := record[field]; exists {
				if !dp.validateConstraint(value, constraint) {
					return false
				}
			}
		}
	}

	return true
}

// validateConstraint validates a field constraint
func (dp *BaseDataProcessor) validateConstraint(value, constraint interface{}) bool {
	// Simplified validation - can be extended
	if constraintMap, ok := constraint.(map[string]interface{}); ok {
		if minVal, exists := constraintMap["min"]; exists {
			if numVal, ok := value.(float64); ok {
				if minFloat, ok := minVal.(float64); ok && numVal < minFloat {
					return false
				}
			}
		}
	}
	return true
}

// enrichRecord enriches a record with additional data
func (dp *BaseDataProcessor) enrichRecord(record map[string]interface{}) map[string]interface{} {
	enriched := make(map[string]interface{})
	
	// Copy original fields
	for k, v := range record {
		enriched[k] = v
	}

	// Add enrichment fields
	enriched["_processed_at"] = time.Now().Unix()
	enriched["_processor"] = "data-processor"
	
	// Add computed fields based on config
	if enrichments, ok := dp.config["enrichments"].(map[string]interface{}); ok {
		for field, computation := range enrichments {
			enriched[field] = dp.computeField(record, computation)
		}
	}

	return enriched
}

// computeField computes a field value based on configuration
func (dp *BaseDataProcessor) computeField(record map[string]interface{}, computation interface{}) interface{} {
	// Simplified computation - can be extended
	if computeStr, ok := computation.(string); ok {
		switch computeStr {
		case "record_size":
			return len(record)
		case "current_timestamp":
			return time.Now().Unix()
		default:
			return computation
		}
	}
	return computation
}

// applyTransformation applies configured transformations
func (dp *BaseDataProcessor) applyTransformation(result map[string]interface{}, transform string) map[string]interface{} {
	switch transform {
	case "nest":
		return nestMap(result)
	case "flatten":
		return flattenMap(result)
	default:
		return result
	}
}

// Utility functions

// getHostname returns the hostname, fallback to localhost
func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "localhost"
}

// nestMap creates nested structure - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 23-line duplication (mass=117) between two main.go files
func nestMap(data map[string]interface{}) map[string]interface{} {
	nested := make(map[string]interface{})
	
	for key, value := range data {
		parts := strings.Split(key, ".")
		current := nested
		
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = value
			} else {
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				if nextMap, ok := current[part].(map[string]interface{}); ok {
					current = nextMap
				}
			}
		}
	}
	
	return nested
}

// flattenMap flattens nested structure
func flattenMap(data map[string]interface{}) map[string]interface{} {
	flattened := make(map[string]interface{})
	flattenMapRecursive(data, "", flattened)
	return flattened
}

// flattenMapRecursive recursively flattens map
func flattenMapRecursive(data map[string]interface{}, prefix string, result map[string]interface{}) {
	for key, value := range data {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}
		
		if mapValue, ok := value.(map[string]interface{}); ok {
			flattenMapRecursive(mapValue, newKey, result)
		} else {
			result[newKey] = value
		}
	}
}

// HealthCheckError represents a health check error
type HealthCheckError struct {
	Message string
}

func (e *HealthCheckError) Error() string {
	return e.Message
}