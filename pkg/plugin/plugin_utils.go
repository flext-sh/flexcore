// Package plugin provides utility functions for plugin operations
// SOLID SRP: Dedicated module for utility and helper functions
package plugin

import (
	"os"
	"time"
)

// PluginUtilities provides common utility functions for plugins
// SOLID SRP: Single responsibility for utility operations
type PluginUtilities struct{}

// NewPluginUtilities creates a new plugin utilities instance
func NewPluginUtilities() *PluginUtilities {
	return &PluginUtilities{}
}

// GetHostname returns the hostname with fallback to localhost
func (pu *PluginUtilities) GetHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "localhost"
}

// CreateBaseResult creates a base result structure with common fields
func (pu *PluginUtilities) CreateBaseResult(processorName string) map[string]interface{} {
	return map[string]interface{}{
		"processor": processorName,
		"timestamp": time.Now().Unix(),
		"node_info": map[string]interface{}{
			"hostname": pu.GetHostname(),
			"pid":      os.Getpid(),
		},
	}
}

// CreatePluginInfo creates standardized plugin information
func (pu *PluginUtilities) CreatePluginInfo(id, name, version, description string, startTime time.Time) PluginInfo {
	return PluginInfo{
		ID:          id,
		Name:        name,
		Version:     version,
		Description: description,
		Type:        "processor",
		Author:      "FLEXT Team",
		Tags:        []string{"data", "processing", "transformation", "enrichment"},
		Config: map[string]string{
			"mode":         "production",
			"batch_size":   "1000",
			"timeout":      "30s",
			"enable_cache": "true",
			"max_memory":   "512MB",
		},
		Status:   "active",
		LoadedAt: startTime,
		Health:   "healthy",
	}
}
