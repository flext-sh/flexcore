package infrastructure

import (
	"bytes"
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
}

// NewFlextHTTPAdapter creates a new HTTP adapter for FLEXT service
func NewFlextHTTPAdapter(flextBaseURL string, logger logging.LoggerInterface) *FlextHTTPAdapter {
	return &FlextHTTPAdapter{
		flextBaseURL: flextBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
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

// checkHealth checks FLEXT service health
func (fha *FlextHTTPAdapter) checkHealth(ctx context.Context) (interface{}, error) {
	url := fmt.Sprintf("%s/health", fha.flextBaseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := fha.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("FLEXT health check failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read health response: %w", err)
	}

	var healthData map[string]interface{}
	if err := json.Unmarshal(body, &healthData); err != nil {
		return nil, fmt.Errorf("failed to parse health response: %w", err)
	}

	fha.logger.Info("FLEXT health check successful", zap.Any("health", healthData))

	return map[string]interface{}{
		"adapter":     fha.Name(),
		"operation":   "health",
		"status":      "success",
		"flext_data":  healthData,
		"timestamp":   time.Now().UTC(),
	}, nil
}

// listPlugins lists available FLEXT plugins
func (fha *FlextHTTPAdapter) listPlugins(ctx context.Context) (interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/flexcore/plugins", fha.flextBaseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list plugins request: %w", err)
	}

	resp, err := fha.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("FLEXT list plugins failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins response: %w", err)
	}

	var pluginsData map[string]interface{}
	if err := json.Unmarshal(body, &pluginsData); err != nil {
		return nil, fmt.Errorf("failed to parse plugins response: %w", err)
	}

	fha.logger.Info("FLEXT plugins listed successfully", zap.Any("plugins", pluginsData))

	return map[string]interface{}{
		"adapter":     fha.Name(),
		"operation":   "list_plugins",
		"status":      "success",
		"flext_data":  pluginsData,
		"timestamp":   time.Now().UTC(),
	}, nil
}

// executePlugin executes a specific FLEXT plugin
func (fha *FlextHTTPAdapter) executePlugin(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	pluginName, ok := params["plugin_name"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_name parameter is required")
	}

	command, ok := params["command"].(string)
	if !ok {
		command = "--version" // Default command
	}

	args, _ := params["args"].([]interface{})

	url := fmt.Sprintf("%s/api/v1/flexcore/plugins/%s/execute", fha.flextBaseURL, pluginName)
	
	requestBody := map[string]interface{}{
		"command": command,
		"args":    args,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create execute plugin request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	fha.logger.Info("Executing FLEXT plugin via HTTP", 
		zap.String("plugin", pluginName),
		zap.String("command", command),
		zap.Any("args", args))

	resp, err := fha.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("FLEXT plugin execution failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read execution response: %w", err)
	}

	var execData map[string]interface{}
	if err := json.Unmarshal(body, &execData); err != nil {
		return nil, fmt.Errorf("failed to parse execution response: %w", err)
	}

	fha.logger.Info("FLEXT plugin executed successfully", 
		zap.String("plugin", pluginName),
		zap.Any("result", execData))

	return map[string]interface{}{
		"adapter":      fha.Name(),
		"operation":    "execute_plugin",
		"plugin_name":  pluginName,
		"status":       "success",
		"flext_data":   execData,
		"timestamp":    time.Now().UTC(),
	}, nil
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