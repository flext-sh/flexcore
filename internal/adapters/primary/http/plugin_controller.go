// Package http - FlexCore Plugin Management HTTP Controller
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	
	"github.com/flext-sh/flext/pkg/plugins"
	"github.com/flext-sh/flext/pkg/plugins/communication"
	"github.com/flext-sh/flext/pkg/plugins/loader"
	flextlogging "github.com/flext-sh/flext/pkg/logging"
)

// PluginController handles HTTP requests for plugin management
type PluginController struct {
	pluginLoader     *loader.PluginLoader
	communicationBus *communication.FlexCoreCommunicationBus
	loadedPlugins    map[string]plugins.Plugin
	logger           *zap.Logger
}

// PluginDeploymentRequest represents a plugin deployment request from FLEXT Service
type PluginDeploymentRequest struct {
	PluginID     string                 `json:"plugin_id" binding:"required"`
	PluginType   string                 `json:"plugin_type" binding:"required"`
	BinaryData   []byte                 `json:"binary_data,omitempty"`
	BinaryURL    string                 `json:"binary_url,omitempty"`
	Config       map[string]interface{} `json:"config"`
	SourceNodeID string                 `json:"source_node_id" binding:"required"`
}

// PluginExecutionRequest represents a plugin execution request
type PluginExecutionRequest struct {
	PluginID  string                 `json:"plugin_id" binding:"required"`
	Operation string                 `json:"operation" binding:"required"`
	Params    map[string]interface{} `json:"params"`
}

// NewPluginController creates a new plugin controller
func NewPluginController(pluginLoader *loader.PluginLoader, communicationBus *communication.FlexCoreCommunicationBus) *PluginController {
	logger, _ := zap.NewProduction()
	return &PluginController{
		pluginLoader:     pluginLoader,
		communicationBus: communicationBus,
		loadedPlugins:    make(map[string]plugins.Plugin),
		logger:           logger,
	}
}

// RegisterRoutes registers plugin management routes
func (pc *PluginController) RegisterRoutes(router *gin.RouterGroup) {
	pluginRoutes := router.Group("/plugins")
	{
		// Plugin management endpoints
		pluginRoutes.GET("/list", pc.listPlugins)
		pluginRoutes.POST("/deploy", pc.deployPlugin)
		pluginRoutes.DELETE("/:plugin_id", pc.unloadPlugin)
		pluginRoutes.GET("/:plugin_id/health", pc.getPluginHealth)
		pluginRoutes.GET("/:plugin_id/metadata", pc.getPluginMetadata)
		
		// Plugin execution endpoints
		pluginRoutes.POST("/:plugin_id/execute", pc.executePlugin)
		pluginRoutes.GET("/:plugin_id/capabilities", pc.getPluginCapabilities)
		
		// Plugin communication endpoints
		pluginRoutes.POST("/broadcast", pc.broadcastMessage)
		pluginRoutes.GET("/communication/status", pc.getCommunicationStatus)
	}
	
	// FlexCore node management
	nodeRoutes := router.Group("/node")
	{
		nodeRoutes.GET("/info", pc.getNodeInfo)
		nodeRoutes.GET("/status", pc.getNodeStatus)
		nodeRoutes.POST("/register", pc.registerNode)
	}
}

// listPlugins handles GET /api/v1/plugins/list
func (pc *PluginController) listPlugins(c *gin.Context) {
	pluginList := make([]gin.H, 0, len(pc.loadedPlugins))
	
	for pluginID, plugin := range pc.loadedPlugins {
		metadata := plugin.Metadata()
		pluginList = append(pluginList, gin.H{
			"plugin_id":    pluginID,
			"name":         metadata.Name,
			"type":         metadata.Type,
			"version":      metadata.Version,
			"description":  metadata.Description,
			"author":       metadata.Author,
			"capabilities": metadata.Capabilities,
			"dependencies": metadata.Dependencies,
			"created_at":   metadata.CreatedAt,
			"updated_at":   metadata.UpdatedAt,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"plugins": pluginList,
		"count":   len(pluginList),
		"node_id": pc.getNodeID(),
	})
}

// deployPlugin handles POST /api/v1/plugins/deploy
func (pc *PluginController) deployPlugin(c *gin.Context) {
	var request PluginDeploymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		pc.logger.Warn("Invalid plugin deployment request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	pc.logger.Info("Received plugin deployment request",
		zap.String("plugin_id", request.PluginID),
		zap.String("plugin_type", request.PluginType),
		zap.String("source_node_id", request.SourceNodeID))
	
	// Check if plugin is already loaded
	if _, exists := pc.loadedPlugins[request.PluginID]; exists {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Plugin already deployed",
			"plugin_id": request.PluginID,
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	var plugin plugins.Plugin
	var err error
	
	// Load plugin from binary data or binary path
	if len(request.BinaryData) > 0 {
		// For binary data, we would need to write it to a temporary file first
		// For now, return an error as this needs more implementation
		err = fmt.Errorf("loading plugins from binary data not yet implemented")
	} else if request.BinaryURL != "" {
		// For binary URL, we would need to download and then load
		// For now, treat as binary path
		plugin, err = pc.pluginLoader.LoadPluginFromBinary(ctx, request.BinaryURL, request.Config)
	} else {
		// Try to load from factory (for built-in plugins)
		plugin, err = pc.pluginLoader.CreatePluginFromFactory(plugins.PluginType(request.PluginType))
		if err == nil && plugin != nil {
			// Set communicator if supported
			if communicatorSetter, ok := plugin.(interface{ SetCommunicator(plugins.PluginCommunicator) }); ok {
				communicatorSetter.SetCommunicator(pc.communicationBus)
			}
			// Initialize the plugin
			err = plugin.Initialize(ctx, request.Config)
		}
	}
	
	if err != nil {
		pc.logger.Error("Failed to load plugin",
			zap.String("plugin_id", request.PluginID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load plugin",
			"details": err.Error(),
		})
		return
	}
	
	// Store loaded plugin
	pc.loadedPlugins[request.PluginID] = plugin
	
	// Notify FLEXT Service about successful deployment
	if pc.communicationBus != nil {
		message := map[string]interface{}{
			"operation":      "plugin.deployed",
			"plugin_id":      request.PluginID,
			"plugin_type":    request.PluginType,
			"flexcore_node":  pc.getNodeID(),
			"deployment_time": time.Now(),
			"status":         "success",
		}
		
		go func() {
			err := pc.communicationBus.SendMessage(ctx, 
				plugins.PluginID(pc.getNodeID()), 
				plugins.PluginID(request.SourceNodeID), 
				message)
			if err != nil {
				pc.logger.Warn("Failed to notify plugin deployment", zap.Error(err))
			}
		}()
	}
	
	metadata := plugin.Metadata()
	pc.logger.Info("Plugin deployed successfully",
		zap.String("plugin_id", request.PluginID),
		zap.String("plugin_name", metadata.Name),
		zap.String("version", metadata.Version))
	
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"plugin_id":     request.PluginID,
		"plugin_name":   metadata.Name,
		"plugin_type":   metadata.Type,
		"version":       metadata.Version,
		"capabilities":  metadata.Capabilities,
		"deployed_at":   time.Now(),
		"node_id":       pc.getNodeID(),
		"message":       "Plugin deployed successfully",
	})
}

// unloadPlugin handles DELETE /api/v1/plugins/:plugin_id
func (pc *PluginController) unloadPlugin(c *gin.Context) {
	pluginID := c.Param("plugin_id")
	if pluginID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Plugin ID is required",
		})
		return
	}
	
	plugin, exists := pc.loadedPlugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "Plugin not found",
			"plugin_id": pluginID,
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Shutdown plugin gracefully
	if err := plugin.Shutdown(ctx); err != nil {
		pc.logger.Warn("Plugin shutdown error",
			zap.String("plugin_id", pluginID),
			zap.Error(err))
	}
	
	// Remove from loaded plugins
	delete(pc.loadedPlugins, pluginID)
	
	pc.logger.Info("Plugin unloaded successfully",
		zap.String("plugin_id", pluginID))
	
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"plugin_id":  pluginID,
		"unloaded_at": time.Now(),
		"message":    "Plugin unloaded successfully",
	})
}

// getPluginHealth handles GET /api/v1/plugins/:plugin_id/health
func (pc *PluginController) getPluginHealth(c *gin.Context) {
	pluginID := c.Param("plugin_id")
	plugin, exists := pc.loadedPlugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "Plugin not found",
			"plugin_id": pluginID,
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	health, err := plugin.Health(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "Failed to check plugin health",
			"plugin_id": pluginID,
			"details":   err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"plugin_id": pluginID,
		"health":    health,
		"checked_at": time.Now(),
	})
}

// getPluginMetadata handles GET /api/v1/plugins/:plugin_id/metadata
func (pc *PluginController) getPluginMetadata(c *gin.Context) {
	pluginID := c.Param("plugin_id")
	plugin, exists := pc.loadedPlugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "Plugin not found",
			"plugin_id": pluginID,
		})
		return
	}
	
	metadata := plugin.Metadata()
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"plugin_id": pluginID,
		"metadata": metadata,
	})
}

// executePlugin handles POST /api/v1/plugins/:plugin_id/execute
func (pc *PluginController) executePlugin(c *gin.Context) {
	pluginID := c.Param("plugin_id")
	plugin, exists := pc.loadedPlugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "Plugin not found",
			"plugin_id": pluginID,
		})
		return
	}
	
	var request PluginExecutionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid execution request",
			"details": err.Error(),
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	pc.logger.Info("Executing plugin operation",
		zap.String("plugin_id", pluginID),
		zap.String("operation", request.Operation))
	
	result, err := plugin.Execute(ctx, request.Operation, request.Params)
	if err != nil {
		pc.logger.Error("Plugin execution failed",
			zap.String("plugin_id", pluginID),
			zap.String("operation", request.Operation),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "Plugin execution failed",
			"plugin_id": pluginID,
			"operation": request.Operation,
			"details":   err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"plugin_id":  pluginID,
		"operation":  request.Operation,
		"result":     result,
		"executed_at": time.Now(),
	})
}

// getPluginCapabilities handles GET /api/v1/plugins/:plugin_id/capabilities
func (pc *PluginController) getPluginCapabilities(c *gin.Context) {
	pluginID := c.Param("plugin_id")
	plugin, exists := pc.loadedPlugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "Plugin not found",
			"plugin_id": pluginID,
		})
		return
	}
	
	capabilities := plugin.GetCapabilities()
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"plugin_id":    pluginID,
		"capabilities": capabilities,
		"count":        len(capabilities),
	})
}

// broadcastMessage handles POST /api/v1/plugins/broadcast
func (pc *PluginController) broadcastMessage(c *gin.Context) {
	var request struct {
		PluginType string                 `json:"plugin_type" binding:"required"`
		Message    map[string]interface{} `json:"message" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid broadcast request",
			"details": err.Error(),
		})
		return
	}
	
	if pc.communicationBus == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Communication bus not available",
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err := pc.communicationBus.BroadcastMessage(ctx, 
		plugins.PluginID(pc.getNodeID()), 
		plugins.PluginType(request.PluginType), 
		request.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to broadcast message",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"plugin_type": request.PluginType,
		"broadcasted_at": time.Now(),
		"message":     "Message broadcasted successfully",
	})
}

// getCommunicationStatus handles GET /api/v1/plugins/communication/status  
func (pc *PluginController) getCommunicationStatus(c *gin.Context) {
	if pc.communicationBus == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Communication bus not available",
		})
		return
	}
	
	nodes := pc.communicationBus.DiscoverNodes()
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"communication_bus": gin.H{
			"status":      "active",
			"node_count":  len(nodes),
			"nodes":       nodes,
			"local_node":  pc.getNodeID(),
		},
		"checked_at": time.Now(),
	})
}

// getNodeInfo handles GET /api/v1/node/info
func (pc *PluginController) getNodeInfo(c *gin.Context) {
	nodeInfo := gin.H{
		"node_id":       pc.getNodeID(),
		"node_type":     "flexcore",
		"version":       "2.0.0",
		"capabilities": []string{
			"plugin.deployment",
			"plugin.execution", 
			"plugin.management",
			"inter.flexcore.communication",
			"distributed.coordination",
		},
		"loaded_plugins": len(pc.loadedPlugins),
		"status":         "healthy",
		"uptime":         time.Now(), // Would be calculated from start time
	}
	
	if pc.communicationBus != nil {
		nodeInfo["communication_status"] = "connected"
		nodeInfo["discovered_nodes"] = len(pc.communicationBus.DiscoverNodes())
	} else {
		nodeInfo["communication_status"] = "disconnected"
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"node":    nodeInfo,
	})
}

// getNodeStatus handles GET /api/v1/node/status
func (pc *PluginController) getNodeStatus(c *gin.Context) {
	status := gin.H{
		"node_id":        pc.getNodeID(),
		"status":         "healthy",
		"loaded_plugins": len(pc.loadedPlugins),
		"timestamp":      time.Now(),
	}
	
	// Add plugin health summary
	healthySummary := gin.H{
		"total":     len(pc.loadedPlugins),
		"healthy":   0,
		"unhealthy": 0,
		"unknown":   0,
	}
	
	for _, plugin := range pc.loadedPlugins {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		health, err := plugin.Health(ctx)
		cancel()
		
		if err != nil {
			healthySummary["unknown"] = healthySummary["unknown"].(int) + 1
		} else if status, ok := health["status"].(string); ok && status == "healthy" {
			healthySummary["healthy"] = healthySummary["healthy"].(int) + 1
		} else {
			healthySummary["unhealthy"] = healthySummary["unhealthy"].(int) + 1
		}
	}
	
	status["plugin_health"] = healthySummary
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  status,
	})
}

// registerNode handles POST /api/v1/node/register
func (pc *PluginController) registerNode(c *gin.Context) {
	var request struct {
		NodeID       string                 `json:"node_id"`
		NodeURL      string                 `json:"node_url"`
		Capabilities []string               `json:"capabilities"`
		Metadata     map[string]interface{} `json:"metadata"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid registration request",
			"details": err.Error(),
		})
		return
	}
	
	// This would register the node with the distributed FlexCore network
	// For now, just return success
	pc.logger.Info("Node registration request received",
		zap.String("node_id", request.NodeID),
		zap.String("node_url", request.NodeURL))
	
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"node_id":       request.NodeID,
		"registered_at": time.Now(),
		"message":       "Node registration acknowledged",
	})
}

// Helper methods

// getNodeID returns the current FlexCore node ID
func (pc *PluginController) getNodeID() string {
	if pc.communicationBus != nil {
		nodes := pc.communicationBus.DiscoverNodes()
		if len(nodes) > 0 {
			return nodes[0].NodeID
		}
	}
	return "flexcore-local"
}

// simpleLoggerWrapper adapts zap.Logger to FLEXT logging interface for plugins
type simpleLoggerWrapper struct {
	logger *zap.Logger
}

func (w *simpleLoggerWrapper) Debug(msg string) {
	w.logger.Debug(msg)
}

func (w *simpleLoggerWrapper) Info(msg string, fields ...flextlogging.Field) {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	w.logger.Info(msg, zapFields...)
}

func (w *simpleLoggerWrapper) Warn(msg string, fields ...flextlogging.Field) {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	w.logger.Warn(msg, zapFields...)
}

func (w *simpleLoggerWrapper) Error(msg string, fields ...flextlogging.Field) {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	w.logger.Error(msg, zapFields...)
}

func (w *simpleLoggerWrapper) Fatal(msg string, fields ...flextlogging.Field) {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	w.logger.Fatal(msg, zapFields...)
}