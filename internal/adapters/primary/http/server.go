// Package http provides HTTP adapter for FlexCore
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flext/flexcore/pkg/config"
	"github.com/flext/flexcore/pkg/logging"
	"go.uber.org/zap"
)

// Server represents the HTTP server adapter
type Server struct {
	config *config.Config
	server *http.Server
	mux    *http.ServeMux
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config) *Server {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	s := &Server{
		config: cfg,
		server: server,
		mux:    mux,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/", s.handleRoot)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/ready", s.handleReady)
	s.mux.HandleFunc("/api/v1/pipelines", s.handlePipelines)
}

// handleRoot provides API information
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service":     "flexcore",
		"version":     s.config.App.Version,
		"environment": s.config.App.Environment,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleHealth provides health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleReady provides readiness check
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handlePipelines provides pipeline API
func (s *Server) handlePipelines(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPipelines(w, r)
	case http.MethodPost:
		s.createPipeline(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listPipelines lists all pipelines
func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters for filtering/pagination
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")
	filter := r.URL.Query().Get("filter")

	response := map[string]interface{}{
		"pipelines": []interface{}{},
		"count":     0,
		"timestamp": time.Now().Format(time.RFC3339),
		"params": map[string]string{
			"limit":  limit,
			"offset": offset,
			"filter": filter,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// createPipeline creates a new pipeline
func (s *Server) createPipeline(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"message":   "Pipeline creation not yet implemented",
		"request":   req,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	logging.Logger.Info("Starting HTTP server",
		zap.String("address", s.server.Addr),
	)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logger.Error("HTTP server failed", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the HTTP server gracefully
func (s *Server) Stop(ctx context.Context) error {
	logging.Logger.Info("Stopping HTTP server")
	return s.server.Shutdown(ctx)
}
