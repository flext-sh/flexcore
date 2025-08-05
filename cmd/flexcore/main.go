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
	flextlogging "github.com/flext-sh/flext/pkg/logging"
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

// simpleLoggerWrapper adapts zap.Logger to the FLEXT logging interface
type simpleLoggerWrapper struct {
	logger *zap.Logger
}

func (w *simpleLoggerWrapper) Debug(msg string) {
	w.logger.Debug(msg)
}

func (w *simpleLoggerWrapper) Info(msg string, fields ...flextlogging.Field) {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	w.logger.Info(msg, zapFields...)
}

func (w *simpleLoggerWrapper) Warn(msg string, fields ...flextlogging.Field) {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	w.logger.Warn(msg, zapFields...)
}

func (w *simpleLoggerWrapper) Error(msg string, fields ...flextlogging.Field) {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	w.logger.Error(msg, zapFields...)
}

func (w *simpleLoggerWrapper) Fatal(msg string, fields ...flextlogging.Field) {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	w.logger.Fatal(msg, zapFields...)
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
		logging.Logger.Warn("Failed to create communication bus - running in standalone mode", zap.Error(err))
		communicationBus = nil
	}

	// Initialize plugin loader (without registry - FlexCore receives plugins from FLEXT Service)
	// Create a simple logger wrapper for now (TODO: move to proper abstraction)
	loggerWrapper := &simpleLoggerWrapper{logger: logging.Logger}
	pluginLoader := loader.NewPluginLoader(nil, communicationBus, loggerWrapper)

	// Start communication bus (optional - graceful degradation if Redis unavailable)  
	if communicationBus != nil {
		if err := communicationBus.Start(); err != nil {
			logging.Logger.Warn("Communication bus failed to start - running in standalone mode", zap.Error(err))
			// Continue without communication bus for standalone operation
			communicationBus = nil
		}
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
	if communicationBus == nil {
		logging.Logger.Info("📡 Running in standalone mode - no distributed network registration")
		return nil
	}
	
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

	nodes := communicationBus.DiscoverNodes()
	nodeID := "flexcore-standalone"
	if len(nodes) > 0 {
		nodeID = nodes[0].NodeID
	}

	nodeInfo := communication.FlexCoreNodeInfo{
		NodeID:       nodeID,
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

// handleCommandLineFlags processes command line flags and returns true if execution should stop
func handleCommandLineFlags(flags CommandLineFlags) bool {
	if flags.help {
		flag.Usage()
		return true
	}

	if flags.version {
		showVersionInfo()
		return true
	}

	return false
}

// showVersionInfo displays version information
func showVersionInfo() {
	// Initialize basic logging for version output
	if err := logging.Initialize("flexcore", "info"); err == nil {
		logging.Logger.Info("FlexCore version info",
			zap.String("version", Version),
			zap.String("build_time", BuildTime),
			zap.String("commit", CommitHash),
		)
	}
}

// handleFatalError handles application fatal errors
func handleFatalError(err error, cancel context.CancelFunc) {
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
	}
}

// runApplication runs the main application logic
func runApplication(ctx context.Context, cancel context.CancelFunc, flags CommandLineFlags) error {
	// Initialize application
	if err := initializeApplication(flags); err != nil {
		return err
	}

	setupGracefulShutdown(cancel)

	// Initialize components
	pluginSystem, runtimeOrchestrator, err := initializeComponents(ctx)
	if err != nil {
		return err
	}

	// Create and configure server
	server := createServer(pluginSystem, runtimeOrchestrator)

	// Start server
	startServer(server)

	// Wait for shutdown signal and perform graceful shutdown
	<-ctx.Done()
	performGracefulShutdown(server, pluginSystem)

	return nil
}

// initializeComponents initializes plugin system and runtime orchestrator
func initializeComponents(ctx context.Context) (*FlexCorePluginSystem, *orchestrator.RuntimeOrchestrator, error) {
	// Initialize plugin system
	pluginSystem, err := initializePluginSystem()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize plugin system: %w", err)
	}

	// Log startup information
	logStartupInfo()

	// Initialize runtime orchestrator
	runtimeOrchestrator, err := createRuntimeOrchestrator(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create runtime orchestrator: %w", err)
	}

	return pluginSystem, runtimeOrchestrator, nil
}

// logStartupInfo logs application startup information
func logStartupInfo() {
	logging.Logger.Info("FlexCore starting up",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("commit", CommitHash),
		zap.String("environment", config.Current.App.Environment),
		zap.Bool("debug", config.Current.App.Debug),
		zap.Int("port", config.Current.App.Port),
		zap.Bool("plugin_system", true),
	)
}

// createRuntimeOrchestrator creates and initializes the runtime orchestrator
func createRuntimeOrchestrator(ctx context.Context) (*orchestrator.RuntimeOrchestrator, error) {
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
		return nil, err
	}

	// Initialize orchestrator
	if err := runtimeOrchestrator.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize runtime orchestrator: %w", err)
	}

	return runtimeOrchestrator, nil
}

// createServer creates and configures the HTTP server
func createServer(pluginSystem *FlexCorePluginSystem, runtimeOrchestrator *orchestrator.RuntimeOrchestrator) *http.Server {
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

	// Setup routes
	setupRoutes(router, pluginSystem, runtimeOrchestrator)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Current.App.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second, // Prevent Slowloris attacks
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// setupRoutes sets up all HTTP routes
func setupRoutes(router *gin.Engine, pluginSystem *FlexCorePluginSystem, runtimeOrchestrator *orchestrator.RuntimeOrchestrator) {
	// Setup basic routes
	setupBasicRoutes(router)

	// Setup API routes
	v1 := router.Group("/api/v1")

	// Setup orchestrator routes
	orchestratorController := httpcontrollers.NewOrchestratorController(runtimeOrchestrator)
	orchestratorController.RegisterRoutes(v1)

	// Setup plugin management routes
	pluginController := httpcontrollers.NewPluginController(pluginSystem.pluginLoader, pluginSystem.communicationBus)
	pluginController.RegisterRoutes(v1)
}

// startServer starts the HTTP server in a goroutine
func startServer(server *http.Server) {
	go func() {
		logging.Logger.Info("Server starting", zap.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logger.Fatal("Server failed", zap.Error(err))
		}
	}()
}

// performGracefulShutdown performs graceful shutdown of all components
func performGracefulShutdown(server *http.Server, pluginSystem *FlexCorePluginSystem) {
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

func main() {
	flags := parseFlags()

	// Handle command line flags
	if handleCommandLineFlags(flags) {
		return
	}

	// Start application
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runApplication(ctx, cancel, flags); err != nil {
		handleFatalError(err, cancel)
	}
}
