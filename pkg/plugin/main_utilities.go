// Package plugin provides shared utilities for plugin main functions
// DRY PRINCIPLE: Eliminates main() function duplication between multiple plugins
package plugin

import (
	"log"
	"os"

	hashicorpPlugin "github.com/hashicorp/go-plugin"
)

// PluginMainConfig represents configuration for plugin main function
type PluginMainConfig struct {
	PluginName      string
	LogPrefix       string
	StartMsg        string
	StopMsg         string
	Version         string  // Plugin version for --version flag
	HandshakeValue  string  // Custom handshake value (optional)
}

// SetupPluginLogging configures logging for a plugin with consistent format
// DRY PRINCIPLE: Eliminates 26-line duplication (mass=110) in main() functions
func SetupPluginLogging(config PluginMainConfig) {
	log.SetPrefix(config.LogPrefix)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println(config.StartMsg)
}

// HandleVersionFlag checks for --version flag and exits if found
// DRY PRINCIPLE: Eliminates version handling duplication across plugins
func HandleVersionFlag(config PluginMainConfig) {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		log.Printf("%s %s", config.PluginName, config.Version)
		os.Exit(0)
	}
}

// CreateHandshakeConfig creates standard handshake configuration for FlexCore plugins
// DRY PRINCIPLE: Eliminates handshake config duplication across plugins
func CreateHandshakeConfig() hashicorpPlugin.HandshakeConfig {
	return hashicorpPlugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: "flexcore-plugin-magic-cookie",
	}
}

// CreateCustomHandshakeConfig creates handshake configuration with custom value
// DRY PRINCIPLE: Supports legacy plugins with different handshake values
func CreateCustomHandshakeConfig(customValue string) hashicorpPlugin.HandshakeConfig {
	return hashicorpPlugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: customValue,
	}
}

// ServePlugin serves a plugin with standard configuration
// DRY PRINCIPLE: Eliminates plugin serve logic duplication
func ServePlugin(pluginMap map[string]hashicorpPlugin.Plugin, stopMsg string) {
	ServePluginWithHandshake(pluginMap, CreateHandshakeConfig(), stopMsg)
}

// ServePluginWithHandshake serves a plugin with custom handshake configuration
// DRY PRINCIPLE: Eliminates plugin serve logic duplication while supporting custom handshakes
func ServePluginWithHandshake(pluginMap map[string]hashicorpPlugin.Plugin, handshakeConfig hashicorpPlugin.HandshakeConfig, stopMsg string) {
	defer log.Println(stopMsg)

	// Serve the plugin with provided handshake
	hashicorpPlugin.Serve(&hashicorpPlugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}

// RunPluginMain executes the complete plugin main flow with standard configuration
// DRY PRINCIPLE: Eliminates complete main() function duplication between plugins
func RunPluginMain(config PluginMainConfig, pluginFactory func() hashicorpPlugin.Plugin) {
	// Handle version flag first (exits if found)
	if config.Version != "" {
		HandleVersionFlag(config)
	}

	// Setup logging
	SetupPluginLogging(config)

	// Create plugin map
	pluginMap := map[string]hashicorpPlugin.Plugin{
		"flexcore": pluginFactory(),
	}

	// Determine handshake configuration
	var handshakeConfig hashicorpPlugin.HandshakeConfig
	if config.HandshakeValue != "" {
		handshakeConfig = CreateCustomHandshakeConfig(config.HandshakeValue)
	} else {
		handshakeConfig = CreateHandshakeConfig()
	}

	// Serve plugin with appropriate handshake
	ServePluginWithHandshake(pluginMap, handshakeConfig, config.StopMsg)
}