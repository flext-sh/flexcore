// Package app provides the application layer implementation
package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flext/flexcore/pkg/config"
	"github.com/flext/flexcore/pkg/logging"
	"go.uber.org/zap"
)

// Application represents the FlexCore application
type Application struct {
	config *config.Config
	server *http.Server
	mux    *http.ServeMux
}

// NewApplication creates a new application instance
func NewApplication(cfg *config.Config) (*Application, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.App.Environment == "" {
		return nil, fmt.Errorf("environment is required")
	}

	// Initialize logging if not already done
	if logging.Logger == nil {
		logLevel := "info"
		if cfg.App.Debug {
			logLevel = "debug"
		}
		if err := logging.Initialize(cfg.App.Environment, logLevel); err != nil {
			return nil, fmt.Errorf("failed to initialize logging: %w", err)
		}
	}

	// Create HTTP server
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	app := &Application{
		config: cfg,
		server: server,
		mux:    mux,
	}

	// Setup routes
	app.setupRoutes()

	return app, nil
}

// setupRoutes configures HTTP routes
func (a *Application) setupRoutes() {
	a.mux.HandleFunc("/health", a.handleHealth)
	a.mux.HandleFunc("/", a.handleRoot)
}

// handleHealth provides health check endpoint
func (a *Application) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// handleRoot provides basic API information
func (a *Application) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"flexcore","version":"%s","environment":"%s"}`, a.config.App.Version, a.config.App.Environment)
}

// Start starts the application
func (a *Application) Start(ctx context.Context) error {
	logging.Logger.Info("Starting FlexCore application",
		zap.String("address", a.server.Addr),
		zap.String("environment", a.config.App.Environment),
	)

	// Start server in goroutine
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logger.Error("Server failed", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the application gracefully
func (a *Application) Stop(ctx context.Context) error {
	logging.Logger.Info("Stopping FlexCore application")
	return a.server.Shutdown(ctx)
}

// Config returns the application configuration
func (a *Application) Config() *config.Config {
	return a.config
}
