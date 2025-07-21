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
			os.Stderr.WriteString("Failed to initialize application: " + err.Error() + "\n")
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

	// Create simple HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Current.App.Port),
		Handler: mux,
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
