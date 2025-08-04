// Package plugin provides shared plugin utilities for FlexCore
// DRY PRINCIPLE: Eliminates 18-line duplication (mass=119) in init() functions across 4 plugins
package plugin

import (
	"encoding/gob"
	"time"
)

// PluginGobRegistration provides shared gob type registration for plugins
// SOLID SRP: Single responsibility for handling RPC serialization type registration
type PluginGobRegistration struct{}

// RegisterStandardTypes registers all standard types for gob RPC serialization
// DRY PRINCIPLE: Eliminates init() duplication (mass=119) across json-processor, simple-processor, postgres-processor plugins
func (pgr *PluginGobRegistration) RegisterStandardTypes() {
	// Register collection types for RPC serialization
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register([]map[string]interface{}{})

	// Register primitive types
	gob.Register(string(""))
	gob.Register(int(0))
	gob.Register(int64(0))
	gob.Register(float64(0))
	gob.Register(bool(false))
	gob.Register(time.Time{})

	// Register plugin types
	gob.Register(PluginInfo{})
	gob.Register(ProcessingStats{})
}

// RegisterAllPluginTypes is a convenience function that registers all types
// DRY PRINCIPLE: Single call to eliminate 18 lines of duplicated registration code
func RegisterAllPluginTypes() {
	registration := &PluginGobRegistration{}
	registration.RegisterStandardTypes()
}

// RegisterPluginTypesForRPC provides context-specific registration for RPC systems
// SOLID OCP: Open for extension by allowing additional type registration
func RegisterPluginTypesForRPC() {
	RegisterAllPluginTypes()

	// Additional RPC-specific registrations can be added here in the future
	// without modifying existing plugin init() functions
}
