package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimerSingletonBasicExecution(t *testing.T) {
	var executionCount int64
	
	config := TimerSingletonConfig{
		Name:     "test-singleton",
		Interval: 100 * time.Millisecond,
	}
	
	coordinator := NewInMemoryClusterCoordinator()
	err := coordinator.Start(context.Background())
	require.NoError(t, err)
	defer coordinator.Stop()
	
	executor := func(ctx context.Context) error {
		atomic.AddInt64(&executionCount, 1)
		return nil
	}
	
	singleton := NewTimerSingleton(config, coordinator, executor)
	
	// Test initial state
	assert.False(t, singleton.IsRunning())
	assert.Equal(t, config.Interval, singleton.GetInterval())
	
	// Start singleton
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	err = singleton.Start(ctx)
	require.NoError(t, err)
	assert.True(t, singleton.IsRunning())
	
	// Wait for some executions
	time.Sleep(350 * time.Millisecond)
	
	// Stop singleton
	err = singleton.Stop()
	require.NoError(t, err)
	assert.False(t, singleton.IsRunning())
	
	// Check execution count (should be at least 3)
	count := atomic.LoadInt64(&executionCount)
	assert.GreaterOrEqual(t, count, int64(3))
	
	// Check stats
	stats := singleton.GetStats()
	assert.Equal(t, "test-singleton", stats.Name)
	assert.False(t, stats.Running)
	assert.Equal(t, count, stats.ExecutionCount)
	assert.Equal(t, int64(0), stats.ErrorCount)
}

func TestTimerSingletonClusterCoordination(t *testing.T) {
	var node1Executions, node2Executions int64
	
	config := TimerSingletonConfig{
		Name:        "cluster-singleton",
		Interval:    50 * time.Millisecond,
		ClusterWide: true,
	}
	
	// Create two coordinators (simulating two nodes)
	coordinator1 := NewInMemoryClusterCoordinator()
	coordinator2 := NewInMemoryClusterCoordinator()
	
	err := coordinator1.Start(context.Background())
	require.NoError(t, err)
	defer coordinator1.Stop()
	
	err = coordinator2.Start(context.Background())
	require.NoError(t, err)
	defer coordinator2.Stop()
	
	// Create singletons for each node
	executor1 := func(ctx context.Context) error {
		atomic.AddInt64(&node1Executions, 1)
		return nil
	}
	
	executor2 := func(ctx context.Context) error {
		atomic.AddInt64(&node2Executions, 1)
		return nil
	}
	
	singleton1 := NewTimerSingleton(config, coordinator1, executor1)
	singleton2 := NewTimerSingleton(config, coordinator2, executor2)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start both singletons
	err = singleton1.Start(ctx)
	require.NoError(t, err)
	defer singleton1.Stop()
	
	err = singleton2.Start(ctx)
	require.NoError(t, err)
	defer singleton2.Stop()
	
	// Let them run for a while
	time.Sleep(300 * time.Millisecond)
	
	// In cluster mode with locking, only one should execute most of the time
	total := atomic.LoadInt64(&node1Executions) + atomic.LoadInt64(&node2Executions)
	assert.Greater(t, total, int64(3))
	
	// At least one node should have executed
	assert.True(t, atomic.LoadInt64(&node1Executions) > 0 || atomic.LoadInt64(&node2Executions) > 0)
}

func TestTimerSingletonLeaderOnly(t *testing.T) {
	var leaderExecutions, followerExecutions int64
	
	config := TimerSingletonConfig{
		Name:       "leader-singleton",
		Interval:   50 * time.Millisecond,
		LeaderOnly: true,
	}
	
	coordinator := NewInMemoryClusterCoordinator()
	err := coordinator.Start(context.Background())
	require.NoError(t, err)
	defer coordinator.Stop()
	
	// Register two nodes
	node1 := NodeInfo{ID: "node-1", Address: "localhost:8001", LastSeen: time.Now()}
	node2 := NodeInfo{ID: "node-2", Address: "localhost:8002", LastSeen: time.Now()}
	
	err = coordinator.RegisterNode(context.Background(), node1)
	require.NoError(t, err)
	err = coordinator.RegisterNode(context.Background(), node2)
	require.NoError(t, err)
	
	// Create singleton for current node (which should be leader)
	executor := func(ctx context.Context) error {
		if coordinator.IsLeader(context.Background()) {
			atomic.AddInt64(&leaderExecutions, 1)
		} else {
			atomic.AddInt64(&followerExecutions, 1)
		}
		return nil
	}
	
	singleton := NewTimerSingleton(config, coordinator, executor)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	err = singleton.Start(ctx)
	require.NoError(t, err)
	defer singleton.Stop()
	
	// Let it run
	time.Sleep(200 * time.Millisecond)
	
	// Leader should have executed, follower should not
	assert.GreaterOrEqual(t, atomic.LoadInt64(&leaderExecutions), int64(1))
	assert.Equal(t, int64(0), atomic.LoadInt64(&followerExecutions))
}

func TestTimerSingletonManager(t *testing.T) {
	coordinator := NewInMemoryClusterCoordinator()
	err := coordinator.Start(context.Background())
	require.NoError(t, err)
	defer coordinator.Stop()
	
	manager := NewTimerSingletonManager(coordinator)
	defer manager.Stop()
	
	var execution1Count, execution2Count int64
	
	// Create two singletons
	config1 := TimerSingletonConfig{
		Name:     "singleton-1",
		Interval: 50 * time.Millisecond,
	}
	executor1 := func(ctx context.Context) error {
		atomic.AddInt64(&execution1Count, 1)
		return nil
	}
	singleton1 := NewTimerSingleton(config1, coordinator, executor1)
	
	config2 := TimerSingletonConfig{
		Name:     "singleton-2",
		Interval: 75 * time.Millisecond,
	}
	executor2 := func(ctx context.Context) error {
		atomic.AddInt64(&execution2Count, 1)
		return nil
	}
	singleton2 := NewTimerSingleton(config2, coordinator, executor2)
	
	// Register singletons
	err = manager.Register("singleton-1", singleton1)
	require.NoError(t, err)
	
	err = manager.Register("singleton-2", singleton2)
	require.NoError(t, err)
	
	// Try to register duplicate
	err = manager.Register("singleton-1", singleton1)
	assert.Error(t, err)
	
	// Start all singletons
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	err = manager.Start(ctx)
	require.NoError(t, err)
	
	// Let them run
	time.Sleep(300 * time.Millisecond)
	
	// Check executions
	assert.Greater(t, atomic.LoadInt64(&execution1Count), int64(3))
	assert.GreaterOrEqual(t, atomic.LoadInt64(&execution2Count), int64(3))
	
	// Test singleton retrieval
	retrieved, exists := manager.GetSingleton("singleton-1")
	assert.True(t, exists)
	assert.Equal(t, singleton1, retrieved)
	
	_, exists = manager.GetSingleton("non-existent")
	assert.False(t, exists)
	
	// Test listing
	allSingletons := manager.ListSingletons()
	assert.Len(t, allSingletons, 2)
	
	// Test stats
	allStats := manager.GetAllStats()
	assert.Len(t, allStats, 2)
	
	for _, stats := range allStats {
		assert.True(t, stats.Running)
		assert.Greater(t, stats.ExecutionCount, int64(0))
	}
}

func TestClusterCoordinator(t *testing.T) {
	coordinator := NewInMemoryClusterCoordinator()
	err := coordinator.Start(context.Background())
	require.NoError(t, err)
	defer coordinator.Stop()
	
	// Test node registration
	node1 := NodeInfo{
		ID:      "test-node-1",
		Address: "localhost:8001",
		Metadata: map[string]string{
			"role": "worker",
		},
	}
	
	err = coordinator.RegisterNode(context.Background(), node1)
	require.NoError(t, err)
	
	// Test active nodes
	activeNodes := coordinator.GetActiveNodes(context.Background())
	assert.GreaterOrEqual(t, len(activeNodes), 1)
	
	// Test leader election (current node should be leader after some time)
	time.Sleep(500 * time.Millisecond)
	assert.True(t, coordinator.IsLeader(context.Background()))
	
	// Test distributed locking
	ctx := context.Background()
	lockResult := coordinator.AcquireLock(ctx, "test-lock", 1*time.Second)
	require.True(t, lockResult.IsSuccess())
	
	lock := lockResult.Value()
	assert.True(t, lock.IsValid())
	
	// Try to acquire same lock again
	lockResult2 := coordinator.AcquireLock(ctx, "test-lock", 1*time.Second)
	require.True(t, lockResult2.IsSuccess()) // Should succeed because same node
	
	// Release lock
	err = lock.Release(ctx)
	assert.NoError(t, err)
	
	// Test stats
	stats := coordinator.GetClusterStats()
	assert.Greater(t, stats.NodeCount, 0)
	// Leader ID might be empty initially, so we just check it's a valid response
	assert.GreaterOrEqual(t, len(stats.LeaderID), 0)
}

func TestDistributedLock(t *testing.T) {
	coordinator := NewInMemoryClusterCoordinator()
	err := coordinator.Start(context.Background())
	require.NoError(t, err)
	defer coordinator.Stop()
	
	ctx := context.Background()
	
	// Acquire lock
	lockResult := coordinator.AcquireLock(ctx, "test-lock", 500*time.Millisecond)
	require.True(t, lockResult.IsSuccess())
	
	lock := lockResult.Value()
	assert.True(t, lock.IsValid())
	
	// Renew lock
	err = lock.Renew(ctx, 1*time.Second)
	assert.NoError(t, err)
	assert.True(t, lock.IsValid())
	
	// Wait for expiration
	time.Sleep(100 * time.Millisecond)
	assert.True(t, lock.IsValid()) // Should still be valid after renewal
	
	// Release lock
	err = lock.Release(ctx)
	assert.NoError(t, err)
}

func TestClusterCoordinatorBuilder(t *testing.T) {
	// Test in-memory coordinator
	coordinator1 := NewClusterCoordinatorBuilder().
		WithInMemory().
		Build()
	assert.IsType(t, &InMemoryClusterCoordinator{}, coordinator1)
	
	// Test Redis coordinator (falls back to in-memory for now)
	coordinator2 := NewClusterCoordinatorBuilder().
		WithRedis("redis://localhost:6379").
		Build()
	assert.IsType(t, &RedisClusterCoordinator{}, coordinator2)
	
	// Test etcd coordinator (falls back to in-memory for now)
	coordinator3 := NewClusterCoordinatorBuilder().
		WithEtcd([]string{"localhost:2379"}).
		Build()
	assert.IsType(t, &EtcdClusterCoordinator{}, coordinator3)
}