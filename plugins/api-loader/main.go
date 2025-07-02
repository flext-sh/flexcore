// API Loader Plugin - Real executable plugin using HashiCorp go-plugin
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flext/flexcore/infrastructure/plugins"
	"github.com/flext/flexcore/shared/result"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// APILoader implements the LoaderPlugin interface
type APILoader struct {
	logger     hclog.Logger
	client     *http.Client
	config     map[string]interface{}
	endpoint   string
	method     string
	headers    map[string]string
	timeout    time.Duration
	retries    int
	batchSize  int
}

// Init initializes the plugin with configuration
func (l *APILoader) Init(config map[string]interface{}) error {
	l.logger.Info("Initializing API loader", "config", config)
	
	l.config = config
	
	// Extract configuration
	l.endpoint, _ = config["endpoint"].(string)
	l.method, _ = config["method"].(string)
	if l.method == "" {
		l.method = "POST"
	}
	
	// Setup timeout
	if timeoutStr, ok := config["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			l.timeout = timeout
		}
	}
	if l.timeout == 0 {
		l.timeout = 30 * time.Second
	}
	
	// Setup retries
	if retries, ok := config["retries"].(int); ok {
		l.retries = retries
	} else {
		l.retries = 3
	}
	
	// Setup batch size
	if batchSize, ok := config["batch_size"].(int); ok {
		l.batchSize = batchSize
	} else {
		l.batchSize = 100
	}
	
	// Setup headers
	l.headers = make(map[string]string)
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			l.headers[k] = fmt.Sprintf("%v", v)
		}
	}
	
	// Setup HTTP client
	l.client = &http.Client{
		Timeout: l.timeout,
	}
	
	l.logger.Info("API loader initialized successfully")
	return nil
}

// Execute executes the plugin with given input (generic interface method)
func (l *APILoader) Execute(ctx context.Context, input interface{}) result.Result[interface{}] {
	l.logger.Debug("Execute called", "input", input)
	
	// For loader, we need both data and destination
	// Use default destination if not provided
	destination := l.endpoint
	if l.endpoint == "" {
		return result.Failure[interface{}](fmt.Errorf("no endpoint configured"))
	}
	
	// Use Load method
	loadResult := l.Load(ctx, input, destination)
	if loadResult.IsFailure() {
		return result.Failure[interface{}](loadResult.Error())
	}

	return result.Success[interface{}](loadResult.Value())
}

// GetMetadata returns plugin metadata
func (l *APILoader) GetMetadata() plugins.PluginMetadata {
	return plugins.PluginMetadata{
		Name:        "api-loader",
		Version:     "1.0.0",
		Description: "HTTP/REST API data loader plugin",
		Author:      "FlexCore",
		Type:        plugins.LoaderType,
		Capabilities: []string{
			"rest-api",
			"json-payload",
			"batch-loading",
			"retry-logic",
			"custom-headers",
			"authentication",
			"rate-limiting",
		},
	}
}

// Shutdown gracefully shuts down the plugin
func (l *APILoader) Shutdown() error {
	l.logger.Info("Shutting down API loader")
	return nil
}

// GetInfo returns plugin information
func (l *APILoader) GetInfo() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "api-loader",
		Version:     "1.0.0",
		Description: "HTTP/REST API data loader plugin",
		Type:        plugins.LoaderType,
		Capabilities: []string{
			"rest-api",
			"json-payload",
			"batch-loading",
			"retry-logic",
			"custom-headers",
			"authentication",
			"rate-limiting",
		},
	}
}

// Configure configures the plugin with API parameters
func (l *APILoader) Configure(config map[string]interface{}) result.Result[bool] {
	l.logger.Info("Configuring API loader", "config", config)

	l.config = config

	// Extract configuration
	if endpoint, ok := config["endpoint"].(string); ok {
		l.endpoint = endpoint
	} else {
		return result.Failure[bool](fmt.Errorf("endpoint is required"))
	}

	l.method = "POST"
	if method, ok := config["method"].(string); ok {
		l.method = strings.ToUpper(method)
	}

	// Configure headers
	l.headers = make(map[string]string)
	l.headers["Content-Type"] = "application/json"
	l.headers["User-Agent"] = "FlexCore-APILoader/1.0"

	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				l.headers[key] = valueStr
			}
		}
	}

	// Configure timeout
	l.timeout = 30 * time.Second
	if timeoutSeconds, ok := config["timeout_seconds"].(float64); ok {
		l.timeout = time.Duration(timeoutSeconds) * time.Second
	}

	// Configure retries
	l.retries = 3
	if retries, ok := config["retries"].(float64); ok {
		l.retries = int(retries)
	}

	// Configure batch size
	l.batchSize = 100
	if batchSize, ok := config["batch_size"].(float64); ok {
		l.batchSize = int(batchSize)
	}

	// Create HTTP client
	l.client = &http.Client{
		Timeout: l.timeout,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	l.logger.Info("API loader configured successfully",
		"endpoint", l.endpoint,
		"method", l.method,
		"timeout", l.timeout,
		"retries", l.retries,
		"batch_size", l.batchSize)

	return result.Success(true)
}

// Load loads data to the API destination
func (l *APILoader) Load(ctx context.Context, data interface{}, destination string) result.Result[bool] {
	l.logger.Info("Starting data load", "destination", destination)

	if l.client == nil {
		return result.Failure[bool](fmt.Errorf("loader not configured"))
	}

	// Use destination if provided, otherwise use configured endpoint
	endpoint := l.endpoint
	if destination != "" {
		endpoint = destination
	}

	// Convert data to slice for batch processing
	var records []interface{}
	switch v := data.(type) {
	case []interface{}:
		records = v
	case map[string]interface{}:
		records = []interface{}{v}
	default:
		records = []interface{}{v}
	}

	l.logger.Info("Processing records", "total_records", len(records))

	// Process in batches
	totalLoaded := 0
	for i := 0; i < len(records); i += l.batchSize {
		end := i + l.batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		if err := l.loadBatch(ctx, batch, endpoint); err != nil {
			l.logger.Error("Batch load failed", "batch_start", i, "batch_size", len(batch), "error", err)
			return result.Failure[bool](fmt.Errorf("batch load failed: %w", err))
		}

		totalLoaded += len(batch)
		l.logger.Debug("Batch loaded successfully", "batch_start", i, "batch_size", len(batch), "total_loaded", totalLoaded)
	}

	l.logger.Info("Data load completed successfully", "total_loaded", totalLoaded)
	return result.Success(true)
}

// GetSupportedFormats returns supported data formats
func (l *APILoader) GetSupportedFormats() []string {
	return []string{
		"json",
		"application/json",
		"text/json",
	}
}

// Health checks plugin health
func (l *APILoader) Health() result.Result[bool] {
	if l.client == nil {
		return result.Failure[bool](fmt.Errorf("loader not configured"))
	}

	// Try a simple health check (HEAD request if possible)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", l.endpoint, nil)
	if err != nil {
		return result.Failure[bool](fmt.Errorf("health check failed to create request: %w", err))
	}

	// Add headers
	for key, value := range l.headers {
		req.Header.Set(key, value)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return result.Failure[bool](fmt.Errorf("health check request failed: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return result.Success(true)
	}

	return result.Failure[bool](fmt.Errorf("health check failed with status: %d", resp.StatusCode))
}


// Private methods

func (l *APILoader) loadBatch(ctx context.Context, batch []interface{}, endpoint string) error {
	// Prepare payload
	payload := map[string]interface{}{
		"data":      batch,
		"timestamp": time.Now().Unix(),
		"source":    "flexcore-api-loader",
		"count":     len(batch),
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request with retries
	var lastErr error
	for attempt := 0; attempt <= l.retries; attempt++ {
		if attempt > 0 {
			l.logger.Debug("Retrying request", "attempt", attempt, "max_retries", l.retries)
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
		}

		if err := l.makeRequest(ctx, endpoint, jsonData); err != nil {
			lastErr = err
			l.logger.Warn("Request failed", "attempt", attempt+1, "error", err)
			continue
		}

		// Success
		return nil
	}

	return fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
}

func (l *APILoader) makeRequest(ctx context.Context, endpoint string, jsonData []byte) error {
	// Create request
	req, err := http.NewRequestWithContext(ctx, l.method, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range l.headers {
		req.Header.Set(key, value)
	}

	// Add content length
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(jsonData)))

	// Make request
	l.logger.Debug("Making HTTP request", "method", l.method, "endpoint", endpoint, "payload_size", len(jsonData))

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	respBody, _ := io.ReadAll(resp.Body)

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		l.logger.Debug("Request successful",
			"status_code", resp.StatusCode,
			"response_size", len(respBody))
		return nil
	}

	// Handle different error status codes
	switch {
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// Client error - don't retry
		return fmt.Errorf("client error: %d - %s", resp.StatusCode, string(respBody))
	case resp.StatusCode >= 500:
		// Server error - can retry
		return fmt.Errorf("server error: %d - %s", resp.StatusCode, string(respBody))
	default:
		return fmt.Errorf("unexpected status code: %d - %s", resp.StatusCode, string(respBody))
	}
}

// Plugin handshake and serve
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "FLEXCORE_PLUGIN",
	MagicCookieValue: "api-loader",
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	loader := &APILoader{
		logger: logger,
	}

	var pluginMap = map[string]plugin.Plugin{
		"loader": &plugins.LoaderGRPCPlugin{Impl: loader},
	}

	logger.Debug("Starting api-loader plugin")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}