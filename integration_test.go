// Integration test for FlexCore - validates complete functionality
//go:build integration
// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPort    = "8090"
	baseURL     = "http://localhost:8090"
	testTimeout = 30 * time.Second
)

// TestMain controls the setup and teardown for integration tests
func TestMain(m *testing.M) {
	// Build the application
	cmd := exec.Command("go", "build", "-o", "build/flexcore-test", "./cmd/flexcore")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to build application: %v\n", err)
		os.Exit(1)
	}

	// Start the server
	serverCmd := exec.Command("./build/flexcore-test", "--port", testPort)
	if err := serverCmd.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}

	// Wait for server to be ready
	ready := false
	for i := 0; i < 30; i++ {
		if resp, err := http.Get(baseURL + "/health"); err == nil && resp.StatusCode == 200 {
			ready = true
			resp.Body.Close()
			break
		}
		time.Sleep(time.Second)
	}

	if !ready {
		serverCmd.Process.Kill()
		fmt.Println("Server failed to start within timeout")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	serverCmd.Process.Kill()
	os.Remove("build/flexcore-test")

	os.Exit(code)
}

func TestIntegration_HealthEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, "dev", health["version"])
	assert.NotNil(t, health["timestamp"])
}

func TestIntegration_InfoEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/info")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var info map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&info)
	require.NoError(t, err)

	assert.Equal(t, "flexcore", info["service"])
	assert.Equal(t, "dev", info["version"])
	assert.Equal(t, "flexcore-cluster", info["cluster"])
	assert.NotEmpty(t, info["node_id"])
}

func TestIntegration_MetricsEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
}

func TestIntegration_SendEvent(t *testing.T) {
	eventData := map[string]interface{}{
		"type":         "test-event",
		"aggregate_id": "test-aggregate",
		"data": map[string]interface{}{
			"test_key": "test_value",
		},
	}

	body, _ := json.Marshal(eventData)
	resp, err := http.Post(baseURL+"/events", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "accepted", result["status"])
	assert.NotEmpty(t, result["id"])
}

func TestIntegration_BatchEvents(t *testing.T) {
	events := []map[string]interface{}{
		{
			"type":         "batch-event-1",
			"aggregate_id": "batch-aggregate-1",
			"data":         map[string]interface{}{"batch": 1},
		},
		{
			"type":         "batch-event-2",
			"aggregate_id": "batch-aggregate-2",
			"data":         map[string]interface{}{"batch": 2},
		},
	}

	body, _ := json.Marshal(events)
	resp, err := http.Post(baseURL+"/events/batch", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(2), result["total"])

	results := result["results"].([]interface{})
	assert.Len(t, results, 2)

	for _, r := range results {
		res := r.(map[string]interface{})
		assert.Equal(t, "accepted", res["status"])
		assert.NotEmpty(t, res["id"])
	}
}

func TestIntegration_SendMessage(t *testing.T) {
	messageData := map[string]interface{}{
		"test_message": "hello",
		"priority":     "high",
	}

	body, _ := json.Marshal(messageData)
	resp, err := http.Post(baseURL+"/queues/test-queue/messages", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "queued", result["status"])
	assert.Equal(t, "test-queue", result["queue"])
	assert.NotEmpty(t, result["id"])
}

func TestIntegration_ReceiveMessages(t *testing.T) {
	// First send a message
	messageData := map[string]interface{}{
		"test_message": "receive_test",
	}

	body, _ := json.Marshal(messageData)
	sendResp, err := http.Post(baseURL+"/queues/receive-test-queue/messages", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	sendResp.Body.Close()

	// Then receive messages
	resp, err := http.Get(baseURL + "/queues/receive-test-queue/messages")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "receive-test-queue", result["queue"])
	assert.GreaterOrEqual(t, int(result["count"].(float64)), 1)

	messages := result["messages"].([]interface{})
	assert.GreaterOrEqual(t, len(messages), 1)

	// Check the message we sent
	message := messages[0].(map[string]interface{})
	data := message["data"].(map[string]interface{})
	assert.Equal(t, "receive_test", data["test_message"])
}

func TestIntegration_ExecuteWorkflow(t *testing.T) {
	workflowData := map[string]interface{}{
		"path": "test-workflow",
		"input": map[string]interface{}{
			"workflow_param": "test_value",
		},
	}

	body, _ := json.Marshal(workflowData)
	resp, err := http.Post(baseURL+"/workflows/execute", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "started", result["status"])
	jobID := result["job_id"].(string)
	assert.NotEmpty(t, jobID)

	// Test workflow status
	statusResp, err := http.Get(baseURL + "/workflows/" + jobID + "/status")
	require.NoError(t, err)
	defer statusResp.Body.Close()

	assert.Equal(t, http.StatusOK, statusResp.StatusCode)

	var status map[string]interface{}
	err = json.NewDecoder(statusResp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "test-workflow", status["path"])
	assert.Equal(t, "running", status["status"])
}

func TestIntegration_ClusterStatus(t *testing.T) {
	resp, err := http.Get(baseURL + "/cluster/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "flexcore-cluster", status["cluster_name"])
	assert.Equal(t, "healthy", status["status"])
	assert.NotEmpty(t, status["node_id"])
	assert.GreaterOrEqual(t, int(status["cluster_size"].(float64)), 1)
}

func TestIntegration_ErrorHandling(t *testing.T) {
	// Test invalid JSON
	resp, err := http.Post(baseURL+"/events", "application/json", bytes.NewBufferString("invalid json"))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test missing required fields
	emptyEvent := map[string]interface{}{}
	body, _ := json.Marshal(emptyEvent)
	resp, err = http.Post(baseURL+"/events", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test nonexistent workflow status
	resp, err = http.Get(baseURL + "/workflows/nonexistent/status")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	const numRequests = 10
	
	// Create a channel to collect results
	results := make(chan bool, numRequests)
	
	// Send concurrent health check requests
	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get(baseURL + "/health")
			if err != nil {
				results <- false
				return
			}
			defer resp.Body.Close()
			results <- resp.StatusCode == 200
		}()
	}
	
	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if success := <-results; success {
			successCount++
		}
	}
	
	assert.Equal(t, numRequests, successCount, "All concurrent requests should succeed")
}

// Benchmark tests
func BenchmarkIntegration_HealthEndpoint(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkIntegration_SendEvent(b *testing.B) {
	eventData := map[string]interface{}{
		"type":         "benchmark-event",
		"aggregate_id": "benchmark-aggregate",
		"data":         map[string]interface{}{"benchmark": true},
	}
	body, _ := json.Marshal(eventData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(baseURL+"/events", "application/json", bytes.NewBuffer(body))
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}