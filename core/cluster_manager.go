// Package core provides distributed cluster management using Windmill
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flext/flexcore/infrastructure/windmill"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
)

// ClusterManager handles distributed cluster coordination using Windmill
type ClusterManager struct {
	windmillClient *windmill.Client
	config         *FlexCoreConfig
	nodeInfo       *NodeInfo
	clusterState   *ClusterState
	metrics        *ClusterMetrics
	mu             sync.RWMutex
	running        bool
	stopChan       chan struct{}
	heartbeatTicker *time.Ticker
}

// ClusterMetrics tracks cluster metrics
type ClusterMetrics struct {
	NodesJoined       int64
	NodesLeft         int64
	LeaderElections   int64
	HeartbeatsSent    int64
	HeartbeatsReceived int64
	ClusterOperations int64
	NetworkLatency    int64
}

// ClusterState represents the current state of the cluster
type ClusterState struct {
	ClusterName   string              `json:"cluster_name"`
	Nodes         map[string]*NodeInfo `json:"nodes"`
	LeaderNode    string              `json:"leader_node"`
	Epoch         int64               `json:"epoch"`
	LastUpdated   time.Time           `json:"last_updated"`
	CustomState   map[string]interface{} `json:"custom_state"`
	mu            sync.RWMutex
}

// ClusterOperation represents a cluster operation
type ClusterOperation struct {
	ID          string                 `json:"id"`
	Type        ClusterOperationType   `json:"type"`
	NodeID      string                 `json:"node_id"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time             `json:"timestamp"`
	Completed   bool                   `json:"completed"`
	Error       string                 `json:"error,omitempty"`
}

// ClusterOperationType represents types of cluster operations
type ClusterOperationType string

const (
	ClusterOpJoin          ClusterOperationType = "join"
	ClusterOpLeave         ClusterOperationType = "leave"
	ClusterOpHeartbeat     ClusterOperationType = "heartbeat"
	ClusterOpLeaderElect   ClusterOperationType = "leader_elect"
	ClusterOpStateSync     ClusterOperationType = "state_sync"
	ClusterOpCustom        ClusterOperationType = "custom"
)

// NewClusterManager creates a new cluster manager
func NewClusterManager(windmillClient *windmill.Client, config *FlexCoreConfig) *ClusterManager {
	nodeInfo := &NodeInfo{
		ID:       config.NodeID,
		Address:  fmt.Sprintf("node-%s", config.NodeID),
		Status:   "initializing",
		LastSeen: time.Now(),
		Metrics:  make(map[string]interface{}),
	}

	clusterState := &ClusterState{
		ClusterName: config.ClusterName,
		Nodes:       make(map[string]*NodeInfo),
		Epoch:       0,
		LastUpdated: time.Now(),
		CustomState: make(map[string]interface{}),
	}

	return &ClusterManager{
		windmillClient: windmillClient,
		config:         config,
		nodeInfo:       nodeInfo,
		clusterState:   clusterState,
		metrics:        &ClusterMetrics{},
		stopChan:       make(chan struct{}),
	}
}

// Start starts the cluster manager
func (cm *ClusterManager) Start(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running {
		return errors.ValidationError("cluster manager already running")
	}

	// Initialize cluster workflows
	if err := cm.initializeClusterWorkflows(ctx); err != nil {
		return fmt.Errorf("failed to initialize cluster workflows: %w", err)
	}

	// Join cluster
	if err := cm.joinCluster(ctx); err != nil {
		return fmt.Errorf("failed to join cluster: %w", err)
	}

	// Start heartbeat process
	cm.heartbeatTicker = time.NewTicker(10 * time.Second)
	go cm.runHeartbeatLoop()

	// Start cluster state synchronization
	go cm.runStateSyncLoop()

	cm.running = true
	return nil
}

// Stop stops the cluster manager
func (cm *ClusterManager) Stop(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return nil
	}

	// Leave cluster gracefully
	cm.leaveCluster(ctx)

	// Stop background processes
	close(cm.stopChan)
	if cm.heartbeatTicker != nil {
		cm.heartbeatTicker.Stop()
	}

	cm.running = false
	return nil
}

// GetStatus gets the current cluster status
func (cm *ClusterManager) GetStatus(ctx context.Context) result.Result[*ClusterStatus] {
	cm.clusterState.mu.RLock()
	defer cm.clusterState.mu.RUnlock()

	activeNodes := make([]NodeInfo, 0, len(cm.clusterState.Nodes))
	for _, node := range cm.clusterState.Nodes {
		if node.Status == "active" {
			activeNodes = append(activeNodes, *node)
		}
	}

	status := &ClusterStatus{
		ClusterName:   cm.clusterState.ClusterName,
		NodeID:        cm.nodeInfo.ID,
		NodeCount:     len(activeNodes),
		ActiveNodes:   activeNodes,
		LeaderNode:    cm.clusterState.LeaderNode,
		LastHeartbeat: cm.nodeInfo.LastSeen,
		CustomStatus:  cm.clusterState.CustomState,
	}

	return result.Success(status)
}

// JoinCluster manually joins the cluster
func (cm *ClusterManager) JoinCluster(ctx context.Context) result.Result[bool] {
	if err := cm.joinCluster(ctx); err != nil {
		return result.Failure[bool](err)
	}
	return result.Success(true)
}

// LeaveCluster manually leaves the cluster
func (cm *ClusterManager) LeaveCluster(ctx context.Context) result.Result[bool] {
	if err := cm.leaveCluster(ctx); err != nil {
		return result.Failure[bool](err)
	}
	return result.Success(true)
}

// ElectLeader initiates leader election
func (cm *ClusterManager) ElectLeader(ctx context.Context) result.Result[string] {
	atomic.AddInt64(&cm.metrics.LeaderElections, 1)

	// Execute leader election using Windmill workflow
	workflowPath := fmt.Sprintf("cluster/%s/leader_election", cm.config.ClusterName)
	
	input := map[string]interface{}{
		"cluster_name": cm.config.ClusterName,
		"node_id":      cm.nodeInfo.ID,
		"node_info":    cm.nodeInfo,
		"epoch":        cm.clusterState.Epoch + 1,
	}

	runReq := windmill.RunWorkflowRequest{
		Args: input,
		Tag:  &cm.config.ClusterName,
	}

	jobResult := cm.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
	if jobResult.IsFailure() {
		return result.Failure[string](jobResult.Error())
	}

	// Wait for election completion
	job := jobResult.Value()
	finalJobResult := cm.windmillClient.WaitForJob(ctx, job.ID, 10*time.Second)
	if finalJobResult.IsFailure() {
		return result.Failure[string](finalJobResult.Error())
	}

	finalJob := finalJobResult.Value()
	if finalJob.Result == nil {
		return result.Failure[string](errors.InternalError("leader election failed"))
	}

	// Parse election result
	var electionResult map[string]interface{}
	resultBytes, err := json.Marshal(finalJob.Result)
	if err != nil {
		return result.Failure[string](fmt.Errorf("failed to marshal election result: %w", err))
	}

	if err := json.Unmarshal(resultBytes, &electionResult); err != nil {
		return result.Failure[string](fmt.Errorf("failed to unmarshal election result: %w", err))
	}

	if leaderID, ok := electionResult["leader_id"].(string); ok {
		cm.clusterState.mu.Lock()
		cm.clusterState.LeaderNode = leaderID
		cm.clusterState.Epoch++
		cm.clusterState.mu.Unlock()
		return result.Success(leaderID)
	}

	return result.Failure[string](errors.InternalError("invalid election result"))
}

// SynchronizeState synchronizes cluster state
func (cm *ClusterManager) SynchronizeState(ctx context.Context) result.Result[bool] {
	return cm.synchronizeClusterState(ctx)
}

// BroadcastOperation broadcasts an operation to all cluster nodes
func (cm *ClusterManager) BroadcastOperation(ctx context.Context, operation *ClusterOperation) result.Result[bool] {
	atomic.AddInt64(&cm.metrics.ClusterOperations, 1)

	// Execute broadcast using Windmill workflow
	workflowPath := fmt.Sprintf("cluster/%s/broadcast", cm.config.ClusterName)
	
	input := map[string]interface{}{
		"operation":    operation,
		"cluster_name": cm.config.ClusterName,
		"sender_node":  cm.nodeInfo.ID,
	}

	runReq := windmill.RunWorkflowRequest{
		Args: input,
		Tag:  &cm.config.ClusterName,
	}

	jobResult := cm.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
	if jobResult.IsFailure() {
		return result.Failure[bool](jobResult.Error())
	}

	return result.Success(true)
}

// GetNodeInfo gets information about a specific node
func (cm *ClusterManager) GetNodeInfo(nodeID string) result.Result[*NodeInfo] {
	cm.clusterState.mu.RLock()
	defer cm.clusterState.mu.RUnlock()

	node, exists := cm.clusterState.Nodes[nodeID]
	if !exists {
		return result.Failure[*NodeInfo](errors.NotFoundError("node not found: " + nodeID))
	}

	return result.Success(node)
}

// IsLeader checks if this node is the cluster leader
func (cm *ClusterManager) IsLeader() bool {
	cm.clusterState.mu.RLock()
	defer cm.clusterState.mu.RUnlock()
	return cm.clusterState.LeaderNode == cm.nodeInfo.ID
}

// GetMetrics gets cluster metrics
func (cm *ClusterManager) GetMetrics() *ClusterMetrics {
	return &ClusterMetrics{
		NodesJoined:        atomic.LoadInt64(&cm.metrics.NodesJoined),
		NodesLeft:          atomic.LoadInt64(&cm.metrics.NodesLeft),
		LeaderElections:    atomic.LoadInt64(&cm.metrics.LeaderElections),
		HeartbeatsSent:     atomic.LoadInt64(&cm.metrics.HeartbeatsSent),
		HeartbeatsReceived: atomic.LoadInt64(&cm.metrics.HeartbeatsReceived),
		ClusterOperations:  atomic.LoadInt64(&cm.metrics.ClusterOperations),
		NetworkLatency:     atomic.LoadInt64(&cm.metrics.NetworkLatency),
	}
}

// PerformMaintenance performs periodic cluster maintenance
func (cm *ClusterManager) PerformMaintenance(ctx context.Context) {
	// Remove inactive nodes
	cm.cleanupInactiveNodes()
	
	// Check leader health
	cm.checkLeaderHealth(ctx)
	
	// Update node metrics
	cm.updateNodeMetrics()
}

// SetCustomState sets custom cluster state
func (cm *ClusterManager) SetCustomState(key string, value interface{}) {
	cm.clusterState.mu.Lock()
	defer cm.clusterState.mu.Unlock()
	cm.clusterState.CustomState[key] = value
}

// GetCustomState gets custom cluster state
func (cm *ClusterManager) GetCustomState(key string) (interface{}, bool) {
	cm.clusterState.mu.RLock()
	defer cm.clusterState.mu.RUnlock()
	value, exists := cm.clusterState.CustomState[key]
	return value, exists
}

// Private methods

func (cm *ClusterManager) initializeClusterWorkflows(ctx context.Context) error {
	// Create cluster coordination workflows
	workflows := []*windmill.WorkflowDefinition{
		cm.createJoinClusterWorkflow(),
		cm.createLeaveClusterWorkflow(),
		cm.createHeartbeatWorkflow(),
		cm.createLeaderElectionWorkflow(),
		cm.createStateSyncWorkflow(),
		cm.createBroadcastWorkflow(),
	}

	for _, workflow := range workflows {
		createReq := windmill.CreateWorkflowRequest{
			Path:        workflow.Path,
			Summary:     workflow.Name,
			Description: workflow.Description,
			Value:       cm.convertClusterWorkflowToWindmill(workflow),
			Schema:      cm.generateClusterSchema(),
		}

		createResult := cm.windmillClient.CreateWorkflow(ctx, createReq)
		if createResult.IsFailure() {
			return createResult.Error()
		}
	}

	return nil
}

func (cm *ClusterManager) joinCluster(ctx context.Context) error {
	atomic.AddInt64(&cm.metrics.NodesJoined, 1)

	// Update node status
	cm.nodeInfo.Status = "joining"
	cm.nodeInfo.LastSeen = time.Now()

	// Execute join workflow
	workflowPath := fmt.Sprintf("cluster/%s/join", cm.config.ClusterName)
	
	input := map[string]interface{}{
		"cluster_name": cm.config.ClusterName,
		"node_info":    cm.nodeInfo,
	}

	runReq := windmill.RunWorkflowRequest{
		Args: input,
		Tag:  &cm.config.ClusterName,
	}

	jobResult := cm.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
	if jobResult.IsFailure() {
		return jobResult.Error()
	}

	// Update local state
	cm.clusterState.mu.Lock()
	cm.clusterState.Nodes[cm.nodeInfo.ID] = cm.nodeInfo
	cm.nodeInfo.Status = "active"
	cm.clusterState.mu.Unlock()

	// Trigger leader election if no leader exists
	if cm.clusterState.LeaderNode == "" {
		cm.ElectLeader(ctx)
	}

	return nil
}

func (cm *ClusterManager) leaveCluster(ctx context.Context) error {
	atomic.AddInt64(&cm.metrics.NodesLeft, 1)

	// Update node status
	cm.nodeInfo.Status = "leaving"

	// Execute leave workflow
	workflowPath := fmt.Sprintf("cluster/%s/leave", cm.config.ClusterName)
	
	input := map[string]interface{}{
		"cluster_name": cm.config.ClusterName,
		"node_id":      cm.nodeInfo.ID,
	}

	runReq := windmill.RunWorkflowRequest{
		Args: input,
		Tag:  &cm.config.ClusterName,
	}

	jobResult := cm.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
	if jobResult.IsFailure() {
		return jobResult.Error()
	}

	// Update local state
	cm.clusterState.mu.Lock()
	delete(cm.clusterState.Nodes, cm.nodeInfo.ID)
	if cm.clusterState.LeaderNode == cm.nodeInfo.ID {
		cm.clusterState.LeaderNode = ""
	}
	cm.clusterState.mu.Unlock()

	return nil
}

func (cm *ClusterManager) runHeartbeatLoop() {
	for {
		select {
		case <-cm.stopChan:
			return
		case <-cm.heartbeatTicker.C:
			ctx := context.Background()
			cm.sendHeartbeat(ctx)
		}
	}
}

func (cm *ClusterManager) sendHeartbeat(ctx context.Context) {
	atomic.AddInt64(&cm.metrics.HeartbeatsSent, 1)

	// Update node metrics
	cm.nodeInfo.LastSeen = time.Now()
	cm.nodeInfo.Metrics["uptime"] = time.Since(cm.nodeInfo.LastSeen).Seconds()

	// Execute heartbeat workflow
	workflowPath := fmt.Sprintf("cluster/%s/heartbeat", cm.config.ClusterName)
	
	input := map[string]interface{}{
		"cluster_name": cm.config.ClusterName,
		"node_info":    cm.nodeInfo,
		"timestamp":    time.Now(),
	}

	runReq := windmill.RunWorkflowRequest{
		Args: input,
		Tag:  &cm.config.ClusterName,
	}

	cm.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
}

func (cm *ClusterManager) runStateSyncLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopChan:
			return
		case <-ticker.C:
			ctx := context.Background()
			cm.synchronizeClusterState(ctx)
		}
	}
}

func (cm *ClusterManager) synchronizeClusterState(ctx context.Context) result.Result[bool] {
	// Execute state sync workflow
	workflowPath := fmt.Sprintf("cluster/%s/state_sync", cm.config.ClusterName)
	
	input := map[string]interface{}{
		"cluster_name": cm.config.ClusterName,
		"node_id":      cm.nodeInfo.ID,
		"current_state": cm.clusterState,
	}

	runReq := windmill.RunWorkflowRequest{
		Args: input,
		Tag:  &cm.config.ClusterName,
	}

	jobResult := cm.windmillClient.RunWorkflow(ctx, workflowPath, runReq)
	if jobResult.IsFailure() {
		return result.Failure[bool](jobResult.Error())
	}

	return result.Success(true)
}

func (cm *ClusterManager) cleanupInactiveNodes() {
	cm.clusterState.mu.Lock()
	defer cm.clusterState.mu.Unlock()

	cutoff := time.Now().Add(-60 * time.Second) // 60 seconds timeout
	
	for _, node := range cm.clusterState.Nodes {
		if node.LastSeen.Before(cutoff) && node.Status == "active" {
			node.Status = "inactive"
		}
	}
}

func (cm *ClusterManager) checkLeaderHealth(ctx context.Context) {
	cm.clusterState.mu.RLock()
	leaderID := cm.clusterState.LeaderNode
	cm.clusterState.mu.RUnlock()

	if leaderID == "" {
		// No leader, trigger election
		cm.ElectLeader(ctx)
		return
	}

	// Check if leader is still active
	if leaderNode, exists := cm.clusterState.Nodes[leaderID]; exists {
		if leaderNode.Status != "active" {
			// Leader is inactive, trigger new election
			cm.ElectLeader(ctx)
		}
	}
}

func (cm *ClusterManager) updateNodeMetrics() {
	cm.nodeInfo.Metrics["timestamp"] = time.Now().Unix()
	cm.nodeInfo.Metrics["heartbeats_sent"] = atomic.LoadInt64(&cm.metrics.HeartbeatsSent)
	cm.nodeInfo.Metrics["cluster_operations"] = atomic.LoadInt64(&cm.metrics.ClusterOperations)
}

func (cm *ClusterManager) createJoinClusterWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        fmt.Sprintf("cluster/%s/join", cm.config.ClusterName),
		Name:        "Join Cluster",
		Description: "Handles node joining the cluster",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "join_cluster",
				Name:     "Join Cluster",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(cluster_name: str, node_info: dict):
    """Handle node joining cluster"""
    import time
    
    # Register node in distributed state
    # In real implementation, would use distributed storage
    
    result = {
        "status": "joined",
        "cluster_name": cluster_name,
        "node_id": node_info["id"],
        "join_time": time.time()
    }
    
    return result
`,
				Input: map[string]interface{}{
					"cluster_name": "{{ input.cluster_name }}",
					"node_info":    "{{ input.node_info }}",
				},
			},
		},
		Timeout: time.Minute * 2,
	}
}

func (cm *ClusterManager) createLeaveClusterWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        fmt.Sprintf("cluster/%s/leave", cm.config.ClusterName),
		Name:        "Leave Cluster",
		Description: "Handles node leaving the cluster",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "leave_cluster",
				Name:     "Leave Cluster",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(cluster_name: str, node_id: str):
    """Handle node leaving cluster"""
    import time
    
    # Remove node from distributed state
    # In real implementation, would use distributed storage
    
    result = {
        "status": "left",
        "cluster_name": cluster_name,
        "node_id": node_id,
        "leave_time": time.time()
    }
    
    return result
`,
				Input: map[string]interface{}{
					"cluster_name": "{{ input.cluster_name }}",
					"node_id":      "{{ input.node_id }}",
				},
			},
		},
		Timeout: time.Minute,
	}
}

func (cm *ClusterManager) createHeartbeatWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        fmt.Sprintf("cluster/%s/heartbeat", cm.config.ClusterName),
		Name:        "Heartbeat",
		Description: "Handles node heartbeat",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "heartbeat",
				Name:     "Process Heartbeat",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(cluster_name: str, node_info: dict, timestamp: float):
    """Process node heartbeat"""
    import time
    
    # Update node status in distributed state
    # In real implementation, would use distributed storage
    
    result = {
        "status": "heartbeat_processed",
        "cluster_name": cluster_name,
        "node_id": node_info["id"],
        "timestamp": timestamp,
        "processed_at": time.time()
    }
    
    return result
`,
				Input: map[string]interface{}{
					"cluster_name": "{{ input.cluster_name }}",
					"node_info":    "{{ input.node_info }}",
					"timestamp":    "{{ input.timestamp }}",
				},
			},
		},
		Timeout: time.Minute,
	}
}

func (cm *ClusterManager) createLeaderElectionWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        fmt.Sprintf("cluster/%s/leader_election", cm.config.ClusterName),
		Name:        "Leader Election",
		Description: "Handles cluster leader election",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "elect_leader",
				Name:     "Elect Leader",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(cluster_name: str, node_id: str, node_info: dict, epoch: int):
    """Elect cluster leader"""
    import time
    import random
    
    # Simple leader election - in real implementation would use Raft/Paxos
    # For now, elect the node with the lowest ID
    candidate_nodes = [node_id]  # Would query all active nodes
    
    leader_id = min(candidate_nodes)
    
    result = {
        "status": "election_completed",
        "cluster_name": cluster_name,
        "leader_id": leader_id,
        "epoch": epoch,
        "election_time": time.time()
    }
    
    return result
`,
				Input: map[string]interface{}{
					"cluster_name": "{{ input.cluster_name }}",
					"node_id":      "{{ input.node_id }}",
					"node_info":    "{{ input.node_info }}",
					"epoch":        "{{ input.epoch }}",
				},
			},
		},
		Timeout: time.Minute * 2,
	}
}

func (cm *ClusterManager) createStateSyncWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        fmt.Sprintf("cluster/%s/state_sync", cm.config.ClusterName),
		Name:        "State Sync",
		Description: "Synchronizes cluster state",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "sync_state",
				Name:     "Sync State",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(cluster_name: str, node_id: str, current_state: dict):
    """Synchronize cluster state"""
    import time
    
    # Sync with distributed state store
    # In real implementation, would use consensus algorithm
    
    result = {
        "status": "state_synchronized",
        "cluster_name": cluster_name,
        "node_id": node_id,
        "sync_time": time.time(),
        "synchronized_state": current_state
    }
    
    return result
`,
				Input: map[string]interface{}{
					"cluster_name":  "{{ input.cluster_name }}",
					"node_id":       "{{ input.node_id }}",
					"current_state": "{{ input.current_state }}",
				},
			},
		},
		Timeout: time.Minute,
	}
}

func (cm *ClusterManager) createBroadcastWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        fmt.Sprintf("cluster/%s/broadcast", cm.config.ClusterName),
		Name:        "Broadcast Operation",
		Description: "Broadcasts operations to cluster nodes",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "broadcast_operation",
				Name:     "Broadcast Operation",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(operation: dict, cluster_name: str, sender_node: str):
    """Broadcast operation to cluster nodes"""
    import time
    
    # Broadcast to all active nodes
    # In real implementation, would use message queuing or gossip protocol
    
    result = {
        "status": "broadcast_completed",
        "cluster_name": cluster_name,
        "operation_id": operation["id"],
        "sender_node": sender_node,
        "broadcast_time": time.time(),
        "nodes_reached": []  # Would contain actual node list
    }
    
    return result
`,
				Input: map[string]interface{}{
					"operation":    "{{ input.operation }}",
					"cluster_name": "{{ input.cluster_name }}",
					"sender_node":  "{{ input.sender_node }}",
				},
			},
		},
		Timeout: time.Minute,
	}
}

func (cm *ClusterManager) convertClusterWorkflowToWindmill(workflow *windmill.WorkflowDefinition) windmill.WorkflowValue {
	modules := make([]windmill.WorkflowModule, len(workflow.Steps))
	
	for i, step := range workflow.Steps {
		modules[i] = windmill.WorkflowModule{
			ID: step.ID,
			Value: windmill.ModuleValue{
				Type:     string(step.Type),
				Content:  step.Script,
				Language: step.Language,
				Input:    step.Input,
			},
			Summary: step.Name,
		}
	}
	
	return windmill.WorkflowValue{
		Modules: modules,
	}
}

func (cm *ClusterManager) generateClusterSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]interface{}{
			"cluster_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the cluster",
			},
			"node_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the node",
			},
		},
	}
}