package infrastructure

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flext/flexcore/internal/application/services"
	"github.com/flext/flexcore/pkg/logging"
	"github.com/gin-gonic/gin"
)

// HealthCheckStrategy defines strategy for component health checking
// SOLID Strategy Pattern: Reduces complexity by extracting health check logic
type HealthCheckStrategy interface {
	CheckHealth(ctx context.Context) (status string, data map[string]interface{}, err error)
	ComponentName() string
}

// WorkflowServiceHealthCheck implements health check for workflow service
type WorkflowServiceHealthCheck struct{}

func (w *WorkflowServiceHealthCheck) CheckHealth(ctx context.Context) (string, map[string]interface{}, error) {
	return "healthy", nil, nil
}

func (w *WorkflowServiceHealthCheck) ComponentName() string {
	return "workflow_service"
}

// PluginSystemHealthCheck implements health check for plugin system
type PluginSystemHealthCheck struct {
	pluginLoader *HashicorpStyleLoader
}

func (p *PluginSystemHealthCheck) CheckHealth(ctx context.Context) (string, map[string]interface{}, error) {
	if p.pluginLoader != nil {
		pluginCount := len(p.pluginLoader.ListPlugins())
		if pluginCount > 0 {
			return "healthy", map[string]interface{}{"plugin_count": pluginCount}, nil
		}
		return "no_plugins", map[string]interface{}{"plugin_count": 0}, nil
	}
	return "unavailable", nil, nil
}

func (p *PluginSystemHealthCheck) ComponentName() string {
	return "plugin_system"
}

// EventStoreHealthCheck implements health check for event store
type EventStoreHealthCheck struct {
	eventStore services.EventStore
}

func (e *EventStoreHealthCheck) CheckHealth(ctx context.Context) (string, map[string]interface{}, error) {
	if eventCount, err := e.getEventStoreHealth(ctx); err != nil {
		return "unhealthy", nil, err
	} else {
		return "healthy", map[string]interface{}{"event_count": eventCount}, nil
	}
}

func (e *EventStoreHealthCheck) ComponentName() string {
	return "event_store"
}

func (e *EventStoreHealthCheck) getEventStoreHealth(ctx context.Context) (int64, error) {
	if postgresStore, ok := e.eventStore.(*PostgreSQLEventStore); ok {
		return postgresStore.GetEventCount(ctx)
	}
	return 0, nil
}

// ClusterCoordinatorHealthCheck implements health check for cluster coordinator
type ClusterCoordinatorHealthCheck struct{}

func (c *ClusterCoordinatorHealthCheck) CheckHealth(ctx context.Context) (string, map[string]interface{}, error) {
	return "healthy", nil, nil
}

func (c *ClusterCoordinatorHealthCheck) ComponentName() string {
	return "cluster_coordinator"
}

// HealthCheckOrchestrator orchestrates multiple health check strategies
// SOLID Composite Pattern: Manages multiple health check strategies
type HealthCheckOrchestrator struct {
	strategies []HealthCheckStrategy
	logger     logging.LoggerInterface
}

// NewHealthCheckOrchestrator creates a new health check orchestrator
func NewHealthCheckOrchestrator(
	pluginLoader *HashicorpStyleLoader,
	eventStore services.EventStore,
	logger logging.LoggerInterface,
) *HealthCheckOrchestrator {
	return &HealthCheckOrchestrator{
		strategies: []HealthCheckStrategy{
			&WorkflowServiceHealthCheck{},
			&PluginSystemHealthCheck{pluginLoader: pluginLoader},
			&EventStoreHealthCheck{eventStore: eventStore},
			&ClusterCoordinatorHealthCheck{},
		},
		logger: logger,
	}
}

// CheckAllComponents checks health of all components
func (hco *HealthCheckOrchestrator) CheckAllComponents(ctx context.Context) gin.H {
	overallStatus := "healthy"
	components := make(map[string]map[string]interface{})
	
	for _, strategy := range hco.strategies {
		status, data, err := strategy.CheckHealth(ctx)
		componentInfo := map[string]interface{}{
			"status": status,
		}
		
		if data != nil {
			componentInfo["data"] = data
		}
		
		if err != nil {
			componentInfo["error"] = err.Error()
			overallStatus = "degraded"
			hco.logger.Warn(fmt.Sprintf("Health check failed for %s: %v", strategy.ComponentName(), err))
		}
		
		components[strategy.ComponentName()] = componentInfo
	}
	
	return gin.H{
		"status":     overallStatus,
		"timestamp":  time.Now().UTC(),
		"components": components,
	}
}

// HealthEndpointHandler handles health check endpoints
type HealthEndpointHandler struct {
	orchestrator *HealthCheckOrchestrator
	logger       logging.LoggerInterface
}

// NewHealthEndpointHandler creates a new health endpoint handler
func NewHealthEndpointHandler(orchestrator *HealthCheckOrchestrator, logger logging.LoggerInterface) *HealthEndpointHandler {
	return &HealthEndpointHandler{
		orchestrator: orchestrator,
		logger:       logger,
	}
}

// HandleHealthCheck handles comprehensive health check requests
func (heh *HealthEndpointHandler) HandleHealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	result := heh.orchestrator.CheckAllComponents(ctx)
	
	status := http.StatusOK
	if result["status"] == "degraded" {
		status = http.StatusServiceUnavailable
	}
	
	c.JSON(status, result)
}