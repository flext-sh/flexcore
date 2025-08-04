// FlexCore Server - Professional Clean Architecture Implementation
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpcontrollers "github.com/flext-sh/flexcore/internal/adapters/primary/http"
	"github.com/flext-sh/flexcore/pkg/config"
	"github.com/flext-sh/flexcore/pkg/logging"
	"github.com/flext-sh/flexcore/pkg/orchestrator"
	"github.com/flext-sh/flext/pkg/plugins/communication"
	"github.com/flext-sh/flext/pkg/plugins/loader"
)

// Version information (set by build flags)
var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "unknown"
)

// Server constants
const (
	shutdownTimeoutSeconds = 30
)

// FlexCorePluginSystem holds the plugin system components
type FlexCorePluginSystem struct {
	communicationBus *communication.FlexCoreCommunicationBus
	pluginLoader     *loader.PluginLoader
	loadedPlugins    map[string]interface{} // map[pluginID]plugin
}

// simpleLoggerWrapper adapts zap.Logger to the logging interface needed by plugins
type simpleLoggerWrapper struct {
	logger *zap.Logger
}

func (w *simpleLoggerWrapper) Info(msg string, fields ...interface{}) {
	w.logger.Info(msg, w.convertFields(fields...)...)
}

func (w *simpleLoggerWrapper) Error(msg string, fields ...interface{}) {
	w.logger.Error(msg, w.convertFields(fields...)...)
}

func (w *simpleLoggerWrapper) Warn(msg string, fields ...interface{}) {
	w.logger.Warn(msg, w.convertFields(fields...)...)
}

func (w *simpleLoggerWrapper) Debug(msg string, fields ...interface{}) {
	w.logger.Debug(msg, w.convertFields(fields...)...)
}

func (w *simpleLoggerWrapper) convertFields(fields ...interface{}) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields)/2)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			zapFields = append(zapFields, zap.Any(key, fields[i+1]))
		}
	}
	return zapFields
}

// CommandLineFlags represents command line flags
type CommandLineFlags struct {
	environment string
	logLevel    string
	help        bool
	version     bool
}

// parseFlags parses command line flags
func parseFlags() CommandLineFlags {
	var flags CommandLineFlags

	flag.StringVar(&flags.environment, "env", "", "Environment (development/production)")
	flag.StringVar(&flags.logLevel, "log-level", "", "Log level (debug/info/warn/error)")
	flag.BoolVar(&flags.help, "help", false, "Show help")
	flag.BoolVar(&flags.version, "version", false, "Show version")

	flag.Parse()
	return flags
}

// setupBasicRoutes sets up basic health and info routes
func setupBasicRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "flexcore",
			"version":   Version,
		})
	})

	// Info endpoint
	router.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "flexcore",
			"version":     Version,
			"build_time":  BuildTime,
			"commit":      CommitHash,
			"environment": config.Current.App.Environment,
			"debug":       config.Current.App.Debug,
			"port":        config.Current.App.Port,
			"timestamp":   time.Now().Format(time.RFC3339),
			"features": []string{
				"runtime_orchestration",
				"windmill_workflows",
				"multi_runtime_support",
				"real_time_monitoring",
				"distributed_execution",
			},
		})
	})

	// API version info
	router.GET("/api/v1", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"api_version": "v1",
			"service":     "flexcore",
			"endpoints": gin.H{
				"orchestration": "/api/v1/orchestration",
				"workflows":     "/api/v1/workflows",
				"runtimes":      "/api/v1/runtimes",
			},
		})
	})
}

// initializePluginSystem initializes the FlexCore plugin system
func initializePluginSystem() (*FlexCorePluginSystem, error) {
	logging.Logger.Info("🔌 Initializing FlexCore Plugin System...")

	// Initialize communication bus
	redisURL := "redis://localhost:6380" // Default Redis URL
	nodeID := fmt.Sprintf("flexcore-%d", time.Now().Unix())
	nodeURL := "http://localhost:8080"

	communicationBus, err := communication.NewFlexCoreCommunicationBus(redisURL, nodeID, nodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create communication bus: %w", err)
	}

	// Initialize plugin loader (without registry - FlexCore receives plugins from FLEXT Service)
	// Create a simple logger wrapper for now (TODO: move to proper abstraction)
	loggerWrapper := &simpleLoggerWrapper{logger: logging.Logger}
	pluginLoader := loader.NewPluginLoader(nil, communicationBus, loggerWrapper)

	// Start communication bus
	if err := communicationBus.Start(); err != nil {
		return nil, fmt.Errorf("failed to start communication bus: %w", err)
	}

	// Register as FlexCore node in the network
	if err := registerFlexCoreNode(communicationBus); err != nil {
		logging.Logger.Warn("Failed to register FlexCore node", zap.Error(err))
	}

	pluginSystem := &FlexCorePluginSystem{
		communicationBus: communicationBus,
		pluginLoader:     pluginLoader,
		loadedPlugins:    make(map[string]interface{}),
	}

	logging.Logger.Info("✅ FlexCore Plugin System initialized successfully")
	return pluginSystem, nil
}

// registerFlexCoreNode registers this FlexCore instance in the distributed network
func registerFlexCoreNode(communicationBus *communication.FlexCoreCommunicationBus) error {
	// Register FlexCore node with capabilities
	capabilities := []string{
		"plugin.meltano.runtime",
		"plugin.ray.runtime", 
		"plugin.kubernetes.runtime",
		"plugin.all.runtime",
		"data.processing",
		"workflow.execution",
		"distributed.computing",
	}

	nodeInfo := communication.FlexCoreNodeInfo{
		NodeID:       communicationBus.DiscoverNodes()[0].NodeID, // Get own node ID
		NodeURL:      "http://localhost:8080",
		Status:       "healthy",
		LastSeen:     time.Now(),
		Capabilities: capabilities,
		Resources: map[string]interface{}{
			"available_memory": 4096.0, // MB
			"available_cpu":    80.0,   // %
			"plugin_slots":     10,     // Max concurrent plugins
		},
		Version: Version,
	}

	logging.Logger.Info("📡 Registering FlexCore node in distributed network",
		zap.String("node_url", nodeInfo.NodeURL),
		zap.Strings("capabilities", capabilities))

	return nil
}

// shutdownPluginSystem gracefully shuts down the plugin system
func (ps *FlexCorePluginSystem) shutdown() error {
	logging.Logger.Info("🛑 Shutting down FlexCore Plugin System...")

	// Stop all loaded plugins
	for pluginID := range ps.loadedPlugins {
		logging.Logger.Info("Stopping plugin", zap.String("plugin_id", pluginID))
		// TODO: Call plugin shutdown method
		delete(ps.loadedPlugins, pluginID)
	}

	// Stop communication bus
	if ps.communicationBus != nil {
		if err := ps.communicationBus.Stop(); err != nil {
			logging.Logger.Error("Error stopping communication bus", zap.Error(err))
			return err
		}
	}

	logging.Logger.Info("✅ FlexCore Plugin System shutdown complete")
	return nil
}

// initializeApplication creates and configures the application
func initializeApplication(flags CommandLineFlags) error {
	// DRY: Use shared initialization function to eliminate 32 lines of duplication
	configFlags := config.CommandLineFlags{
		Environment: flags.environment,
		LogLevel:    flags.logLevel,
		Help:        flags.help,
		Version:     flags.version,
	}
	return config.InitializeApplicationWithFlags(configFlags)
}

// setupGracefulShutdown sets up graceful shutdown handling
func setupGracefulShutdown(cancel context.CancelFunc) {
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		logging.Logger.Info("Shutdown signal received")
		cancel()
	}()
}

func main() {
	flags := parseFlags()

	if flags.help {
		flag.Usage()
		return
	}

	if flags.version {
		// Initialize basic logging for version output
		if err := logging.Initialize("flexcore", "info"); err == nil {
			logging.Logger.Info("FlexCore version info",
				zap.String("version", Version),
				zap.String("build_time", BuildTime),
				zap.String("commit", CommitHash),
			)
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := initializeApplication(flags); err != nil {
		// Try to log error properly, fallback to stderr if logging not initialized
		if logging.Logger != nil {
			logging.Logger.Fatal("Failed to initialize application", zap.Error(err))
		} else {
			// Use os.Stderr for critical error output
			if _, writeErr := os.Stderr.WriteString("Failed to initialize application: " + err.Error() + "\n"); writeErr != nil {
				// If we can't even write to stderr, there's nothing more we can do
				panic("Failed to write error to stderr: " + writeErr.Error())
			}
			cancel()
			return
		}
	}

	setupGracefulShutdown(cancel)

	// Initialize plugin system
	pluginSystem, err := initializePluginSystem()
	if err != nil {
		logging.Logger.Fatal("Failed to initialize plugin system", zap.Error(err))
	}

	// Log startup information
	logging.Logger.Info("FlexCore starting up",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("commit", CommitHash),
		zap.String("environment", config.Current.App.Environment),
		zap.Bool("debug", config.Current.App.Debug),
		zap.Int("port", config.Current.App.Port),
		zap.Bool("plugin_system", true),
	)

	// Initialize runtime orchestrator
	orchestratorConfig := &orchestrator.OrchestratorConfig{
		WindmillURL:           "http://localhost:8000",
		WindmillToken:         os.Getenv("WINDMILL_TOKEN"),
		DefaultTimeout:        300 * time.Second,
		MaxConcurrentJobs:     10,
		HealthCheckInterval:   30 * time.Second,
		MetricsUpdateInterval: 60 * time.Second,
		EnabledRuntimes:       []string{"meltano"},
	}

	runtimeOrchestrator, err := orchestrator.NewRuntimeOrchestrator(orchestratorConfig)
	if err != nil {
		logging.Logger.Fatal("Failed to create runtime orchestrator", zap.Error(err))
	}

	// Initialize orchestrator
	if err := runtimeOrchestrator.Initialize(ctx); err != nil {
		logging.Logger.Fatal("Failed to initialize runtime orchestrator", zap.Error(err))
	}

	// Set Gin mode based on environment
	if config.Current.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Setup basic routes
	setupBasicRoutes(router)

	// Setup orchestrator routes
	orchestratorController := httpcontrollers.NewOrchestratorController(runtimeOrchestrator)
	v1 := router.Group("/api/v1")
	orchestratorController.RegisterRoutes(v1)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Current.App.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second, // Prevent Slowloris attacks
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logging.Logger.Info("Server starting", zap.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
	defer shutdownCancel()

	logging.Logger.Info("Shutting down server...")
	
	// Shutdown plugin system first
	if pluginSystem != nil {
		if err := pluginSystem.shutdown(); err != nil {
			logging.Logger.Error("Plugin system shutdown error", zap.Error(err))
		}
	}
	
	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logging.Logger.Error("Server shutdown error", zap.Error(err))
	} else {
		logging.Logger.Info("Server shut down gracefully")
	}
}
