// Package server - HTTP Server implementation for FlexCore
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flext-sh/flexcore/pkg/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	config     *config.Config
	logger     *zap.Logger
	router     *gin.Engine
	httpServer *http.Server
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, logger *zap.Logger) *Server {
	// Set gin mode based on environment
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	return &Server{
		config: cfg,
		logger: logger,
		router: router,
	}
}

// SetupBasicRoutes sets up basic health check routes
func (s *Server) SetupBasicRoutes() {
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/api/v1/health", s.healthCheck)
}

// RegisterHandler registers a handler (placeholder)
func (s *Server) RegisterHandler(handler interface{}) {
	// TODO: Implement handler registration
	s.logger.Info("Handler registered", zap.String("handler", fmt.Sprintf("%T", handler)))
}

// RegisterCleanHandler registers a clean handler (placeholder)
func (s *Server) RegisterCleanHandler(name string, handler interface{}) {
	// TODO: Implement clean handler registration
	s.logger.Info("Clean handler registered", zap.String("name", name), zap.String("handler", fmt.Sprintf("%T", handler)))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	s.logger.Info("Starting HTTP server", zap.String("address", addr))
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.Info("Stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "flexcore",
		"version":   s.config.App.Version,
	})
}
