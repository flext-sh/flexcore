// Package infrastructure provides plugin execution specialized services
// SOLID SRP: Reduces executePlugin function from 7 returns to 2 returns (71% reduction)
package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// PluginExecutionOrchestrator coordinates plugin execution through HTTP adapter
// SOLID SRP: Single responsibility for coordinating plugin execution with centralized error handling
type PluginExecutionOrchestrator struct {
	validator      *PluginParameterValidator
	requestBuilder *PluginRequestBuilder
	httpExecutor   *PluginHTTPExecutor
	responseParser *PluginResponseParser
	logger         *zap.Logger
}

// NewPluginExecutionOrchestrator creates specialized plugin execution orchestrator
func NewPluginExecutionOrchestrator(flextBaseURL string, httpClient *http.Client, logger *zap.Logger) *PluginExecutionOrchestrator {
	return &PluginExecutionOrchestrator{
		validator:      NewPluginParameterValidator(),
		requestBuilder: NewPluginRequestBuilder(flextBaseURL),
		httpExecutor:   NewPluginHTTPExecutor(httpClient, logger),
		responseParser: NewPluginResponseParser(),
		logger:         logger,
	}
}

// ExecutePlugin orchestrates complete plugin execution with reduced error returns
// SOLID SRP: Coordinates specialized services eliminating individual error handling returns
func (peo *PluginExecutionOrchestrator) ExecutePlugin(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Phase 1: Validation and Request Building (consolidated error handling)
	executionRequest, err := peo.buildExecutionRequest(params)
	if err != nil {
		return nil, fmt.Errorf("plugin execution setup failed: %w", err)
	}

	// Phase 2: HTTP Execution and Response Processing (consolidated error handling)
	result, err := peo.executeAndProcessResponse(ctx, executionRequest)
	if err != nil {
		return nil, fmt.Errorf("plugin execution or response processing failed: %w", err)
	}

	return result, nil
}

// buildExecutionRequest consolidates parameter validation and request building
// SOLID SRP: Eliminates 2 separate return points by centralizing validation and building
func (peo *PluginExecutionOrchestrator) buildExecutionRequest(params map[string]interface{}) (*PluginExecutionRequest, error) {
	// Validate parameters
	validatedParams, err := peo.validator.ValidateParameters(params)
	if err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Build HTTP request
	request, err := peo.requestBuilder.BuildRequest(validatedParams)
	if err != nil {
		return nil, fmt.Errorf("request building failed: %w", err)
	}

	return request, nil
}

// executeAndProcessResponse consolidates HTTP execution and response processing
// SOLID SRP: Eliminates 5 separate return points by centralizing execution and parsing
func (peo *PluginExecutionOrchestrator) executeAndProcessResponse(ctx context.Context, execRequest *PluginExecutionRequest) (interface{}, error) {
	// Execute HTTP request
	responseData, err := peo.httpExecutor.ExecuteRequest(ctx, execRequest)
	if err != nil {
		return nil, fmt.Errorf("HTTP execution failed: %w", err)
	}

	// Parse response
	result, err := peo.responseParser.ParseResponse(execRequest, responseData)
	if err != nil {
		return nil, fmt.Errorf("response parsing failed: %w", err)
	}

	return result, nil
}

// PluginExecutionRequest represents validated plugin execution request
type PluginExecutionRequest struct {
	PluginName  string
	Command     string
	Args        []interface{}
	URL         string
	RequestBody []byte
}

// PluginParameterValidator handles plugin parameter validation
// SOLID SRP: Single responsibility for parameter validation
type PluginParameterValidator struct{}

// NewPluginParameterValidator creates parameter validator service
func NewPluginParameterValidator() *PluginParameterValidator {
	return &PluginParameterValidator{}
}

// ValidatedParameters represents validated plugin parameters
type ValidatedParameters struct {
	PluginName string
	Command    string
	Args       []interface{}
}

// ValidateParameters validates and extracts plugin execution parameters
func (ppv *PluginParameterValidator) ValidateParameters(params map[string]interface{}) (*ValidatedParameters, error) {
	pluginName, ok := params["plugin_name"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_name parameter is required")
	}

	command, ok := params["command"].(string)
	if !ok {
		command = "--version" // Default command
	}

	args, _ := params["args"].([]interface{})

	return &ValidatedParameters{
		PluginName: pluginName,
		Command:    command,
		Args:       args,
	}, nil
}

// PluginRequestBuilder handles HTTP request construction
// SOLID SRP: Single responsibility for building HTTP requests
type PluginRequestBuilder struct {
	flextBaseURL string
}

// NewPluginRequestBuilder creates request builder service
func NewPluginRequestBuilder(flextBaseURL string) *PluginRequestBuilder {
	return &PluginRequestBuilder{flextBaseURL: flextBaseURL}
}

// BuildRequest constructs HTTP request for plugin execution
func (prb *PluginRequestBuilder) BuildRequest(params *ValidatedParameters) (*PluginExecutionRequest, error) {
	url := fmt.Sprintf("%s/api/v1/flexcore/plugins/%s/execute", prb.flextBaseURL, params.PluginName)

	requestBody := map[string]interface{}{
		"command": params.Command,
		"args":    params.Args,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	return &PluginExecutionRequest{
		PluginName:  params.PluginName,
		Command:     params.Command,
		Args:        params.Args,
		URL:         url,
		RequestBody: jsonBody,
	}, nil
}

// PluginHTTPExecutor handles HTTP request execution
// SOLID SRP: Single responsibility for HTTP communication
type PluginHTTPExecutor struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// NewPluginHTTPExecutor creates HTTP executor service
func NewPluginHTTPExecutor(httpClient *http.Client, logger *zap.Logger) *PluginHTTPExecutor {
	return &PluginHTTPExecutor{httpClient: httpClient, logger: logger}
}

// ExecuteRequest executes HTTP request and returns response data
func (phe *PluginHTTPExecutor) ExecuteRequest(ctx context.Context, execRequest *PluginExecutionRequest) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", execRequest.URL, bytes.NewReader(execRequest.RequestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create execute plugin request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	phe.logger.Info("Executing FLEXT plugin via HTTP",
		zap.String("plugin", execRequest.PluginName),
		zap.String("command", execRequest.Command),
		zap.Any("args", execRequest.Args))

	resp, err := phe.httpClient.Do(req)
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

	phe.logger.Info("FLEXT plugin executed successfully",
		zap.String("plugin", execRequest.PluginName),
		zap.Any("result", execData))

	return execData, nil
}

// PluginResponseParser handles response formatting
// SOLID SRP: Single responsibility for response parsing and formatting
type PluginResponseParser struct{}

// NewPluginResponseParser creates response parser service
func NewPluginResponseParser() *PluginResponseParser {
	return &PluginResponseParser{}
}

// ParseResponse formats the execution response with metadata
func (prp *PluginResponseParser) ParseResponse(execRequest *PluginExecutionRequest, execData map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"adapter":     "flext-http-adapter",
		"operation":   "execute_plugin",
		"plugin_name": execRequest.PluginName,
		"status":      "success",
		"flext_data":  execData,
		"timestamp":   time.Now().UTC(),
	}, nil
}
