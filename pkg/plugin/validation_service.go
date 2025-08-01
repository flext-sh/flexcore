// Package plugin provides validation functionality
// SOLID SRP: Dedicated module for validation operations
package plugin

// ValidationService handles all validation operations
// SOLID SRP: Single responsibility for data validation
type ValidationService struct {
	config map[string]interface{}
}

// NewValidationService creates a new validation service
func NewValidationService(config map[string]interface{}) *ValidationService {
	return &ValidationService{
		config: config,
	}
}

// ValidateRecord validates a record against configuration rules
// SOLID OCP: Open for extension via configuration
func (vs *ValidationService) ValidateRecord(record map[string]interface{}) bool {
	// Check required fields based on configuration
	if !vs.validateRequiredFields(record) {
		return false
	}

	// Check field constraints
	if !vs.validateFieldConstraints(record) {
		return false
	}

	return true
}

// validateRequiredFields checks if all required fields are present
func (vs *ValidationService) validateRequiredFields(record map[string]interface{}) bool {
	requiredFields, ok := vs.config["required_fields"].([]string)
	if !ok {
		return true // No required fields configured
	}

	for _, field := range requiredFields {
		if _, exists := record[field]; !exists {
			return false
		}
	}
	return true
}

// validateFieldConstraints validates field constraints
func (vs *ValidationService) validateFieldConstraints(record map[string]interface{}) bool {
	constraints, ok := vs.config["field_constraints"].(map[string]interface{})
	if !ok {
		return true // No constraints configured
	}

	for field, constraint := range constraints {
		if value, exists := record[field]; exists {
			if !vs.validateConstraint(value, constraint) {
				return false
			}
		}
	}
	return true
}

// validateConstraint validates a single field constraint
func (vs *ValidationService) validateConstraint(value, constraint interface{}) bool {
	constraintMap, ok := constraint.(map[string]interface{})
	if !ok {
		return true
	}

	// Validate minimum value constraint
	if minVal, exists := constraintMap["min"]; exists {
		if !vs.validateMinimumValue(value, minVal) {
			return false
		}
	}

	// Validate maximum value constraint
	if maxVal, exists := constraintMap["max"]; exists {
		if !vs.validateMaximumValue(value, maxVal) {
			return false
		}
	}

	return true
}

// validateMinimumValue validates minimum value constraint
func (vs *ValidationService) validateMinimumValue(value, minVal interface{}) bool {
	numVal, ok := value.(float64)
	if !ok {
		return true // Skip validation for non-numeric values
	}

	minFloat, ok := minVal.(float64)
	if !ok {
		return true // Skip validation if constraint is not numeric
	}

	return numVal >= minFloat
}

// validateMaximumValue validates maximum value constraint
func (vs *ValidationService) validateMaximumValue(value, maxVal interface{}) bool {
	numVal, ok := value.(float64)
	if !ok {
		return true // Skip validation for non-numeric values
	}

	maxFloat, ok := maxVal.(float64)
	if !ok {
		return true // Skip validation if constraint is not numeric
	}

	return numVal <= maxFloat
}