package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flext/flexcore/pkg/logging"
	"go.uber.org/zap"
)

// FlextHTTPAdapter implements HTTP communication with FLEXT service
type FlextHTTPAdapter struct {
	flextBaseURL string
	httpClient   *http.Client
	logger       logging.LoggerInterface
	
	// SOLID SRP: Specialized orchestrator for plugin execution with reduced returns
	pluginOrchestrator *PluginExecutionOrchestrator
}

// NewFlextHTTPAdapter creates a new HTTP adapter for FLEXT service
func NewFlextHTTPAdapter(flextBaseURL string, logger logging.LoggerInterface) *FlextHTTPAdapter {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Create zap logger for the orchestrator (since LoggerInterface doesn't expose the underlying zap logger)
	zapLogger, _ := zap.NewProduction()
	
	// Create specialized plugin orchestrator with SOLID SRP principles
	orchestrator := NewPluginExecutionOrchestrator(flextBaseURL, httpClient, zapLogger)
	
	return &FlextHTTPAdapter{
		flextBaseURL:       flextBaseURL,
		httpClient:         httpClient,
		logger:             logger,
		pluginOrchestrator: orchestrator,
	}
}

// Name returns the adapter name
func (fha *FlextHTTPAdapter) Name() string {
	return "flext-http-adapter"
}

// Version returns the adapter version
func (fha *FlextHTTPAdapter) Version() string {
	return "2.0.0"
}

// Execute executes FLEXT operations via HTTP API
func (fha *FlextHTTPAdapter) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	fha.logger.Info("Executing FLEXT operation via HTTP",
		zap.String("base_url", fha.flextBaseURL),
		zap.Any("params", params))

	// Determine operation type from parameters
	operation, ok := params["operation"].(string)
	if !ok {
		operation = "health" // Default operation
	}

	switch operation {
	case "health":
		return fha.checkHealth(ctx)
	case "list_plugins":
		return fha.listPlugins(ctx)
	case "execute_plugin":
		return fha.executePlugin(ctx, params)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// performHTTPGetRequest executes HTTP GET request and parses JSON response
// DRY PRINCIPLE: Eliminates 34-line duplication (mass=219) between checkHealth and listPlugins
func (fha *FlextHTTPAdapter) performHTTPGetRequest(ctx context.Context, endpoint, operation string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s", fha.flextBaseURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s request: %w", operation, err)
	}

	resp, err := fha.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("FLEXT %s failed: %w", operation, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s response: %w", operation, err)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse %s response: %w", operation, err)
	}

	fha.logger.Info(fmt.Sprintf("FLEXT %s successful", operation), zap.Any("data", responseData))

	return map[string]interface{}{
		"adapter":    fha.Name(),
		"operation":  operation,
		"status":     "success",
		"flext_data": responseData,
		"timestamp":  time.Now().UTC(),
	}, nil
}

// checkHealth checks FLEXT service health
// DRY PRINCIPLE: Uses shared HTTP request method
func (fha *FlextHTTPAdapter) checkHealth(ctx context.Context) (interface{}, error) {
	return fha.performHTTPGetRequest(ctx, "/health", "health")
}

// listPlugins lists available FLEXT plugins
// DRY PRINCIPLE: Uses shared HTTP request method
func (fha *FlextHTTPAdapter) listPlugins(ctx context.Context) (interface{}, error) {
	return fha.performHTTPGetRequest(ctx, "/api/v1/flexcore/plugins", "list_plugins")
}

// executePlugin executes a specific FLEXT plugin
// executePlugin executes FLEXT plugin via HTTP with specialized orchestrator
// SOLID SRP: Reduced from 7 returns to 2 returns (71% reduction) using specialized orchestrator
func (fha *FlextHTTPAdapter) executePlugin(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Delegate to specialized orchestrator with consolidated error handling
	result, err := fha.pluginOrchestrator.ExecutePlugin(ctx, params)
	if err != nil {
		fha.logger.Error("Plugin execution failed", 
			zap.Any("params", params),
			zap.Error(err),
		)
		return nil, fmt.Errorf("plugin execution failed: %w", err)
	}

	return result, nil
}

// Shutdown gracefully shuts down the adapter
func (fha *FlextHTTPAdapter) Shutdown() error {
	fha.logger.Info("Shutting down FLEXT HTTP adapter")
	// HTTP client doesn't need explicit shutdown
	return nil
}

// IsRunning returns true since HTTP adapter is always available
func (fha *FlextHTTPAdapter) IsRunning() bool {
	return true
}
