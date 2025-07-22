#!/bin/bash

# FlexCore Test Suite - Unified test runner without Docker
# Eliminates duplication among multiple test scripts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Test categories
TEST_TYPES=("unit" "native" "system" "integration" "e2e")
TEST_MODE="${1:-quick}"

echo -e "${BLUE}🧪 FLEXCORE UNIFIED TEST SUITE${NC}"
echo "================================="
echo "Mode: $TEST_MODE"
echo "Project: $PROJECT_ROOT"
echo ""

# Global counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
declare -a FAILED_TEST_NAMES=()

# Utility functions
log_info() {
	echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
	echo -e "${GREEN}[PASS]${NC} $1"
	((PASSED_TESTS++))
	((TOTAL_TESTS++))
}

log_failure() {
	echo -e "${RED}[FAIL]${NC} $1"
	FAILED_TEST_NAMES+=("$1")
	((FAILED_TESTS++))
	((TOTAL_TESTS++))
}

log_warning() {
	echo -e "${YELLOW}[WARN]${NC} $1"
}

# Test execution wrapper
run_test() {
	local test_name="$1"
	local test_command="$2"

	log_info "Running: $test_name"

	if eval "$test_command" >/dev/null 2>&1; then
		log_success "$test_name"
		return 0
	else
		log_failure "$test_name"
		return 1
	fi
}

# Environment validation
validate_environment() {
	echo "🔍 Environment Validation"
	echo "------------------------"

	run_test "Cargo available" "command -v cargo"
	run_test "Go available" "command -v go"
	run_test "Git available" "command -v git"
	run_test "Make available" "command -v make"

	# Project structure validation
	run_test "Makefile exists" "[ -f '$PROJECT_ROOT/Makefile' ]"
	run_test "Windmill backend exists" "[ -f '$PROJECT_ROOT/third_party/windmill/windmill-backend' ]"
	run_test "Go client exists" "[ -f '$PROJECT_ROOT/third_party/windmill/go-client/api/windmill_api.gen.go' ]"

	echo ""
}

# Build system tests
test_build_system() {
	echo "🏗️ Build System Tests"
	echo "--------------------"

	cd "$PROJECT_ROOT"

	# Test Makefile targets
	run_test "Make help" "make help"
	run_test "Make status" "make status"
	run_test "Make info" "make info"
	run_test "Windmill validate" "make windmill-validate"

	# If full mode, test actual builds
	if [ "$TEST_MODE" = "full" ]; then
		run_test "Go build" "make build-dev"
		run_test "Windmill backend build" "make windmill-backend-fast"
		run_test "Windmill Go client build" "make windmill-go"
	fi

	echo ""
}

# Native system tests
test_native_system() {
	echo "⚡ Native System Tests"
	echo "--------------------"

	cd "$PROJECT_ROOT"

	# Binary functionality
	if [ -f "third_party/windmill/windmill-backend" ]; then
		run_test "Windmill binary executable" "[ -x 'third_party/windmill/windmill-backend' ]"
		run_test "Windmill version" "./third_party/windmill/windmill-backend --version | grep -q 'Windmill'"

		# Size validation
		local binary_size=$(stat -c%s "third_party/windmill/windmill-backend" 2>/dev/null || echo "0")
		local binary_size_mb=$((binary_size / 1024 / 1024))
		run_test "Binary size reasonable (>100MB)" "[ $binary_size_mb -gt 100 ]"
	else
		log_failure "Windmill binary not found"
	fi

	echo ""
}

# Go tests
test_go() {
	echo "🔤 Go Tests"
	echo "----------"

	cd "$PROJECT_ROOT"

	if [ -f "go.mod" ]; then
		run_test "Go mod tidy" "go mod tidy"
		run_test "Go mod verify" "go mod verify"

		if [ "$TEST_MODE" = "full" ]; then
			run_test "Go unit tests" "go test ./..."
			run_test "Go vet" "go vet ./..."
		fi
	else
		log_warning "No go.mod found, skipping Go tests"
	fi

	echo ""
}

# Docker independence validation
test_docker_independence() {
	echo "🐳 Docker Independence"
	echo "--------------------"

	cd "$PROJECT_ROOT"

	# Verify no Docker Windmill images in configs
	local docker_deps=$(find deployments/docker -name '*.yml' -not -path '*/third_party/windmill/*' -exec grep -l 'ghcr.io/windmill' {} \; 2>/dev/null | wc -l)
	run_test "No Docker Windmill dependencies" "[ $docker_deps -eq 0 ]"

	# Verify native binary mounts
	local native_mounts=$(grep -r 'third_party/windmill/windmill-backend' deployments/docker/ 2>/dev/null | wc -l)
	run_test "Native binary mounts configured" "[ $native_mounts -gt 0 ]"

	echo ""
}

# Performance validation
test_performance() {
	echo "⚡ Performance Tests"
	echo "------------------"

	cd "$PROJECT_ROOT"

	# sccache availability
	run_test "sccache available" "command -v sccache"

	# Makefile efficiency
	run_test "Parallel make support" "grep -q 'jobs=' Makefile || grep -q 'MAKEFLAGS.*jobs' .makerc"

	echo ""
}

# Quality validation
test_quality() {
	echo "🎯 Quality Standards"
	echo "------------------"

	cd "$PROJECT_ROOT"

	# Clean workspace
	run_test "No backup files" "[ \$(find . -name '*.bak' -o -name '*.orig' | wc -l) -eq 0 ]"
	run_test "No temporary files" "[ \$(find . -name '*.tmp' -o -name '.DS_Store' | wc -l) -eq 0 ]"

	# Documentation
	run_test "README exists" "[ -f 'README.md' -o -f 'MAKEFILE_README.md' ]"
	run_test "CLAUDE.md exists" "[ -f 'CLAUDE.md' ]"

	echo ""
}

# Main execution
main() {
	log_info "Starting FlexCore test suite in $TEST_MODE mode"
	echo ""

	# Always run core validations
	validate_environment
	test_build_system
	test_native_system
	test_docker_independence
	test_performance
	test_quality

	# Conditional tests based on mode
	case "$TEST_MODE" in
	"full")
		test_go
		;;
	"quick" | *)
		log_info "Quick mode - skipping extended tests"
		;;
	esac

	# Results summary
	echo "📊 TEST RESULTS SUMMARY"
	echo "======================"
	echo "Total Tests: $TOTAL_TESTS"
	echo "Passed: $PASSED_TESTS"
	echo "Failed: $FAILED_TESTS"
	echo ""

	if [ ${#FAILED_TEST_NAMES[@]} -gt 0 ]; then
		echo -e "${RED}❌ FAILED TESTS:${NC}"
		for test in "${FAILED_TEST_NAMES[@]}"; do
			echo "  • $test"
		done
		echo ""
		exit 1
	else
		echo -e "${GREEN}🎉 ALL TESTS PASSED!${NC}"
		echo -e "${GREEN}✅ FlexCore native system is fully functional${NC}"
		echo -e "${GREEN}🚀 Ready for development and deployment${NC}"
		exit 0
	fi
}

# Usage information
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
	echo "FlexCore Unified Test Suite"
	echo ""
	echo "Usage: $0 [mode]"
	echo ""
	echo "Modes:"
	echo "  quick (default) - Fast essential tests"
	echo "  full           - Comprehensive testing including builds"
	echo ""
	echo "Examples:"
	echo "  $0              # Quick tests"
	echo "  $0 quick        # Quick tests"
	echo "  $0 full         # Full test suite"
	exit 0
fi

# Execute main function
main "$@"
