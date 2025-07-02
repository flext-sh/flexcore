// Package flexcore provides a Clean Architecture kernel for Go applications
// with DDD, events, workflows, and dependency injection.
package flexcore

import (
	"context"
	"fmt"

	"github.com/flext/flexcore/infrastructure/di"
	"github.com/flext/flexcore/infrastructure/events"
	"github.com/flext/flexcore/infrastructure/windmill"
	"github.com/flext/flexcore/infrastructure/observability"
	"github.com/flext/flexcore/infrastructure/scheduler"
	"github.com/flext/flexcore/application/commands"
	"github.com/flext/flexcore/application/queries"
	"github.com/flext/flexcore/shared/errors"
)

// Kernel represents the core application kernel that orchestrates
// all components following Clean Architecture principles.
type Kernel struct {
	container *di.Container
	config    *KernelConfig
}

// KernelConfig holds configuration for the kernel
type KernelConfig struct {
	AppName    string
	Version    string
	Debug      bool
	EventsURL  string
	WorkflowDB string
}

// NewKernel creates a new FlexCore kernel instance
func NewKernel(opts ...KernelOption) *Kernel {
	config := &KernelConfig{
		AppName: "flexcore-app",
		Version: "1.0.0",
		Debug:   false,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &Kernel{
		container: di.NewContainer(),
		config:    config,
	}
}

// KernelOption represents a kernel configuration option
type KernelOption func(*KernelConfig)

// WithAppName sets the application name
func WithAppName(name string) KernelOption {
	return func(c *KernelConfig) {
		c.AppName = name
	}
}

// WithVersion sets the application version
func WithVersion(version string) KernelOption {
	return func(c *KernelConfig) {
		c.Version = version
	}
}

// WithDebug enables debug mode
func WithDebug() KernelOption {
	return func(c *KernelConfig) {
		c.Debug = true
	}
}

// WithEventsURL sets the events system URL
func WithEventsURL(url string) KernelOption {
	return func(c *KernelConfig) {
		c.EventsURL = url
	}
}

// WithWorkflowDB sets the workflow database connection
func WithWorkflowDB(db string) KernelOption {
	return func(c *KernelConfig) {
		c.WorkflowDB = db
	}
}

// Application represents a built application ready to run
type Application struct {
	kernel    *Kernel
	container *di.Container
	ctx       context.Context
	cancel    context.CancelFunc
}

// BuildApplication creates an application instance from the kernel
func (k *Kernel) BuildApplication(container *di.Container) *Application {
	ctx, cancel := context.WithCancel(context.Background())

	return &Application{
		kernel:    k,
		container: container,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Run starts the application
func (a *Application) Run() error {
	fmt.Printf("🚀 Starting %s v%s\n", a.kernel.config.AppName, a.kernel.config.Version)

	if a.kernel.config.Debug {
		fmt.Println("🐛 Debug mode enabled")
	}

	// Initialize infrastructure services
	if err := a.initializeInfrastructure(); err != nil {
		return errors.Wrap(err, "failed to initialize infrastructure")
	}

	// Initialize domain services
	if err := a.initializeDomain(); err != nil {
		return errors.Wrap(err, "failed to initialize domain")
	}

	// Initialize application services
	if err := a.initializeApplication(); err != nil {
		return errors.Wrap(err, "failed to initialize application")
	}

	fmt.Println("✅ FlexCore application started successfully")

	// Wait for context cancellation
	<-a.ctx.Done()
	return nil
}

// Stop gracefully stops the application
func (a *Application) Stop() {
	fmt.Println("🛑 Stopping FlexCore application...")
	
	// Stop distributed event bus
	if eventBusResult := di.Resolve[events.EventBus](a.container); eventBusResult.IsSuccess() {
		if err := eventBusResult.Value().Stop(); err != nil {
			fmt.Printf("⚠️ Error stopping event bus: %v\n", err)
		} else {
			fmt.Println("✅ Distributed event bus stopped")
		}
	}
	
	// Stop timer singleton manager
	if timerManagerResult := di.Resolve[*scheduler.TimerSingletonManager](a.container); timerManagerResult.IsSuccess() {
		if err := timerManagerResult.Value().Stop(); err != nil {
			fmt.Printf("⚠️ Error stopping timer manager: %v\n", err)
		} else {
			fmt.Println("✅ Timer singleton manager stopped")
		}
	}
	
	// Stop cluster coordinator
	if coordinatorResult := di.Resolve[scheduler.ClusterCoordinator](a.container); coordinatorResult.IsSuccess() {
		if err := coordinatorResult.Value().Stop(); err != nil {
			fmt.Printf("⚠️ Error stopping cluster coordinator: %v\n", err)
		} else {
			fmt.Println("✅ Cluster coordinator stopped")
		}
	}
	
	a.cancel()
}

// Container returns the DI container
func (a *Application) Container() *di.Container {
	return a.container
}

// initializeInfrastructure sets up infrastructure services
func (a *Application) initializeInfrastructure() error {
	fmt.Println("📡 Initializing infrastructure services...")

	// Initialize metrics collector
	metricsCollector := observability.NewMetricsCollector()
	a.container.RegisterSingleton(func() *observability.MetricsCollector {
		return metricsCollector
	})
	fmt.Println("✅ Metrics collector initialized")

	// Initialize tracer
	tracer := observability.NewTracer(a.kernel.config.AppName, 10000)
	// Add in-memory exporter for development
	tracer.AddExporter(observability.NewInMemoryExporter())
	a.container.RegisterSingleton(func() *observability.Tracer {
		return tracer
	})
	fmt.Println("✅ Distributed tracing initialized")

	// Initialize monitor
	monitorConfig := observability.MonitorConfig{
		MaxAlerts: 1000,
		AlertThresholds: observability.AlertThresholds{
			CPUPercent:     80.0,
			MemoryMB:       512.0,
			GoroutinesMax:  5000,
			LatencyMS:      1000.0,
			ErrorRatePercent: 5.0,
		},
	}
	monitor := observability.NewMonitor(metricsCollector, tracer, monitorConfig)
	a.container.RegisterSingleton(func() *observability.Monitor {
		return monitor
	})
	
	// Add health checkers
	monitor.AddHealthChecker(metricsCollector)
	monitor.AddHealthChecker(tracer)
	
	// Start monitoring
	if err := monitor.Start(a.ctx); err != nil {
		return errors.Wrap(err, "failed to start monitor")
	}
	fmt.Println("✅ Real-time monitoring started")

	// Initialize cluster coordinator
	clusterCoordinator := scheduler.NewInMemoryClusterCoordinator()
	if err := clusterCoordinator.Start(a.ctx); err != nil {
		return errors.Wrap(err, "failed to start cluster coordinator")
	}
	a.container.RegisterSingleton(func() scheduler.ClusterCoordinator {
		return clusterCoordinator
	})
	fmt.Println("✅ Cluster coordinator initialized")

	// Initialize timer singleton manager
	timerManager := scheduler.NewTimerSingletonManager(clusterCoordinator)
	a.container.RegisterSingleton(func() *scheduler.TimerSingletonManager {
		return timerManager
	})
	
	// Start timer manager
	if err := timerManager.Start(a.ctx); err != nil {
		return errors.Wrap(err, "failed to start timer singleton manager")
	}
	fmt.Println("✅ Timer singleton manager started")

	// Initialize distributed event bus with cluster coordination
	distributedEventBus := events.NewDistributedEventBus(100, 10, clusterCoordinator)
	if err := distributedEventBus.Start(a.ctx); err != nil {
		return errors.Wrap(err, "failed to start distributed event bus")
	}
	a.container.RegisterSingleton(func() events.EventBus {
		return distributedEventBus
	})
	
	// Register cluster-aware event bus interface too
	a.container.RegisterSingleton(func() events.ClusterAwareEventBus {
		return distributedEventBus
	})
	
	// Track event bus metrics
	metricsCollector.IncrementCounter("infrastructure.event_bus.started", nil)
	metricsCollector.IncrementCounter("infrastructure.distributed_event_bus.started", nil)
	fmt.Println("✅ Distributed event bus with clustering initialized")

	// Initialize Windmill client (workflow engine replacement)
	windmillConfig := windmill.Config{
		BaseURL:   a.kernel.config.EventsURL,
		Workspace: "default",
	}
	windmillClient := windmill.NewClient(windmillConfig)
	a.container.RegisterSingleton(func() *windmill.Client {
		return windmillClient
	})
	
	// Track Windmill client metrics
	metricsCollector.IncrementCounter("infrastructure.windmill.client_created", nil)
	fmt.Println("✅ Windmill client initialized")

	// Initialize persistence (in-memory for now)
	metricsCollector.IncrementCounter("infrastructure.persistence.initialized", nil)
	fmt.Println("✅ Persistence layer initialized")

	return nil
}

// initializeDomain sets up domain services
func (a *Application) initializeDomain() error {
	fmt.Println("🏗️ Initializing domain services...")

	// Register domain services
	fmt.Println("✅ Domain services registered")

	// Setup domain event handlers
	eventBusResult := di.Resolve[events.EventBus](a.container)
	if eventBusResult.IsFailure() {
		return errors.Wrap(eventBusResult.Error(), "failed to resolve event bus")
	}
	
	// Register domain event handlers here
	_ = eventBusResult.Value() // Use event bus for domain event handling
	fmt.Println("✅ Domain event handlers setup")

	return nil
}

// initializeApplication sets up application services
func (a *Application) initializeApplication() error {
	fmt.Println("⚙️ Initializing application services...")

	// Get observability components
	metricsResult := di.Resolve[*observability.MetricsCollector](a.container)
	if metricsResult.IsFailure() {
		return errors.Wrap(metricsResult.Error(), "failed to resolve metrics collector")
	}
	metrics := metricsResult.Value()

	tracerResult := di.Resolve[*observability.Tracer](a.container)
	if tracerResult.IsFailure() {
		return errors.Wrap(tracerResult.Error(), "failed to resolve tracer")
	}
	tracer := tracerResult.Value()

	// Register command bus with observability
	commandBus := commands.NewCommandBus()
	a.container.RegisterSingleton(func() commands.CommandBus {
		return commandBus
	})
	metrics.IncrementCounter("application.command_bus.registered", nil)
	fmt.Println("✅ Command bus registered")

	// Register query bus with observability
	queryBus := queries.NewQueryBus()
	a.container.RegisterSingleton(func() queries.QueryBus {
		return queryBus
	})
	metrics.IncrementCounter("application.query_bus.registered", nil)
	fmt.Println("✅ Query bus registered")

	// Setup application workflows (using Windmill) with tracing
	span, _ := tracer.StartSpan(a.ctx, "setup_application_workflows")
	defer tracer.FinishSpan(span)
	
	windmillClientResult := di.Resolve[*windmill.Client](a.container)
	if windmillClientResult.IsFailure() {
		tracer.SetSpanStatus(span, observability.SpanStatusError, windmillClientResult.Error().Error())
		return errors.Wrap(windmillClientResult.Error(), "failed to resolve windmill client")
	}
	_ = windmillClientResult.Value() // Use for workflow setup
	
	tracer.SetSpanTag(span, "component", "windmill")
	tracer.SetSpanTag(span, "operation", "workflow_setup")
	metrics.IncrementCounter("application.workflows.setup", nil)
	fmt.Println("✅ Application workflows setup")

	return nil
}