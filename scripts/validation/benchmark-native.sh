#!/bin/bash

# FlexCore Native Performance Benchmarks

set -e

echo "⚡ FLEXCORE NATIVE PERFORMANCE BENCHMARKS"
echo "========================================"

# Benchmark 1: Build times
echo "🏗️ Build Performance Tests"
echo "-------------------------"

echo "Testing clean development build time..."
START_TIME=$(date +%s)
make clean-windmill >/dev/null 2>&1
make build-windmill-backend-dev >/dev/null 2>&1
DEV_TIME=$(($(date +%s) - START_TIME))
echo "✅ Development build: ${DEV_TIME}s"

echo "Testing incremental build time..."
START_TIME=$(date +%s)
make build-windmill-backend-dev >/dev/null 2>&1
INCREMENTAL_TIME=$(($(date +%s) - START_TIME))
echo "✅ Incremental build: ${INCREMENTAL_TIME}s"

echo "Testing release build time..."
START_TIME=$(date +%s)
make clean-windmill >/dev/null 2>&1
make build-windmill-backend-release >/dev/null 2>&1
RELEASE_TIME=$(($(date +%s) - START_TIME))
echo "✅ Release build: ${RELEASE_TIME}s"

# Benchmark 2: Binary analysis
echo ""
echo "📊 Binary Analysis"
echo "-----------------"

BINARY_PATH="third_party/windmill/windmill-backend"
if [ -f "$BINARY_PATH" ]; then
    BINARY_SIZE=$(stat -c%s "$BINARY_PATH")
    BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))
    echo "✅ Binary size: ${BINARY_SIZE_MB}MB"
    
    echo "✅ Binary format: $(file "$BINARY_PATH" | cut -d: -f2 | xargs)"
    
    # Test binary startup time
    echo "Testing binary startup time..."
    START_TIME=$(date +%s%N)
    timeout 5s "$BINARY_PATH" --version >/dev/null 2>&1 || true
    STARTUP_TIME=$((($(date +%s%N) - START_TIME) / 1000000))
    echo "✅ Startup time: ${STARTUP_TIME}ms"
else
    echo "❌ Binary not found at $BINARY_PATH"
fi

# Benchmark 3: Go client performance
echo ""
echo "🐹 Go Client Performance"
echo "----------------------"

cd third_party/windmill/go-client

echo "Testing Go API generation time..."
START_TIME=$(date +%s)
bash build.sh >/dev/null 2>&1
GENERATION_TIME=$(($(date +%s) - START_TIME))
echo "✅ API generation: ${GENERATION_TIME}s"

echo "Testing Go compilation time..."
START_TIME=$(date +%s)
go build -v . >/dev/null 2>&1
GO_BUILD_TIME=$(($(date +%s) - START_TIME))
echo "✅ Go build: ${GO_BUILD_TIME}s"

echo "Testing Go module verification..."
START_TIME=$(date +%s)
go mod verify >/dev/null 2>&1
MOD_VERIFY_TIME=$(($(date +%s) - START_TIME))
echo "✅ Module verify: ${MOD_VERIFY_TIME}s"

cd - >/dev/null

# Benchmark 4: Makefile target performance
echo ""
echo "🔧 Makefile Performance"
echo "---------------------"

echo "Testing environment validation time..."
START_TIME=$(date +%s)
make validate-windmill-env >/dev/null 2>&1
VALIDATION_TIME=$(($(date +%s) - START_TIME))
echo "✅ Environment validation: ${VALIDATION_TIME}s"

echo "Testing complete build suite time..."
START_TIME=$(date +%s)
make clean-windmill >/dev/null 2>&1
make build-windmill >/dev/null 2>&1
COMPLETE_BUILD_TIME=$(($(date +%s) - START_TIME))
echo "✅ Complete build suite: ${COMPLETE_BUILD_TIME}s"

# Benchmark 5: Cache performance
echo ""
echo "💾 Cache Performance"
echo "------------------"

if command -v sccache >/dev/null 2>&1; then
    echo "✅ sccache available"
    sccache --show-stats | grep "Cache hits rate" | head -1
    
    CACHE_SIZE=$(sccache --show-stats | grep "Cache size" | awk '{print $3}')
    echo "✅ Cache size: $CACHE_SIZE"
else
    echo "❌ sccache not available"
fi

# Summary
echo ""
echo "📈 PERFORMANCE SUMMARY"
echo "====================="
echo "Development build: ${DEV_TIME}s"
echo "Incremental build: ${INCREMENTAL_TIME}s"
echo "Release build: ${RELEASE_TIME}s"
echo "Binary size: ${BINARY_SIZE_MB}MB"
echo "Go API generation: ${GENERATION_TIME}s"
echo "Complete build: ${COMPLETE_BUILD_TIME}s"

echo ""
echo "🎯 PERFORMANCE TARGETS"
echo "====================="

# Evaluate performance against targets
PERFORMANCE_SCORE=0
TOTAL_CHECKS=6

if [ $DEV_TIME -lt 300 ]; then
    echo "✅ Development build < 300s: PASS"
    ((PERFORMANCE_SCORE++))
else
    echo "❌ Development build < 300s: FAIL (${DEV_TIME}s)"
fi

if [ $INCREMENTAL_TIME -lt 60 ]; then
    echo "✅ Incremental build < 60s: PASS"
    ((PERFORMANCE_SCORE++))
else
    echo "❌ Incremental build < 60s: FAIL (${INCREMENTAL_TIME}s)"
fi

if [ $RELEASE_TIME -lt 600 ]; then
    echo "✅ Release build < 600s: PASS"
    ((PERFORMANCE_SCORE++))
else
    echo "❌ Release build < 600s: FAIL (${RELEASE_TIME}s)"
fi

if [ $BINARY_SIZE_MB -lt 100 ] && [ $BINARY_SIZE_MB -gt 50 ]; then
    echo "✅ Binary size 50-100MB: PASS"
    ((PERFORMANCE_SCORE++))
else
    echo "❌ Binary size 50-100MB: FAIL (${BINARY_SIZE_MB}MB)"
fi

if [ $GENERATION_TIME -lt 10 ]; then
    echo "✅ API generation < 10s: PASS"
    ((PERFORMANCE_SCORE++))
else
    echo "❌ API generation < 10s: FAIL (${GENERATION_TIME}s)"
fi

if [ $COMPLETE_BUILD_TIME -lt 360 ]; then
    echo "✅ Complete build < 360s: PASS"
    ((PERFORMANCE_SCORE++))
else
    echo "❌ Complete build < 360s: FAIL (${COMPLETE_BUILD_TIME}s)"
fi

echo ""
echo "🏆 OVERALL PERFORMANCE: $PERFORMANCE_SCORE/$TOTAL_CHECKS"

if [ $PERFORMANCE_SCORE -eq $TOTAL_CHECKS ]; then
    echo "🚀 EXCELLENT: All performance targets met!"
    exit 0
elif [ $PERFORMANCE_SCORE -ge 4 ]; then
    echo "✅ GOOD: Most performance targets met"
    exit 0
else
    echo "⚠️  NEEDS IMPROVEMENT: Some performance targets missed"
    exit 1
fi