// Package entities contains domain entities for FlexCore
package entities

import (
	"time"

	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
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

// PluginID represents a unique plugin identifier
type PluginID string

// NewPluginID creates a new plugin ID
func NewPluginID() PluginID {
	return PluginID(uuid.New().String())
}

// String returns the string representation of the plugin ID
func (id PluginID) String() string {
	return string(id)
}

// PluginType represents the type of plugin
type PluginType int

const (
	PluginTypeExtractor PluginType = iota
	PluginTypeTransformer
	PluginTypeLoader
	PluginTypeProcessor
)

// String returns the string representation of the plugin type
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

// PluginStatus represents the status of a plugin
type PluginStatus int

const (
	PluginStatusRegistered PluginStatus = iota
	PluginStatusActive
	PluginStatusInactive
	PluginStatusError
)

// String returns the string representation of the plugin status
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

// Plugin represents a plugin in the domain
type Plugin struct {
	id           PluginID
	name         string
	version      string
	description  string
	pluginType   PluginType
	status       PluginStatus
	config       map[string]interface{}
	capabilities []string
	createdAt    time.Time
	updatedAt    time.Time
	events       []DomainEvent
}

// NewPlugin creates a new plugin
func NewPlugin(name, version, description string, pluginType PluginType) result.Result[*Plugin] {
	if name == "" {
		return result.Failure[*Plugin](errors.ValidationError("plugin name is required"))
	}

	if version == "" {
		return result.Failure[*Plugin](errors.ValidationError("plugin version is required"))
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

	return result.Success(plugin)
}

// ID returns the plugin ID
func (p *Plugin) ID() PluginID {
	return p.id
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return p.version
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return p.description
}

// Type returns the plugin type
func (p *Plugin) Type() PluginType {
	return p.pluginType
}

// Status returns the plugin status
func (p *Plugin) Status() PluginStatus {
	return p.status
}

// Config returns the plugin configuration
func (p *Plugin) Config() map[string]interface{} {
	// Return a copy to prevent modification
	config := make(map[string]interface{})
	for k, v := range p.config {
		config[k] = v
	}
	return config
}

// Capabilities returns the plugin capabilities
func (p *Plugin) Capabilities() []string {
	// Return a copy to prevent modification
	capabilities := make([]string, len(p.capabilities))
	copy(capabilities, p.capabilities)
	return capabilities
}

// CreatedAt returns the creation time
func (p *Plugin) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update time
func (p *Plugin) UpdatedAt() time.Time {
	return p.updatedAt
}

// Activate activates the plugin
func (p *Plugin) Activate() result.Result[bool] {
	if p.status == PluginStatusActive {
		return result.Failure[bool](errors.ValidationError("plugin is already active"))
	}

	if p.status == PluginStatusError {
		return result.Failure[bool](errors.ValidationError("cannot activate plugin in error status"))
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

	return result.Success(true)
}

// Deactivate deactivates the plugin
func (p *Plugin) Deactivate() result.Result[bool] {
	if p.status == PluginStatusInactive {
		return result.Failure[bool](errors.ValidationError("plugin is already inactive"))
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

	return result.Success(true)
}

// SetError sets the plugin to error status
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

// UpdateConfig updates the plugin configuration
func (p *Plugin) UpdateConfig(config map[string]interface{}) result.Result[bool] {
	if config == nil {
		return result.Failure[bool](errors.ValidationError("config cannot be nil"))
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

	return result.Success(true)
}

// AddCapability adds a capability to the plugin
func (p *Plugin) AddCapability(capability string) result.Result[bool] {
	if capability == "" {
		return result.Failure[bool](errors.ValidationError("capability cannot be empty"))
	}

	// Check if capability already exists
	for _, existing := range p.capabilities {
		if existing == capability {
			return result.Failure[bool](errors.ValidationError("capability already exists"))
		}
	}

	p.capabilities = append(p.capabilities, capability)
	p.updatedAt = time.Now()

	return result.Success(true)
}

// RemoveCapability removes a capability from the plugin
func (p *Plugin) RemoveCapability(capability string) result.Result[bool] {
	for i, existing := range p.capabilities {
		if existing == capability {
			p.capabilities = append(p.capabilities[:i], p.capabilities[i+1:]...)
			p.updatedAt = time.Now()
			return result.Success(true)
		}
	}

	return result.Failure[bool](errors.ValidationError("capability not found"))
}

// HasCapability checks if the plugin has a specific capability
func (p *Plugin) HasCapability(capability string) bool {
	for _, existing := range p.capabilities {
		if existing == capability {
			return true
		}
	}
	return false
}

// GetEvents returns the domain events
func (p *Plugin) GetEvents() []DomainEvent {
	return p.events
}

// ClearEvents clears the domain events
func (p *Plugin) ClearEvents() {
	p.events = make([]DomainEvent, 0)
}

// addEvent adds a domain event
func (p *Plugin) addEvent(event DomainEvent) {
	p.events = append(p.events, event)
}

// IsCompatibleWith checks if this plugin is compatible with another plugin
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
func (v *PluginValidator) ValidatePlugin(plugin *Plugin) result.Result[bool] {
	for _, rule := range v.getPluginValidationRules() {
		if !rule.Check(plugin) {
			return result.Failure[bool](errors.ValidationError(rule.Message))
		}
	}
	return result.Success(true)
}

// Validate validates the plugin state
// DRY PRINCIPLE: Delegates to specialized validator eliminating multiple returns
func (p *Plugin) Validate() result.Result[bool] {
	validator := NewPluginValidator()
	return validator.ValidatePlugin(p)
}
