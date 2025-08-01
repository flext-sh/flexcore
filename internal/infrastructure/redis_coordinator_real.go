package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/flext/flexcore/pkg/logging"
	"github.com/flext/flexcore/pkg/result"
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
	nodePrefix     string
	lockPrefix     string
	leaderPrefix   string
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
// SOLID SRP: Reduced from 7 returns to 1 return using Result pattern and CoordinationOrchestrator
func (rc *RealRedisCoordinator) CoordinateExecution(ctx context.Context, workflowID string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Use orchestrator for centralized error handling
	orchestrator := rc.createCoordinationOrchestrator(ctx, workflowID)
	coordinationResult := orchestrator.CoordinateDistributedExecution()
	
	if coordinationResult.IsFailure() {
		return coordinationResult.Error()
	}

	return nil
}

// CoordinationOrchestrator handles complete coordination execution with Result pattern
// SOLID SRP: Single responsibility for coordination orchestration
type CoordinationOrchestrator struct {
	coordinator *RealRedisCoordinator
	ctx         context.Context
	workflowID  string
}

// createCoordinationOrchestrator creates a specialized coordination orchestrator
// SOLID SRP: Factory method for creating specialized orchestrators
func (rc *RealRedisCoordinator) createCoordinationOrchestrator(ctx context.Context, workflowID string) *CoordinationOrchestrator {
	return &CoordinationOrchestrator{
		coordinator: rc,
		ctx:         ctx,
		workflowID:  workflowID,
	}
}

// CoordinateDistributedExecution coordinates distributed execution with centralized error handling
// SOLID SRP: Single responsibility for complete distributed coordination
func (orchestrator *CoordinationOrchestrator) CoordinateDistributedExecution() result.Result[bool] {
	orchestrator.coordinator.logger.Info("Starting distributed coordination using Redis",
		zap.String("workflow_id", orchestrator.workflowID),
		zap.String("node_id", orchestrator.coordinator.nodeID))

	// Phase 1: Register node in cluster
	registrationResult := orchestrator.registerNodeInCluster()
	if registrationResult.IsFailure() {
		return result.Failure[bool](registrationResult.Error())
	}

	// Phase 2: Acquire distributed lock
	lockResult := orchestrator.acquireDistributedLock()
	if lockResult.IsFailure() {
		return result.Failure[bool](lockResult.Error())
	}

	lockKey := lockResult.Value()
	defer func() {
		if err := orchestrator.coordinator.releaseLock(orchestrator.ctx, lockKey); err != nil {
			orchestrator.coordinator.logger.Error("Failed to release lock", zap.Error(err))
		}
	}()

	// Phase 3: Elect leader for workflow
	leaderResult := orchestrator.electWorkflowLeader()
	if leaderResult.IsFailure() {
		return result.Failure[bool](leaderResult.Error())
	}

	isLeader := leaderResult.Value()

	// Phase 4: Execute coordination based on leadership
	executionResult := orchestrator.executeCoordinationPhase(isLeader)
	if executionResult.IsFailure() {
		return result.Failure[bool](executionResult.Error())
	}

	orchestrator.coordinator.logger.Info("Distributed coordination completed successfully",
		zap.String("workflow_id", orchestrator.workflowID),
		zap.Bool("is_leader", isLeader))

	return result.Success(true)
}

// registerNodeInCluster registers node in Redis cluster registry
// SOLID SRP: Single responsibility for node registration
func (orchestrator *CoordinationOrchestrator) registerNodeInCluster() result.Result[bool] {
	if err := orchestrator.coordinator.registerNode(orchestrator.ctx); err != nil {
		return result.Failure[bool](fmt.Errorf("failed to register node in cluster: %w", err))
	}
	return result.Success(true)
}

// acquireDistributedLock acquires distributed lock for workflow
// SOLID SRP: Single responsibility for lock acquisition
func (orchestrator *CoordinationOrchestrator) acquireDistributedLock() result.Result[string] {
	lockKey := orchestrator.coordinator.lockPrefix + orchestrator.workflowID
	acquired, err := orchestrator.coordinator.acquireLock(orchestrator.ctx, lockKey, 2*time.Minute)
	if err != nil {
		return result.Failure[string](fmt.Errorf("failed to acquire coordination lock: %w", err))
	}
	if !acquired {
		return result.Failure[string](fmt.Errorf("coordination lock is held by another node"))
	}
	return result.Success(lockKey)
}

// electWorkflowLeader elects leader for workflow execution
// SOLID SRP: Single responsibility for leader election
func (orchestrator *CoordinationOrchestrator) electWorkflowLeader() result.Result[bool] {
	leaderKey := orchestrator.coordinator.leaderPrefix + orchestrator.workflowID
	isLeader, err := orchestrator.coordinator.electLeader(orchestrator.ctx, leaderKey, orchestrator.workflowID)
	if err != nil {
		return result.Failure[bool](fmt.Errorf("leader election failed: %w", err))
	}
	return result.Success(isLeader)
}

// executeCoordinationPhase executes coordination based on leadership role
// SOLID SRP: Single responsibility for coordination execution
func (orchestrator *CoordinationOrchestrator) executeCoordinationPhase(isLeader bool) result.Result[bool] {
	if isLeader {
		// Leader distributes workload
		distributionResult := orchestrator.executeLeaderWorkflow()
		if distributionResult.IsFailure() {
			return result.Failure[bool](distributionResult.Error())
		}
	}

	// All nodes register their execution
	registrationResult := orchestrator.registerWorkflowExecution()
	if registrationResult.IsFailure() {
		return result.Failure[bool](registrationResult.Error())
	}

	return result.Success(true)
}

// executeLeaderWorkflow executes leader-specific workflow coordination
// SOLID SRP: Single responsibility for leader workflow execution
func (orchestrator *CoordinationOrchestrator) executeLeaderWorkflow() result.Result[bool] {
	if err := orchestrator.coordinator.distributeWorkload(orchestrator.ctx, orchestrator.workflowID); err != nil {
		return result.Failure[bool](fmt.Errorf("workload distribution failed: %w", err))
	}
	return result.Success(true)
}

// registerWorkflowExecution registers workflow execution in Redis
// SOLID SRP: Single responsibility for execution registration
func (orchestrator *CoordinationOrchestrator) registerWorkflowExecution() result.Result[bool] {
	if err := orchestrator.coordinator.registerWorkflowExecution(orchestrator.ctx, orchestrator.workflowID); err != nil {
		return result.Failure[bool](fmt.Errorf("workflow execution registration failed: %w", err))
	}
	return result.Success(true)
}

// registerNode registers this node in Redis cluster registry
// RedisOperationTemplate provides template method for Redis operations
// SOLID Template Method Pattern: Eliminates repetitive error handling and logging patterns
type RedisOperationTemplate struct {
	coordinator *RealRedisCoordinator
	ctx         context.Context
	operation   string
}

// NewRedisOperationTemplate creates a new Redis operation template
func NewRedisOperationTemplate(coordinator *RealRedisCoordinator, ctx context.Context, operation string) *RedisOperationTemplate {
	return &RedisOperationTemplate{
		coordinator: coordinator,
		ctx:         ctx,
		operation:   operation,
	}
}

// ExecuteRedisOperation executes Redis operation using template method pattern
// SOLID Template Method: Common algorithm with customizable operation execution
func (rot *RedisOperationTemplate) ExecuteRedisOperation(
	executeOp func() error,
	logFields []zap.Field,
) error {
	// Step 1: Log operation start
	rot.coordinator.logger.Debug(fmt.Sprintf("Starting Redis %s operation", rot.operation), logFields...)

	// Step 2: Execute operation
	if err := executeOp(); err != nil {
		// Step 3a: Handle error
		errorFields := append(logFields, zap.Error(err))
		rot.coordinator.logger.Error(fmt.Sprintf("Redis %s operation failed", rot.operation), errorFields...)
		return err
	}

	// Step 3b: Log success
	rot.coordinator.logger.Debug(fmt.Sprintf("Redis %s operation completed", rot.operation), logFields...)
	return nil
}

func (rc *RealRedisCoordinator) registerNode(ctx context.Context) error {
	template := NewRedisOperationTemplate(rc, ctx, "node registration")
	
	return template.ExecuteRedisOperation(
		func() error {
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

			return nil
		},
		[]zap.Field{
			zap.String("node_id", rc.nodeID),
			zap.String("node_key", rc.nodePrefix+rc.nodeID),
		},
	)
}

// LockOperationResult encapsulates lock operation results
// SOLID Result Pattern: Consolidates lock operation outcomes reducing conditional complexity
type LockOperationResult struct {
	Success bool
	Error   error
}

// NewLockOperationResult creates a lock operation result
func NewLockOperationResult(success bool, err error) *LockOperationResult {
	return &LockOperationResult{Success: success, Error: err}
}

// acquireLock acquires a distributed lock using template method pattern
func (rc *RealRedisCoordinator) acquireLock(ctx context.Context, lockKey string, expiration time.Duration) (bool, error) {
	template := NewRedisOperationTemplate(rc, ctx, "lock acquisition")
	
	var acquired bool
	err := template.ExecuteRedisOperation(
		func() error {
			lockValue := rc.nodeID + ":" + time.Now().Format(time.RFC3339)

			// Use SET NX EX for atomic lock acquisition
			result := rc.client.SetNX(ctx, lockKey, lockValue, expiration)
			if result.Err() != nil {
				return fmt.Errorf("Redis lock acquisition error: %w", result.Err())
			}

			acquired = result.Val()
			return nil
		},
		[]zap.Field{
			zap.String("lock_key", lockKey),
			zap.String("node_id", rc.nodeID),
			zap.Duration("expiration", expiration),
		},
	)
	
	return acquired, err
}

// releaseLock releases a distributed lock using template method pattern
func (rc *RealRedisCoordinator) releaseLock(ctx context.Context, lockKey string) error {
	template := NewRedisOperationTemplate(rc, ctx, "lock release")
	
	return template.ExecuteRedisOperation(
		func() error {
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

			return nil
		},
		[]zap.Field{
			zap.String("lock_key", lockKey),
			zap.String("node_id", rc.nodeID),
		},
	)
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
// SOLID SRP: Reduced complexity using specialized WorkloadDistributor
func (rc *RealRedisCoordinator) distributeWorkload(ctx context.Context, workflowID string) error {
	distributor := rc.createWorkloadDistributor(ctx, workflowID)
	distributionResult := distributor.DistributeWorkloadAcrossNodes()
	
	if distributionResult.IsFailure() {
		return distributionResult.Error()
	}
	
	return nil
}

// WorkloadDistributor handles workload distribution operations
// SOLID SRP: Single responsibility for workload distribution
type WorkloadDistributor struct {
	coordinator *RealRedisCoordinator
	ctx         context.Context
	workflowID  string
}

// createWorkloadDistributor creates specialized workload distributor
// SOLID SRP: Factory method for creating specialized distributors
func (rc *RealRedisCoordinator) createWorkloadDistributor(ctx context.Context, workflowID string) *WorkloadDistributor {
	return &WorkloadDistributor{
		coordinator: rc,
		ctx:         ctx,
		workflowID:  workflowID,
	}
}

// DistributeWorkloadAcrossNodes distributes workload with centralized error handling
// SOLID SRP: Single responsibility for complete workload distribution
func (distributor *WorkloadDistributor) DistributeWorkloadAcrossNodes() result.Result[bool] {
	distributor.coordinator.logger.Info("Distributing workload as leader using Redis",
		zap.String("workflow_id", distributor.workflowID),
		zap.String("node_id", distributor.coordinator.nodeID))

	// Phase 1: Discover available nodes
	nodesResult := distributor.discoverAvailableNodes()
	if nodesResult.IsFailure() {
		return result.Failure[bool](nodesResult.Error())
	}
	
	availableNodes := nodesResult.Value()

	// Phase 2: Create and store distribution plan
	planResult := distributor.createAndStoreDistributionPlan(availableNodes)
	if planResult.IsFailure() {
		return result.Failure[bool](planResult.Error())
	}

	distributor.coordinator.logger.Info("Workload distribution completed using Redis",
		zap.String("workflow_id", distributor.workflowID),
		zap.Int("available_nodes", len(availableNodes)),
		zap.String("strategy", "round_robin"))

	return result.Success(true)
}

// discoverAvailableNodes discovers healthy nodes in the cluster
// SOLID SRP: Single responsibility for node discovery
func (distributor *WorkloadDistributor) discoverAvailableNodes() result.Result[[]string] {
	// Get all node keys from Redis
	nodeKeys, err := distributor.coordinator.client.Keys(distributor.ctx, distributor.coordinator.nodePrefix+"*").Result()
	if err != nil {
		return result.Failure[[]string](fmt.Errorf("failed to get active nodes: %w", err))
	}

	// Filter healthy nodes
	healthyNodes := distributor.filterHealthyNodes(nodeKeys)
	return result.Success(healthyNodes)
}

// filterHealthyNodes filters nodes by health status
// SOLID SRP: Single responsibility for health filtering
func (distributor *WorkloadDistributor) filterHealthyNodes(nodeKeys []string) []string {
	var availableNodes []string
	
	for _, nodeKey := range nodeKeys {
		if nodeInfo := distributor.getNodeInfo(nodeKey); nodeInfo != nil && nodeInfo.Status == "healthy" {
			availableNodes = append(availableNodes, nodeInfo.ID)
		}
	}
	
	return availableNodes
}

// getNodeInfo retrieves node information from Redis
// SOLID SRP: Single responsibility for node info retrieval
func (distributor *WorkloadDistributor) getNodeInfo(nodeKey string) *NodeInfo {
	nodeData, err := distributor.coordinator.client.Get(distributor.ctx, nodeKey).Result()
	if err != nil {
		return nil
	}

	var nodeInfo NodeInfo
	if err := json.Unmarshal([]byte(nodeData), &nodeInfo); err != nil {
		return nil
	}

	return &nodeInfo
}

// createAndStoreDistributionPlan creates and stores the distribution plan
// SOLID SRP: Single responsibility for plan creation and storage
func (distributor *WorkloadDistributor) createAndStoreDistributionPlan(availableNodes []string) result.Result[bool] {
	// Create distribution plan
	distributionPlan := distributor.createDistributionPlan(availableNodes)
	
	// Marshal plan to JSON
	planData, err := json.Marshal(distributionPlan)
	if err != nil {
		return result.Failure[bool](fmt.Errorf("failed to marshal distribution plan: %w", err))
	}

	// Store plan in Redis
	distributionKey := distributor.coordinator.workflowPrefix + distributor.workflowID + ":distribution"
	if err := distributor.coordinator.client.Set(distributor.ctx, distributionKey, planData, 1*time.Hour).Err(); err != nil {
		return result.Failure[bool](fmt.Errorf("failed to store distribution plan: %w", err))
	}

	return result.Success(true)
}

// createDistributionPlan creates the distribution plan structure
// SOLID SRP: Single responsibility for plan structure creation
func (distributor *WorkloadDistributor) createDistributionPlan(availableNodes []string) map[string]interface{} {
	return map[string]interface{}{
		"workflow_id":     distributor.workflowID,
		"leader_node":     distributor.coordinator.nodeID,
		"available_nodes": availableNodes,
		"strategy":        "round_robin",
		"created_at":      time.Now().UTC(),
	}
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
// SOLID SRP: Reduced complexity using specialized ClusterStatusCollector
func (rc *RealRedisCoordinator) GetClusterStatus() (map[string]NodeInfo, error) {
	collector := rc.createClusterStatusCollector()
	statusResult := collector.CollectClusterStatus()
	
	if statusResult.IsFailure() {
		return nil, statusResult.Error()
	}
	
	return statusResult.Value(), nil
}

// ClusterStatusCollector handles cluster status collection operations
// SOLID SRP: Single responsibility for cluster status collection
type ClusterStatusCollector struct {
	coordinator *RealRedisCoordinator
}

// createClusterStatusCollector creates specialized cluster status collector
// SOLID SRP: Factory method for creating specialized collectors
func (rc *RealRedisCoordinator) createClusterStatusCollector() *ClusterStatusCollector {
	return &ClusterStatusCollector{
		coordinator: rc,
	}
}

// CollectClusterStatus collects cluster status with centralized error handling
// SOLID SRP: Single responsibility for complete cluster status collection
func (collector *ClusterStatusCollector) CollectClusterStatus() result.Result[map[string]NodeInfo] {
	ctx := context.Background()
	
	// Phase 1: Get all node keys
	nodeKeysResult := collector.getAllNodeKeys(ctx)
	if nodeKeysResult.IsFailure() {
		return result.Failure[map[string]NodeInfo](nodeKeysResult.Error())
	}
	
	nodeKeys := nodeKeysResult.Value()
	
	// Phase 2: Collect all node information
	cluster := collector.collectAllNodeInformation(ctx, nodeKeys)
	
	return result.Success(cluster)
}

// getAllNodeKeys retrieves all node keys from Redis
// SOLID SRP: Single responsibility for node key retrieval
func (collector *ClusterStatusCollector) getAllNodeKeys(ctx context.Context) result.Result[[]string] {
	nodeKeys, err := collector.coordinator.client.Keys(ctx, collector.coordinator.nodePrefix+"*").Result()
	if err != nil {
		return result.Failure[[]string](fmt.Errorf("failed to get cluster nodes: %w", err))
	}
	
	return result.Success(nodeKeys)
}

// collectAllNodeInformation collects information for all nodes
// SOLID SRP: Single responsibility for node information collection
func (collector *ClusterStatusCollector) collectAllNodeInformation(ctx context.Context, nodeKeys []string) map[string]NodeInfo {
	cluster := make(map[string]NodeInfo)
	
	for _, nodeKey := range nodeKeys {
		if nodeInfo := collector.getNodeInformation(ctx, nodeKey); nodeInfo != nil {
			cluster[nodeInfo.ID] = *nodeInfo
		}
	}
	
	return cluster
}

// getNodeInformation retrieves and parses node information
// SOLID SRP: Single responsibility for individual node information retrieval
func (collector *ClusterStatusCollector) getNodeInformation(ctx context.Context, nodeKey string) *NodeInfo {
	nodeData, err := collector.coordinator.client.Get(ctx, nodeKey).Result()
	if err != nil {
		return nil
	}

	var nodeInfo NodeInfo
	if err := json.Unmarshal([]byte(nodeData), &nodeInfo); err != nil {
		return nil
	}

	return &nodeInfo
}
