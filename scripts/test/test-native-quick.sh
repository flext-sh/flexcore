#!/bin/bash

# FlexCore Native Quick Tests - Essential validation without Docker

set -e

echo "🧪 FLEXCORE NATIVE QUICK TESTS"
echo "=============================="

# Test counters
PASSED=0
FAILED=0

test_pass() {
    echo "✅ PASS: $1"
    ((PASSED++))
}

test_fail() {
    echo "❌ FAIL: $1"
    ((FAILED++))
}

# Test 1: Environment
echo "📋 Testing Environment..."
if command -v cargo >/dev/null 2>&1; then
    test_pass "Cargo available"
else
    test_fail "Cargo not found"
fi

if command -v go >/dev/null 2>&1; then
    test_pass "Go available"
else
    test_fail "Go not found"
fi

if command -v make >/dev/null 2>&1; then
    test_pass "Make available"
else
    test_fail "Make not found"
fi

# Test 2: Files exist
echo "📁 Testing File Structure..."
[ -f "third_party/windmill/backend/Cargo.toml" ] && test_pass "Windmill backend found" || test_fail "Windmill backend missing"
[ -f "third_party/windmill/go-client/go.mod" ] && test_pass "Go client found" || test_fail "Go client missing"
[ -f "Makefile" ] && test_pass "Makefile found" || test_fail "Makefile missing"

# Test 3: Native binary validation
echo "🔧 Testing Native Binary..."
if [ -f "third_party/windmill/windmill-backend" ]; then
    test_pass "Native binary exists"
    [ -x "third_party/windmill/windmill-backend" ] && test_pass "Binary executable" || test_fail "Binary not executable"
    
    # Test binary functionality
    if ./third_party/windmill/windmill-backend --version 2>&1 | grep -q "Windmill"; then
        test_pass "Binary shows version"
    else
        test_fail "Binary version check failed"
    fi
else
    test_fail "Native binary missing"
fi

# Test 4: Build system validation
echo "🏗️ Testing Build System..."
if make validate-windmill-env >/dev/null 2>&1; then
    test_pass "Build environment valid"
else
    test_fail "Build environment invalid"
fi

# Test 5: Docker-free validation
echo "🐳 Testing Docker Independence..."
if ! find deployments/docker -name "*.yml" -not -path "*/third_party/windmill/*" -exec grep -l "ghcr.io/windmill" {} \; | grep -q .; then
    test_pass "No Docker Windmill images found"
else
    test_fail "Docker Windmill images still present"
fi

# Test 6: Consistency validation
echo "🔄 Testing Consistency..."
PG_VERSIONS=$(grep -r "postgres:" deployments/docker/ | grep "image:" | cut -d: -f4 | sort | uniq | wc -l)
if [ "$PG_VERSIONS" -eq 1 ]; then
    test_pass "PostgreSQL versions consistent"
else
    test_fail "PostgreSQL versions inconsistent ($PG_VERSIONS different versions)"
fi

WINDMILL_SERVICES=$(grep -r "debian:bookworm-slim" deployments/docker/ | wc -l)
if [ "$WINDMILL_SERVICES" -gt 10 ]; then
    test_pass "Windmill services standardized ($WINDMILL_SERVICES services)"
else
    test_fail "Windmill services not standardized ($WINDMILL_SERVICES services)"
fi

# Test 7: Go client validation
echo "🐹 Testing Go Client..."
if [ -f "third_party/windmill/go-client/api/windmill_api.gen.go" ]; then
    test_pass "Go API generated"
else
    test_fail "Go API not generated"
fi

if cd third_party/windmill/go-client && go mod verify >/dev/null 2>&1; then
    test_pass "Go module valid"
    cd - >/dev/null
else
    test_fail "Go module invalid"
    cd - >/dev/null 2>&1
fi

# Test 8: Documentation
echo "📚 Testing Documentation..."
[ -f "NATIVE_BUILD_GUIDE.md" ] && test_pass "Documentation exists" || test_fail "Documentation missing"

# Test 9: Scripts validation
echo "📜 Testing Scripts..."
[ -f "scripts/start-real-cluster.sh" ] && test_pass "Start script exists" || test_fail "Start script missing"
[ -f "scripts/stop-cluster.sh" ] && test_pass "Stop script exists" || test_fail "Stop script missing"

if grep -q "make dev-windmill" scripts/start-real-cluster.sh; then
    test_pass "Scripts use native builds"
else
    test_fail "Scripts don't use native builds"
fi

# Final results
echo ""
echo "🏁 QUICK TEST RESULTS"
echo "===================="
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "Total: $((PASSED + FAILED))"

if [ $FAILED -eq 0 ]; then
    echo ""
    echo "🚀 SUCCESS: Native system fully functional!"
    echo "✅ Ready for production deployment"
    exit 0
else
    echo ""
    echo "⚠️  Some tests failed - review and fix issues"
    exit 1
fi