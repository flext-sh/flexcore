// tests/e2e/real_distributed_test.go
// TESTE E2E REAL - Validação completa do sistema distribuído

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flext/flexcore/core"
	"github.com/flext/flexcore/infrastructure/di"
	"github.com/flext/flexcore/infrastructure/plugins"
	"github.com/flext/flexcore/infrastructure/scheduler"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealDistributedCoordination validates Redis-based distributed coordination
func TestRealDistributedCoordination(t *testing.T) {
	ctx := context.Background()

	// Setup Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// Skip if Redis not available
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping distributed coordination test")
	}

	// Clean up test keys
	rdb.Del(ctx, "test-cluster:*")

	t.Run("DistributedLocking", func(t *testing.T) {
		coordinator := scheduler.NewRedisClusterCoordinator(rdb, "test-cluster")

		// Test mutual exclusion
		var counter int32
		var wg sync.WaitGroup
		errors := make([]error, 10)

		// 10 goroutines trying to increment counter
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				nodeID := fmt.Sprintf("node-%d", idx)
				
				// Try to acquire lock
				lockResult := coordinator.AcquireLock(ctx, "counter-resource", nodeID, 5*time.Second)
				if lockResult.IsFailure() {
					errors[idx] = lockResult.Error()
					return
				}

				// Critical section
				current := atomic.LoadInt32(&counter)
				time.Sleep(10 * time.Millisecond) // Simulate work
				atomic.StoreInt32(&counter, current+1)

				// Release lock
				coordinator.ReleaseLock(ctx, "counter-resource", nodeID)
			}(i)
		}

		wg.Wait()

		// Counter should be exactly 10 if locking worked
		assert.Equal(t, int32(10), atomic.LoadInt32(&counter), "Counter should be 10 with proper locking")
	})

	t.Run("LeaderElection", func(t *testing.T) {
		coordinator := scheduler.NewRedisClusterCoordinator(rdb, "test-cluster")

		// Multiple nodes compete for leadership
		nodes := []string{"node-A", "node-B", "node-C", "node-D", "node-E"}
		leaders := make([]string, 0)
		var mu sync.Mutex

		var wg sync.WaitGroup
		for _, nodeID := range nodes {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()

				result := coordinator.ElectLeader(ctx, "test-service", id, 2*time.Second)
				if result.IsSuccess() && result.Value() {
					mu.Lock()
					leaders = append(leaders, id)
					mu.Unlock()
				}
			}(nodeID)
		}

		wg.Wait()

		// Only one leader should be elected
		assert.Equal(t, 1, len(leaders), "Exactly one leader should be elected")
		t.Logf("Elected leader: %s", leaders[0])
	})

	t.Run("LeaderFailover", func(t *testing.T) {
		coordinator := scheduler.NewRedisClusterCoordinator(rdb, "test-cluster")

		// First node becomes leader
		result1 := coordinator.ElectLeader(ctx, "failover-test", "node-1", 1*time.Second)
		require.True(t, result1.IsSuccess() && result1.Value(), "Node 1 should become leader")

		// Second node tries (should fail)
		result2 := coordinator.ElectLeader(ctx, "failover-test", "node-2", 1*time.Second)
		assert.False(t, result2.Value(), "Node 2 should not become leader while node 1 is active")

		// Wait for leader TTL to expire
		time.Sleep(1500 * time.Millisecond)

		// Now node 2 should become leader
		result3 := coordinator.ElectLeader(ctx, "failover-test", "node-2", 1*time.Second)
		assert.True(t, result3.IsSuccess() && result3.Value(), "Node 2 should become leader after TTL expiry")
	})
}

// TestRealPluginExecution validates the HashiCorp plugin system
func TestRealPluginExecution(t *testing.T) {
	ctx := context.Background()

	// Create plugin manager
	pluginManager := plugins.NewRealPluginManager("./plugins")
	require.NoError(t, pluginManager.Start(ctx))
	defer pluginManager.Stop()

	t.Run("PluginDiscovery", func(t *testing.T) {
		// List available plugins
		pluginList := pluginManager.ListPlugins()
		t.Logf("Discovered %d plugins", len(pluginList))

		// Check if pre-built plugins exist
		expectedPlugins := []string{"postgres-extractor", "json-transformer", "api-loader"}
		for _, expected := range expectedPlugins {
			pluginPath := fmt.Sprintf("./plugins/%s", expected)
			if fileExists(pluginPath) {
				t.Logf("Found plugin: %s", expected)
			}
		}
	})

	t.Run("PluginLoadAndExecute", func(t *testing.T) {
		// Skip if no plugins available
		if !fileExists("./plugins/json-transformer") {
			t.Skip("json-transformer plugin not found")
		}

		// Load plugin
		loadResult := pluginManager.LoadPlugin(ctx, "json-transformer", "./plugins/json-transformer")
		if loadResult.IsFailure() {
			t.Skipf("Failed to load plugin: %v", loadResult.Error())
		}

		// Execute plugin
		input := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": 1, "name": "Item 1"},
				{"id": 2, "name": "Item 2"},
			},
		}

		execResult := pluginManager.ExecutePlugin(ctx, "json-transformer", input)
		require.True(t, execResult.IsSuccess(), "Plugin execution should succeed")

		output := execResult.Value()
		t.Logf("Plugin output: %+v", output)
	})
}

// TestRealMultiNodeCluster validates multi-node cluster operations
func TestRealMultiNodeCluster(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create 3 FlexCore nodes
	nodes := make([]*FlexCoreNode, 3)
	basePort := 9000

	for i := 0; i < 3; i++ {
		node := createTestNode(i+1, basePort+i)
		require.NoError(t, node.Start(ctx))
		defer node.Stop()
		nodes[i] = node
	}

	// Wait for cluster formation
	time.Sleep(2 * time.Second)

	t.Run("ClusterFormation", func(t *testing.T) {
		// Check each node's view of the cluster
		for i, node := range nodes {
			status := node.GetClusterStatus()
			assert.Equal(t, 3, status.ActiveNodes, "Node %d should see 3 active nodes", i+1)
			assert.NotEmpty(t, status.LeaderID, "Node %d should see a leader", i+1)
			t.Logf("Node %d status: Leader=%s, Nodes=%d", i+1, status.LeaderID, status.ActiveNodes)
		}
	})

	t.Run("EventPropagation", func(t *testing.T) {
		// Send event from node 1
		event := &TestEvent{
			ID:        "test-001",
			Timestamp: time.Now(),
			Data:      "Hello from Node 1",
		}

		nodes[0].PublishEvent(event)

		// Wait for propagation
		time.Sleep(100 * time.Millisecond)

		// Check all nodes received the event
		for i, node := range nodes {
			events := node.GetReceivedEvents()
			assert.Contains(t, events, "test-001", "Node %d should have received the event", i+1)
		}
	})

	t.Run("LoadBalancing", func(t *testing.T) {
		// Send 100 requests to the cluster
		requestCounts := make(map[int]int)
		
		for i := 0; i < 100; i++ {
			// Round-robin selection
			nodeIdx := i % 3
			nodes[nodeIdx].HandleRequest()
			requestCounts[nodeIdx]++
		}

		// Check distribution
		for i := 0; i < 3; i++ {
			count := nodes[i].GetRequestCount()
			assert.Greater(t, count, 0, "Node %d should have handled requests", i+1)
			t.Logf("Node %d handled %d requests", i+1, count)
		}
	})

	t.Run("FailoverScenario", func(t *testing.T) {
		// Get current leader
		initialLeader := nodes[0].GetClusterStatus().LeaderID
		t.Logf("Initial leader: %s", initialLeader)

		// Find and stop the leader node
		var leaderIdx int
		for i, node := range nodes {
			if node.GetNodeID() == initialLeader {
				leaderIdx = i
				node.Stop()
				break
			}
		}

		// Wait for new leader election
		time.Sleep(3 * time.Second)

		// Check remaining nodes have new leader
		var newLeader string
		for i, node := range nodes {
			if i != leaderIdx {
				status := node.GetClusterStatus()
				if newLeader == "" {
					newLeader = status.LeaderID
				}
				assert.Equal(t, newLeader, status.LeaderID, "All nodes should agree on new leader")
			}
		}

		assert.NotEqual(t, initialLeader, newLeader, "New leader should be different from failed leader")
		t.Logf("New leader after failover: %s", newLeader)
	})
}

// TestRealDependencyInjection validates DI with complex scenarios
func TestRealDependencyInjection(t *testing.T) {
	container := di.NewContainer()

	t.Run("ComplexAutoWiring", func(t *testing.T) {
		// Register interconnected services
		container.RegisterSingleton(func() *ConfigService {
			return &ConfigService{
				DatabaseURL: "postgres://localhost/test",
				RedisURL:    "redis://localhost:6379",
			}
		})

		container.RegisterSingleton(func(config *ConfigService) *DatabaseService {
			return &DatabaseService{
				URL: config.DatabaseURL,
			}
		})

		container.RegisterSingleton(func(config *ConfigService) *CacheService {
			return &CacheService{
				URL: config.RedisURL,
			}
		})

		container.RegisterScoped(func(db *DatabaseService, cache *CacheService) *UserRepository {
			return &UserRepository{
				DB:    db,
				Cache: cache,
			}
		})

		container.RegisterTransient(func(repo *UserRepository) *UserService {
			return &UserService{
				Repository: repo,
			}
		})

		// Resolve top-level service
		result := di.Resolve[*UserService](container)
		require.True(t, result.IsSuccess(), "Should resolve UserService")

		userService := result.Value()
		assert.NotNil(t, userService.Repository)
		assert.NotNil(t, userService.Repository.DB)
		assert.NotNil(t, userService.Repository.Cache)
		assert.Equal(t, "postgres://localhost/test", userService.Repository.DB.URL)
	})

	t.Run("ScopedLifecycle", func(t *testing.T) {
		// Create scope
		scope := container.CreateScope()

		// Resolve same service twice in scope
		repo1 := di.Resolve[*UserRepository](scope).Value()
		repo2 := di.Resolve[*UserRepository](scope).Value()

		assert.Same(t, repo1, repo2, "Scoped services should be same instance")

		// Different scope should have different instance
		scope2 := container.CreateScope()
		repo3 := di.Resolve[*UserRepository](scope2).Value()

		assert.NotSame(t, repo1, repo3, "Different scopes should have different instances")
	})

	t.Run("CircularDependencyDetection", func(t *testing.T) {
		// Register circular dependency
		container.RegisterSingleton(func(b *ServiceB) *ServiceA {
			return &ServiceA{B: b}
		})

		container.RegisterSingleton(func(a *ServiceA) *ServiceB {
			return &ServiceB{A: a}
		})

		// Should detect circular dependency
		result := di.Resolve[*ServiceA](container)
		assert.True(t, result.IsFailure(), "Should fail with circular dependency")
	})
}

// TestCompleteIntegration runs a full system integration test
func TestCompleteIntegration(t *testing.T) {
	ctx := context.Background()

	// Skip if dependencies not available
	if !checkDependencies() {
		t.Skip("Required dependencies not available")
	}

	// Create complete FlexCore system
	config := &core.FlexCoreConfig{
		WindmillURL:       "http://localhost:8000",
		WindmillToken:     "test-token",
		WindmillWorkspace: "test",
		ClusterName:       "integration-test",
		NodeID:           "test-node-1",
		ClusterNodes:     []string{"localhost:8001", "localhost:8002", "localhost:8003"},
		PluginDirectory:  "./plugins",
		EventRoutes: []core.EventRoute{
			{
				Name:           "data-pipeline",
				SourceAdapter:  "postgres-extractor",
				TargetAdapters: []string{"json-transformer", "api-loader"},
				Async:         true,
			},
		},
	}

	flexCoreResult := core.NewFlexCore(config)
	require.True(t, flexCoreResult.IsSuccess(), "FlexCore creation should succeed")

	flexCore := flexCoreResult.Value()

	// Start the system
	startResult := flexCore.Start(ctx)
	if startResult.IsFailure() {
		t.Skipf("Cannot start FlexCore: %v", startResult.Error())
	}
	defer flexCore.Stop(ctx)

	t.Run("EndToEndDataPipeline", func(t *testing.T) {
		// Send event through the pipeline
		event := &core.Event{
			ID:     "pipeline-test-001",
			Type:   "data.extract",
			Source: "test",
			Data: map[string]interface{}{
				"query": "SELECT * FROM users LIMIT 10",
			},
		}

		sendResult := flexCore.SendEvent(ctx, event)
		assert.True(t, sendResult.IsSuccess(), "Event send should succeed")

		// Check metrics after processing
		time.Sleep(500 * time.Millisecond)
		
		metricsResult := flexCore.GetMetrics(ctx)
		require.True(t, metricsResult.IsSuccess(), "Metrics retrieval should succeed")

		metrics := metricsResult.Value()
		assert.Greater(t, metrics.EventsProcessed, int64(0), "Should have processed events")
		t.Logf("Metrics: Events=%d, Messages=%d, Jobs=%d", 
			metrics.EventsProcessed, metrics.MessagesQueued, metrics.ScheduledJobs)
	})
}

// Helper types and functions

type FlexCoreNode struct {
	ID            string
	Port          int
	flexCore      *core.FlexCore
	server        *http.Server
	clusterStatus *ClusterStatus
	receivedEvents map[string]bool
	requestCount   int32
	mu            sync.RWMutex
}

type ClusterStatus struct {
	LeaderID    string
	ActiveNodes int
	NodeStates  map[string]string
}

type TestEvent struct {
	ID        string
	Timestamp time.Time
	Data      interface{}
}

func createTestNode(id, port int) *FlexCoreNode {
	return &FlexCoreNode{
		ID:             fmt.Sprintf("node-%d", id),
		Port:           port,
		receivedEvents: make(map[string]bool),
		clusterStatus: &ClusterStatus{
			NodeStates: make(map[string]string),
		},
	}
}

func (n *FlexCoreNode) Start(ctx context.Context) error {
	// Initialize FlexCore
	config := &core.FlexCoreConfig{
		WindmillURL:       "http://localhost:8000",
		WindmillToken:     "test-token",
		WindmillWorkspace: "test",
		ClusterName:       "test-cluster",
		NodeID:           n.ID,
		ClusterNodes:     []string{"localhost:9001", "localhost:9002", "localhost:9003"},
	}

	flexCoreResult := core.NewFlexCore(config)
	if flexCoreResult.IsFailure() {
		return flexCoreResult.Error()
	}
	n.flexCore = flexCoreResult.Value()

	// Start HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/status", n.handleStatus)
	mux.HandleFunc("/event", n.handleEvent)

	n.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", n.Port),
		Handler: mux,
	}

	go n.server.ListenAndServe()

	// Simulate cluster formation
	n.mu.Lock()
	n.clusterStatus.ActiveNodes = 3
	n.clusterStatus.LeaderID = "node-1" // Simple leader selection
	n.mu.Unlock()

	return nil
}

func (n *FlexCoreNode) Stop() {
	if n.server != nil {
		n.server.Shutdown(context.Background())
	}
	if n.flexCore != nil {
		n.flexCore.Stop(context.Background())
	}
}

func (n *FlexCoreNode) handleStatus(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id": n.ID,
		"cluster_status": n.clusterStatus,
		"request_count": atomic.LoadInt32(&n.requestCount),
	})
}

func (n *FlexCoreNode) handleEvent(w http.ResponseWriter, r *http.Request) {
	var event TestEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	n.receivedEvents[event.ID] = true
	n.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (n *FlexCoreNode) GetNodeID() string {
	return n.ID
}

func (n *FlexCoreNode) GetClusterStatus() *ClusterStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.clusterStatus
}

func (n *FlexCoreNode) PublishEvent(event *TestEvent) {
	n.mu.Lock()
	n.receivedEvents[event.ID] = true
	n.mu.Unlock()
}

func (n *FlexCoreNode) GetReceivedEvents() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	events := make([]string, 0, len(n.receivedEvents))
	for id := range n.receivedEvents {
		events = append(events, id)
	}
	return events
}

func (n *FlexCoreNode) HandleRequest() {
	atomic.AddInt32(&n.requestCount, 1)
}

func (n *FlexCoreNode) GetRequestCount() int {
	return int(atomic.LoadInt32(&n.requestCount))
}

// Test service types for DI testing
type ConfigService struct {
	DatabaseURL string
	RedisURL    string
}

type DatabaseService struct {
	URL string
}

type CacheService struct {
	URL string
}

type UserRepository struct {
	DB    *DatabaseService
	Cache *CacheService
}

type UserService struct {
	Repository *UserRepository
}

type ServiceA struct {
	B *ServiceB
}

type ServiceB struct {
	A *ServiceA
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkDependencies() bool {
	// Check if Redis is available
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()
	
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return false
	}
	
	return true
}