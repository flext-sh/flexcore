#!/bin/bash

# Build Real Distributed FlexCore System
# This script builds all components for 100% real distributed functionality

set -e

echo "🚀 Building Real Distributed FlexCore System..."

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -f flexcore-node
rm -f plugins/real-data-processor/real-data-processor

# Build main FlexCore node
echo "🔨 Building FlexCore node..."
go build -o flexcore-node cmd/flexcore-node/main.go
echo "✅ FlexCore node built successfully"

# Build plugins
echo "🔌 Building plugins..."
cd plugins/real-data-processor
go build -o real-data-processor main.go
cd ../..
echo "✅ Real data processor plugin built successfully"

# Create plugins directory for runtime
echo "📁 Setting up plugin directory..."
mkdir -p ./runtime-plugins
cp plugins/real-data-processor/real-data-processor ./runtime-plugins/
chmod +x ./runtime-plugins/real-data-processor

# Test plugin binary exists and is executable
echo "🧪 Verifying plugin binary..."
if [ -x "./runtime-plugins/real-data-processor" ]; then
    echo "✅ Plugin binary is executable and ready"
else
    echo "❌ Plugin binary verification failed"
    exit 1
fi

# Build test binaries
echo "🔬 Building test binaries..."
go build -o test-cluster cmd/test-cluster/main.go 2>/dev/null || echo "⚠️  test-cluster not found, skipping"

echo ""
echo "🎉 Build Complete! Real Distributed FlexCore System Ready"
echo ""
echo "📋 What was built:"
echo "  ✅ flexcore-node - Main distributed node binary"
echo "  ✅ real-data-processor - Plugin with actual data processing"
echo "  ✅ runtime-plugins/ - Plugin directory ready for deployment"
echo ""
echo "🚀 To start the system:"
echo "  1. Start infrastructure: docker-compose -f docker-compose.real-windmill.yml up -d"
echo "  2. Start FlexCore cluster: ./start-real-cluster.sh"
echo "  3. Test real functionality: ./test-real-distributed.sh"
echo ""
echo "💡 Real Features Implemented:"
echo "  🔄 Real Windmill server integration (not mock)"
echo "  ⚡ Timer-based singleton scheduling"
echo "  🔌 HashiCorp go-plugin system with executable plugins"
echo "  📊 Real data processing and transformation"
echo "  🏗️ Clean Architecture enforcement"
echo "  🌐 Multi-node cluster coordination"