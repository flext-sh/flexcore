// Package http - FlexCore Orchestrator HTTP Controller
package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/flext-sh/flexcore/pkg/orchestrator"
)

// OrchestratorController handles HTTP requests for runtime orchestration
type OrchestratorController struct {
	orchestrator *orchestrator.RuntimeOrchestrator
	logger       *zap.Logger
}

// NewOrchestratorController creates a new orchestrator controller
func NewOrchestratorController(orch *orchestrator.RuntimeOrchestrator) *OrchestratorController {
	logger, err := zap.NewProduction()
	if err != nil {
		// Fallback to no-op logger if production logger fails
		logger = zap.NewNop()
	}
	return &OrchestratorController{
		orchestrator: orch,
		logger:       logger,
	}
}

// RegisterRoutes registers orchestrator routes
func (oc *OrchestratorController) RegisterRoutes(router *gin.RouterGroup) {
	orchestrationRoutes := router.Group("/orchestration")
	{
		orchestrationRoutes.POST("/execute", oc.executeOrchestration)
		orchestrationRoutes.GET("/list", oc.listOrchestrations)
		orchestrationRoutes.GET("/:id", oc.getOrchestration)
		orchestrationRoutes.POST("/:id/cancel", oc.cancelOrchestration)
		orchestrationRoutes.GET("/info", oc.getOrchestratorInfo)
		orchestrationRoutes.GET("/metrics", oc.getOrchestratorMetrics)
	}

	// Workflow management routes
	workflowRoutes := router.Group("/workflows")
	{
		workflowRoutes.GET("/definitions", oc.getWorkflowDefinitions)
		workflowRoutes.GET("/active", oc.getActiveWorkflows)
	}

	// Runtime management routes
	runtimeRoutes := router.Group("/runtimes")
	{
		runtimeRoutes.GET("/list", oc.listRuntimes)
		runtimeRoutes.GET("/:type/status", oc.getRuntimeStatus)
		runtimeRoutes.POST("/:type/health", oc.checkRuntimeHealth)
	}
}

// executeOrchestration handles POST /api/v1/orchestration/execute
func (oc *OrchestratorController) executeOrchestration(c *gin.Context) {
	var request orchestrator.OrchestrationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		oc.logger.Warn("Invalid orchestration request",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate request
	if request.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Orchestration name is required",
		})
		return
	}

	if request.RuntimeType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Runtime type is required",
		})
		return
	}

	if request.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Command is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute orchestration
	result, err := oc.orchestrator.ExecuteOrchestration(ctx, &request)
	if err != nil {
		oc.logger.Error("Failed to execute orchestration",
			zap.Error(err),
			zap.Any("request", request))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to execute orchestration",
			"details": err.Error(),
		})
		return
	}

	oc.logger.Info("Orchestration executed successfully",
		zap.String("orchestration_id", result.OrchestrationID),
		zap.String("job_id", result.JobID),
		zap.String("runtime_type", request.RuntimeType))

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"orchestration_id": result.OrchestrationID,
		"job_id":           result.JobID,
		"status":           result.Status,
		"start_time":       result.StartTime,
		"duration":         result.Duration.String(),
		"message":          "Orchestration executed successfully",
	})
}

// listOrchestrations handles GET /api/v1/orchestration/list
func (oc *OrchestratorController) listOrchestrations(c *gin.Context) {
	// Parse query parameters
	status := c.Query("status")
	runtimeType := c.Query("runtime_type")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid limit parameter",
		})
		return
	}

	// Get orchestrations
	orchestrations, err := oc.orchestrator.ListOrchestrations(status, runtimeType, limit)
	if err != nil {
		oc.logger.Error("Failed to list orchestrations",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list orchestrations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"orchestrations": orchestrations,
		"count":          len(orchestrations),
		"filters": gin.H{
			"status":       status,
			"runtime_type": runtimeType,
			"limit":        limit,
		},
	})
}

// getOrchestration handles GET /api/v1/orchestration/:id
func (oc *OrchestratorController) getOrchestration(c *gin.Context) {
	orchestrationID := c.Param("id")
	if orchestrationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Orchestration ID is required",
		})
		return
	}

	// Get orchestration
	orchestration, err := oc.orchestrator.GetOrchestration(orchestrationID)
	if err != nil {
		oc.logger.Warn("Orchestration not found",
			zap.String("orchestration_id", orchestrationID))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Orchestration not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"orchestration": orchestration,
	})
}

// cancelOrchestration handles POST /api/v1/orchestration/:id/cancel
func (oc *OrchestratorController) cancelOrchestration(c *gin.Context) {
	orchestrationID := c.Param("id")
	if orchestrationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Orchestration ID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Cancel orchestration
	if err := oc.orchestrator.CancelOrchestration(ctx, orchestrationID); err != nil {
		oc.logger.Warn("Failed to cancel orchestration",
			zap.String("orchestration_id", orchestrationID),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to cancel orchestration",
			"details": err.Error(),
		})
		return
	}

	oc.logger.Info("Orchestration cancelled successfully",
		zap.String("orchestration_id", orchestrationID))

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"orchestration_id": orchestrationID,
		"status":           "cancelled",
		"message":          "Orchestration cancelled successfully",
	})
}

// getOrchestratorInfo handles GET /api/v1/orchestration/info
func (oc *OrchestratorController) getOrchestratorInfo(c *gin.Context) {
	info := oc.orchestrator.GetOrchestratorInfo()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"info":    info,
	})
}

// getOrchestratorMetrics handles GET /api/v1/orchestration/metrics
func (oc *OrchestratorController) getOrchestratorMetrics(c *gin.Context) {
	info := oc.orchestrator.GetOrchestratorInfo()
	
	// Extract metrics from info
	metrics, exists := info["metrics"]
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Metrics not available",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"metrics": metrics,
		"windmill_metrics": info["windmill"].(map[string]interface{})["metrics"],
	})
}

// getWorkflowDefinitions handles GET /api/v1/workflows/definitions
func (oc *OrchestratorController) getWorkflowDefinitions(c *gin.Context) {
	// This would get workflow definitions from the Windmill engine
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"workflows": []gin.H{
			{
				"id":           "meltano_execution",
				"name":         "Meltano Data Pipeline Execution",
				"runtime_type": "meltano",
				"description":  "Execute Meltano data pipelines with Singer taps and DBT models",
			},
			{
				"id":           "ray_execution",
				"name":         "Ray Distributed Computing Execution", 
				"runtime_type": "ray",
				"description":  "Execute distributed computing jobs on Ray cluster (future)",
			},
			{
				"id":           "k8s_execution",
				"name":         "Kubernetes Job Execution",
				"runtime_type": "kubernetes",
				"description":  "Execute containerized jobs on Kubernetes cluster (future)",
			},
		},
	})
}

// getActiveWorkflows handles GET /api/v1/workflows/active
func (oc *OrchestratorController) getActiveWorkflows(c *gin.Context) {
	// Get active orchestrations
	orchestrations, err := oc.orchestrator.ListOrchestrations("running", "", 100)
	if err != nil {
		oc.logger.Error("Failed to get active workflows",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get active workflows",
			"details": err.Error(),
		})
		return
	}

	// Transform to workflow format
	workflows := make([]gin.H, 0, len(orchestrations))
	for _, orch := range orchestrations {
		workflows = append(workflows, gin.H{
			"orchestration_id": orch.ID,
			"name":             orch.Name,
			"runtime_type":     orch.RuntimeType,
			"job_id":           orch.JobID,
			"status":           orch.Status,
			"progress":         orch.Progress,
			"start_time":       orch.StartTime,
			"duration":         orch.Duration.String(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"workflows": workflows,
		"count":     len(workflows),
	})
}

// listRuntimes handles GET /api/v1/runtimes/list
func (oc *OrchestratorController) listRuntimes(c *gin.Context) {
	includeInactiveStr := c.DefaultQuery("include_inactive", "false")
	includeInactive, _ := strconv.ParseBool(includeInactiveStr)

	// This would get runtimes from the runtime manager
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"runtimes": []gin.H{
			{
				"type":         "meltano",
				"status":       "running",
				"capabilities": []string{"singer_taps", "singer_targets", "dbt_models"},
				"last_seen":    time.Now(),
			},
			{
				"type":         "ray",
				"status":       "not_implemented",
				"capabilities": []string{"distributed_computing", "ml_training"},
				"last_seen":    nil,
			},
			{
				"type":         "kubernetes",
				"status":       "not_implemented", 
				"capabilities": []string{"container_orchestration", "auto_scaling"},
				"last_seen":    nil,
			},
		},
		"include_inactive": includeInactive,
	})
}

// getRuntimeStatus handles GET /api/v1/runtimes/:type/status
func (oc *OrchestratorController) getRuntimeStatus(c *gin.Context) {
	runtimeType := c.Param("type")
	if runtimeType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Runtime type is required",
		})
		return
	}

	// This would get status from the runtime manager
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"runtime": gin.H{
			"type":         runtimeType,
			"status":       "running",
			"last_seen":    time.Now(),
			"capabilities": []string{"data_processing"},
			"config": gin.H{
				"max_concurrency": 4,
				"timeout":         "300s",
			},
		},
	})
}

// checkRuntimeHealth handles POST /api/v1/runtimes/:type/health
func (oc *OrchestratorController) checkRuntimeHealth(c *gin.Context) {
	runtimeType := c.Param("type")
	if runtimeType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Runtime type is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This would perform health check via runtime manager
	// For now, return a placeholder response
	_ = ctx

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"runtime": runtimeType,
		"status":  "healthy",
		"checked_at": time.Now(),
		"details": gin.H{
			"response_time": "150ms",
			"availability":  "100%",
		},
	})
}