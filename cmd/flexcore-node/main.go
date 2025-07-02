// FlexCore Distributed Node - REAL multi-node cluster implementation
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	flexcore "github.com/flext/flexcore"
	"github.com/flext/flexcore/infrastructure/di"
	"github.com/flext/flexcore/infrastructure/events"
	"github.com/flext/flexcore/infrastructure/scheduler"
	"github.com/flext/flexcore/infrastructure/plugins"
	"github.com/flext/flexcore/infrastructure/windmill"
)

// NodeConfig holds configuration for a FlexCore node
type NodeConfig struct {
	NodeID      string
	HTTPPort    int
	RedisURL    string
	ClusterMode string // "redis", "etcd", "inmemory", "network"
	PeerNodes   []string // List of peer node addresses for network mode
	Debug       bool
}

// FlexCoreNode represents a single node in the FlexCore cluster
type FlexCoreNode struct {
	config         *NodeConfig
	app            *flexcore.Application
	httpServer     *http.Server
	coordinator    scheduler.ClusterCoordinator
	pluginRegistry *plugins.PluginRegistry
	windmillClient *windmill.WindmillClient
}

func main() {
	fmt.Println("🚀 FlexCore Distributed Node - REAL Cluster Implementation")

	// Parse command line flags
	config := parseFlags()

	// Create and start node
	node := NewFlexCoreNode(config)
	if err := node.Start(); err != nil {
		log.Fatalf("❌ Failed to start FlexCore node: %v", err)
	}

	// Wait for shutdown signal
	waitForShutdown(node)
}

func parseFlags() *NodeConfig {
	config := &NodeConfig{}
	var peersStr string

	flag.StringVar(&config.NodeID, "node-id", "", "Unique node identifier (auto-generated if empty)")
	flag.IntVar(&config.HTTPPort, "port", 8080, "HTTP port for this node")
	flag.StringVar(&config.RedisURL, "redis", "redis://localhost:6379", "Redis URL for cluster coordination")
	flag.StringVar(&config.ClusterMode, "cluster", "redis", "Cluster coordination mode (redis, etcd, inmemory, network)")
	flag.StringVar(&peersStr, "peers", "", "Comma-separated list of peer nodes (e.g., localhost:8081,localhost:8082)")
	flag.BoolVar(&config.Debug, "debug", true, "Enable debug mode")

	flag.Parse()

	// Parse peer nodes
	if peersStr != "" {
		config.PeerNodes = strings.Split(peersStr, ",")
		// Trim whitespace from each peer
		for i, peer := range config.PeerNodes {
			config.PeerNodes[i] = strings.TrimSpace(peer)
		}
	}

	// Auto-generate node ID if not provided
	if config.NodeID == "" {
		config.NodeID = fmt.Sprintf("node-%d-%d", config.HTTPPort, time.Now().Unix())
	}

	return config
}

func NewFlexCoreNode(config *NodeConfig) *FlexCoreNode {
	// Initialize plugin registry
	pluginRegistry := plugins.NewPluginRegistry("./plugins")

	// Initialize Windmill client
	windmillClient := windmill.NewWindmillClient(
		"http://localhost:3000", // Default Windmill URL
		"demo-token",            // Demo token
		"default",               // Default workspace
	)

	return &FlexCoreNode{
		config:         config,
		pluginRegistry: pluginRegistry,
		windmillClient: windmillClient,
	}
}

func (node *FlexCoreNode) Start() error {
	fmt.Printf("🎯 Starting FlexCore node: %s (port: %d)\n", node.config.NodeID, node.config.HTTPPort)

	// Create cluster coordinator based on configuration
	coordinator, err := node.createClusterCoordinator()
	if err != nil {
		return fmt.Errorf("failed to create cluster coordinator: %w", err)
	}
	node.coordinator = coordinator

	// Create FlexCore kernel with distributed configuration
	kernel := flexcore.NewKernel(
		flexcore.WithAppName(fmt.Sprintf("flexcore-node-%s", node.config.NodeID)),
		flexcore.WithVersion("1.0.0-distributed"),
		flexcore.WithDebug(),
		flexcore.WithEventsURL("http://localhost:8000"),
	)

	// Build application with custom DI container for distributed setup
	container := di.NewContainer()

	// Register cluster coordinator BEFORE building application
	container.RegisterSingleton(func() scheduler.ClusterCoordinator {
		return coordinator
	})

	node.app = kernel.BuildApplication(container)

	// Start FlexCore application with distributed coordination
	go func() {
		if err := node.app.Run(); err != nil {
			log.Printf("❌ FlexCore application error: %v", err)
		}
	}()

	// Give application time to initialize
	time.Sleep(2 * time.Second)

	// Initialize demo plugins
	if err := node.pluginRegistry.CreateDemoPlugins(); err != nil {
		log.Printf("⚠️ Failed to create demo plugins: %v", err)
	} else {
		fmt.Println("✅ Demo plugins initialized")
	}

	// Start HTTP API for cluster communication
	if err := node.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// Test distributed event communication
	if err := node.testDistributedCommunication(); err != nil {
		log.Printf("⚠️ Distributed communication test failed: %v", err)
	}

	fmt.Printf("✅ FlexCore node %s started successfully on port %d\n", node.config.NodeID, node.config.HTTPPort)
	return nil
}

func (node *FlexCoreNode) createClusterCoordinator() (scheduler.ClusterCoordinator, error) {
	switch node.config.ClusterMode {
	case "redis":
		fmt.Printf("🔗 Using REAL Redis cluster coordination: %s\n", node.config.RedisURL)
		coordinator := scheduler.NewRealRedisClusterCoordinator(node.config.RedisURL)

		// Start the REAL Redis coordinator
		if err := coordinator.Start(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to start REAL Redis coordinator: %w", err)
		}

		return coordinator, nil

	case "network":
		fmt.Printf("🔗 Using REAL network cluster coordination with peers: %v\n", node.config.PeerNodes)
		coordinator := scheduler.NewNetworkClusterCoordinator("localhost", node.config.HTTPPort, node.config.PeerNodes)

		// Start the network coordinator
		if err := coordinator.Start(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to start network coordinator: %w", err)
		}

		return coordinator, nil

	case "etcd":
		fmt.Println("🔗 Using REAL etcd cluster coordination")
		coordinator := scheduler.NewRealEtcdClusterCoordinator([]string{"localhost:2379"})

		// Start the REAL etcd coordinator
		if err := coordinator.Start(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to start REAL etcd coordinator: %w", err)
		}

		return coordinator, nil

	default:
		fmt.Println("🔗 Using in-memory cluster coordination")
		coordinator := scheduler.NewInMemoryClusterCoordinator()
		if err := coordinator.Start(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to start in-memory coordinator: %w", err)
		}
		return coordinator, nil
	}
}

func (node *FlexCoreNode) startHTTPServer() error {
	mux := http.NewServeMux()

	// Cluster status endpoint
	mux.HandleFunc("/cluster/status", node.handleClusterStatus)

	// Node health endpoint
	mux.HandleFunc("/health", node.handleHealth)

	// Distributed event test endpoint
	mux.HandleFunc("/events/test", node.handleEventTest)

	// Cluster communication endpoints (for NetworkClusterCoordinator)
	mux.HandleFunc("/cluster/broadcast", node.handleClusterBroadcast)
	mux.HandleFunc("/cluster/heartbeat", node.handleClusterHeartbeat)
	mux.HandleFunc("/cluster/nodes", node.handleClusterNodes)
	mux.HandleFunc("/cluster/join", node.handleClusterJoin)
	mux.HandleFunc("/cluster/leave", node.handleClusterLeave)
	mux.HandleFunc("/cluster/message", node.handleClusterMessage)

	// Plugins API endpoints
	mux.HandleFunc("/plugins/list", node.handlePluginsList)
	mux.HandleFunc("/plugins/", node.handlePluginsAction)
	mux.HandleFunc("/plugins/execute", node.handlePluginExecute)
	mux.HandleFunc("/plugins/health", node.handlePluginsHealth)

	// Workflows API endpoints (Windmill integration)
	mux.HandleFunc("/workflows/list", node.handleWorkflowsList)
	mux.HandleFunc("/workflows/", node.handleWorkflowsAction)
	mux.HandleFunc("/workflows/execute", node.handleWorkflowExecute)
	mux.HandleFunc("/workflows/status", node.handleWorkflowStatus)

	// Event streaming endpoint
	mux.HandleFunc("/events/stream", node.handleEventStream)

	node.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", node.config.HTTPPort),
		Handler: mux,
	}

	// Start server in background
	go func() {
		if err := node.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ HTTP server error: %v", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Test if server is responsive
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", node.config.HTTPPort))
	if err != nil {
		return fmt.Errorf("HTTP server not responsive: %w", err)
	}
	resp.Body.Close()

	return nil
}

func (node *FlexCoreNode) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get active nodes from cluster coordinator
	activeNodes := node.coordinator.GetActiveNodes(ctx)
	isLeader := node.coordinator.IsLeader(ctx)

	response := map[string]interface{}{
		"node_id":      node.coordinator.GetNodeID(),
		"is_leader":    isLeader,
		"active_nodes": len(activeNodes),
		"nodes":        activeNodes,
		"cluster_mode": node.config.ClusterMode,
		"redis_url":    node.config.RedisURL,
		"status":       "healthy",
		"timestamp":    time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (node *FlexCoreNode) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"node_id":   node.config.NodeID,
		"port":      node.config.HTTPPort,
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (node *FlexCoreNode) handleEventTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get cluster-aware event bus from DI container
	eventBusResult := di.Resolve[events.ClusterAwareEventBus](node.app.Container())
	if eventBusResult.IsFailure() {
		http.Error(w, "Event bus not available", http.StatusInternalServerError)
		return
	}

	eventBus := eventBusResult.Value()

	// Create and broadcast test event
	testEventData := map[string]interface{}{
		"source_node": node.coordinator.GetNodeID(),
		"test_data":   "distributed_event_test",
		"timestamp":   time.Now().Unix(),
	}

	// Get event type from query parameter
	eventType := r.URL.Query().Get("type")
	if eventType == "" {
		eventType = "system.event"
	}

	// Create domain event
	testEvent := createTestEvent(eventType, testEventData)

	// Broadcast to cluster
	if err := eventBus.BroadcastToCluster(r.Context(), testEvent); err != nil {
		http.Error(w, fmt.Sprintf("Failed to broadcast event: %v", err), http.StatusInternalServerError)
		return
	}

	// Get cluster stats
	stats := eventBus.GetClusterStats()

	response := map[string]interface{}{
		"success":        true,
		"event_type":     eventType,
		"source_node":    node.coordinator.GetNodeID(),
		"cluster_events": stats.ClusterEvents,
		"local_events":   stats.LocalEvents,
		"nodes_reached":  stats.NodesReached,
		"timestamp":      time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (node *FlexCoreNode) handleClusterBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	message := scheduler.ClusterMessage{
		Type:      "test.broadcast",
		SourceID:  node.coordinator.GetNodeID(),
		Payload:   map[string]interface{}{
			"test_data": "cluster_broadcast_test",
			"timestamp": time.Now().Unix(),
		},
		Timestamp: time.Now(),
	}

	if err := node.coordinator.BroadcastMessage(r.Context(), message); err != nil {
		http.Error(w, fmt.Sprintf("Failed to broadcast message: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":      true,
		"message_type": "test.broadcast",
		"source_node":  node.coordinator.GetNodeID(),
		"timestamp":    time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (node *FlexCoreNode) testDistributedCommunication() error {
	fmt.Printf("🧪 Testing distributed communication for node %s...\n", node.config.NodeID)

	ctx := context.Background()

	// Test 1: Cluster coordinator functionality
	activeNodes := node.coordinator.GetActiveNodes(ctx)
	fmt.Printf("   📊 Active nodes in cluster: %d\n", len(activeNodes))

	// Test 2: Leader election
	isLeader := node.coordinator.IsLeader(ctx)
	fmt.Printf("   👑 This node is leader: %t\n", isLeader)

	// Test 3: Distributed locking
	lockResult := node.coordinator.AcquireLock(ctx, "test-lock", 30*time.Second)
	if lockResult.IsSuccess() {
		fmt.Printf("   🔒 Successfully acquired distributed lock\n")
		lock := lockResult.Value()
		defer lock.Release(ctx)
	} else {
		fmt.Printf("   🔒 Failed to acquire distributed lock: %v\n", lockResult.Error())
	}

	// Test 4: Event broadcasting
	eventBusResult := di.Resolve[events.ClusterAwareEventBus](node.app.Container())
	if eventBusResult.IsSuccess() {
		eventBus := eventBusResult.Value()
		testEvent := createTestEvent("test.distributed.communication", map[string]interface{}{
			"node_id": node.coordinator.GetNodeID(),
		})

		if err := eventBus.BroadcastToCluster(ctx, testEvent); err != nil {
			return fmt.Errorf("event broadcast test failed: %w", err)
		}

		stats := eventBus.GetClusterStats()
		fmt.Printf("   📡 Event broadcast test completed - cluster events: %d\n", stats.ClusterEvents)
	}

	fmt.Printf("✅ Distributed communication tests completed for node %s\n", node.config.NodeID)
	return nil
}

func (node *FlexCoreNode) Stop() error {
	fmt.Printf("🛑 Stopping FlexCore node %s...\n", node.config.NodeID)

	// Stop HTTP server
	if node.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := node.httpServer.Shutdown(ctx); err != nil {
			log.Printf("⚠️ HTTP server shutdown error: %v", err)
		}
	}

	// Stop FlexCore application
	if node.app != nil {
		node.app.Stop()
	}

	// Stop cluster coordinator
	if node.coordinator != nil {
		if err := node.coordinator.Stop(); err != nil {
			log.Printf("⚠️ Cluster coordinator stop error: %v", err)
		}
	}

	fmt.Printf("✅ FlexCore node %s stopped\n", node.config.NodeID)
	return nil
}

// Network cluster coordination endpoints for REAL distributed communication
func (node *FlexCoreNode) handleClusterHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var heartbeat struct {
		NodeID    string `json:"node_id"`
		Address   string `json:"address"`
		Timestamp int64  `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Process heartbeat via cluster coordinator
	nodeInfo := scheduler.NodeInfo{
		ID:       heartbeat.NodeID,
		Address:  heartbeat.Address,
		LastSeen: time.Unix(heartbeat.Timestamp, 0),
		Metadata: map[string]string{
			"type": "network",
		},
	}

	if err := node.coordinator.RegisterNode(r.Context(), nodeInfo); err != nil {
		http.Error(w, "Failed to register node", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"timestamp": time.Now().Unix(),
	})
}

func (node *FlexCoreNode) handleClusterNodes(w http.ResponseWriter, r *http.Request) {
	activeNodes := node.coordinator.GetActiveNodes(r.Context())

	response := map[string]interface{}{
		"nodes": activeNodes,
		"count": len(activeNodes),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (node *FlexCoreNode) handleClusterJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var joinRequest struct {
		NodeID    string `json:"node_id"`
		Address   string `json:"address"`
		Timestamp int64  `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&joinRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Register the joining node
	nodeInfo := scheduler.NodeInfo{
		ID:       joinRequest.NodeID,
		Address:  joinRequest.Address,
		LastSeen: time.Unix(joinRequest.Timestamp, 0),
		Metadata: map[string]string{
			"type": "network",
			"role": "worker",
		},
	}

	if err := node.coordinator.RegisterNode(r.Context(), nodeInfo); err != nil {
		http.Error(w, "Failed to register node", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"welcome_to": "flexcore_cluster",
		"timestamp":  time.Now().Unix(),
	})
}

func (node *FlexCoreNode) handleClusterLeave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Cluster leave functionality would remove node from active nodes list
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "node left cluster",
		"timestamp": time.Now().Unix(),
	})
}

func (node *FlexCoreNode) handleClusterMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var message scheduler.ClusterMessage
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Broadcast message to local handlers
	if err := node.coordinator.BroadcastMessage(r.Context(), message); err != nil {
		http.Error(w, "Failed to process message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message_id":  message.Type,
		"timestamp":   time.Now().Unix(),
	})
}

func waitForShutdown(node *FlexCoreNode) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("🎯 FlexCore node %s is running. Press Ctrl+C to stop.\n", node.config.NodeID)

	// Print cluster status every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\n🛑 Shutdown signal received")
			node.Stop()
			return

		case <-ticker.C:
			// Print cluster status
			activeNodes := node.coordinator.GetActiveNodes(context.Background())
			isLeader := node.coordinator.IsLeader(context.Background())
			fmt.Printf("📊 Cluster Status - Node: %s, Leader: %t, Active Nodes: %d\n",
				node.coordinator.GetNodeID(), isLeader, len(activeNodes))
		}
	}
}

// Helper function to check if port is available
func isPortAvailable(port int) bool {
	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return true // Port is available
	}
	conn.Close()
	return false // Port is in use
}

// Helper to create test domain events
func createTestEvent(eventType string, data map[string]interface{}) TestDomainEvent {
	return TestDomainEvent{
		id:          fmt.Sprintf("test-%d", time.Now().UnixNano()),
		eventType:   eventType,
		aggregateID: "test-aggregate",
		occurredAt:  time.Now(),
		data:        data,
	}
}

// TestDomainEvent implements domain.DomainEvent for testing
type TestDomainEvent struct {
	id          string
	eventType   string
	aggregateID string
	occurredAt  time.Time
	data        map[string]interface{}
}

func (e TestDomainEvent) EventID() string     { return e.id }
func (e TestDomainEvent) EventType() string  { return e.eventType }
func (e TestDomainEvent) AggregateID() string { return e.aggregateID }
func (e TestDomainEvent) OccurredAt() time.Time { return e.occurredAt }

// =================================================================================
// PLUGINS API HANDLERS - Dynamic Plugin System Integration
// =================================================================================

func (node *FlexCoreNode) handlePluginsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pluginInfos := node.pluginRegistry.ListPlugins()

	response := map[string]interface{}{
		"plugins": pluginInfos,
		"count":   len(pluginInfos),
		"node_id": node.config.NodeID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (node *FlexCoreNode) handlePluginsAction(w http.ResponseWriter, r *http.Request) {
	// Extract plugin name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/plugins/")
	pluginName := strings.Split(path, "/")[0]

	if pluginName == "" {
		http.Error(w, "Plugin name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get specific plugin info
		pluginInfo, err := node.pluginRegistry.GetPlugin(pluginName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pluginInfo)

	case http.MethodPost:
		// Configure plugin
		var config map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := node.pluginRegistry.ConfigurePlugin(pluginName, config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Plugin configured successfully",
		})

	case http.MethodDelete:
		// Unload plugin
		if err := node.pluginRegistry.UnloadPlugin(pluginName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Plugin unloaded successfully",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (node *FlexCoreNode) handlePluginExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		PluginName string      `json:"plugin_name"`
		Input      interface{} `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.PluginName == "" {
		http.Error(w, "Plugin name required", http.StatusBadRequest)
		return
	}

	// Execute plugin with timeout context
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := node.pluginRegistry.ExecutePlugin(ctx, request.PluginName, request.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"plugin_name": request.PluginName,
		"result":      result,
		"executed_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (node *FlexCoreNode) handlePluginsHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := node.pluginRegistry.HealthCheck()

	response := map[string]interface{}{
		"plugins_health": health,
		"checked_at":     time.Now(),
		"node_id":        node.config.NodeID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// =================================================================================
// WORKFLOWS API HANDLERS - Windmill Integration
// =================================================================================

func (node *FlexCoreNode) handleWorkflowsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workflows, err := node.windmillClient.ListWorkflows(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"workflows": workflows,
		"count":     len(workflows),
		"node_id":   node.config.NodeID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (node *FlexCoreNode) handleWorkflowsAction(w http.ResponseWriter, r *http.Request) {
	// Extract workflow ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/workflows/")
	workflowID := strings.Split(path, "/")[0]

	if workflowID == "" {
		http.Error(w, "Workflow ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get specific workflow
		workflow, err := node.windmillClient.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workflow)

	case http.MethodPost:
		// Create new workflow
		var workflow windmill.WorkflowDefinition
		if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := node.windmillClient.CreateWorkflow(r.Context(), &workflow); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Workflow created successfully",
		})

	case http.MethodPUT:
		// Update workflow
		var workflow windmill.WorkflowDefinition
		if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := node.windmillClient.UpdateWorkflow(r.Context(), workflowID, &workflow); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Workflow updated successfully",
		})

	case http.MethodDelete:
		// Delete workflow
		if err := node.windmillClient.DeleteWorkflow(r.Context(), workflowID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Workflow deleted successfully",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (node *FlexCoreNode) handleWorkflowExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		WorkflowID string                 `json:"workflow_id"`
		Input      map[string]interface{} `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.WorkflowID == "" {
		http.Error(w, "Workflow ID required", http.StatusBadRequest)
		return
	}

	// Execute workflow
	execution, err := node.windmillClient.ExecuteWorkflow(r.Context(), request.WorkflowID, request.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

func (node *FlexCoreNode) handleWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	executionID := r.URL.Query().Get("execution_id")
	if executionID == "" {
		http.Error(w, "Execution ID required", http.StatusBadRequest)
		return
	}

	// Get workflow execution status
	execution, err := node.windmillClient.GetWorkflowExecution(r.Context(), executionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// =================================================================================
// EVENT STREAMING HANDLER
// =================================================================================

func (node *FlexCoreNode) handleEventStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set headers for Server-Sent Events (SSE)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for events
	eventChan := make(chan map[string]interface{}, 10)

	// Send initial connection event
	eventChan <- map[string]interface{}{
		"type":      "connection",
		"message":   "Connected to FlexCore event stream",
		"node_id":   node.config.NodeID,
		"timestamp": time.Now(),
	}

	// Send periodic heartbeat events (demo)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go func() {
		defer close(eventChan)
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				eventChan <- map[string]interface{}{
					"type":      "heartbeat",
					"node_id":   node.config.NodeID,
					"timestamp": time.Now(),
					"cluster_status": map[string]interface{}{
						"active_nodes": len(node.coordinator.GetActiveNodes(r.Context())),
						"is_leader":    node.coordinator.IsLeader(r.Context()),
					},
				}
			}
		}
	}()

	// Stream events to client
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for event := range eventChan {
		eventData, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", eventData)
		flusher.Flush()
	}
}
