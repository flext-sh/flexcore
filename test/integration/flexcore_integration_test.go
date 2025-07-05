//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/flext/flexcore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlexCoreIntegration(t *testing.T) {
	ctx := context.Background()
	
	// Create configuration
	config := domain.DefaultConfig()
	config.NodeID = "test-node-1"
	
	// Create FlexCore instance
	result := domain.NewFlexCore(config)
	require.True(t, result.IsSuccess(), "Failed to create FlexCore instance")
	
	core := result.Value()
	
	// Test Start
	startResult := core.Start(ctx)
	assert.True(t, startResult.IsSuccess(), "Failed to start FlexCore")
	
	// Test event sending
	event := domain.NewEvent("test.event", "test-aggregate-123", map[string]interface{}{
		"message": "Hello from integration test",
		"timestamp": time.Now().Unix(),
	})
	
	eventResult := core.SendEvent(ctx, event)
	assert.True(t, eventResult.IsSuccess(), "Failed to send event")
	
	// Test message queue
	message := domain.NewMessage("test-queue", map[string]interface{}{
		"content": "Test message content",
		"priority": 1,
	})
	
	sendResult := core.SendMessage(ctx, "test-queue", message)
	assert.True(t, sendResult.IsSuccess(), "Failed to send message")
	
	// Test receive messages
	receiveResult := core.ReceiveMessages(ctx, "test-queue", 10)
	assert.True(t, receiveResult.IsSuccess(), "Failed to receive messages")
	
	// Test workflow execution
	workflowResult := core.ExecuteWorkflow(ctx, "/test/workflow", map[string]interface{}{
		"input": "test data",
	})
	assert.True(t, workflowResult.IsSuccess(), "Failed to execute workflow")
	
	jobID := workflowResult.Value()
	assert.NotEmpty(t, jobID, "Job ID should not be empty")
	
	// Test workflow status
	statusResult := core.GetWorkflowStatus(ctx, jobID)
	assert.True(t, statusResult.IsSuccess(), "Failed to get workflow status")
	
	status := statusResult.Value()
	assert.Equal(t, jobID, status.JobID, "Job ID should match")
	
	// Test cluster status
	clusterResult := core.GetClusterStatus(ctx)
	assert.True(t, clusterResult.IsSuccess(), "Failed to get cluster status")
	
	cluster := clusterResult.Value()
	assert.Equal(t, config.NodeID, cluster.NodeID, "Node ID should match")
	assert.Equal(t, config.ClusterName, cluster.Cluster, "Cluster name should match")
	assert.NotEmpty(t, cluster.Nodes, "Should have at least one node")
	
	// Test Stop
	stopResult := core.Stop(ctx)
	assert.True(t, stopResult.IsSuccess(), "Failed to stop FlexCore")
}

func TestFlexCoreClusterIntegration(t *testing.T) {
	ctx := context.Background()
	
	// Create multiple FlexCore instances to simulate cluster
	configs := []*domain.FlexCoreConfig{
		{NodeID: "node-1", ClusterName: "test-cluster"},
		{NodeID: "node-2", ClusterName: "test-cluster"},
		{NodeID: "node-3", ClusterName: "test-cluster"},
	}
	
	var cores []*domain.FlexCore
	
	// Start all nodes
	for _, config := range configs {
		result := domain.NewFlexCore(config)
		require.True(t, result.IsSuccess(), "Failed to create FlexCore instance")
		
		core := result.Value()
		cores = append(cores, core)
		
		startResult := core.Start(ctx)
		require.True(t, startResult.IsSuccess(), "Failed to start FlexCore node")
	}
	
	// Test cross-node communication
	event := domain.NewEvent("cluster.test", "cluster-123", map[string]interface{}{
		"source_node": "node-1",
		"target_node": "node-2",
		"message": "Cross-node test message",
	})
	
	// Send event from first node
	eventResult := cores[0].SendEvent(ctx, event)
	assert.True(t, eventResult.IsSuccess(), "Failed to send cross-node event")
	
	// Verify cluster status from different nodes
	for i, core := range cores {
		clusterResult := core.GetClusterStatus(ctx)
		assert.True(t, clusterResult.IsSuccess(), "Failed to get cluster status from node %d", i)
		
		cluster := clusterResult.Value()
		assert.Equal(t, "test-cluster", cluster.Cluster, "Cluster name should match")
		assert.Equal(t, configs[i].NodeID, cluster.NodeID, "Node ID should match")
	}
	
	// Stop all nodes
	for i, core := range cores {
		stopResult := core.Stop(ctx)
		assert.True(t, stopResult.IsSuccess(), "Failed to stop FlexCore node %d", i)
	}
}