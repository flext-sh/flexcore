package infrastructure

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CommonEndpointProcessor implements SOLID DRY pattern to eliminate code duplication
// Consolidates common endpoint response patterns identified by qlty smells (mass=144)
type CommonEndpointProcessor struct{}

// NewCommonEndpointProcessor creates a new processor for common endpoint patterns
// SOLID REFACTORING: Extract Method pattern to eliminate 25-line duplications
func NewCommonEndpointProcessor() *CommonEndpointProcessor {
	return &CommonEndpointProcessor{}
}

// ProcessPluginListRequest implements DRY pattern for plugin listing endpoints
// Eliminates identical 25-line code blocks in real_endpoints.go and real_endpoints_refactored.go
func (cep *CommonEndpointProcessor) ProcessPluginListRequest(
	c *gin.Context,
	pluginLoader *HashicorpStyleLoader,
) {
	// SOLID Extract Method: Common plugin list processing logic
	pluginNames := pluginLoader.ListPlugins()

	var plugins []map[string]interface{}
	for _, name := range pluginNames {
		plugin, err := pluginLoader.LoadPlugin(name)
		if err != nil {
			// Log error but continue with other plugins
			continue
		}

		// Create consistent plugin representation
		pluginInfo := map[string]interface{}{
			"name":        name,
			"status":      "loaded",
			"type":        "flexcore",
			"version":     "1.0.0", // Could be extracted from plugin metadata
			"description": "FlexCore managed plugin",
		}

		if plugin != nil {
			pluginInfo["active"] = true
		} else {
			pluginInfo["active"] = false
		}

		plugins = append(plugins, pluginInfo)
	}

	// SOLID DRY: Consistent response format
	cep.SendSuccessResponse(c, map[string]interface{}{
		"plugins": plugins,
		"count":   len(plugins),
	})
}

// SendSuccessResponse provides DRY pattern for success responses
// SOLID Extract Method: Eliminates response formatting duplication
func (cep *CommonEndpointProcessor) SendSuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// SendErrorResponse provides DRY pattern for error responses
// SOLID Extract Method: Eliminates error formatting duplication
func (cep *CommonEndpointProcessor) SendErrorResponse(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": err.Error(),
	})
}

// SendBadRequestResponse provides DRY pattern for bad request responses
// SOLID Extract Method: Eliminates validation error formatting duplication
func (cep *CommonEndpointProcessor) SendBadRequestResponse(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": message,
	})
}
