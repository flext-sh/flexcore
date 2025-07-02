// FlexCore E2E Test Application
// Real application for testing FlexCore in containerized environment

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flext/flexcore/core"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("🚀 Starting FlexCore E2E Test Application")

	// Load configuration from environment
	config := loadConfigFromEnv()

	// Initialize FlexCore
	flexCoreResult := core.NewFlexCore(config)
	if flexCoreResult.IsFailure() {
		log.Fatalf("Failed to create FlexCore: %v", flexCoreResult.Error())
	}

	flexCore := flexCoreResult.Value()

	// Start FlexCore
	ctx := context.Background()
	startResult := flexCore.Start(ctx)
	if startResult.IsFailure() {
		log.Fatalf("Failed to start FlexCore: %v", startResult.Error())
	}

	log.Println("✅ FlexCore started successfully")

	// Register test adapters
	registerTestAdapters(ctx, flexCore)

	// Start HTTP server for testing
	router := setupHTTPServer(flexCore)
	server := &http.Server{
		Addr:    ":9000",
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Println("🌐 Starting HTTP server on :9000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down FlexCore...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	stopResult := flexCore.Stop(ctx)
	if stopResult.IsFailure() {
		log.Printf("Error stopping FlexCore: %v", stopResult.Error())
	}

	log.Println("✅ FlexCore shutdown completed")
}

func loadConfigFromEnv() *core.FlexCoreConfig {
	return &core.FlexCoreConfig{
		// Windmill configuration
		WindmillURL:       getEnv("WINDMILL_URL", "http://localhost:8000"),
		WindmillToken:     getEnv("WINDMILL_TOKEN", "test-token"),
		WindmillWorkspace: getEnv("WINDMILL_WORKSPACE", "demo"),

		// Cluster configuration
		ClusterName:  getEnv("FLEXCORE_CLUSTER_NAME", "test-cluster"),
		NodeID:       getEnv("FLEXCORE_NODE_ID", "test-node-1"),
		ClusterNodes: []string{"node-1", "node-2"},

		// Event routes
		EventRoutes: []core.EventRoute{
			{
				Name:           "test-data-pipeline",
				SourceAdapter:  "postgres-extractor",
				TargetAdapters: []string{"json-transformer", "api-loader"},
				EventFilter: core.EventFilter{
					EventTypes: []string{"data.extracted"},
				},
				Async:      true,
				MaxRetries: 3,
			},
		},

		// Message queues
		MessageQueues: []core.QueueConfig{
			{
				Name:    "test-queue",
				Type:    "fifo",
				MaxSize: 1000,
				TTL:     time.Hour,
			},
		},

		// Schedulers
		Schedulers: []core.SchedulerConfig{
			{
				Name:           "test-scheduler",
				CronExpression: "0 */1 * * * *", // Every minute
				WorkflowPath:   "test/scheduled_job",
				Singleton:      true,
				Input: map[string]interface{}{
					"test": true,
				},
			},
		},

		// Plugin configuration
		PluginDirectory: "/app/plugins",
		EnabledPlugins:  []string{"postgres-extractor", "json-transformer", "api-loader"},

		// Performance settings
		MaxConcurrentJobs: 5,
		EventBufferSize:   100,
		RetryPolicy: core.RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  time.Second,
			MaxDelay:      time.Minute,
			BackoffFactor: 2.0,
		},

		// Custom parameters
		CustomParams: map[string]interface{}{
			"test_mode":   true,
			"environment": "testing",
		},
	}
}

func registerTestAdapters(ctx context.Context, flexCore *core.FlexCore) {
	// PostgreSQL extractor adapter
	postgresAdapter := &core.AdapterConfig{
		Name:       "postgres-extractor",
		Type:       "extractor",
		PluginName: "postgres-extractor",
		Config: map[string]interface{}{
			"host":     getEnv("POSTGRES_HOST", "localhost"),
			"port":     getEnv("POSTGRES_PORT", "5432"),
			"database": getEnv("POSTGRES_DB", "flexcore_test"),
			"username": getEnv("POSTGRES_USER", "testuser"),
			"password": getEnv("POSTGRES_PASSWORD", "testpass"),
			"sslmode":  "disable",
		},
		EventTypes: []string{"data.extracted"},
	}

	// JSON transformer adapter
	transformerAdapter := &core.AdapterConfig{
		Name:       "json-transformer",
		Type:       "transformer",
		PluginName: "json-transformer",
		Config: map[string]interface{}{
			"transformations":  []string{"clean", "normalize", "validate"},
			"required_fields":  []string{"name", "email"},
			"field_mapping":    map[string]string{"email": "email_address"},
		},
		EventTypes: []string{"data.transformed"},
	}

	// API loader adapter
	apiAdapter := &core.AdapterConfig{
		Name:       "api-loader",
		Type:       "loader",
		PluginName: "api-loader",
		Config: map[string]interface{}{
			"endpoint":        getEnv("MOCK_API_URL", "http://localhost:8080") + "/api/data",
			"method":          "POST",
			"timeout_seconds": 30,
			"retries":         3,
			"batch_size":      50,
		},
		EventTypes: []string{"data.loaded"},
	}

	// Register all adapters
	adapters := []*core.AdapterConfig{postgresAdapter, transformerAdapter, apiAdapter}
	for _, adapter := range adapters {
		result := flexCore.RegisterAdapter(ctx, adapter)
		if result.IsFailure() {
			log.Printf("Failed to register adapter %s: %v", adapter.Name, result.Error())
		} else {
			log.Printf("✅ Registered adapter: %s", adapter.Name)
		}
	}
}

func setupHTTPServer(flexCore *core.FlexCore) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"node_id":   getEnv("FLEXCORE_NODE_ID", "unknown"),
		})
	})

	// Get cluster status
	router.GET("/cluster/status", func(c *gin.Context) {
		ctx := c.Request.Context()
		result := flexCore.GetClusterStatus(ctx)
		if result.IsFailure() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error().Error()})
			return
		}
		c.JSON(http.StatusOK, result.Value())
	})

	// Get metrics
	router.GET("/metrics", func(c *gin.Context) {
		ctx := c.Request.Context()
		result := flexCore.GetMetrics(ctx)
		if result.IsFailure() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error().Error()})
			return
		}
		c.JSON(http.StatusOK, result.Value())
	})

	// Send event
	router.POST("/events", func(c *gin.Context) {
		var event core.Event
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		result := flexCore.SendEvent(ctx, &event)
		if result.IsFailure() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error().Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Send message
	router.POST("/messages/:queue", func(c *gin.Context) {
		queueName := c.Param("queue")
		var message core.Message
		if err := c.ShouldBindJSON(&message); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		result := flexCore.SendMessage(ctx, queueName, &message)
		if result.IsFailure() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error().Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Receive messages
	router.GET("/messages/:queue", func(c *gin.Context) {
		queueName := c.Param("queue")
		maxMessages := 10

		ctx := c.Request.Context()
		result := flexCore.ReceiveMessages(ctx, queueName, maxMessages)
		if result.IsFailure() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error().Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"messages": result.Value(),
			"count":    len(result.Value()),
		})
	})

	// Schedule job
	router.POST("/schedule/:scheduler", func(c *gin.Context) {
		schedulerName := c.Param("scheduler")
		var input map[string]interface{}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		result := flexCore.ScheduleJob(ctx, schedulerName, input)
		if result.IsFailure() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error().Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"job_id": result.Value(),
		})
	})

	// Execute workflow
	router.POST("/workflows/:path", func(c *gin.Context) {
		workflowPath := c.Param("path")
		var input map[string]interface{}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		result := flexCore.ExecuteWorkflow(ctx, workflowPath, input)
		if result.IsFailure() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error().Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"job_id": result.Value(),
		})
	})

	// Test data extraction endpoint
	router.POST("/test/extract", func(c *gin.Context) {
		var request struct {
			Source string `json:"source"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create test event for data extraction
		event := &core.Event{
			ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
			Type:      "data.extract_request",
			Source:    "test-client",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"source": request.Source,
				"test":   true,
			},
		}

		ctx := c.Request.Context()
		result := flexCore.SendEvent(ctx, event)
		if result.IsFailure() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error().Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Extraction started",
			"event_id": event.ID,
		})
	})

	return router
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}