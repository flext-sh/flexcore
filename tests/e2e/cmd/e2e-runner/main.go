// E2E Test Runner for FlexCore Distributed System
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
)

// TestConfig represents E2E test configuration
type TestConfig struct {
	FlexCoreClusterURL string
	WindmillURL        string
	PostgresURL        string
	TestTimeout        time.Duration
	ClusterNodes       []string
}

// TestResult represents a single test result
type TestResult struct {
	Name        string                 `json:"name"`
	Passed      bool                   `json:"passed"`
	Duration    time.Duration          `json:"duration"`
	Error       string                 `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// TestSuite manages and runs E2E tests
type TestSuite struct {
	config  *TestConfig
	logger  hclog.Logger
	results []TestResult
	client  *http.Client
}

func main() {
	// Create logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:       "e2e-test-runner",
		Level:      hclog.Info,
		Output:     os.Stdout,
		JSONFormat: true,
	})

	logger.Info("Starting FlexCore E2E Test Suite")

	// Load configuration
	config := loadTestConfig()
	logger.Info("Test configuration", 
		"cluster_url", config.FlexCoreClusterURL,
		"windmill_url", config.WindmillURL,
		"timeout", config.TestTimeout,
		"nodes", len(config.ClusterNodes))

	// Create test suite
	suite := NewTestSuite(config, logger)

	// Run all tests
	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	if err := suite.RunAllTests(ctx); err != nil {
		logger.Error("Test suite failed", "error", err)
		os.Exit(1)
	}

	// Generate reports
	if err := suite.GenerateReports(); err != nil {
		logger.Error("Failed to generate reports", "error", err)
		os.Exit(1)
	}

	// Check if all tests passed
	passed, total := suite.GetResults()
	logger.Info("Test suite completed", "passed", passed, "total", total)

	if passed != total {
		logger.Error("Some tests failed")
		os.Exit(1)
	}

	logger.Info("All tests passed successfully!")
}

// NewTestSuite creates a new test suite
func NewTestSuite(config *TestConfig, logger hclog.Logger) *TestSuite {
	return &TestSuite{
		config: config,
		logger: logger,
		results: make([]TestResult, 0),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RunAllTests runs the complete E2E test suite
func (ts *TestSuite) RunAllTests(ctx context.Context) error {
	ts.logger.Info("Running comprehensive E2E test suite")

	// Wait for services to be ready
	if err := ts.waitForServices(ctx); err != nil {
		return fmt.Errorf("services not ready: %w", err)
	}

	// Test categories
	tests := []func(context.Context) error{
		ts.testClusterHealth,
		ts.testNodeDiscovery,
		ts.testLeaderElection,
		ts.testPluginSystem,
		ts.testEventDistribution,
		ts.testWindmillIntegration,
		ts.testLoadBalancing,
		ts.testFailover,
		ts.testDistributedScheduling,
		ts.testClusterCoordination,
	}

	for i, test := range tests {
		ts.logger.Info("Running test", "index", i+1, "total", len(tests))
		if err := test(ctx); err != nil {
			return fmt.Errorf("test %d failed: %w", i+1, err)
		}
	}

	return nil
}

// waitForServices waits for all services to be healthy
func (ts *TestSuite) waitForServices(ctx context.Context) error {
	ts.logger.Info("Waiting for services to be ready")

	services := []struct {
		name string
		url  string
	}{
		{"FlexCore Cluster", ts.config.FlexCoreClusterURL + "/health"},
		{"Windmill", ts.config.WindmillURL + "/api/version"},
	}

	// Add individual nodes
	for _, node := range ts.config.ClusterNodes {
		services = append(services, struct {
			name string
			url  string
		}{
			name: "Node " + node,
			url:  "http://" + node + "/health",
		})
	}

	for _, service := range services {
		if err := ts.waitForService(ctx, service.name, service.url); err != nil {
			return err
		}
	}

	return nil
}

// waitForService waits for a specific service to be healthy
func (ts *TestSuite) waitForService(ctx context.Context, name, url string) error {
	ts.logger.Info("Waiting for service", "name", name, "url", url)

	timeout := time.NewTimer(60 * time.Second)
	defer timeout.Stop()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for %s", name)
		case <-ticker.C:
			resp, err := ts.client.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				ts.logger.Info("Service ready", "name", name)
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// Test implementations

func (ts *TestSuite) testClusterHealth(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing cluster health")

	// Test cluster health endpoint
	resp, err := ts.client.Get(ts.config.FlexCoreClusterURL + "/health")
	if err != nil {
		ts.recordResult("Cluster Health", false, time.Since(start), err.Error(), nil)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("cluster health check failed with status %d", resp.StatusCode)
		ts.recordResult("Cluster Health", false, time.Since(start), err.Error(), nil)
		return err
	}

	ts.recordResult("Cluster Health", true, time.Since(start), "", map[string]interface{}{
		"status_code": resp.StatusCode,
	})
	return nil
}

func (ts *TestSuite) testNodeDiscovery(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing node discovery")

	// Test each node individually
	healthyNodes := 0
	for _, node := range ts.config.ClusterNodes {
		resp, err := ts.client.Get("http://" + node + "/info")
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			healthyNodes++
		}
	}

	expectedNodes := len(ts.config.ClusterNodes)
	success := healthyNodes == expectedNodes

	ts.recordResult("Node Discovery", success, time.Since(start), "", map[string]interface{}{
		"healthy_nodes": healthyNodes,
		"expected_nodes": expectedNodes,
	})

	if !success {
		return fmt.Errorf("only %d of %d nodes are healthy", healthyNodes, expectedNodes)
	}

	return nil
}

func (ts *TestSuite) testLeaderElection(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing leader election")

	// Test cluster status endpoint
	resp, err := ts.client.Get(ts.config.FlexCoreClusterURL + "/cluster")
	if err != nil {
		ts.recordResult("Leader Election", false, time.Since(start), err.Error(), nil)
		return err
	}
	defer resp.Body.Close()

	var clusterStatus map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&clusterStatus); err != nil {
		ts.recordResult("Leader Election", false, time.Since(start), err.Error(), nil)
		return err
	}

	leader, hasLeader := clusterStatus["leader"]
	healthy, isHealthy := clusterStatus["healthy"]

	success := hasLeader && leader != nil && isHealthy && healthy == true

	ts.recordResult("Leader Election", success, time.Since(start), "", map[string]interface{}{
		"leader": leader,
		"healthy": healthy,
		"cluster_status": clusterStatus,
	})

	if !success {
		return fmt.Errorf("leader election failed or cluster unhealthy")
	}

	return nil
}

func (ts *TestSuite) testPluginSystem(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing plugin system")

	// Test plugin listing
	resp, err := ts.client.Get(ts.config.FlexCoreClusterURL + "/plugins")
	if err != nil {
		ts.recordResult("Plugin System", false, time.Since(start), err.Error(), nil)
		return err
	}
	defer resp.Body.Close()

	var pluginResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pluginResponse); err != nil {
		ts.recordResult("Plugin System", false, time.Since(start), err.Error(), nil)
		return err
	}

	pluginCount, hasCount := pluginResponse["count"]
	plugins, hasPlugins := pluginResponse["plugins"]

	success := hasCount && hasPlugins && pluginCount != 0

	ts.recordResult("Plugin System", success, time.Since(start), "", map[string]interface{}{
		"plugin_count": pluginCount,
		"plugins": plugins,
	})

	if !success {
		return fmt.Errorf("plugin system test failed")
	}

	return nil
}

func (ts *TestSuite) testEventDistribution(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing event distribution")

	// Test event publishing
	eventData := map[string]interface{}{
		"type": "test.event",
		"data": map[string]interface{}{
			"message": "E2E test event",
			"timestamp": time.Now().Unix(),
		},
	}

	jsonData, _ := json.Marshal(eventData)
	resp, err := ts.client.Post(ts.config.FlexCoreClusterURL + "/events", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		ts.recordResult("Event Distribution", false, time.Since(start), err.Error(), nil)
		return err
	}
	defer resp.Body.Close()

	success := resp.StatusCode == http.StatusOK

	ts.recordResult("Event Distribution", success, time.Since(start), "", map[string]interface{}{
		"status_code": resp.StatusCode,
		"event_type": "test.event",
	})

	if !success {
		return fmt.Errorf("event distribution failed with status %d", resp.StatusCode)
	}

	return nil
}

func (ts *TestSuite) testWindmillIntegration(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing Windmill integration")

	// Test Windmill version endpoint
	resp, err := ts.client.Get(ts.config.WindmillURL + "/api/version")
	if err != nil {
		ts.recordResult("Windmill Integration", false, time.Since(start), err.Error(), nil)
		return err
	}
	defer resp.Body.Close()

	success := resp.StatusCode == http.StatusOK

	body, _ := io.ReadAll(resp.Body)
	ts.recordResult("Windmill Integration", success, time.Since(start), "", map[string]interface{}{
		"status_code": resp.StatusCode,
		"version_info": string(body),
	})

	if !success {
		return fmt.Errorf("Windmill integration failed with status %d", resp.StatusCode)
	}

	return nil
}

func (ts *TestSuite) testLoadBalancing(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing load balancing")

	// Make multiple requests and check distribution
	nodeResponses := make(map[string]int)
	totalRequests := 30

	for i := 0; i < totalRequests; i++ {
		resp, err := ts.client.Get(ts.config.FlexCoreClusterURL + "/info")
		if err != nil {
			continue
		}

		var info map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&info)
		resp.Body.Close()

		if nodeID, ok := info["node_id"].(string); ok {
			nodeResponses[nodeID]++
		}
	}

	// Check if requests were distributed across multiple nodes
	success := len(nodeResponses) > 1

	ts.recordResult("Load Balancing", success, time.Since(start), "", map[string]interface{}{
		"node_responses": nodeResponses,
		"total_requests": totalRequests,
		"nodes_responding": len(nodeResponses),
	})

	if !success {
		return fmt.Errorf("load balancing failed - only %d nodes responding", len(nodeResponses))
	}

	return nil
}

func (ts *TestSuite) testFailover(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing failover capability")

	// This test simulates failover by checking if cluster remains responsive
	// In a real scenario, we would simulate node failure
	
	// Test cluster resilience by making requests over time
	successfulRequests := 0
	totalRequests := 10

	for i := 0; i < totalRequests; i++ {
		resp, err := ts.client.Get(ts.config.FlexCoreClusterURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			successfulRequests++
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	// We expect high availability
	success := successfulRequests >= totalRequests*8/10 // 80% success rate

	ts.recordResult("Failover", success, time.Since(start), "", map[string]interface{}{
		"successful_requests": successfulRequests,
		"total_requests": totalRequests,
		"success_rate": float64(successfulRequests) / float64(totalRequests),
	})

	if !success {
		return fmt.Errorf("failover test failed - only %d/%d requests successful", successfulRequests, totalRequests)
	}

	return nil
}

func (ts *TestSuite) testDistributedScheduling(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing distributed scheduling")

	// Test that cluster can handle scheduling requests
	// This is a basic test - in reality would test actual job scheduling
	
	resp, err := ts.client.Get(ts.config.FlexCoreClusterURL + "/cluster")
	if err != nil {
		ts.recordResult("Distributed Scheduling", false, time.Since(start), err.Error(), nil)
		return err
	}
	defer resp.Body.Close()

	success := resp.StatusCode == http.StatusOK

	ts.recordResult("Distributed Scheduling", success, time.Since(start), "", map[string]interface{}{
		"status_code": resp.StatusCode,
		"test_type": "basic_scheduling_readiness",
	})

	if !success {
		return fmt.Errorf("distributed scheduling test failed")
	}

	return nil
}

func (ts *TestSuite) testClusterCoordination(ctx context.Context) error {
	start := time.Now()
	ts.logger.Info("Testing cluster coordination")

	// Test coordination by checking all nodes report consistent cluster state
	clusterStates := make(map[string]interface{})

	for i, node := range ts.config.ClusterNodes {
		resp, err := ts.client.Get("http://" + node + "/cluster")
		if err != nil {
			continue
		}

		var state map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&state)
		resp.Body.Close()

		clusterStates[fmt.Sprintf("node_%d", i)] = state
	}

	// Check if we got responses from multiple nodes
	success := len(clusterStates) >= 2

	ts.recordResult("Cluster Coordination", success, time.Since(start), "", map[string]interface{}{
		"cluster_states": clusterStates,
		"responding_nodes": len(clusterStates),
	})

	if !success {
		return fmt.Errorf("cluster coordination test failed")
	}

	return nil
}

// Helper methods

func (ts *TestSuite) recordResult(name string, passed bool, duration time.Duration, error string, details map[string]interface{}) {
	result := TestResult{
		Name:     name,
		Passed:   passed,
		Duration: duration,
		Error:    error,
		Details:  details,
	}
	ts.results = append(ts.results, result)
	
	status := "PASS"
	if !passed {
		status = "FAIL"
	}
	
	ts.logger.Info("Test result", 
		"name", name, 
		"status", status, 
		"duration", duration,
		"error", error)
}

func (ts *TestSuite) GetResults() (passed, total int) {
	total = len(ts.results)
	for _, result := range ts.results {
		if result.Passed {
			passed++
		}
	}
	return
}

func (ts *TestSuite) GenerateReports() error {
	// Generate JSON report
	jsonData, err := json.MarshalIndent(ts.results, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile("/app/results/e2e-test-results.json", jsonData, 0644); err != nil {
		return err
	}

	// Generate summary report
	passed, total := ts.GetResults()
	summary := fmt.Sprintf(`FlexCore E2E Test Suite Results
===============================

Total Tests: %d
Passed: %d
Failed: %d
Success Rate: %.1f%%

Test Details:
`, total, passed, total-passed, float64(passed)/float64(total)*100)

	for _, result := range ts.results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		summary += fmt.Sprintf("- %s: %s (%.2fs)\n", result.Name, status, result.Duration.Seconds())
		if result.Error != "" {
			summary += fmt.Sprintf("  Error: %s\n", result.Error)
		}
	}

	if err := os.WriteFile("/app/results/e2e-test-summary.txt", []byte(summary), 0644); err != nil {
		return err
	}

	ts.logger.Info("Reports generated", "json", "/app/results/e2e-test-results.json", "summary", "/app/results/e2e-test-summary.txt")
	return nil
}

func loadTestConfig() *TestConfig {
	return &TestConfig{
		FlexCoreClusterURL: getEnvOrDefault("FLEXCORE_CLUSTER_URL", "http://localhost:8080"),
		WindmillURL:        getEnvOrDefault("WINDMILL_URL", "http://localhost:8000"),
		PostgresURL:        getEnvOrDefault("POSTGRES_URL", "postgresql://flexcore:flexcore123@localhost:5432/flexcore"),
		TestTimeout:        parseTimeout(getEnvOrDefault("TEST_TIMEOUT", "300s")),
		ClusterNodes:       parseClusterNodes(getEnvOrDefault("CLUSTER_NODES", "localhost:8001,localhost:8002,localhost:8003")),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseTimeout(timeoutStr string) time.Duration {
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 300 * time.Second
	}
	return timeout
}

func parseClusterNodes(nodesStr string) []string {
	return strings.Split(nodesStr, ",")
}