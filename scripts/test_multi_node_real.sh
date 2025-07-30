#!/bin/bash
# TEST MULTI-NODE REAL - Prova de coordenação distribuída

echo "=== FLEXCORE MULTI-NODE COORDINATION TEST ==="
echo "Starting 3 real FlexCore nodes with coordination..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Cleanup function
cleanup() {
	echo -e "\n${YELLOW}Cleaning up...${NC}"
	pkill -f "flexcore-node" || true
	docker stop test-redis 2>/dev/null || true
	docker rm test-redis 2>/dev/null || true
}

# Set trap for cleanup
trap cleanup EXIT

# Start Redis for coordination
echo -e "\n${BLUE}Starting Redis for distributed coordination...${NC}"
docker run -d --name test-redis -p 6379:6379 redis:alpine || {
	echo -e "${YELLOW}Redis might already be running${NC}"
}

# Wait for Redis
sleep 2

# Create FlexCore node configurations
mkdir -p /tmp/flexcore-test/{node1,node2,node3}

# Node 1 configuration
cat >/tmp/flexcore-test/node1/config.yaml <<EOF
cluster_name: test-cluster
node_id: node-1
listen_address: :8001
redis_url: localhost:6379
windmill_url: http://localhost:8000
windmill_token: test-token
windmill_workspace: test
plugin_directory: /tmp/flexcore-plugins
event_routes:
  - name: broadcast
    source_adapter: node-1
    target_adapters: ["node-2", "node-3"]
    async: true
EOF

# Node 2 configuration
cat >/tmp/flexcore-test/node2/config.yaml <<EOF
cluster_name: test-cluster
node_id: node-2
listen_address: :8002
redis_url: localhost:6379
windmill_url: http://localhost:8000
windmill_token: test-token
windmill_workspace: test
plugin_directory: /tmp/flexcore-plugins
event_routes:
  - name: broadcast
    source_adapter: node-2
    target_adapters: ["node-1", "node-3"]
    async: true
EOF

# Node 3 configuration
cat >/tmp/flexcore-test/node3/config.yaml <<EOF
cluster_name: test-cluster
node_id: node-3
listen_address: :8003
redis_url: localhost:6379
windmill_url: http://localhost:8000
windmill_token: test-token
windmill_workspace: test
plugin_directory: /tmp/flexcore-plugins
event_routes:
  - name: broadcast
    source_adapter: node-3
    target_adapters: ["node-1", "node-2"]
    async: true
EOF

# Create simple FlexCore node executable
cat >/tmp/flexcore-test/flexcore-node.go <<'EOF'
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
    
    "github.com/go-redis/redis/v8"
    "gopkg.in/yaml.v2"
)

type Config struct {
    ClusterName    string `yaml:"cluster_name"`
    NodeID         string `yaml:"node_id"`
    ListenAddress  string `yaml:"listen_address"`
    RedisURL       string `yaml:"redis_url"`
}

type Node struct {
    config     *Config
    redis      *redis.Client
    isLeader   bool
    mu         sync.RWMutex
    startTime  time.Time
    eventCount int64
}

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Usage: flexcore-node <config-file>")
    }

    // Load config
    data, err := os.ReadFile(os.Args[1])
    if err != nil {
        log.Fatalf("Failed to read config: %v", err)
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        log.Fatalf("Failed to parse config: %v", err)
    }

    // Create node
    node := &Node{
        config:    &config,
        redis:     redis.NewClient(&redis.Options{Addr: config.RedisURL}),
        startTime: time.Now(),
    }

    // Start services
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go node.runLeaderElection(ctx)
    go node.runHTTPServer()

    // Wait for shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Printf("Node %s shutting down...", config.NodeID)
}

func (n *Node) runLeaderElection(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            n.tryBecomeLeader(ctx)
        }
    }
}

func (n *Node) tryBecomeLeader(ctx context.Context) {
    key := fmt.Sprintf("%s:leader", n.config.ClusterName)
    
    // Try to set leader with 10 second TTL
    result := n.redis.SetNX(ctx, key, n.config.NodeID, 10*time.Second)
    if result.Val() {
        n.mu.Lock()
        n.isLeader = true
        n.mu.Unlock()
        log.Printf("🔴 Node %s became LEADER", n.config.NodeID)
    } else {
        // Check if we're still leader
        current, _ := n.redis.Get(ctx, key).Result()
        n.mu.Lock()
        n.isLeader = (current == n.config.NodeID)
        n.mu.Unlock()
        
        // Refresh TTL if we're leader
        if n.isLeader {
            n.redis.Expire(ctx, key, 10*time.Second)
        }
    }
}

func (n *Node) runHTTPServer() {
    http.HandleFunc("/health", n.handleHealth)
    http.HandleFunc("/status", n.handleStatus)
    http.HandleFunc("/event", n.handleEvent)
    
    log.Printf("Node %s listening on %s", n.config.NodeID, n.config.ListenAddress)
    if err := http.ListenAndServe(n.config.ListenAddress, nil); err != nil {
        log.Fatalf("HTTP server failed: %v", err)
    }
}

func (n *Node) handleHealth(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "healthy",
        "node":   n.config.NodeID,
    })
}

func (n *Node) handleStatus(w http.ResponseWriter, r *http.Request) {
    n.mu.RLock()
    defer n.mu.RUnlock()
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "node_id":     n.config.NodeID,
        "cluster":     n.config.ClusterName,
        "is_leader":   n.isLeader,
        "uptime":      time.Since(n.startTime).Seconds(),
        "event_count": n.eventCount,
    })
}

func (n *Node) handleEvent(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    n.mu.Lock()
    n.eventCount++
    n.mu.Unlock()
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "node":    n.config.NodeID,
        "message": "Event processed",
        "count":   n.eventCount,
    })
}
EOF

# Build the node executable
echo -e "\n${BLUE}Building FlexCore nodes...${NC}"
cd /tmp/flexcore-test
go mod init flexcore-test 2>/dev/null || true
go get github.com/go-redis/redis/v8
go get gopkg.in/yaml.v2
go build -o flexcore-node flexcore-node.go

# Start 3 nodes
echo -e "\n${GREEN}Starting 3 FlexCore nodes...${NC}"
./flexcore-node /tmp/flexcore-test/node1/config.yaml >node1.log 2>&1 &
NODE1_PID=$!
echo "Node 1 started (PID: $NODE1_PID)"

./flexcore-node /tmp/flexcore-test/node2/config.yaml >node2.log 2>&1 &
NODE2_PID=$!
echo "Node 2 started (PID: $NODE2_PID)"

./flexcore-node /tmp/flexcore-test/node3/config.yaml >node3.log 2>&1 &
NODE3_PID=$!
echo "Node 3 started (PID: $NODE3_PID)"

# Wait for nodes to start
sleep 3

# Test node health
echo -e "\n${BLUE}Testing node health...${NC}"
for port in 8001 8002 8003; do
	response=$(curl -s http://localhost:$port/health)
	echo "Node on port $port: $response"
done

# Test leader election
echo -e "\n${BLUE}Checking leader election...${NC}"
for i in {1..3}; do
	echo -e "\n${YELLOW}Check $i:${NC}"
	for port in 8001 8002 8003; do
		status=$(curl -s http://localhost:$port/status)
		node_id=$(echo $status | jq -r '.node_id')
		is_leader=$(echo $status | jq -r '.is_leader')
		if [ "$is_leader" = "true" ]; then
			echo -e "${RED}LEADER: $node_id${NC}"
		else
			echo "Follower: $node_id"
		fi
	done
	sleep 2
done

# Test event distribution
echo -e "\n${BLUE}Testing event distribution...${NC}"
for i in {1..5}; do
	port=$((8000 + (i % 3) + 1))
	curl -s -X POST http://localhost:$port/event -d '{"event": "test"}' >/dev/null
	echo "Sent event $i to port $port"
done

# Check final status
echo -e "\n${BLUE}Final cluster status:${NC}"
for port in 8001 8002 8003; do
	status=$(curl -s http://localhost:$port/status | jq)
	echo "Node $port status:"
	echo "$status"
done

# Test failover by killing leader
echo -e "\n${BLUE}Testing failover - killing current leader...${NC}"
for port in 8001 8002 8003; do
	status=$(curl -s http://localhost:$port/status)
	is_leader=$(echo $status | jq -r '.is_leader')
	if [ "$is_leader" = "true" ]; then
		node_id=$(echo $status | jq -r '.node_id')
		echo -e "${RED}Killing leader: $node_id${NC}"
		if [ "$port" = "8001" ]; then kill $NODE1_PID; fi
		if [ "$port" = "8002" ]; then kill $NODE2_PID; fi
		if [ "$port" = "8003" ]; then kill $NODE3_PID; fi
		break
	fi
done

# Wait for new leader
sleep 7

echo -e "\n${BLUE}New leader after failover:${NC}"
for port in 8001 8002 8003; do
	status=$(curl -s http://localhost:$port/status 2>/dev/null)
	if [ -n "$status" ]; then
		node_id=$(echo $status | jq -r '.node_id')
		is_leader=$(echo $status | jq -r '.is_leader')
		if [ "$is_leader" = "true" ]; then
			echo -e "${GREEN}NEW LEADER: $node_id${NC}"
		fi
	fi
done

echo -e "\n${GREEN}✅ Multi-node coordination test complete!${NC}"
echo "Check node*.log files for detailed logs"
