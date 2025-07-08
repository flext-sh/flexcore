//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8080"
)

func TestHealthEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
	assert.NotEmpty(t, health["timestamp"])
	assert.NotEmpty(t, health["version"])
}

func TestInfoEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/info")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var info map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&info)
	require.NoError(t, err)

	assert.Equal(t, "flexcore", info["service"])
	assert.NotEmpty(t, info["version"])
	assert.NotEmpty(t, info["node_id"])
}

func TestEventEndpoint(t *testing.T) {
	event := map[string]interface{}{
		"type":         "test.event",
		"aggregate_id": "test-123",
		"data": map[string]interface{}{
			"message":   "Hello from E2E test",
			"timestamp": time.Now().Unix(),
		},
	}

	eventJSON, err := json.Marshal(event)
	require.NoError(t, err)

	resp, err := http.Post(baseURL+"/events", "application/json", bytes.NewBuffer(eventJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "accepted", result["status"])
}

func TestBatchEventsEndpoint(t *testing.T) {
	events := []map[string]interface{}{
		{
			"type":         "test.batch.event1",
			"aggregate_id": "batch-1",
			"data":         map[string]interface{}{"index": 1},
		},
		{
			"type":         "test.batch.event2",
			"aggregate_id": "batch-2",
			"data":         map[string]interface{}{"index": 2},
		},
	}

	eventsJSON, err := json.Marshal(events)
	require.NoError(t, err)

	resp, err := http.Post(baseURL+"/events/batch", "application/json", bytes.NewBuffer(eventsJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(2), result["total"])
	assert.NotEmpty(t, result["results"])
}

func TestMessageQueueEndpoints(t *testing.T) {
	queueName := "test-queue"

	// Test sending message
	message := map[string]interface{}{
		"content":  "Test message content",
		"priority": 1,
	}

	messageJSON, err := json.Marshal(message)
	require.NoError(t, err)

	resp, err := http.Post(baseURL+"/queues/"+queueName+"/messages", "application/json", bytes.NewBuffer(messageJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var sendResult map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&sendResult)
	require.NoError(t, err)

	assert.NotEmpty(t, sendResult["id"])
	assert.Equal(t, queueName, sendResult["queue"])
	assert.Equal(t, "queued", sendResult["status"])

	// Test receiving messages
	resp, err = http.Get(baseURL + "/queues/" + queueName + "/messages")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var receiveResult map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&receiveResult)
	require.NoError(t, err)

	assert.Equal(t, queueName, receiveResult["queue"])
	assert.NotNil(t, receiveResult["messages"])
	assert.NotNil(t, receiveResult["count"])
}

func TestWorkflowEndpoints(t *testing.T) {
	// Test workflow execution
	workflow := map[string]interface{}{
		"path": "/test/workflow",
		"input": map[string]interface{}{
			"data": "test workflow data",
		},
	}

	workflowJSON, err := json.Marshal(workflow)
	require.NoError(t, err)

	resp, err := http.Post(baseURL+"/workflows/execute", "application/json", bytes.NewBuffer(workflowJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var execResult map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&execResult)
	require.NoError(t, err)

	jobID := execResult["job_id"].(string)
	assert.NotEmpty(t, jobID)
	assert.Equal(t, "started", execResult["status"])

	// Test workflow status
	resp, err = http.Get(baseURL + "/workflows/" + jobID + "/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var statusResult map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&statusResult)
	require.NoError(t, err)

	assert.Equal(t, jobID, statusResult["job_id"])
	assert.NotEmpty(t, statusResult["status"])
}

func TestClusterStatusEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/cluster/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var clusterStatus map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&clusterStatus)
	require.NoError(t, err)

	assert.NotEmpty(t, clusterStatus["node_id"])
	assert.NotEmpty(t, clusterStatus["cluster"])
	assert.Equal(t, "healthy", clusterStatus["health"])
	assert.NotEmpty(t, clusterStatus["nodes"])
}

func TestMetricsEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

}
