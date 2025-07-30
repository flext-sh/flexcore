// Package http contains tests for the HTTP adapter
package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flext/flexcore/pkg/config"
	"github.com/flext/flexcore/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig() *config.Config {
	return &config.Config{
		App: struct {
			Name        string `mapstructure:"name"`
			Version     string `mapstructure:"version"`
			Environment string `mapstructure:"environment"`
			Debug       bool   `mapstructure:"debug"`
			Port        int    `mapstructure:"port"`
		}{
			Name:        "flexcore",
			Version:     "0.9.0",
			Environment: "test",
			Debug:       true,
			Port:        8080,
		},
		Server: struct {
			Host         string        `mapstructure:"host"`
			Port         int           `mapstructure:"port"`
			ReadTimeout  time.Duration `mapstructure:"read_timeout"`
			WriteTimeout time.Duration `mapstructure:"write_timeout"`
			IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
		}{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func TestNewServer(t *testing.T) {
	// Initialize logging
	err := logging.Initialize("test", "debug")
	require.NoError(t, err)

	cfg := setupTestConfig()
	server := NewServer(cfg)

	assert.NotNil(t, server)
	assert.Equal(t, cfg, server.config)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.mux)
}

func TestServer_HandleHealth(t *testing.T) {
	// Initialize logging
	err := logging.Initialize("test", "debug")
	require.NoError(t, err)

	cfg := setupTestConfig()
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestServer_HandleRoot(t *testing.T) {
	// Initialize logging
	err := logging.Initialize("test", "debug")
	require.NoError(t, err)

	cfg := setupTestConfig()
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "flexcore")
	assert.Contains(t, w.Body.String(), "test")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestServer_HandleReady(t *testing.T) {
	// Initialize logging
	err := logging.Initialize("test", "debug")
	require.NoError(t, err)

	cfg := setupTestConfig()
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/ready", http.NoBody)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestServer_HandlePipelinesList(t *testing.T) {
	// Initialize logging
	err := logging.Initialize("test", "debug")
	require.NoError(t, err)

	cfg := setupTestConfig()
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/api/v1/pipelines", http.NoBody)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pipelines")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestServer_HandlePipelinesCreate(t *testing.T) {
	// Initialize logging
	err := logging.Initialize("test", "debug")
	require.NoError(t, err)

	cfg := setupTestConfig()
	server := NewServer(cfg)

	// Test with valid JSON body
	body := `{"name":"test","description":"test pipeline"}`
	req := httptest.NewRequest("POST", "/api/v1/pipelines", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Contains(t, w.Body.String(), "not yet implemented")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestServer_StartStop(t *testing.T) {
	// Initialize logging
	err := logging.Initialize("test", "debug")
	require.NoError(t, err)

	cfg := setupTestConfig()
	cfg.Server.Port = 8082 // Use different port for testing

	server := NewServer(cfg)

	ctx := context.Background()

	// Test start
	err = server.Start(ctx)
	assert.NoError(t, err)

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test stop
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = server.Stop(stopCtx)
	assert.NoError(t, err)
}

func TestServer_MethodNotAllowed(t *testing.T) {
	// Initialize logging
	err := logging.Initialize("test", "debug")
	require.NoError(t, err)

	cfg := setupTestConfig()
	server := NewServer(cfg)

	req := httptest.NewRequest("DELETE", "/api/v1/pipelines", http.NoBody)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
