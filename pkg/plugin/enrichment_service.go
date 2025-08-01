// Package plugin provides enrichment functionality
// SOLID SRP: Dedicated module for data enrichment operations
package plugin

import (
	"time"
)

// EnrichmentService handles all data enrichment operations
// SOLID SRP: Single responsibility for data enrichment
type EnrichmentService struct {
	config map[string]interface{}
}

// NewEnrichmentService creates a new enrichment service
func NewEnrichmentService(config map[string]interface{}) *EnrichmentService {
	return &EnrichmentService{
		config: config,
	}
}

// EnrichRecord enriches a record with additional computed data
// SOLID OCP: Open for extension via configuration
func (es *EnrichmentService) EnrichRecord(record map[string]interface{}) map[string]interface{} {
	enriched := es.copyOriginalFields(record)
	es.addStandardEnrichmentFields(enriched)
	es.addConfiguredComputedFields(enriched, record)
	return enriched
}

// copyOriginalFields creates a copy of the original record
func (es *EnrichmentService) copyOriginalFields(record map[string]interface{}) map[string]interface{} {
	enriched := make(map[string]interface{})
	for k, v := range record {
		enriched[k] = v
	}
	return enriched
}

// addStandardEnrichmentFields adds standard enrichment fields
func (es *EnrichmentService) addStandardEnrichmentFields(enriched map[string]interface{}) {
	enriched["_processed_at"] = time.Now().Unix()
	enriched["_processor"] = "data-processor"
}

// addConfiguredComputedFields adds computed fields based on configuration
func (es *EnrichmentService) addConfiguredComputedFields(enriched, original map[string]interface{}) {
	enrichments, ok := es.config["enrichments"].(map[string]interface{})
	if !ok {
		return
	}

	for field, computation := range enrichments {
		enriched[field] = es.computeField(original, computation)
	}
}

// computeField computes a field value based on configuration
func (es *EnrichmentService) computeField(record map[string]interface{}, computation interface{}) interface{} {
	computeStr, ok := computation.(string)
	if !ok {
		return computation
	}

	computers := es.getFieldComputers()
	if computer, exists := computers[computeStr]; exists {
		return computer(record)
	}
	
	return computation
}

// getFieldComputers returns map of available field computations - Strategy pattern
// SOLID OCP: Open for extension via map configuration
func (es *EnrichmentService) getFieldComputers() map[string]FieldComputer {
	return map[string]FieldComputer{
		"record_size": func(record map[string]interface{}) interface{} {
			return len(record)
		},
		"current_timestamp": func(_ map[string]interface{}) interface{} {
			return time.Now().Unix()
		},
		"record_keys": func(record map[string]interface{}) interface{} {
			keys := make([]string, 0, len(record))
			for k := range record {
				keys = append(keys, k)
			}
			return keys
		},
		"record_hash": func(record map[string]interface{}) interface{} {
			// Simple hash based on field count and first key
			hash := len(record)
			for k := range record {
				hash ^= len(k)
				break // Use first key only for performance
			}
			return hash
		},
	}
}