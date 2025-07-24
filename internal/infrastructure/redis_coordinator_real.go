package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/flext/flexcore/pkg/logging"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RealRedisCoordinator implements distributed cluster coordination using actual Redis
type RealRedisCoordinator struct {
	nodeID     string
	client     *redis.Client
	mu         sync.RWMutex
	logger     logging.LoggerInterface
	healthTick *time.Ticker
	running    bool
	
	// Redis keys
	nodePrefix    string
	lockPrefix    string
	leaderPrefix  string
	workflowPrefix string
}

// NewRealRedisCoordinator creates a new Redis-based distributed coordinator
func NewRealRedisCoordinator(redisURL string, logger logging.LoggerInterface) *RealRedisCoordinator {
	// Parse Redis URL or use defaults
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fall back to default options
		opts = &redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}
	}
	
	client := redis.NewClient(opts)
	nodeID := uuid.New().String()
	
	return &RealRedisCoordinator{
		nodeID:         nodeID,
		client:         client,
		logger:         logger,
		nodePrefix:     "flexcore:nodes:",
		lockPrefix:     "flexcore:locks:",
		leaderPrefix:   "flexcore:leaders:",
		workflowPrefix: "flexcore:workflows:",
	}
}

// GetNodeID returns the current node ID
func (rc *RealRedisCoordinator) GetNodeID() string {
	return rc.nodeID
}

// CoordinateExecution coordinates pipeline execution across cluster nodes using Redis
func (rc *RealRedisCoordinator) CoordinateExecution(ctx context.Context, workflowID string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	rc.logger.Info("Coordinating distributed execution with Redis",
		zap.String("workflow_id", workflowID),
		zap.String("node_id", rc.nodeID))
	
	// 1. Register this node as active
	if err := rc.registerNode(ctx); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}
	
	// 2. Acquire distributed lock for workflow
	lockKey := rc.lockPrefix + workflowID
	acquired, err := rc.acquireLock(ctx, lockKey, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to acquire workflow lock: %w", err)
	}
	defer rc.releaseLock(ctx, lockKey)
	
	if !acquired {
		return fmt.Errorf("another node is already processing workflow %s", workflowID)
	}
	
	// 3. Elect leader for this workflow
	leaderKey := rc.leaderPrefix + workflowID
	isLeader, err := rc.electLeader(ctx, leaderKey, workflowID)
	if err != nil {
		return fmt.Errorf("leader election failed: %w", err)
	}
	
	rc.logger.Info("Leadership determined for workflow",
		zap.String("workflow_id", workflowID),
		zap.Bool("is_leader", isLeader),
		zap.String("node_id", rc.nodeID))
	
	// 4. If leader, coordinate workload distribution
	if isLeader {
		if err := rc.distributeWorkload(ctx, workflowID); err != nil {
			return fmt.Errorf("workload distribution failed: %w", err)
		}
	}
	
	// 5. Register workflow execution
	if err := rc.registerWorkflowExecution(ctx, workflowID); err != nil {
		return fmt.Errorf("workflow registration failed: %w", err)
	}
	
	rc.logger.Info("Distributed execution coordination completed with Redis",
		zap.String("workflow_id", workflowID),
		zap.String("node_id", rc.nodeID),
		zap.Bool("is_leader", isLeader))
	
	return nil
}

// registerNode registers this node in Redis cluster registry
func (rc *RealRedisCoordinator) registerNode(ctx context.Context) error {
	nodeKey := rc.nodePrefix + rc.nodeID
	nodeInfo := NodeInfo{
		ID:        rc.nodeID,
		Address:   "localhost:8080", // In real implementation, get actual address
		LastSeen:  time.Now().UTC(),
		Status:    "healthy",
		Workloads: []string{},
	}
	
	nodeData, err := json.Marshal(nodeInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal node info: %w", err)
	}
	
	// Set node info with expiration (heartbeat mechanism)
	if err := rc.client.Set(ctx, nodeKey, nodeData, 60*time.Second).Err(); err != nil {
		return fmt.Errorf("failed to register node in Redis: %w", err)
	}
	
	rc.logger.Debug("Node registered in Redis cluster",
		zap.String("node_id", rc.nodeID),
		zap.String("node_key", nodeKey))
	
	return nil
}

// acquireLock acquires a distributed lock in Redis
func (rc *RealRedisCoordinator) acquireLock(ctx context.Context, lockKey string, expiration time.Duration) (bool, error) {
	lockValue := rc.nodeID + ":" + time.Now().Format(time.RFC3339)
	
	// Use SET NX EX for atomic lock acquisition
	result := rc.client.SetNX(ctx, lockKey, lockValue, expiration)
	if result.Err() != nil {
		return false, fmt.Errorf("Redis lock acquisition error: %w", result.Err())
	}
	
	acquired := result.Val()
	
	rc.logger.Debug("Lock acquisition attempt",
		zap.String("lock_key", lockKey),
		zap.Bool("acquired", acquired),
		zap.String("node_id", rc.nodeID))
	
	return acquired, nil
}

// releaseLock releases a distributed lock in Redis
func (rc *RealRedisCoordinator) releaseLock(ctx context.Context, lockKey string) error {
	// Use Lua script to ensure we only delete our own lock
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	
	lockValue := rc.nodeID + ":"
	result := rc.client.Eval(ctx, luaScript, []string{lockKey}, lockValue)
	if result.Err() != nil {
		return fmt.Errorf("Redis lock release error: %w", result.Err())
	}
	
	rc.logger.Debug("Lock released",
		zap.String("lock_key", lockKey),
		zap.String("node_id", rc.nodeID))
	
	return nil
}

// electLeader elects a leader for workflow execution using Redis
func (rc *RealRedisCoordinator) electLeader(ctx context.Context, leaderKey, workflowID string) (bool, error) {
	// Try to become leader by setting key with expiration
	leaderValue := rc.nodeID + ":" + time.Now().Format(time.RFC3339)
	
	result := rc.client.SetNX(ctx, leaderKey, leaderValue, 5*time.Minute)
	if result.Err() != nil {
		return false, fmt.Errorf("leader election error: %w", result.Err())
	}
	
	isLeader := result.Val()
	
	if isLeader {
		rc.logger.Info("Elected as leader for workflow",
			zap.String("workflow_id", workflowID),
			zap.String("node_id", rc.nodeID))
	} else {
		// Get current leader
		currentLeader, err := rc.client.Get(ctx, leaderKey).Result()
		if err != nil && err != redis.Nil {
			return false, fmt.Errorf("failed to get current leader: %w", err)
		}
		
		rc.logger.Info("Another node is leader for workflow",
			zap.String("workflow_id", workflowID),
			zap.String("current_leader", currentLeader),
			zap.String("node_id", rc.nodeID))
	}
	
	return isLeader, nil
}

// distributeWorkload distributes workload across available nodes using Redis
func (rc *RealRedisCoordinator) distributeWorkload(ctx context.Context, workflowID string) error {
	rc.logger.Info("Distributing workload as leader using Redis",
		zap.String("workflow_id", workflowID),
		zap.String("node_id", rc.nodeID))
	
	// Get all active nodes from Redis
	nodeKeys, err := rc.client.Keys(ctx, rc.nodePrefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get active nodes: %w", err)
	}
	
	var availableNodes []string
	for _, nodeKey := range nodeKeys {
		nodeData, err := rc.client.Get(ctx, nodeKey).Result()
		if err != nil {
			continue
		}
		
		var nodeInfo NodeInfo
		if err := json.Unmarshal([]byte(nodeData), &nodeInfo); err != nil {
			continue
		}
		
		if nodeInfo.Status == "healthy" {
			availableNodes = append(availableNodes, nodeInfo.ID)
		}
	}
	
	// Store workload distribution plan in Redis
	distributionKey := rc.workflowPrefix + workflowID + ":distribution"
	distributionPlan := map[string]interface{}{
		"workflow_id":      workflowID,
		"leader_node":      rc.nodeID,
		"available_nodes":  availableNodes,
		"strategy":         "round_robin",
		"created_at":       time.Now().UTC(),
	}
	
	planData, err := json.Marshal(distributionPlan)
	if err != nil {
		return fmt.Errorf("failed to marshal distribution plan: %w", err)
	}
	
	if err := rc.client.Set(ctx, distributionKey, planData, 1*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store distribution plan: %w", err)
	}
	
	rc.logger.Info("Workload distribution completed using Redis",
		zap.String("workflow_id", workflowID),
		zap.Int("available_nodes", len(availableNodes)),
		zap.String("strategy", "round_robin"))
	
	return nil
}

// registerWorkflowExecution registers workflow execution in Redis
func (rc *RealRedisCoordinator) registerWorkflowExecution(ctx context.Context, workflowID string) error {
	executionKey := rc.workflowPrefix + workflowID + ":executions:" + rc.nodeID
	
	execution := map[string]interface{}{
		"workflow_id": workflowID,
		"node_id":     rc.nodeID,
		"status":      "running",
		"started_at":  time.Now().UTC(),
	}
	
	executionData, err := json.Marshal(execution)
	if err != nil {
		return fmt.Errorf("failed to marshal execution info: %w", err)
	}
	
	if err := rc.client.Set(ctx, executionKey, executionData, 1*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to register workflow execution: %w", err)
	}
	
	rc.logger.Debug("Workflow execution registered in Redis",
		zap.String("workflow_id", workflowID),
		zap.String("node_id", rc.nodeID))
	
	return nil
}

// Start starts the Redis coordinator
func (rc *RealRedisCoordinator) Start(ctx context.Context) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	if rc.running {
		return nil
	}
	
	// Test Redis connection
	if err := rc.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	rc.logger.Info("Starting Redis coordinator",
		zap.String("node_id", rc.nodeID))
	
	// Register initial node info
	if err := rc.registerNode(ctx); err != nil {
		return fmt.Errorf("failed to register initial node: %w", err)
	}
	
	// Start heartbeat ticker
	rc.healthTick = time.NewTicker(30 * time.Second)
	rc.running = true
	
	// Start background heartbeat
	go rc.heartbeatMonitor(ctx)
	
	rc.logger.Info("Redis coordinator started successfully",
		zap.String("node_id", rc.nodeID))
	
	return nil
}

// heartbeatMonitor maintains node heartbeat in Redis
func (rc *RealRedisCoordinator) heartbeatMonitor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-rc.healthTick.C:
			if err := rc.registerNode(ctx); err != nil {
				rc.logger.Error("Heartbeat registration failed", zap.Error(err))
			}
		}
	}
}

// Stop stops the Redis coordinator
func (rc *RealRedisCoordinator) Stop() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	if !rc.running {
		return nil
	}
	
	rc.logger.Info("Stopping Redis coordinator", zap.String("node_id", rc.nodeID))
	
	// Stop heartbeat
	if rc.healthTick != nil {
		rc.healthTick.Stop()
	}
	
	// Remove node from Redis
	nodeKey := rc.nodePrefix + rc.nodeID
	if err := rc.client.Del(context.Background(), nodeKey).Err(); err != nil {
		rc.logger.Warn("Failed to remove node from Redis", zap.Error(err))
	}
	
	// Close Redis connection
	if err := rc.client.Close(); err != nil {
		rc.logger.Warn("Failed to close Redis connection", zap.Error(err))
	}
	
	rc.running = false
	
	rc.logger.Info("Redis coordinator stopped", zap.String("node_id", rc.nodeID))
	return nil
}

// GetClusterStatus returns the current cluster status from Redis
func (rc *RealRedisCoordinator) GetClusterStatus() (map[string]NodeInfo, error) {
	ctx := context.Background()
	nodeKeys, err := rc.client.Keys(ctx, rc.nodePrefix+"*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}
	
	cluster := make(map[string]NodeInfo)
	for _, nodeKey := range nodeKeys {
		nodeData, err := rc.client.Get(ctx, nodeKey).Result()
		if err != nil {
			continue
		}
		
		var nodeInfo NodeInfo
		if err := json.Unmarshal([]byte(nodeData), &nodeInfo); err != nil {
			continue
		}
		
		cluster[nodeInfo.ID] = nodeInfo
	}
	
	return cluster, nil
}