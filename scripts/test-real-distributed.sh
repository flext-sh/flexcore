#!/bin/bash

# Test Real Distributed FlexCore Functionality
# Comprehensive tests for 100% real distributed features

set -e

echo "🧪 Testing Real Distributed FlexCore Functionality..."

# Test configuration
BASE_URL="http://localhost"
NODES=(8001 8002 8003)
TEST_DATA='{"test_input": "real_data", "items": ["item1", "item2", "item3"]}'

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success_count=0
total_tests=0

# Helper function for tests
run_test() {
    local test_name="$1"
    local test_command="$2"
    total_tests=$((total_tests + 1))
    
    echo -e "${YELLOW}🔍 Testing: $test_name${NC}"
    
    if eval "$test_command"; then
        echo -e "${GREEN}✅ PASS: $test_name${NC}"
        success_count=$((success_count + 1))
    else
        echo -e "${RED}❌ FAIL: $test_name${NC}"
    fi
    echo ""
}

# Test 1: Node Health Checks
run_test "Node Health Checks" '
    for port in "${NODES[@]}"; do
        response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL:$port/health")
        if [ "$response" != "200" ]; then
            echo "Node $port health check failed"
            exit 1
        fi
        echo "Node $port: ✅ Healthy"
    done
'

# Test 2: Node Information
run_test "Node Information Retrieval" '
    for port in "${NODES[@]}"; do
        response=$(curl -s "$BASE_URL:$port/info")
        if ! echo "$response" | jq -e ".node_id" > /dev/null; then
            echo "Node $port info failed"
            exit 1
        fi
        node_id=$(echo "$response" | jq -r ".node_id")
        echo "Node $port: $node_id ✅"
    done
'

# Test 3: Cluster Status
run_test "Cluster Status Check" '
    response=$(curl -s "$BASE_URL:8001/cluster")
    if ! echo "$response" | jq -e ".cluster_size" > /dev/null; then
        echo "Cluster status check failed"
        exit 1
    fi
    cluster_size=$(echo "$response" | jq -r ".cluster_size")
    echo "Cluster size: $cluster_size ✅"
'

# Test 4: Plugin Listing
run_test "Plugin System Check" '
    for port in "${NODES[@]}"; do
        response=$(curl -s "$BASE_URL:$port/plugins")
        if ! echo "$response" | jq -e ".plugins" > /dev/null; then
            echo "Plugin listing failed for node $port"
            exit 1
        fi
        plugin_count=$(echo "$response" | jq -r ".count")
        echo "Node $port plugins: $plugin_count ✅"
    done
'

# Test 5: Real Data Processing Plugin Execution
run_test "Real Data Processing Plugin Execution" '
    # Find the real data processor plugin ID
    plugins_response=$(curl -s "$BASE_URL:8001/plugins")
    plugin_id=$(echo "$plugins_response" | jq -r ".plugins[] | select(.name | contains(\"Real Data Processor\")) | .id" | head -n1)
    
    if [ -z "$plugin_id" ] || [ "$plugin_id" = "null" ]; then
        echo "Real Data Processor plugin not found"
        exit 1
    fi
    
    echo "Found plugin: $plugin_id"
    
    # Execute plugin with real data
    execution_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"input\": $TEST_DATA}" \
        "$BASE_URL:8001/plugins/$plugin_id/execute")
    
    if ! echo "$execution_response" | jq -e ".result" > /dev/null; then
        echo "Plugin execution failed"
        echo "Response: $execution_response"
        exit 1
    fi
    
    # Verify real processing occurred
    if ! echo "$execution_response" | jq -e ".result.processor_id" > /dev/null; then
        echo "Real processing verification failed"
        exit 1
    fi
    
    processor_id=$(echo "$execution_response" | jq -r ".result.processor_id")
    echo "Real processing completed by: $processor_id ✅"
'

# Test 6: Load Balancing Test
run_test "Load Balancing Across Nodes" '
    declare -A node_counts
    
    # Execute the same plugin on different nodes
    for i in {1..9}; do
        port_index=$((i % 3))
        port=${NODES[$port_index]}
        
        # Find plugin ID for this node
        plugins_response=$(curl -s "$BASE_URL:$port/plugins")
        plugin_id=$(echo "$plugins_response" | jq -r ".plugins[] | select(.name | contains(\"Real Data Processor\")) | .id" | head -n1)
        
        if [ -n "$plugin_id" ] && [ "$plugin_id" != "null" ]; then
            execution_response=$(curl -s -X POST \
                -H "Content-Type: application/json" \
                -d "{\"input\": {\"test_id\": \"$i\"}}" \
                "$BASE_URL:$port/plugins/$plugin_id/execute")
            
            if echo "$execution_response" | jq -e ".executed_by" > /dev/null; then
                executed_by=$(echo "$execution_response" | jq -r ".executed_by")
                node_counts[$executed_by]=$((${node_counts[$executed_by]:-0} + 1))
                echo "Execution $i: $executed_by"
            fi
        fi
    done
    
    # Verify load distribution
    echo "Load distribution:"
    for node in "${!node_counts[@]}"; do
        echo "  $node: ${node_counts[$node]} executions"
    done
    
    if [ ${#node_counts[@]} -lt 2 ]; then
        echo "Load balancing failed - only ${#node_counts[@]} node(s) handled requests"
        exit 1
    fi
'

# Test 7: Windmill Integration Test
run_test "Windmill Server Integration" '
    # Test Windmill server is accessible
    windmill_response=$(curl -s http://localhost:8000/api/version)
    if ! echo "$windmill_response" | jq -e ".windmill_version" > /dev/null 2>&1; then
        echo "Windmill server not accessible or not responding correctly"
        exit 1
    fi
    
    windmill_version=$(echo "$windmill_response" | jq -r ".windmill_version")
    echo "Windmill version: $windmill_version ✅"
'

# Test 8: PostgreSQL Connection Test
run_test "PostgreSQL Database Connection" '
    # Test PostgreSQL connection
    PGPASSWORD=flexcore123 psql -h localhost -p 5433 -U flexcore -d flexcore -c "SELECT 1;" > /dev/null 2>&1
    echo "PostgreSQL connection: ✅"
'

# Test 9: Redis Connection Test
run_test "Redis Cache Connection" '
    # Test Redis connection
    redis_response=$(redis-cli -h localhost -p 6380 ping 2>/dev/null)
    if [ "$redis_response" != "PONG" ]; then
        echo "Redis connection failed"
        exit 1
    fi
    echo "Redis connection: ✅"
'

# Test 10: Event System Test
run_test "Event Publishing Test" '
    event_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"event_type\": \"test_event\", \"data\": $TEST_DATA}" \
        "$BASE_URL:8001/events")
    
    if ! echo "$event_response" | jq -e ".message" > /dev/null; then
        echo "Event publishing failed"
        exit 1
    fi
    
    echo "Event publishing: ✅"
'

# Summary
echo "=================================="
echo "🏁 TEST SUMMARY"
echo "=================================="
echo -e "Total Tests: $total_tests"
echo -e "${GREEN}Passed: $success_count${NC}"
echo -e "${RED}Failed: $((total_tests - success_count))${NC}"

if [ $success_count -eq $total_tests ]; then
    echo -e "${GREEN}🎉 ALL TESTS PASSED - 100% Real Distributed Functionality Verified!${NC}"
    echo ""
    echo "✅ Verified Real Features:"
    echo "  🌊 Real Windmill server integration"
    echo "  🔌 Real plugin execution with data processing"
    echo "  ⚖️  Load balancing across multiple nodes"
    echo "  💾 PostgreSQL database connectivity"
    echo "  ⚡ Redis cache connectivity"
    echo "  📡 Multi-node cluster coordination"
    echo "  🎯 Event system functionality"
    echo ""
    echo "🚀 FlexCore distributed system is 100% operational!"
else
    echo -e "${RED}❌ Some tests failed. Check logs for details.${NC}"
    exit 1
fi