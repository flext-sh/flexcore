// Package entities provides FlexCore's core domain entities implementing Domain-Driven Design patterns.
//
// This package contains the rich domain model for FlexCore's data processing pipeline
// orchestration system, including both Pipeline and Plugin aggregates. All entities
// follow DDD principles with strong business logic encapsulation, comprehensive
// validation, and domain event generation for CQRS and Event Sourcing integration.
//
// Plugin Entity:
// The Plugin entity represents executable data processing components that can be
// dynamically loaded and executed within pipelines. Plugins follow a standardized
// interface for data extraction, transformation, and loading operations.
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package entities

import (
	"fmt"
	"time"

	"github.com/flext-sh/flexcore/pkg/errors"
	"github.com/google/uuid"
)

// DomainEvent represents a domain event
type DomainEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// CreatePluginEvent creates a plugin-related domain event
func CreatePluginEvent(eventType, aggregateID string, data map[string]interface{}) DomainEvent {
	return DomainEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// PluginID represents a unique, strongly-typed identifier for Plugin aggregates.
//
// PluginID provides type safety by preventing accidental mixing of different
// entity identifiers and ensures consistent ID format throughout the plugin
// management system. All plugin operations require a valid PluginID for
// entity identification and event correlation.
//
// The underlying type is string to enable easy serialization and database
// storage while maintaining type safety at the application level.
//
// Example:
//
//	Working with plugin identifiers:
//
//	  id := entities.NewPluginID()
//	  fmt.Println("Generated ID:", id.String())
//
//	  // Type safety prevents mixing different ID types
//	  plugin := findPlugin(id) // Compiler enforces correct ID type
//
// Integration:
//   - Used as aggregate identifier in domain events
//   - Serves as primary key in persistence layer
//   - Enables type-safe repository and service method signatures
//   - Compatible with CQRS command and query message routing
type PluginID string

// NewPluginID generates a new unique plugin identifier using UUID v4.
//
// This function creates cryptographically random UUIDs that are globally unique
// and suitable for distributed systems. The generated IDs are compatible with
// database storage and provide sufficient entropy for collision avoidance.
//
// Returns:
//
//	PluginID: A new unique identifier for plugin entities
//
// Example:
//
//	Creating plugins with unique identifiers:
//
//	  id1 := entities.NewPluginID()
//	  id2 := entities.NewPluginID()
//
//	  // IDs are guaranteed to be different
//	  fmt.Printf("ID1: %s\nID2: %s\n", id1.String(), id2.String())
//
// Performance:
//
//	Single UUID generation call with minimal allocation overhead.
//	Suitable for high-frequency plugin creation scenarios.
//
// Thread Safety:
//
//	Safe for concurrent use as UUID generation is thread-safe.
func NewPluginID() PluginID {
	return PluginID(uuid.New().String())
}

// String returns the string representation of the plugin identifier.
//
// This method enables PluginID to implement the fmt.Stringer interface,
// providing consistent string conversion for logging, debugging, and
// serialization scenarios.
//
// Returns:
//
//	string: The underlying UUID string value
//
// Example:
//
//	String conversion and formatting:
//
//	  id := entities.NewPluginID()
//
//	  // Implicit string conversion
//	  fmt.Printf("Plugin ID: %s\n", id)
//
//	  // Explicit string conversion
//	  idString := id.String()
//	  log.Info("Loading plugin", "id", idString)
//
// Integration:
//   - Compatible with logging frameworks expecting string values
//   - Enables JSON marshaling/unmarshaling
//   - Supports database primary key conversion
//   - Works with template systems and string formatting
func (id PluginID) String() string {
	return string(id)
}

// PluginType represents the functional category of a plugin in the data processing pipeline.
//
// PluginType categorizes plugins based on their role in data processing workflows,
// following the Extract-Transform-Load (ETL) pattern and extending it with general
// processing capabilities. This classification enables proper plugin selection,
// workflow orchestration, and resource allocation.
//
// The type system ensures that plugins are used in appropriate contexts and
// supports validation of data pipeline construction and execution.
//
// Integration:
//   - Pipeline Construction: Validates plugin types in workflow steps
//   - Resource Allocation: Different types may have different resource requirements
//   - Performance Optimization: Type-specific optimizations and caching strategies
//   - Monitoring: Type-based metrics and performance tracking
//
// Thread Safety:
//
//	Plugin types are immutable constants, safe for concurrent access.
type PluginType int

const (
	// PluginTypeExtractor represents data extraction plugins.
	// These plugins connect to source systems and extract data for processing.
	// Examples: database extractors, API clients, file readers, message queue consumers.
	PluginTypeExtractor PluginType = iota

	// PluginTypeTransformer represents data transformation plugins.
	// These plugins modify, enrich, or restructure data between extraction and loading.
	// Examples: data cleaners, format converters, business logic processors, aggregators.
	PluginTypeTransformer

	// PluginTypeLoader represents data loading plugins.
	// These plugins write processed data to target systems and destinations.
	// Examples: database writers, file exporters, API publishers, data warehouse loaders.
	PluginTypeLoader

	// PluginTypeProcessor represents general-purpose processing plugins.
	// These plugins perform specialized operations that don't fit ETL categories.
	// Examples: validators, monitors, notification senders, custom business logic.
	PluginTypeProcessor
)

// String returns the string representation of the plugin type for serialization and display.
//
// This method implements the fmt.Stringer interface, enabling automatic string
// conversion for logging, API responses, and user interface display. It provides
// consistent string representations across the system.
//
// Returns:
//
//	string: Lowercase string representation of the plugin type, or "unknown" for invalid values
//
// Example:
//
//	Plugin type display and logging:
//
//	  pluginType := entities.PluginTypeExtractor
//	  fmt.Printf("Plugin type: %s\n", pluginType) // "Plugin type: extractor"
//
//	  log.Info("Loading plugin", "type", pluginType.String(), "id", pluginID)
//
// String Mappings:
//   - PluginTypeExtractor → "extractor"
//   - PluginTypeTransformer → "transformer"
//   - PluginTypeLoader → "loader"
//   - PluginTypeProcessor → "processor"
//   - Invalid values → "unknown"
//
// Integration:
//   - API serialization for plugin type information
//   - Database storage as string values
//   - Configuration file representation
//   - User interface display and filtering
//   - Logging and monitoring systems
//
// Thread Safety:
//
//	Safe for concurrent use as it only reads immutable constant values.
func (t PluginType) String() string {
	switch t {
	case PluginTypeExtractor:
		return "extractor"
	case PluginTypeTransformer:
		return "transformer"
	case PluginTypeLoader:
		return "loader"
	case PluginTypeProcessor:
		return "processor"
	default:
		return "unknown"
	}
}

// PluginStatus represents the current operational state of a plugin in its lifecycle.
//
// PluginStatus tracks the plugin's availability and operational state, enabling
// proper plugin lifecycle management, error handling, and resource allocation.
// The status controls which operations are allowed on a plugin and determines
// its availability for pipeline execution.
//
// Status transitions follow a defined lifecycle:
//
//	Registered → Active → (Inactive|Error) → Active
//
// Integration:
//   - Plugin Management: Controls plugin availability and operations
//   - Resource Allocation: Status affects resource allocation decisions
//   - Error Handling: Error status triggers recovery and notification procedures
//   - Monitoring: Status changes generate metrics and alerts
//   - Load Balancing: Only active plugins participate in execution
//
// Thread Safety:
//
//	Status values are immutable and safe for concurrent access.
//	Status transitions should be managed through Plugin entity methods.
type PluginStatus int

const (
	// PluginStatusRegistered represents a newly registered plugin awaiting activation.
	// Registered plugins have been discovered and validated but are not yet available for execution.
	PluginStatusRegistered PluginStatus = iota

	// PluginStatusActive represents a fully operational plugin ready for execution.
	// Active plugins are available for pipeline execution and resource allocation.
	PluginStatusActive

	// PluginStatusInactive represents a temporarily disabled plugin.
	// Inactive plugins are not available for execution but can be reactivated.
	PluginStatusInactive

	// PluginStatusError represents a plugin in an error state requiring attention.
	// Error plugins are unavailable for execution and require manual intervention or automatic recovery.
	PluginStatusError
)

// String returns the string representation of the plugin status for serialization and display.
//
// This method implements the fmt.Stringer interface, enabling automatic string
// conversion for logging, API responses, monitoring dashboards, and user interface
// display. It provides consistent status representations across the system.
//
// Returns:
//
//	string: Lowercase string representation of the plugin status, or "unknown" for invalid values
//
// Example:
//
//	Plugin status monitoring and logging:
//
//	  plugin := loadPlugin()
//	  fmt.Printf("Plugin status: %s\n", plugin.Status) // "Plugin status: active"
//
//	  log.Info("Plugin state changed", "status", plugin.Status.String(), "id", plugin.ID)
//
//	  // Conditional logic based on status
//	  if plugin.Status.String() == "error" {
//	      triggerErrorRecovery(plugin)
//	  }
//
// Status String Mappings:
//   - PluginStatusRegistered → "registered"
//   - PluginStatusActive → "active"
//   - PluginStatusInactive → "inactive"
//   - PluginStatusError → "error"
//   - Invalid values → "unknown"
//
// Integration:
//   - Monitoring dashboards and status displays
//   - API responses and plugin management interfaces
//   - Database storage and serialization
//   - Configuration files and deployment manifests
//   - Logging systems and audit trails
//   - Alerting and notification systems
//
// Thread Safety:
//
//	Safe for concurrent use as it only reads immutable constant values.
func (s PluginStatus) String() string {
	switch s {
	case PluginStatusRegistered:
		return "registered"
	case PluginStatusActive:
		return "active"
	case PluginStatusInactive:
		return "inactive"
	case PluginStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Plugin represents an executable data processing component within the FlexCore ecosystem.
//
// Plugin serves as an aggregate root in the domain model, encapsulating all
// functionality for dynamic plugin management, lifecycle control, and execution
// coordination. Plugins are dynamically loaded components that provide specific
// data processing capabilities within pipeline workflows.
//
// Architecture:
//
//	The Plugin aggregate follows Domain-Driven Design principles with:
//	- Rich domain model with business logic encapsulation
//	- Domain event generation for state changes
//	- Result pattern integration for explicit error handling
//	- Configuration management with flexible parameter support
//	- Capability declaration for plugin discovery and selection
//
// Fields:
//
//	id PluginID: Unique identifier for the plugin aggregate
//	name string: Human-readable plugin name (required, used for identification)
//	version string: Plugin version following semantic versioning (required)
//	description string: Detailed description of plugin functionality
//	pluginType PluginType: Functional category (extractor, transformer, loader, processor)
//	status PluginStatus: Current operational state (registered, active, inactive, error)
//	config map[string]interface{}: Flexible configuration parameters
//	capabilities []string: List of plugin capabilities and supported operations
//	createdAt time.Time: Plugin registration timestamp
//	updatedAt time.Time: Last modification timestamp
//	events []DomainEvent: Uncommitted domain events for event sourcing
//
// Plugin Lifecycle:
//  1. **Registration**: Plugin discovered and registered with basic metadata
//  2. **Activation**: Plugin validated and made available for execution
//  3. **Execution**: Plugin used in pipeline processing workflows
//  4. **Deactivation**: Plugin temporarily disabled for maintenance
//  5. **Error Handling**: Plugin marked as error state when issues occur
//
// Example:
//
//	Complete plugin creation and management:
//
//	  pluginResult := entities.NewPlugin(
//	      "oracle-extractor",
//	      "1.2.3",
//	      "Oracle database data extraction plugin",
//	      entities.PluginTypeExtractor,
//	  )
//
//	  if pluginResult.IsFailure() {
//	      return pluginResult.Error()
//	  }
//
//	  plugin := pluginResult.Value()
//
//	  // Configure plugin
//	  plugin.SetConfig(map[string]interface{}{
//	      "host":     "oracle.prod.company.com",
//	      "port":     1521,
//	      "database": "PROD",
//	      "timeout":  30,
//	  })
//
//	  // Add capabilities
//	  plugin.AddCapability("incremental-extraction")
//	  plugin.AddCapability("bulk-extraction")
//
//	  // Activate for use
//	  plugin.Activate()
//
// Integration:
//   - Pipeline Execution: Plugins are executed as part of pipeline steps
//   - Plugin Loader: Dynamic loading and instantiation of plugin implementations
//   - Resource Manager: Resource allocation and limits for plugin execution
//   - Configuration Service: Plugin configuration management and validation
//   - Monitoring System: Plugin performance tracking and health monitoring
//
// Thread Safety:
//
//	Plugin instances should be accessed by single threads within aggregate boundaries.
//	Concurrent access should be managed at the application layer through proper
//	synchronization mechanisms or message-based coordination.
type Plugin struct {
	id           PluginID               // Unique plugin identifier
	name         string                 // Human-readable plugin name
	version      string                 // Plugin version (semantic versioning)
	description  string                 // Detailed plugin description
	pluginType   PluginType             // Functional category of the plugin
	status       PluginStatus           // Current operational state
	config       map[string]interface{} // Flexible configuration parameters
	capabilities []string               // List of supported capabilities
	createdAt    time.Time              // Registration timestamp
	updatedAt    time.Time              // Last modification timestamp
	events       []DomainEvent          // Uncommitted domain events
}

// NewPlugin creates a new plugin aggregate with comprehensive validation and event generation.
//
// This factory function implements the aggregate root creation pattern, performing
// business rule validation, initializing the plugin in a valid state, and
// generating appropriate domain events for event sourcing and CQRS integration.
//
// Business Rules Enforced:
//   - Plugin name must not be empty (required for identification and discovery)
//   - Plugin version must not be empty (required for dependency management)
//   - Plugin starts in Registered status (safe initial state)
//   - Configuration and capabilities are initialized as empty collections
//   - Unique ID is generated for aggregate identification
//
// Parameters:
//
//	name string: Human-readable plugin name (required, used for identification)
//	version string: Plugin version following semantic versioning (required)
//	description string: Detailed description of plugin functionality (optional)
//	pluginType PluginType: Functional category of the plugin (extractor, transformer, loader, processor)
//
// Returns:
//
//	*Plugin: The new plugin aggregate, or error if validation fails
//
// Example:
//
//	Creating different types of plugins:
//
//	  // Data extraction plugin
//	  extractorResult := entities.NewPlugin(
//	      "postgres-extractor",
//	      "2.1.0",
//	      "PostgreSQL database data extraction with incremental support",
//	      entities.PluginTypeExtractor,
//	  )
//
//	  // Data transformation plugin
//	  transformerResult := entities.NewPlugin(
//	      "json-transformer",
//	      "1.5.2",
//	      "JSON data transformation and validation",
//	      entities.PluginTypeTransformer,
//	  )
//
//	  // Handle validation errors
//	  if extractorResult.IsFailure() {
//	      log.Error("Plugin creation failed", "error", extractorResult.Error())
//	      return extractorResult.Error()
//	  }
//
//	  plugin := extractorResult.Value()
//	  log.Info("Plugin created", "id", plugin.ID(), "name", plugin.Name())
//
// Validation Errors:
//   - ValidationError: "plugin name is required" - when name parameter is empty
//   - ValidationError: "plugin version is required" - when version parameter is empty
//
// Events Generated:
//   - PluginCreated: Contains plugin ID, name, version, type, and creation timestamp
//
// Initial State:
//   - Status: PluginStatusRegistered (ready for activation)
//   - Config: Empty map ready for configuration
//   - Capabilities: Empty slice ready for capability declarations
//   - Timestamps: CreatedAt and UpdatedAt set to current time
//   - Events: Contains PluginCreated event
//
// Integration:
//   - Plugin Registry: Registers plugin for discovery and management
//   - Event Store: Persists plugin creation event for audit trail
//   - Configuration Service: Ready to receive plugin configuration
//   - Plugin Loader: Plugin available for dynamic loading
//
// Thread Safety:
//
//	Safe for concurrent use as each call creates an independent plugin instance.
func NewPlugin(name, version, description string, pluginType PluginType) (*Plugin, error) {
	if name == "" {
		return nil, errors.ValidationError("plugin name is required")
	}

	if version == "" {
		return nil, errors.ValidationError("plugin version is required")
	}

	now := time.Now()
	plugin := &Plugin{
		id:           NewPluginID(),
		name:         name,
		version:      version,
		description:  description,
		pluginType:   pluginType,
		status:       PluginStatusRegistered,
		config:       make(map[string]interface{}),
		capabilities: make([]string, 0),
		createdAt:    now,
		updatedAt:    now,
		events:       make([]DomainEvent, 0),
	}

	// Add domain event
	event := CreatePluginEvent("PluginCreated", plugin.id.String(), map[string]interface{}{
		"plugin_id":  plugin.id.String(),
		"name":       plugin.name,
		"version":    plugin.version,
		"type":       plugin.pluginType.String(),
		"created_at": plugin.createdAt,
	})
	plugin.addEvent(event)

	return plugin, nil
}

// ID returns the unique identifier of the plugin aggregate.
//
// This method provides access to the plugin's immutable identity, which is used
// for persistence, event correlation, and aggregate references throughout the system.
// The ID is generated during plugin creation and never changes.
//
// Returns:
//
//	PluginID: The unique identifier for this plugin aggregate
//
// Example:
//
//	Using plugin ID for operations:
//
//	  plugin := loadPlugin()
//	  pluginID := plugin.ID()
//
//	  log.Info("Processing plugin", "id", pluginID.String())
//
//	  // Use ID for persistence operations
//	  if err := repository.Save(pluginID, plugin); err != nil {
//	      return err
//	  }
//
// Integration:
//   - Repository operations for persistence
//   - Event correlation and aggregate identification
//   - Cache keys and lookup operations
//   - API responses and client identification
//
// Thread Safety:
//
//	Safe for concurrent access as ID is immutable after creation.
func (p *Plugin) ID() PluginID {
	return p.id
}

// Name returns the human-readable name of the plugin.
//
// This method provides access to the plugin's display name, which is used for
// identification, user interfaces, logging, and configuration references.
// The name is set during plugin creation and is immutable.
//
// Returns:
//
//	string: The human-readable name of the plugin
//
// Example:
//
//	Using plugin name for display and identification:
//
//	  plugin := loadPlugin()
//	  name := plugin.Name()
//
//	  fmt.Printf("Loading plugin: %s\n", name)
//	  log.Info("Plugin operation completed", "name", name, "duration", duration)
//
//	  // Use name for configuration lookups
//	  config := configService.GetPluginConfig(name)
//
// Integration:
//   - User interface display and plugin management
//   - Configuration file references and lookups
//   - Logging and monitoring system identification
//   - Error messages and diagnostic information
//   - Plugin discovery and selection workflows
//
// Thread Safety:
//
//	Safe for concurrent access as name is immutable after creation.
func (p *Plugin) Name() string {
	return p.name
}

// Version returns the semantic version of the plugin.
//
// Returns:
//
//	string: Plugin version following semantic versioning (e.g., "1.2.3")
//
// Thread Safety:
//
//	Safe for concurrent access as version is immutable after creation.
func (p *Plugin) Version() string {
	return p.version
}

// Description returns the detailed description of the plugin's functionality.
//
// Returns:
//
//	string: Detailed description of what the plugin does and its capabilities
//
// Thread Safety:
//
//	Safe for concurrent access as description is immutable after creation.
func (p *Plugin) Description() string {
	return p.description
}

// Type returns the functional category of the plugin.
//
// Returns:
//
//	PluginType: The plugin's functional category (extractor, transformer, loader, processor)
//
// Thread Safety:
//
//	Safe for concurrent access as type is immutable after creation.
func (p *Plugin) Type() PluginType {
	return p.pluginType
}

// Status returns the current operational state of the plugin.
//
// Returns:
//
//	PluginStatus: Current plugin status (registered, active, inactive, error)
//
// Thread Safety:
//
//	Safe for concurrent read access. Status changes managed through business methods.
func (p *Plugin) Status() PluginStatus {
	return p.status
}

// Config returns a copy of the plugin's configuration parameters.
//
// This method returns a defensive copy of the configuration to prevent
// external modification of the plugin's internal state. Configuration
// changes should be made through the UpdateConfig method.
//
// Returns:
//
//	map[string]interface{}: Copy of plugin configuration parameters
//
// Example:
//
//	Reading plugin configuration:
//
//	  config := plugin.Config()
//	  host := config["host"].(string)
//	  port := config["port"].(int)
//
//	  // Safe to modify returned copy
//	  config["new_param"] = "value" // Does not affect plugin's internal config
//
// Thread Safety:
//
//	Safe for concurrent access as it returns a copy of the configuration.
func (p *Plugin) Config() map[string]interface{} {
	// Return a copy to prevent modification
	config := make(map[string]interface{})
	for k, v := range p.config {
		config[k] = v
	}
	return config
}

// Capabilities returns a defensive copy of the plugin's capability list.
//
// This method provides access to the plugin's declared capabilities without
// allowing external modification of the internal state. Capabilities represent
// specific features, operations, or integrations that the plugin supports.
//
// Returns:
//
//	[]string: Copy of plugin capabilities list
//
// Example:
//
//	Reading plugin capabilities:
//
//	  plugin := loadPlugin("oracle-extractor")
//	  capabilities := plugin.Capabilities()
//
//	  // Check for specific capabilities
//	  for _, capability := range capabilities {
//	      switch capability {
//	      case "incremental-extraction":
//	          fmt.Println("Plugin supports incremental data extraction")
//	      case "bulk-extraction":
//	          fmt.Println("Plugin supports bulk data extraction")
//	      case "schema-discovery":
//	          fmt.Println("Plugin can discover source schema automatically")
//	      }
//	  }
//
//	  // Safe to modify returned copy
//	  capabilities = append(capabilities, "new-capability") // Does not affect plugin
//
// Common Capabilities:
//   - "incremental-extraction": Supports delta/incremental data extraction
//   - "bulk-extraction": Supports full data extraction
//   - "schema-discovery": Can automatically discover source data schema
//   - "real-time-streaming": Supports real-time data streaming
//   - "data-validation": Includes built-in data validation
//   - "error-recovery": Supports automatic error recovery
//   - "parallel-processing": Can process data in parallel
//   - "compression-support": Supports data compression
//
// Integration:
//   - Plugin Discovery: Used for plugin matching and selection
//   - Pipeline Configuration: Validates plugin compatibility with requirements
//   - Resource Planning: Capability-based resource allocation
//   - Feature Detection: Runtime capability checking
//
// Thread Safety:
//
//	Safe for concurrent access as it returns a defensive copy.
func (p *Plugin) Capabilities() []string {
	// Return a copy to prevent modification
	capabilities := make([]string, len(p.capabilities))
	copy(capabilities, p.capabilities)
	return capabilities
}

// CreatedAt returns the plugin's creation timestamp.
//
// This method provides access to when the plugin was first registered in the
// system, useful for auditing, lifecycle management, and analytics.
//
// Returns:
//
//	time.Time: Plugin creation timestamp in UTC
//
// Example:
//
//	Using creation timestamp for analytics:
//
//	  plugin := loadPlugin()
//	  createdAt := plugin.CreatedAt()
//
//	  age := time.Since(createdAt)
//	  fmt.Printf("Plugin age: %s\n", age)
//
//	  // Check if plugin is newly created
//	  if age < 24*time.Hour {
//	      log.Info("New plugin detected", "name", plugin.Name(), "age", age)
//	  }
//
// Integration:
//   - Auditing: Track plugin registration and lifecycle
//   - Analytics: Plugin usage patterns and adoption metrics
//   - Cleanup: Identify old or unused plugins
//   - Reporting: Plugin inventory and age distribution
//
// Thread Safety:
//
//	Safe for concurrent access as creation timestamp is immutable.
func (p *Plugin) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the plugin's last modification timestamp.
//
// This method provides access to when the plugin was last modified, including
// configuration changes, status updates, capability modifications, or any other
// state changes. Useful for tracking plugin activity and change management.
//
// Returns:
//
//	time.Time: Last modification timestamp in UTC
//
// Example:
//
//	Tracking plugin modifications:
//
//	  plugin := loadPlugin()
//	  updatedAt := plugin.UpdatedAt()
//
//	  timeSinceUpdate := time.Since(updatedAt)
//	  fmt.Printf("Last updated: %s ago\n", timeSinceUpdate)
//
//	  // Check for recent changes
//	  if timeSinceUpdate < time.Hour {
//	      log.Info("Recently modified plugin", "name", plugin.Name())
//	  }
//
//	  // Compare with creation time for change tracking
//	  if !updatedAt.Equal(plugin.CreatedAt()) {
//	      fmt.Println("Plugin has been modified since creation")
//	  }
//
// Integration:
//   - Change Tracking: Monitor plugin modifications and activity
//   - Cache Invalidation: Determine when cached data should be refreshed
//   - Synchronization: Track changes for distributed system coordination
//   - Analytics: Plugin modification patterns and frequency analysis
//
// Thread Safety:
//
//	Safe for concurrent read access. Modifications managed through business methods.
func (p *Plugin) UpdatedAt() time.Time {
	return p.updatedAt
}

// Activate transitions the plugin to active status, making it available for execution.
//
// This method implements a critical state transition in the plugin lifecycle,
// enabling the plugin for use in pipeline execution. It enforces business rules
// to ensure only valid plugins can be activated and generates appropriate events
// for system coordination.
//
// Business Rules Enforced:
//   - Cannot activate already active plugins (idempotent operation)
//   - Cannot activate plugins in error status (must resolve errors first)
//   - Only registered or inactive plugins can be activated
//   - Activation generates audit events for tracking
//
// Returns:
//
//	error: nil if activated successfully, error if activation fails
//
// Example:
//
//	Plugin activation workflow:
//
//	  plugin := loadRegisteredPlugin()
//
//	  // Validate plugin is ready for activation
//	  if plugin.Status() == entities.PluginStatusError {
//	      log.Warn("Cannot activate plugin in error state", "id", plugin.ID())
//	      return errors.New("resolve plugin errors before activation")
//	  }
//
//	  // Activate plugin
//	  result := plugin.Activate()
//	  if result.IsFailure() {
//	      return result.Error()
//	  }
//
//	  log.Info("Plugin activated", "id", plugin.ID(), "name", plugin.Name())
//
// State Transitions:
//
//	Registered → Active: Normal activation after registration
//	Inactive → Active: Reactivation after temporary deactivation
//	Error → Active: Not allowed, will return validation error
//	Active → Active: Idempotent, will return validation error
//
// Events Generated:
//   - PluginActivated: Contains plugin ID, name, and activation timestamp
//
// Integration:
//   - Plugin Registry: Plugin becomes available for discovery and selection
//   - Execution Engine: Plugin registered for pipeline execution
//   - Resource Manager: Plugin eligible for resource allocation
//   - Monitoring: Plugin activation tracked for availability metrics
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Plugin) Activate() error {
	if p.status == PluginStatusActive {
		return errors.ValidationError("plugin is already active")
	}

	if p.status == PluginStatusError {
		return errors.ValidationError("cannot activate plugin in error status")
	}

	p.status = PluginStatusActive
	p.updatedAt = time.Now()

	// Add domain event
	event := CreatePluginEvent("PluginActivated", p.id.String(), map[string]interface{}{
		"plugin_id":    p.id.String(),
		"name":         p.name,
		"activated_at": p.updatedAt,
	})
	p.addEvent(event)

	return nil
}

// Deactivate transitions the plugin to inactive status, removing it from execution availability.
//
// This method safely deactivates plugins to prevent their use in new pipeline
// executions while maintaining their configuration and state for potential
// reactivation. It's useful for maintenance, debugging, or temporary removal.
//
// Business Rules Enforced:
//   - Cannot deactivate already inactive plugins (idempotent operation)
//   - Can deactivate plugins in any status except inactive
//   - Deactivation generates audit events for tracking
//   - Plugin configuration and capabilities are preserved
//
// Returns:
//
//	error: nil if deactivated successfully, error if deactivation fails
//
// Example:
//
//	Plugin maintenance workflow:
//
//	  plugin := loadActivePlugin()
//
//	  // Deactivate for maintenance
//	  result := plugin.Deactivate()
//	  if result.IsFailure() {
//	      return result.Error()
//	  }
//
//	  log.Info("Plugin deactivated for maintenance", "id", plugin.ID())
//
//	  // Perform maintenance operations
//	  performMaintenanceOperations(plugin)
//
//	  // Reactivate after maintenance
//	  plugin.Activate()
//
// State Transitions:
//
//	Active → Inactive: Normal deactivation for maintenance
//	Error → Inactive: Deactivation during error recovery
//	Registered → Inactive: Deactivation before first activation
//	Inactive → Inactive: Idempotent, will return validation error
//
// Events Generated:
//   - PluginDeactivated: Contains plugin ID, name, and deactivation timestamp
//
// Integration:
//   - Plugin Registry: Plugin removed from available plugin list
//   - Execution Engine: Plugin excluded from new pipeline executions
//   - Resource Manager: Plugin resources can be deallocated
//   - Monitoring: Plugin deactivation tracked for availability metrics
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Plugin) Deactivate() error {
	if p.status == PluginStatusInactive {
		return errors.ValidationError("plugin is already inactive")
	}

	p.status = PluginStatusInactive
	p.updatedAt = time.Now()

	// Add domain event
	event := CreatePluginEvent("PluginDeactivated", p.id.String(), map[string]interface{}{
		"plugin_id":      p.id.String(),
		"name":           p.name,
		"deactivated_at": p.updatedAt,
	})
	p.addEvent(event)

	return nil
}

// SetError transitions the plugin to error status with detailed error information.
//
// This method marks the plugin as being in an error state, making it unavailable
// for execution while preserving error information for diagnosis and recovery.
// It's typically called by the plugin execution infrastructure when unrecoverable
// errors occur during plugin operation.
//
// Business Rules Enforced:
//   - Error message is preserved for diagnostic purposes
//   - Plugin becomes unavailable for new executions
//   - Error status can be cleared through recovery or reactivation
//   - Error transitions generate audit events for monitoring
//
// Parameters:
//
//	errorMessage string: Detailed error description for diagnostic purposes
//
// Example:
//
//	Error handling during plugin execution:
//
//	  plugin := loadActivePlugin()
//
//	  // Simulate plugin execution failure
//	  err := executePluginOperation(plugin)
//	  if err != nil {
//	      plugin.SetError(err.Error())
//
//	      log.Error("Plugin execution failed",
//	          "plugin", plugin.Name(),
//	          "error", err.Error(),
//	          "status", plugin.Status().String())
//
//	      // Trigger error recovery workflow
//	      notifyPluginError(plugin.ID(), err.Error())
//	  }
//
//	  // Check if plugin is in error state
//	  if plugin.Status() == entities.PluginStatusError {
//	      fmt.Println("Plugin requires manual intervention")
//	  }
//
// State Transitions:
//
//	Any Status → Error: Plugin encounters unrecoverable error
//	Error → Active: Manual recovery or automatic error resolution
//
// Events Generated:
//   - PluginError: Contains plugin ID, name, error message, and error timestamp
//
// Integration:
//   - Error Monitoring: Plugin errors tracked for alerting and metrics
//   - Recovery Systems: Error status triggers automatic or manual recovery
//   - Diagnostic Tools: Error information used for troubleshooting
//   - Plugin Registry: Error plugins excluded from execution selection
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Plugin) SetError(errorMessage string) {
	p.status = PluginStatusError
	p.updatedAt = time.Now()

	// Add domain event
	event := CreatePluginEvent("PluginError", p.id.String(), map[string]interface{}{
		"plugin_id": p.id.String(),
		"name":      p.name,
		"error":     errorMessage,
		"error_at":  p.updatedAt,
	})
	p.addEvent(event)
}

// UpdateConfig replaces the plugin's configuration with new parameters.
//
// This method completely replaces the plugin's configuration with the provided
// parameters, enabling runtime reconfiguration without requiring plugin restart.
// Configuration changes generate events for system coordination and audit trails.
//
// Business Rules Enforced:
//   - Configuration cannot be nil (must provide valid configuration object)
//   - Configuration is completely replaced (not merged with existing)
//   - Configuration changes update the plugin's modification timestamp
//   - Configuration updates generate audit events for tracking
//
// Parameters:
//
//	config map[string]interface{}: New configuration parameters to apply
//
// Returns:
//
//	error: nil if configuration updated successfully, error if update fails
//
// Example:
//
//	Updating plugin configuration:
//
//	  plugin := loadPlugin("database-extractor")
//
//	  // New configuration with updated connection parameters
//	  newConfig := map[string]interface{}{
//	      "host":               "new-db-server.company.com",
//	      "port":               5432,
//	      "database":           "production_db",
//	      "username":           "app_user",
//	      "password":           secretManager.GetPassword("db_password"),
//	      "connection_timeout": 30,
//	      "max_connections":    10,
//	      "ssl_mode":           "require",
//	  }
//
//	  result := plugin.UpdateConfig(newConfig)
//	  if result.IsFailure() {
//	      log.Error("Failed to update plugin config", "error", result.Error())
//	      return result.Error()
//	  }
//
//	  log.Info("Plugin configuration updated", "plugin", plugin.Name())
//
// Configuration Examples:
//
//	Database plugins: host, port, credentials, connection pooling
//	API plugins: endpoints, authentication, rate limiting, timeouts
//	File plugins: paths, formats, compression, encoding
//	Processing plugins: batch size, parallelism, memory limits
//
// Validation Errors:
//   - ValidationError: "config cannot be nil" - when config parameter is nil
//
// Events Generated:
//   - PluginConfigUpdated: Contains plugin ID, name, and update timestamp
//
// Integration:
//   - Configuration Management: Runtime plugin reconfiguration
//   - Secret Management: Integration with secure credential storage
//   - Hot Reload: Configuration changes without service restart
//   - Audit Trail: Track configuration changes for compliance
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Plugin) UpdateConfig(config map[string]interface{}) error {
	if config == nil {
		return errors.ValidationError("config cannot be nil")
	}

	p.config = make(map[string]interface{})
	for k, v := range config {
		p.config[k] = v
	}
	p.updatedAt = time.Now()

	// Add domain event
	event := CreatePluginEvent("PluginConfigUpdated", p.id.String(), map[string]interface{}{
		"plugin_id":  p.id.String(),
		"name":       p.name,
		"updated_at": p.updatedAt,
	})
	p.addEvent(event)

	return nil
}

// AddCapability adds a new capability to the plugin's capability list.
//
// This method extends the plugin's declared capabilities with a new feature
// or operation. Capabilities are used for plugin discovery, compatibility
// checking, and feature-based plugin selection in pipeline construction.
//
// Business Rules Enforced:
//   - Capability name cannot be empty (must be valid identifier)
//   - Duplicate capabilities are not allowed (prevents redundancy)
//   - Capability additions update the plugin's modification timestamp
//   - Capability list maintains insertion order for consistent behavior
//
// Parameters:
//
//	capability string: Name of the capability to add (must be non-empty)
//
// Returns:
//
//	error: nil if capability added successfully, error if addition fails
//
// Example:
//
//	Adding capabilities during plugin configuration:
//
//	  plugin := loadPlugin("data-processor")
//
//	  // Add data processing capabilities
//	  capabilities := []string{
//	      "json-transformation",
//	      "csv-parsing",
//	      "data-validation",
//	      "schema-inference",
//	      "parallel-processing",
//	  }
//
//	  for _, capability := range capabilities {
//	      result := plugin.AddCapability(capability)
//	      if result.IsFailure() {
//	          if result.Error().Error() == "capability already exists" {
//	              log.Debug("Capability already exists", "capability", capability)
//	              continue
//	          }
//	          return result.Error()
//	      }
//	      log.Info("Added capability", "plugin", plugin.Name(), "capability", capability)
//	  }
//
//	  // Verify capabilities were added
//	  allCapabilities := plugin.Capabilities()
//	  fmt.Printf("Plugin now has %d capabilities\n", len(allCapabilities))
//
// Capability Naming Conventions:
//   - Use kebab-case: "incremental-extraction", "real-time-streaming"
//   - Be descriptive: "bulk-data-processing", "schema-auto-discovery"
//   - Include context: "oracle-specific-optimizations", "json-schema-validation"
//
// Validation Errors:
//   - ValidationError: "capability cannot be empty" - when capability parameter is empty
//   - ValidationError: "capability already exists" - when capability is already in the list
//
// Integration:
//   - Plugin Discovery: Capabilities used for plugin matching and filtering
//   - Pipeline Builder: Feature-based plugin selection and compatibility
//   - Resource Planning: Capability-aware resource allocation
//   - Feature Gates: Runtime feature detection and enabling
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Plugin) AddCapability(capability string) error {
	if capability == "" {
		return errors.ValidationError("capability cannot be empty")
	}

	// Check if capability already exists
	for _, existing := range p.capabilities {
		if existing == capability {
			return errors.ValidationError("capability already exists")
		}
	}

	p.capabilities = append(p.capabilities, capability)
	p.updatedAt = time.Now()

	return nil
}

// RemoveCapability removes an existing capability from the plugin's capability list.
//
// This method removes a previously declared capability from the plugin,
// effectively disabling or removing support for a specific feature or operation.
// This is useful for plugin updates, feature deprecation, or dynamic capability
// management based on runtime conditions.
//
// Business Rules Enforced:
//   - Capability must exist in the current list to be removed
//   - Capability removal updates the plugin's modification timestamp
//   - List order is preserved after removal (no gaps or reordering)
//   - Removal operation is idempotent-safe (fails if capability not found)
//
// Parameters:
//
//	capability string: Name of the capability to remove
//
// Returns:
//
//	error: nil if capability removed successfully, error if not found
//
// Example:
//
//	Managing plugin capabilities dynamically:
//
//	  plugin := loadPlugin("multi-format-processor")
//
//	  // Check current capabilities
//	  capabilities := plugin.Capabilities()
//	  fmt.Printf("Current capabilities: %v\n", capabilities)
//
//	  // Remove deprecated or unsupported capabilities
//	  deprecatedCapabilities := []string{
//	      "legacy-xml-format",
//	      "deprecated-api-v1",
//	      "experimental-feature",
//	  }
//
//	  for _, capability := range deprecatedCapabilities {
//	      result := plugin.RemoveCapability(capability)
//	      if result.IsFailure() {
//	          log.Debug("Capability not found for removal", "capability", capability)
//	          continue
//	      }
//	      log.Info("Removed capability", "plugin", plugin.Name(), "capability", capability)
//	  }
//
//	  // Verify capabilities were removed
//	  updatedCapabilities := plugin.Capabilities()
//	  fmt.Printf("Updated capabilities: %v\n", updatedCapabilities)
//
// Use Cases:
//   - Feature Deprecation: Remove support for deprecated features
//   - Runtime Adaptation: Disable capabilities based on environment
//   - Security Hardening: Remove potentially dangerous capabilities
//   - Plugin Updates: Capability changes during plugin version updates
//
// Validation Errors:
//   - ValidationError: "capability not found" - when capability doesn't exist in the list
//
// Integration:
//   - Plugin Management: Dynamic capability management and updates
//   - Feature Flags: Runtime capability enabling/disabling
//   - Security Policies: Capability-based access control
//   - Version Migration: Capability changes during plugin updates
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Plugin) RemoveCapability(capability string) error {
	for i, existing := range p.capabilities {
		if existing == capability {
			p.capabilities = append(p.capabilities[:i], p.capabilities[i+1:]...)
			p.updatedAt = time.Now()
			return nil
		}
	}

	return errors.ValidationError("capability not found")
}

// HasCapability checks if the plugin declares a specific capability.
//
// This method performs a case-sensitive lookup to determine if the plugin
// supports a particular feature or operation. It's commonly used for plugin
// selection, compatibility checking, and feature detection in pipeline construction.
//
// Parameters:
//
//	capability string: Name of the capability to check for
//
// Returns:
//
//	bool: true if the plugin has the capability, false otherwise
//
// Example:
//
//	Feature-based plugin selection:
//
//	  plugin := loadPlugin("oracle-extractor")
//
//	  // Check for required capabilities
//	  requiredCapabilities := []string{
//	      "incremental-extraction",
//	      "schema-discovery",
//	      "bulk-processing",
//	  }
//
//	  allSupported := true
//	  for _, required := range requiredCapabilities {
//	      if !plugin.HasCapability(required) {
//	          log.Warn("Plugin missing required capability",
//	              "plugin", plugin.Name(),
//	              "capability", required)
//	          allSupported = false
//	      }
//	  }
//
//	  if allSupported {
//	      fmt.Println("Plugin supports all required capabilities")
//	  }
//
//	  // Check for optional capabilities
//	  if plugin.HasCapability("parallel-processing") {
//	      fmt.Println("Plugin supports parallel processing - enabling optimization")
//	      enableParallelMode(plugin)
//	  }
//
//	  // Conditional feature usage
//	  if plugin.HasCapability("real-time-streaming") {
//	      setupStreamingPipeline(plugin)
//	  } else {
//	      setupBatchPipeline(plugin)
//	  }
//
// Common Usage Patterns:
//   - Pipeline Builder: Select plugins based on required capabilities
//   - Feature Detection: Enable/disable features based on plugin support
//   - Compatibility Check: Validate plugin compatibility with requirements
//   - Performance Optimization: Use advanced features when available
//
// Integration:
//   - Plugin Registry: Filter plugins by capability requirements
//   - Pipeline Validation: Ensure all required capabilities are available
//   - Feature Gates: Runtime feature detection and conditional execution
//   - Resource Planning: Capability-aware resource allocation and optimization
//
// Thread Safety:
//
//	Safe for concurrent access as it only reads the capabilities list.
func (p *Plugin) HasCapability(capability string) bool {
	for _, existing := range p.capabilities {
		if existing == capability {
			return true
		}
	}
	return false
}

// GetEvents returns the accumulated domain events for this plugin aggregate.
//
// This method provides access to the uncommitted domain events that have been
// generated by business operations on this plugin. Events are used for event
// sourcing, CQRS read model updates, and integration with external systems.
//
// Returns:
//
//	[]DomainEvent: List of uncommitted domain events generated by plugin operations
//
// Example:
//
//	Processing plugin domain events:
//
//	  plugin := loadPlugin()
//
//	  // Perform business operations that generate events
//	  plugin.Activate()
//	  plugin.UpdateConfig(newConfig)
//	  plugin.AddCapability("new-feature")
//
//	  // Retrieve and process events
//	  events := plugin.GetEvents()
//	  fmt.Printf("Plugin generated %d events\n", len(events))
//
//	  for _, event := range events {
//	      fmt.Printf("Event: %s at %s\n", event.Type, event.Timestamp)
//
//	      // Process events for different purposes
//	      switch event.Type {
//	      case "PluginActivated":
//	          updatePluginStatus(event)
//	          notifyPluginAvailable(event)
//	      case "PluginConfigUpdated":
//	          invalidateConfigCache(event)
//	          auditConfigChange(event)
//	      case "PluginCapabilityAdded":
//	          updatePluginIndex(event)
//	          notifyCapabilityChange(event)
//	      }
//	  }
//
//	  // Clear events after processing
//	  plugin.ClearEvents()
//
// Event Processing Workflow:
//  1. Business operations generate domain events
//  2. Events accumulate in the aggregate until persistence
//  3. Application layer retrieves events for processing
//  4. Events are published to event bus for handlers
//  5. Events are cleared after successful processing
//
// Integration:
//   - Event Sourcing: Events stored in event store for aggregate reconstruction
//   - CQRS: Events update read models and projections
//   - Integration Events: External system notifications and webhooks
//   - Audit Trail: Complete history of plugin operations and changes
//
// Thread Safety:
//
//	Safe for concurrent read access. Event generation managed through business methods.
func (p *Plugin) GetEvents() []DomainEvent {
	return p.events
}

// ClearEvents removes all accumulated domain events from the plugin aggregate.
//
// This method resets the event collection after successful event processing,
// preventing duplicate event processing and maintaining clean aggregate state.
// It should be called by the application layer after events have been successfully
// published and processed.
//
// Example:
//
//	Complete event processing workflow:
//
//	  plugin := loadPlugin()
//
//	  // Perform business operations (generates events)
//	  plugin.Activate()
//	  plugin.UpdateConfig(newConfig)
//
//	  // Process events
//	  events := plugin.GetEvents()
//
//	  // Publish events to event bus
//	  for _, event := range events {
//	      if err := eventBus.Publish(event); err != nil {
//	          log.Error("Failed to publish event", "error", err, "event", event.Type)
//	          return err // Don't clear events if publishing failed
//	      }
//	  }
//
//	  // Save aggregate to repository
//	  if err := pluginRepository.Save(plugin); err != nil {
//	      log.Error("Failed to save plugin", "error", err)
//	      return err // Don't clear events if save failed
//	  }
//
//	  // Clear events only after successful processing
//	  plugin.ClearEvents()
//	  log.Info("Plugin events processed successfully", "count", len(events))
//
// Event Lifecycle:
//  1. Business operations generate and accumulate events
//  2. Application layer retrieves events with GetEvents()
//  3. Events are persisted to event store
//  4. Events are published to event bus for handlers
//  5. Events are cleared with ClearEvents() after success
//  6. Failed processing preserves events for retry
//
// Integration:
//   - Event Processing: Reset event collection after successful publishing
//   - Error Recovery: Events preserved if processing fails
//   - Aggregate Persistence: Clean state after successful operations
//   - Memory Management: Prevent event accumulation and memory leaks
//
// Thread Safety:
//
//	Not thread-safe - should be called from single thread managing the aggregate.
func (p *Plugin) ClearEvents() {
	p.events = make([]DomainEvent, 0)
}

// addEvent adds a domain event
func (p *Plugin) addEvent(event DomainEvent) {
	p.events = append(p.events, event)
}

// IsCompatibleWith checks if this plugin can be chained with another plugin in a pipeline.
//
// This method determines plugin compatibility based on their functional types
// and data flow patterns in ETL pipelines. It enforces proper data processing
// workflow where extractors feed transformers, transformers feed loaders, and
// processors can integrate anywhere in the chain.
//
// Compatibility Rules:
//   - Extractors: Compatible with Transformers and Loaders (data source → processing/destination)
//   - Transformers: Compatible with Loaders and other Transformers (processing → processing/destination)
//   - Loaders: Not compatible with other plugins (data destinations are endpoints)
//   - Processors: Compatible with all plugin types (general-purpose processing)
//
// Parameters:
//
//	other *Plugin: The plugin to check compatibility with
//
// Returns:
//
//	bool: true if plugins can be chained together, false otherwise
//
// Example:
//
//	Pipeline construction with compatibility checking:
//
//	  extractor := loadPlugin("oracle-extractor")     // PluginTypeExtractor
//	  transformer := loadPlugin("json-transformer")    // PluginTypeTransformer
//	  loader := loadPlugin("postgres-loader")          // PluginTypeLoader
//	  processor := loadPlugin("data-validator")        // PluginTypeProcessor
//
//	  // Valid pipeline chains
//	  if extractor.IsCompatibleWith(transformer) {
//	      fmt.Println("Can chain: Extractor → Transformer")
//	  }
//
//	  if transformer.IsCompatibleWith(loader) {
//	      fmt.Println("Can chain: Transformer → Loader")
//	  }
//
//	  if extractor.IsCompatibleWith(loader) {
//	      fmt.Println("Can chain: Extractor → Loader (direct)")
//	  }
//
//	  // Processor compatibility
//	  if processor.IsCompatibleWith(transformer) {
//	      fmt.Println("Processor can work with Transformer")
//	  }
//
//	  // Invalid chains
//	  if !loader.IsCompatibleWith(transformer) {
//	      fmt.Println("Cannot chain: Loader → Transformer (loaders are endpoints)")
//	  }
//
//	  // Build valid pipeline
//	  pipeline := []Plugin{extractor, processor, transformer, loader}
//	  if validatePipelineCompatibility(pipeline) {
//	      fmt.Println("Pipeline is valid")
//	  }
//
// Common Pipeline Patterns:
//   - Extract → Load: Direct data movement without transformation
//   - Extract → Transform → Load: Full ETL pipeline with data processing
//   - Extract → Process → Transform → Load: Enhanced ETL with validation/enrichment
//   - Extract → Transform → Transform → Load: Multi-stage data transformation
//
// Integration:
//   - Pipeline Builder: Validate plugin chains during pipeline construction
//   - Workflow Validation: Ensure proper data flow in processing workflows
//   - Plugin Selection: Filter compatible plugins during pipeline design
//   - Error Prevention: Catch invalid plugin combinations before execution
//
// Thread Safety:
//
//	Safe for concurrent access as it only reads immutable plugin properties.
func (p *Plugin) IsCompatibleWith(other *Plugin) bool {
	// Basic compatibility check based on types
	switch p.pluginType {
	case PluginTypeExtractor:
		return other.pluginType == PluginTypeTransformer || other.pluginType == PluginTypeLoader
	case PluginTypeTransformer:
		return other.pluginType == PluginTypeLoader || other.pluginType == PluginTypeTransformer
	case PluginTypeLoader:
		return false // Loaders are typically endpoints
	case PluginTypeProcessor:
		return true // Processors can work with any type
	default:
		return false
	}
}

// PluginValidator provides specialized validation for plugin entities
// SOLID SRP: Single responsibility for plugin validation eliminating 6 returns
type PluginValidator struct{}

// NewPluginValidator creates a new plugin validator
func NewPluginValidator() *PluginValidator {
	return &PluginValidator{}
}

// PluginValidationRule represents a single validation rule for plugins
type PluginValidationRule struct {
	FieldName string
	Check     func(*Plugin) bool
	Message   string
}

// getPluginValidationRules returns all validation rules for Plugin
// SOLID OCP: Open for extension with new validation rules
func (v *PluginValidator) getPluginValidationRules() []PluginValidationRule {
	return []PluginValidationRule{
		{"ID", func(p *Plugin) bool { return p.id != "" }, "plugin ID is required"},
		{"Name", func(p *Plugin) bool { return p.name != "" }, "plugin name is required"},
		{"Version", func(p *Plugin) bool { return p.version != "" }, "plugin version is required"},
		{"CreatedAt", func(p *Plugin) bool { return !p.createdAt.IsZero() }, "created at time is required"},
		{"UpdatedAt", func(p *Plugin) bool { return !p.updatedAt.IsZero() }, "updated at time is required"},
	}
}

// ValidatePlugin validates plugin using rule-based approach
// DRY PRINCIPLE: Eliminates 6 return statements using validation rules
func (v *PluginValidator) ValidatePlugin(plugin *Plugin) error {
	for _, rule := range v.getPluginValidationRules() {
		if !rule.Check(plugin) {
			return errors.ValidationError(rule.Message)
		}
	}
	return nil
}

// Validate validates the plugin state
// DRY PRINCIPLE: Delegates to specialized validator eliminating multiple returns
func (p *Plugin) Validate() error {
	validator := NewPluginValidator()
	return validator.ValidatePlugin(p)
}
