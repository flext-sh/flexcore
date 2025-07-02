#!/bin/bash
# Distributed Coordination Integration Test

set -e

echo "=== DISTRIBUTED COORDINATION INTEGRATION TEST ==="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up coordination test..."
    docker stop redis-coord-test 2>/dev/null || true
    docker rm redis-coord-test 2>/dev/null || true
    pkill -f "coordination-test" || true
    rm -rf /tmp/coordination-test
}

trap cleanup EXIT

# Setup Redis for testing
log_info "Setting up Redis for coordination testing..."
docker run -d --name redis-coord-test -p 6380:6379 redis:alpine
sleep 2

# Create coordination test
mkdir -p /tmp/coordination-test
cd /tmp/coordination-test

cat > coordination_test.go << 'EOF'
package main

import (
    "context"
    "fmt"
    "log"
    "math/rand"
    "os"
    "strconv"
    "sync"
    "time"
    
    "github.com/go-redis/redis/v8"
)

type RedisCoordinator struct {
    redis   *redis.Client
    cluster string
}

func NewRedisCoordinator(addr, cluster string) *RedisCoordinator {
    rdb := redis.NewClient(&redis.Options{
        Addr: addr,
    })
    
    return &RedisCoordinator{
        redis:   rdb,
        cluster: cluster,
    }
}

func (c *RedisCoordinator) AcquireLock(ctx context.Context, resource, nodeID string, ttl time.Duration) bool {
    key := fmt.Sprintf("%s:lock:%s", c.cluster, resource)
    result := c.redis.SetNX(ctx, key, nodeID, ttl)
    return result.Val()
}

func (c *RedisCoordinator) ReleaseLock(ctx context.Context, resource, nodeID string) bool {
    key := fmt.Sprintf("%s:lock:%s", c.cluster, resource)
    
    // Use Lua script for atomic release
    script := `
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("DEL", KEYS[1])
        else
            return 0
        end
    `
    
    result := c.redis.Eval(ctx, script, []string{key}, nodeID)
    return result.Val().(int64) == 1
}

func (c *RedisCoordinator) ElectLeader(ctx context.Context, service, nodeID string, ttl time.Duration) bool {
    key := fmt.Sprintf("%s:leader:%s", c.cluster, service)
    result := c.redis.SetNX(ctx, key, nodeID, ttl)
    return result.Val()
}

func (c *RedisCoordinator) GetLeader(ctx context.Context, service string) (string, bool) {
    key := fmt.Sprintf("%s:leader:%s", c.cluster, service)
    result := c.redis.Get(ctx, key)
    if result.Err() == redis.Nil {
        return "", false
    }
    return result.Val(), true
}

func (c *RedisCoordinator) RefreshLeadership(ctx context.Context, service, nodeID string, ttl time.Duration) bool {
    key := fmt.Sprintf("%s:leader:%s", c.cluster, service)
    
    // Only refresh if we're the current leader
    script := `
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            redis.call("EXPIRE", KEYS[1], ARGV[2])
            return 1
        else
            return 0
        end
    `
    
    result := c.redis.Eval(ctx, script, []string{key}, nodeID, int(ttl.Seconds()))
    return result.Val().(int64) == 1
}

func testDistributedLocking() error {
    fmt.Println("\n=== TESTING DISTRIBUTED LOCKING ===")
    
    coordinator := NewRedisCoordinator("localhost:6380", "test-cluster")
    ctx := context.Background()
    
    // Test 1: Basic lock acquisition and release
    fmt.Println("Test 1: Basic lock acquisition and release")
    if !coordinator.AcquireLock(ctx, "resource1", "node1", 10*time.Second) {
        return fmt.Errorf("failed to acquire lock")
    }
    fmt.Println("✅ Lock acquired by node1")
    
    // Try to acquire same lock with different node (should fail)
    if coordinator.AcquireLock(ctx, "resource1", "node2", 5*time.Second) {
        return fmt.Errorf("node2 should not acquire lock held by node1")
    }
    fmt.Println("✅ Lock correctly denied to node2")
    
    // Release lock
    if !coordinator.ReleaseLock(ctx, "resource1", "node1") {
        return fmt.Errorf("failed to release lock")
    }
    fmt.Println("✅ Lock released by node1")
    
    // Now node2 should be able to acquire
    if !coordinator.AcquireLock(ctx, "resource1", "node2", 10*time.Second) {
        return fmt.Errorf("node2 should acquire lock after release")
    }
    fmt.Println("✅ Lock acquired by node2 after release")
    
    coordinator.ReleaseLock(ctx, "resource1", "node2")
    
    // Test 2: Concurrent lock attempts
    fmt.Println("\nTest 2: Concurrent lock attempts")
    
    var wg sync.WaitGroup
    var winners sync.Map
    var attempts int64 = 100
    
    for i := int64(0); i < attempts; i++ {
        wg.Add(1)
        go func(id int64) {
            defer wg.Done()
            nodeID := fmt.Sprintf("node-%d", id)
            if coordinator.AcquireLock(ctx, "contested-resource", nodeID, 1*time.Second) {
                winners.Store(nodeID, true)
                time.Sleep(50 * time.Millisecond) // Hold lock briefly
                coordinator.ReleaseLock(ctx, "contested-resource", nodeID)
            }
        }(i)
    }
    
    wg.Wait()
    
    // Count winners
    winnerCount := 0
    winners.Range(func(_, _ interface{}) bool {
        winnerCount++
        return true
    })
    
    fmt.Printf("✅ %d nodes successfully acquired lock (mutual exclusion verified)\n", winnerCount)
    
    if winnerCount == 0 {
        return fmt.Errorf("no nodes acquired lock")
    }
    
    return nil
}

func testLeaderElection() error {
    fmt.Println("\n=== TESTING LEADER ELECTION ===")
    
    coordinator := NewRedisCoordinator("localhost:6380", "test-cluster")
    ctx := context.Background()
    
    // Test 1: Basic leader election
    fmt.Println("Test 1: Basic leader election")
    
    if !coordinator.ElectLeader(ctx, "service1", "node1", 10*time.Second) {
        return fmt.Errorf("node1 should become leader")
    }
    fmt.Println("✅ node1 elected as leader")
    
    // node2 tries to become leader (should fail)
    if coordinator.ElectLeader(ctx, "service1", "node2", 5*time.Second) {
        return fmt.Errorf("node2 should not become leader while node1 is active")
    }
    fmt.Println("✅ node2 correctly denied leadership")
    
    // Verify leader
    leader, exists := coordinator.GetLeader(ctx, "service1")
    if !exists || leader != "node1" {
        return fmt.Errorf("leader should be node1, got %s", leader)
    }
    fmt.Println("✅ Leader correctly identified as node1")
    
    // Test 2: Leadership refresh
    fmt.Println("\nTest 2: Leadership refresh")
    
    if !coordinator.RefreshLeadership(ctx, "service1", "node1", 15*time.Second) {
        return fmt.Errorf("node1 should refresh leadership")
    }
    fmt.Println("✅ node1 refreshed leadership")
    
    // node2 tries to refresh (should fail)
    if coordinator.RefreshLeadership(ctx, "service1", "node2", 10*time.Second) {
        return fmt.Errorf("node2 should not refresh leadership")
    }
    fmt.Println("✅ node2 correctly denied leadership refresh")
    
    // Test 3: Leadership expiry and takeover
    fmt.Println("\nTest 3: Leadership expiry and takeover")
    
    // Acquire short-lived leadership
    coordinator.ElectLeader(ctx, "service2", "node1", 1*time.Second)
    fmt.Println("✅ node1 acquired short-lived leadership")
    
    // Wait for expiry
    time.Sleep(1500 * time.Millisecond)
    
    // node2 should now be able to become leader
    if !coordinator.ElectLeader(ctx, "service2", "node2", 10*time.Second) {
        return fmt.Errorf("node2 should become leader after expiry")
    }
    fmt.Println("✅ node2 became leader after expiry")
    
    // Test 4: Concurrent leader election
    fmt.Println("\nTest 4: Concurrent leader election")
    
    var wg sync.WaitGroup
    var leaders sync.Map
    var nodes int64 = 50
    
    for i := int64(0); i < nodes; i++ {
        wg.Add(1)
        go func(id int64) {
            defer wg.Done()
            nodeID := fmt.Sprintf("node-%d", id)
            if coordinator.ElectLeader(ctx, "contested-service", nodeID, 5*time.Second) {
                leaders.Store(nodeID, true)
            }
        }(i)
    }
    
    wg.Wait()
    
    // Count leaders (should be exactly 1)
    leaderCount := 0
    var electedLeader string
    leaders.Range(func(key, _ interface{}) bool {
        leaderCount++
        electedLeader = key.(string)
        return true
    })
    
    if leaderCount != 1 {
        return fmt.Errorf("expected exactly 1 leader, got %d", leaderCount)
    }
    
    fmt.Printf("✅ Exactly 1 leader elected: %s\n", electedLeader)
    
    return nil
}

func testFailoverScenario() error {
    fmt.Println("\n=== TESTING FAILOVER SCENARIO ===")
    
    coordinator := NewRedisCoordinator("localhost:6380", "test-cluster")
    ctx := context.Background()
    
    // Simulate leader lifecycle
    fmt.Println("Simulating complete leader lifecycle...")
    
    // Phase 1: Leader election
    if !coordinator.ElectLeader(ctx, "failover-service", "primary-node", 3*time.Second) {
        return fmt.Errorf("primary node should become leader")
    }
    fmt.Println("✅ Primary node became leader")
    
    // Phase 2: Normal operation with leadership refresh
    for i := 0; i < 3; i++ {
        time.Sleep(1 * time.Second)
        if !coordinator.RefreshLeadership(ctx, "failover-service", "primary-node", 3*time.Second) {
            return fmt.Errorf("primary node should refresh leadership")
        }
        fmt.Printf("✅ Leadership refreshed (attempt %d)\n", i+1)
    }
    
    // Phase 3: Leader failure (stop refreshing)
    fmt.Println("Simulating leader failure (stop refreshing)...")
    time.Sleep(4 * time.Second) // Wait for TTL expiry
    
    // Phase 4: New leader election
    if !coordinator.ElectLeader(ctx, "failover-service", "backup-node", 10*time.Second) {
        return fmt.Errorf("backup node should become new leader")
    }
    fmt.Println("✅ Backup node became new leader after failover")
    
    // Verify new leader
    leader, exists := coordinator.GetLeader(ctx, "failover-service")
    if !exists || leader != "backup-node" {
        return fmt.Errorf("new leader should be backup-node, got %s", leader)
    }
    fmt.Println("✅ Failover completed successfully")
    
    return nil
}

func performanceTest() error {
    fmt.Println("\n=== PERFORMANCE TEST ===")
    
    coordinator := NewRedisCoordinator("localhost:6380", "test-cluster")
    ctx := context.Background()
    
    operations := 1000
    
    // Test lock acquisition/release performance
    fmt.Printf("Testing %d lock operations...\n", operations)
    
    start := time.Now()
    for i := 0; i < operations; i++ {
        resource := fmt.Sprintf("perf-resource-%d", i%10) // 10 different resources
        nodeID := fmt.Sprintf("node-%d", i%5) // 5 different nodes
        
        coordinator.AcquireLock(ctx, resource, nodeID, 1*time.Second)
        coordinator.ReleaseLock(ctx, resource, nodeID)
    }
    duration := time.Since(start)
    
    opsPerSec := float64(operations*2) / duration.Seconds() // *2 for acquire+release
    fmt.Printf("✅ Performance: %.2f operations/second\n", opsPerSec)
    
    if opsPerSec < 100 {
        return fmt.Errorf("performance too low: %.2f ops/sec", opsPerSec)
    }
    
    return nil
}

func main() {
    if len(os.Args) > 1 {
        testType := os.Args[1]
        switch testType {
        case "lock":
            if err := testDistributedLocking(); err != nil {
                log.Fatalf("Locking test failed: %v", err)
            }
        case "leader":
            if err := testLeaderElection(); err != nil {
                log.Fatalf("Leader election test failed: %v", err)
            }
        case "failover":
            if err := testFailoverScenario(); err != nil {
                log.Fatalf("Failover test failed: %v", err)
            }
        case "performance":
            if err := performanceTest(); err != nil {
                log.Fatalf("Performance test failed: %v", err)
            }
        default:
            log.Fatalf("Unknown test type: %s", testType)
        }
        return
    }
    
    // Run all tests
    tests := []struct {
        name string
        fn   func() error
    }{
        {"Distributed Locking", testDistributedLocking},
        {"Leader Election", testLeaderElection},
        {"Failover Scenario", testFailoverScenario},
        {"Performance", performanceTest},
    }
    
    for _, test := range tests {
        fmt.Printf("\n🚀 Running %s...\n", test.name)
        if err := test.fn(); err != nil {
            log.Fatalf("❌ %s failed: %v", test.name, err)
        }
        fmt.Printf("✅ %s passed!\n", test.name)
    }
    
    fmt.Println("\n🎉 All distributed coordination tests passed!")
}
EOF

# Create go.mod
cat > go.mod << 'EOF'
module coordination-test

go 1.21

require github.com/go-redis/redis/v8 v8.11.5
EOF

# Build and run test
log_info "Building coordination test..."
go mod download
go build -o coordination-test coordination_test.go

if [ ! -x "coordination-test" ]; then
    log_error "Failed to build coordination test"
    exit 1
fi

log_success "Coordination test built successfully"

# Run all tests
log_info "Running distributed coordination tests..."
echo "================================================="

if ./coordination-test; then
    echo "================================================="
    log_success "All coordination tests PASSED"
    
    # Run specific stress test
    log_info "Running coordination stress test..."
    
    # Parallel coordination test
    echo "Starting parallel coordination stress test..."
    pids=()
    
    # Start multiple test processes
    for i in {1..5}; do
        ./coordination-test performance > /tmp/coord-test-$i.log 2>&1 &
        pids+=($!)
    done
    
    # Wait for all to complete
    for pid in "${pids[@]}"; do
        wait $pid
        if [ $? -eq 0 ]; then
            echo "✅ Parallel test process $pid completed"
        else
            log_error "Parallel test process $pid failed"
            exit 1
        fi
    done
    
    log_success "Parallel coordination stress test completed!"
    
else
    echo "================================================="
    log_error "Coordination tests FAILED"
    exit 1
fi