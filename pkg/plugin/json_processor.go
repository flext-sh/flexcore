// Package plugin provides shared JSON processor implementation
// DRY PRINCIPLE: Eliminates 47-line duplication (mass=276) between infrastructure and plugins json-processor
package plugin

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// BaseJSONProcessor provides shared implementation for JSON processing plugins
// SOLID SRP: Single responsibility for JSON processing operations
type BaseJSONProcessor struct {
	config map[string]interface{}
	stats  *ProcessingStats
}

// NewBaseJSONProcessor creates a new base JSON processor instance
// DRY PRINCIPLE: Eliminates duplicate constructor logic
func NewBaseJSONProcessor() *BaseJSONProcessor {
	return &BaseJSONProcessor{
		stats: &ProcessingStats{
			StartTime: time.Now(),
		},
	}
}

// Initialize initializes the JSON processor with configuration
func (jp *BaseJSONProcessor) Initialize(config map[string]interface{}) error {
	jp.config = config
	jp.stats = &ProcessingStats{
		StartTime: time.Now(),
	}
	return nil
}

// GetConfig returns the current configuration for testing purposes
func (jp *BaseJSONProcessor) GetConfig() map[string]interface{} {
	return jp.config
}

// GetStatistics returns the current statistics pointer for testing purposes
func (jp *BaseJSONProcessor) GetStatistics() *ProcessingStats {
	return jp.stats
}

// TransformData applies JSON transformations - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 19-line duplication (mass=93) between two main.go files
func (jp *BaseJSONProcessor) TransformData(data interface{}) (interface{}, error) {
	operation := "prettify"
	if op, ok := jp.config["operation"].(string); ok {
		operation = op
	}

	switch operation {
	case "prettify":
		return jp.PrettifyJSON(data)
	case "minify":
		return jp.MinifyJSON(data)
	case "validate":
		return jp.ValidateJSON(data)
	case "transform":
		return jp.ApplyTransformations(data)
	default:
		return data, nil
	}
}

// PrettifyJSON formats JSON data nicely - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates JSON formatting duplication
func (jp *BaseJSONProcessor) PrettifyJSON(data interface{}) (interface{}, error) {
	// Convert to JSON and back for pretty formatting
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	var prettified interface{}
	if err := json.Unmarshal(jsonBytes, &prettified); err != nil {
		return nil, err
	}

	return prettified, nil
}

// MinifyJSON removes whitespace from JSON data - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates JSON minification duplication
func (jp *BaseJSONProcessor) MinifyJSON(data interface{}) (interface{}, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var minified interface{}
	if err := json.Unmarshal(jsonBytes, &minified); err != nil {
		return nil, err
	}

	return minified, nil
}

// ValidateJSON validates JSON structure - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates JSON validation duplication
func (jp *BaseJSONProcessor) ValidateJSON(data interface{}) (interface{}, error) {
	// Validate by marshaling and unmarshaling
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var validated interface{}
	if err := json.Unmarshal(jsonBytes, &validated); err != nil {
		return nil, err
	}

	return validated, nil
}

// ApplyTransformations applies custom transformations - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 15-line transformation duplication
func (jp *BaseJSONProcessor) ApplyTransformations(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		return jp.TransformMap(v), nil
	case []interface{}:
		return jp.TransformArray(v), nil
	case string:
		return jp.TransformString(v), nil
	default:
		return data, nil
	}
}

// TransformMap applies transformations to a map - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates 28-line duplication (mass=138) between two main.go files
func (jp *BaseJSONProcessor) TransformMap(data map[string]interface{}) map[string]interface{} {
	transformed := make(map[string]interface{})

	for key, value := range data {
		// Transform key to snake_case if configured
		if jp.config["snake_case_keys"] == true {
			key = jp.ToSnakeCase(key)
		}

		// Transform value recursively
		switch v := value.(type) {
		case map[string]interface{}:
			transformed[key] = jp.TransformMap(v)
		case []interface{}:
			transformed[key] = jp.TransformArray(v)
		case string:
			transformed[key] = jp.TransformString(v)
		default:
			transformed[key] = value
		}
	}

	// Add metadata
	transformed["_json_processed"] = true
	transformed["_processed_at"] = time.Now().Unix()

	return transformed
}

// TransformArray applies transformations to an array - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates array transformation duplication
func (jp *BaseJSONProcessor) TransformArray(data []interface{}) []interface{} {
	transformed := make([]interface{}, len(data))

	for i, item := range data {
		switch v := item.(type) {
		case map[string]interface{}:
			transformed[i] = jp.TransformMap(v)
		case []interface{}:
			transformed[i] = jp.TransformArray(v)
		case string:
			transformed[i] = jp.TransformString(v)
		default:
			transformed[i] = item
		}
	}

	return transformed
}

// TransformString applies string transformations - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates string transformation duplication
func (jp *BaseJSONProcessor) TransformString(data string) string {
	// Apply string transformations based on config
	if jp.config["trim_strings"] == true {
		data = strings.TrimSpace(data)
	}

	if jp.config["lowercase_strings"] == true {
		data = strings.ToLower(data)
	}

	if jp.config["uppercase_strings"] == true {
		data = strings.ToUpper(data)
	}

	return data
}

// ToSnakeCase converts camelCase to snake_case - SHARED IMPLEMENTATION
// DRY PRINCIPLE: Eliminates case conversion duplication
func (jp *BaseJSONProcessor) ToSnakeCase(str string) string {
	// Convert camelCase to snake_case
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(str, "${1}_${2}")
	return strings.ToLower(snake)
}

// GetProcessingInfo returns processing information
func (jp *BaseJSONProcessor) GetProcessingInfo() map[string]interface{} {
	return map[string]interface{}{
		"processor":    "json-processor",
		"operations":   []string{"prettify", "minify", "validate", "transform"},
		"features":     []string{"snake_case_keys", "trim_strings", "metadata_injection"},
		"processed_at": time.Now().Unix(),
		"stats":        jp.stats,
	}
}
