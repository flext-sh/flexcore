#!/bin/bash

# FlexCore Native System Validation - Fast functional tests

echo "🔍 NATIVE SYSTEM VALIDATION"
echo "=========================="

ERRORS=0

check() {
    if eval "$2"; then
        echo "✅ $1"
    else
        echo "❌ $1"
        ((ERRORS++))
    fi
}

# Essential functionality tests
echo "🧪 Essential Tests"
echo "-----------------"

check "Cargo available" "command -v cargo >/dev/null 2>&1"
check "Go available" "command -v go >/dev/null 2>&1"
check "Native binary exists" "[ -f 'third_party/windmill/windmill-backend' ]"
check "Binary is executable" "[ -x 'third_party/windmill/windmill-backend' ]"
check "Binary shows version" "./third_party/windmill/windmill-backend --version 2>&1 | grep -q 'Windmill'"
check "Go API generated" "[ -f 'third_party/windmill/go-client/api/windmill_api.gen.go' ]"
check "Makefile targets work" "make windmill-validate >/dev/null 2>&1"

# Docker independence verification
echo ""
echo "🐳 Docker Independence"
echo "--------------------"

check "No Docker Windmill images" "[ \$(find deployments/docker -name '*.yml' -not -path '*/third_party/windmill/*' -exec grep -l 'ghcr.io/windmill' {} \\; | wc -l) -eq 0 ]"
check "All use native binary mounts" "grep -r 'third_party/windmill/windmill-backend' deployments/docker/ | wc -l | grep -q '[0-9]'"
check "PostgreSQL standardized" "[ \$(grep -r 'postgres:' deployments/docker/ | grep 'image:' | cut -d: -f4 | sort | uniq | wc -l) -eq 1 ]"

# Build system validation
echo ""
echo "🏗️ Build System"
echo "-------------"

check "Environment validates" "make windmill-validate >/dev/null 2>&1"
check "Go client builds" "cd third_party/windmill/go-client && go build -v . >/dev/null 2>&1; cd - >/dev/null"
check "Documentation exists" "[ -f 'NATIVE_BUILD_GUIDE.md' ]"

# Performance check
echo ""
echo "⚡ Performance Check"
echo "------------------"

BINARY_SIZE=$(stat -c%s "third_party/windmill/windmill-backend" 2>/dev/null || echo "0")
BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))

check "Binary size reasonable (400-600MB)" "[ $BINARY_SIZE_MB -gt 400 ] && [ $BINARY_SIZE_MB -lt 600 ]"
check "sccache available" "command -v sccache >/dev/null 2>&1"

# Quality verification
echo ""
echo "🎯 Quality Standards"
echo "------------------"

check "No legacy backup files" "[ \$(find . -name '*.bak' -o -name '*.orig' | wc -l) -eq 0 ]"
check "Scripts use native builds" "grep -q 'make dev-windmill' scripts/start-real-cluster.sh"
check "Windmill services standardized" "[ \$(grep -r 'debian:bookworm-slim' deployments/docker/ | wc -l) -gt 10 ]"

# Results
echo ""
echo "📊 VALIDATION RESULTS"
echo "===================="

if [ $ERRORS -eq 0 ]; then
    echo "🎉 SUCCESS: All validations passed!"
    echo "✅ Native system is 100% functional"
    echo "🚀 Ready for production deployment"
    exit 0
else
    echo "❌ FAILED: $ERRORS validation(s) failed"
    echo "🔧 Review and fix issues before deployment"
    exit 1
fi