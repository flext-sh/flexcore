// Package scheduler provides timer-based singleton services with cluster coordination
package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
)

// TimerSingleton represents a service that runs on a timer with cluster coordination
type TimerSingleton interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
	GetInterval() time.Duration
	GetLastExecution() time.Time
	Execute(ctx context.Context) error
}

// ClusterCoordinator handles coordination between cluster nodes
type ClusterCoordinator interface {
	AcquireLock(ctx context.Context, lockKey string, ttl time.Duration) result.Result[Lock]
	IsLeader(ctx context.Context) bool
	GetNodeID() string
	RegisterNode(ctx context.Context, nodeInfo NodeInfo) error
	GetActiveNodes(ctx context.Context) []NodeInfo
	BroadcastMessage(ctx context.Context, message ClusterMessage) error
	SubscribeToMessages(ctx context.Context, handler ClusterMessageHandler) error
	Stop() error
}

// Lock represents a distributed lock
type Lock interface {
	Release(ctx context.Context) error
	Renew(ctx context.Context, ttl time.Duration) error
	IsValid() bool
}

// NodeInfo contains information about a cluster node
type NodeInfo struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	LastSeen time.Time `json:"last_seen"`
	Metadata map[string]string `json:"metadata"`
}

// ClusterMessage and ClusterMessageHandler are defined in cluster_coordinator.go

// TimerSingletonConfig configures a timer singleton
type TimerSingletonConfig struct {
	Name             string
	Interval         time.Duration
	ClusterWide      bool
	LeaderOnly       bool
	MaxRetries       int
	RetryDelay       time.Duration
	HealthCheckInterval time.Duration
	LockTTL          time.Duration
}

// BaseTimerSingleton provides base implementation for timer singletons
type BaseTimerSingleton struct {
	config      TimerSingletonConfig
	coordinator ClusterCoordinator
	executor    func(ctx context.Context) error
	
	mu           sync.RWMutex
	running      bool
	ticker       *time.Ticker
	stopChan     chan struct{}
	lastExecution time.Time
	executionCount int64
	errorCount    int64
}

// NewTimerSingleton creates a new timer singleton
func NewTimerSingleton(config TimerSingletonConfig, coordinator ClusterCoordinator, executor func(ctx context.Context) error) *BaseTimerSingleton {
	if config.Interval <= 0 {
		config.Interval = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 5 * time.Second
	}
	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 10 * time.Second
	}
	if config.LockTTL <= 0 {
		config.LockTTL = config.Interval + 30*time.Second
	}

	return &BaseTimerSingleton{
		config:      config,
		coordinator: coordinator,
		executor:    executor,
		stopChan:    make(chan struct{}),
	}
}

// Start starts the timer singleton
func (ts *BaseTimerSingleton) Start(ctx context.Context) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.running {
		return errors.ValidationError("timer singleton is already running")
	}

	ts.running = true
	ts.ticker = time.NewTicker(ts.config.Interval)

	// Start the main execution loop
	go ts.executionLoop(ctx)

	return nil
}

// Stop stops the timer singleton
func (ts *BaseTimerSingleton) Stop() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.running {
		return nil
	}

	ts.running = false
	if ts.ticker != nil {
		ts.ticker.Stop()
	}
	close(ts.stopChan)

	return nil
}

// IsRunning returns whether the singleton is running
func (ts *BaseTimerSingleton) IsRunning() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.running
}

// GetInterval returns the execution interval
func (ts *BaseTimerSingleton) GetInterval() time.Duration {
	return ts.config.Interval
}

// GetLastExecution returns the last execution time
func (ts *BaseTimerSingleton) GetLastExecution() time.Time {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.lastExecution
}

// Execute executes the singleton's function
func (ts *BaseTimerSingleton) Execute(ctx context.Context) error {
	if ts.executor == nil {
		return errors.ValidationError("no executor function defined")
	}

	// Check if this node should execute (cluster coordination)
	if !ts.shouldExecute(ctx) {
		return nil
	}

	ts.mu.Lock()
	ts.lastExecution = time.Now()
	ts.executionCount++
	ts.mu.Unlock()

	// Execute with retries
	var lastErr error
	for attempt := 0; attempt <= ts.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(ts.config.RetryDelay)
		}

		if err := ts.executor(ctx); err != nil {
			lastErr = err
			ts.mu.Lock()
			ts.errorCount++
			ts.mu.Unlock()
			continue
		}

		return nil
	}

	return errors.Wrap(lastErr, "timer singleton execution failed after retries")
}

// executionLoop runs the main execution loop
func (ts *BaseTimerSingleton) executionLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ts.stopChan:
			return
		case <-ts.ticker.C:
			if err := ts.Execute(ctx); err != nil {
				// Log error but continue execution
				// In production, this would use a proper logger
			}
		}
	}
}

// shouldExecute determines if this node should execute the singleton
func (ts *BaseTimerSingleton) shouldExecute(ctx context.Context) bool {
	if ts.coordinator == nil {
		return true // No coordination, always execute
	}

	// If leader-only mode, check if this node is the leader
	if ts.config.LeaderOnly {
		return ts.coordinator.IsLeader(ctx)
	}

	// If cluster-wide mode, use distributed locking
	if ts.config.ClusterWide {
		lockKey := "timer_singleton:" + ts.config.Name
		lockResult := ts.coordinator.AcquireLock(ctx, lockKey, ts.config.LockTTL)
		if lockResult.IsFailure() {
			return false
		}

		lock := lockResult.Value()
		defer func() {
			if err := lock.Release(ctx); err != nil {
				// Log error but don't fail
			}
		}()

		return lock.IsValid()
	}

	return true
}

// GetStats returns execution statistics
func (ts *BaseTimerSingleton) GetStats() TimerSingletonStats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return TimerSingletonStats{
		Name:           ts.config.Name,
		Running:        ts.running,
		ExecutionCount: ts.executionCount,
		ErrorCount:     ts.errorCount,
		LastExecution:  ts.lastExecution,
		Interval:       ts.config.Interval,
	}
}

// TimerSingletonStats contains execution statistics
type TimerSingletonStats struct {
	Name           string        `json:"name"`
	Running        bool          `json:"running"`
	ExecutionCount int64         `json:"execution_count"`
	ErrorCount     int64         `json:"error_count"`
	LastExecution  time.Time     `json:"last_execution"`
	Interval       time.Duration `json:"interval"`
}

// TimerSingletonManager manages multiple timer singletons
type TimerSingletonManager struct {
	mu          sync.RWMutex
	singletons  map[string]TimerSingleton
	coordinator ClusterCoordinator
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTimerSingletonManager creates a new timer singleton manager
func NewTimerSingletonManager(coordinator ClusterCoordinator) *TimerSingletonManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &TimerSingletonManager{
		singletons:  make(map[string]TimerSingleton),
		coordinator: coordinator,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Register registers a timer singleton
func (mgr *TimerSingletonManager) Register(name string, singleton TimerSingleton) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if _, exists := mgr.singletons[name]; exists {
		return errors.AlreadyExistsError("timer singleton " + name)
	}

	mgr.singletons[name] = singleton
	return nil
}

// Start starts all registered singletons
func (mgr *TimerSingletonManager) Start(ctx context.Context) error {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	for name, singleton := range mgr.singletons {
		if err := singleton.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start timer singleton "+name)
		}
	}

	return nil
}

// Stop stops all singletons
func (mgr *TimerSingletonManager) Stop() error {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	mgr.cancel()

	var lastErr error
	for _, singleton := range mgr.singletons {
		if err := singleton.Stop(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetSingleton returns a singleton by name
func (mgr *TimerSingletonManager) GetSingleton(name string) (TimerSingleton, bool) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	singleton, exists := mgr.singletons[name]
	return singleton, exists
}

// ListSingletons returns all registered singletons
func (mgr *TimerSingletonManager) ListSingletons() map[string]TimerSingleton {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	result := make(map[string]TimerSingleton)
	for name, singleton := range mgr.singletons {
		result[name] = singleton
	}
	return result
}

// GetAllStats returns statistics for all singletons
func (mgr *TimerSingletonManager) GetAllStats() []TimerSingletonStats {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	stats := make([]TimerSingletonStats, 0, len(mgr.singletons))
	for _, singleton := range mgr.singletons {
		if baseSingleton, ok := singleton.(*BaseTimerSingleton); ok {
			stats = append(stats, baseSingleton.GetStats())
		}
	}
	return stats
}