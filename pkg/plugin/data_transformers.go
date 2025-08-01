// Package plugin provides data transformation functionality
// SOLID SRP: Dedicated module for data transformation operations
package plugin

import (
	"strings"
)

// StringTransformer defines a function that transforms a string
type StringTransformer func(string) string

// MapTransformer defines a function that transforms a map
type MapTransformer func(map[string]interface{}) map[string]interface{}

// FieldComputer defines a function that computes a field value
type FieldComputer func(map[string]interface{}) interface{}

// DataTransformationService handles all data transformation operations
// SOLID SRP: Single responsibility for data transformations
type DataTransformationService struct {
	config            map[string]interface{}
	validationService *ValidationService
	enrichmentService *EnrichmentService
}

// NewDataTransformationService creates a new data transformation service
func NewDataTransformationService(config map[string]interface{}) *DataTransformationService {
	return &DataTransformationService{
		config:            config,
		validationService: NewValidationService(config),
		enrichmentService: NewEnrichmentService(config),
	}
}

// ProcessArray processes array data with filtering and enrichment
// DRY PRINCIPLE: Eliminates 49-line duplication (mass=207) between two main.go files
func (dts *DataTransformationService) ProcessArray(data []interface{}) ([]interface{}, map[string]interface{}) {
	processed := make([]interface{}, 0, len(data))
	filtered := 0
	enriched := 0

	for _, item := range data {
		// Validate and filter item
		if mapItem, ok := item.(map[string]interface{}); ok {
			if dts.validationService.ValidateRecord(mapItem) {
				// Enrich the item
				enrichedItem := dts.enrichmentService.EnrichRecord(mapItem)
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

// ProcessMap processes map data with field transformations
// DRY PRINCIPLE: Eliminates 25-line duplication (mass=108) between two main.go files
func (dts *DataTransformationService) ProcessMap(data map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for key, value := range data {
		// Skip private fields
		if strings.HasPrefix(key, "_") {
			continue
		}

		// Apply field transformations
		if stringValue, ok := value.(string); ok {
			processed[key] = dts.ProcessString(stringValue)
		} else if arrayValue, ok := value.([]interface{}); ok {
			processedArray, _ := dts.ProcessArray(arrayValue)
			processed[key] = processedArray
		} else {
			processed[key] = value
		}
	}

	return processed
}

// ProcessString processes string data with configurable transformations
// DRY PRINCIPLE: Eliminates 25-line duplication + Strategy pattern reduces complexity
func (dts *DataTransformationService) ProcessString(data string) string {
	result := data
	transforms, ok := dts.config["transform"].(string)
	if !ok || transforms == "" {
		return result
	}

	transformers := dts.getStringTransformers()
	for _, transform := range strings.Split(transforms, ",") {
		transformName := strings.TrimSpace(transform)
		if transformer, exists := transformers[transformName]; exists {
			result = transformer(result)
		}
	}

	return result
}

// getStringTransformers returns map of available string transformations - DRY pattern
// SOLID OCP: Open for extension via map configuration
func (dts *DataTransformationService) getStringTransformers() map[string]StringTransformer {
	return map[string]StringTransformer{
		"uppercase": strings.ToUpper,
		"lowercase": strings.ToLower,
		"trim":      strings.TrimSpace,
		"reverse": func(s string) string {
			runes := []rune(s)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return string(runes)
		},
	}
}

// ApplyTransformation applies configured transformations to result
func (dts *DataTransformationService) ApplyTransformation(result map[string]interface{}, transform string) map[string]interface{} {
	transformers := dts.getMapTransformers()
	if transformer, exists := transformers[transform]; exists {
		return transformer(result)
	}
	return result
}

// getMapTransformers returns map of available map transformations - Strategy pattern
// SOLID OCP: Open for extension via map configuration
func (dts *DataTransformationService) getMapTransformers() map[string]MapTransformer {
	return map[string]MapTransformer{
		"nest":    NestMap,
		"flatten": FlattenMap,
	}
}


// Utility functions

// NestMap creates nested structure from flat keys with dots
// DRY PRINCIPLE: Eliminates 23-line duplication (mass=117) between two main.go files
func NestMap(data map[string]interface{}) map[string]interface{} {
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

// FlattenMap flattens nested structure to dot-separated keys
func FlattenMap(data map[string]interface{}) map[string]interface{} {
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