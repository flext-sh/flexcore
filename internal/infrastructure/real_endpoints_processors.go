package infrastructure

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/flext/flexcore/internal/application/services"
	"github.com/gin-gonic/gin"
)

// WorkflowDataProcessor processes workflow-related operations
// SOLID SRP: Single responsibility for workflow data processing
type WorkflowDataProcessor struct {
	eventStore services.EventStore
	nodeID     string
}

// NewWorkflowDataProcessor creates a new workflow data processor
func NewWorkflowDataProcessor(eventStore services.EventStore, nodeID string) *WorkflowDataProcessor {
	return &WorkflowDataProcessor{
		eventStore: eventStore,
		nodeID:     nodeID,
	}
}

// ListWorkflows retrieves and processes workflow data from events
func (wdp *WorkflowDataProcessor) ListWorkflows(ctx context.Context) ([]map[string]interface{}, error) {
	events, err := wdp.eventStore.GetAllEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	workflows := make(map[string]map[string]interface{})
	for _, event := range events {
		workflowID := wdp.extractWorkflowID(event)
		if workflowID != "" {
			if _, exists := workflows[workflowID]; !exists {
				workflows[workflowID] = map[string]interface{}{
					"id":         workflowID,
					"status":     "active",
					"events":     0,
					"created_at": event.Timestamp,
				}
			}
			workflow := workflows[workflowID]
			workflow["events"] = workflow["events"].(int) + 1
			workflow["updated_at"] = event.Timestamp
		}
	}

	var result []map[string]interface{}
	for _, workflow := range workflows {
		result = append(result, workflow)
	}

	return result
}

// extractWorkflowID extracts workflow ID from event data
// SOLID SRP: Single responsibility for workflow ID extraction
func (wdp *WorkflowDataProcessor) extractWorkflowID(event EventEntry) string {
	if data, ok := event.Data.(map[string]interface{}); ok {
		if pipelineID, exists := data["PipelineID"]; exists {
			return pipelineID.(string)
		}
	}
	return ""
}

// EventQueryProcessor processes event-related operations
// SOLID SRP: Single responsibility for event query processing
type EventQueryProcessor struct {
	eventStore services.EventStore
	nodeID     string
}

// NewEventQueryProcessor creates a new event query processor
func NewEventQueryProcessor(eventStore services.EventStore, nodeID string) *EventQueryProcessor {
	return &EventQueryProcessor{
		eventStore: eventStore,
		nodeID:     nodeID,
	}
}

// GetEventsWithLimit retrieves events with specified limit
func (eqp *EventQueryProcessor) GetEventsWithLimit(ctx context.Context, limit int) (gin.H, error) {
	allEvents, err := eqp.eventStore.GetAllEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Apply limit
	events := allEvents
	if len(allEvents) > limit {
		events = allEvents[:limit]
	}

	return gin.H{
		"events":    events,
		"count":     len(events),
		"limit":     limit,
		"node_id":   eqp.nodeID,
		"timestamp": time.Now().UTC(),
	}, nil
}

// PluginExecutionProcessor processes plugin execution operations
// SOLID SRP: Single responsibility for plugin execution processing
type PluginExecutionProcessor struct {
	pluginLoader *HashicorpStyleLoader
	nodeID       string
}

// NewPluginExecutionProcessor creates a new plugin execution processor
func NewPluginExecutionProcessor(pluginLoader *HashicorpStyleLoader, nodeID string) *PluginExecutionProcessor {
	return &PluginExecutionProcessor{
		pluginLoader: pluginLoader,
		nodeID:       nodeID,
	}
}

// ProcessPluginExecution handles plugin execution with proper error handling
func (pep *PluginExecutionProcessor) ProcessPluginExecution(ctx context.Context, pluginName string, params map[string]interface{}) (gin.H, error) {
	plugin, err := pep.pluginLoader.LoadPlugin(pluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %s: %w", pluginName, err)
	}

	if p, ok := plugin.(Plugin); ok {
		result, err := p.Process(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("plugin execution failed: %w", err)
		}

		return gin.H{
			"plugin":    pluginName,
			"result":    result,
			"status":    "completed",
			"timestamp": time.Now().UTC(),
			"node_id":   pep.nodeID,
		}, nil
	}

	return nil, fmt.Errorf("plugin %s does not implement required interface", pluginName)
}

// ClusterStatusProcessor processes cluster-related operations
// SOLID SRP: Single responsibility for cluster status processing
type ClusterStatusProcessor struct {
	coordinator services.CoordinationLayer
}

// NewClusterStatusProcessor creates a new cluster status processor
func NewClusterStatusProcessor(coordinator services.CoordinationLayer) *ClusterStatusProcessor {
	return &ClusterStatusProcessor{
		coordinator: coordinator,
	}
}

// GetClusterStatus retrieves comprehensive cluster status information
func (csp *ClusterStatusProcessor) GetClusterStatus(ctx context.Context) gin.H {
	return gin.H{
		"status":      "active",
		"leader":      csp.coordinator.GetNodeID(),
		"nodes":       1,
		"timestamp":   time.Now().UTC(),
		"version":     "2.0.0",
		"node_id":     csp.coordinator.GetNodeID(),
	}
}

// pluginExecutionRequest represents a plugin execution request
type pluginExecutionRequest struct {
	PluginName string                 `json:"plugin_name"`
	Params     map[string]interface{} `json:"params"`
}

// Plugin interface for consistent plugin handling
type Plugin interface {
	Name() string
	Version() string
	Process(ctx context.Context, data interface{}) (interface{}, error)
}

// EventEntry represents an event in the event store
type EventEntry struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// PluginListProcessor processes plugin listing operations
// SOLID SRP: Single responsibility for plugin listing and metadata processing
type PluginListProcessor struct {
	pluginLoader services.PluginLoader
	coordinator  services.CoordinationLayer
}

// NewPluginListProcessor creates a new plugin list processor
func NewPluginListProcessor(pluginLoader services.PluginLoader, coordinator services.CoordinationLayer) *PluginListProcessor {
	return &PluginListProcessor{
		pluginLoader: pluginLoader,
		coordinator:  coordinator,
	}
}

// ListPlugins retrieves and processes all available plugins with metadata
// SOLID SRP: Centralized plugin listing logic eliminating duplication
func (plp *PluginListProcessor) ListPlugins(ctx context.Context) (map[string]interface{}, error) {
	pluginNames := plp.pluginLoader.ListPlugins()

	var plugins []map[string]interface{}
	for _, name := range pluginNames {
		plugin, err := plp.pluginLoader.LoadPlugin(name)
		if err != nil {
			continue
		}

		if p, ok := plugin.(Plugin); ok {
			plugins = append(plugins, map[string]interface{}{
				"name":    p.Name(),
				"version": p.Version(),
				"status":  "loaded",
			})
		}
	}

	return map[string]interface{}{
		"plugins": plugins,
		"count":   len(plugins),
		"node_id": plp.coordinator.GetNodeID(),
	}, nil
}