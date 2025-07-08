#!/bin/bash
# Script para executar testes E2E reais do FlexCore

set -e

echo "=== FLEXCORE E2E REAL TESTS ==="
echo "Testing all components without simulation..."

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Check if we're in flexcore directory
if [ ! -f "go.mod" ] || [ ! -d "core" ]; then
    echo -e "${RED}Error: Must run from flexcore directory${NC}"
    exit 1
fi

# Function to check service
check_service() {
    local name=$1
    local check_cmd=$2
    
    echo -n "Checking $name... "
    if eval $check_cmd > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${YELLOW}✗${NC}"
        return 1
    fi
}

# Start dependencies if needed
echo -e "\n${BLUE}Checking dependencies...${NC}"

# Check Redis
if ! check_service "Redis" "redis-cli ping"; then
    echo "Starting Redis in Docker..."
    docker run -d --name flexcore-redis -p 6379:6379 redis:alpine > /dev/null 2>&1 || true
    sleep 2
fi

# Check PostgreSQL
if ! check_service "PostgreSQL" "pg_isready -h localhost -p 5432"; then
    echo "Starting PostgreSQL in Docker..."
    docker run -d --name flexcore-postgres \
        -e POSTGRES_PASSWORD=flexcore123 \
        -e POSTGRES_DB=flexcore \
        -p 5432:5432 \
        postgres:15-alpine > /dev/null 2>&1 || true
    sleep 3
fi

# Build test plugins
echo -e "\n${BLUE}Building test plugins...${NC}"
mkdir -p plugins

# Build postgres-extractor plugin
if [ -d "plugins/postgres-extractor" ]; then
    echo "Building postgres-extractor..."
    (cd plugins/postgres-extractor && go build -o postgres-extractor) || true
fi

# Build json-transformer plugin  
if [ -d "plugins/json-transformer" ]; then
    echo "Building json-transformer..."
    (cd plugins/json-transformer && go build -o json-transformer) || true
fi

# Build api-loader plugin
if [ -d "plugins/api-loader" ]; then
    echo "Building api-loader..."
    (cd plugins/api-loader && go build -o api-loader) || true
fi

# Get test dependencies
echo -e "\n${BLUE}Getting test dependencies...${NC}"
go get -t ./tests/e2e/...

# Run the tests
echo -e "\n${BLUE}Running E2E tests...${NC}"

# Test categories
declare -A test_categories=(
    ["1_DependencyInjection"]="TestRealDependencyInjection"
    ["2_DistributedCoordination"]="TestRealDistributedCoordination"
    ["3_PluginSystem"]="TestRealPluginExecution"
    ["4_MultiNodeCluster"]="TestRealMultiNodeCluster"
    ["5_CompleteIntegration"]="TestCompleteIntegration"
)

# Track results
total_tests=0
passed_tests=0
failed_tests=0

# Run each test category
for category in "${!test_categories[@]}"; do
    test_name="${test_categories[$category]}"
    echo -e "\n${YELLOW}Running $category: $test_name${NC}"
    
    if go test -v -timeout 30s -run "^${test_name}$" ./tests/e2e/...; then
        echo -e "${GREEN}✓ $category passed${NC}"
        ((passed_tests++))
    else
        echo -e "${RED}✗ $category failed${NC}"
        ((failed_tests++))
    fi
    ((total_tests++))
done

# Run all tests with coverage
echo -e "\n${BLUE}Running all tests with coverage...${NC}"
go test -v -cover -coverprofile=coverage.out ./tests/e2e/...

# Generate coverage report
if [ -f "coverage.out" ]; then
    echo -e "\n${BLUE}Coverage Report:${NC}"
    go tool cover -func=coverage.out | grep -E "(total:|TestReal)" | tail -10
fi

# Summary
echo -e "\n${BLUE}=== TEST SUMMARY ===${NC}"
echo -e "Total test categories: $total_tests"
echo -e "Passed: ${GREEN}$passed_tests${NC}"
echo -e "Failed: ${RED}$failed_tests${NC}"

if [ $failed_tests -eq 0 ]; then
    echo -e "\n${GREEN}🎉 ALL E2E TESTS PASSED!${NC}"
    echo -e "${GREEN}FlexCore distributed system is 100% functional!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed. Check logs above.${NC}"
    exit 1
fi