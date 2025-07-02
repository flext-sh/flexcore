// Package adapter defines types for FlexCore adapters
package adapter

import (
	"context"
	"time"
)

// Adapter is the main interface that all adapters must implement
type Adapter interface {
	// Metadata
	Name() string
	Version() string

	// Configuration
	Configure(config map[string]interface{}) error

	// Operations
	Extract(ctx context.Context, request ExtractRequest) (*ExtractResponse, error)
	Load(ctx context.Context, request LoadRequest) (*LoadResponse, error)
	Transform(ctx context.Context, request TransformRequest) (*TransformResponse, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// ExtractRequest represents a data extraction request
type ExtractRequest struct {
	// Source identification
	Source string `json:"source" validate:"required"`
	
	// Query parameters
	Query  string                 `json:"query,omitempty"`
	Filter map[string]interface{} `json:"filter,omitempty"`
	
	// Pagination
	Limit  int `json:"limit,omitempty" validate:"min=0,max=10000"`
	Offset int `json:"offset,omitempty" validate:"min=0"`
	
	// Time range
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	
	// Additional options
	Options map[string]interface{} `json:"options,omitempty"`
}

// ExtractResponse represents the response from an extraction
type ExtractResponse struct {
	// Extracted records
	Records []Record `json:"records"`
	
	// Pagination info
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
	
	// Metadata
	ExtractedAt time.Time              `json:"extracted_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LoadRequest represents a data loading request
type LoadRequest struct {
	// Target identification
	Target string `json:"target" validate:"required"`
	
	// Records to load
	Records []Record `json:"records" validate:"required,min=1"`
	
	// Load options
	Mode    LoadMode               `json:"mode" validate:"required,oneof=append replace upsert"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// LoadResponse represents the response from a load operation
type LoadResponse struct {
	// Load statistics
	RecordsLoaded int64 `json:"records_loaded"`
	RecordsFailed int64 `json:"records_failed"`
	
	// Failed records details
	FailedRecords []FailedRecord `json:"failed_records,omitempty"`
	
	// Metadata
	LoadedAt time.Time              `json:"loaded_at"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TransformRequest represents a data transformation request
type TransformRequest struct {
	// Input records
	Records []Record `json:"records" validate:"required,min=1"`
	
	// Transformation spec
	Transformations []Transformation `json:"transformations" validate:"required,min=1"`
	
	// Options
	Options map[string]interface{} `json:"options,omitempty"`
}

// TransformResponse represents the response from a transformation
type TransformResponse struct {
	// Transformed records
	Records []Record `json:"records"`
	
	// Transformation statistics
	RecordsTransformed int64 `json:"records_transformed"`
	RecordsFailed      int64 `json:"records_failed"`
	
	// Failed records
	FailedRecords []FailedRecord `json:"failed_records,omitempty"`
	
	// Metadata
	TransformedAt time.Time              `json:"transformed_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Record represents a single data record
type Record struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// FailedRecord represents a record that failed to process
type FailedRecord struct {
	Record Record `json:"record"`
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
}

// LoadMode represents the mode of loading data
type LoadMode string

const (
	LoadModeAppend  LoadMode = "append"
	LoadModeReplace LoadMode = "replace"
	LoadModeUpsert  LoadMode = "upsert"
)

// Transformation represents a data transformation
type Transformation struct {
	Type    TransformationType     `json:"type" validate:"required"`
	Field   string                 `json:"field,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// TransformationType represents the type of transformation
type TransformationType string

const (
	TransformTypeRename    TransformationType = "rename"
	TransformTypeMap       TransformationType = "map"
	TransformTypeFilter    TransformationType = "filter"
	TransformTypeAggregate TransformationType = "aggregate"
	TransformTypeCustom    TransformationType = "custom"
)

// Schema represents the schema of data
type Schema struct {
	Fields []Field                `json:"fields"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

// Field represents a field in a schema
type Field struct {
	Name        string                 `json:"name"`
	Type        FieldType              `json:"type"`
	Required    bool                   `json:"required"`
	Description string                 `json:"description,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

// FieldType represents the type of a field
type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeInteger  FieldType = "integer"
	FieldTypeFloat    FieldType = "float"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeDate     FieldType = "date"
	FieldTypeDateTime FieldType = "datetime"
	FieldTypeJSON     FieldType = "json"
	FieldTypeArray    FieldType = "array"
	FieldTypeObject   FieldType = "object"
)