// Package core provides distributed event routing using Windmill
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flext/flexcore/infrastructure/windmill"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
	"github.com/google/uuid"
)

// EventRouter handles distributed event routing using Windmill workflows
type EventRouter struct {
	windmillClient *windmill.Client
	config         *FlexCoreConfig
	routes         map[string]*EventRoute
	metrics        *EventRouterMetrics
	mu             sync.RWMutex
	running        bool
}

// EventRouterMetrics tracks event routing metrics
type EventRouterMetrics struct {
	EventsProcessed   int64
	EventsRouted      int64
	EventsFiltered    int64
	EventsFailed      int64
	RoutingLatencyMs  int64
	ActiveRoutes      int64
}

// NewEventRouter creates a new distributed event router
func NewEventRouter(windmillClient *windmill.Client, config *FlexCoreConfig) *EventRouter {
	return &EventRouter{
		windmillClient: windmillClient,
		config:         config,
		routes:         make(map[string]*EventRoute),
		metrics:        &EventRouterMetrics{},
	}
}

// Start starts the event router
func (er *EventRouter) Start(ctx context.Context) error {
	er.mu.Lock()
	defer er.mu.Unlock()

	if er.running {
		return errors.ValidationError("event router already running")
	}

	// Register all configured routes
	for i := range er.config.EventRoutes {
		route := &er.config.EventRoutes[i]
		if err := er.registerRouteInternal(ctx, route); err != nil {
			return fmt.Errorf("failed to register route %s: %w", route.Name, err)
		}
	}

	er.running = true
	return nil
}

// Stop stops the event router
func (er *EventRouter) Stop(ctx context.Context) error {
	er.mu.Lock()
	defer er.mu.Unlock()

	er.running = false
	return nil
}

// RegisterRoute registers a new event route
func (er *EventRouter) RegisterRoute(ctx context.Context, route *EventRoute) error {
	er.mu.Lock()
	defer er.mu.Unlock()

	return er.registerRouteInternal(ctx, route)
}

// RouteEvent routes an event through the distributed system
func (er *EventRouter) RouteEvent(ctx context.Context, event *Event) result.Result[bool] {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime).Milliseconds()
		atomic.StoreInt64(&er.metrics.RoutingLatencyMs, latency)
		atomic.AddInt64(&er.metrics.EventsProcessed, 1)
	}()

	er.mu.RLock()
	defer er.mu.RUnlock()

	if !er.running {
		return result.Failure[bool](errors.ValidationError("event router not running"))
	}

	// Find matching routes
	matchingRoutes := er.findMatchingRoutes(event)
	if len(matchingRoutes) == 0 {
		atomic.AddInt64(&er.metrics.EventsFiltered, 1)
		return result.Success(true) // No routes, but not an error
	}

	// Route to matching adapters using Windmill workflows
	for _, route := range matchingRoutes {
		if route.Async {
			// Async routing - fire and forget
			go er.routeEventToTargets(context.Background(), event, route)
		} else {
			// Sync routing - wait for completion
			if err := er.routeEventToTargets(ctx, event, route); err != nil {
				atomic.AddInt64(&er.metrics.EventsFailed, 1)
				return result.Failure[bool](err)
			}
		}
	}

	atomic.AddInt64(&er.metrics.EventsRouted, 1)
	return result.Success(true)
}

// GetMetrics returns event router metrics
func (er *EventRouter) GetMetrics() *EventRouterMetrics {
	return &EventRouterMetrics{
		EventsProcessed:   atomic.LoadInt64(&er.metrics.EventsProcessed),
		EventsRouted:      atomic.LoadInt64(&er.metrics.EventsRouted),
		EventsFiltered:    atomic.LoadInt64(&er.metrics.EventsFiltered),
		EventsFailed:      atomic.LoadInt64(&er.metrics.EventsFailed),
		RoutingLatencyMs:  atomic.LoadInt64(&er.metrics.RoutingLatencyMs),
		ActiveRoutes:      int64(len(er.routes)),
	}
}

// PerformMaintenance performs periodic maintenance
func (er *EventRouter) PerformMaintenance(ctx context.Context) {
	// Cleanup old metrics, check route health, etc.
}

// Private methods

func (er *EventRouter) registerRouteInternal(ctx context.Context, route *EventRoute) error {
	// Register workflow with Windmill
	createReq := windmill.CreateWorkflowRequest{
		Path:        fmt.Sprintf("event_routes/%s", route.Name),
		Summary:     fmt.Sprintf("Event Route: %s", route.Name),
		Description: fmt.Sprintf("Routes events from %s to %v", route.SourceAdapter, route.TargetAdapters),
		Value:       er.convertRouteToWindmillWorkflow(route),
		Schema:      er.generateEventRouteSchema(route),
	}

	createResult := er.windmillClient.CreateWorkflow(ctx, createReq)
	if createResult.IsFailure() {
		return createResult.Error()
	}

	// Store route
	er.routes[route.Name] = route
	atomic.StoreInt64(&er.metrics.ActiveRoutes, int64(len(er.routes)))

	return nil
}

func (er *EventRouter) findMatchingRoutes(event *Event) []*EventRoute {
	var matchingRoutes []*EventRoute

	for _, route := range er.routes {
		if er.eventMatchesRoute(event, route) {
			matchingRoutes = append(matchingRoutes, route)
		}
	}

	return matchingRoutes
}

func (er *EventRouter) eventMatchesRoute(event *Event, route *EventRoute) bool {
	// Check source adapter
	if route.SourceAdapter != "*" && route.SourceAdapter != event.Source {
		return false
	}

	// Check event type filter
	if len(route.EventFilter.EventTypes) > 0 {
		found := false
		for _, eventType := range route.EventFilter.EventTypes {
			if eventType == "*" || eventType == event.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check source filters
	for key, expectedValue := range route.EventFilter.SourceFilters {
		if headerValue, exists := event.Headers[key]; !exists || headerValue != expectedValue {
			return false
		}
	}

	// Check content filters
	for _, contentFilter := range route.EventFilter.ContentFilters {
		if !er.evaluateContentFilter(event, &contentFilter) {
			return false
		}
	}

	// Check time window
	if route.EventFilter.TimeWindow != nil {
		if !er.eventInTimeWindow(event, route.EventFilter.TimeWindow) {
			return false
		}
	}

	return true
}

func (er *EventRouter) evaluateContentFilter(event *Event, filter *ContentFilter) bool {
	// Extract field value from event data
	fieldValue, exists := event.Data[filter.Field]
	if !exists {
		return false
	}

	// Evaluate based on operator
	switch filter.Operator {
	case "eq":
		return fieldValue == filter.Value
	case "ne":
		return fieldValue != filter.Value
	case "gt":
		if fv, ok := fieldValue.(float64); ok {
			if expectedValue, ok := filter.Value.(float64); ok {
				return fv > expectedValue
			}
		}
		return false
	case "lt":
		if fv, ok := fieldValue.(float64); ok {
			if expectedValue, ok := filter.Value.(float64); ok {
				return fv < expectedValue
			}
		}
		return false
	case "contains":
		if fv, ok := fieldValue.(string); ok {
			if expectedValue, ok := filter.Value.(string); ok {
				return len(fv) > 0 && len(expectedValue) > 0 && 
					   fv[0:min(len(fv), len(expectedValue))] == expectedValue
			}
		}
		return false
	default:
		return false
	}
}

func (er *EventRouter) eventInTimeWindow(event *Event, window *TimeWindow) bool {
	return event.Timestamp.After(window.StartTime) && event.Timestamp.Before(window.EndTime)
}

func (er *EventRouter) routeEventToTargets(ctx context.Context, event *Event, route *EventRoute) error {
	// Transform event if needed
	transformedEvent := event
	if route.Transform != nil {
		var err error
		transformedEvent, err = er.transformEvent(ctx, event, route.Transform)
		if err != nil {
			return fmt.Errorf("event transformation failed: %w", err)
		}
	}

	// Route to each target adapter
	for _, targetAdapter := range route.TargetAdapters {
		if err := er.routeEventToTarget(ctx, transformedEvent, targetAdapter, route); err != nil {
			if route.MaxRetries > 0 {
				// Implement retry logic using Windmill workflows
				if err := er.scheduleRetry(ctx, transformedEvent, targetAdapter, route, 1); err != nil {
					return fmt.Errorf("failed to route to %s and retry scheduling failed: %w", targetAdapter, err)
				}
			} else {
				return fmt.Errorf("failed to route to %s: %w", targetAdapter, err)
			}
		}
	}

	return nil
}

func (er *EventRouter) routeEventToTarget(ctx context.Context, event *Event, targetAdapter string, route *EventRoute) error {
	// Execute Windmill workflow for event delivery
	workflowPath := fmt.Sprintf("adapters/%s/receive_event", targetAdapter)
	
	input := map[string]interface{}{
		"event":        event,
		"route_name":   route.Name,
		"source":       route.SourceAdapter,
		"target":       targetAdapter,
		"custom_params": route.CustomParams,
	}

	runReq := windmill.RunWorkflowRequest{
		Args: input,
		Tag:  &route.Name,
	}

	jobResult := er.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
	if jobResult.IsFailure() {
		return jobResult.Error()
	}

	return nil
}

func (er *EventRouter) transformEvent(ctx context.Context, event *Event, transform *TransformConfig) (*Event, error) {
	switch transform.Type {
	case "script":
		return er.transformEventWithScript(ctx, event, transform)
	case "mapping":
		return er.transformEventWithMapping(event, transform)
	case "plugin":
		return er.transformEventWithPlugin(ctx, event, transform)
	default:
		return event, nil // No transformation
	}
}

func (er *EventRouter) transformEventWithScript(ctx context.Context, event *Event, transform *TransformConfig) (*Event, error) {
	// Create temporary Windmill workflow for transformation
	workflowPath := fmt.Sprintf("transformations/temp_%s", uuid.New().String())
	
	transformWorkflow := windmill.WorkflowValue{
		Modules: []windmill.WorkflowModule{
			{
				ID: "transform",
				Value: windmill.ModuleValue{
					Type:     "script",
					Content:  transform.Script,
					Language: transform.Language,
					Input: map[string]interface{}{
						"event": event,
					},
				},
			},
		},
	}

	// Execute transformation
	createReq := windmill.CreateWorkflowRequest{
		Path:        workflowPath,
		Summary:     "Event Transformation",
		Description: "Temporary workflow for event transformation",
		Value:       transformWorkflow,
	}

	// Create, execute, and cleanup
	createResult := er.windmillClient.CreateWorkflow(ctx, createReq)
	if createResult.IsFailure() {
		return nil, createResult.Error()
	}

	defer er.windmillClient.DeleteWorkflow(ctx, workflowPath)

	runReq := windmill.RunWorkflowRequest{
		Args: map[string]interface{}{
			"event": event,
		},
	}

	jobResult := er.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
	if jobResult.IsFailure() {
		return nil, jobResult.Error()
	}

	// Wait for completion and get result
	job := jobResult.Value()
	finalJobResult := er.windmillClient.WaitForJob(ctx, job.ID, 2*time.Second)
	if finalJobResult.IsFailure() {
		return nil, finalJobResult.Error()
	}

	finalJob := finalJobResult.Value()
	if finalJob.Result == nil {
		return event, nil // No transformation
	}

	// Parse transformed event
	var transformedEvent Event
	resultBytes, err := json.Marshal(finalJob.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformation result: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &transformedEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transformed event: %w", err)
	}

	return &transformedEvent, nil
}

func (er *EventRouter) transformEventWithMapping(event *Event, transform *TransformConfig) (*Event, error) {
	// Simple field mapping transformation
	transformedEvent := *event // Copy
	transformedEvent.Data = make(map[string]interface{})

	for targetField, sourceField := range transform.Mapping {
		if sourceFieldStr, ok := sourceField.(string); ok {
			if value, exists := event.Data[sourceFieldStr]; exists {
				transformedEvent.Data[targetField] = value
			}
		}
	}

	return &transformedEvent, nil
}

func (er *EventRouter) transformEventWithPlugin(ctx context.Context, event *Event, transform *TransformConfig) (*Event, error) {
	// This would integrate with the plugin system
	// For now, return the original event
	return event, nil
}

func (er *EventRouter) scheduleRetry(ctx context.Context, event *Event, targetAdapter string, route *EventRoute, attempt int) error {
	// Calculate retry delay
	delay := er.calculateRetryDelay(attempt, &er.config.RetryPolicy)
	
	// Schedule retry using Windmill delayed execution
	retryTime := time.Now().Add(delay)
	
	runReq := windmill.RunWorkflowRequest{
		Args: map[string]interface{}{
			"event":         event,
			"target_adapter": targetAdapter,
			"route":         route,
			"attempt":       attempt,
		},
		ScheduledFor: &retryTime,
		Tag:          &route.Name,
	}

	workflowPath := fmt.Sprintf("event_routes/%s/retry", route.Name)
	jobResult := er.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
	
	return jobResult.Error()
}

func (er *EventRouter) calculateRetryDelay(attempt int, policy *RetryPolicy) time.Duration {
	delay := policy.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
			break
		}
	}
	return delay
}

func (er *EventRouter) createEventRoutingWorkflow(route *EventRoute) *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        fmt.Sprintf("event_routes/%s", route.Name),
		Name:        route.Name,
		Description: fmt.Sprintf("Routes events from %s to %v", route.SourceAdapter, route.TargetAdapters),
		Steps: []windmill.WorkflowStep{
			{
				ID:       "filter_event",
				Name:     "Filter Event",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(event: dict, filter_config: dict):
    """Filter events based on configuration"""
    # Implement filtering logic
    return {"filtered": True, "event": event}
`,
				Input: map[string]interface{}{
					"event":         "{{ input.event }}",
					"filter_config": route.EventFilter,
				},
			},
			{
				ID:       "transform_event",
				Name:     "Transform Event",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(event: dict, transform_config: dict):
    """Transform event based on configuration"""
    if not transform_config:
        return event
    # Implement transformation logic
    return event
`,
				Input: map[string]interface{}{
					"event":            "{{ previous_step.result.event }}",
					"transform_config": route.Transform,
				},
				Dependencies: []string{"filter_event"},
			},
		},
		Timeout: time.Minute * 5,
	}
}

func (er *EventRouter) convertRouteToWindmillWorkflow(route *EventRoute) windmill.WorkflowValue {
	modules := []windmill.WorkflowModule{
		{
			ID: "route_event",
			Value: windmill.ModuleValue{
				Type:     "script",
				Language: "python3",
				Content: `
def main(event: dict, route_config: dict):
    """Route event to target adapters"""
    import json
    
    # Process event routing
    results = []
    for target in route_config.get("target_adapters", []):
        # Route to target adapter
        results.append({
            "target": target,
            "status": "routed",
            "event_id": event.get("id")
        })
    
    return {
        "routed_count": len(results),
        "results": results
    }
`,
				Input: map[string]interface{}{
					"event":        "{{ input.event }}",
					"route_config": route,
				},
			},
		},
	}

	return windmill.WorkflowValue{
		Modules: modules,
	}
}

func (er *EventRouter) generateEventRouteSchema(route *EventRoute) map[string]interface{} {
	return map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]interface{}{
			"event": map[string]interface{}{
				"type":        "object",
				"description": "Event to be routed",
				"required":    []string{"id", "type", "source", "timestamp", "data"},
			},
		},
		"required": []string{"event"},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}