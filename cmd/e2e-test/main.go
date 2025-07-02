// Real E2E Test for FlexCore
// Tests actual components working together
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	flexcore "github.com/flext/flexcore"
	"github.com/flext/flexcore/infrastructure/di"
	"github.com/flext/flexcore/infrastructure/observability"
	"github.com/flext/flexcore/infrastructure/plugins"
	"github.com/flext/flexcore/infrastructure/events"
	"github.com/flext/flexcore/infrastructure/windmill"
	"github.com/flext/flexcore/infrastructure/scheduler"
	"github.com/flext/flexcore/application/commands"
	"github.com/flext/flexcore/application/queries"
	"github.com/flext/flexcore/domain"
	"github.com/flext/flexcore/shared/result"
	"github.com/flext/flexcore/shared/errors"
)

// E2ETestResult represents test results
type E2ETestResult struct {
	Name     string        `json:"name"`
	Passed   bool          `json:"passed"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
	Details  interface{}   `json:"details,omitempty"`
}

// E2ETestSuite manages E2E tests
type E2ETestSuite struct {
	app     *flexcore.Application
	results []E2ETestResult
}

func main() {
	fmt.Println("🧪 FlexCore Real E2E Test Suite")
	fmt.Println("==============================")

	suite := &E2ETestSuite{
		results: make([]E2ETestResult, 0),
	}

	// Initialize FlexCore application
	if err := suite.setupApplication(); err != nil {
		log.Fatalf("❌ Failed to setup application: %v", err)
	}

	// Run all E2E tests
	ctx := context.Background()
	suite.runAllTests(ctx)

	// Generate results
	suite.generateReport()

	// Check results
	passed, total := suite.getResults()
	fmt.Printf("\n📊 E2E Test Results: %d/%d passed\n", passed, total)

	if passed == total {
		fmt.Println("✅ ALL E2E TESTS PASSED!")
	} else {
		fmt.Printf("❌ %d tests failed\n", total-passed)
	}
}

func (suite *E2ETestSuite) setupApplication() error {
	fmt.Println("🚀 Setting up FlexCore application...")

	// Create kernel with configuration
	kernel := flexcore.NewKernel(
		flexcore.WithAppName("e2e-test-app"),
		flexcore.WithVersion("1.0.0"),
		flexcore.WithDebug(),
		flexcore.WithEventsURL("http://localhost:8000"),
	)

	container := di.NewContainer()
	suite.app = kernel.BuildApplication(container)

	// Start application in background
	go func() {
		if err := suite.app.Run(); err != nil {
			log.Printf("Application error: %v", err)
		}
	}()

	// Give application time to initialize
	time.Sleep(2 * time.Second)

	return nil
}

func (suite *E2ETestSuite) runAllTests(ctx context.Context) {
	fmt.Println("\n🔬 Running E2E Tests...")

	// Test categories
	tests := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Core Architecture Initialization", suite.testCoreArchitecture},
		{"Dependency Injection System", suite.testDependencyInjection},
		{"Command/Query Bus Integration", suite.testCQRSIntegration},
		{"Event System Integration", suite.testEventSystem},
		{"Observability Integration", suite.testObservability},
		{"Plugin System E2E", suite.testPluginSystemE2E},
		{"Windmill Client Integration", suite.testWindmillIntegration},
		{"Timer Singletons & Cluster Coordination", suite.testTimerSingletons},
		{"Distributed Event System & Clustering", suite.testDistributedEventSystem},
		{"Cross-Component Integration", suite.testCrossComponentIntegration},
		{"Error Handling & Recovery", suite.testErrorHandling},
		{"Performance & Memory", suite.testPerformance},
	}

	for i, test := range tests {
		fmt.Printf("\n%d. 🧪 %s\n", i+1, test.name)
		start := time.Now()
		
		err := test.fn(ctx)
		duration := time.Since(start)
		
		passed := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}

		suite.recordResult(test.name, passed, duration, errorMsg, nil)
		
		if passed {
			fmt.Printf("   ✅ PASSED (%.2fs)\n", duration.Seconds())
		} else {
			fmt.Printf("   ❌ FAILED (%.2fs): %s\n", duration.Seconds(), errorMsg)
		}
	}
}

func (suite *E2ETestSuite) testCoreArchitecture(ctx context.Context) error {
	// Test that all core components are properly initialized
	container := suite.app.Container()
	
	// Test event bus resolution using the correct type
	eventBusResult := di.Resolve[events.EventBus](container)
	if eventBusResult.IsFailure() {
		return fmt.Errorf("event bus not properly registered: %v", eventBusResult.Error())
	}

	// Test DI container functionality
	container.RegisterSingleton(func() string {
		return "e2e-test-service"
	})

	testResult := di.Resolve[string](container)
	if testResult.IsFailure() || testResult.Value() != "e2e-test-service" {
		return fmt.Errorf("DI container not working properly")
	}

	return nil
}

func (suite *E2ETestSuite) testDependencyInjection(ctx context.Context) error {
	container := suite.app.Container()

	// Test complex dependency resolution
	container.RegisterSingleton(func() *TestService {
		return &TestService{Name: "e2e-service", ID: 123}
	})

	container.Register(func(service *TestService) *TestClient {
		return &TestClient{Service: service, Connected: true}
	})

	// Resolve complex dependency
	clientResult := di.Resolve[*TestClient](container)
	if clientResult.IsFailure() {
		return fmt.Errorf("complex dependency resolution failed: %v", clientResult.Error())
	}

	client := clientResult.Value()
	if client.Service == nil || client.Service.Name != "e2e-service" || !client.Connected {
		return fmt.Errorf("dependency injection produced invalid object")
	}

	return nil
}

func (suite *E2ETestSuite) testCQRSIntegration(ctx context.Context) error {
	container := suite.app.Container()

	// Test command bus
	commandBusResult := di.Resolve[commands.CommandBus](container)
	if commandBusResult.IsFailure() {
		return fmt.Errorf("command bus not available: %v", commandBusResult.Error())
	}

	commandBus := commandBusResult.Value()

	// Create test command
	testCommand := commands.NewBaseCommand("test.command")
	
	// Create test handler
	testHandler := &TestCommandHandler{}
	
	// Register test command handler
	err := commandBus.RegisterHandler(testCommand, testHandler)
	if err != nil {
		return fmt.Errorf("failed to register command handler: %v", err)
	}

	// Execute command
	result := commandBus.Execute(ctx, testCommand)
	if result.IsFailure() {
		return fmt.Errorf("command execution failed: %v", result.Error())
	}

	// Test query bus
	queryBusResult := di.Resolve[queries.QueryBus](container)
	if queryBusResult.IsFailure() {
		return fmt.Errorf("query bus not available: %v", queryBusResult.Error())
	}

	return nil
}

func (suite *E2ETestSuite) testEventSystem(ctx context.Context) error {
	container := suite.app.Container()

	// Test event bus resolution
	eventBusResult := di.Resolve[events.EventBus](container)
	if eventBusResult.IsFailure() {
		return fmt.Errorf("event system not properly integrated: %v", eventBusResult.Error())
	}

	eventBus := eventBusResult.Value()
	
	// Test basic event bus functionality
	testEvent := domain.CreateSystemEvent("test.event", map[string]interface{}{
		"test_data": "e2e_test",
	})
	
	// Try to publish an event
	err := eventBus.Publish(ctx, testEvent)
	if err != nil {
		return fmt.Errorf("event publishing failed: %v", err)
	}

	return nil
}

func (suite *E2ETestSuite) testObservability(ctx context.Context) error {
	container := suite.app.Container()

	// Test metrics collector
	metricsResult := di.Resolve[*observability.MetricsCollector](container)
	if metricsResult.IsFailure() {
		return fmt.Errorf("metrics collector not available: %v", metricsResult.Error())
	}

	metrics := metricsResult.Value()
	if !metrics.IsEnabled() {
		return fmt.Errorf("metrics collector not enabled")
	}

	// Test metrics functionality
	metrics.IncrementCounter("e2e.test.counter", map[string]string{"test": "true"})
	metrics.SetGauge("e2e.test.gauge", 42.5, nil)

	count := metrics.GetCounter("e2e.test.counter", map[string]string{"test": "true"})
	if count != 1 {
		return fmt.Errorf("metrics counter not working: expected 1, got %d", count)
	}

	gauge := metrics.GetGauge("e2e.test.gauge", nil)
	if gauge != 42.5 {
		return fmt.Errorf("metrics gauge not working: expected 42.5, got %f", gauge)
	}

	// Test tracer
	tracerResult := di.Resolve[*observability.Tracer](container)
	if tracerResult.IsFailure() {
		return fmt.Errorf("tracer not available: %v", tracerResult.Error())
	}

	tracer := tracerResult.Value()
	if !tracer.IsEnabled() {
		return fmt.Errorf("tracer not enabled")
	}

	// Test tracing functionality
	span, _ := tracer.StartSpan(ctx, "e2e.test.operation")
	if span == nil {
		return fmt.Errorf("span creation failed")
	}

	tracer.SetSpanTag(span, "test", "e2e")
	tracer.AddSpanEvent(span, "test_event", map[string]interface{}{"data": "test"})
	tracer.FinishSpan(span)

	// Verify span was recorded
	allSpans := tracer.GetAllSpans()
	if len(allSpans) == 0 {
		return fmt.Errorf("no spans recorded")
	}

	// Test monitor
	monitorResult := di.Resolve[*observability.Monitor](container)
	if monitorResult.IsFailure() {
		return fmt.Errorf("monitor not available: %v", monitorResult.Error())
	}

	monitor := monitorResult.Value()
	if !monitor.IsRunning() {
		return fmt.Errorf("monitor not running")
	}

	// Test health check
	healthStatus := monitor.GetHealthStatus(ctx)
	if healthStatus.Overall != observability.HealthStateHealthy {
		return fmt.Errorf("system not healthy: %s", healthStatus.Overall)
	}

	return nil
}

func (suite *E2ETestSuite) testPluginSystemE2E(ctx context.Context) error {
	// Test plugin manager creation and basic functionality
	pluginManager := plugins.NewPluginManager("./plugins")
	
	// Register plugin types
	pluginManager.RegisterPluginType("plugin", &plugins.ExtractorGRPCPlugin{})

	// Scan for plugins
	scanResult := pluginManager.ScanPluginDirectory(ctx)
	if scanResult.IsFailure() {
		return fmt.Errorf("plugin scan failed: %v", scanResult.Error())
	}

	pluginPaths := scanResult.Value()
	if len(pluginPaths) == 0 {
		// Plugin system is working, just no plugins built
		return nil
	}

	// Try to load plugins
	loadResult := pluginManager.LoadAllPlugins(ctx)
	if loadResult.IsFailure() {
		return fmt.Errorf("plugin loading failed: %v", loadResult.Error())
	}

	loadedCount := loadResult.Value()
	if loadedCount > 0 {
		// Test plugin listing
		loadedPlugins := pluginManager.ListPlugins()
		if len(loadedPlugins) != loadedCount {
			return fmt.Errorf("plugin count mismatch: loaded %d, listed %d", loadedCount, len(loadedPlugins))
		}

		// Test at least one plugin is active
		activeFound := false
		for _, plugin := range loadedPlugins {
			if plugin.IsActive {
				activeFound = true
				break
			}
		}

		if !activeFound {
			return fmt.Errorf("no active plugins found")
		}
	}

	// Cleanup
	pluginManager.Shutdown(ctx)
	return nil
}

func (suite *E2ETestSuite) testWindmillIntegration(ctx context.Context) error {
	container := suite.app.Container()

	// Test Windmill client availability
	windmillResult := di.Resolve[*windmill.Client](container)
	if windmillResult.IsFailure() {
		return fmt.Errorf("windmill client resolution failed: %v", windmillResult.Error())
	}

	windmillClient := windmillResult.Value()
	if windmillClient == nil {
		return fmt.Errorf("windmill client is nil")
	}

	// Test basic client functionality - this doesn't require real Windmill server
	// Just verifies the client is properly instantiated
	return nil
}

func (suite *E2ETestSuite) testTimerSingletons(ctx context.Context) error {
	container := suite.app.Container()

	// Test cluster coordinator availability
	coordinatorResult := di.Resolve[scheduler.ClusterCoordinator](container)
	if coordinatorResult.IsFailure() {
		return fmt.Errorf("cluster coordinator not available: %v", coordinatorResult.Error())
	}

	coordinator := coordinatorResult.Value()
	
	// Test basic coordinator functionality
	nodeID := coordinator.GetNodeID()
	if nodeID == "" {
		return fmt.Errorf("node ID should not be empty")
	}

	// Test timer singleton manager availability
	managerResult := di.Resolve[*scheduler.TimerSingletonManager](container)
	if managerResult.IsFailure() {
		return fmt.Errorf("timer singleton manager not available: %v", managerResult.Error())
	}

	manager := managerResult.Value()
	
	// Create a test timer singleton
	var executionCount int64
	config := scheduler.TimerSingletonConfig{
		Name:     "e2e-test-singleton",
		Interval: 50 * time.Millisecond,
	}
	
	executor := func(ctx context.Context) error {
		executionCount++
		return nil
	}
	
	singleton := scheduler.NewTimerSingleton(config, coordinator, executor)
	
	// Register and start the singleton
	err := manager.Register("e2e-test-singleton", singleton)
	if err != nil {
		return fmt.Errorf("failed to register singleton: %v", err)
	}
	
	err = singleton.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start singleton: %v", err)
	}
	defer singleton.Stop()
	
	// Let it run for a short time
	time.Sleep(150 * time.Millisecond)
	
	// Verify it executed
	if executionCount < 1 {
		return fmt.Errorf("singleton should have executed at least once, got %d", executionCount)
	}
	
	// Test singleton retrieval
	retrieved, exists := manager.GetSingleton("e2e-test-singleton")
	if !exists {
		return fmt.Errorf("singleton should be retrievable")
	}
	
	if retrieved != singleton {
		return fmt.Errorf("retrieved singleton should match original")
	}
	
	// Test stats
	stats := singleton.GetStats()
	if stats.Name != "e2e-test-singleton" {
		return fmt.Errorf("stats name mismatch")
	}
	
	if !stats.Running {
		return fmt.Errorf("stats should show running")
	}
	
	if stats.ExecutionCount != executionCount {
		return fmt.Errorf("stats execution count mismatch: expected %d, got %d", executionCount, stats.ExecutionCount)
	}

	return nil
}

func (suite *E2ETestSuite) testDistributedEventSystem(ctx context.Context) error {
	container := suite.app.Container()

	// Test cluster-aware event bus availability
	clusterEventBusResult := di.Resolve[events.ClusterAwareEventBus](container)
	if clusterEventBusResult.IsFailure() {
		return fmt.Errorf("cluster-aware event bus not available: %v", clusterEventBusResult.Error())
	}

	clusterEventBus := clusterEventBusResult.Value()

	// Test basic event bus functionality (inherited from base)
	eventBusResult := di.Resolve[events.EventBus](container)
	if eventBusResult.IsFailure() {
		return fmt.Errorf("event bus interface not available: %v", eventBusResult.Error())
	}

	eventBus := eventBusResult.Value()

	// Create test event
	testEvent := domain.CreateSystemEvent("distributed.test.event", map[string]interface{}{
		"test_data": "cluster_test",
		"node_id":   "test_node",
	})

	// Test local event publishing
	if err := eventBus.Publish(ctx, testEvent); err != nil {
		return fmt.Errorf("failed to publish test event: %v", err)
	}

	// Test cluster event broadcasting
	if err := clusterEventBus.BroadcastToCluster(ctx, testEvent); err != nil {
		return fmt.Errorf("failed to broadcast event to cluster: %v", err)
	}

	// Get cluster statistics
	stats := clusterEventBus.GetClusterStats()
	if stats.LocalEvents < 1 {
		return fmt.Errorf("expected at least 1 local event, got %d", stats.LocalEvents)
	}

	// Should have at least attempted cluster broadcast
	if stats.ClusterEvents < 1 && stats.BroadcastErrors < 1 {
		return fmt.Errorf("expected cluster broadcast attempt, got cluster_events=%d, broadcast_errors=%d", 
			stats.ClusterEvents, stats.BroadcastErrors)
	}

	// Test cluster coordinator integration
	coordinatorResult := di.Resolve[scheduler.ClusterCoordinator](container)
	if coordinatorResult.IsFailure() {
		return fmt.Errorf("cluster coordinator not available for distributed events: %v", coordinatorResult.Error())
	}

	coordinator := coordinatorResult.Value()
	
	// Verify cluster coordinator has active nodes
	activeNodes := coordinator.GetActiveNodes(ctx)
	if len(activeNodes) < 1 {
		return fmt.Errorf("cluster coordinator should have at least 1 active node, got %d", len(activeNodes))
	}

	// Test that cluster stats reflect node count
	if stats.NodesReached != len(activeNodes) {
		return fmt.Errorf("cluster event stats should reflect active nodes: expected %d, got %d", 
			len(activeNodes), stats.NodesReached)
	}

	// Test cluster event subscription
	var receivedClusterEvent bool
	testHandler := func(ctx context.Context, event domain.DomainEvent) error {
		if event.EventType() == "distributed.test.event" {
			receivedClusterEvent = true
		}
		return nil
	}

	// Subscribe to the test event type
	if err := eventBus.Subscribe("distributed.test.event", testHandler); err != nil {
		return fmt.Errorf("failed to subscribe to test event: %v", err)
	}

	// Publish another test event to trigger handler
	testEvent2 := domain.CreateSystemEvent("distributed.test.event", map[string]interface{}{
		"test_data": "handler_test",
	})

	if err := eventBus.Publish(ctx, testEvent2); err != nil {
		return fmt.Errorf("failed to publish second test event: %v", err)
	}

	// Give event processing time
	time.Sleep(100 * time.Millisecond)

	// Verify handler was called
	if !receivedClusterEvent {
		return fmt.Errorf("event handler was not called for distributed test event")
	}

	return nil
}

func (suite *E2ETestSuite) testCrossComponentIntegration(ctx context.Context) error {
	container := suite.app.Container()

	// Test that observability is integrated with other components
	metricsResult := di.Resolve[*observability.MetricsCollector](container)
	if metricsResult.IsFailure() {
		return fmt.Errorf("metrics not available for integration test")
	}

	metrics := metricsResult.Value()

	// Check that infrastructure metrics were recorded during startup
	allMetrics := metrics.GetAllMetrics()
	infrastructureMetricsFound := false
	
	for _, metric := range allMetrics {
		if metric.Name == "infrastructure.event_bus.started" ||
		   metric.Name == "infrastructure.windmill.client_created" ||
		   metric.Name == "infrastructure.persistence.initialized" {
			infrastructureMetricsFound = true
			break
		}
	}

	if !infrastructureMetricsFound {
		return fmt.Errorf("infrastructure metrics not recorded - integration incomplete")
	}

	// Test that tracing is working across components
	tracerResult := di.Resolve[*observability.Tracer](container)
	if tracerResult.IsFailure() {
		return fmt.Errorf("tracer not available for integration test")
	}

	tracer := tracerResult.Value()
	allSpans := tracer.GetAllSpans()
	
	// Should have spans from application startup
	startupSpansFound := false
	for _, span := range allSpans {
		if span.OperationName == "setup_application_workflows" {
			startupSpansFound = true
			break
		}
	}

	if !startupSpansFound {
		return fmt.Errorf("startup tracing not working - integration incomplete")
	}

	return nil
}

func (suite *E2ETestSuite) testErrorHandling(ctx context.Context) error {
	container := suite.app.Container()

	// Test error handling in DI container
	errorResult := di.Resolve[*NonExistentService](container)
	if errorResult.IsSuccess() {
		return fmt.Errorf("DI container should fail for non-existent service")
	}

	// Test error handling in command bus
	commandBusResult := di.Resolve[commands.CommandBus](container)
	if commandBusResult.IsFailure() {
		return fmt.Errorf("command bus not available for error test")
	}

	commandBus := commandBusResult.Value()
	
	// Try to execute non-existent command
	nonExistentCommand := commands.NewBaseCommand("non.existent.command")
	result := commandBus.Execute(ctx, nonExistentCommand)
	
	if result.IsSuccess() {
		return fmt.Errorf("command bus should fail for non-existent command")
	}

	return nil
}

func (suite *E2ETestSuite) testPerformance(ctx context.Context) error {
	container := suite.app.Container()

	// Register a test service for performance testing
	container.RegisterSingleton(func() string {
		return "performance-test-service"
	})

	// Test DI container performance
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		result := di.Resolve[string](container)
		if result.IsFailure() {
			return fmt.Errorf("DI resolution failed during performance test: %v", result.Error())
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	// Should be fast (less than 1ms per resolution on average)
	if avgDuration > time.Millisecond {
		return fmt.Errorf("DI container too slow: %v per resolution", avgDuration)
	}

	// Test metrics performance
	metricsResult := di.Resolve[*observability.MetricsCollector](container)
	if metricsResult.IsFailure() {
		return fmt.Errorf("metrics not available for performance test")
	}

	metrics := metricsResult.Value()
	
	start = time.Now()
	for i := 0; i < iterations; i++ {
		metrics.IncrementCounter("performance.test", nil)
	}
	duration = time.Since(start)
	avgDuration = duration / time.Duration(iterations)

	// Should be very fast (less than 100μs per increment)
	if avgDuration > 100*time.Microsecond {
		return fmt.Errorf("metrics too slow: %v per increment", avgDuration)
	}

	return nil
}

// Helper types and methods

type TestService struct {
	Name string
	ID   int
}

type TestClient struct {
	Service   *TestService
	Connected bool
}

type NonExistentService struct{}

// TestCommandHandler handles test commands for E2E testing
type TestCommandHandler struct{}

// Handle processes the test command
func (h *TestCommandHandler) Handle(ctx context.Context, cmd commands.Command) result.Result[interface{}] {
	if cmd.CommandType() == "test.command" {
		return result.Success[interface{}]("command processed successfully")
	}
	return result.Failure[interface{}](errors.ValidationError("unknown command type"))
}

func (suite *E2ETestSuite) recordResult(name string, passed bool, duration time.Duration, error string, details interface{}) {
	result := E2ETestResult{
		Name:     name,
		Passed:   passed,
		Duration: duration,
		Error:    error,
		Details:  details,
	}
	suite.results = append(suite.results, result)
}

func (suite *E2ETestSuite) getResults() (passed, total int) {
	total = len(suite.results)
	for _, result := range suite.results {
		if result.Passed {
			passed++
		}
	}
	return
}

func (suite *E2ETestSuite) generateReport() {
	fmt.Println("\n📋 Generating E2E Test Report...")

	// Generate JSON report
	jsonData, err := json.MarshalIndent(suite.results, "", "  ")
	if err != nil {
		log.Printf("Failed to generate JSON report: %v", err)
		return
	}

	if err := os.WriteFile("e2e-test-results.json", jsonData, 0644); err != nil {
		log.Printf("Failed to write JSON report: %v", err)
	} else {
		fmt.Println("   ✅ JSON report: e2e-test-results.json")
	}

	// Generate summary
	passed, total := suite.getResults()
	summary := fmt.Sprintf(`FlexCore E2E Test Suite - Real Implementation
============================================

Total Tests: %d
Passed: %d
Failed: %d
Success Rate: %.1f%%

Test Details:
`, total, passed, total-passed, float64(passed)/float64(total)*100)

	for _, result := range suite.results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		summary += fmt.Sprintf("- %s: %s (%.3fs)\n", result.Name, status, result.Duration.Seconds())
		if result.Error != "" {
			summary += fmt.Sprintf("  Error: %s\n", result.Error)
		}
	}

	if err := os.WriteFile("e2e-test-summary.txt", []byte(summary), 0644); err != nil {
		log.Printf("Failed to write summary report: %v", err)
	} else {
		fmt.Println("   ✅ Summary report: e2e-test-summary.txt")
	}

	// Cleanup
	suite.app.Stop()
}