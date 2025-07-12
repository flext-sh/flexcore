#!/bin/bash
# E2E Test Script - Tests FlexCore system end-to-end

set -e

echo "=== FLEXCORE E2E INTEGRATION TESTS ==="
echo "Testing complete distributed system with real components..."

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
FLEXCORE_URL="http://localhost:8080"
REDIS_URL="localhost:6379"
POSTGRES_URL="postgresql://flexcore:flexcore123@localhost:5432/flexcore"
TEST_TIMEOUT="120s"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Functions
log_info() {
	echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
	echo -e "${GREEN}[SUCCESS]${NC} $1"
	((TESTS_PASSED++))
}

log_error() {
	echo -e "${RED}[ERROR]${NC} $1"
	((TESTS_FAILED++))
}

log_warning() {
	echo -e "${YELLOW}[WARNING]${NC} $1"
}

run_test() {
	local test_name="$1"
	local test_command="$2"

	((TOTAL_TESTS++))
	log_info "Running test: $test_name"

	if eval "$test_command" >/tmp/test_output.log 2>&1; then
		log_success "$test_name"
		return 0
	else
		log_error "$test_name"
		cat /tmp/test_output.log
		return 1
	fi
}

wait_for_service() {
	local service_name="$1"
	local url="$2"
	local timeout=30

	log_info "Waiting for $service_name to be ready..."

	for i in $(seq 1 $timeout); do
		if curl -sf "$url" >/dev/null 2>&1; then
			log_success "$service_name is ready"
			return 0
		fi
		sleep 1
	done

	log_error "$service_name failed to start within ${timeout}s"
	return 1
}

cleanup() {
	log_info "Cleaning up test environment..."

	# Stop FlexCore instances
	pkill -f "flexcore" || true

	# Clean Docker containers if they exist
	docker stop flexcore-test-redis flexcore-test-postgres 2>/dev/null || true
	docker rm flexcore-test-redis flexcore-test-postgres 2>/dev/null || true

	# Clean test data
	rm -f /tmp/flexcore-test-*
	rm -f /tmp/test_output.log
}

# Trap cleanup on exit
trap cleanup EXIT

# 1. Setup test environment
log_info "Setting up test environment..."

# Start Redis
if ! docker ps --format "table {{.Names}}" | grep -q flexcore-test-redis; then
	log_info "Starting Redis for testing..."
	docker run -d --name flexcore-test-redis -p 6379:6379 redis:alpine
	sleep 2
fi

# Start PostgreSQL
if ! docker ps --format "table {{.Names}}" | grep -q flexcore-test-postgres; then
	log_info "Starting PostgreSQL for testing..."
	docker run -d --name flexcore-test-postgres \
		-e POSTGRES_PASSWORD=flexcore123 \
		-e POSTGRES_DB=flexcore \
		-e POSTGRES_USER=flexcore \
		-p 5432:5432 \
		postgres:15-alpine
	sleep 5
fi

# Build FlexCore
log_info "Building FlexCore..."
cd /home/marlonsc/flext/flexcore
go build -o flexcore-test ./cmd/flexcore/

# Build plugins
log_info "Building plugins..."
mkdir -p plugins-built
if [ -d "plugins/data-processor" ]; then
	cd plugins/data-processor
	go mod init data-processor-plugin 2>/dev/null || true
	go get github.com/hashicorp/go-plugin
	go build -o ../../plugins-built/data-processor
	cd ../..
fi

# Start FlexCore nodes
log_info "Starting FlexCore nodes..."

# Node 1 - Primary
FLEXCORE_PORT=8080 \
	FLEXCORE_NODE_ID=node-1 \
	REDIS_ADDR=localhost:6379 \
	PLUGIN_DIR=./plugins-built \
	./flexcore-test >/tmp/flexcore-node-1.log 2>&1 &
NODE1_PID=$!

# Node 2 - Secondary
FLEXCORE_PORT=8081 \
	FLEXCORE_NODE_ID=node-2 \
	REDIS_ADDR=localhost:6379 \
	PLUGIN_DIR=./plugins-built \
	./flexcore-test -port 8081 -node node-2 >/tmp/flexcore-node-2.log 2>&1 &
NODE2_PID=$!

# Node 3 - Tertiary
FLEXCORE_PORT=8082 \
	FLEXCORE_NODE_ID=node-3 \
	REDIS_ADDR=localhost:6379 \
	PLUGIN_DIR=./plugins-built \
	./flexcore-test -port 8082 -node node-3 >/tmp/flexcore-node-3.log 2>&1 &
NODE3_PID=$!

# Wait for services
wait_for_service "FlexCore Node 1" "$FLEXCORE_URL/health"
wait_for_service "FlexCore Node 2" "http://localhost:8081/health"
wait_for_service "FlexCore Node 3" "http://localhost:8082/health"

# 2. Run E2E Tests

echo -e "\n${YELLOW}=== RUNNING E2E TESTS ===${NC}"

# Test 1: Health Check All Nodes
run_test "Health Check - All Nodes" '
    curl -sf http://localhost:8080/health | jq -e ".status == \"healthy\"" &&
    curl -sf http://localhost:8081/health | jq -e ".status == \"healthy\"" &&
    curl -sf http://localhost:8082/health | jq -e ".status == \"healthy\""
'

# Test 2: Cluster Status and Leader Election
run_test "Cluster Status and Leader Election" '
    # Check cluster status
    response=$(curl -sf http://localhost:8080/cluster)
    echo "$response" | jq -e ".active_nodes >= 1" &&
    
    # Check leader election
    leader_count=0
    for port in 8080 8081 8082; do
        if curl -sf "http://localhost:$port/cluster/leader" | jq -e ".is_leader == true" > /dev/null; then
            ((leader_count++))
        fi
    done
    [ $leader_count -eq 1 ]
'

# Test 3: Event Sending and Processing
run_test "Event Sending and Processing" '
    # Send single event
    event_data="{
        \"id\": \"test-001\",
        \"type\": \"test.event\",
        \"source\": \"e2e-test\",
        \"data\": {
            \"message\": \"Hello FlexCore\",
            \"timestamp\": $(date +%s)
        }
    }"
    
    response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d "$event_data" http://localhost:8080/events)
    
    echo "$response" | jq -e ".status == \"accepted\""
'

# Test 4: Batch Event Processing
run_test "Batch Event Processing" '
    # Send batch of events
    batch_data="[
        {
            \"type\": \"user.created\",
            \"source\": \"e2e-test\",
            \"data\": {\"user_id\": \"user-001\", \"email\": \"test1@test.com\"}
        },
        {
            \"type\": \"order.placed\",
            \"source\": \"e2e-test\", 
            \"data\": {\"order_id\": \"order-001\", \"amount\": 99.99}
        },
        {
            \"type\": \"payment.processed\",
            \"source\": \"e2e-test\",
            \"data\": {\"payment_id\": \"pay-001\", \"status\": \"completed\"}
        }
    ]"
    
    response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d "$batch_data" http://localhost:8080/events/batch)
    
    echo "$response" | jq -e ".total == 3" &&
    echo "$response" | jq -e ".results | length == 3"
'

# Test 5: Plugin System
run_test "Plugin System - List and Execute" '
    # List plugins
    plugins_response=$(curl -sf http://localhost:8080/plugins)
    echo "$plugins_response" | jq -e ".count >= 0" &&
    
    # If data-processor plugin exists, test execution
    if echo "$plugins_response" | jq -e ".plugins[] | select(.name == \"data-processor\")" > /dev/null; then
        execute_data="{
            \"data\": [
                {\"id\": 1, \"name\": \"Item 1\"},
                {\"id\": 2, \"name\": \"Item 2\"}
            ],
            \"transform\": \"uppercase,trim\"
        }"
        
        plugin_response=$(curl -sf -X POST -H "Content-Type: application/json" \
            -d "$execute_data" http://localhost:8080/plugins/data-processor/execute)
        
        echo "$plugin_response" | jq -e ".processor == \"data-processor\""
    else
        echo "No plugins available for testing" >&2
        true
    fi
'

# Test 6: Message Queue Operations
run_test "Message Queue Operations" '
    # Send message to queue
    message_data="{
        \"content\": {
            \"type\": \"notification\",
            \"recipient\": \"user@test.com\",
            \"message\": \"Test notification\"
        },
        \"priority\": 5
    }"
    
    send_response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d "$message_data" http://localhost:8080/queues/notifications/messages)
    
    echo "$send_response" | jq -e ".status == \"queued\"" &&
    
    # Receive messages from queue
    receive_response=$(curl -sf "http://localhost:8080/queues/notifications/messages?max=5")
    echo "$receive_response" | jq -e ".count >= 0"
'

# Test 7: Scheduling Operations
run_test "Scheduling Operations" '
    # Create schedule
    schedule_data="{
        \"name\": \"test-schedule\",
        \"cron\": \"0 */5 * * * *\",
        \"workflow\": \"test-workflow\",
        \"input\": {\"test\": true}
    }"
    
    create_response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d "$schedule_data" http://localhost:8080/schedules)
    
    echo "$create_response" | jq -e ".status == \"created\"" &&
    
    # List schedules
    list_response=$(curl -sf http://localhost:8080/schedules)
    echo "$list_response" | jq -e ".count >= 0"
'

# Test 8: Workflow Execution
run_test "Workflow Execution" '
    # Execute workflow
    workflow_data="{
        \"path\": \"test/simple-workflow\",
        \"input\": {
            \"message\": \"Hello from E2E test\",
            \"timestamp\": $(date +%s)
        }
    }"
    
    exec_response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d "$workflow_data" http://localhost:8080/workflows/execute)
    
    echo "$exec_response" | jq -e ".status == \"started\"" &&
    echo "$exec_response" | jq -e ".job_id"
'

# Test 9: Metrics Collection
run_test "Metrics Collection" '
    # Get Prometheus metrics
    metrics_response=$(curl -sf http://localhost:8080/metrics)
    
    # Check for key metrics
    echo "$metrics_response" | grep -q "flexcore_events_received_total" &&
    echo "$metrics_response" | grep -q "flexcore_active_nodes" &&
    echo "$metrics_response" | grep -q "flexcore_request_duration_seconds"
'

# Test 10: Load Balancing Across Nodes
run_test "Load Balancing Across Nodes" '
    # Send requests to all nodes and verify distribution
    declare -A node_responses
    
    for i in {1..30}; do
        port=$((8080 + (i % 3)))
        response=$(curl -sf "http://localhost:$port/info" 2>/dev/null || echo "{}")
        node_id=$(echo "$response" | jq -r ".node_id // \"unknown\"")
        ((node_responses[$node_id]++))
    done
    
    # Check if requests were distributed
    responding_nodes=${#node_responses[@]}
    [ $responding_nodes -ge 2 ]
'

# Test 11: Failover Simulation
run_test "Failover Simulation" '
    # Get current leader
    current_leader=""
    leader_port=""
    for port in 8080 8081 8082; do
        response=$(curl -sf "http://localhost:$port/cluster/leader" 2>/dev/null || echo "{}")
        if echo "$response" | jq -e ".is_leader == true" > /dev/null; then
            current_leader=$(echo "$response" | jq -r ".node_id")
            leader_port=$port
            break
        fi
    done
    
    if [ -n "$current_leader" ] && [ -n "$leader_port" ]; then
        # Kill the leader process
        if [ "$leader_port" == "8080" ]; then
            kill $NODE1_PID 2>/dev/null || true
        elif [ "$leader_port" == "8081" ]; then
            kill $NODE2_PID 2>/dev/null || true  
        elif [ "$leader_port" == "8082" ]; then
            kill $NODE3_PID 2>/dev/null || true
        fi
        
        # Wait for new leader election
        sleep 8
        
        # Check for new leader
        new_leader_found=false
        for port in 8080 8081 8082; do
            if [ "$port" != "$leader_port" ]; then
                response=$(curl -sf "http://localhost:$port/cluster/leader" 2>/dev/null || echo "{}")
                if echo "$response" | jq -e ".is_leader == true" > /dev/null; then
                    new_leader=$(echo "$response" | jq -r ".node_id")
                    if [ "$new_leader" != "$current_leader" ]; then
                        new_leader_found=true
                        break
                    fi
                fi
            fi
        done
        
        $new_leader_found
    else
        echo "No leader found for failover test" >&2
        false
    fi
'

# Test 12: Stress Test (Mini)
run_test "Mini Stress Test" '
    # Run mini stress test with 100 events
    export TARGET_URL=http://localhost:8080
    export EVENTS_PER_SECOND=50
    export DURATION_SECONDS=2
    export NUM_WORKERS=5
    export BATCH_SIZE=5
    
    timeout 10s go run test_stress_1000_events.go > /tmp/stress_test.log 2>&1
    
    # Check results
    grep -q "events/second" /tmp/stress_test.log &&
    grep -q "Success Rate" /tmp/stress_test.log
'

# 3. Generate Test Report
echo -e "\n${YELLOW}=== TEST RESULTS ===${NC}"

echo "Total Tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
	echo -e "\n${GREEN}🎉 ALL TESTS PASSED! FlexCore system is fully functional!${NC}"
	exit 0
else
	echo -e "\n${RED}❌ $TESTS_FAILED tests failed. Check logs above.${NC}"

	# Show node logs for debugging
	echo -e "\n${YELLOW}=== NODE LOGS ===${NC}"
	for i in 1 2 3; do
		if [ -f "/tmp/flexcore-node-$i.log" ]; then
			echo -e "\n${BLUE}Node $i logs:${NC}"
			tail -20 "/tmp/flexcore-node-$i.log"
		fi
	done

	exit 1
fi
