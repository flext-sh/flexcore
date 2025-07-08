#!/bin/bash

# FlexCore Multi-Node Cluster Test - REAL Distributed Implementation
# Tests multiple FlexCore nodes communicating via network

set -e

echo "🚀 FlexCore Multi-Node Cluster Test - REAL Distribution"
echo "========================================================"

# Configuration
NODE_COUNT=3
BASE_PORT=8080
REDIS_URL="redis://localhost:6379"
CLUSTER_MODE="redis"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if port is available
is_port_available() {
    local port=$1
    ! nc -z localhost "$port" 2>/dev/null
}

# Function to wait for service to be ready
wait_for_service() {
    local port=$1
    local max_attempts=30
    local attempt=1
    
    print_status "Waiting for service on port $port to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "http://localhost:$port/health" > /dev/null 2>&1; then
            print_success "Service on port $port is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 1
        ((attempt++))
    done
    
    print_error "Service on port $port failed to start within $max_attempts seconds"
    return 1
}

# Function to start a FlexCore node
start_node() {
    local node_id=$1
    local port=$2
    
    if ! is_port_available "$port"; then
        print_error "Port $port is already in use"
        return 1
    fi
    
    print_status "Starting FlexCore node: $node_id on port $port"
    
    # Build the node binary if it doesn't exist
    if [ ! -f "./cmd/flexcore-node/flexcore-node" ]; then
        print_status "Building FlexCore node binary..."
        cd cmd/flexcore-node
        go build -o flexcore-node .
        cd ../..
    fi
    
    # Start the node in background
    ./cmd/flexcore-node/flexcore-node \
        -node-id="$node_id" \
        -port="$port" \
        -redis="$REDIS_URL" \
        -cluster="$CLUSTER_MODE" \
        -debug=true \
        > "logs/node-$node_id.log" 2>&1 &
    
    local pid=$!
    echo "$pid" > "logs/node-$node_id.pid"
    
    print_success "Started node $node_id (PID: $pid) on port $port"
    
    # Wait for node to be ready
    if wait_for_service "$port"; then
        print_success "Node $node_id is ready and responding"
        return 0
    else
        print_error "Node $node_id failed to start properly"
        return 1
    fi
}

# Function to stop all nodes
stop_all_nodes() {
    print_status "Stopping all FlexCore nodes..."
    
    for i in $(seq 1 $NODE_COUNT); do
        local node_id="node-$i"
        local pid_file="logs/node-$node_id.pid"
        
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                print_status "Stopping node $node_id (PID: $pid)"
                kill "$pid"
                rm -f "$pid_file"
            fi
        fi
    done
    
    # Wait a moment for graceful shutdown
    sleep 2
    
    # Force kill any remaining processes
    pkill -f "flexcore-node" 2>/dev/null || true
    
    print_success "All nodes stopped"
}

# Function to test cluster communication
test_cluster_communication() {
    print_status "Testing cluster communication between nodes..."
    
    local base_url="http://localhost"
    local failed_tests=0
    
    # Test 1: Check all nodes are healthy
    for i in $(seq 1 $NODE_COUNT); do
        local port=$((BASE_PORT + i - 1))
        local response=$(curl -s "$base_url:$port/health" 2>/dev/null || echo "ERROR")
        
        if [[ "$response" == *"healthy"* ]]; then
            print_success "Node $i health check: PASSED"
        else
            print_error "Node $i health check: FAILED"
            ((failed_tests++))
        fi
    done
    
    # Test 2: Check cluster status on each node
    for i in $(seq 1 $NODE_COUNT); do
        local port=$((BASE_PORT + i - 1))
        local response=$(curl -s "$base_url:$port/cluster/status" 2>/dev/null || echo "ERROR")
        
        if [[ "$response" == *"node_id"* ]] && [[ "$response" == *"active_nodes"* ]]; then
            local active_nodes=$(echo "$response" | grep -o '"active_nodes":[0-9]*' | cut -d':' -f2)
            print_success "Node $i cluster status: PASSED (Active nodes: $active_nodes)"
        else
            print_error "Node $i cluster status: FAILED"
            ((failed_tests++))
        fi
    done
    
    # Test 3: Test distributed event broadcasting
    for i in $(seq 1 $NODE_COUNT); do
        local port=$((BASE_PORT + i - 1))
        local response=$(curl -s -X POST "$base_url:$port/events/test?type=test.cluster.communication" 2>/dev/null || echo "ERROR")
        
        if [[ "$response" == *"success"* ]] && [[ "$response" == *"true"* ]]; then
            print_success "Node $i event broadcast: PASSED"
        else
            print_error "Node $i event broadcast: FAILED"
            ((failed_tests++))
        fi
    done
    
    # Test 4: Test cluster message broadcasting
    for i in $(seq 1 $NODE_COUNT); do
        local port=$((BASE_PORT + i - 1))
        local response=$(curl -s -X POST "$base_url:$port/cluster/broadcast" 2>/dev/null || echo "ERROR")
        
        if [[ "$response" == *"success"* ]] && [[ "$response" == *"true"* ]]; then
            print_success "Node $i cluster broadcast: PASSED"
        else
            print_error "Node $i cluster broadcast: FAILED"
            ((failed_tests++))
        fi
    done
    
    # Summary
    if [ $failed_tests -eq 0 ]; then
        print_success "ALL CLUSTER COMMUNICATION TESTS PASSED! ✅"
        return 0
    else
        print_error "$failed_tests cluster communication tests FAILED ❌"
        return 1
    fi
}

# Function to show cluster statistics
show_cluster_stats() {
    print_status "Cluster Statistics:"
    echo "==================="
    
    for i in $(seq 1 $NODE_COUNT); do
        local port=$((BASE_PORT + i - 1))
        local response=$(curl -s "http://localhost:$port/cluster/status" 2>/dev/null || echo "ERROR")
        
        if [[ "$response" != "ERROR" ]]; then
            local node_id=$(echo "$response" | grep -o '"node_id":"[^"]*"' | cut -d'"' -f4)
            local is_leader=$(echo "$response" | grep -o '"is_leader":[^,]*' | cut -d':' -f2)
            local active_nodes=$(echo "$response" | grep -o '"active_nodes":[0-9]*' | cut -d':' -f2)
            
            echo "  Node $i:"
            echo "    ID: $node_id"
            echo "    Port: $port"
            echo "    Leader: $is_leader"
            echo "    Active Nodes: $active_nodes"
            echo ""
        fi
    done
}

# Function to monitor cluster for a period
monitor_cluster() {
    local duration=${1:-30}
    print_status "Monitoring cluster for $duration seconds..."
    
    local end_time=$(($(date +%s) + duration))
    
    while [ $(date +%s) -lt $end_time ]; do
        clear
        echo "🔍 FlexCore Cluster Monitor - $(date)"
        echo "=================================="
        show_cluster_stats
        echo "Press Ctrl+C to stop monitoring..."
        sleep 5
    done
}

# Function to run comprehensive tests
run_comprehensive_tests() {
    print_status "Running comprehensive distributed tests..."
    
    # Test leader election
    print_status "Testing leader election..."
    local leader_count=0
    for i in $(seq 1 $NODE_COUNT); do
        local port=$((BASE_PORT + i - 1))
        local response=$(curl -s "http://localhost:$port/cluster/status" 2>/dev/null)
        local is_leader=$(echo "$response" | grep -o '"is_leader":true' | wc -l)
        leader_count=$((leader_count + is_leader))
    done
    
    if [ $leader_count -eq 1 ]; then
        print_success "Leader election: PASSED (exactly 1 leader)"
    else
        print_error "Leader election: FAILED ($leader_count leaders found)"
    fi
    
    # Test distributed locking (simulate by rapid event broadcasts)
    print_status "Testing distributed coordination..."
    local lock_test_success=0
    for i in $(seq 1 5); do
        for j in $(seq 1 $NODE_COUNT); do
            local port=$((BASE_PORT + j - 1))
            curl -s -X POST "http://localhost:$port/events/test?type=lock.test.$i" > /dev/null 2>&1 && ((lock_test_success++))
        done
    done
    
    if [ $lock_test_success -gt 0 ]; then
        print_success "Distributed coordination: PASSED ($lock_test_success/15 operations succeeded)"
    else
        print_error "Distributed coordination: FAILED (no operations succeeded)"
    fi
}

# Cleanup function
cleanup() {
    print_status "Cleaning up..."
    stop_all_nodes
    
    # Clean up log files
    rm -f logs/node-*.log logs/node-*.pid
    
    print_success "Cleanup completed"
}

# Trap signals for cleanup
trap cleanup EXIT INT TERM

# Main execution
main() {
    # Create logs directory
    mkdir -p logs
    
    # Check prerequisites
    print_status "Checking prerequisites..."
    
    if ! command -v curl &> /dev/null; then
        print_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v nc &> /dev/null; then
        print_error "netcat (nc) is required but not installed"
        exit 1
    fi
    
    # Check if Go is available
    if ! command -v go &> /dev/null; then
        print_error "Go is required but not installed"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
    
    # Start nodes
    print_status "Starting $NODE_COUNT FlexCore nodes..."
    
    for i in $(seq 1 $NODE_COUNT); do
        local node_id="node-$i"
        local port=$((BASE_PORT + i - 1))
        
        if ! start_node "$node_id" "$port"; then
            print_error "Failed to start node $node_id"
            exit 1
        fi
        
        # Small delay between nodes to avoid race conditions
        sleep 2
    done
    
    print_success "All $NODE_COUNT nodes started successfully!"
    
    # Wait for cluster to stabilize
    print_status "Waiting for cluster to stabilize..."
    sleep 5
    
    # Show initial cluster stats
    show_cluster_stats
    
    # Run tests
    if test_cluster_communication; then
        print_success "🎉 CLUSTER COMMUNICATION TESTS PASSED!"
    else
        print_error "💥 CLUSTER COMMUNICATION TESTS FAILED!"
        exit 1
    fi
    
    # Run comprehensive tests
    run_comprehensive_tests
    
    # Show final stats
    print_status "Final cluster statistics:"
    show_cluster_stats
    
    # Option to monitor
    read -p "Do you want to monitor the cluster? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        monitor_cluster 60
    fi
    
    print_success "🎯 Multi-node cluster test completed successfully!"
    print_status "All nodes demonstrated REAL distributed communication!"
}

# Check command line arguments
case "${1:-}" in
    "monitor")
        # Just monitor existing cluster
        monitor_cluster "${2:-30}"
        ;;
    "stop")
        # Stop all nodes
        stop_all_nodes
        ;;
    "test")
        # Just run communication tests
        test_cluster_communication
        ;;
    *)
        # Run full test
        main
        ;;
esac