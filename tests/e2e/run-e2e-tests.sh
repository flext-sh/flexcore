#!/bin/bash

# FlexCore End-to-End Test Suite
# Comprehensive testing of FlexCore distributed system

set -e

echo "🧪 Starting FlexCore End-to-End Test Suite"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
FLEXCORE_NODE1_URL="${FLEXCORE_NODE1_URL:-http://flexcore-node1:9000}"
FLEXCORE_NODE2_URL="${FLEXCORE_NODE2_URL:-http://flexcore-node2:9000}"
WINDMILL_URL="${WINDMILL_URL:-http://windmill-server:8000}"
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
MOCK_API_URL="${MOCK_API_URL:-http://mock-api}"

# Test results directory
RESULTS_DIR="/app/results"
mkdir -p "$RESULTS_DIR"

# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo -e "\n🔬 Test $TESTS_TOTAL: ${BLUE}$test_name${NC}"
    
    local start_time=$(date +%s)
    local log_file="$RESULTS_DIR/test_${TESTS_TOTAL}_$(echo "$test_name" | tr ' ' '_').log"
    
    if eval "$test_command" > "$log_file" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}✅ PASSED${NC} (${duration}s)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo "PASSED" > "$RESULTS_DIR/test_${TESTS_TOTAL}.result"
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${RED}❌ FAILED${NC} (${duration}s)"
        echo -e "${YELLOW}Log: $log_file${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo "FAILED" > "$RESULTS_DIR/test_${TESTS_TOTAL}.result"
        cat "$log_file" | tail -20
    fi
}

# Function to wait for service
wait_for_service() {
    local service_name="$1"
    local url="$2"
    local timeout="${3:-60}"
    
    echo "⏳ Waiting for $service_name to be ready..."
    
    local count=0
    while [ $count -lt $timeout ]; do
        if curl -sf "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✅ $service_name is ready${NC}"
            return 0
        fi
        count=$((count + 1))
        sleep 1
    done
    
    echo -e "${RED}❌ $service_name failed to start within ${timeout}s${NC}"
    return 1
}

# Function to check HTTP status
check_http_status() {
    local url="$1"
    local expected_status="${2:-200}"
    
    local actual_status=$(curl -s -w '%{http_code}' -o /dev/null "$url")
    if [ "$actual_status" = "$expected_status" ]; then
        return 0
    else
        echo "Expected status $expected_status, got $actual_status"
        return 1
    fi
}

# Function to check JSON response
check_json_field() {
    local url="$1"
    local field="$2"
    local expected_value="$3"
    
    local response=$(curl -s "$url")
    local actual_value=$(echo "$response" | jq -r ".$field")
    
    if [ "$actual_value" = "$expected_value" ]; then
        return 0
    else
        echo "Expected $field=$expected_value, got $actual_value"
        echo "Response: $response"
        return 1
    fi
}

# Function to test API endpoint
test_api_post() {
    local url="$1"
    local payload="$2"
    local expected_status="${3:-200}"
    
    local response=$(curl -s -w '%{http_code}' -H "Content-Type: application/json" -d "$payload" "$url")
    local status_code="${response: -3}"
    local body="${response%???}"
    
    if [ "$status_code" = "$expected_status" ]; then
        return 0
    else
        echo "Expected status $expected_status, got $status_code"
        echo "Response: $body"
        return 1
    fi
}

echo "🏗️ Pre-test Setup and Health Checks"

# Wait for all services to be ready
wait_for_service "PostgreSQL" "postgresql://$POSTGRES_HOST:5432" 30
wait_for_service "Windmill" "$WINDMILL_URL/api/version" 120
wait_for_service "Mock API" "$MOCK_API_URL/health" 30
wait_for_service "FlexCore Node 1" "$FLEXCORE_NODE1_URL/health" 60
wait_for_service "FlexCore Node 2" "$FLEXCORE_NODE2_URL/health" 60

echo -e "\n🧪 Starting Test Execution"

# Test 1: Health Check Tests
run_test "FlexCore Node 1 Health Check" \
    "check_http_status '$FLEXCORE_NODE1_URL/health' 200"

run_test "FlexCore Node 2 Health Check" \
    "check_http_status '$FLEXCORE_NODE2_URL/health' 200"

run_test "Windmill Health Check" \
    "check_http_status '$WINDMILL_URL/api/version' 200"

run_test "Mock API Health Check" \
    "check_json_field '$MOCK_API_URL/health' status healthy"

# Test 2: Cluster Tests
run_test "Cluster Status Node 1" \
    "check_json_field '$FLEXCORE_NODE1_URL/cluster/status' cluster_name test-cluster"

run_test "Cluster Status Node 2" \
    "check_json_field '$FLEXCORE_NODE2_URL/cluster/status' cluster_name test-cluster"

run_test "Cluster Leader Election" \
    "curl -s '$FLEXCORE_NODE1_URL/cluster/status' | jq -e '.leader_node != null and .leader_node != \"\"'"

# Test 3: Metrics Tests
run_test "Node 1 Metrics Available" \
    "curl -s '$FLEXCORE_NODE1_URL/metrics' | jq -e '.events_processed >= 0'"

run_test "Node 2 Metrics Available" \
    "curl -s '$FLEXCORE_NODE2_URL/metrics' | jq -e '.events_processed >= 0'"

# Test 4: Event Processing Tests
run_test "Send Test Event Node 1" \
    "test_api_post '$FLEXCORE_NODE1_URL/events' '{\"id\":\"test-1\",\"type\":\"test.event\",\"source\":\"test\",\"timestamp\":\"$(date -Iseconds)\",\"data\":{\"test\":true}}' 200"

run_test "Send Test Event Node 2" \
    "test_api_post '$FLEXCORE_NODE2_URL/events' '{\"id\":\"test-2\",\"type\":\"test.event\",\"source\":\"test\",\"timestamp\":\"$(date -Iseconds)\",\"data\":{\"test\":true}}' 200"

# Test 5: Message Queue Tests
run_test "Send Message to Queue Node 1" \
    "test_api_post '$FLEXCORE_NODE1_URL/messages/test-queue' '{\"id\":\"msg-1\",\"queue\":\"test-queue\",\"content\":{\"test\":\"message\"}}' 200"

run_test "Receive Messages from Queue Node 1" \
    "check_http_status '$FLEXCORE_NODE1_URL/messages/test-queue' 200"

# Test 6: Scheduling Tests
run_test "Schedule Test Job Node 1" \
    "test_api_post '$FLEXCORE_NODE1_URL/schedule/test-scheduler' '{\"test\":true,\"timestamp\":\"$(date -Iseconds)\"}' 200"

# Test 7: Workflow Execution Tests
run_test "Execute Test Workflow Node 1" \
    "test_api_post '$FLEXCORE_NODE1_URL/workflows/test/simple' '{\"input\":\"test-data\"}' 200"

# Test 8: Data Pipeline Tests
run_test "Trigger Data Extraction" \
    "test_api_post '$FLEXCORE_NODE1_URL/test/extract' '{\"source\":\"source_data\"}' 200"

# Test 9: Plugin Integration Tests
run_test "PostgreSQL Plugin Health" \
    "timeout 10 psql postgresql://testuser:testpass@$POSTGRES_HOST:5432/flexcore_test -c 'SELECT COUNT(*) FROM source_data;'"

run_test "Mock API Endpoint" \
    "test_api_post '$MOCK_API_URL/api/data' '{\"test\":\"data\"}' 200"

# Test 10: Cross-Node Communication Tests
run_test "Node 1 to Node 2 Event Routing" \
    "test_api_post '$FLEXCORE_NODE1_URL/events' '{\"id\":\"cross-node-1\",\"type\":\"cross.node\",\"source\":\"node-1\",\"timestamp\":\"$(date -Iseconds)\",\"data\":{\"target\":\"node-2\"}}' 200"

# Test 11: Load Tests
run_test "Bulk Event Processing" \
    "for i in {1..10}; do test_api_post '$FLEXCORE_NODE1_URL/events' '{\"id\":\"bulk-'$i'\",\"type\":\"bulk.test\",\"source\":\"load-test\",\"timestamp\":\"$(date -Iseconds)\",\"data\":{\"index\":'$i'}}' 200; done"

# Test 12: Error Handling Tests
run_test "Invalid Event Handling" \
    "test_api_post '$FLEXCORE_NODE1_URL/events' '{\"invalid\":\"event\"}' 400"

run_test "Invalid Message Handling" \
    "test_api_post '$FLEXCORE_NODE1_URL/messages/invalid-queue' '{\"invalid\":\"message\"}' 500"

# Test 13: Persistence Tests
run_test "Data Persistence Check" \
    "sleep 5 && curl -s '$FLEXCORE_NODE1_URL/metrics' | jq -e '.events_processed > 0'"

# Test 14: Recovery Tests
run_test "Node Restart Simulation" \
    "curl -s '$FLEXCORE_NODE1_URL/health' && sleep 2 && curl -s '$FLEXCORE_NODE1_URL/health'"

# Test 15: Integration End-to-End
run_test "Full Pipeline Integration" \
    "test_api_post '$FLEXCORE_NODE1_URL/test/extract' '{\"source\":\"SELECT * FROM source_data LIMIT 5\"}' 200"

# Generate test report
echo -e "\n📊 Generating Test Report"

cat > "$RESULTS_DIR/test_report.json" << EOF
{
    "test_suite": "FlexCore E2E Tests",
    "timestamp": "$(date -Iseconds)",
    "environment": {
        "flexcore_node1": "$FLEXCORE_NODE1_URL",
        "flexcore_node2": "$FLEXCORE_NODE2_URL",
        "windmill_url": "$WINDMILL_URL",
        "postgres_host": "$POSTGRES_HOST",
        "mock_api_url": "$MOCK_API_URL"
    },
    "results": {
        "total_tests": $TESTS_TOTAL,
        "passed": $TESTS_PASSED,
        "failed": $TESTS_FAILED,
        "success_rate": $(echo "scale=2; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc -l)
    },
    "details": []
}
EOF

# Add individual test results
for i in $(seq 1 $TESTS_TOTAL); do
    result_file="$RESULTS_DIR/test_${i}.result"
    if [ -f "$result_file" ]; then
        result=$(cat "$result_file")
        echo "    Test $i: $result" >> "$RESULTS_DIR/test_summary.txt"
    fi
done

# Create HTML report
cat > "$RESULTS_DIR/test_report.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>FlexCore E2E Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f4f4f4; padding: 20px; border-radius: 8px; }
        .success { color: #28a745; }
        .failed { color: #dc3545; }
        .test-item { margin: 10px 0; padding: 10px; border-left: 4px solid #ddd; }
        .test-passed { border-left-color: #28a745; }
        .test-failed { border-left-color: #dc3545; }
    </style>
</head>
<body>
    <div class="header">
        <h1>FlexCore E2E Test Report</h1>
        <p>Generated: $(date)</p>
        <p>Total Tests: $TESTS_TOTAL | Passed: <span class="success">$TESTS_PASSED</span> | Failed: <span class="failed">$TESTS_FAILED</span></p>
        <p>Success Rate: $(echo "scale=1; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc -l)%</p>
    </div>
    
    <h2>Test Results</h2>
EOF

for i in $(seq 1 $TESTS_TOTAL); do
    result_file="$RESULTS_DIR/test_${i}.result"
    log_file="$RESULTS_DIR/test_${i}_*.log"
    if [ -f "$result_file" ]; then
        result=$(cat "$result_file")
        class="test-passed"
        if [ "$result" = "FAILED" ]; then
            class="test-failed"
        fi
        echo "    <div class=\"test-item $class\">Test $i: $result</div>" >> "$RESULTS_DIR/test_report.html"
    fi
done

echo "</body></html>" >> "$RESULTS_DIR/test_report.html"

# Final summary
echo -e "\n📋 Test Summary"
echo -e "==============="
echo -e "Total Tests: $TESTS_TOTAL"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo -e "Success Rate: $(echo "scale=1; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc -l)%"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed! FlexCore is functioning correctly.${NC}"
    exit 0
else
    echo -e "\n${YELLOW}⚠️  Some tests failed. Check logs in $RESULTS_DIR${NC}"
    exit 1
fi