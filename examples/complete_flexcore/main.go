// Complete FlexCore distributed event-driven system example
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/flext/flexcore/core"
	"github.com/flext/flexcore/infrastructure/windmill"
)

func main() {
	fmt.Println("🚀 Starting FlexCore Distributed Event-Driven System Example")

	// Create FlexCore configuration
	config := &core.FlexCoreConfig{
		// Windmill configuration
		WindmillURL:       "http://localhost:8000",
		WindmillToken:     "your-windmill-token",
		WindmillWorkspace: "demo",

		// Cluster configuration
		ClusterName:  "flexcore-demo",
		NodeID:       "node-1",
		ClusterNodes: []string{"node-1", "node-2"},

		// Event routes for distributed processing
		EventRoutes: []core.EventRoute{
			{
				Name:           "data-pipeline",
				SourceAdapter:  "extractor",
				TargetAdapters: []string{"transformer", "loader"},
				EventFilter: core.EventFilter{
					EventTypes: []string{"data.extracted"},
				},
				Async:      true,
				MaxRetries: 3,
			},
			{
				Name:           "notification-system",
				SourceAdapter:  "processor",
				TargetAdapters: []string{"notifier"},
				EventFilter: core.EventFilter{
					EventTypes: []string{"processing.completed"},
				},
				Transform: &core.TransformConfig{
					Type:     "script",
					Language: "python3",
					Script: `
def main(event):
    # Transform event for notification
    return {
        "id": event["id"],
        "type": "notification.ready",
        "data": {
            "message": f"Processing completed for {event['data']['source']}",
            "timestamp": event["timestamp"]
        }
    }
`,
				},
			},
		},

		// Message queues for distributed communication
		MessageQueues: []core.QueueConfig{
			{
				Name:    "high-priority",
				Type:    "priority",
				MaxSize: 1000,
				TTL:     time.Hour,
			},
			{
				Name:    "delayed-processing",
				Type:    "delayed",
				MaxSize: 500,
				TTL:     time.Hour * 24,
			},
		},

		// Distributed schedulers with singleton constraints
		Schedulers: []core.SchedulerConfig{
			{
				Name:           "data-sync",
				CronExpression: "0 */5 * * * *", // Every 5 minutes
				WorkflowPath:   "schedulers/data_sync",
				Singleton:      true, // Only one instance across cluster
				Input: map[string]interface{}{
					"source": "external-api",
					"target": "database",
				},
			},
			{
				Name:         "cleanup",
				WorkflowPath: "schedulers/cleanup",
				MaxInstances: 2, // Maximum 2 instances across cluster
				Input: map[string]interface{}{
					"older_than": "7d",
				},
			},
		},

		// Plugin configuration
		PluginDirectory: "./plugins",
		EnabledPlugins:  []string{"postgres-extractor", "json-transformer", "api-loader"},

		// Performance settings
		MaxConcurrentJobs: 10,
		EventBufferSize:   1000,
		RetryPolicy: core.RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  time.Second * 2,
			MaxDelay:      time.Minute * 5,
			BackoffFactor: 2.0,
		},

		// Custom parameters for library users
		CustomParams: map[string]interface{}{
			"company":     "Demo Corp",
			"environment": "development",
			"version":     "1.0.0",
		},
	}

	// Initialize FlexCore
	fmt.Println("📦 Initializing FlexCore...")
	flexCoreResult := core.NewFlexCore(config)
	if flexCoreResult.IsFailure() {
		log.Fatalf("Failed to create FlexCore: %v", flexCoreResult.Error())
	}

	flexCore := flexCoreResult.Value()

	// Start FlexCore distributed system
	fmt.Println("🔄 Starting FlexCore distributed components...")
	ctx := context.Background()
	startResult := flexCore.Start(ctx)
	if startResult.IsFailure() {
		log.Fatalf("Failed to start FlexCore: %v", startResult.Error())
	}

	fmt.Println("✅ FlexCore started successfully!")

	// Register sample adapters
	fmt.Println("🔌 Registering adapters...")
	
	extractorAdapter := &core.AdapterConfig{
		Name:       "extractor",
		Type:       "extractor",
		PluginName: "postgres-extractor",
		Config: map[string]interface{}{
			"host":     "localhost",
			"database": "demo_db",
			"table":    "source_data",
		},
		EventTypes: []string{"data.extracted"},
	}

	transformerAdapter := &core.AdapterConfig{
		Name:       "transformer",
		Type:       "transformer",
		PluginName: "json-transformer",
		Config: map[string]interface{}{
			"transformations": []string{"clean", "normalize", "validate"},
		},
		EventTypes: []string{"data.transformed"},
	}

	loaderAdapter := &core.AdapterConfig{
		Name:       "loader",
		Type:       "loader",
		PluginName: "api-loader",
		Config: map[string]interface{}{
			"endpoint": "https://api.example.com/data",
			"method":   "POST",
		},
		EventTypes: []string{"data.loaded"},
	}

	// Register all adapters
	for _, adapter := range []*core.AdapterConfig{extractorAdapter, transformerAdapter, loaderAdapter} {
		registerResult := flexCore.RegisterAdapter(ctx, adapter)
		if registerResult.IsFailure() {
			log.Printf("Failed to register adapter %s: %v", adapter.Name, registerResult.Error())
		} else {
			fmt.Printf("✅ Registered adapter: %s\n", adapter.Name)
		}
	}

	// Demonstrate distributed event processing
	fmt.Println("\n🌐 Demonstrating distributed event processing...")

	// Create and send events
	events := []*core.Event{
		{
			ID:        "event-1",
			Type:      "data.extracted",
			Source:    "extractor",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"source": "database",
				"records": 150,
				"size_mb": 25.5,
			},
			Headers: map[string]string{
				"priority": "high",
				"batch_id": "batch-001",
			},
		},
		{
			ID:        "event-2",
			Type:      "processing.completed",
			Source:    "processor",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"source":      "api-service",
				"duration_ms": 1250,
				"status":      "success",
			},
		},
	}

	for _, event := range events {
		sendResult := flexCore.SendEvent(ctx, event)
		if sendResult.IsFailure() {
			log.Printf("Failed to send event %s: %v", event.ID, sendResult.Error())
		} else {
			fmt.Printf("📤 Sent event: %s (type: %s)\n", event.ID, event.Type)
		}
	}

	// Demonstrate distributed message queuing
	fmt.Println("\n📬 Demonstrating distributed message queuing...")

	// Send high-priority message
	priorityMessage := &core.Message{
		ID:       "msg-priority-1",
		Queue:    "high-priority",
		Content:  map[string]interface{}{"task": "urgent-sync", "deadline": time.Now().Add(time.Hour)},
		Priority: 10,
		Headers:  map[string]string{"type": "urgent"},
	}

	sendMsgResult := flexCore.SendMessage(ctx, "high-priority", priorityMessage)
	if sendMsgResult.IsFailure() {
		log.Printf("Failed to send priority message: %v", sendMsgResult.Error())
	} else {
		fmt.Println("📤 Sent high-priority message")
	}

	// Send delayed message
	delayedMessage := &core.Message{
		ID:      "msg-delayed-1",
		Queue:   "delayed-processing",
		Content: map[string]interface{}{"task": "cleanup", "target": "old-logs"},
		ExpiresAt: func() *time.Time {
			t := time.Now().Add(time.Minute * 30)
			return &t
		}(),
		Headers: map[string]string{"type": "maintenance"},
	}

	sendDelayedResult := flexCore.SendMessage(ctx, "delayed-processing", delayedMessage)
	if sendDelayedResult.IsFailure() {
		log.Printf("Failed to send delayed message: %v", sendDelayedResult.Error())
	} else {
		fmt.Println("📤 Sent delayed message (30min delay)")
	}

	// Demonstrate distributed scheduling
	fmt.Println("\n⏰ Demonstrating distributed scheduling...")

	// Schedule singleton job
	singletonResult := flexCore.ScheduleJob(ctx, "data-sync", map[string]interface{}{
		"batch_size": 1000,
		"incremental": true,
	})
	if singletonResult.IsFailure() {
		log.Printf("Failed to schedule singleton job: %v", singletonResult.Error())
	} else {
		fmt.Printf("⏰ Scheduled singleton job: %s\n", singletonResult.Value())
	}

	// Schedule cleanup job
	cleanupResult := flexCore.ScheduleJob(ctx, "cleanup", map[string]interface{}{
		"target_dirs": []string{"/tmp", "/var/log"},
		"dry_run":     true,
	})
	if cleanupResult.IsFailure() {
		log.Printf("Failed to schedule cleanup job: %v", cleanupResult.Error())
	} else {
		fmt.Printf("⏰ Scheduled cleanup job: %s\n", cleanupResult.Value())
	}

	// Demonstrate custom workflow execution
	fmt.Println("\n🔧 Demonstrating custom workflow execution...")

	// Create custom workflow using the pipeline builder
	pipelineBuilder := windmill.NewPipelineWorkflowBuilder("data-processing", "Complete data processing pipeline")
	
	customWorkflow := pipelineBuilder.
		AddExtractorStep("source", "database", map[string]interface{}{
			"plugin_name": "postgres-extractor",
			"table":       "source_data",
		}).
		AddTransformerStep("clean", []string{"clean", "normalize"}, map[string]interface{}{
			"required_fields": []string{"id", "name", "timestamp"},
		}).
		AddLoaderStep("target", "api", map[string]interface{}{
			"plugin_name": "api-loader",
			"endpoint":    "https://api.example.com/processed",
		}).
		SetRetryPolicy(3, time.Second*5, 2.0).
		SetTimeout(time.Minute * 30).
		AddTag("type", "data-pipeline").
		Build()

	// Register and execute custom workflow
	registerWorkflowResult := flexCore.RegisterCustomWorkflow(ctx, customWorkflow)
	if registerWorkflowResult.IsFailure() {
		log.Printf("Failed to register custom workflow: %v", registerWorkflowResult.Error())
	} else {
		fmt.Println("✅ Registered custom data processing workflow")

		// Execute the workflow
		executeResult := flexCore.ExecuteWorkflow(ctx, customWorkflow.Path, map[string]interface{}{
			"batch_id":   "custom-001",
			"source_db":  "production",
			"target_api": "staging",
		})
		if executeResult.IsFailure() {
			log.Printf("Failed to execute workflow: %v", executeResult.Error())
		} else {
			fmt.Printf("🚀 Executing custom workflow: %s\n", executeResult.Value())
		}
	}

	// Demonstrate cluster management
	fmt.Println("\n🏢 Demonstrating cluster management...")

	clusterStatusResult := flexCore.GetClusterStatus(ctx)
	if clusterStatusResult.IsFailure() {
		log.Printf("Failed to get cluster status: %v", clusterStatusResult.Error())
	} else {
		status := clusterStatusResult.Value()
		fmt.Printf("🏢 Cluster: %s (Nodes: %d, Leader: %s)\n", 
			status.ClusterName, status.NodeCount, status.LeaderNode)
	}

	// Demonstrate metrics collection
	fmt.Println("\n📊 Demonstrating metrics collection...")

	metricsResult := flexCore.GetMetrics(ctx)
	if metricsResult.IsFailure() {
		log.Printf("Failed to get metrics: %v", metricsResult.Error())
	} else {
		metrics := metricsResult.Value()
		fmt.Printf("📊 Events: %d, Messages: %d, Jobs: %d, Plugins: %d\n",
			metrics.EventsProcessed, metrics.MessagesQueued, 
			metrics.ScheduledJobs, metrics.ActivePlugins)
	}

	// Demonstrate custom parameters
	fmt.Println("\n⚙️ Demonstrating custom parameters...")
	
	flexCore.SetCustomParameter("last_sync", time.Now())
	flexCore.SetCustomParameter("batch_count", 42)

	if company, exists := flexCore.GetCustomParameter("company"); exists {
		fmt.Printf("⚙️ Company: %s\n", company)
	}
	if lastSync, exists := flexCore.GetCustomParameter("last_sync"); exists {
		fmt.Printf("⚙️ Last sync: %s\n", lastSync)
	}

	// Demonstrate receiving messages
	fmt.Println("\n📥 Demonstrating message receiving...")

	receiveResult := flexCore.ReceiveMessages(ctx, "high-priority", 5)
	if receiveResult.IsFailure() {
		log.Printf("Failed to receive messages: %v", receiveResult.Error())
	} else {
		messages := receiveResult.Value()
		fmt.Printf("📥 Received %d messages from high-priority queue\n", len(messages))
	}

	// Let the system run for a while to show background processing
	fmt.Println("\n⏱️ Running for 30 seconds to demonstrate background processing...")
	time.Sleep(30 * time.Second)

	// Get final metrics
	finalMetricsResult := flexCore.GetMetrics(ctx)
	if finalMetricsResult.IsFailure() {
		log.Printf("Failed to get final metrics: %v", finalMetricsResult.Error())
	} else {
		metrics := finalMetricsResult.Value()
		fmt.Printf("📊 Final metrics - Events: %d, Messages: %d, Jobs: %d, Uptime: %.1fs\n",
			metrics.EventsProcessed, metrics.MessagesQueued, 
			metrics.ScheduledJobs, metrics.UptimeSeconds)
	}

	// Graceful shutdown
	fmt.Println("\n🛑 Shutting down FlexCore...")
	stopResult := flexCore.Stop(ctx)
	if stopResult.IsFailure() {
		log.Printf("Error during shutdown: %v", stopResult.Error())
	} else {
		fmt.Println("✅ FlexCore shutdown completed")
	}

	fmt.Println("\n🎉 FlexCore Distributed Event-Driven System Demo Completed!")
	fmt.Println("\n📋 Summary of FlexCore capabilities demonstrated:")
	fmt.Println("   ✅ Distributed event routing with Windmill workflows")
	fmt.Println("   ✅ Multi-type message queuing (FIFO, priority, delayed)")
	fmt.Println("   ✅ Distributed scheduling with singleton constraints")
	fmt.Println("   ✅ Plugin system with HashiCorp go-plugin")
	fmt.Println("   ✅ Cluster management and coordination")
	fmt.Println("   ✅ Custom workflow creation and execution")
	fmt.Println("   ✅ Event transformation and filtering")
	fmt.Println("   ✅ Parameterizable configuration as library")
	fmt.Println("   ✅ Comprehensive metrics and monitoring")
	fmt.Println("   ✅ Graceful lifecycle management")
}