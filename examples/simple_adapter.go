// Package examples demonstrates FlexCore usage
package examples

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/flext/flexcore"
	"github.com/flext/flexcore/adapters"
	"github.com/flext/flexcore/application/commands"
	"github.com/flext/flexcore/domain/entities"
	"github.com/flext/flexcore/infrastructure/config"
	"github.com/flext/flexcore/infrastructure/di"
	"github.com/flext/flexcore/infrastructure/events"
	"github.com/flext/flexcore/infrastructure/handlers"
	"github.com/flext/flexcore/shared/result"
)

// Example: Creating a simple PostgreSQL adapter using FlexCore patterns

// PostgreSQLAdapter demonstrates a simple database adapter
type PostgreSQLAdapter struct {
	*adapters.BaseAdapter
	connectionString string
	maxConnections   int
}

// NewPostgreSQLAdapter creates a new PostgreSQL adapter
func NewPostgreSQLAdapter() *PostgreSQLAdapter {
	adapter := &PostgreSQLAdapter{}
	
	// Use the adapter builder pattern
	base := adapters.NewAdapterBuilder("postgresql", "1.0.0", adapters.AdapterTypeConnector).
		OnInitialize(adapter.initialize).
		OnStart(adapter.start).
		OnStop(adapter.stop).
		OnHealthCheck(adapter.healthCheck).
		Build()
	
	adapter.BaseAdapter = base.(*adapters.BaseAdapter)
	return adapter
}

func (a *PostgreSQLAdapter) initialize(ctx context.Context) error {
	// Load configuration
	a.connectionString = a.Config().GetString("database.connection_string")
	a.maxConnections = a.Config().GetInt("database.max_connections")
	
	log.Printf("PostgreSQL adapter initialized with max connections: %d", a.maxConnections)
	return nil
}

func (a *PostgreSQLAdapter) start(ctx context.Context) error {
	log.Println("PostgreSQL adapter started")
	// Here you would establish database connection
	return nil
}

func (a *PostgreSQLAdapter) stop(ctx context.Context) error {
	log.Println("PostgreSQL adapter stopped")
	// Here you would close database connection
	return nil
}

func (a *PostgreSQLAdapter) healthCheck(ctx context.Context) adapters.HealthStatus {
	// Here you would ping the database
	return adapters.HealthStatus{
		Healthy:   true,
		Message:   "PostgreSQL connection is healthy",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"connections": 5,
			"latency":     "2ms",
		},
	}
}

// Example: Using advanced DI container

func setupDIContainer() *di.AdvancedContainer {
	container := di.NewContainerBuilder().
		WithSingleton("config", func(ctx context.Context) (interface{}, error) {
			// Create configuration manager
			cfg, err := config.NewManager(
				config.WithConfigFile("config.yaml"),
				config.WithEnvPrefix("FLEXCORE"),
			)
			if err != nil {
				return nil, err
			}
			return cfg.Value(), nil
		}).
		WithFactory("pipeline_service", func(ctx context.Context) (interface{}, error) {
			// Create pipeline service with dependencies
			return &PipelineService{}, nil
		}).
		WithValue("app_name", "FlexCore Example").
		Build()
		
	return container
}

// PipelineService demonstrates a service using FlexCore patterns
type PipelineService struct {
	commandBus commands.CommandBus
	eventBus   events.EventBus
}

func (s *PipelineService) CreatePipeline(ctx context.Context, name, description, owner string) result.Result[*entities.Pipeline] {
	cmd := commands.NewCreatePipelineCommand(name, description, owner, []string{"example"})
	
	result := s.commandBus.Execute(ctx, cmd)
	if result.IsFailure() {
		return result.Result[*entities.Pipeline]{}.Failure(result.Error())
	}
	
	pipeline := result.Value().(*entities.Pipeline)
	return result.Success(pipeline)
}

// Example: Using handler chain with middlewares

func setupHandlerChain() handlers.Handler {
	// Create a simple logger
	logger := &SimpleLogger{}
	
	// Build handler chain
	chain := handlers.NewChainBuilder().
		UseRecovery().
		UseLogging(logger).
		UseTimeout(30 * time.Second).
		UseRetry(3, 100*time.Millisecond).
		UseMetrics(&SimpleMetricsCollector{}).
		Build()
	
	// Final handler
	handler := handlers.HandlerFunc(func(ctx context.Context, req handlers.Request) result.Result[handlers.Response] {
		// Process request
		response := &handlers.BasicResponse{
			StatusCode: 200,
			Body: map[string]interface{}{
				"message": "Request processed successfully",
				"id":      req.ID(),
			},
		}
		return result.Success[handlers.Response](response)
	})
	
	return chain.Then(handler)
}

// SimpleLogger implements basic logging
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

// SimpleMetricsCollector implements basic metrics
type SimpleMetricsCollector struct{}

func (m *SimpleMetricsCollector) RecordRequest(method, path string, status int, duration time.Duration) {
	log.Printf("[METRICS] %s %s - Status: %d, Duration: %v", method, path, status, duration)
}

// Example: Complete application setup

func main() {
	// Create FlexCore kernel
	kernel := flexcore.NewKernel(
		flexcore.WithAppName("FlexCore Example"),
		flexcore.WithVersion("1.0.0"),
		flexcore.WithDebug(),
	)
	
	// Setup DI container
	container := setupDIContainer()
	
	// Setup adapter registry
	registry := adapters.NewAdapterRegistry()
	
	// Create and register PostgreSQL adapter
	pgAdapter := NewPostgreSQLAdapter()
	if err := registry.Register(pgAdapter); err != nil {
		log.Fatal(err)
	}
	
	// Create configuration
	configManager, _ := config.NewManager(
		config.WithConfigFile("config.yaml"),
		config.WithEnvPrefix("FLEXCORE"),
	)
	
	// Configure adapter
	pgAdapter.BaseAdapter = adapters.NewAdapterBuilder("postgresql", "1.0.0", adapters.AdapterTypeConnector).
		WithConfiguration(configManager.Value()).
		WithDependencyInjection(container).
		Build().(*adapters.BaseAdapter)
	
	// Initialize and start adapter
	ctx := context.Background()
	
	if err := pgAdapter.Initialize(ctx); err != nil {
		log.Fatal("Failed to initialize adapter:", err)
	}
	
	if err := pgAdapter.Start(ctx); err != nil {
		log.Fatal("Failed to start adapter:", err)
	}
	
	// Check adapter health
	health := pgAdapter.HealthCheck(ctx)
	fmt.Printf("Adapter health: %+v\n", health)
	
	// Setup handler chain
	handler := setupHandlerChain()
	
	// Process a request
	request := &handlers.BasicRequest{
		Context: ctx,
		ID:      "req-123",
		Method:  "POST",
		Path:    "/pipelines",
		Body: map[string]interface{}{
			"name":        "Example Pipeline",
			"description": "A demo pipeline",
		},
	}
	
	response := handler.Handle(ctx, request)
	if response.IsSuccess() {
		fmt.Printf("Response: %+v\n", response.Value())
	} else {
		fmt.Printf("Error: %v\n", response.Error())
	}
	
	// Demonstrate Railway pattern
	result := createPipelineWithValidation("Test Pipeline", "Demo", "user@example.com")
	result.
		Tap(func(p *entities.Pipeline) {
			fmt.Printf("Pipeline created: %s\n", p.Name)
		}).
		TapError(func(err error) {
			fmt.Printf("Pipeline creation failed: %v\n", err)
		})
	
	// Build and run application
	app := kernel.BuildApplication(container)
	
	// Graceful shutdown
	fmt.Println("\nApplication is running. Press Ctrl+C to exit.")
	select {}
}

// Example: Railway-oriented programming
func createPipelineWithValidation(name, description, owner string) result.Railway[*entities.Pipeline] {
	return validatePipelineName(name).
		Then(func(validName string) result.Railway[*entities.Pipeline] {
			return validateOwner(owner).
				ThenMap(func(validOwner string) *entities.Pipeline {
					pipeline, _ := entities.NewPipeline(validName, description, validOwner)
					return pipeline.Value()
				})
		})
}

func validatePipelineName(name string) result.Railway[string] {
	if len(name) < 3 {
		return result.Failure[string](fmt.Errorf("pipeline name must be at least 3 characters"))
	}
	return result.Success(name)
}

func validateOwner(owner string) result.Railway[string] {
	if owner == "" {
		return result.Failure[string](fmt.Errorf("owner cannot be empty"))
	}
	return result.Success(owner)
}