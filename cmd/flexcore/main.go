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

	// Log startup information
	logging.Logger.Info("FlexCore starting up",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("commit", CommitHash),
		zap.String("environment", config.Current.App.Environment),
		zap.Bool("debug", config.Current.App.Debug),
		zap.Int("port", config.Current.App.Port),
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
	if err := server.Shutdown(shutdownCtx); err != nil {
		logging.Logger.Error("Server shutdown error", zap.Error(err))
	} else {
		logging.Logger.Info("Server shut down gracefully")
	}
}
