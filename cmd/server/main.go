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

	"go.uber.org/zap"

	"github.com/flext/flexcore/internal/application/services"
	"github.com/flext/flexcore/internal/infrastructure"
	"github.com/flext/flexcore/pkg/config"
	"github.com/flext/flexcore/pkg/logging"
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

// initializeApplication creates and configures the application
func initializeApplication(flags CommandLineFlags) error {
	// Initialize configuration
	if err := config.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Override environment if provided
	if flags.environment != "" {
		config.V.Set("app.environment", flags.environment)
		config.Current.App.Environment = flags.environment
	}

	// Determine log level
	logLevel := flags.logLevel
	if logLevel == "" {
		if config.Current.App.Debug {
			logLevel = "debug"
		} else {
			logLevel = "info"
		}
	}

	// Initialize logging
	if err := logging.Initialize(config.Current.App.Environment, logLevel); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	// Enable config hot reloading
	config.Watch()

	return nil
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
	// Parse command line flags
	flags := parseFlags()
	
	// Show help or version if requested
	if flags.help {
		flag.Usage()
		return
	}
	
	if flags.version {
		fmt.Printf("FlexCore Server v%s\nBuild: %s\nCommit: %s\n", Version, BuildTime, CommitHash)
		return
	}
	
	// Initialize application
	if err := initializeApplication(flags); err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Setup graceful shutdown
	setupGracefulShutdown(cancel)
	
	// 1. Initialize FLEXCORE distributed runtime with real persistence
	useRealPersistence := config.Current.App.Environment == "production"
	flexcoreContainer := infrastructure.NewFlexCoreContainer(useRealPersistence)
	if err := flexcoreContainer.Initialize(ctx); err != nil {
		logging.Logger.Fatal("Failed to initialize FLEXCORE container", zap.Error(err))
	}
	
	// 2. Setup event sourcing + CQRS
	eventStore := flexcoreContainer.GetEventStore()
	commandBus := flexcoreContainer.GetCommandBus()
	_ = flexcoreContainer.GetQueryBus() // Not used in this implementation
	
	// 3. Setup plugin system for FLEXT service
	pluginLoader := infrastructure.NewHashicorpStyleLoader()
	
	// 4. Register FLEXT service as proxy adapter in FLEXCORE (pragmatic implementation)
	logger := logging.NewLogger("flext-plugin")
	flextPlugin := infrastructure.NewFlextProxyAdapter(logger)
	pluginLoader.RegisterPlugin("flext-service", flextPlugin)
	
	// 5. Setup distributed cluster coordination (real Redis or fallback)
	var cluster services.CoordinationLayer
	redisLogger := logging.NewLogger("redis-coordinator")
	
	if config.GetString("redis.url") != "" {
		// Use real Redis coordinator
		cluster = infrastructure.NewRealRedisCoordinator(config.GetString("redis.url"), redisLogger)
		logging.Logger.Info("Using real Redis coordinator")
	} else {
		// Fallback to simulated coordinator for development
		cluster = infrastructure.NewRedisCoordinator()
		logging.Logger.Info("Using simulated Redis coordinator (development mode)")
	}
	
	if err := cluster.Start(ctx); err != nil {
		logging.Logger.Fatal("Failed to start cluster coordinator", zap.Error(err))
	}
	defer cluster.Stop()
	
	// 6. Initialize workflow engine with FLEXT service
	workflowLogger := logging.NewLogger("workflow-service")
	workflowService := services.NewWorkflowService(
		flexcoreContainer.GetEventBus(),
		pluginLoader,
		cluster,
		eventStore,
		commandBus,
		workflowLogger,
	)
	
	// 7. Start FLEXCORE container with real endpoints
	server := infrastructure.NewRealFlexcoreServer(workflowService, pluginLoader, cluster, eventStore)
	logging.Logger.Info("Starting FLEXCORE container with real FLEXT service...")
	
	// Start server in a goroutine
	go func() {
		if err := server.Start(":8080"); err != nil && err != http.ErrServerClosed {
			logging.Logger.Fatal("FLEXCORE container failed to start", zap.Error(err))
		}
	}()
	
	// Wait for shutdown signal
	<-ctx.Done()
	
	// Graceful shutdown
	logging.Logger.Info("Shutting down FLEXCORE container...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeoutSeconds)*time.Second)
	defer shutdownCancel()
	
	// Stop server
	if err := server.Stop(shutdownCtx); err != nil {
		logging.Logger.Error("Server shutdown error", zap.Error(err))
	}
	
	// Stop plugin loader
	if err := pluginLoader.Shutdown(); err != nil {
		logging.Logger.Error("Plugin loader shutdown error", zap.Error(err))
	}
	
	// Stop container
	if err := flexcoreContainer.Shutdown(shutdownCtx); err != nil {
		logging.Logger.Error("Container shutdown error", zap.Error(err))
	}
	
	logging.Logger.Info("FLEXCORE container shutdown complete")
}
